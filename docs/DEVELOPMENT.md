# Development Guide for rwwwrse

This guide provides comprehensive instructions for setting up a local development environment for rwwwrse, including support for testing multiple deployment scenarios, development workflows, and debugging techniques.

## Overview

The rwwwrse development environment supports:

- **Local development** with hot reloading and debugging
- **Multi-deployment testing** to validate deployment scenarios locally
- **Integration testing** with backend services and dependencies
- **Performance profiling** and optimization
- **CI/CD pipeline testing** before deployment
- **Cross-platform development** on Windows, macOS, and Linux

## Prerequisites

### Required Tools

```bash
# Go development environment
go version # >= 1.21

# Container tools
docker --version
docker-compose --version

# Kubernetes tools (optional)
kubectl version --client
minikube version

# Development utilities
git --version
make --version
curl --version

# Code quality tools
golangci-lint --version
```

### Installation Script

```bash
#!/bin/bash
# setup-dev-environment.sh

set -e

echo "Setting up rwwwrse development environment"

# Install Go (if not installed)
if ! command -v go >/dev/null 2>&1; then
    echo "Installing Go..."
    curl -L https://golang.org/dl/go1.21.0.linux-amd64.tar.gz | sudo tar -C /usr/local -xz
    echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
    source ~/.bashrc
fi

# Install Docker (if not installed)
if ! command -v docker >/dev/null 2>&1; then
    echo "Installing Docker..."
    curl -fsSL https://get.docker.com | sh
    sudo usermod -aG docker $USER
fi

# Install development tools
echo "Installing development tools..."

# golangci-lint
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.54.2

# air for hot reloading
go install github.com/cosmtrek/air@latest

# dlv for debugging
go install github.com/go-delve/delve/cmd/dlv@latest

# godoc for documentation
go install golang.org/x/tools/cmd/godoc@latest

# Create development directories
mkdir -p ~/go/src/rwwwrse
mkdir -p ~/.config/rwwwrse

echo "Development environment setup completed"
echo "Please log out and back in for Docker group changes to take effect"
```

## Local Development Setup

### Project Structure

```bash
rwwwrse/
├── cmd/
│   └── rwwwrse/
│       └── main.go
├── internal/
│   ├── config/
│   ├── proxy/
│   ├── middleware/
│   └── health/
├── pkg/
│   └── metrics/
├── test/
│   ├── integration/
│   ├── e2e/
│   └── fixtures/
├── examples/
│   ├── docker-compose/
│   ├── kubernetes/
│   ├── cloud-specific/
│   └── bare-metal/
├── docs/
├── scripts/
├── .air.toml
├── Makefile
├── docker-compose.dev.yml
└── go.mod
```

### Development Configuration

#### Air Configuration for Hot Reloading

```toml
# .air.toml
root = "."
testdata_dir = "testdata"
tmp_dir = "tmp"

[build]
  args_bin = []
  bin = "./tmp/main"
  cmd = "go build -o ./tmp/main ./cmd/rwwwrse"
  delay = 1000
  exclude_dir = ["assets", "tmp", "vendor", "testdata", "examples"]
  exclude_file = []
  exclude_regex = ["_test.go"]
  exclude_unchanged = false
  follow_symlink = false
  full_bin = ""
  include_dir = []
  include_ext = ["go", "tpl", "tmpl", "html"]
  include_file = []
  kill_delay = "0s"
  log = "build-errors.log"
  rerun = false
  rerun_delay = 500
  send_interrupt = false
  stop_on_root = false

[color]
  app = ""
  build = "yellow"
  main = "magenta"
  runner = "green"
  watcher = "cyan"

[log]
  main_only = false
  time = false

[misc]
  clean_on_exit = false

[screen]
  clear_on_rebuild = false
  keep_scroll = true
```

#### Development Docker Compose

