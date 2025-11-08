package proxy

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Config holds all configuration for the proxy
type Config struct {
	Port          string `env:"PORT" envDefault:"8080"`
	TargetURL     string `env:"TARGET_URL"`
	MaxBodySize   int    `env:"MAX_BODY_SIZE" envDefault:"0"`
	OnlyHeaders   bool   `env:"ONLY_HEADERS" envDefault:"false"`
	OnlyBody      bool   `env:"ONLY_BODY" envDefault:"false"`
	OnlyJSON      bool   `env:"ONLY_JSON" envDefault:"false"`
	SkipTLSVerify bool   `env:"SKIP_TLS_VERIFY" envDefault:"false"`
}

// PrettyPrinter handles pretty printing of HTTP requests and responses
type PrettyPrinter struct {
	output io.Writer
	config *Config
}

// NewPrettyPrinter creates a new PrettyPrinter
func NewPrettyPrinter(output io.Writer, config *Config) *PrettyPrinter {
	return &PrettyPrinter{
		output: output,
		config: config,
	}
}

// PrintRequest pretty prints an HTTP request
func (pp *PrettyPrinter) PrintRequest(req *http.Request) error {
	if !pp.config.OnlyBody && !pp.config.OnlyJSON {
		fmt.Fprintf(pp.output, "\n%s REQUEST %s\n", strings.Repeat("=", 40), strings.Repeat("=", 40))
		fmt.Fprintf(pp.output, "%s %s %s\n", req.Method, req.URL.String(), req.Proto)
		fmt.Fprintf(pp.output, "Host: %s\n", req.Host)

		for key, values := range req.Header {
			for _, value := range values {
				fmt.Fprintf(pp.output, "%s: %s\n", key, value)
			}
		}
	}

	if req.Body != nil {
		bodyBytes, err := io.ReadAll(req.Body)
		if err != nil {
			return err
		}
		req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

		if len(bodyBytes) > 0 && !pp.config.OnlyHeaders {
			contentType := req.Header.Get("Content-Type")

			// Skip if OnlyJSON is set and content is not JSON
			if pp.config.OnlyJSON && !strings.Contains(contentType, "application/json") {
				return nil
			}

			if pp.config.OnlyBody || pp.config.OnlyJSON {
				fmt.Fprintf(pp.output, "%s\n", pp.formatBody(bodyBytes, contentType))
			} else {
				fmt.Fprintf(pp.output, "\n%s\n", pp.formatBody(bodyBytes, contentType))
			}
		}
	}

	if !pp.config.OnlyBody && !pp.config.OnlyJSON {
		fmt.Fprintf(pp.output, "%s\n", strings.Repeat("=", 88))
	}
	return nil
}

// PrintResponse pretty prints an HTTP response
func (pp *PrettyPrinter) PrintResponse(resp *http.Response) error {
	if !pp.config.OnlyBody && !pp.config.OnlyJSON {
		fmt.Fprintf(pp.output, "\n%s RESPONSE %s\n", strings.Repeat("=", 39), strings.Repeat("=", 39))
		fmt.Fprintf(pp.output, "%s %s\n", resp.Proto, resp.Status)

		for key, values := range resp.Header {
			for _, value := range values {
				fmt.Fprintf(pp.output, "%s: %s\n", key, value)
			}
		}
	}

	if resp.Body != nil {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		resp.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

		if len(bodyBytes) > 0 && !pp.config.OnlyHeaders {
			contentType := resp.Header.Get("Content-Type")

			// Skip if OnlyJSON is set and content is not JSON
			if pp.config.OnlyJSON && !strings.Contains(contentType, "application/json") {
				return nil
			}

			if pp.config.OnlyBody || pp.config.OnlyJSON {
				fmt.Fprintf(pp.output, "%s\n", pp.formatBody(bodyBytes, contentType))
			} else {
				fmt.Fprintf(pp.output, "\n%s\n", pp.formatBody(bodyBytes, contentType))
			}
		}
	}

	if !pp.config.OnlyBody && !pp.config.OnlyJSON {
		fmt.Fprintf(pp.output, "%s\n\n", strings.Repeat("=", 88))
	}
	return nil
}

// formatBody attempts to pretty print the body based on content type
func (pp *PrettyPrinter) formatBody(body []byte, contentType string) string {
	// Truncate if maxBodySize is set and body exceeds it
	truncated := false
	if pp.config.MaxBodySize > 0 && len(body) > pp.config.MaxBodySize {
		body = body[:pp.config.MaxBodySize]
		truncated = true
	}

	var result string
	if strings.Contains(contentType, "application/json") {
		var prettyJSON bytes.Buffer
		if err := json.Indent(&prettyJSON, body, "", "  "); err == nil {
			result = prettyJSON.String()
		} else {
			result = string(body)
		}
	} else {
		result = string(body)
	}

	if truncated {
		result += fmt.Sprintf("\n... [truncated, showing first %d bytes]", pp.config.MaxBodySize)
	}
	return result
}

// Handler creates an HTTP handler that proxies requests and pretty prints them
type Handler struct {
	printer *PrettyPrinter
	client  *http.Client
	config  *Config
}

// NewHandler creates a new proxy handler
func NewHandler(printer *PrettyPrinter, config *Config) *Handler {
	client := &http.Client{}
	if config.SkipTLSVerify {
		client.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	}
	return &Handler{
		printer: printer,
		client:  client,
		config:  config,
	}
}

// ServeHTTP handles the proxy request
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Print the incoming request
	if err := h.printer.PrintRequest(r); err != nil {
		http.Error(w, fmt.Sprintf("Error printing request: %v", err), http.StatusInternalServerError)
		return
	}

	// Read the body if present
	var bodyBytes []byte
	if r.Body != nil {
		var err error
		bodyBytes, err = io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error reading request body: %v", err), http.StatusInternalServerError)
			return
		}
	}

	// Build the full target URL with the incoming request path and query
	targetURL := h.config.TargetURL + r.URL.Path
	if r.URL.RawQuery != "" {
		targetURL += "?" + r.URL.RawQuery
	}

	// Create the proxied request
	proxyReq, err := http.NewRequest(r.Method, targetURL, bytes.NewBuffer(bodyBytes))
	if err != nil {
		http.Error(w, fmt.Sprintf("Error creating proxy request: %v", err), http.StatusBadGateway)
		return
	}

	// Copy headers (excluding Host and connection-related headers)
	for key, values := range r.Header {
		if key == "Host" || strings.HasPrefix(key, "X-Forwarded") {
			continue
		}
		for _, value := range values {
			proxyReq.Header.Add(key, value)
		}
	}

	// Execute the request
	resp, err := h.client.Do(proxyReq)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error executing proxy request: %v", err), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Print the response
	if err := h.printer.PrintResponse(resp); err != nil {
		http.Error(w, fmt.Sprintf("Error printing response: %v", err), http.StatusInternalServerError)
		return
	}

	// Copy response headers
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	// Write status code
	w.WriteHeader(resp.StatusCode)

	// Copy response body
	if _, err := io.Copy(w, resp.Body); err != nil {
		fmt.Fprintf(h.printer.output, "Error copying response body: %v\n", err)
	}
}
