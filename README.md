# Command Line Utilities

This repository contains various command line utilities that can be installed using Go.

## Installation

You can install these utilities using one of the following methods:

### Using `go install` (recommended for Go 1.16+)

```bash
# Install all tools
go install github.com/presbrey/cmd/jls@latest
go install github.com/presbrey/cmd/ss@latest
go install github.com/presbrey/cmd/tq@latest

# Or install individual tools as needed
```

### Using `go get` (for older Go versions)

```bash
go get -u github.com/presbrey/cmd/jls
go get -u github.com/presbrey/cmd/ss
go get -u github.com/presbrey/cmd/tq
```

## Available Commands

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
