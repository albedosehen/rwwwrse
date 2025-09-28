GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=$(GOCMD) fmt
GOVET=$(GOCMD) vet
GOLINT=golangci-lint

BINARY_NAME=rwwwrse
BINARY_PATH=./bin/$(BINARY_NAME)
MAIN_PATH=./cmd/rwwwrse

DOCKER_IMAGE=rwwwrse
DOCKER_TAG=latest

VERSION ?= dev
COMMIT_SHA ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

LDFLAGS=-ldflags "-w -s -X main.version=$(VERSION) -X main.commit=$(COMMIT_SHA) -X main.buildDate=$(BUILD_DATE)"

.PHONY: help build clean test test-race test-coverage deps deps-update fmt vet lint \
        docker-build docker-run docker-clean run dev wire generate \
        install-tools check-tools pre-commit all

all: deps check build test

help:
	@echo "Available targets:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p bin
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_PATH) $(MAIN_PATH)

clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	@rm -rf bin/
	@rm -rf coverage/

run: build
	$(BINARY_PATH)

dev:
	@echo "Starting development server..."
	@air || $(GOBUILD) -o $(BINARY_PATH) $(MAIN_PATH) && $(BINARY_PATH)

wire:
	@echo "Generating wire code..."
	@wire ./internal/di

generate: wire
	@echo "Running go generate..."
	$(GOCMD) generate ./...

test:
	@echo "Running tests..."
	$(GOTEST) -v ./...

test-race:
	@echo "Running tests with race detection..."
	$(GOTEST) -race -v ./...

test-coverage:
	@echo "Running tests with coverage..."
	@mkdir -p coverage
	$(GOTEST) -coverprofile=coverage/coverage.out ./...
	$(GOCMD) tool cover -html=coverage/coverage.out -o coverage/coverage.html
	@echo "Coverage report generated: coverage/coverage.html"

test-benchmark:
	@echo "Running benchmark tests..."
	$(GOTEST) -bench=. -benchmem ./...

fmt:
	@echo "Formatting code..."
	$(GOFMT) ./...

vet:
	@echo "Running go vet..."
	$(GOVET) ./...

lint:
	@echo "Running golangci-lint..."
	@if command -v $(GOLINT) >/dev/null 2>&1; then \
		$(GOLINT) run; \
	else \
		echo "golangci-lint not found. Install it with: make install-tools"; \
	fi

check: fmt vet
	@echo "Code quality checks completed"

pre-commit: check test
	@echo "Pre-commit checks completed"

deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) verify

deps-update:
	@echo "Updating dependencies..."
	$(GOMOD) get -u ./...
	$(GOMOD) tidy

deps-tidy:
	@echo "Tidying dependencies..."
	$(GOMOD) tidy

docker-build:
	@echo "Building Docker image..."
	docker build \
		--build-arg VERSION=$(VERSION) \
		--build-arg COMMIT_SHA=$(COMMIT_SHA) \
		--build-arg BUILD_DATE=$(BUILD_DATE) \
		-t $(DOCKER_IMAGE):$(DOCKER_TAG) \
		-t $(DOCKER_IMAGE):$(VERSION) \
		.

docker-run:
	@echo "Running Docker container..."
	docker run --rm -it \
		-p 8080:8080 \
		-p 8443:8443 \
		-p 9090:9090 \
		$(DOCKER_IMAGE):$(DOCKER_TAG)

docker-clean:
	@echo "Cleaning Docker images..."
	docker rmi $(DOCKER_IMAGE):$(DOCKER_TAG) $(DOCKER_IMAGE):$(VERSION) 2>/dev/null || true

install-tools:
	@echo "Installing development tools..."
	$(GOGET) github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	$(GOGET) github.com/google/wire/cmd/wire@latest
	@echo "Development tools installed"

check-tools:
	@echo "Checking development tools..."
	@command -v wire >/dev/null 2>&1 || (echo "wire not found. Install with: make install-tools" && exit 1)
	@command -v golangci-lint >/dev/null 2>&1 || (echo "golangci-lint not found. Install with: make install-tools" && exit 1)
	@echo "All development tools are installed"

docs-serve:
	@echo "Serving documentation..."
	@command -v godoc >/dev/null 2>&1 || $(GOGET) golang.org/x/tools/cmd/godoc@latest
	godoc -http=:6060 -play

release-build:
	@echo "Building release binaries..."
	@mkdir -p bin/release
	
	# Linux amd64
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o bin/release/$(BINARY_NAME)-linux-amd64 $(MAIN_PATH)
	
	# Linux arm64
	GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o bin/release/$(BINARY_NAME)-linux-arm64 $(MAIN_PATH)
	
	# Windows amd64
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o bin/release/$(BINARY_NAME)-windows-amd64.exe $(MAIN_PATH)
	
	# macOS amd64
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o bin/release/$(BINARY_NAME)-darwin-amd64 $(MAIN_PATH)
	
	# macOS arm64
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o bin/release/$(BINARY_NAME)-darwin-arm64 $(MAIN_PATH)
	
	@echo "Release binaries built in bin/release/"

validate-config: build
	@echo "Validating configuration..."
	@RWWWRSE_TLS_AUTO_CERT=false \
		RWWWRSE_BACKENDS_ROUTES_EXAMPLE_URL=http://example.com \
		$(BINARY_PATH) --validate-config || echo "Add --validate-config flag to main.go for this to work"

example-config:
	@echo "Generating example configuration files..."
	@mkdir -p examples
	@cat > examples/config.yaml << 'EOF'
	server:
	  host: "0.0.0.0"
	  port: 8080
	  https_port: 8443
	  read_timeout: "30s"
	  write_timeout: "30s"
	  idle_timeout: "60s"
	  graceful_timeout: "30s"
	
	tls:
	  enabled: true
	  auto_cert: false  # Set to true for automatic Let's Encrypt certificates
	  email: "admin@example.com"  # Required when auto_cert is true
	  domains: []  # Required when auto_cert is true
	  cache_dir: "/tmp/certs"
	  staging: false
	  renew_before: "720h"
	
	backends:
	  routes:
	    api:
	      url: "http://api.example.com:3000"
	      health_path: "/health"
	      health_interval: "30s"
	      timeout: "30s"
	      max_idle_conns: 100
	      max_idle_per_host: 10
	      dial_timeout: "10s"
	    
	    web:
	      url: "http://web.example.com:8080"
	      health_path: "/status"
	      health_interval: "30s"
	      timeout: "30s"
	
	security:
	  rate_limit_enabled: true
	  cors_enabled: true
	  cors_origins: ["*"]
	  headers:
	    content_type_nosniff: true
	    frame_options: "DENY"
	    content_security_policy: "default-src 'self'"
	    strict_transport_security: "max-age=31536000; includeSubDomains"
	    referrer_policy: "strict-origin-when-cross-origin"
	
	logging:
	  level: "info"
	  format: "json"
	  output: "stdout"
	
	metrics:
	  enabled: true
	  port: 9090
	  path: "/metrics"
	
	health:
	  enabled: true
	  path: "/health"
	  timeout: "5s"
	  interval: "30s"
	  unhealthy_threshold: 3
	  healthy_threshold: 2
	
	ratelimit:
	  requests_per_second: 100.0
	  burst_size: 200
	  cleanup_interval: "10m"
	EOF
	@echo "Example configuration generated: examples/config.yaml"

quick-start: deps generate build
	@echo "Quick start completed. Run 'make dev' to start development server."

.DEFAULT_GOAL := help