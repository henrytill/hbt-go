package internal

import (
	"fmt"
	"io"

	"github.com/henrytill/hbt-go/internal/types"
)

type Formatter interface {
	Format(w io.Writer, collection *types.Collection) error
}

type FormatterRegistry struct {
	formatters map[Format]Formatter
}

func NewFormatterRegistry() *FormatterRegistry {
	return &FormatterRegistry{
		formatters: make(map[Format]Formatter),
	}
}

func (r *FormatterRegistry) Register(format Format, formatter Formatter) error {
	if !format.CanOutput() {
		return fmt.Errorf("format %s cannot be used for output", format.Name)
	}
	r.formatters[format] = formatter
	return nil
}

func (r *FormatterRegistry) GetFormatter(format Format) (Formatter, error) {
	if !format.CanOutput() {
		return nil, fmt.Errorf("format %s cannot be used for output", format.Name)
	}
	formatter, exists := r.formatters[format]
	if !exists {
		return nil, fmt.Errorf("no formatter available for format: %s", format.Name)
	}
	return formatter, nil
}
