package internal

import (
	"fmt"
	"io"
)

// Formatter defines the interface for formatting collection output
type Formatter interface {
	Format(w io.Writer, collection *Collection) error
}

// FormatterRegistry holds all available formatters
type FormatterRegistry struct {
	formatters map[OutputFormat]Formatter
}

// NewFormatterRegistry creates a new formatter registry
func NewFormatterRegistry() *FormatterRegistry {
	return &FormatterRegistry{
		formatters: make(map[OutputFormat]Formatter),
	}
}

// Register adds a formatter for a specific output format
func (r *FormatterRegistry) Register(format OutputFormat, formatter Formatter) {
	r.formatters[format] = formatter
}

// GetFormatter returns a formatter for the specified output format
func (r *FormatterRegistry) GetFormatter(format OutputFormat) (Formatter, error) {
	formatter, exists := r.formatters[format]
	if !exists {
		return nil, fmt.Errorf("no formatter available for format: %s", format)
	}
	return formatter, nil
}

// OutputFormat represents the supported output formats
type OutputFormat string

const (
	OutputYAML OutputFormat = "yaml"
	OutputHTML OutputFormat = "html"
)
