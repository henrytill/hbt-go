package formatter

import (
	"io"

	"github.com/goccy/go-yaml"
	"github.com/henrytill/hbt-go/internal/types"
)

type YAMLFormatter struct{}

func (f *YAMLFormatter) Format(w io.Writer, collection *types.Collection) error {
	encoder := yaml.NewEncoder(w,
		yaml.UseSingleQuote(true),
		yaml.Indent(2),
	)
	defer encoder.Close()

	return encoder.Encode(collection)
}
