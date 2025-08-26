package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/henrytill/hbt-go/internal"
	"github.com/henrytill/hbt-go/internal/formatter"
	"github.com/henrytill/hbt-go/internal/parser"
)

const version = "0.1.0-dev"

type InputFormat = internal.InputFormat
type OutputFormat = internal.OutputFormat

const (
	FormatHTML     = internal.FormatHTML
	FormatJSON     = internal.FormatJSON
	FormatXML      = internal.FormatXML
	FormatMarkdown = internal.FormatMarkdown
)

const (
	OutputYAML = internal.OutputYAML
	OutputHTML = internal.OutputHTML
)

type Config struct {
	InputFormat  *InputFormat
	OutputFormat *OutputFormat
	OutputFile   *string
	Info         *bool
	ListTags     *bool
	Schema       *bool
	Mappings     *string
	InputFile    string
}

func detectInputFormat(filename string) (InputFormat, error) {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".html":
		return FormatHTML, nil
	case ".json":
		return FormatJSON, nil
	case ".xml":
		return FormatXML, nil
	case ".md":
		return FormatMarkdown, nil
	default:
		return "", fmt.Errorf("no parser for extension: %s", ext)
	}
}

func showVersion() {
	fmt.Printf("hbt %s\n", version)
}

func showUsage() {
	fmt.Printf("Usage: %s [OPTIONS] FILE\n\n", os.Args[0])
	fmt.Println("Process bookmark files in various formats")
	fmt.Println("\nOptions:")
	flag.PrintDefaults()
}

func parseInputFormat(s string) (InputFormat, error) {
	switch strings.ToLower(s) {
	case "html":
		return FormatHTML, nil
	case "json":
		return FormatJSON, nil
	case "xml":
		return FormatXML, nil
	case "markdown":
		return FormatMarkdown, nil
	default:
		return "", fmt.Errorf("invalid input format: %s", s)
	}
}

func parseOutputFormat(s string) (OutputFormat, error) {
	switch strings.ToLower(s) {
	case "yaml":
		return OutputYAML, nil
	case "html":
		return OutputHTML, nil
	default:
		return "", fmt.Errorf("invalid output format: %s", s)
	}
}