```yaml
# docker-compose.dev.yml
version: '3.8'

services:
  # Backend services for testing
  backend1:
    image: httpd:alpine
    volumes:
      - ./test/fixtures/backend1:/usr/local/apache2/htdocs/
    ports:
      - "8081:80"
    environment:
      - SERVER_NAME=backend1

  backend2:
    image: nginx:alpine
    volumes:
      - ./test/fixtures/backend2:/usr/share/nginx/html/
    ports:
      - "8082:80"
    environment:
      - SERVER_NAME=backend2

  # Database for testing
  postgres:
    image: postgres:15-alpine
    environment:
      - POSTGRES_DB=testdb
      - POSTGRES_USER=testuser
      - POSTGRES_PASSWORD=testpass
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data

  # Redis for caching tests
  redis:
    image: redis:alpine
    ports:
      - "6379:6379"

  # Monitoring services
  prometheus:
    image: prom/prometheus:latest
    ports:
      - "9090:9090"
    volumes:
      - ./examples/docker-compose/production/monitoring/prometheus.yml:/etc/prometheus/prometheus.yml

  grafana:
    image: grafana/grafana:latest
    ports:
      - "3000:3000"
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=admin
    volumes:
      - grafana_data:/var/lib/grafana

  # Log aggregation
  elasticsearch:
    image: docker.elastic.co/elasticsearch/elasticsearch:8.8.0
    environment:
      - discovery.type=single-node
      - xpack.security.enabled=false
    ports:
      - "9200:9200"

  kibana:
    image: docker.elastic.co/kibana/kibana:8.8.0
    ports:
      - "5601:5601"
    environment:
      - ELASTICSEARCH_HOSTS=http://elasticsearch:9200

volumes:
  postgres_data:
  grafana_data:
```

#### Development Environment Variables

```bash
# .env.dev
# Development configuration for rwwwrse

# Basic configuration
RWWWRSE_PORT=8080
RWWWRSE_BACKENDS=http://localhost:8081,http://localhost:8082
RWWWRSE_LOG_LEVEL=debug
RWWWRSE_ENABLE_DEBUG=true

# TLS configuration (disabled for local development)
RWWWRSE_ENABLE_TLS=false
RWWWRSE_TLS_CERT_FILE=./test/fixtures/ssl/server.crt
RWWWRSE_TLS_KEY_FILE=./test/fixtures/ssl/server.key

# Rate limiting (relaxed for development)
RWWWRSE_ENABLE_RATE_LIMITING=true
RWWWRSE_RATE_LIMIT=1000

# Health check configuration
RWWWRSE_HEALTH_CHECK_PATH=/health
RWWWRSE_HEALTH_CHECK_INTERVAL=30s

# Metrics and monitoring
RWWWRSE_ENABLE_METRICS=true
RWWWRSE_METRICS_PORT=9091
RWWWRSE_METRICS_PATH=/metrics

# Development-specific features
RWWWRSE_ENABLE_PROFILING=true
RWWWRSE_PROFILING_PORT=6060

# CORS (permissive for development)
RWWWRSE_ENABLE_CORS=true
RWWWRSE_CORS_ALLOWED_ORIGINS=*
```

### Development Makefile

