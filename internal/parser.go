package internal

import (
	"fmt"
	"io"
)

type Parser interface {
	Parse(r io.Reader) (*Collection, error)
}

type ParserRegistry struct {
	parsers map[InputFormat]Parser
}

func NewParserRegistry() *ParserRegistry {
	return &ParserRegistry{
		parsers: make(map[InputFormat]Parser),
	}
}

func (r *ParserRegistry) Register(format InputFormat, parser Parser) {
	r.parsers[format] = parser
}

func (r *ParserRegistry) GetParser(format InputFormat) (Parser, error) {
	parser, exists := r.parsers[format]
	if !exists {
		return nil, fmt.Errorf("no parser available for format: %s", format)
	}
	return parser, nil
}

type InputFormat string

const (
	FormatHTML     InputFormat = "html"
	FormatJSON     InputFormat = "json"
	FormatXML      InputFormat = "xml"
	FormatMarkdown InputFormat = "markdown"
)
