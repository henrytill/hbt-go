package formatter

import (
	"io"

	"github.com/goccy/go-yaml"
	"github.com/henrytill/hbt-go/internal"
)

// YAMLFormatter implements YAML output formatting
type YAMLFormatter struct{}

// NewYAMLFormatter creates a new YAML formatter
func NewYAMLFormatter() *YAMLFormatter {
	return &YAMLFormatter{}
}

// Format writes the collection as YAML to the provided writer
func (f *YAMLFormatter) Format(w io.Writer, collection *internal.Collection) error {
	encoder := yaml.NewEncoder(w,
		yaml.UseSingleQuote(true),
		yaml.Indent(2),
	)
	defer encoder.Close()

	return encoder.Encode(collection)
}