```makefile
# Makefile
.PHONY: help build test run clean dev deps lint fmt vet security

# Default target
help: ## Show this help message
 @echo 'Usage: make [target]'
 @echo ''
 @echo 'Targets:'
 @awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Development
dev: ## Start development environment with hot reloading
 @echo "Starting development environment..."
 docker-compose -f docker-compose.dev.yml up -d
 @echo "Backend services started, launching rwwwrse with hot reloading..."
 air

dev-stop: ## Stop development environment
 docker-compose -f docker-compose.dev.yml down

dev-logs: ## Show development environment logs
 docker-compose -f docker-compose.dev.yml logs -f

# Build
build: ## Build the rwwwrse binary
 go build -o bin/rwwwrse ./cmd/rwwwrse

build-race: ## Build with race detector
 go build -race -o bin/rwwwrse-race ./cmd/rwwwrse

build-debug: ## Build with debug symbols
 go build -gcflags="all=-N -l" -o bin/rwwwrse-debug ./cmd/rwwwrse

# Testing
test: ## Run unit tests
 go test -v ./...

test-race: ## Run tests with race detector
 go test -race -v ./...

test-coverage: ## Run tests with coverage
 go test -coverprofile=coverage.out ./...
 go tool cover -html=coverage.out -o coverage.html

test-integration: ## Run integration tests
 go test -tags=integration -v ./test/integration/...

test-e2e: ## Run end-to-end tests
 go test -tags=e2e -v ./test/e2e/...

benchmark: ## Run benchmarks
 go test -bench=. -benchmem ./...

# Code quality
deps: ## Download dependencies
 go mod download
 go mod tidy

lint: ## Run linter
 golangci-lint run

fmt: ## Format code
 go fmt ./...
 goimports -w .

vet: ## Run go vet
 go vet ./...

security: ## Run security scan
 gosec ./...

# Docker
docker-build: ## Build Docker image
 docker build -t rwwwrse:dev .

docker-run: ## Run Docker container
 docker run -p 8080:8080 --env-file .env.dev rwwwrse:dev

# Deployment testing
test-docker-compose: ## Test Docker Compose deployments
 ./scripts/test-deployments.sh docker-compose

test-kubernetes: ## Test Kubernetes deployments
 ./scripts/test-deployments.sh kubernetes

test-all-deployments: ## Test all deployment scenarios
 ./scripts/test-deployments.sh all

# Documentation
docs: ## Generate documentation
 godoc -http=:6060

docs-serve: ## Serve documentation
 @echo "Documentation available at http://localhost:6060"
 godoc -http=:6060

# Cleanup
clean: ## Clean build artifacts
 rm -rf bin/ tmp/ coverage.out coverage.html
 docker-compose -f docker-compose.dev.yml down -v
 docker system prune -f

# Debugging
debug: ## Start with debugger
 dlv debug ./cmd/rwwwrse

debug-test: ## Debug tests
 dlv test ./internal/proxy

# Performance
profile-cpu: ## Run CPU profiling
 go tool pprof http://localhost:6060/debug/pprof/profile

profile-mem: ## Run memory profiling
 go tool pprof http://localhost:6060/debug/pprof/heap

profile-trace: ## Run execution tracing
 curl http://localhost:6060/debug/pprof/trace?seconds=30 > trace.out
 go tool trace trace.out
```

## Development Workflow

### Daily Development Cycle

```bash
#!/bin/bash
# daily-dev-cycle.sh

echo "Starting daily development cycle for rwwwrse"

# 1. Update dependencies and check for security issues
echo "1. Updating dependencies..."
make deps
make security

# 2. Run code quality checks
echo "2. Running code quality checks..."
make fmt
make lint
make vet

# 3. Run tests
echo "3. Running tests..."
make test
make test-race

# 4. Start development environment
echo "4. Starting development environment..."
make dev
```

### Feature Development Workflow

```bash
#!/bin/bash
# feature-development.sh

FEATURE_NAME=${1:-"new-feature"}

echo "Starting feature development: $FEATURE_NAME"

# Create feature branch
git checkout -b "feature/$FEATURE_NAME"

# Set up development environment
make dev-stop
make clean
make dev

echo "Development environment ready for feature: $FEATURE_NAME"
echo "Available services:"
echo "  - rwwwrse: http://localhost:8080"
echo "  - Backend 1: http://localhost:8081"
echo "  - Backend 2: http://localhost:8082"
echo "  - Prometheus: http://localhost:9090"
echo "  - Grafana: http://localhost:3000"
echo "  - Profiling: http://localhost:6060"
```

## Testing Multiple Deployment Scenarios

### Deployment Testing Script

