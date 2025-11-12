.PHONY: build install clean test run help

# Binary name
BINARY_NAME=git-status-walker

# Build directory
BUILD_DIR=./bin

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build the binary
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	@go build -o $(BUILD_DIR)/$(BINARY_NAME) .
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

install: ## Install to $GOPATH/bin
	@echo "Installing $(BINARY_NAME)..."
	@go install
	@echo "Installed to $$(go env GOPATH)/bin/$(BINARY_NAME)"

clean: ## Remove build artifacts
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@go clean
	@echo "Clean complete"

test: ## Run tests
	@echo "Running tests..."
	@go test -v ./...

run: build ## Build and run with default options
	@$(BUILD_DIR)/$(BINARY_NAME)

run-verbose: build ## Build and run with verbose output
	@$(BUILD_DIR)/$(BINARY_NAME) -verbose

run-all: build ## Build and run showing all branches
	@$(BUILD_DIR)/$(BINARY_NAME) -show-clean

fmt: ## Format code
	@echo "Formatting code..."
	@go fmt ./...

vet: ## Run go vet
	@echo "Running go vet..."
	@go vet ./...

lint: fmt vet ## Run formatters and linters
	@echo "Linting complete"

.DEFAULT_GOAL := help
