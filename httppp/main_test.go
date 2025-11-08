package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/presbrey/cmd/httppp/internal/proxy"
)

func TestProxyHandler(t *testing.T) {
	tests := []struct {
		name           string
		targetResponse string
		targetStatus   int
		targetHeaders  map[string]string
		method         string
		requestBody    string
		expectedStatus int
	}{
		{
			name:           "Simple GET request",
			targetResponse: "Hello, World!",
			targetStatus:   http.StatusOK,
			method:         "GET",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "POST request with body",
			targetResponse: `{"success": true}`,
			targetStatus:   http.StatusCreated,
			targetHeaders:  map[string]string{"Content-Type": "application/json"},
			method:         "POST",
			requestBody:    `{"name": "test"}`,
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "JSON response formatting",
			targetResponse: `{"user":"john","age":30,"active":true}`,
			targetStatus:   http.StatusOK,
			targetHeaders:  map[string]string{"Content-Type": "application/json"},
			method:         "GET",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test server to act as the target
			targetServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Set headers
				for key, value := range tt.targetHeaders {
					w.Header().Set(key, value)
				}
				w.WriteHeader(tt.targetStatus)
				w.Write([]byte(tt.targetResponse))
			}))
			defer targetServer.Close()

			// Create a buffer to capture pretty printed output
			var output bytes.Buffer
			cfg := &proxy.Config{
				TargetURL: targetServer.URL,
			}
			printer := proxy.NewPrettyPrinter(&output, cfg)
			handler := proxy.NewHandler(printer, cfg)

			// Create a test proxy server
			proxyServer := httptest.NewServer(handler)
			defer proxyServer.Close()

			// Build the proxy URL
			proxyURL := proxyServer.URL

			// Create the request
			var body io.Reader
			if tt.requestBody != "" {
				body = strings.NewReader(tt.requestBody)
			}

			req, err := http.NewRequest(tt.method, proxyURL, body)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			if tt.requestBody != "" {
				req.Header.Set("Content-Type", "application/json")
			}

			// Execute the request
			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("Failed to execute request: %v", err)
			}
			defer resp.Body.Close()

			// Check status code
			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}

			// Read response body
			respBody, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("Failed to read response body: %v", err)
			}

			if string(respBody) != tt.targetResponse {
				t.Errorf("Expected response body %q, got %q", tt.targetResponse, string(respBody))
			}

			// Verify that output contains request and response markers
			outputStr := output.String()
			if !strings.Contains(outputStr, "REQUEST") {
				t.Error("Output should contain REQUEST marker")
			}
			if !strings.Contains(outputStr, "RESPONSE") {
				t.Error("Output should contain RESPONSE marker")
			}

			// For JSON responses, verify pretty printing
			if tt.targetHeaders["Content-Type"] == "application/json" {
				var prettyJSON bytes.Buffer
				if err := json.Indent(&prettyJSON, []byte(tt.targetResponse), "", "  "); err == nil {
					if !strings.Contains(outputStr, prettyJSON.String()) {
						t.Error("JSON response should be pretty printed")
					}
				}
			}
		})
	}
}