```bash
#!/bin/bash
# scripts/test-deployments.sh

DEPLOYMENT_TYPE=${1:-"all"}
TEST_TIMEOUT=300
HEALTH_CHECK_URL="http://localhost:8080/health"

set -e

test_docker_compose() {
    local example_dir=$1
    echo "Testing Docker Compose example: $example_dir"
    
    cd "examples/docker-compose/$example_dir"
    
    # Start services
    docker-compose up -d
    
    # Wait for services to be ready
    local attempt=1
    while [ $attempt -le 30 ]; do
        if curl -sf "$HEALTH_CHECK_URL" >/dev/null 2>&1; then
            echo "  ✓ Health check passed for $example_dir"
            break
        fi
        echo "  Waiting for services to be ready (attempt $attempt/30)"
        sleep 10
        ((attempt++))
    done
    
    if [ $attempt -gt 30 ]; then
        echo "  ✗ Health check failed for $example_dir"
        docker-compose logs rwwwrse
        docker-compose down
        return 1
    fi
    
    # Run basic functionality tests
    test_basic_functionality "$example_dir"
    
    # Cleanup
    docker-compose down
    cd - >/dev/null
    
    echo "  ✓ Docker Compose test completed for $example_dir"
}

test_kubernetes() {
    local example_dir=$1
    echo "Testing Kubernetes example: $example_dir"
    
    cd "examples/kubernetes/$example_dir"
    
    # Check if minikube is running
    if ! minikube status >/dev/null 2>&1; then
        echo "  Starting minikube..."
        minikube start
    fi
    
    # Apply manifests
    kubectl apply -f .
    
    # Wait for deployment
    kubectl wait --for=condition=available --timeout=300s deployment/rwwwrse
    
    # Port forward for testing
    kubectl port-forward service/rwwwrse 8080:80 &
    PF_PID=$!
    
    # Wait for port forward
    sleep 10
    
    # Test basic functionality
    if test_basic_functionality "$example_dir"; then
        echo "  ✓ Kubernetes test passed for $example_dir"
    else
        echo "  ✗ Kubernetes test failed for $example_dir"
        kubectl logs deployment/rwwwrse
    fi
    
    # Cleanup
    kill $PF_PID 2>/dev/null || true
    kubectl delete -f .
    cd - >/dev/null
}

test_basic_functionality() {
    local example_name=$1
    
    echo "    Testing basic functionality for $example_name"
    
    # Health check
    if ! curl -sf "$HEALTH_CHECK_URL" >/dev/null; then
        echo "    ✗ Health check failed"
        return 1
    fi
    
    # Basic proxy functionality
    if ! curl -sf "http://localhost:8080/" >/dev/null; then
        echo "    ✗ Basic proxy functionality failed"
        return 1
    fi
    
    # Metrics endpoint (if enabled)
    if curl -sf "http://localhost:9091/metrics" >/dev/null 2>&1; then
        echo "    ✓ Metrics endpoint accessible"
    fi
    
    echo "    ✓ Basic functionality tests passed"
    return 0
}

case $DEPLOYMENT_TYPE in
    docker-compose)
        echo "Testing Docker Compose deployments..."
        for example in simple microservices development production; do
            test_docker_compose "$example"
        done
        ;;
        
    kubernetes)
        echo "Testing Kubernetes deployments..."
        for example in minikube; do
            test_kubernetes "$example"
        done
        ;;
        
    all)
        echo "Testing all deployment scenarios..."
        $0 docker-compose
        $0 kubernetes
        ;;
        
    *)
        echo "Usage: $0 {docker-compose|kubernetes|all}"
        exit 1
        ;;
esac

echo "All deployment tests completed successfully"
```

### Integration Testing Framework

