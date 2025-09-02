package internal

import (
	"fmt"
	"io"
)

type Formatter interface {
	Format(w io.Writer, collection *Collection) error
}

type FormatterRegistry struct {
	formatters map[OutputFormat]Formatter
}

func NewFormatterRegistry() *FormatterRegistry {
	return &FormatterRegistry{
		formatters: make(map[OutputFormat]Formatter),
	}
}

func (r *FormatterRegistry) Register(format OutputFormat, formatter Formatter) {
	r.formatters[format] = formatter
}

func (r *FormatterRegistry) GetFormatter(format OutputFormat) (Formatter, error) {
	formatter, exists := r.formatters[format]
	if !exists {
		return nil, fmt.Errorf("no formatter available for format: %s", format)
	}
	return formatter, nil
}

type OutputFormat string

const (
	OutputYAML OutputFormat = "yaml"
	OutputHTML OutputFormat = "html"
)
