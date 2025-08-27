package parser

import (
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/henrytill/hbt-go/internal"
	"golang.org/x/net/html"
)

type HTMLParser struct{}

// NewHTMLParser creates a new HTML parser
func NewHTMLParser() *HTMLParser {
	return &HTMLParser{}
}

type pendingBookmarkData struct {
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

func (p *HTMLParser) Parse(reader io.Reader) (*internal.Collection, error) {
	doc, err := html.Parse(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	collection := internal.NewCollection()
	return p.parseUsingStack(doc, collection)
}

type stackItem struct {
	node     *html.Node
	popGroup bool
}

func (p *HTMLParser) parseUsingStack(root *html.Node, collection *internal.Collection) (*internal.Collection, error) {
	var stack []stackItem
	var folderStack []string
	var pendingBookmark *pendingBookmarkData

	// Initialize stack with root's children in reverse order (like Rust implementation)
	for c := root.LastChild; c != nil; c = c.PrevSibling {
		if c.Type == html.ElementNode {
			stack = append(stack, stackItem{node: c, popGroup: false})
		}
	}

	for len(stack) > 0 {
		item := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		if item.popGroup {
			// Process any pending bookmark before popping folder
			if pendingBookmark != nil {
				if err := processPendingBookmark(collection, folderStack, *pendingBookmark); err != nil {
					return nil, err
				}
				pendingBookmark = nil
			}

			// Pop a folder from the stack
			if len(folderStack) > 0 {
				folderStack = folderStack[:len(folderStack)-1]
			}
			continue
		}

		node := item.node
		nodeName := strings.ToLower(node.Data)

		switch nodeName {
		case "dt":
			if err := p.handleDTStack(node, collection, &folderStack, &pendingBookmark); err != nil {
				return nil, err
			}
			// Also push children for processing (like nested DL elements)
			for c := node.LastChild; c != nil; c = c.PrevSibling {
				if c.Type == html.ElementNode {
					stack = append(stack, stackItem{node: c, popGroup: false})
				}
			}
		case "dd":
			if pendingBookmark != nil {
				description := strings.TrimSpace(getTextContent(node))
				if description != "" {
					pendingBookmark.description = description
				}
			}
		case "dl":
			// Push PopGroup marker first, then children in reverse order
			stack = append(stack, stackItem{popGroup: true})
			for c := node.LastChild; c != nil; c = c.PrevSibling {
				if c.Type == html.ElementNode {
					stack = append(stack, stackItem{node: c, popGroup: false})
				}
			}
		default:
			// Push children in reverse order for other elements
			for c := node.LastChild; c != nil; c = c.PrevSibling {
				if c.Type == html.ElementNode {
					stack = append(stack, stackItem{node: c, popGroup: false})
				}
			}
		}
	}

	// Process any remaining pending bookmark
	if pendingBookmark != nil {
		if err := processPendingBookmark(collection, folderStack, *pendingBookmark); err != nil {
			return nil, err
		}
	}

	return collection, nil
}

func (p *HTMLParser) handleDTStack(dtNode *html.Node, collection *internal.Collection, folderStack *[]string, pendingBookmark **pendingBookmarkData) error {
	// Process any pending bookmark first
	if *pendingBookmark != nil {
		if err := processPendingBookmark(collection, *folderStack, **pendingBookmark); err != nil {
			return err
		}
		*pendingBookmark = nil
	}

	// Look for A (bookmark) as direct child
	aNode := findDirectChildElement(dtNode, "a")
	if aNode != nil {
		bookmark := &pendingBookmarkData{
			href:         getAttr(aNode, "href"),
			title:        strings.TrimSpace(getTextContent(aNode)),
			addDate:      getAttr(aNode, "add_date"),
			lastModified: getAttr(aNode, "last_modified"),
			tags:         getAttr(aNode, "tags"),
			private:      getAttr(aNode, "private"),
			toread:       getAttr(aNode, "toread"),
			lastVisit:    getAttr(aNode, "last_visit"),
			feed:         getAttr(aNode, "feed"),
		}

		*pendingBookmark = bookmark
		return nil
	}

	// Look for H3 (folder) as direct child
	h3Node := findDirectChildElement(dtNode, "h3")
	if h3Node != nil {
		folderName := strings.TrimSpace(getTextContent(h3Node))
		if folderName != "" {
			*folderStack = append(*folderStack, folderName)
		}
	}

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

func getAttr(n *html.Node, attrName string) string {
	for _, attr := range n.Attr {
		if strings.EqualFold(attr.Key, attrName) {
			return attr.Val
		}
	}
	return ""
}

func getTextContent(n *html.Node) string {
	if n.Type == html.TextNode {
		return n.Data
	}
	var result strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		result.WriteString(getTextContent(c))
	}
	return result.String()
}

func processPendingBookmark(collection *internal.Collection, folderStack []string, bookmark pendingBookmarkData) error {
	if bookmark.href == "" {
		return nil
	}

	// Parse URL
	parsedURL, err := url.Parse(bookmark.href)
	if err != nil {
		return fmt.Errorf("failed to parse URL %s: %w", bookmark.href, err)
	}

	// Normalize URL by ensuring it has trailing slash for consistency with test data
	if parsedURL.Path == "" {
		parsedURL.Path = "/"
	}

	// Parse timestamps
	var createdAt time.Time
	if bookmark.addDate != "" {
		if parsed, err := strconv.ParseInt(bookmark.addDate, 10, 64); err == nil {
			createdAt = time.Unix(parsed, 0)
		}
	}
	if createdAt.IsZero() {
		createdAt = time.Now()
	}

	var lastVisitedAt *time.Time
	if bookmark.lastVisit != "" {
		if parsed, err := strconv.ParseInt(bookmark.lastVisit, 10, 64); err == nil {
			t := time.Unix(parsed, 0)
			lastVisitedAt = &t
		}
	}

	// Parse updatedAt from LAST_MODIFIED
	var updatedAt []time.Time
	if bookmark.lastModified != "" {
		if parsed, err := strconv.ParseInt(bookmark.lastModified, 10, 64); err == nil {
			updatedAt = append(updatedAt, time.Unix(parsed, 0))
		}
	}

	// Parse tags
	labels := make(map[string]struct{})
	if bookmark.tags != "" {
		tagList := strings.Split(bookmark.tags, ",")
		for _, tag := range tagList {
			tag = strings.TrimSpace(tag)
			if tag != "" && tag != "toread" {
				labels[tag] = struct{}{}
			}
		}
	}

	// Add folder labels
	for _, folder := range folderStack {
		labels[folder] = struct{}{}
	}

	// Parse privacy
	shared := true // default to public
	if bookmark.private == "1" {
		shared = false
	}

	// Parse toread
	toRead := bookmark.toread == "1" || strings.Contains(bookmark.tags, "toread")

	// Parse feed
	isFeed := bookmark.feed == "true"

	// Create names map
	names := make(map[string]struct{})
	if bookmark.title != "" {
		names[bookmark.title] = struct{}{}
	}

	// Create entity
	entity := internal.Entity{
		URI:       parsedURL,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
		Names:     names,
		Labels:    labels,
		Shared:    shared,
		ToRead:    toRead,
		IsFeed:    isFeed,
	}

	if bookmark.description != "" {
		entity.Extended = &bookmark.description
	}

	if lastVisitedAt != nil {
		entity.LastVisitedAt = lastVisitedAt
	}

	// Add to collection
	collection.AddEntity(entity)

	return nil
}