```go
// test/integration/proxy_test.go
package integration

import (
    "net/http"
    "net/http/httptest"
    "testing"
    "time"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestProxyIntegration(t *testing.T) {
    tests := []struct {
        name           string
        backends       []string
        expectedStatus int
        expectedBody   string
    }{
        {
            name:           "single backend",
            backends:       []string{"http://backend1:8080"},
            expectedStatus: http.StatusOK,
            expectedBody:   "backend1 response",
        },
        {
            name:           "multiple backends",
            backends:       []string{"http://backend1:8080", "http://backend2:8080"},
            expectedStatus: http.StatusOK,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Set up test backends
            backend1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                w.WriteHeader(http.StatusOK)
                w.Write([]byte("backend1 response"))
            }))
            defer backend1.Close()

            backend2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                w.WriteHeader(http.StatusOK)
                w.Write([]byte("backend2 response"))
            }))
            defer backend2.Close()

            // Start proxy with test configuration
            proxy := startTestProxy(t, []string{backend1.URL, backend2.URL})
            defer proxy.Close()

            // Test proxy functionality
            resp, err := http.Get(proxy.URL)
            require.NoError(t, err)
            defer resp.Body.Close()

            assert.Equal(t, tt.expectedStatus, resp.StatusCode)
        })
    }
}

func TestHealthCheck(t *testing.T) {
    proxy := startTestProxy(t, []string{"http://localhost:8081"})
    defer proxy.Close()

    resp, err := http.Get(proxy.URL + "/health")
    require.NoError(t, err)
    defer resp.Body.Close()

    assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestMetrics(t *testing.T) {
    proxy := startTestProxy(t, []string{"http://localhost:8081"})
    defer proxy.Close()

    // Make some requests to generate metrics
    for i := 0; i < 10; i++ {
        http.Get(proxy.URL)
    }

    // Check metrics endpoint
    resp, err := http.Get(proxy.URL + "/metrics")
    require.NoError(t, err)
    defer resp.Body.Close()

    assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func startTestProxy(t *testing.T, backends []string) *httptest.Server {
    // Implementation would start rwwwrse with test configuration
    // This is a simplified version for illustration
    return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        w.Write([]byte("test response"))
    }))
}
```

### End-to-End Testing

```go
// test/e2e/deployment_test.go
//go:build e2e

package e2e

import (
    "os/exec"
    "testing"
    "time"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestDockerComposeDeployment(t *testing.T) {
    // Start Docker Compose environment
    cmd := exec.Command("docker-compose", "-f", "examples/docker-compose/simple/docker-compose.yml", "up", "-d")
    require.NoError(t, cmd.Run())

    defer func() {
        exec.Command("docker-compose", "-f", "examples/docker-compose/simple/docker-compose.yml", "down").Run()
    }()

    // Wait for services to start
    time.Sleep(30 * time.Second)

    // Test health endpoint
    resp, err := http.Get("http://localhost:8080/health")
    require.NoError(t, err)
    defer resp.Body.Close()

    assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestKubernetesDeployment(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping Kubernetes e2e test in short mode")
    }

    // Apply Kubernetes manifests
    cmd := exec.Command("kubectl", "apply", "-f", "examples/kubernetes/minikube/")
    require.NoError(t, cmd.Run())

    defer func() {
        exec.Command("kubectl", "delete", "-f", "examples/kubernetes/minikube/").Run()
    }()

    // Wait for deployment
    cmd = exec.Command("kubectl", "wait", "--for=condition=available", "--timeout=300s", "deployment/rwwwrse")
    require.NoError(t, cmd.Run())

    // Port forward and test
    portForwardCmd := exec.Command("kubectl", "port-forward", "service/rwwwrse", "8080:80")
    require.NoError(t, portForwardCmd.Start())

    defer portForwardCmd.Process.Kill()

    time.Sleep(10 * time.Second)

    // Test functionality
    resp, err := http.Get("http://localhost:8080/health")
    require.NoError(t, err)
    defer resp.Body.Close()

    assert.Equal(t, http.StatusOK, resp.StatusCode)
}
```

## Debugging and Troubleshooting

### Debug Configuration

```bash
# Debug with delve
make debug

# Debug specific package
dlv test ./internal/proxy

# Remote debugging (for containerized development)
dlv debug --listen=:2345 --headless=true --api-version=2 ./cmd/rwwwrse
```

### Performance Profiling

