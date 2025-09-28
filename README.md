# rwwwrse  - Modern Go Reverse Proxy Server

[![Go Version](https://img.shields.io/badge/Go-1.21+-blue.svg)](https://golang.org) [![Docker](https://img.shields.io/badge/Docker-Ready-blue.svg)](Dockerfile) [![Kubernetes](https://img.shields.io/badge/Kubernetes-Ready-blue.svg)](examples/kubernetes/) [![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

A reverse proxy server built with modern Go, automatic TLS certificate management (Let's Encrypt), structured logging, and observability.

## Features

- **Performant**: Built with Go's excellent concurrency primitives and optimized for high throughput
- **Automatic TLS (Let's Encrypt)**: Let's Encrypt integration with automatic certificate renewal and ACME challenge handling
- **Observability**: Structured logging with slog, Prometheus metrics, health checks, and distributed tracing
- **Configuration**: Environment-variable based configuration with validation and hot reloading
- **Modern Architecture**: Clean architecture with dependency injection using Google Wire
- **Production Ready**: Comprehensive error handling, graceful shutdown, circuit breakers, and rate limiting
- **Extensible**: Interface-driven design for easy testing, mocking, and extensibility
- **Cloud Native**: Docker and Kubernetes ready with comprehensive deployment examples

## Quick Start

Choose your deployment path:

### Option 1: Docker (Recommended for testing)

```bash
# Quick test with Docker
docker run -p 8080:8080 -p 8443:8443 \
  -e RWWWRSE_BACKENDS_ROUTES='{"localhost":{"url":"http://httpbin.org"}}' \
  ghcr.io/albedosehen/rwwwrse:latest

# Test the proxy
curl http://localhost:8080/get
```

### Option 2: Local Build

```bash
# Clone and build
git clone https://github.com/albedosehen/rwwwrse.git
cd rwwwrse
make build

# Configure basic backend
export RWWWRSE_BACKENDS_ROUTES='{"localhost":{"url":"http://httpbin.org"}}'
./bin/rwwwrse
```

### Option 3: Docker Compose (Recommended for development)

```bash
git clone https://github.com/albedosehen/rwwwrse.git

cd rwwwrse/examples/docker-compose/development

docker-compose up
```

## Deployment Scenarios

rwwwrse supports multiple deployment scenarios. Choose the one that fits your needs:

### Local Development

#### Google Wire

You will need [`google/wire`](https://github.com/google/wire) for dependency injection.

```bash
# Install Wire for dependency injection
go install github.com/google/wire/cmd/wire@latest
```

#### Further development setups

- **[Docker Compose Development](examples/docker-compose/development/)** - Full local environment with mock services
- **[Local Build](docs/DEVELOPMENT.md#local-setup)** - Direct Go build for development
- **[Minikube](examples/kubernetes/minikube/)** - Local Kubernetes testing

### Production Deployments

#### Container Orchestration

- **[Docker Compose Production](examples/docker-compose/production/)** - Production-ready Docker Compose with monitoring
- **[Kubernetes](examples/kubernetes/)** - Complete Kubernetes manifests
  - [Cloud Providers (EKS, GKE, AKS)](examples/kubernetes/cloud/)
  - [Ingress Controllers (Nginx, Traefik, Istio)](examples/kubernetes/ingress/)

#### Cloud Platforms

- **[AWS ECS/Fargate](examples/cloud-specific/aws/)** - Amazon container services
- **[Azure Container Instances](examples/cloud-specific/azure/)** - Azure container service / AKS
- **[Google Cloud Run](examples/cloud-specific/gcp/)** - Google container service

#### Traditional Infrastructure

- **[Bare Metal/VPS](examples/bare-metal/)** - systemd services and traditional deployments
- **[Behind Existing Proxies](examples/nginx/)** - Integration with nginx/apache

### CI/CD Integration

- **[GitHub Actions](examples/cicd/github/)** - Automated testing and deployment
- **[GitLab CI](examples/cicd/gitlab/)** - GitLab pipelines
- **[Jenkins](examples/cicd/jenkins/)** - Jenkins pipeline examples

## Rwwrse Architecture Overview

```text
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   Client        │────│   rwwwrse        │────│   Backend       │
│   Requests      │    │   Proxy          │    │   Services      │
└─────────────────┘    └──────────────────┘    └─────────────────┘
                              │
                    ┌─────────┼─────────┐
                    │         │         │
              ┌──────▼──┐ ┌────▼────┐ ┌──▼─────┐
              │   TLS   │ │ Health  │ │ Metrics│
              │ Manager │ │ Monitor │ │& Logs  │
              └─────────┘ └─────────┘ └────────┘
```

### Core Components

- **[Proxy Engine](docs/ARCHITECTURE.md#proxy-engine)** - Host-based routing with health checks
- **[TLS Manager](docs/SSL-TLS.md)** - Automatic certificate management with Let's Encrypt
- **[Middleware Pipeline](docs/ARCHITECTURE.md#middleware)** - Security, rate limiting, CORS, logging
- **[Observability Stack](docs/OPERATIONS.md#monitoring)** - Metrics, logging, health checks

## Configuration

rwwwrse uses environment variables with the `RWWWRSE_` prefix. Here are the essential settings:

### Basic Configuration

```bash
# Server settings
RWWWRSE_SERVER_HOST=0.0.0.0
RWWWRSE_SERVER_PORT=8080
RWWWRSE_SERVER_HTTPS_PORT=8443

# Backend routing (JSON format)
RWWWRSE_BACKENDS_ROUTES='{"api.example.com":{"url":"http://backend:8080","health_path":"/health"}}'

# TLS with Let's Encrypt
RWWWRSE_TLS_EMAIL=admin@example.com
RWWWRSE_TLS_DOMAINS=example.com,api.example.com
```

### Complete Configuration Reference

- **[Configuration Guide](docs/CONFIGURATION.md)** - All environment variables explained
- **[Security Configuration](docs/CONFIGURATION.md#security)** - Security headers, CORS, rate limiting
- **[TLS/SSL Setup](docs/SSL-TLS.md)** - Certificate management strategies

## Documentation

### Getting Started

- **[Quick Start Examples](docs/DEPLOYMENT.md#quick-start)** - Get running in minutes
- **[Configuration Guide](docs/CONFIGURATION.md)** - Complete configuration reference
- **[SSL/TLS Setup](docs/SSL-TLS.md)** - Certificate management

### Operations

- **[Deployment Guide](docs/DEPLOYMENT.md)** - Comprehensive deployment instructions
- **[Operations Manual](docs/OPERATIONS.md)** - Monitoring, logging, troubleshooting
- **[Performance Tuning](docs/OPERATIONS.md#performance)** - Optimization and benchmarks

### Development

- **[Development Setup](docs/DEVELOPMENT.md)** - Local development environment
- **[Architecture Guide](docs/ARCHITECTURE.md)** - Technical architecture details
- **[Migration Guide](docs/MIGRATION.md)** - Upgrade from previous versions

## 🔧 Use Cases

### Simple Website Proxy

```bash
# Route www.example.com to backend server
export RWWWRSE_BACKENDS_ROUTES='{"www.example.com":{"url":"http://192.168.1.100:3000"}}'
export RWWWRSE_TLS_EMAIL=admin@example.com
export RWWWRSE_TLS_DOMAINS=www.example.com
./rwwwrse
```

### Microservices Gateway

```bash
# Route multiple services with health checks
export RWWWRSE_BACKENDS_ROUTES='{
  "api.example.com": {"url":"http://api-service:8080","health_path":"/health"},
  "auth.example.com": {"url":"http://auth-service:8080","health_path":"/status"},
  "web.example.com": {"url":"http://web-service:3000"}
}'
```

### Development Proxy

```bash
# Local development with mock services
export RWWWRSE_BACKENDS_ROUTES='{"localhost":{"url":"http://localhost:3000"}}'
export RWWWRSE_TLS_AUTO_CERT=false  # Disable HTTPS for local dev
./rwwwrse
```

## Monitoring & Health

### Health Checks

```bash
# Check proxy health
curl http://localhost:8080/health

# Check specific backend health
curl http://localhost:8080/health/api.example.com
```

### Metrics (Prometheus format)

View metrics at `/metrics` endpoint:

```bash
curl http://localhost:9090/metrics
```

### Structured Logging

```json
{
  "time": "2024-01-01T12:00:00Z",
  "level": "INFO",
  "msg": "Request processed",
  "request_id": "req-123",
  "method": "GET",
  "path": "/api/v1/users",
  "host": "api.example.com",
  "status": 200,
  "duration": "45ms",
  "backend_url": "http://backend:8080"
}
```

## Security Features

- **Automatic HTTPS** with Let's Encrypt certificates
- **Security Headers** (HSTS, CSP, X-Frame-Options, etc.)
- **Rate Limiting** with configurable rules
- **CORS Support** with flexible origin policies
- **Request Validation** and sanitization
- **Circuit Breaker** patterns for backend protection

## Testing

```bash
make test
make test-race
make test-coverage
make test-benchmark
```

### Development Workflow

```bash

make deps
make generate
make build
make check
make test
make pre-commit
```

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
