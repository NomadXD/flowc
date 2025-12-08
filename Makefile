# FlowC Makefile
# Envoy xDS Control Plane

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOVET=$(GOCMD) vet
GOFMT=$(GOCMD) fmt
GOMOD=$(GOCMD) mod

# Binary name and paths
BINARY_NAME=flowc
CMD_PATH=./cmd/flowc

# Ports
API_PORT?=8080
XDS_PORT?=18000

# Build flags
LDFLAGS=-ldflags "-s -w"

.PHONY: all build run clean test test-cover test-verbose lint fmt vet tidy help \
        envoy deploy validate health

# Default target
all: build

## Build

# Build the FlowC server binary
build:
	@echo "Building $(BINARY_NAME)..."
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME) $(CMD_PATH)
	@echo "Build complete: ./$(BINARY_NAME)"

# Build with race detector enabled
build-race:
	@echo "Building $(BINARY_NAME) with race detector..."
	$(GOBUILD) -race -o $(BINARY_NAME) $(CMD_PATH)

## Run

# Run the FlowC server
run: build
	@echo "Starting FlowC server (API: $(API_PORT), xDS: $(XDS_PORT))..."
	./$(BINARY_NAME)

# Run without rebuilding
run-only:
	@echo "Starting FlowC server..."
	./$(BINARY_NAME)

# Run with custom config file
run-config:
	@echo "Starting FlowC with custom config..."
	FLOWC_CONFIG=$(CONFIG) ./$(BINARY_NAME)

# Run with debug logging
run-debug:
	@echo "Starting FlowC with debug logging..."
	FLOWC_LOG_LEVEL=debug ./$(BINARY_NAME)

## Testing

# Run all tests
test:
	@echo "Running tests..."
	$(GOTEST) ./...

# Run tests with coverage
test-cover:
	@echo "Running tests with coverage..."
	$(GOTEST) -cover ./...

# Run tests with coverage report
test-cover-html:
	@echo "Generating coverage report..."
	$(GOTEST) -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# Run tests with verbose output
test-verbose:
	@echo "Running tests (verbose)..."
	$(GOTEST) -v ./...

# Run tests for a specific package (usage: make test-pkg PKG=./pkg/bundle/...)
test-pkg:
	@echo "Running tests for $(PKG)..."
	$(GOTEST) -v $(PKG)

# Run a specific test (usage: make test-run TEST=TestFunctionName PKG=./pkg/bundle/...)
test-run:
	@echo "Running test $(TEST) in $(PKG)..."
	$(GOTEST) -v -run $(TEST) $(PKG)

## Code Quality

# Format all Go files
fmt:
	@echo "Formatting code..."
	$(GOFMT) ./...

# Vet code for issues
vet:
	@echo "Vetting code..."
	$(GOVET) ./...

# Run linter (requires golangci-lint)
lint:
	@echo "Running linter..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
		exit 1; \
	fi

# Run all code quality checks
check: fmt vet lint

## Dependencies

# Tidy go modules
tidy:
	@echo "Tidying modules..."
	$(GOMOD) tidy

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download

# Verify dependencies
verify:
	@echo "Verifying dependencies..."
	$(GOMOD) verify

## Envoy

# Run Envoy connected to FlowC (requires Envoy installed)
envoy:
	@echo "Starting Envoy proxy..."
	@cd scripts && ./run-envoy.sh

## API Operations

# Deploy an API bundle (usage: make deploy BUNDLE=path/to/bundle.zip)
deploy:
	@if [ -z "$(BUNDLE)" ]; then \
		echo "Usage: make deploy BUNDLE=path/to/bundle.zip"; \
		exit 1; \
	fi
	@echo "Deploying $(BUNDLE)..."
	curl -X POST http://localhost:$(API_PORT)/api/v1/deployments \
		-F "file=@$(BUNDLE)" \
		-F "description=Deployed via Makefile"

# Validate a bundle without deploying (usage: make validate BUNDLE=path/to/bundle.zip)
validate:
	@if [ -z "$(BUNDLE)" ]; then \
		echo "Usage: make validate BUNDLE=path/to/bundle.zip"; \
		exit 1; \
	fi
	@echo "Validating $(BUNDLE)..."
	curl -X POST http://localhost:$(API_PORT)/api/v1/validate \
		-F "file=@$(BUNDLE)"

# List all deployments
list:
	@echo "Listing deployments..."
	curl -s http://localhost:$(API_PORT)/api/v1/deployments | jq .

# Get deployment stats
stats:
	@echo "Getting deployment stats..."
	curl -s http://localhost:$(API_PORT)/api/v1/deployments/stats | jq .

# Health check
health:
	@echo "Checking health..."
	curl -s http://localhost:$(API_PORT)/health | jq .

## Examples

# Create and deploy the example API
example-deploy:
	@echo "Creating example deployment bundle..."
	@cd examples/api-deployment && zip -r api-deployment.zip flowc.yaml openapi.yaml
	@echo "Deploying example..."
	curl -X POST http://localhost:$(API_PORT)/api/v1/deployments \
		-F "file=@examples/api-deployment/api-deployment.zip" \
		-F "description=Example API deployment"
	@rm examples/api-deployment/api-deployment.zip

## Cleanup

# Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -f $(BINARY_NAME)
	rm -f coverage.out coverage.html
	$(GOCMD) clean

# Clean and rebuild
rebuild: clean build

## Help

# Show help
help:
	@echo "FlowC - Envoy xDS Control Plane"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Build:"
	@echo "  build          Build the FlowC server binary"
	@echo "  build-race     Build with race detector enabled"
	@echo "  clean          Clean build artifacts"
	@echo "  rebuild        Clean and rebuild"
	@echo ""
	@echo "Run:"
	@echo "  run            Build and run the server"
	@echo "  run-only       Run without rebuilding"
	@echo "  run-debug      Run with debug logging"
	@echo "  run-config     Run with custom config (CONFIG=path/to/config.yaml)"
	@echo ""
	@echo "Testing:"
	@echo "  test           Run all tests"
	@echo "  test-cover     Run tests with coverage"
	@echo "  test-cover-html Generate HTML coverage report"
	@echo "  test-verbose   Run tests with verbose output"
	@echo "  test-pkg       Test specific package (PKG=./pkg/bundle/...)"
	@echo "  test-run       Run specific test (TEST=TestName PKG=./pkg/...)"
	@echo ""
	@echo "Code Quality:"
	@echo "  fmt            Format code"
	@echo "  vet            Vet code for issues"
	@echo "  lint           Run golangci-lint"
	@echo "  check          Run all code quality checks"
	@echo ""
	@echo "Dependencies:"
	@echo "  tidy           Tidy go modules"
	@echo "  deps           Download dependencies"
	@echo "  verify         Verify dependencies"
	@echo ""
	@echo "Envoy:"
	@echo "  envoy          Run Envoy connected to FlowC"
	@echo ""
	@echo "API Operations:"
	@echo "  deploy         Deploy API bundle (BUNDLE=path/to/bundle.zip)"
	@echo "  validate       Validate bundle (BUNDLE=path/to/bundle.zip)"
	@echo "  list           List all deployments"
	@echo "  stats          Get deployment statistics"
	@echo "  health         Health check"
	@echo ""
	@echo "Examples:"
	@echo "  example-deploy Deploy the example API"
	@echo ""
	@echo "Help:"
	@echo "  help           Show this help message"

