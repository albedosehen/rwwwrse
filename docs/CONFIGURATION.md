# Configuration Guide for rwwwrse

This guide provides comprehensive configuration examples for rwwwrse across all deployment scenarios. It covers environment variables, configuration files, secrets management, and platform-specific settings.

## Overview

rwwwrse is configured through environment variables following the `RWWWRSE_` prefix convention. This approach provides:

- **Consistency** across all deployment platforms
- **Security** through external secret management
- **Flexibility** for different environments
- **12-Factor App compliance** for cloud-native deployments

## Core Configuration

### Basic Server Configuration

```bash
# Server binding and networking
RWWWRSE_HOST=0.0.0.0                    # Interface to bind to
RWWWRSE_PORT=8080                       # Port to listen on
RWWWRSE_ENABLE_TLS=false                # Enable HTTPS (usually handled by load balancer)
RWWWRSE_TLS_CERT_FILE=/path/to/cert.pem # TLS certificate file
RWWWRSE_TLS_KEY_FILE=/path/to/key.pem   # TLS private key file

# Timeouts and performance
RWWWRSE_READ_TIMEOUT=30s                # Request read timeout
RWWWRSE_WRITE_TIMEOUT=30s               # Response write timeout
RWWWRSE_IDLE_TIMEOUT=120s               # Connection idle timeout
RWWWRSE_READ_HEADER_TIMEOUT=10s         # Header read timeout
RWWWRSE_MAX_HEADER_BYTES=1048576        # Maximum header size (1MB)

# Graceful shutdown
RWWWRSE_SHUTDOWN_TIMEOUT=30s            # Graceful shutdown timeout
RWWWRSE_SHUTDOWN_DELAY=5s               # Delay before starting shutdown
```

### Logging Configuration

```bash
# Log settings
RWWWRSE_LOG_LEVEL=info                  # Logging level: debug, info, warn, error
RWWWRSE_LOG_FORMAT=json                 # Log format: json, text
RWWWRSE_LOG_OUTPUT=stdout               # Log output: stdout, stderr, file path
RWWWRSE_ACCESS_LOG_ENABLED=true         # Enable access logging
RWWWRSE_ERROR_STACK_TRACE=false         # Include stack traces in error logs

# Request logging
RWWWRSE_REQUEST_ID_HEADER=X-Request-ID  # Request ID header name
RWWWRSE_LOG_REQUEST_HEADERS=false       # Log request headers (security sensitive)
RWWWRSE_LOG_RESPONSE_HEADERS=false      # Log response headers
RWWWRSE_LOG_REQUEST_BODY=false          # Log request body (debug only)
RWWWRSE_LOG_RESPONSE_BODY=false         # Log response body (debug only)
```

### Health and Metrics

```bash
# Health check configuration
RWWWRSE_HEALTH_PATH=/health             # Health check endpoint path
RWWWRSE_HEALTH_TIMEOUT=10s              # Health check timeout
RWWWRSE_READINESS_PATH=/ready           # Readiness check endpoint path

# Metrics configuration
RWWWRSE_METRICS_ENABLED=true            # Enable Prometheus metrics
RWWWRSE_METRICS_PATH=/metrics           # Metrics endpoint path
RWWWRSE_METRICS_NAMESPACE=rwwwrse       # Metrics namespace
RWWWRSE_DETAILED_METRICS=false          # Enable detailed metrics (higher cardinality)
```

### Route Configuration

```bash
# Default route settings
RWWWRSE_DEFAULT_BACKEND=http://localhost:3000  # Default backend when no route matches
RWWWRSE_STRIP_PATH=false                       # Strip matched path from upstream request
RWWWRSE_PRESERVE_HOST=true                     # Preserve original Host header

# Route definitions (can be repeated with different prefixes)
RWWWRSE_ROUTES_API_HOST=api.example.com        # Host to match
RWWWRSE_ROUTES_API_PATH=/api                   # Path prefix to match (optional)
RWWWRSE_ROUTES_API_TARGET=http://api-service:3001  # Backend target URL
RWWWRSE_ROUTES_API_TIMEOUT=30s                 # Backend timeout
RWWWRSE_ROUTES_API_RETRIES=3                   # Number of retries
RWWWRSE_ROUTES_API_STRIP_PATH=true             # Strip /api from upstream

# Additional routes
RWWWRSE_ROUTES_APP_HOST=app.example.com
RWWWRSE_ROUTES_APP_TARGET=http://app-service:3000
RWWWRSE_ROUTES_WEB_HOST=web.example.com
RWWWRSE_ROUTES_WEB_TARGET=http://web-service:3002
RWWWRSE_ROUTES_ADMIN_HOST=admin.example.com
RWWWRSE_ROUTES_ADMIN_TARGET=http://admin-service:3003
```

