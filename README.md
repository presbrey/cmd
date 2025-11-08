# Command Line Utilities

This repository contains various command line utilities that can be installed using Go.

## Installation

You can install these utilities using one of the following methods:

### Using `go install` (recommended for Go 1.16+)

```bash
# Install all tools
go install github.com/presbrey/cmd/httppp@latest
go install github.com/presbrey/cmd/jls@latest
go install github.com/presbrey/cmd/ss@latest
go install github.com/presbrey/cmd/tq@latest

# Or install individual tools as needed
```

### Using `go get` (for older Go versions)

```bash
go get -u github.com/presbrey/cmd/httppp
go get -u github.com/presbrey/cmd/jls
go get -u github.com/presbrey/cmd/ss
go get -u github.com/presbrey/cmd/tq
```

## Available Commands

### httppp (HTTP Pretty Printer Proxy)
A debugging HTTP proxy that pretty prints requests and responses. Supports filtering by headers/body/JSON, body size limits, and TLS verification skipping.

**Usage:**
```bash
# Basic usage
httppp -url https://api.example.com

# Filter options
httppp -url https://api.example.com -only-json -max-body 1000
```

**Flags:**
- `-port`: Port to listen on (default: 8080)
- `-url`: Target URL to proxy requests to (required)
- `-max-body`: Maximum bytes to print from request/response bodies
- `-only-headers`: Print only headers, skip body content
- `-only-body`: Print only body, skip headers
- `-only-json`: Print only JSON bodies, skip non-JSON content
- `-skip-tls-verify`: Skip TLS certificate verification

### jls (JSON Directory Listing)
A simple utility that outputs the contents of all files in the current directory as a JSON object, with filenames as keys and file contents as values.

### ss (Socket Statistics)
A cross-platform socket statistics utility for displaying information about network connections, similar to the Linux `ss` command but available on macOS.

### tq (TOML/JSON Processor)
A lightweight and flexible command-line TOML/JSON processor, similar to `jq`, that lets you slice, filter, and transform structured data between TOML and JSON formats.

## Requirements

- Go 1.23.6 or later

## License

MIT License. See [LICENSE](LICENSE) file for details.
