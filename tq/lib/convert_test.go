package lib

import (
	"bytes"
	"strings"
	"testing"
)

func TestTomlToJson(t *testing.T) {
	tomlData := `
title = "TOML Example"

[owner]
name = "Tom Preston-Werner"
dob = 1979-05-27T07:32:00-08:00
`
	expectedJson := `{
  "owner": {
    "dob": "1979-05-27T07:32:00-08:00",
    "name": "Tom Preston-Werner"
  },
  "title": "TOML Example"
}
`

	input := strings.NewReader(tomlData)
	output := &bytes.Buffer{}

	err := TomlToJson(input, output)
	if err != nil {
		t.Fatalf("TomlToJson failed: %v", err)
	}

	// Normalize line endings for comparison
	actual := strings.TrimSpace(output.String())
	expected := strings.TrimSpace(expectedJson)
	
	if actual != expected {
		t.Errorf("Expected JSON:\n%s\n\nGot:\n%s", expected, actual)
	}
}

func TestJsonToToml(t *testing.T) {
	jsonData := `{
  "title": "JSON Example",
  "owner": {
    "name": "Tom Preston-Werner",
    "dob": "1979-05-27T07:32:00-08:00"
  }
}
`

	input := strings.NewReader(jsonData)
	output := &bytes.Buffer{}

	err := JsonToToml(input, output)
	if err != nil {
		t.Fatalf("JsonToToml failed: %v", err)
	}

	// Get the actual output
	actual := strings.TrimSpace(output.String())
	
	// Check that the output contains the expected data, regardless of quote style
	if !strings.Contains(actual, "title") || 
	   !strings.Contains(actual, "JSON Example") ||
	   !strings.Contains(actual, "owner") ||
	   !strings.Contains(actual, "Tom Preston-Werner") ||
	   !strings.Contains(actual, "1979-05-27T07:32:00-08:00") {
		t.Errorf("TOML output missing expected content:\n%s", actual)
	}
}

func TestRoundTrip(t *testing.T) {
	// Test TOML -> JSON -> TOML
	originalToml := `
title = "Round Trip Test"

[nested]
value = 42
enabled = true
`
	
	// First convert TOML to JSON
	tomlInput := strings.NewReader(originalToml)
	jsonOutput := &bytes.Buffer{}
	
	err := TomlToJson(tomlInput, jsonOutput)
	if err != nil {
		t.Fatalf("TomlToJson failed: %v", err)
	}
	
	// Then convert JSON back to TOML
	jsonInput := strings.NewReader(jsonOutput.String())
	tomlOutput := &bytes.Buffer{}
	
	err = JsonToToml(jsonInput, tomlOutput)
	if err != nil {
		t.Fatalf("JsonToToml failed: %v", err)
	}
	
	// Check that the output contains the expected data
	finalToml := strings.TrimSpace(tomlOutput.String())
	
	// Check for key elements regardless of formatting
	if !strings.Contains(finalToml, "title") || 
	   !strings.Contains(finalToml, "Round Trip Test") ||
	   !strings.Contains(finalToml, "nested") ||
	   !strings.Contains(finalToml, "enabled") ||
	   !strings.Contains(finalToml, "true") ||
	   !strings.Contains(finalToml, "value") {
		t.Errorf("Round trip conversion failed.\nOriginal TOML:\n%s\n\nFinal TOML:\n%s", 
			strings.TrimSpace(originalToml), finalToml)
	}
}
