package internal

import (
	"fmt"
	"io"

	"github.com/henrytill/hbt-go/internal/types"
)

type Parser interface {
	Parse(r io.Reader) (*types.Collection, error)
}

type ParserRegistry struct {
	parsers map[Format]Parser
}

func NewParserRegistry() *ParserRegistry {
	return &ParserRegistry{
		parsers: make(map[Format]Parser),
	}
}

func (r *ParserRegistry) Register(format Format, parser Parser) error {
	if !format.CanInput() {
		return fmt.Errorf("format %s cannot be used for input", format.Name)
	}
	r.parsers[format] = parser
	return nil
}

func (r *ParserRegistry) GetParser(format Format) (Parser, error) {
	if !format.CanInput() {
		return nil, fmt.Errorf("format %s cannot be used for input", format.Name)
	}
	parser, exists := r.parsers[format]
	if !exists {
		return nil, fmt.Errorf("no parser available for format: %s", format.Name)
	}
	return parser, nil
}