```bash
#!/bin/bash
# scripts/profile.sh

PROFILE_TYPE=${1:-"cpu"}
DURATION=${2:-"30s"}

echo "Starting $PROFILE_TYPE profiling for $DURATION"

case $PROFILE_TYPE in
    cpu)
        go tool pprof "http://localhost:6060/debug/pprof/profile?seconds=${DURATION%s}"
        ;;
    memory)
        go tool pprof "http://localhost:6060/debug/pprof/heap"
        ;;
    goroutine)
        go tool pprof "http://localhost:6060/debug/pprof/goroutine"
        ;;
    trace)
        curl "http://localhost:6060/debug/pprof/trace?seconds=${DURATION%s}" > trace.out
        go tool trace trace.out
        ;;
    *)
        echo "Usage: $0 {cpu|memory|goroutine|trace} [duration]"
        exit 1
        ;;
esac
```

### Log Analysis Tools

```bash
#!/bin/bash
# scripts/analyze-logs.sh

LOG_SOURCE=${1:-"docker"}
ANALYSIS_TYPE=${2:-"errors"}

echo "Analyzing logs from $LOG_SOURCE for $ANALYSIS_TYPE"

case $LOG_SOURCE in
    docker)
        case $ANALYSIS_TYPE in
            errors)
                docker-compose -f docker-compose.dev.yml logs rwwwrse | grep -i error
                ;;
            performance)
                docker-compose -f docker-compose.dev.yml logs rwwwrse | grep -E "latency|duration|timeout"
                ;;
            requests)
                docker-compose -f docker-compose.dev.yml logs rwwwrse | grep -E "GET|POST|PUT|DELETE"
                ;;
        esac
        ;;
    kubernetes)
        kubectl logs deployment/rwwwrse | grep -i "$ANALYSIS_TYPE"
        ;;
    systemd)
        journalctl -u rwwwrse | grep -i "$ANALYSIS_TYPE"
        ;;
esac
```

## Development Tools and Utilities

### Code Generation

```bash
#!/bin/bash
# scripts/generate.sh

echo "Generating code for rwwwrse"

# Generate mocks
go generate ./...

# Generate documentation
go doc -all > docs/api.txt

# Update dependencies
go mod tidy

# Verify everything builds
go build ./...

echo "Code generation completed"
```

### Testing Utilities

```go
// test/testutil/helpers.go
package testutil

import (
    "net/http"
    "net/http/httptest"
    "testing"
    "time"
)

// NewTestBackend creates a test HTTP server for backend simulation
func NewTestBackend(t *testing.T, response string, delay time.Duration) *httptest.Server {
    return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if delay > 0 {
            time.Sleep(delay)
        }
        w.WriteHeader(http.StatusOK)
        w.Write([]byte(response))
    }))
}

// NewFailingBackend creates a test HTTP server that returns errors
func NewFailingBackend(t *testing.T, statusCode int) *httptest.Server {
    return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(statusCode)
    }))
}

// WaitForHealthy waits for a service to become healthy
func WaitForHealthy(url string, timeout time.Duration) error {
    client := &http.Client{Timeout: 5 * time.Second}
    deadline := time.Now().Add(timeout)
    
    for time.Now().Before(deadline) {
        resp, err := client.Get(url)
        if err == nil && resp.StatusCode == http.StatusOK {
            resp.Body.Close()
            return nil
        }
        if resp != nil {
            resp.Body.Close()
        }
        time.Sleep(2 * time.Second)
    }
    
    return fmt.Errorf("service did not become healthy within %v", timeout)
}
```

### Configuration Management for Development