func TestProxyHandlerWithPath(t *testing.T) {
	// Create a test server to act as the target
	targetServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/users" {
			t.Errorf("Expected path /api/users, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer targetServer.Close()

	var output bytes.Buffer
	cfg := &proxy.Config{
		TargetURL: targetServer.URL,
	}
	printer := proxy.NewPrettyPrinter(&output, cfg)
	handler := proxy.NewHandler(printer, cfg)

	req := httptest.NewRequest("GET", "/api/users", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestProxyHandlerInvalidURL(t *testing.T) {
	var output bytes.Buffer
	cfg := &proxy.Config{
		TargetURL: "not-a-valid-url",
	}
	printer := proxy.NewPrettyPrinter(&output, cfg)
	handler := proxy.NewHandler(printer, cfg)

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// Invalid URLs that can't be reached result in 502 Bad Gateway
	if w.Code != http.StatusBadGateway {
		t.Errorf("Expected status %d, got %d", http.StatusBadGateway, w.Code)
	}
}

func TestPrettyPrinterOutput(t *testing.T) {
	var output bytes.Buffer
	cfg := &proxy.Config{}
	printer := proxy.NewPrettyPrinter(&output, cfg)

	// Test request printing
	req := httptest.NewRequest("POST", "http://example.com/api/test", strings.NewReader(`{"test": "data"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer token123")

	err := printer.PrintRequest(req)
	if err != nil {
		t.Fatalf("PrintRequest failed: %v", err)
	}

	outputStr := output.String()

	// Verify request details are present
	if !strings.Contains(outputStr, "POST") {
		t.Error("Output should contain method POST")
	}
	if !strings.Contains(outputStr, "http://example.com/api/test") {
		t.Error("Output should contain request URL")
	}
	if !strings.Contains(outputStr, "Content-Type: application/json") {
		t.Error("Output should contain Content-Type header")
	}
	if !strings.Contains(outputStr, "Authorization: Bearer token123") {
		t.Error("Output should contain Authorization header")
	}

	// Test response printing
	output.Reset()
	resp := &http.Response{
		Status:     "200 OK",
		StatusCode: 200,
		Proto:      "HTTP/1.1",
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(`{"result": "success"}`)),
	}
	resp.Header.Set("Content-Type", "application/json")

	err = printer.PrintResponse(resp)
	if err != nil {
		t.Fatalf("PrintResponse failed: %v", err)
	}

	outputStr = output.String()

	// Verify response details are present
	if !strings.Contains(outputStr, "200 OK") {
		t.Error("Output should contain status 200 OK")
	}
	if !strings.Contains(outputStr, "Content-Type: application/json") {
		t.Error("Output should contain Content-Type header")
	}

	// Verify JSON is pretty printed
	if !strings.Contains(outputStr, "\"result\": \"success\"") {
		t.Error("JSON should be pretty printed with proper spacing")
	}
}

func TestMaxBodySize(t *testing.T) {
	var output bytes.Buffer
	maxSize := 20
	cfg := &proxy.Config{
		MaxBodySize: maxSize,
	}
	printer := proxy.NewPrettyPrinter(&output, cfg)

	// Create a response with a body larger than maxSize
	largeBody := `{"data": "this is a very long response body that should be truncated"}`
	resp := &http.Response{
		Status:     "200 OK",
		StatusCode: 200,
		Proto:      "HTTP/1.1",
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(largeBody)),
	}
	resp.Header.Set("Content-Type", "application/json")

	err := printer.PrintResponse(resp)
	if err != nil {
		t.Fatalf("PrintResponse failed: %v", err)
	}

	outputStr := output.String()

	// Verify truncation message is present
	if !strings.Contains(outputStr, "truncated") {
		t.Error("Output should contain truncation message")
	}
	if !strings.Contains(outputStr, fmt.Sprintf("first %d bytes", maxSize)) {
		t.Errorf("Output should mention first %d bytes", maxSize)
	}

	// Verify the full body is not present
	if strings.Contains(outputStr, "should be truncated") {
		t.Error("Output should not contain the end of the body")
	}
}

func TestOnlyHeaders(t *testing.T) {
	var output bytes.Buffer
	cfg := &proxy.Config{
		OnlyHeaders: true,
	}
	printer := proxy.NewPrettyPrinter(&output, cfg)

	// Test request with body
	reqBody := `{"test": "data"}`
	req := httptest.NewRequest("POST", "http://example.com/api/test", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")

	err := printer.PrintRequest(req)
	if err != nil {
		t.Fatalf("PrintRequest failed: %v", err)
	}

	outputStr := output.String()

	// Verify headers are present
	if !strings.Contains(outputStr, "POST") {
		t.Error("Output should contain method POST")
	}
	if !strings.Contains(outputStr, "Content-Type: application/json") {
		t.Error("Output should contain Content-Type header")
	}

	// Verify body is NOT present
	if strings.Contains(outputStr, reqBody) {
		t.Error("Output should not contain request body when onlyHeaders is true")
	}

	// Test response with body
	output.Reset()
	respBody := `{"result": "success"}`
	resp := &http.Response{
		Status:     "200 OK",
		StatusCode: 200,
		Proto:      "HTTP/1.1",
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(respBody)),
	}
	resp.Header.Set("Content-Type", "application/json")

	err = printer.PrintResponse(resp)
	if err != nil {
		t.Fatalf("PrintResponse failed: %v", err)
	}

	outputStr = output.String()

	// Verify headers are present
	if !strings.Contains(outputStr, "200 OK") {
		t.Error("Output should contain status 200 OK")
	}
	if !strings.Contains(outputStr, "Content-Type: application/json") {
		t.Error("Output should contain Content-Type header")
	}

	// Verify body is NOT present
	if strings.Contains(outputStr, respBody) {
		t.Error("Output should not contain response body when onlyHeaders is true")
	}
}

func TestOnlyBody(t *testing.T) {
	var output bytes.Buffer
	cfg := &proxy.Config{
		OnlyBody: true,
	}
	printer := proxy.NewPrettyPrinter(&output, cfg)

	// Test request with body
	reqBody := `{"test": "data"}`
	req := httptest.NewRequest("POST", "http://example.com/api/test", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")

	err := printer.PrintRequest(req)
	if err != nil {
		t.Fatalf("PrintRequest failed: %v", err)
	}

	outputStr := output.String()

	// Verify headers are NOT present
	if strings.Contains(outputStr, "REQUEST") {
		t.Error("Output should not contain REQUEST marker when onlyBody is true")
	}
	if strings.Contains(outputStr, "POST") {
		t.Error("Output should not contain method POST when onlyBody is true")
	}

	// Verify body IS present
	if !strings.Contains(outputStr, "test") {
		t.Error("Output should contain request body when onlyBody is true")
	}

	// Test response with body
	output.Reset()
	respBody := `{"result": "success"}`
	resp := &http.Response{
		Status:     "200 OK",
		StatusCode: 200,
		Proto:      "HTTP/1.1",
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(respBody)),
	}
	resp.Header.Set("Content-Type", "application/json")

	err = printer.PrintResponse(resp)
	if err != nil {
		t.Fatalf("PrintResponse failed: %v", err)
	}

	outputStr = output.String()

	// Verify headers are NOT present
	if strings.Contains(outputStr, "RESPONSE") {
		t.Error("Output should not contain RESPONSE marker when onlyBody is true")
	}
	if strings.Contains(outputStr, "200 OK") {
		t.Error("Output should not contain status when onlyBody is true")
	}

	// Verify body IS present
	if !strings.Contains(outputStr, "result") {
		t.Error("Output should contain response body when onlyBody is true")
	}
}

func TestOnlyJSON(t *testing.T) {
	var output bytes.Buffer
	cfg := &proxy.Config{
		OnlyJSON: true,
	}
	printer := proxy.NewPrettyPrinter(&output, cfg)

	// Test JSON response
	jsonBody := `{"result": "success"}`
	resp := &http.Response{
		Status:     "200 OK",
		StatusCode: 200,
		Proto:      "HTTP/1.1",
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(jsonBody)),
	}
	resp.Header.Set("Content-Type", "application/json")

	err := printer.PrintResponse(resp)
	if err != nil {
		t.Fatalf("PrintResponse failed: %v", err)
	}

	outputStr := output.String()

	// Verify headers are NOT present
	if strings.Contains(outputStr, "RESPONSE") {
		t.Error("Output should not contain RESPONSE marker when onlyJSON is true")
	}

	// Verify JSON body IS present and pretty printed
	if !strings.Contains(outputStr, "\"result\": \"success\"") {
		t.Error("Output should contain pretty printed JSON body when onlyJSON is true")
	}

	// Test non-JSON response (should be skipped)
	output.Reset()
	htmlBody := `<html><body>Hello</body></html>`
	resp2 := &http.Response{
		Status:     "200 OK",
		StatusCode: 200,
		Proto:      "HTTP/1.1",
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(htmlBody)),
	}
	resp2.Header.Set("Content-Type", "text/html")

	err = printer.PrintResponse(resp2)
	if err != nil {
		t.Fatalf("PrintResponse failed: %v", err)
	}

	outputStr = output.String()

	// Verify nothing was printed for non-JSON content
	if strings.Contains(outputStr, "html") || strings.Contains(outputStr, "Hello") {
		t.Error("Output should not contain non-JSON content when onlyJSON is true")
	}
}
