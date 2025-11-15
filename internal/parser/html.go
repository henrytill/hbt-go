package parser

import (
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/henrytill/hbt-go/internal/types"
	"golang.org/x/net/html"
)

type HTMLParser struct{}

type pendingBookmark struct {
	href         string
	title        string
	addDate      string
	lastModified string
	tags         string
	private      string
	toRead       string
	lastVisit    string
	feed         string
	description  string
}

func add(
	coll *types.Collection,
	folders []string,
	pending pendingBookmark,
) error {
	if pending.href == "" {
		return nil
	}

	parsedURL, err := url.Parse(pending.href)
	if err != nil {
		return fmt.Errorf("failed to parse URL %s: %w", pending.href, err)
	}

	if parsedURL.Path == "" {
		parsedURL.Path = "/"
	}

	var createdAt time.Time
	if pending.addDate != "" {
		if parsed, err := strconv.ParseInt(pending.addDate, 10, 64); err == nil {
			createdAt = time.Unix(parsed, 0)
		}
	}
	if createdAt.IsZero() {
		createdAt = time.Now()
	}

	var lastVisitedAt *time.Time
	if pending.lastVisit != "" {
		if parsed, err := strconv.ParseInt(pending.lastVisit, 10, 64); err == nil {
			t := time.Unix(parsed, 0)
			lastVisitedAt = &t
		}
	}

	var updatedAt []time.Time
	if pending.lastModified != "" {
		if parsed, err := strconv.ParseInt(pending.lastModified, 10, 64); err == nil {
			updatedAt = append(updatedAt, time.Unix(parsed, 0))
		}
	}

	labels := make(map[Label]struct{})
	if pending.tags != "" {
		tagList := strings.SplitSeq(pending.tags, ",")
		for tag := range tagList {
			tag = strings.TrimSpace(tag)
			if tag != "" && tag != "toread" {
				labels[Label(tag)] = struct{}{}
			}
		}
	}

	for _, folder := range folders {
		labels[Label(folder)] = struct{}{}
	}

	shared := true
	if pending.private == "1" {
		shared = false
	}

	toRead := false
	if pending.toRead == "1" {
		toRead = true
	}

	if pending.tags != "" {
		toRead = toRead || strings.Contains(pending.tags, "toread")
	}

	isFeed := false
	if pending.feed == "true" {
		isFeed = true
	}

	names := make(map[Name]struct{})
	if pending.title != "" {
		names[Name(pending.title)] = struct{}{}
	}

	entity := types.Entity{
		URI:       parsedURL,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
		Names:     names,
		Labels:    labels,
		Shared:    shared,
		ToRead:    toRead,
		IsFeed:    isFeed,
	}

	if pending.description != "" {
		ext := types.Extended(pending.description)
		entity.Extended = &ext
	}

	if lastVisitedAt != nil {
		entity.LastVisitedAt = lastVisitedAt
	}

	coll.Upsert(entity)

	return nil
}

func getTextContent(n *html.Node) string {
	var result strings.Builder
	var stack []*html.Node

	stack = append(stack, n)

	for len(stack) > 0 {
		current := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		if current.Type == html.TextNode {
			result.WriteString(current.Data)
			continue
		}

		for c := current.LastChild; c != nil; c = c.PrevSibling {
			stack = append(stack, c)
		}
	}

	return result.String()
}

func handleAnchor(anchor *html.Node) pendingBookmark {
	title := strings.TrimSpace(getTextContent(anchor))

	ret := pendingBookmark{title: title}

	for _, attr := range anchor.Attr {
		switch strings.ToLower(attr.Key) {
		case "href":
			ret.href = attr.Val
		case "add_date":
			ret.addDate = attr.Val
		case "last_modified":
			ret.lastModified = attr.Val
		case "tags":
			ret.tags = attr.Val
		case "private":
			ret.private = attr.Val
		case "toread":
			ret.toRead = attr.Val
		case "last_visit":
			ret.lastVisit = attr.Val
		case "feed":
			ret.feed = attr.Val
		}
	}

	return ret
}

func parse(root *html.Node, coll *types.Collection) (*types.Collection, error) {
	type stackItem struct {
		node     *html.Node
		popGroup bool
	}

	var (
		stack      []stackItem
		folders    []string
		pending    pendingBookmark
		hasPending bool
	)

	for c := root.LastChild; c != nil; c = c.PrevSibling {
		if c.Type == html.ElementNode {
			stack = append(stack, stackItem{node: c, popGroup: false})
		}
	}

	for len(stack) > 0 {
		item := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		if item.popGroup {
			if hasPending {
				if err := add(coll, folders, pending); err != nil {
					return nil, err
				}
				hasPending = false
			}
			if len(folders) > 0 {
				folders = folders[:len(folders)-1]
			}
			continue
		}

		node := item.node
		nodeName := strings.ToLower(node.Data)

		switch nodeName {
		case "dt":
			if hasPending {
				if err := add(coll, folders, pending); err != nil {
					return nil, err
				}
				hasPending = false
			}

			for c := node.FirstChild; c != nil; c = c.NextSibling {
				if c.Type != html.ElementNode {
					continue
				}
				switch strings.ToLower(c.Data) {
				case "a":
					pending = handleAnchor(c)
					hasPending = true
				case "h3":
					folderName := strings.TrimSpace(getTextContent(c))
					if folderName != "" {
						folders = append(folders, folderName)
					}
				}
			}
		case "dd":
			if hasPending {
				description := strings.TrimSpace(getTextContent(node))
				if description != "" {
					pending.description = description
				}
			}
			continue
		case "dl":
			stack = append(stack, stackItem{popGroup: true})
		}

		for c := node.LastChild; c != nil; c = c.PrevSibling {
			if c.Type == html.ElementNode {
				stack = append(stack, stackItem{node: c, popGroup: false})
			}
		}
	}

	if hasPending {
		return nil, fmt.Errorf("unexpected pending bookmark")
	}

	return coll, nil
}

func (p *HTMLParser) Parse(reader io.Reader) (*types.Collection, error) {
	doc, err := html.Parse(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	coll := types.NewCollection()
	return parse(doc, coll)
}
