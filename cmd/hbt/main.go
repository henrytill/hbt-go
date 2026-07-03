package main

import (
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
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
	InputFormat  internal.FormatFlag
	OutputFormat internal.FormatFlag
	OutputFile   *string
	Info         *bool
	ListTags     *bool
	Mappings     *string
	InputFile    string
}

func inputFormats() string {
	formats := internal.AllInputFormats()
	names := make([]string, len(formats))
	for i, f := range formats {
		names[i] = f.Name
	}
	return strings.Join(names, ", ")
}

func outputFormats() string {
	formats := internal.AllOutputFormats()
	names := make([]string, len(formats))
	for i, f := range formats {
		names[i] = f.Name
	}
	return strings.Join(names, ", ")
}

func showUsage() {
	fmt.Printf("Usage: %s [OPTIONS] FILE\n\n", os.Args[0])
	fmt.Println("Process bookmark files in various formats")
	fmt.Println("\nOptions:")
	flag.PrintDefaults()
}

func showVersion() {
	fmt.Printf("hbt %s\n", Version)
	if Commit != "unknown" {
		fmt.Printf("  commit: %s (%s, %s)\n", Commit, CommitDate, TreeState)
	}
}

func detectInputFormat(filename string) (Format, error) {
	format, ok := internal.DetectInputFormat(filename)
	if !ok {
		ext := filepath.Ext(filename)
		if ext == "" {
			return Format{}, fmt.Errorf("cannot detect input format of %s: no file extension (use -f)", filename)
		}
		return Format{}, fmt.Errorf("no parser for extension: %s", ext)
	}
	return format, nil
}

func detectOutputFormat(filename string) (Format, error) {
	format, ok := internal.DetectOutputFormat(filename)
	if !ok {
		ext := filepath.Ext(filename)
		if ext == "" {
			return Format{}, fmt.Errorf("cannot detect output format of %s: no file extension (use -t)", filename)
		}
		return Format{}, fmt.Errorf("no formatter for extension: %s", ext)
	}
	return format, nil
}

func main() {
	config := Config{
		InputFormat:  internal.NewInputFormatFlag(),
		OutputFormat: internal.NewOutputFormatFlag(),
		OutputFile:   flag.String("o", "", "Output file (defaults to stdout)"),
		Info:         flag.Bool("info", false, "Show collection info (entity count)"),
		ListTags:     flag.Bool("list-tags", false, "List all tags"),
		Mappings:     flag.String("mappings", "", "Read mappings from FILE"),
	}

	var showVersionFlag bool

	fromUsage := fmt.Sprintf("Input format (%s)", inputFormats())
	toUsage := fmt.Sprintf("Output format (%s)", outputFormats())

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
	if config.InputFormat.Format.Name == "" {
		format, err := detectInputFormat(config.InputFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		config.InputFormat.Format = format
	}

	// If no output format was specified, detect it from the output filename
	if config.OutputFormat.Format.Name == "" && *config.OutputFile != "" {
		format, err := detectOutputFormat(*config.OutputFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		config.OutputFormat.Format = format
	}

	if !*config.Info && !*config.ListTags && config.OutputFormat.Format.Name == "" {
		fmt.Fprintf(os.Stderr, "Error: Must specify an output format (-t) or analysis flag (--info, --list-tags)\n")
		os.Exit(1)
	}

	inputFile, err := os.Open(config.InputFile)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			fmt.Fprintf(os.Stderr, "Error: Input file does not exist: %s\n", config.InputFile)
		} else {
			fmt.Fprintf(os.Stderr, "Error opening input file: %v\n", err)
		}
		os.Exit(1)
	}
	defer inputFile.Close()

	coll, err := internal.Parse(config.InputFormat.Format, inputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing file: %v\n", err)
		os.Exit(1)
	}

	if *config.Mappings != "" {
		mappings, err := internal.LoadMappings(*config.Mappings)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading mappings file: %v\n", err)
			os.Exit(1)
		}
		coll.ApplyMappings(mappings)
	}

	if *config.Info {
		fmt.Printf("Collection contains %d entities\n", coll.Len())
		return
	}

	if *config.ListTags {
		tags := make(map[string]struct{})
		for entity := range coll.Entities() {
			for label := range entity.Labels {
				if string(label) != "" {
					tags[string(label)] = struct{}{}
				}
			}
		}
		fmt.Println("Tags found:")
		for _, tag := range slices.Sorted(maps.Keys(tags)) {
			fmt.Printf("  %s\n", tag)
		}
		return
	}

	if config.OutputFormat.Format.Name != "" {
		output := os.Stdout
		if *config.OutputFile != "" {
			output, err = os.Create(*config.OutputFile)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error creating output file: %v\n", err)
				os.Exit(1)
			}
		}

		err = internal.Unparse(config.OutputFormat.Format, output, &coll)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
			os.Exit(1)
		}

		// Close errors on the output file mean data may not have reached
		// disk; unlike the read side, they must not be ignored.
		if output != os.Stdout {
			if err := output.Close(); err != nil {
				fmt.Fprintf(os.Stderr, "Error closing output file: %v\n", err)
				os.Exit(1)
			}
		}
	}
}
