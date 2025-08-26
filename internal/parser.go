package internal

import (
	"fmt"
	"io"
)

// Parser defines the interface for parsing different input formats
type Parser interface {
	Parse(r io.Reader) (*Collection, error)
}

// ParserRegistry holds all available parsers
type ParserRegistry struct {
	parsers map[InputFormat]Parser
}

// NewParserRegistry creates a new parser registry
func NewParserRegistry() *ParserRegistry {
	return &ParserRegistry{
		parsers: make(map[InputFormat]Parser),
	}
}

// Register adds a parser for a specific format
func (r *ParserRegistry) Register(format InputFormat, parser Parser) {
	r.parsers[format] = parser
}

// GetParser returns a parser for the specified format
func (r *ParserRegistry) GetParser(format InputFormat) (Parser, error) {
	parser, exists := r.parsers[format]
	if !exists {
		return nil, fmt.Errorf("no parser available for format: %s", format)
	}
	return parser, nil
}

// InputFormat represents the supported input formats
type InputFormat string

const (
	FormatHTML     InputFormat = "html"
	FormatJSON     InputFormat = "json"
	FormatXML      InputFormat = "xml"
	FormatMarkdown InputFormat = "markdown"
)
