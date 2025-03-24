# ss - Socket Statistics

A cross-platform socket statistics utility for displaying information about network connections, similar to the Linux `ss` command but available on macOS.

## Overview

`ss` is a command-line tool that displays socket information, allowing you to view active network connections, listening ports, and associated processes. It's designed to provide similar functionality to the Linux `ss` command but works on macOS by leveraging the `lsof` command under the hood.

## Features

- Display TCP and UDP socket information
- Filter for listening sockets only
- Show process information for each socket
- Display all sockets (both listening and established)
- Numeric output option to avoid hostname resolution

## Installation

```bash
go install github.com/presbrey/cmd/ss@latest
```

## Usage

```
Usage: ss [options]

Options:
  -a    Display all sockets (listening and non-listening)
  -h    Display help
  -l    Display only listening sockets
  -n    Show numeric addresses instead of resolving host names
  -p    Show process using socket
  -t    Display TCP sockets
  -u    Display UDP sockets

Examples:
  ss -t       # Show TCP sockets
  ss -ua      # Show all UDP sockets
  ss -nlpt    # Show listening TCP socket processes in numeric format
```

## Output Format

The output includes the following columns:
- **Netid**: Protocol (tcp, udp)
- **State**: Socket state (LISTEN, ESTABLISHED, etc.)
- **Local Address:Port**: Local address and port
- **Peer Address:Port**: Remote address and port
- **Process**: Process name and PID (when using the `-p` option)

## Implementation Details

This tool is implemented in Go and uses platform-specific methods to gather socket information:
- On macOS, it uses the `lsof` command to collect socket information

## License

[License information]
