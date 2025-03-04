package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	// Create a map to store filename -> content
	files := make(map[string]string)

	// Read current directory
	entries, err := os.ReadDir(".")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading directory: %v\n", err)
		os.Exit(1)
	}

	// Process each file
	for _, entry := range entries {
		if entry.IsDir() {
			continue // Skip directories
		}

		// Read file content
		content, err := os.ReadFile(filepath.Clean(entry.Name()))
		if err != nil {
			fmt.Fprintf(os.Stderr, "error reading %s: %v\n", entry.Name(), err)
			continue
		}

		// Store in map
		files[entry.Name()] = string(content)
	}

	// Encode to JSON
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(files); err != nil {
		fmt.Fprintf(os.Stderr, "error encoding JSON: %v\n", err)
		os.Exit(1)
	}
}
