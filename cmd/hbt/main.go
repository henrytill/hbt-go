package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/henrytill/hbt-go/internal"
)

var (
	Version    = "0.1.0-dev"
	Commit     = "unknown"
	CommitDate = "unknown"
	TreeState  = "unknown"
)

type Format = internal.Format

type Config struct {
	InputFormat  Format
	OutputFormat Format
	OutputFile   *string
	Info         *bool
	ListTags     *bool
	Mappings     *string
	InputFile    string
}

func inputFormatsString() string {
	formats := internal.AllInputFormats()
	names := make([]string, len(formats))
	for i, f := range formats {
		names[i] = f.Name
	}
	return strings.Join(names, ", ")
}

func outputFormatsString() string {
	formats := internal.AllOutputFormats()
	names := make([]string, len(formats))
	for i, f := range formats {
		names[i] = f.Name
	}
	return strings.Join(names, ", ")
}

func detectInputFormat(filename string) (Format, error) {
	format, ok := internal.DetectInputFormat(filename)
	if !ok {
		ext := filepath.Ext(filename)
		return Format{}, fmt.Errorf("no parser for extension: %s", ext)
	}
	return format, nil
}

func showVersion() {
	fmt.Printf("hbt %s\n", Version)
}

func showUsage() {
	fmt.Printf("Usage: %s [OPTIONS] FILE\n\n", os.Args[0])
	fmt.Println("Process bookmark files in various formats")
	fmt.Println("\nOptions:")
	flag.PrintDefaults()
}

func main() {
	config := Config{
		InputFormat:  Format{Capability: internal.CapInput},
		OutputFormat: Format{Capability: internal.CapOutput},
		OutputFile:   flag.String("o", "", "Output file (defaults to stdout)"),
		Info:         flag.Bool("info", false, "Show collection info (entity count)"),
		ListTags:     flag.Bool("list-tags", false, "List all tags"),
		Mappings:     flag.String("mappings", "", "Read mappings from FILE"),
	}

	var showVersionFlag bool

	fromUsage := fmt.Sprintf("Input format (%s)", inputFormatsString())
	toUsage := fmt.Sprintf("Output format (%s)", outputFormatsString())

	flag.Var(&config.InputFormat, "f", fromUsage)
	flag.Var(&config.InputFormat, "from", fromUsage)
	flag.Var(&config.OutputFormat, "t", toUsage)
	flag.Var(&config.OutputFormat, "to", toUsage)
	flag.BoolVar(&showVersionFlag, "version", false, "Show version")
	flag.BoolVar(&showVersionFlag, "V", false, "Show version")

	flag.Usage = showUsage
	flag.Parse()

	if showVersionFlag {
		showVersion()
		return
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

	// If no input format was specified, detect it from the filename
	if config.InputFormat.Name == "" {
		format, err := detectInputFormat(config.InputFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		config.InputFormat = format
	}

	if !*config.Info && !*config.ListTags && config.OutputFormat.Name == "" {
		fmt.Fprintf(os.Stderr, "Error: Must specify an output format (-t) or analysis flag (--info, --list-tags)\n")
		os.Exit(1)
	}

	if _, err := os.Stat(config.InputFile); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error: Input file does not exist: %s\n", config.InputFile)
		os.Exit(1)
	}

	inputFile, err := os.Open(config.InputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening input file: %v\n", err)
		os.Exit(1)
	}
	defer inputFile.Close()

	collection, err := internal.Parse(config.InputFormat, inputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing file: %v\n", err)
		os.Exit(1)
	}

	if *config.Mappings != "" {
		mappings, err := internal.LoadMappingsFromFile(*config.Mappings)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading mappings file: %v\n", err)
			os.Exit(1)
		}
		collection.ApplyMappings(mappings)
	}

	if *config.Info {
		fmt.Printf("Collection contains %d entities\n", collection.Len())
		return
	}

	if *config.ListTags {
		tags := make(map[string]bool)
		for _, entity := range collection.Entities() {
			for label := range entity.Labels {
				if string(label) != "" {
					tags[string(label)] = true
				}
			}
		}
		fmt.Println("Tags found:")
		for tag := range tags {
			fmt.Printf("  %s\n", tag)
		}
		return
	}

	if config.OutputFormat.Name != "" {
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

		err = internal.Unparse(config.OutputFormat, output, collection)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
			os.Exit(1)
		}
	}
}
