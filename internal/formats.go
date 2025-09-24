package internal

import (
	"fmt"
	"io"
	"strings"

	"github.com/henrytill/hbt-go/internal/formatter"
	"github.com/henrytill/hbt-go/internal/parser"
	"github.com/henrytill/hbt-go/internal/parser/pinboard"
	"github.com/henrytill/hbt-go/internal/types"
)

type FormatCapability uint8

const (
	CapInput FormatCapability = 1 << iota
	CapOutput
	CapBoth = CapInput | CapOutput
)

type Format struct {
	Name       string
	Capability FormatCapability
}

func (f Format) CanInput() bool  { return f.Capability&CapInput != 0 }
func (f Format) CanOutput() bool { return f.Capability&CapOutput != 0 }
func (f Format) String() string  { return f.Name }

var (
	JSON     = Format{"json", CapInput}
	XML      = Format{"xml", CapInput}
	Markdown = Format{"markdown", CapInput}
	HTML     = Format{"html", CapBoth}
	YAML     = Format{"yaml", CapOutput}
)

var parsers = map[Format]types.Parser{
	JSON:     &pinboard.PinboardJSONParser{},
	XML:      &pinboard.PinboardXMLParser{},
	Markdown: &parser.MarkdownParser{},
	HTML:     &parser.HTMLParser{},
}

var formatters = map[Format]types.Formatter{
	YAML: &formatter.YAMLFormatter{},
	HTML: &formatter.HTMLFormatter{},
}

var allFormats = []Format{JSON, XML, Markdown, HTML, YAML}

func AllInputFormats() []Format {
	var result []Format
	for _, format := range allFormats {
		if format.CanInput() {
			result = append(result, format)
		}
	}
	return result
}

func AllOutputFormats() []Format {
	var result []Format
	for _, format := range allFormats {
		if format.CanOutput() {
			result = append(result, format)
		}
	}
	return result
}

func parseFormat(name string) (Format, bool) {
	normalized := strings.ToLower(name)
	for _, format := range allFormats {
		if format.Name == normalized {
			return format, true
		}
	}
	return Format{}, false
}

func (f *Format) Set(value string) error {
	parsed, ok := parseFormat(value)
	if !ok {
		return fmt.Errorf("invalid format: %s", value)
	}

	if f.CanInput() && !parsed.CanInput() {
		return fmt.Errorf("format %s cannot be used for input", value)
	}
	if f.CanOutput() && !parsed.CanOutput() {
		return fmt.Errorf("format %s cannot be used for output", value)
	}

	*f = parsed
	return nil
}

func DetectInputFormat(filename string) (Format, bool) {
	switch strings.ToLower(filename[strings.LastIndex(filename, "."):]) {
	case ".html":
		return HTML, true
	case ".json":
		return JSON, true
	case ".xml":
		return XML, true
	case ".md":
		return Markdown, true
	default:
		return Format{}, false
	}
}

func Parse(format Format, r io.Reader) (*types.Collection, error) {
	if !format.CanInput() {
		return nil, fmt.Errorf("format %s cannot be used for input", format.Name)
	}

	parser, ok := parsers[format]
	if !ok {
		return nil, fmt.Errorf("no parser available for format: %s", format.Name)
	}

	return parser.Parse(r)
}

func Unparse(format Format, w io.Writer, collection *types.Collection) error {
	if !format.CanOutput() {
		return fmt.Errorf("format %s cannot be used for output", format.Name)
	}

	formatter, ok := formatters[format]
	if !ok {
		return fmt.Errorf("no formatter available for format: %s", format.Name)
	}

	return formatter.Format(w, collection)
}
