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

	var lastVisitedAt types.LastVisitedAt
	if pending.lastVisit != "" {
		if parsed, err := strconv.ParseInt(pending.lastVisit, 10, 64); err == nil {
			lastVisitedAt = types.NewLastVisitedAt(time.Unix(parsed, 0))
		}
	}

	var updatedAt []types.UpdatedAt
	if pending.lastModified != "" {
		if parsed, err := strconv.ParseInt(pending.lastModified, 10, 64); err == nil {
			updatedAt = append(updatedAt, types.UpdatedAt(time.Unix(parsed, 0)))
		}
	}

	labels := make(map[types.Label]struct{})
	if pending.tags != "" {
		tagList := strings.SplitSeq(pending.tags, ",")
		for tag := range tagList {
			tag = strings.TrimSpace(tag)
			if tag != "" && tag != "toread" {
				labels[types.Label(tag)] = struct{}{}
			}
		}
	}

	for _, folder := range folders {
		labels[types.Label(folder)] = struct{}{}
	}

	var shared types.Shared
	if pending.private != "" {
		shared = types.NewShared(pending.private != "1")
	}

	var toRead types.ToRead
	if pending.toRead != "" {
		toRead = types.NewToRead(pending.toRead == "1")
	} else if pending.tags != "" && strings.Contains(pending.tags, "toread") {
		toRead = types.NewToRead(true)
	}

	var isFeed types.IsFeed
	if pending.feed == "true" {
		isFeed = types.NewIsFeed(true)
	}

	names := make(map[types.Name]struct{})
	if pending.title != "" {
		names[types.Name(pending.title)] = struct{}{}
	}

	entity := types.Entity{
		URI:       parsedURL,
		CreatedAt: types.CreatedAt(createdAt),
		UpdatedAt: updatedAt,
		Names:     names,
		Labels:    labels,
		Shared:    shared,
		ToRead:    toRead,
		IsFeed:    isFeed,
	}

	if pending.description != "" {
		entity.Extended = []types.Extended{types.Extended(pending.description)}
	}

	entity.LastVisitedAt = lastVisitedAt

	coll.Upsert(entity)

	return nil
}

func getTextContent(n *html.Node) string {
	var result strings.Builder
	var worklist []*html.Node

	worklist = append(worklist, n)

	for len(worklist) > 0 {
		current := worklist[len(worklist)-1]
		worklist = worklist[:len(worklist)-1]

		if current.Type == html.TextNode {
			result.WriteString(current.Data)
			continue
		}

		for c := current.LastChild; c != nil; c = c.PrevSibling {
			worklist = append(worklist, c)
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
	type workItem struct {
		node     *html.Node
		popGroup bool
	}

	var (
		worklist   []workItem
		folders    []string
		pending    pendingBookmark
		hasPending bool
	)

	for c := root.LastChild; c != nil; c = c.PrevSibling {
		if c.Type == html.ElementNode {
			worklist = append(worklist, workItem{node: c, popGroup: false})
		}
	}

	for len(worklist) > 0 {
		item := worklist[len(worklist)-1]
		worklist = worklist[:len(worklist)-1]

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
			worklist = append(worklist, workItem{popGroup: true})
		}

		for c := node.LastChild; c != nil; c = c.PrevSibling {
			if c.Type == html.ElementNode {
				worklist = append(worklist, workItem{node: c, popGroup: false})
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
