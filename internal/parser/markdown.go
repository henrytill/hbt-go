package parser

import (
	"bytes"
	"io"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/henrytill/hbt-go/internal"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

// MarkdownParser implements parsing for Markdown bookmark files using goldmark
type MarkdownParser struct{}

// NewMarkdownParser creates a new Markdown parser
func NewMarkdownParser() *MarkdownParser {
	return &MarkdownParser{}
}

type parserState struct {
	collection  *internal.Collection
	currentDate time.Time
	labels      []string
	maybeParent *uint
	parents     []uint
}

// Parse parses a Markdown bookmark file and returns a Collection
func (p *MarkdownParser) Parse(r io.Reader) (*internal.Collection, error) {
	content, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	// Parse with goldmark
	md := goldmark.New()
	doc := md.Parser().Parse(text.NewReader(content))

	state := &parserState{
		collection:  internal.NewCollection(),
		currentDate: time.Time{},
		labels:      []string{},
		maybeParent: nil,
		parents:     []uint{},
	}

	// Walk the AST and process events
	err = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		switch node := n.(type) {
		case *ast.Heading:
			if entering && node.Level == 1 {
				// Extract date from H1 heading
				headingText := extractText(node, content)
				if parsed, err := time.Parse("January 2, 2006", headingText); err == nil {
					state.currentDate = parsed
				}
				state.maybeParent = nil
				state.labels = []string{}
				state.parents = []uint{}
			} else if entering && node.Level > 1 {
				// Category heading
				headingText := extractText(node, content)
				level := int(node.Level) - 2 // Adjust for H2 = level 0

				// Truncate labels to current level
				if level < len(state.labels) {
					state.labels = state.labels[:level]
				}

				// Extend labels if needed and set current level
				for len(state.labels) <= level {
					state.labels = append(state.labels, "")
				}
				state.labels[level] = headingText
			}
		case *ast.List:
			if entering {
				// Starting a list - push maybe parent to parents stack
				if state.maybeParent != nil {
					state.parents = append(state.parents, *state.maybeParent)
				}
			} else {
				// Ending a list - pop from parents and reset maybe parent
				if len(state.parents) > 0 {
					state.parents = state.parents[:len(state.parents)-1]
				}
				state.maybeParent = nil
			}
		case *ast.Link:
			if entering {
				linkURL := string(node.Destination)
				linkTitle := extractText(node, content)

				if linkURL != "" {
					id, err := p.saveEntity(state, linkURL, linkTitle)
					if err != nil {
						return ast.WalkStop, err
					}
					state.maybeParent = &id
				}
			}
		case *ast.AutoLink:
			if entering {
				linkURL := string(node.URL(content))
				linkTitle := "" // Autolinks don't have separate title text

				if linkURL != "" {
					id, err := p.saveEntity(state, linkURL, linkTitle)
					if err != nil {
						return ast.WalkStop, err
					}
					state.maybeParent = &id
				}
			}
		}

		return ast.WalkContinue, nil
	})

	if err != nil {
		return nil, err
	}

	return state.collection, nil
}

func (p *MarkdownParser) saveEntity(state *parserState, linkURL, linkTitle string) (uint, error) {
	// Parse and normalize URL
	parsedURL, err := url.Parse(linkURL)
	if err != nil {
		return 0, err
	}
	if parsedURL.Path == "" {
		parsedURL.Path = "/"
	}

	entity := internal.Entity{
		URI:       parsedURL,
		CreatedAt: state.currentDate,
		UpdatedAt: []time.Time{},
		Names:     make(map[string]struct{}),
		Labels:    make(map[string]struct{}),
		Shared:    false,
		ToRead:    false,
		IsFeed:    false,
	}

	if linkTitle != "" {
		entity.Names = map[string]struct{}{linkTitle: {}}
	} else {
		entity.Names = make(map[string]struct{})
	}

	// Add current labels
	entity.Labels = make(map[string]struct{})
	if len(state.labels) > 0 {
		for _, label := range state.labels {
			if strings.TrimSpace(label) != "" {
				entity.Labels[strings.TrimSpace(label)] = struct{}{}
			}
		}
	}

	// Add entity to collection (or merge if URI already exists)
	nodeID := state.collection.UpsertEntity(entity)

	// Add edges to immediate parent only (like Rust/OCaml implementations)
	if len(state.parents) > 0 {
		immediateParent := state.parents[len(state.parents)-1]
		// Add bidirectional edges
		state.collection.Value[nodeID].Edges = append(state.collection.Value[nodeID].Edges, immediateParent)
		state.collection.Value[immediateParent].Edges = append(state.collection.Value[immediateParent].Edges, nodeID)

		// Sort edges
		slices.Sort(state.collection.Value[nodeID].Edges)
		slices.Sort(state.collection.Value[immediateParent].Edges)
	}

	return nodeID, nil
}

func extractText(node ast.Node, content []byte) string {
	var buf bytes.Buffer
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		switch childNode := child.(type) {
		case *ast.Text:
			buf.Write(childNode.Segment.Value(content))
		case *ast.CodeSpan:
			// Add backticks around code spans and recursively extract the content
			buf.WriteByte('`')
			buf.WriteString(extractText(child, content))
			buf.WriteByte('`')
		default:
			// Recursively extract text from other nodes
			buf.WriteString(extractText(child, content))
		}
	}
	return strings.TrimSpace(buf.String())
}
