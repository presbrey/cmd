package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/caarlos0/env/v11"
	"github.com/presbrey/cmd/httppp/internal/proxy"
)

func main() {
	// Define CLI flags
	port := flag.String("port", "", "Port to listen on (overrides PORT env var)")
	targetURL := flag.String("url", "", "Target URL to proxy requests to (overrides TARGET_URL env var)")
	maxBodySize := flag.Int("max-body", -1, "Maximum bytes to print from request/response bodies (overrides MAX_BODY_SIZE env var)")
	onlyHeaders := flag.Bool("only-headers", false, "Print only headers, skip body content (overrides ONLY_HEADERS env var)")
	onlyBody := flag.Bool("only-body", false, "Print only body, skip headers (overrides ONLY_BODY env var)")
	onlyJSON := flag.Bool("only-json", false, "Print only JSON bodies, skip non-JSON content (overrides ONLY_JSON env var)")
	skipTLSVerify := flag.Bool("skip-tls-verify", false, "Skip TLS certificate verification (overrides SKIP_TLS_VERIFY env var)")
	flag.Parse()

	// Parse environment variables first
	cfg, err := env.ParseAs[proxy.Config]()
	if err != nil {
		log.Fatalf("Failed to parse configuration: %v", err)
	}

	// CLI flags override environment variables
	if *port != "" {
		cfg.Port = *port
	}
	if *targetURL != "" {
		cfg.TargetURL = *targetURL
	}
	if *maxBodySize >= 0 {
		cfg.MaxBodySize = *maxBodySize
	}
	if flag.Lookup("only-headers").Value.String() == "true" {
		cfg.OnlyHeaders = *onlyHeaders
	}
	if flag.Lookup("only-body").Value.String() == "true" {
		cfg.OnlyBody = *onlyBody
	}
	if flag.Lookup("only-json").Value.String() == "true" {
		cfg.OnlyJSON = *onlyJSON
	}
	if flag.Lookup("skip-tls-verify").Value.String() == "true" {
		cfg.SkipTLSVerify = *skipTLSVerify
	}

	// Validate required configuration
	if cfg.TargetURL == "" {
		log.Fatal("TARGET_URL is required (set via environment variable or -url flag)")
	}

	printer := proxy.NewPrettyPrinter(os.Stdout, &cfg)
	handler := proxy.NewHandler(printer, &cfg)

	addr := fmt.Sprintf(":%s", cfg.Port)
	log.Printf("Starting pretty printing HTTP proxy on %s", addr)
	log.Printf("Proxying requests to: %s", cfg.TargetURL)

	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