```go
// internal/config/dev.go
//go:build dev

package config

import (
    "fmt"
    "os"
)

// DevConfig represents development-specific configuration
type DevConfig struct {
    *Config
    HotReload     bool   `env:"RWWWRSE_HOT_RELOAD" envDefault:"true"`
    DebugMode     bool   `env:"RWWWRSE_DEBUG_MODE" envDefault:"true"`
    ProfilePort   int    `env:"RWWWRSE_PROFILE_PORT" envDefault:"6060"`
    TestBackends  string `env:"RWWWRSE_TEST_BACKENDS"`
}

// LoadDevConfig loads development-specific configuration
func LoadDevConfig() (*DevConfig, error) {
    baseConfig, err := Load()
    if err != nil {
        return nil, fmt.Errorf("failed to load base config: %w", err)
    }
    
    devConfig := &DevConfig{
        Config: baseConfig,
    }
    
    if err := env.Parse(devConfig); err != nil {
        return nil, fmt.Errorf("failed to parse dev config: %w", err)
    }
    
    // Override with development-friendly defaults
    if devConfig.LogLevel == "" {
        devConfig.LogLevel = "debug"
    }
    
    if devConfig.TestBackends != "" {
        devConfig.Backends = strings.Split(devConfig.TestBackends, ",")
    }
    
    return devConfig, nil
}
```

## Continuous Integration in Development

### Pre-commit Hooks

```bash
#!/bin/bash
# .git/hooks/pre-commit

set -e

echo "Running pre-commit checks..."

# Format code
make fmt

# Run linter
make lint

# Run tests
make test

# Check for security issues
make security

echo "Pre-commit checks passed"
```

### Local CI Pipeline Simulation

```bash
#!/bin/bash
# scripts/simulate-ci.sh

echo "Simulating CI pipeline locally"

# Clean environment
make clean

# Install dependencies
make deps

# Code quality checks
echo "1. Code quality checks..."
make fmt
make lint
make vet
make security

# Testing
echo "2. Running tests..."
make test
make test-race
make test-coverage

# Build
echo "3. Building..."
make build
make docker-build

# Integration tests
echo "4. Integration tests..."
make test-integration

# Deployment tests
echo "5. Deployment tests..."
make test-docker-compose

echo "CI pipeline simulation completed successfully"
```

## IDE Configuration

### VS Code Configuration

```json
// .vscode/settings.json
{
    "go.lintTool": "golangci-lint",
    "go.lintFlags": ["--fast"],
    "go.testFlags": ["-v", "-race"],
    "go.buildTags": "dev",
    "go.testTags": "integration,e2e",
    "go.coverOnSave": true,
    "go.coverageDecorator": {
        "type": "gutter"
    },
    "files.exclude": {
        "**/tmp": true,
        "**/bin": true,
        "**/.air": true
    }
}
```

```json
// .vscode/launch.json
{
    "version": "0.2.0",
    "configurations": [
        {
            "name": "Debug rwwwrse",
            "type": "go",
            "request": "launch",
            "mode": "debug",
            "program": "${workspaceFolder}/cmd/rwwwrse",
            "envFile": "${workspaceFolder}/.env.dev",
            "args": []
        },
        {
            "name": "Debug tests",
            "type": "go",
            "request": "launch",
            "mode": "test",
            "program": "${workspaceFolder}/internal/proxy",
            "buildFlags": "-tags=dev"
        }
    ]
}
```

### GoLand Configuration

```xml
<!-- .idea/runConfigurations/rwwwrse_dev.xml -->
<component name="ProjectRunConfigurationManager">
  <configuration default="false" name="rwwwrse-dev" type="GoApplicationRunConfiguration" factoryName="Go Application">
    <module name="rwwwrse" />
    <working_directory value="$PROJECT_DIR$" />
    <go_parameters value="-tags=dev" />
    <kind value="FILE" />
    <directory value="$PROJECT_DIR$" />
    <filePath value="$PROJECT_DIR$/cmd/rwwwrse/main.go" />
    <env name="RWWWRSE_LOG_LEVEL" value="debug" />
    <env name="RWWWRSE_ENABLE_DEBUG" value="true" />
    <method v="2" />
  </configuration>
</component>
```

## Performance Testing and Optimization

### Load Testing Scripts