### Security Configuration

```bash
# CORS settings
RWWWRSE_CORS_ENABLED=true               # Enable CORS
RWWWRSE_CORS_ORIGINS=*                  # Allowed origins (comma-separated)
RWWWRSE_CORS_METHODS=GET,POST,PUT,DELETE,OPTIONS  # Allowed methods
RWWWRSE_CORS_HEADERS=Content-Type,Authorization    # Allowed headers
RWWWRSE_CORS_CREDENTIALS=false          # Allow credentials

# Rate limiting
RWWWRSE_RATE_LIMIT_ENABLED=false        # Enable rate limiting
RWWWRSE_RATE_LIMIT_RPS=100              # Requests per second limit
RWWWRSE_RATE_LIMIT_BURST=200            # Burst capacity
RWWWRSE_RATE_LIMIT_WINDOW=60s           # Rate limit window

# Security headers
RWWWRSE_SECURITY_HEADERS_ENABLED=true   # Enable security headers
RWWWRSE_HSTS_ENABLED=true               # Enable HSTS header
RWWWRSE_HSTS_MAX_AGE=31536000           # HSTS max age (1 year)
RWWWRSE_FRAME_OPTIONS=SAMEORIGIN        # X-Frame-Options header
RWWWRSE_CONTENT_TYPE_NOSNIFF=true       # X-Content-Type-Options: nosniff
RWWWRSE_XSS_PROTECTION=1; mode=block    # X-XSS-Protection header
```

## Environment-Specific Configurations

### Development Environment

```bash
# Development-focused settings
RWWWRSE_LOG_LEVEL=debug
RWWWRSE_LOG_FORMAT=text
RWWWRSE_REQUEST_LOGGING=true
RWWWRSE_ERROR_STACK_TRACE=true
RWWWRSE_DETAILED_METRICS=true

# Relaxed timeouts for debugging
RWWWRSE_READ_TIMEOUT=300s
RWWWRSE_WRITE_TIMEOUT=300s
RWWWRSE_SHUTDOWN_TIMEOUT=60s

# Development backends
RWWWRSE_ROUTES_API_TARGET=http://localhost:3001
RWWWRSE_ROUTES_APP_TARGET=http://localhost:3000
RWWWRSE_ROUTES_WEB_TARGET=http://localhost:3002

# Disable security features for easier testing
RWWWRSE_CORS_ORIGINS=*
RWWWRSE_RATE_LIMIT_ENABLED=false
RWWWRSE_SECURITY_HEADERS_ENABLED=false
```

### Staging Environment

```bash
# Staging-specific settings
RWWWRSE_LOG_LEVEL=info
RWWWRSE_LOG_FORMAT=json
RWWWRSE_ACCESS_LOG_ENABLED=true

# Production-like timeouts
RWWWRSE_READ_TIMEOUT=30s
RWWWRSE_WRITE_TIMEOUT=30s
RWWWRSE_IDLE_TIMEOUT=120s

# Staging backends
RWWWRSE_ROUTES_API_TARGET=http://staging-api.internal:3001
RWWWRSE_ROUTES_APP_TARGET=http://staging-app.internal:3000
RWWWRSE_ROUTES_WEB_TARGET=http://staging-web.internal:3002

# Enable security but allow testing
RWWWRSE_CORS_ORIGINS=https://staging.example.com
RWWWRSE_RATE_LIMIT_ENABLED=true
RWWWRSE_RATE_LIMIT_RPS=1000
RWWWRSE_SECURITY_HEADERS_ENABLED=true
```

