# httppp - Pretty Printing HTTP Proxy

A simple HTTP proxy that pretty prints requests and responses to stdout, useful for debugging and inspecting HTTP traffic.

## Features

- Pretty prints HTTP requests and responses with clear formatting
- Automatically formats JSON payloads with indentation
- Preserves all headers and status codes
- Flexible configuration via environment variables or CLI flags (flags take precedence)
- Uses [caarlos0/env](https://github.com/caarlos0/env) for environment variable parsing
- Forwards request paths and query parameters to target

## Installation

```bash
cd httppp
go build -o ../bin/httppp
```

## Usage

### Using Environment Variables

Start the proxy server with environment variables:

```bash
TARGET_URL=https://api.example.com ./bin/httppp
```

To customize the port:

```bash
TARGET_URL=https://api.example.com PORT=3000 ./bin/httppp
```

To limit body output size (useful for large responses):

```bash
TARGET_URL=https://api.example.com MAX_BODY_SIZE=1024 ./bin/httppp
```

To print only headers without body content:

```bash
TARGET_URL=https://api.example.com ONLY_HEADERS=true ./bin/httppp
```

### Using CLI Flags

You can also use CLI flags (which override environment variables):

```bash
./bin/httppp -url https://api.example.com -port 3000
```

Limit body output size:

```bash
./bin/httppp -url https://api.example.com -max-body 1024
```

Print only headers:

```bash
./bin/httppp -url https://api.example.com -only-headers
```

Print only body (no headers):

```bash
./bin/httppp -url https://api.example.com -only-body
```

Print only JSON responses (filters out non-JSON content):

```bash
./bin/httppp -url https://api.example.com -only-json
```

### Combining Both

CLI flags take precedence over environment variables:

```bash
# PORT from env, url from flag
PORT=3000 ./bin/httppp -url https://api.example.com
```

Make requests through the proxy. The proxy will forward the path and query parameters to the target:

```bash
# GET request to https://api.example.com/users
curl http://localhost:8080/users

# POST request with JSON to https://api.example.com/users
curl -X POST http://localhost:8080/users \
  -H "Content-Type: application/json" \
  -d '{"name":"John","email":"john@example.com"}'

# Request with query parameters to https://api.example.com/users?active=true
curl http://localhost:8080/users?active=true
```

## Configuration

Configuration can be provided via environment variables, CLI flags, or both. **CLI flags take precedence** over environment variables.

### Environment Variables

- `TARGET_URL` (required*): Target URL to proxy requests to
- `PORT` (optional): Port to listen on (default: 8080)
- `MAX_BODY_SIZE` (optional): Maximum bytes to print from request/response bodies (default: 0 = unlimited)
- `ONLY_HEADERS` (optional): Print only headers, skip body content (default: false)
- `ONLY_BODY` (optional): Print only body, skip headers (default: false)
- `ONLY_JSON` (optional): Print only JSON bodies, skip non-JSON content (default: false)

*Required unless provided via `-url` flag

#### Using a `.env` file

```bash
# .env
TARGET_URL=https://api.example.com
PORT=8080
MAX_BODY_SIZE=1024
ONLY_HEADERS=false
```

Then run with:

```bash
export $(cat .env | xargs) && ./bin/httppp
```

### CLI Flags

- `-url` (required*): Target URL to proxy requests to (overrides `TARGET_URL`)
- `-port` (optional): Port to listen on (overrides `PORT`)
- `-max-body` (optional): Maximum bytes to print from request/response bodies (overrides `MAX_BODY_SIZE`)
- `-only-headers` (optional): Print only headers, skip body content (overrides `ONLY_HEADERS`)
- `-only-body` (optional): Print only body, skip headers (overrides `ONLY_BODY`)
- `-only-json` (optional): Print only JSON bodies, skip non-JSON content (overrides `ONLY_JSON`)

*Required unless provided via `TARGET_URL` environment variable

## Output Format

The proxy prints both requests and responses to stdout with clear separators:

```
======================================== REQUEST ========================================
GET https://api.example.com/users HTTP/1.1
Host: api.example.com
Content-Type: application/json

{
  "name": "John",
  "email": "john@example.com"
}
========================================================================================

======================================== RESPONSE ========================================
HTTP/1.1 200 OK
Content-Type: application/json

{
  "id": 123,
  "name": "John",
  "email": "john@example.com"
}
========================================================================================
```

## Testing

Run the integration tests:

```bash
go test -v ./httppp/...
```

## Architecture

- **main.go**: Entry point, CLI flag parsing, and HTTP server setup
- **internal/proxy/proxy.go**: Core proxy and pretty printing functionality
  - `Config`: Centralized configuration struct with caarlos0/env tags
  - `PrettyPrinter`: Handles formatting of HTTP requests/responses
  - `Handler`: HTTP proxy handler
- **main_test.go**: Integration tests

The implementation uses an internal package with a centralized `Config` struct that's passed throughout the application, making it easy to add new configuration options without changing function signatures.
