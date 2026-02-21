package parser

import (
	"bytes"
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/henrytill/hbt-go/internal/types"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

type MarkdownParser struct{}

type parserState struct {
	coll        *types.Collection
	currentDate time.Time
	labels      []string
	maybeParent *uint
	parents     []uint
}

func saveEntity(state *parserState, linkURL, linkTitle string) (uint, error) {
	parsedURL, err := url.Parse(linkURL)
	if err != nil {
		return 0, err
	}
	if parsedURL.Path == "" {
		parsedURL.Path = "/"
	}

	entity := types.Entity{
		URI:       parsedURL,
		CreatedAt: types.CreatedAt(state.currentDate),
		UpdatedAt: []types.UpdatedAt{},
		Names:     make(map[Name]struct{}),
		Labels:    make(map[Label]struct{}),
	}

	if linkTitle != "" {
		entity.Names = map[Name]struct{}{Name(linkTitle): {}}
	} else {
		entity.Names = make(map[Name]struct{})
	}

	entity.Labels = make(map[Label]struct{})
	if len(state.labels) > 0 {
		for _, label := range state.labels {
			if trimmedLabel := strings.TrimSpace(label); trimmedLabel != "" {
				entity.Labels[Label(trimmedLabel)] = struct{}{}
			}
		}
	}

	nodeID := state.coll.Upsert(entity)

	if len(state.parents) > 0 {
		immediateParent := state.parents[len(state.parents)-1]
		state.coll.AddEdges(nodeID, immediateParent)
	}

	return nodeID, nil
}

func extractText(node ast.Node, content []byte) string {
	var buf bytes.Buffer

	type workItem struct {
		node        ast.Node
		postProcess string
	}

	var worklist []workItem
	worklist = append(worklist, workItem{node: node})

	for len(worklist) > 0 {
		item := worklist[len(worklist)-1]
		worklist = worklist[:len(worklist)-1]

		if item.postProcess != "" {
			buf.WriteString(item.postProcess)
			continue
		}

		switch currentNode := item.node.(type) {
		case *ast.Text:
			buf.Write(currentNode.Segment.Value(content))
		case *ast.CodeSpan:
			buf.WriteByte('`')
			worklist = append(worklist, workItem{postProcess: "`"})
			for child := item.node.LastChild(); child != nil; child = child.PreviousSibling() {
				worklist = append(worklist, workItem{node: child})
			}
		default:
			for child := item.node.LastChild(); child != nil; child = child.PreviousSibling() {
				worklist = append(worklist, workItem{node: child})
			}
		}
	}

	return strings.TrimSpace(buf.String())
}

func (p *MarkdownParser) Parse(r io.Reader) (*types.Collection, error) {
	content, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	md := goldmark.New()
	doc := md.Parser().Parse(text.NewReader(content))

	state := parserState{
		coll:        types.NewCollection(),
		currentDate: time.Time{},
		labels:      []string{},
		maybeParent: nil,
		parents:     []uint{},
	}

	err = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		switch node := n.(type) {
		case *ast.Heading:
			if entering && node.Level == 1 {
				headingText := extractText(node, content)
				if parsed, err := time.Parse("January 2, 2006", headingText); err == nil {
					state.currentDate = parsed
				}
				state.maybeParent = nil
				state.labels = []string{}
				state.parents = []uint{}
			} else if entering && node.Level > 1 {
				headingText := extractText(node, content)
				level := int(node.Level) - 2

				if level < len(state.labels) {
					state.labels = state.labels[:level]
				}

				for len(state.labels) <= level {
					state.labels = append(state.labels, "")
				}
				state.labels[level] = headingText
			}
		case *ast.List:
			if entering {
				if state.maybeParent != nil {
					state.parents = append(state.parents, *state.maybeParent)
				}
			} else {
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
					id, err := saveEntity(&state, linkURL, linkTitle)
					if err != nil {
						return ast.WalkStop, err
					}
					state.maybeParent = &id
				}
			}
		case *ast.AutoLink:
			if entering {
				linkURL := string(node.URL(content))
				linkTitle := ""

				if linkURL != "" {
					id, err := saveEntity(&state, linkURL, linkTitle)
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

	return state.coll, nil
}
