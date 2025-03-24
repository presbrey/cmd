package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/presbrey/cmd/tq/lib"
)

func printUsage() {
	fmt.Fprintf(os.Stderr, "Usage: tq [options] [filter] [file...]\n\n")
	fmt.Fprintf(os.Stderr, "tq is a lightweight and flexible command-line TOML/JSON processor.\n")
	fmt.Fprintf(os.Stderr, "Similar to jq, it lets you slice, filter, and transform structured data.\n\n")
	fmt.Fprintf(os.Stderr, "Options:\n")
	flag.PrintDefaults()
	fmt.Fprintf(os.Stderr, "\nExamples:\n")
	fmt.Fprintf(os.Stderr, "  tq '.' example.toml            # Output the entire TOML file as JSON\n")
	fmt.Fprintf(os.Stderr, "  tq --toml '.' example.json     # Output the entire JSON file as TOML\n")
	fmt.Fprintf(os.Stderr, "  tq '.users' example.toml       # Extract just the 'users' field\n")
	fmt.Fprintf(os.Stderr, "  tq '.users[0]' example.toml    # Extract the first user\n")
	fmt.Fprintf(os.Stderr, "  cat example.toml | tq '.users' # Read from stdin\n")
}

func main() {
	// Define command-line flags more similar to jq
	toJson := flag.Bool("json", false, "Force JSON output (default for TOML input)")
	toToml := flag.Bool("toml", false, "Force TOML output (default for JSON input)")
	compact := flag.Bool("c", false, "Compact output instead of pretty-printed")
	rawOutput := flag.Bool("r", false, "Raw output (unwrap top-level values)")
	outputFile := flag.String("o", "", "Output file (default: stdout)")
	helpFlag := flag.Bool("help", false, "Show help information")
	flag.Parse()

	if *helpFlag {
		printUsage()
		os.Exit(0)
	}

	// Get filter and input files
	args := flag.Args()
	if len(args) == 0 {
		printUsage()
		os.Exit(1)
	}

	// First argument is the filter (like jq)
	filter := args[0]
	
	// Determine input source
	var input io.Reader
	var filename string
	
	if len(args) > 1 {
		// Input from file argument
		filename = args[1]
		file, err := os.Open(filename)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening input file: %v\n", err)
			os.Exit(1)
		}
		defer file.Close()
		input = file
	} else {
		// Input from stdin
		input = os.Stdin
	}

	// Set up output
	var output io.Writer
	if *outputFile != "" {
		file, err := os.Create(*outputFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating output file: %v\n", err)
			os.Exit(1)
		}
		defer file.Close()
		output = file
	} else {
		output = os.Stdout
	}

	// Determine conversion direction based on file extension if not explicitly specified
	if !*toJson && !*toToml && filename != "" {
		ext := strings.ToLower(filepath.Ext(filename))
		if ext == ".json" {
			*toToml = true
		} else if ext == ".toml" {
			*toJson = true
		}
	}

	// Default to TOML -> JSON if no direction is specified
	if !*toToml {
		*toJson = true
	}

	// Process the data with the filter
	var err error
	if *toJson {
		err = lib.TomlToJsonWithFilter(input, output, filter, *compact, *rawOutput)
	} else {
		err = lib.JsonToTomlWithFilter(input, output, filter, *compact)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error during processing: %v\n", err)
		os.Exit(1)
	}
}