### Production Environment

```bash
# Production settings
RWWWRSE_LOG_LEVEL=warn
RWWWRSE_LOG_FORMAT=json
RWWWRSE_ACCESS_LOG_ENABLED=true
RWWWRSE_ERROR_STACK_TRACE=false

# Optimized timeouts
RWWWRSE_READ_TIMEOUT=30s
RWWWRSE_WRITE_TIMEOUT=30s
RWWWRSE_IDLE_TIMEOUT=60s
RWWWRSE_SHUTDOWN_TIMEOUT=30s

# Production backends
RWWWRSE_ROUTES_API_TARGET=http://prod-api.internal:3001
RWWWRSE_ROUTES_APP_TARGET=http://prod-app.internal:3000
RWWWRSE_ROUTES_WEB_TARGET=http://prod-web.internal:3002

# Strict security settings
RWWWRSE_CORS_ORIGINS=https://example.com,https://app.example.com
RWWWRSE_RATE_LIMIT_ENABLED=true
RWWWRSE_RATE_LIMIT_RPS=500
RWWWRSE_RATE_LIMIT_BURST=1000
RWWWRSE_SECURITY_HEADERS_ENABLED=true
RWWWRSE_HSTS_ENABLED=true
```

## Secrets Management with Doppler

rwwwrse supports [Doppler](https://www.doppler.com/) for secrets management across all deployment environments. This provides a secure, centralized solution for managing sensitive configuration values.

### Doppler Overview

Doppler is a SecretOps platform that provides:

- Centralized secrets management across environments
- Role-based access control for secrets
- Automatic secrets rotation
- Audit logs for security compliance
- Integrations with various deployment platforms

### Doppler Configuration

To use Doppler with rwwwrse, you need to set up:

1. A Doppler account and project
2. Service tokens for authentication
3. Secrets in Doppler that match the RWWWRSE_ prefix convention

The rwwwrse container image includes the Doppler CLI, which is used to fetch secrets at startup.

```bash
# Doppler authentication (required for CLI access)
DOPPLER_TOKEN=your-service-token-here    # Service token for authentication
DOPPLER_PROJECT=rwwwrse                  # Doppler project name
DOPPLER_CONFIG=dev                       # Configuration environment (dev, staging, prod)

# Optional Doppler settings
DOPPLER_FETCH_TIMEOUT=10s                # Timeout for fetching secrets
DOPPLER_RETRY_COUNT=3                    # Number of retries for fetching secrets
```

### Docker with Doppler

When running rwwwrse in Docker with Doppler:

```yaml
# docker-compose.yml
version: '3.8'

services:
  rwwwrse:
    image: rwwwrse:latest
    environment:
      # Doppler authentication
      DOPPLER_TOKEN: ${DOPPLER_TOKEN}
      DOPPLER_PROJECT: rwwwrse
      DOPPLER_CONFIG: dev
      
      # Basic configuration (can be overridden by Doppler)
      RWWWRSE_PORT: 8080
```

This allows rwwwrse to fetch secrets from Doppler at startup and override any environment variables with values from Doppler.

### Kubernetes with Doppler

For Kubernetes deployments, use an init container to fetch secrets:

```yaml
# deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: rwwwrse
spec:
  template:
    spec:
      # Fetch secrets from Doppler before starting the application
      initContainers:
      - name: doppler-secrets
        image: rwwwrse:latest
        command: ["/bin/sh", "-c"]
        args:
        - |
          doppler secrets download --format env-no-quotes > /etc/secrets/doppler-secrets.env
        env:
        - name: DOPPLER_TOKEN
          valueFrom:
            secretKeyRef:
              name: doppler-token
              key: token
        volumeMounts:
        - name: doppler-secrets
          mountPath: /etc/secrets
      
      containers:
      - name: rwwwrse
        image: rwwwrse:latest
        # Load the secrets from the file written by the init container
        envFrom:
        - secretRef:
            name: doppler-secrets
      
      volumes:
      - name: doppler-secrets
        emptyDir: {}
```

### Managing Secrets in Doppler

Typical secrets you might want to manage in Doppler:

```bash
# API keys and authentication
RWWWRSE_API_KEY=secure-api-key-value
RWWWRSE_AUTH_SECRET=secure-auth-secret

# Database connections
RWWWRSE_DATABASE_URL=postgresql://username:password@host:port/database

# TLS certificates
RWWWRSE_TLS_CERT_KEY_PASSWORD=secure-certificate-password

# Service credentials
RWWWRSE_SERVICE_ACCOUNT_TOKEN=secure-service-token
```

### Doppler Best Practices

1. **Environment Separation**: Create separate Doppler configurations for development, staging, and production environments.

2. **Service Tokens**: Use different service tokens for different environments with appropriate access levels.

3. **Minimal Access**: Create service tokens with read-only access to only the projects and configurations they need.

4. **Token Rotation**: Regularly rotate service tokens to minimize security risks.

5. **Logging**: Ensure that Doppler logs are moenitored for unauthorized access attempts.

6. **Fallbacks**: Configure rwwwrse to gracefully handle cases where Doppler might be unavailable.

## Platform-Specific Configurations

### Docker Compose Configuration

#### Environment File (`.env`)

```bash
# Docker Compose environment file
COMPOSE_PROJECT_NAME=rwwwrse
COMPOSE_FILE=docker-compose.yml:docker-compose.override.yml

# rwwwrse configuration
RWWWRSE_IMAGE=rwwwrse:latest
RWWWRSE_PORT=8080
RWWWRSE_LOG_LEVEL=info
RWWWRSE_LOG_FORMAT=json

# Backend services
API_SERVICE_URL=http://api:3001
APP_SERVICE_URL=http://app:3000
WEB_SERVICE_URL=http://web:3002

# Database configuration
DATABASE_URL=postgresql://rwwwrse:password@postgres:5432/rwwwrse
REDIS_URL=redis://redis:6379/0

# Monitoring
PROMETHEUS_ENABLED=true
GRAFANA_ENABLED=true
```

#### Docker Compose Service Configuration

```yaml
# docker-compose.yml
version: '3.8'

services:
  rwwwrse:
    image: ${RWWWRSE_IMAGE}
    environment:
      - RWWWRSE_PORT=${RWWWRSE_PORT}
      - RWWWRSE_LOG_LEVEL=${RWWWRSE_LOG_LEVEL}
      - RWWWRSE_LOG_FORMAT=${RWWWRSE_LOG_FORMAT}
      - RWWWRSE_ROUTES_API_TARGET=${API_SERVICE_URL}
      - RWWWRSE_ROUTES_APP_TARGET=${APP_SERVICE_URL}
      - RWWWRSE_ROUTES_WEB_TARGET=${WEB_SERVICE_URL}
    env_file:
      - .env
      - .env.local  # Local overrides
```

### Kubernetes Configuration

#### ConfigMap for Non-Sensitive Configuration

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: rwwwrse-config
  namespace: rwwwrse
data:
  RWWWRSE_PORT: "8080"
  RWWWRSE_LOG_LEVEL: "info"
  RWWWRSE_LOG_FORMAT: "json"
  RWWWRSE_HEALTH_PATH: "/health"
  RWWWRSE_METRICS_PATH: "/metrics"
  RWWWRSE_METRICS_ENABLED: "true"
  
  # Timeouts
  RWWWRSE_READ_TIMEOUT: "30s"
  RWWWRSE_WRITE_TIMEOUT: "30s"
  RWWWRSE_IDLE_TIMEOUT: "120s"
  
  # Security
  RWWWRSE_CORS_ENABLED: "true"
  RWWWRSE_SECURITY_HEADERS_ENABLED: "true"
  RWWWRSE_RATE_LIMIT_ENABLED: "true"
  RWWWRSE_RATE_LIMIT_RPS: "100"
  
  # Routes (using Kubernetes service discovery)
  RWWWRSE_ROUTES_API_HOST: "api.example.com"
  RWWWRSE_ROUTES_API_TARGET: "http://api-service.rwwwrse.svc.cluster.local:3001"
  RWWWRSE_ROUTES_APP_HOST: "app.example.com"
  RWWWRSE_ROUTES_APP_TARGET: "http://app-service.rwwwrse.svc.cluster.local:3000"
```

#### Secret for Sensitive Configuration

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: rwwwrse-secrets
  namespace: rwwwrse
type: Opaque
stringData:
  DATABASE_URL: "postgresql://rwwwrse:secretpassword@postgres:5432/rwwwrse"
  REDIS_URL: "redis://redis:6379/0"
  JWT_SECRET: "your-jwt-secret-here"
  API_KEY: "your-api-key-here"
```

#### Deployment with Configuration

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: rwwwrse
  namespace: rwwwrse
spec:
  template:
    spec:
      containers:
      - name: rwwwrse
        image: rwwwrse:latest
        envFrom:
        - configMapRef:
            name: rwwwrse-config
        - secretRef:
            name: rwwwrse-secrets
        env:
        - name: POD_NAME
          valueFrom:
            fieldRef:
              fieldPath: metadata.name
        - name: POD_NAMESPACE
          valueFrom:
            fieldRef:
              fieldPath: metadata.namespace
```

### AWS ECS Configuration

#### Task Definition with Environment Variables

```json
{
  "family": "rwwwrse",
  "taskRoleArn": "arn:aws:iam::123456789012:role/rwwwrse-task-role",
  "executionRoleArn": "arn:aws:iam::123456789012:role/ecs-task-execution-role",
  "networkMode": "awsvpc",
  "requiresCompatibilities": ["FARGATE"],
  "cpu": "512",
  "memory": "1024",
  "containerDefinitions": [
    {
      "name": "rwwwrse",
      "image": "123456789012.dkr.ecr.us-east-1.amazonaws.com/rwwwrse:latest",
      "environment": [
        {"name": "RWWWRSE_PORT", "value": "8080"},
        {"name": "RWWWRSE_LOG_LEVEL", "value": "info"},
        {"name": "RWWWRSE_LOG_FORMAT", "value": "json"},
        {"name": "RWWWRSE_HEALTH_PATH", "value": "/health"},
        {"name": "RWWWRSE_METRICS_ENABLED", "value": "true"}
      ],
      "secrets": [
        {
          "name": "DATABASE_URL",
          "valueFrom": "arn:aws:ssm:us-east-1:123456789012:parameter/rwwwrse/database-url"
        },
        {
          "name": "API_KEY",
          "valueFrom": "arn:aws:secretsmanager:us-east-1:123456789012:secret:rwwwrse/api-key"
        }
      ],
      "logConfiguration": {
        "logDriver": "awslogs",
        "options": {
          "awslogs-group": "/ecs/rwwwrse",
          "awslogs-region": "us-east-1",
          "awslogs-stream-prefix": "ecs"
        }
      }
    }
  ]
}
```

#### AWS Systems Manager Parameters

```bash
# Store configuration in SSM Parameter Store
aws ssm put-parameter \
  --name "/rwwwrse/log-level" \
  --value "info" \
  --type "String"

aws ssm put-parameter \
  --name "/rwwwrse/api-target" \
  --value "http://api.internal.example.com:3001" \
  --type "String"

# Store secrets in Secrets Manager
aws secretsmanager create-secret \
  --name "rwwwrse/database-url" \
  --secret-string "postgresql://user:pass@rds-endpoint:5432/db"
```

### Google Cloud Run Configuration

#### Service Configuration with Environment Variables

```yaml
apiVersion: serving.knative.dev/v1
kind: Service
metadata:
  name: rwwwrse
  annotations:
    run.googleapis.com/ingress: all
spec:
  template:
    metadata:
      annotations:
        autoscaling.knative.dev/maxScale: "100"
        run.googleapis.com/memory: "512Mi"
        run.googleapis.com/cpu: "1000m"
    spec:
      containers:
      - image: gcr.io/project-id/rwwwrse:latest
        env:
        - name: RWWWRSE_PORT
          value: "8080"
        - name: RWWWRSE_LOG_LEVEL
          value: "info"
        - name: RWWWRSE_LOG_FORMAT
          value: "json"
        - name: RWWWRSE_ROUTES_API_TARGET
          value: "https://api-service-hash-uc.a.run.app"
        - name: DATABASE_URL
          valueFrom:
            secretKeyRef:
              name: database-url
              key: latest
        - name: API_KEY
          valueFrom:
            secretKeyRef:
              name: api-key
              key: latest
```

#### Google Secret Manager Integration

```bash
# Create secrets in Secret Manager
gcloud secrets create database-url --data-file=database-url.txt
gcloud secrets create api-key --data-file=api-key.txt

# Grant Cloud Run access to secrets
gcloud secrets add-iam-policy-binding database-url \
  --member=serviceAccount:$(gcloud projects describe PROJECT_ID --format="value(projectNumber)")-compute@developer.gserviceaccount.com \
  --role=roles/secretmanager.secretAccessor
```

### Azure Container Instances Configuration

#### ARM Template with Configuration

```json
{
  "$schema": "https://schema.management.azure.com/schemas/2019-04-01/deploymentTemplate.json#",
  "contentVersion": "1.0.0.0",
  "parameters": {
    "containerName": {
      "type": "string",
      "defaultValue": "rwwwrse"
    }
  },
  "resources": [
    {
      "type": "Microsoft.ContainerInstance/containerGroups",
      "apiVersion": "2021-03-01",
      "name": "[parameters('containerName')]",
      "location": "[resourceGroup().location]",
      "properties": {
        "containers": [
          {
            "name": "rwwwrse",
            "properties": {
              "image": "rwwwrseacr.azurecr.io/rwwwrse:latest",
              "environmentVariables": [
                {"name": "RWWWRSE_PORT", "value": "8080"},
                {"name": "RWWWRSE_LOG_LEVEL", "value": "info"},
                {"name": "RWWWRSE_LOG_FORMAT", "value": "json"}
              ],
              "secureEnvironmentVariables": [
                {
                  "name": "DATABASE_URL",
                  "secureValue": "[reference(resourceId('Microsoft.KeyVault/vaults/secrets', 'rwwwrse-kv', 'database-url')).secretValue]"
                }
              ]
            }
          }
        ]
      }
    }
  ]
}
```

### Bare-Metal Configuration

#### Environment File (`/etc/rwwwrse/config.env`)

```bash
# Production bare-metal configuration
RWWWRSE_HOST=127.0.0.1
RWWWRSE_PORT=8080
RWWWRSE_LOG_LEVEL=info
RWWWRSE_LOG_FORMAT=json
RWWWRSE_LOG_OUTPUT=/var/log/rwwwrse/app.log

# Performance tuning for bare-metal
RWWWRSE_READ_TIMEOUT=30s
RWWWRSE_WRITE_TIMEOUT=30s
RWWWRSE_IDLE_TIMEOUT=60s
RWWWRSE_MAX_HEADER_BYTES=1048576

# Health and metrics
RWWWRSE_HEALTH_PATH=/health
RWWWRSE_METRICS_ENABLED=true
RWWWRSE_METRICS_PATH=/metrics

# Routes (direct to backend servers)
RWWWRSE_ROUTES_API_HOST=api.example.com
RWWWRSE_ROUTES_API_TARGET=http://192.168.1.10:3001
RWWWRSE_ROUTES_APP_HOST=app.example.com
RWWWRSE_ROUTES_APP_TARGET=http://192.168.1.11:3000
RWWWRSE_ROUTES_WEB_HOST=web.example.com
RWWWRSE_ROUTES_WEB_TARGET=http://192.168.1.12:3002

# Security (handled by reverse proxy)
RWWWRSE_ENABLE_TLS=false
RWWWRSE_CORS_ENABLED=true
RWWWRSE_SECURITY_HEADERS_ENABLED=false  # Handled by NGINX/Apache

# Database and external services
DATABASE_URL=postgresql://rwwwrse:password@localhost:5432/rwwwrse
REDIS_URL=redis://localhost:6379/0
```

#### Systemd Environment File

```bash
# /etc/systemd/system/rwwwrse.service.d/override.conf
[Service]
EnvironmentFile=/etc/rwwwrse/config.env
EnvironmentFile=-/etc/rwwwrse/local.env  # Optional local overrides
```

## Advanced Configuration Patterns

### Feature Flags and A/B Testing

```bash
# Feature flag configuration
RWWWRSE_FEATURE_FLAGS_ENABLED=true
RWWWRSE_FEATURE_FLAGS_PROVIDER=file  # file, redis, etcd
RWWWRSE_FEATURE_FLAGS_CONFIG=/etc/rwwwrse/features.json

# A/B testing
RWWWRSE_AB_TESTING_ENABLED=true
RWWWRSE_AB_TESTING_HEADER=X-AB-Test
RWWWRSE_AB_TESTING_COOKIE=ab_test
```

### Circuit Breaker Configuration

```bash
# Circuit breaker settings
RWWWRSE_CIRCUIT_BREAKER_ENABLED=true
RWWWRSE_CIRCUIT_BREAKER_THRESHOLD=10      # Failure threshold
RWWWRSE_CIRCUIT_BREAKER_TIMEOUT=60s       # Open state timeout
RWWWRSE_CIRCUIT_BREAKER_MAX_REQUESTS=10   # Half-open state requests
```

### Load Balancing Configuration

```bash
# Load balancing strategy
RWWWRSE_LOAD_BALANCER_STRATEGY=round_robin  # round_robin, least_conn, ip_hash
RWWWRSE_HEALTH_CHECK_ENABLED=true
RWWWRSE_HEALTH_CHECK_INTERVAL=30s
RWWWRSE_HEALTH_CHECK_TIMEOUT=10s
RWWWRSE_HEALTH_CHECK_PATH=/health

# Multiple backend targets
RWWWRSE_ROUTES_API_TARGETS=http://api1:3001,http://api2:3001,http://api3:3001
RWWWRSE_ROUTES_API_WEIGHTS=3,2,1  # Weighted load balancing
```

### Caching Configuration

```bash
# Response caching
RWWWRSE_CACHE_ENABLED=true
RWWWRSE_CACHE_PROVIDER=redis  # memory, redis, memcached
RWWWRSE_CACHE_TTL=300s
RWWWRSE_CACHE_KEY_PREFIX=rwwwrse:cache:

# Cache rules
RWWWRSE_CACHE_RULES=/static/*:3600,/api/users:300,/api/posts:600
```

## Configuration Validation

### Environment Variable Validation

```bash
#!/bin/bash
# Configuration validation script

# Required variables
REQUIRED_VARS=(
    "RWWWRSE_PORT"
    "RWWWRSE_LOG_LEVEL"
    "RWWWRSE_ROUTES_API_TARGET"
)

# Validate required variables
for var in "${REQUIRED_VARS[@]}"; do
    if [[ -z "${!var}" ]]; then
        echo "Error: Required variable $var is not set"
        exit 1
    fi
done

# Validate log level
if [[ ! "$RWWWRSE_LOG_LEVEL" =~ ^(debug|info|warn|error)$ ]]; then
    echo "Error: Invalid log level: $RWWWRSE_LOG_LEVEL"
    exit 1
fi

# Validate port
if [[ ! "$RWWWRSE_PORT" =~ ^[0-9]+$ ]] || [[ "$RWWWRSE_PORT" -lt 1 ]] || [[ "$RWWWRSE_PORT" -gt 65535 ]]; then
    echo "Error: Invalid port: $RWWWRSE_PORT"
    exit 1
fi

echo "Configuration validation passed"
```

### Configuration Schema

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "RWWWRSE_PORT": {
      "type": "integer",
      "minimum": 1,
      "maximum": 65535
    },
    "RWWWRSE_LOG_LEVEL": {
      "type": "string",
      "enum": ["debug", "info", "warn", "error"]
    },
    "RWWWRSE_LOG_FORMAT": {
      "type": "string",
      "enum": ["json", "text"]
    },
    "RWWWRSE_ENABLE_TLS": {
      "type": "boolean"
    }
  },
  "required": [
    "RWWWRSE_PORT",
    "RWWWRSE_LOG_LEVEL"
  ]
}
```

## Configuration Management Best Practices

### 1. Environment Separation

```bash
# Use different configuration files per environment
/etc/rwwwrse/
├── base.env          # Common configuration
├── development.env   # Development overrides
├── staging.env       # Staging overrides
└── production.env    # Production overrides
```

### 2. Secret Management

```bash
# Never store secrets in configuration files
# Use dedicated secret management systems:

# Kubernetes Secrets
kubectl create secret generic rwwwrse-secrets \
  --from-literal=database-url="postgresql://..." \
  --from-literal=api-key="secret-key"

# AWS Secrets Manager
aws secretsmanager create-secret \
  --name "rwwwrse/production/database-url" \
  --secret-string "postgresql://..."

# Azure Key Vault
az keyvault secret set \
  --vault-name rwwwrse-kv \
  --name database-url \
  --value "postgresql://..."

# Doppler (preferred method)
doppler secrets set DATABASE_URL="postgresql://..." --project rwwwrse --config prod
doppler secrets set API_KEY="secret-key" --project rwwwrse --config prod
```

### 3. Configuration as Code

```yaml
# Helm values.yaml for Kubernetes
rwwwrse:
  image:
    repository: rwwwrse
    tag: "1.0.0"
  
  config:
    logLevel: info
    logFormat: json
    port: 8080
    
  routes:
    api:
      host: api.example.com
      target: http://api-service:3001
    app:
      host: app.example.com
      target: http://app-service:3000
      
  secrets:
    databaseUrl: postgresql://...
    apiKey: secret-key
```

### 4. Configuration Testing

```bash
#!/bin/bash
# Test configuration in CI/CD

# Lint configuration files
yamllint config/
jsonlint config/*.json

# Validate against schema
ajv validate -s schema.json -d config.json

# Test with dry-run
rwwwrse --config-check --dry-run
```

### 5. Documentation

Always document configuration changes:

```markdown
## Configuration Changes

### v1.1.0
- Added `RWWWRSE_CIRCUIT_BREAKER_ENABLED` for circuit breaker functionality
- Deprecated `RWWWRSE_OLD_SETTING` in favor of `RWWWRSE_NEW_SETTING`
- Default value for `RWWWRSE_TIMEOUT` changed from 30s to 60s

### Migration Guide
1. Update environment variables
2. Test in staging environment
3. Deploy to production
```

## Troubleshooting Configuration Issues

### Common Configuration Problems

1. **Environment Variable Not Loaded**

   ```bash
   # Check if variable is set
   echo $RWWWRSE_PORT
   
   # Check in container/pod
   kubectl exec -it pod-name -- env | grep RWWWRSE
   docker exec container-name env | grep RWWWRSE
   ```

2. **Invalid Configuration Values**

   ```bash
   # Enable debug logging to see configuration loading
   RWWWRSE_LOG_LEVEL=debug rwwwrse
   
   # Use configuration validation
   rwwwrse --validate-config
   ```

3. **Secret Access Issues**

   ```bash
   # Check secret permissions (Kubernetes)
   kubectl auth can-i get secrets --as=system:serviceaccount:namespace:service-account
   
   # Check IAM permissions (AWS)
   aws sts get-caller-identity
   aws iam simulate-principal-policy --policy-source-arn arn --action-names secretsmanager:GetSecretValue
   
   # Check Doppler access
   doppler run --command="env | grep RWWWRSE"
   ```

4. **Route Configuration Problems**

   ```bash
   # Test backend connectivity
   curl -v http://backend-host:port/health
   
   # Check DNS resolution
   nslookup backend-host
   
   # Verify proxy behavior
   curl -H "Host: api.example.com" http://localhost:8080/health
   ```

## Related Documentation

- [Deployment Guide](DEPLOYMENT.md) - Platform-specific deployment instructions
- [Operations Guide](OPERATIONS.md) - Monitoring and troubleshooting
- [SSL/TLS Guide](SSL-TLS.md) - Certificate management
- [Development Guide](DEVELOPMENT.md) - Local development setup
- [Docker Compose Examples](../examples/docker-compose/) - Container deployments
- [Kubernetes Examples](../examples/kubernetes/) - Kubernetes deployments  
- [Cloud Examples](../examples/cloud-specific/) - Cloud platform deployments
- [CI/CD Examples](../examples/cicd/) - Automated deployment pipelines
