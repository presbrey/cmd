# tq - TOML/JSON Processor

`tq` is a lightweight and flexible command-line processor for TOML and JSON data, inspired by [jq](https://stedolan.github.io/jq/). It allows you to slice, filter, map, and transform structured data with ease.

## Features

- Convert between TOML and JSON formats
- Filter data using jq-like syntax (`.field`, `.field[0]`)
- Pretty-print or compact output
- Raw output mode for unwrapped values
- Pipe-friendly for use in shell scripts

## Installation

```bash
go install github.com/presbrey/cmd/tq@latest
```

Or build from source:

```bash
git clone https://github.com/presbrey/cmd
cd cmd/tq
go build
```

## Usage

```
tq [options] [filter] [file...]
```

If no file is specified, `tq` reads from standard input.

### Options

- `--json`: Force JSON output (default for TOML input)
- `--toml`: Force TOML output (default for JSON input)
- `-c`: Compact output instead of pretty-printed
- `-r`: Raw output (unwrap top-level values)
- `-o FILE`: Write output to FILE instead of stdout
- `--help`: Show help information

### Filter Syntax

`tq` uses a simplified subset of jq's filter syntax:

- `.` - The entire document
- `.field` - Access a field in an object
- `.field1.field2` - Access a nested field
- `.array[0]` - Access an array element by index

## Examples

Convert TOML to JSON:
```bash
tq '.' example.toml
```

Convert JSON to TOML:
```bash
tq --toml '.' example.json
```

Extract a specific field:
```bash
tq '.servers.alpha' example.toml
```

Extract an array element:
```bash
tq '.database.ports[1]' example.toml
```

Use with pipes:
```bash
cat example.toml | tq '.servers'
```

Get raw output (no quotes around strings):
```bash
tq -r '.owner.name' example.toml
```

## Comparison with jq

While `jq` is specialized for JSON processing with a rich expression language, `tq` focuses on:

1. TOML/JSON conversion
2. Basic filtering with a simplified syntax
3. Familiar interface for jq users

`tq` is ideal for quick data extraction and format conversion tasks, especially when working with TOML files.

## License

[MIT License](LICENSE)
