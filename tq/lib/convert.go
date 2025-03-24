package lib

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

// TomlToJson converts TOML data to JSON
func TomlToJson(input io.Reader, output io.Writer) error {
	return TomlToJsonWithFilter(input, output, ".", false, false)
}

// JsonToToml converts JSON data to TOML
func JsonToToml(input io.Reader, output io.Writer) error {
	return JsonToTomlWithFilter(input, output, ".", false)
}

// TomlToJsonWithFilter converts TOML data to JSON with a filter expression
func TomlToJsonWithFilter(input io.Reader, output io.Writer, filter string, compact bool, raw bool) error {
	var data interface{}
	
	// Decode TOML
	decoder := toml.NewDecoder(input)
	if err := decoder.Decode(&data); err != nil {
		return err
	}
	
	// Apply filter
	filtered, err := applyFilter(data, filter)
	if err != nil {
		return err
	}
	
	// Encode as JSON
	encoder := json.NewEncoder(output)
	if !compact {
		encoder.SetIndent("", "  ")
	}
	
	// Handle raw output (unwrap top-level values)
	if raw {
		return outputRaw(filtered, output, compact)
	}
	
	return encoder.Encode(filtered)
}

// JsonToTomlWithFilter converts JSON data to TOML with a filter expression
func JsonToTomlWithFilter(input io.Reader, output io.Writer, filter string, compact bool) error {
	var data interface{}
	
	// Decode JSON
	decoder := json.NewDecoder(input)
	if err := decoder.Decode(&data); err != nil {
		return err
	}
	
	// Apply filter
	filtered, err := applyFilter(data, filter)
	if err != nil {
		return err
	}
	
	// Encode as TOML
	encoder := toml.NewEncoder(output)
	// Note: go-toml/v2 doesn't support indentation control like JSON
	return encoder.Encode(filtered)
}

// applyFilter applies a jq-like filter to the data
// Currently supports basic field access (.field) and array indexing (.field[0])
func applyFilter(data interface{}, filter string) (interface{}, error) {
	// Identity filter returns the entire document
	if filter == "." {
		return data, nil
	}
	
	// Remove leading dot if present
	if strings.HasPrefix(filter, ".") {
		filter = filter[1:]
	}
	
	// Split the filter into parts (handling both field access and array indexing)
	parts := parseFilterParts(filter)
	
	// Apply each part of the filter in sequence
	current := data
	for _, part := range parts {
		// Check if we're accessing an array element
		if strings.HasSuffix(part, "]") && strings.Contains(part, "[") {
			// Split into field name and array index
			idxStart := strings.Index(part, "[")
			fieldName := part[:idxStart]
			idxStr := part[idxStart+1 : len(part)-1]
			
			// Get the array first
			var arr interface{}
			if fieldName == "" {
				// Direct array access
				arr = current
			} else {
				// Field containing an array
				switch m := current.(type) {
				case map[string]interface{}:
					var ok bool
					arr, ok = m[fieldName]
					if !ok {
						return nil, fmt.Errorf("field '%s' not found", fieldName)
					}
				default:
					return nil, errors.New("cannot access field of non-object")
				}
			}
			
			// Parse the index
			var idx int
			if _, err := fmt.Sscanf(idxStr, "%d", &idx); err != nil {
				return nil, fmt.Errorf("invalid array index: %s", idxStr)
			}
			
			// Access the array element
			switch a := arr.(type) {
			case []interface{}:
				if idx < 0 || idx >= len(a) {
					return nil, fmt.Errorf("array index out of bounds: %d", idx)
				}
				current = a[idx]
			default:
				return nil, errors.New("cannot index non-array")
			}
		} else {
			// Regular field access
			switch m := current.(type) {
			case map[string]interface{}:
				var ok bool
				current, ok = m[part]
				if !ok {
					return nil, fmt.Errorf("field '%s' not found", part)
				}
			default:
				return nil, errors.New("cannot access field of non-object")
			}
		}
	}
	
	return current, nil
}

// parseFilterParts splits a filter string into its component parts
// Handles both field access (.field) and array indexing (.field[0])
func parseFilterParts(filter string) []string {
	if filter == "" {
		return []string{}
	}
	
	// Split by dots, but handle array access properly
	var parts []string
	current := ""
	bracketDepth := 0
	
	for _, r := range filter {
		switch r {
		case '.':
			if bracketDepth == 0 {
				if current != "" {
					parts = append(parts, current)
					current = ""
				}
			} else {
				current += string(r)
			}
		case '[':
			bracketDepth++
			current += string(r)
		case ']':
			bracketDepth--
			current += string(r)
		default:
			current += string(r)
		}
	}
	
	if current != "" {
		parts = append(parts, current)
	}
	
	return parts
}

// outputRaw outputs a value directly, without JSON object wrapping
func outputRaw(data interface{}, output io.Writer, compact bool) error {
	switch v := data.(type) {
	case string:
		// For strings, we output the raw string without quotes
		_, err := fmt.Fprint(output, v)
		return err
	case nil:
		// For null, output nothing
		return nil
	default:
		// For other types, use JSON encoding but capture the output
		buf := new(bytes.Buffer)
		encoder := json.NewEncoder(buf)
		if !compact {
			encoder.SetIndent("", "  ")
		}
		if err := encoder.Encode(v); err != nil {
			return err
		}
		
		// Remove the trailing newline that the encoder adds
		str := buf.String()
		str = strings.TrimSuffix(str, "\n")
		
		_, err := fmt.Fprint(output, str)
		return err
	}
}