```bash
#!/bin/bash
# scripts/load-test.sh

CONCURRENT_USERS=${1:-10}
DURATION=${2:-30s}
TARGET_URL=${3:-"http://localhost:8080"}

echo "Running load test with $CONCURRENT_USERS users for $DURATION"

# Using hey (HTTP load testing tool)
if command -v hey >/dev/null 2>&1; then
    hey -c $CONCURRENT_USERS -z $DURATION $TARGET_URL
elif command -v ab >/dev/null 2>&1; then
    # Fallback to Apache Bench
    REQUESTS=$((CONCURRENT_USERS * 100))
    ab -n $REQUESTS -c $CONCURRENT_USERS $TARGET_URL
else
    echo "No load testing tool found. Install hey or apache2-utils"
    exit 1
fi
```

### Benchmark Tests

```go
// internal/proxy/proxy_bench_test.go
package proxy

import (
    "net/http"
    "net/http/httptest"
    "testing"
)

func BenchmarkProxy(b *testing.B) {
    backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        w.Write([]byte("OK"))
    }))
    defer backend.Close()

    proxy := NewProxy([]string{backend.URL})
    
    b.ResetTimer()
    b.RunParallel(func(pb *testing.PB) {
        for pb.Next() {
            req := httptest.NewRequest("GET", "/", nil)
            w := httptest.NewRecorder()
            proxy.ServeHTTP(w, req)
        }
    })
}

func BenchmarkProxyWithMultipleBackends(b *testing.B) {
    backends := make([]*httptest.Server, 5)
    backendUrls := make([]string, 5)
    
    for i := 0; i < 5; i++ {
        backends[i] = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            w.WriteHeader(http.StatusOK)
            w.Write([]byte("OK"))
        }))
        backendUrls[i] = backends[i].URL
    }
    
    defer func() {
        for _, backend := range backends {
            backend.Close()
        }
    }()

    proxy := NewProxy(backendUrls)
    
    b.ResetTimer()
    b.RunParallel(func(pb *testing.PB) {
        for pb.Next() {
            req := httptest.NewRequest("GET", "/", nil)
            w := httptest.NewRecorder()
            proxy.ServeHTTP(w, req)
        }
    })
}
```

## Documentation and Help

### Development Documentation

```bash
#!/bin/bash
# scripts/dev-help.sh

cat << 'EOF'
rwwwrse Development Environment Help

Quick Start:
  make dev              Start development environment
  make test             Run tests
  make build            Build binary

Development Commands:
  make dev              Start with hot reloading
  make debug            Start with debugger
  make profile-cpu      Run CPU profiling
  make docs             Generate documentation

Testing Commands:
  make test             Unit tests
  make test-race        Tests with race detector
  make test-integration Integration tests
  make test-e2e         End-to-end tests

Deployment Testing:
  make test-docker-compose    Test Docker Compose examples
  make test-kubernetes        Test Kubernetes examples
  make test-all-deployments   Test all deployment scenarios

Useful URLs (when dev environment is running):
  Application:     http://localhost:8080
  Health Check:    http://localhost:8080/health
  Metrics:         http://localhost:9091/metrics
  Profiling:       http://localhost:6060/debug/pprof/
  Prometheus:      http://localhost:9090
  Grafana:         http://localhost:3000 (admin/admin)

Environment Files:
  .env.dev             Development configuration
  docker-compose.dev.yml   Development services

EOF
```

## Related Documentation

- [Configuration Guide](CONFIGURATION.md) - Environment-specific configuration
- [Operations Guide](OPERATIONS.md) - Monitoring and troubleshooting
- [Migration Guide](MIGRATION.md) - Upgrade procedures
- [SSL/TLS Guide](SSL-TLS.md) - Certificate management
- [Deployment Guide](DEPLOYMENT.md) - Platform-specific deployments
- [Docker Compose Examples](../examples/docker-compose/) - Container deployment examples
- [Kubernetes Examples](../examples/kubernetes/) - Orchestration examples
- [CI/CD Examples](../examples/cicd/) - Automated deployment pipelines