func main() {
	config := Config{
		InputFormat:  new(InputFormat),
		OutputFormat: new(OutputFormat),
		OutputFile:   flag.String("o", "", "Output file (defaults to stdout)"),
		Info:         flag.Bool("info", false, "Show collection info (entity count)"),
		ListTags:     flag.Bool("list-tags", false, "List all tags"),
		Schema:       flag.Bool("schema", false, "Output Collection JSON schema"),
		Mappings:     flag.String("mappings", "", "Read mappings from FILE"),
	}

	var inputFormatStr, outputFormatStr string
	var showVersionFlag bool

	fromFlag := flag.String("f", "", "Input format (html, json, xml, markdown)")
	fromFlagLong := flag.String("from", "", "Input format (html, json, xml, markdown)")
	toFlag := flag.String("t", "", "Output format (yaml, html)")
	toFlagLong := flag.String("to", "", "Output format (yaml, html)")
	flag.BoolVar(&showVersionFlag, "version", false, "Show version")
	flag.BoolVar(&showVersionFlag, "V", false, "Show version")

	flag.Usage = showUsage
	flag.Parse()

	if showVersionFlag {
		showVersion()
		return
	}

	if *config.Schema {
		fmt.Println("Schema output not yet implemented")
		os.Exit(1)
	}

	args := flag.Args()
	if len(args) != 1 {
		if len(args) == 0 {
			fmt.Fprintf(os.Stderr, "Error: input file required\n\n")
		} else {
			fmt.Fprintf(os.Stderr, "Error: exactly one input file required, got %d\n\n", len(args))
		}
		showUsage()
		os.Exit(1)
	}

	config.InputFile = args[0]

	// Parse input format
	inputFormatStr = *fromFlag
	if inputFormatStr == "" {
		inputFormatStr = *fromFlagLong
	}

	if inputFormatStr != "" {
		format, err := parseInputFormat(inputFormatStr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		*config.InputFormat = format
	} else {
		format, err := detectInputFormat(config.InputFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		*config.InputFormat = format
	}

	// Parse output format
	outputFormatStr = *toFlag
	if outputFormatStr == "" {
		outputFormatStr = *toFlagLong
	}

	if outputFormatStr != "" {
		format, err := parseOutputFormat(outputFormatStr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		*config.OutputFormat = format
	}

	// Validate that we have either an output format or an analysis flag
	if !*config.Info && !*config.ListTags && *config.OutputFormat == "" {
		fmt.Fprintf(os.Stderr, "Error: Must specify an output format (-t) or analysis flag (--info, --list-tags)\n")
		os.Exit(1)
	}

	// Check if input file exists
	if _, err := os.Stat(config.InputFile); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error: Input file does not exist: %s\n", config.InputFile)
		os.Exit(1)
	}

	// Only show processing messages if not outputting to stdout
	if *config.OutputFile != "" {
		fmt.Printf("Processing file: %s\n", config.InputFile)
		fmt.Printf("Input format: %s\n", *config.InputFormat)
		if *config.OutputFormat != "" {
			fmt.Printf("Output format: %s\n", *config.OutputFormat)
		}
	}
	if *config.Info {
		fmt.Println("Will show collection info")
	}
	if *config.ListTags {
		fmt.Println("Will list all tags")
	}

	// Initialize parser and formatter registries
	parserRegistry := internal.NewParserRegistry()
	formatterRegistry := internal.NewFormatterRegistry()

	// Register available parsers
	parserRegistry.Register(FormatMarkdown, parser.NewMarkdownParser())
	parserRegistry.Register(FormatJSON, parser.NewPinboardParser())
	parserRegistry.Register(FormatHTML, parser.NewHTMLParser())
	parserRegistry.Register(FormatXML, parser.NewXMLParser())

	// Register available formatters
	formatterRegistry.Register(OutputYAML, formatter.NewYAMLFormatter())
	formatterRegistry.Register(OutputHTML, formatter.NewHTMLFormatter())

	// Process the file
	inputFile, err := os.Open(config.InputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening input file: %v\n", err)
		os.Exit(1)
	}
	defer inputFile.Close()

	// Get parser for input format
	selectedParser, err := parserRegistry.GetParser(*config.InputFormat)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Parse the input
	collection, err := selectedParser.Parse(inputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing file: %v\n", err)
		os.Exit(1)
	}

	// Apply mappings if provided
	if *config.Mappings != "" {
		mappings, err := internal.LoadMappingsFromFile(*config.Mappings)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading mappings file: %v\n", err)
			os.Exit(1)
		}
		collection.ApplyMappings(mappings)
	}

	// Handle analysis flags
	if *config.Info {
		fmt.Printf("Collection contains %d entities\n", collection.Length)
		return
	}

	if *config.ListTags {
		tags := make(map[string]bool)
		for _, node := range collection.Value {
			for label := range node.Entity.Labels {
				if label != "" {
					tags[label] = true
				}
			}
		}
		fmt.Println("Tags found:")
		for tag := range tags {
			fmt.Printf("  %s\n", tag)
		}
		return
	}

	// Format and output
	if *config.OutputFormat != "" {
		selectedFormatter, err := formatterRegistry.GetFormatter(*config.OutputFormat)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		var output *os.File
		if *config.OutputFile != "" {
			output, err = os.Create(*config.OutputFile)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error creating output file: %v\n", err)
				os.Exit(1)
			}
			defer output.Close()
		} else {
			output = os.Stdout
		}

		err = selectedFormatter.Format(output, collection)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
			os.Exit(1)
		}
	}
}
