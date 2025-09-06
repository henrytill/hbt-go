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

func NewHTMLParser() *HTMLParser {
	return &HTMLParser{}
}

type pendingBookmark struct {
	href         string
	title        string
	addDate      string
	lastModified string
	tags         string
	private      string
	toread       string
	lastVisit    string
	feed         string
	description  string
}

func processPendingBookmark(
	collection *types.Collection,
	folderStack []string,
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

	for _, folder := range folderStack {
		labels[Label(folder)] = struct{}{}
	}

	shared := true
	if pending.private == "1" {
		shared = false
	}

	toRead := false
	if pending.toread == "1" {
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
		entity.Extended = &pending.description
	}

	if lastVisitedAt != nil {
		entity.LastVisitedAt = lastVisitedAt
	}

	collection.UpsertEntity(entity)

	return nil
}

func findDirectChildElement(n *html.Node, tagName string) *html.Node {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && strings.ToLower(c.Data) == tagName {
			return c
		}
	}
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

func handleDt(
	dtNode *html.Node,
	collection *types.Collection,
	folderStack *[]string,
	pending **pendingBookmark,
) error {
	if *pending != nil {
		if err := processPendingBookmark(collection, *folderStack, **pending); err != nil {
			return err
		}
		*pending = nil
	}

	aNode := findDirectChildElement(dtNode, "a")
	if aNode == nil {
		h3Node := findDirectChildElement(dtNode, "h3")
		if h3Node != nil {
			folderName := strings.TrimSpace(getTextContent(h3Node))
			if folderName != "" {
				*folderStack = append(*folderStack, folderName)
			}
		}
		return nil
	}

	title := strings.TrimSpace(getTextContent(aNode))

	p := &pendingBookmark{title: title}

	for _, attr := range aNode.Attr {
		switch strings.ToLower(attr.Key) {
		case "href":
			p.href = attr.Val
		case "add_date":
			p.addDate = attr.Val
		case "last_modified":
			p.lastModified = attr.Val
		case "tags":
			p.tags = attr.Val
		case "private":
			p.private = attr.Val
		case "toread":
			p.toread = attr.Val
		case "last_visit":
			p.lastVisit = attr.Val
		case "feed":
			p.feed = attr.Val
		}
	}

	*pending = p
	return nil
}

func parse(
	root *html.Node,
	collection *types.Collection,
) (*types.Collection, error) {
	type stackItem struct {
		node     *html.Node
		popGroup bool
	}

	var stack []stackItem
	var folderStack []string
	var pending *pendingBookmark

	for c := root.LastChild; c != nil; c = c.PrevSibling {
		if c.Type == html.ElementNode {
			stack = append(stack, stackItem{node: c, popGroup: false})
		}
	}

	for len(stack) > 0 {
		item := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		if item.popGroup {
			if pending != nil {
				if err := processPendingBookmark(collection, folderStack, *pending); err != nil {
					return nil, err
				}
				pending = nil
			}

			if len(folderStack) > 0 {
				folderStack = folderStack[:len(folderStack)-1]
			}
			continue
		}

		node := item.node
		nodeName := strings.ToLower(node.Data)

		switch nodeName {
		case "dt":
			if err := handleDt(node, collection, &folderStack, &pending); err != nil {
				return nil, err
			}
		case "dd":
			if pending != nil {
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

	if pending != nil {
		return nil, fmt.Errorf("unexpected pending bookmark")
	}

	return collection, nil
}

func (p *HTMLParser) Parse(reader io.Reader) (*types.Collection, error) {
	doc, err := html.Parse(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	collection := types.NewCollection()
	return parse(doc, collection)
}
