# Deployment Guide

This guide provides comprehensive deployment instructions for rwwwrse reverse proxy server across different platforms and scenarios.

## Table of Contents

1. [Quick Start](#quick-start)
2. [Docker Deployments](#docker-deployments)
3. [Docker Compose Deployments](#docker-compose-deployments)
4. [Kubernetes Deployments](#kubernetes-deployments)
5. [Cloud Platform Deployments](#cloud-platform-deployments)
6. [Bare Metal Deployments](#bare-metal-deployments)
7. [Development Deployments](#development-deployments)
8. [Production Considerations](#production-considerations)
9. [Troubleshooting](#troubleshooting)

## Quick Start

### 30-Second Test Run

Test rwwwrse quickly with a public backend service:

```bash
# Using Docker (recommended for quick testing)
docker run -p 8080:8080 -p 8443:8443 \
  -e RWWWRSE_TLS_AUTO_CERT=false \
  -e RWWWRSE_BACKENDS_ROUTES='{"localhost":{"url":"http://httpbin.org"}}' \
  ghcr.io/albedosehen/rwwwrse:latest

# Test the proxy
curl http://localhost:8080/get
```

### 5-Minute Local Setup

Set up rwwwrse with your own backend service:

```bash
# 1. Clone and build
git clone https://github.com/albedosehen/rwwwrse.git
cd rwwwrse
make build

# 2. Configure for your backend
export RWWWRSE_TLS_AUTO_CERT=false  # Disable TLS for local testing
export RWWWRSE_BACKENDS_ROUTES='{"localhost":{"url":"http://your-backend:3000"}}'

# 3. Start the proxy
./bin/rwwwrse

# 4. Test
curl http://localhost:8080/your-endpoint
```

## Docker Deployments

### Simple Docker Run

#### Basic Configuration

```bash
# Create environment file
cat > rwwwrse.env << EOF
RWWWRSE_SERVER_HOST=0.0.0.0
RWWWRSE_SERVER_PORT=8080
RWWWRSE_SERVER_HTTPS_PORT=8443
RWWWRSE_TLS_AUTO_CERT=false
RWWWRSE_BACKENDS_ROUTES={"example.com":{"url":"http://backend:8080"}}
RWWWRSE_LOGGING_LEVEL=info
EOF

# Run container
docker run -d \
  --name rwwwrse \
  --env-file rwwwrse.env \
  -p 8080:8080 \
  -p 8443:8443 \
  -p 9090:9090 \
  ghcr.io/albedosehen/rwwwrse:latest
```

#### Production Configuration with TLS

```bash
# Production environment file
cat > rwwwrse-prod.env << EOF
RWWWRSE_SERVER_HOST=0.0.0.0
RWWWRSE_SERVER_PORT=8080
RWWWRSE_SERVER_HTTPS_PORT=443
RWWWRSE_TLS_ENABLED=true
RWWWRSE_TLS_AUTO_CERT=true
RWWWRSE_TLS_EMAIL=admin@yourdomain.com
RWWWRSE_TLS_DOMAINS=yourdomain.com,api.yourdomain.com
RWWWRSE_TLS_CACHE_DIR=/app/certs
RWWWRSE_BACKENDS_ROUTES={"yourdomain.com":{"url":"http://web-backend:3000"},"api.yourdomain.com":{"url":"http://api-backend:8080","health_path":"/health"}}
RWWWRSE_LOGGING_LEVEL=info
RWWWRSE_LOGGING_FORMAT=json
EOF

# Create certificate volume
docker volume create rwwwrse-certs

# Run production container
docker run -d \
  --name rwwwrse-prod \
  --env-file rwwwrse-prod.env \
  -p 80:8080 \
  -p 443:443 \
  -p 9090:9090 \
  -v rwwwrse-certs:/app/certs \
  --restart unless-stopped \
  ghcr.io/albedosehen/rwwwrse:latest
```

### Building Custom Image

```bash
# Clone repository
git clone https://github.com/albedosehen/rwwwrse.git
cd rwwwrse

# Build with custom tags
docker build \
  --build-arg VERSION=v1.0.0 \
  --build-arg COMMIT_SHA=$(git rev-parse --short HEAD) \
  --build-arg BUILD_DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ") \
  -t my-org/rwwwrse:v1.0.0 \
  .

# Run your custom image
docker run -d \
  --name my-rwwwrse \
  -p 8080:8080 \
  -p 8443:8443 \
  my-org/rwwwrse:v1.0.0
```

## Docker Compose Deployments

### Simple Setup

For detailed examples, see [examples/docker-compose/simple/](../examples/docker-compose/simple/).

```yaml
# docker-compose.yml
version: '3.8'

services:
  rwwwrse:
    image: ghcr.io/albedosehen/rwwwrse:latest
    ports:
      - "8080:8080"
      - "8443:8443"
      - "9090:9090"
    environment:
      RWWWRSE_BACKENDS_ROUTES: '{"localhost":{"url":"http://backend:3000"}}'
      RWWWRSE_TLS_AUTO_CERT: false
    depends_on:
      - backend
    networks:
      - proxy-net

  backend:
    image: nginx:alpine
    networks:
      - proxy-net

networks:
  proxy-net:
    driver: bridge
```

```bash
# Deploy
docker-compose up -d

# Check logs
docker-compose logs -f rwwwrse

# Test
curl http://localhost:8080
```

### Microservices Architecture

For detailed examples, see [examples/docker-compose/microservices/](../examples/docker-compose/microservices/).

```yaml
# docker-compose.yml
version: '3.8'

services:
  rwwwrse:
    image: ghcr.io/albedosehen/rwwwrse:latest
    ports:
      - "80:8080"
      - "443:8443"
      - "9090:9090"
    environment:
      RWWWRSE_TLS_EMAIL: admin@example.com
      RWWWRSE_TLS_DOMAINS: api.example.com,auth.example.com,web.example.com
      RWWWRSE_BACKENDS_ROUTES: |
        {
          "api.example.com": {"url":"http://api-service:8080","health_path":"/health"},
          "auth.example.com": {"url":"http://auth-service:8080","health_path":"/status"},
          "web.example.com": {"url":"http://web-service:3000"}
        }
    volumes:
      - certs:/app/certs
    depends_on:
      - api-service
      - auth-service
      - web-service
    networks:
      - microservices

  api-service:
    image: my-org/api-service:latest
    networks:
      - microservices
    environment:
      - DATABASE_URL=postgresql://user:pass@db:5432/api

  auth-service:
    image: my-org/auth-service:latest
    networks:
      - microservices
    environment:
      - REDIS_URL=redis://redis:6379

  web-service:
    image: my-org/web-frontend:latest
    networks:
      - microservices

  db:
    image: postgres:15-alpine
    networks:
      - microservices
    environment:
      POSTGRES_DB: api
      POSTGRES_USER: user
      POSTGRES_PASSWORD: pass
    volumes:
      - postgres_data:/var/lib/postgresql/data

  redis:
    image: redis:7-alpine
    networks:
      - microservices

volumes:
  certs:
  postgres_data:

networks:
  microservices:
    driver: bridge
```

### Production Setup with Monitoring

For detailed examples, see [examples/docker-compose/production/](../examples/docker-compose/production/).

```yaml
# docker-compose.prod.yml
version: '3.8'

services:
  rwwwrse:
    image: ghcr.io/albedosehen/rwwwrse:latest
    ports:
      - "80:8080"
      - "443:8443"
    environment:
      RWWWRSE_TLS_EMAIL: admin@yourdomain.com
      RWWWRSE_TLS_DOMAINS: yourdomain.com,api.yourdomain.com
      RWWWRSE_BACKENDS_ROUTES: '{"yourdomain.com":{"url":"http://app:3000","health_path":"/health"}}'
      RWWWRSE_LOGGING_LEVEL: info
      RWWWRSE_LOGGING_FORMAT: json
    volumes:
      - certs:/app/certs
      - logs:/app/logs
    depends_on:
      - app
    networks:
      - frontend
      - monitoring
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "wget", "--no-verbose", "--tries=1", "--spider", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3

  app:
    image: my-org/app:latest
    networks:
      - frontend
    restart: unless-stopped

  prometheus:
    image: prom/prometheus:latest
    ports:
      - "9091:9090"
    volumes:
      - ./monitoring/prometheus.yml:/etc/prometheus/prometheus.yml
      - prometheus_data:/prometheus
    networks:
      - monitoring
    restart: unless-stopped

  grafana:
    image: grafana/grafana:latest
    ports:
      - "3000:3000"
    volumes:
      - grafana_data:/var/lib/grafana
      - ./monitoring/grafana/dashboards:/etc/grafana/provisioning/dashboards
      - ./monitoring/grafana/datasources:/etc/grafana/provisioning/datasources
    networks:
      - monitoring
    restart: unless-stopped

volumes:
  certs:
  logs:
  prometheus_data:
  grafana_data:

networks:
  frontend:
    driver: bridge
  monitoring:
    driver: bridge
```

## Kubernetes Deployments

### Prerequisites

```bash
# Ensure kubectl is configured
kubectl cluster-info

# Create namespace
kubectl create namespace rwwwrse
```

### Basic Kubernetes Deployment

For detailed examples, see [examples/kubernetes/](../examples/kubernetes/).

```yaml
# deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: rwwwrse
  namespace: rwwwrse
spec:
  replicas: 3
  selector:
    matchLabels:
      app: rwwwrse
  template:
    metadata:
      labels:
        app: rwwwrse
    spec:
      containers:
      - name: rwwwrse
        image: ghcr.io/albedosehen/rwwwrse:latest
        ports:
        - containerPort: 8080
          name: http
        - containerPort: 8443
          name: https
        - containerPort: 9090
          name: metrics
        env:
        - name: RWWWRSE_BACKENDS_ROUTES
          value: '{"api.example.com":{"url":"http://backend-service:8080","health_path":"/health"}}'
        - name: RWWWRSE_TLS_EMAIL
          value: admin@example.com
        - name: RWWWRSE_TLS_DOMAINS
          value: api.example.com
        volumeMounts:
        - name: certs
          mountPath: /app/certs
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
        resources:
          requests:
            memory: "128Mi"
            cpu: "100m"
          limits:
            memory: "512Mi"
            cpu: "500m"
      volumes:
      - name: certs
        persistentVolumeClaim:
          claimName: rwwwrse-certs

---
apiVersion: v1
kind: Service
metadata:
  name: rwwwrse-service
  namespace: rwwwrse
spec:
  selector:
    app: rwwwrse
  ports:
  - name: http
    port: 80
    targetPort: 8080
  - name: https
    port: 443
    targetPort: 8443
  - name: metrics
    port: 9090
    targetPort: 9090
  type: LoadBalancer

---
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: rwwwrse-certs
  namespace: rwwwrse
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 1Gi
```

```bash
# Deploy
kubectl apply -f deployment.yaml

# Check status
kubectl get pods -n rwwwrse
kubectl get services -n rwwwrse

# View logs
kubectl logs -f deployment/rwwwrse -n rwwwrse
```

### Kubernetes with Ingress

```yaml
# ingress.yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: rwwwrse-ingress
  namespace: rwwwrse
  annotations:
    nginx.ingress.kubernetes.io/ssl-redirect: "true"
    cert-manager.io/cluster-issuer: "letsencrypt-prod"
spec:
  tls:
  - hosts:
    - api.example.com
    secretName: rwwwrse-tls
  rules:
  - host: api.example.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: rwwwrse-service
            port:
              number: 80
```

### Helm Deployment

For detailed examples, see [examples/kubernetes/helm/](../examples/kubernetes/helm/).

```bash
# Add repository (when available)
helm repo add rwwwrse https://charts.rwwwrse.io
helm repo update

# Install with custom values
cat > values.yaml << EOF
replicaCount: 3

image:
  repository: ghcr.io/albedosehen/rwwwrse
  tag: latest

config:
  backends:
    routes:
      api.example.com:
        url: "http://backend-service:8080"
        health_path: "/health"
  
  tls:
    email: "admin@example.com"
    domains:
      - "api.example.com"

ingress:
  enabled: true
  className: "nginx"
  hosts:
    - host: api.example.com
      paths:
        - path: /
          pathType: Prefix
  tls:
    - secretName: rwwwrse-tls
      hosts:
        - api.example.com

monitoring:
  enabled: true
  serviceMonitor:
    enabled: true
EOF

# Deploy
helm install rwwwrse rwwwrse/rwwwrse -f values.yaml -n rwwwrse --create-namespace

# Upgrade
helm upgrade rwwwrse rwwwrse/rwwwrse -f values.yaml -n rwwwrse
```

## Cloud Platform Deployments

### AWS ECS/Fargate

For detailed examples, see [examples/cloud-specific/aws/](../examples/cloud-specific/aws/).

```json
{
  "family": "rwwwrse",
  "networkMode": "awsvpc",
  "requiresCompatibilities": ["FARGATE"],
  "cpu": "256",
  "memory": "512",
  "executionRoleArn": "arn:aws:iam::123456789012:role/ecsTaskExecutionRole",
  "taskRoleArn": "arn:aws:iam::123456789012:role/ecsTaskRole",
  "containerDefinitions": [
    {
      "name": "rwwwrse",
      "image": "ghcr.io/albedosehen/rwwwrse:latest",
      "portMappings": [
        {
          "containerPort": 8080,
          "protocol": "tcp"
        },
        {
          "containerPort": 8443,
          "protocol": "tcp"
        }
      ],
      "environment": [
        {
          "name": "RWWWRSE_TLS_EMAIL",
          "value": "admin@example.com"
        },
        {
          "name": "RWWWRSE_TLS_DOMAINS",
          "value": "api.example.com"
        },
        {
          "name": "RWWWRSE_BACKENDS_ROUTES",
          "value": "{\"api.example.com\":{\"url\":\"http://backend.internal:8080\",\"health_path\":\"/health\"}}"
        }
      ],
      "logConfiguration": {
        "logDriver": "awslogs",
        "options": {
          "awslogs-group": "/ecs/rwwwrse",
          "awslogs-region": "us-west-2",
          "awslogs-stream-prefix": "ecs"
        }
      },
      "healthCheck": {
        "command": [
          "CMD-SHELL",
          "wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1"
        ],
        "interval": 30,
        "timeout": 5,
        "retries": 3
      }
    }
  ]
}
```

### Google Cloud Run

```yaml
# cloudrun.yaml
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
        autoscaling.knative.dev/maxScale: "10"
        run.googleapis.com/cpu-throttling: "false"
        run.googleapis.com/execution-environment: gen2
    spec:
      containerConcurrency: 100
      containers:
      - image: ghcr.io/albedosehen/rwwwrse:latest
        ports:
        - containerPort: 8080
        env:
        - name: RWWWRSE_SERVER_PORT
          value: "8080"
        - name: RWWWRSE_TLS_AUTO_CERT
          value: "false"  # Cloud Run handles TLS
        - name: RWWWRSE_BACKENDS_ROUTES
          value: '{"api.example.com":{"url":"https://backend-service-url"}}'
        resources:
          limits:
            cpu: "1"
            memory: "512Mi"
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        startupProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 10
          periodSeconds: 5
          failureThreshold: 6
```

```bash
# Deploy to Cloud Run
gcloud run services replace cloudrun.yaml --region=us-central1

# Configure custom domain
gcloud run domain-mappings create --service=rwwwrse --domain=api.example.com --region=us-central1
```

### Azure Container Instances

```json
{
  "properties": {
    "containers": [
      {
        "name": "rwwwrse",
        "properties": {
          "image": "ghcr.io/albedosehen/rwwwrse:latest",
          "ports": [
            {
              "port": 8080,
              "protocol": "TCP"
            },
            {
              "port": 8443,
              "protocol": "TCP"
            }
          ],
          "environmentVariables": [
            {
              "name": "RWWWRSE_TLS_EMAIL",
              "value": "admin@example.com"
            },
            {
              "name": "RWWWRSE_TLS_DOMAINS",
              "value": "api.example.com"
            },
            {
              "name": "RWWWRSE_BACKENDS_ROUTES",
              "value": "{\"api.example.com\":{\"url\":\"http://backend:8080\"}}"
            }
          ],
          "resources": {
            "requests": {
              "cpu": 0.5,
              "memoryInGB": 1
            }
          }
        }
      }
    ],
    "osType": "Linux",
    "ipAddress": {
      "type": "Public",
      "ports": [
        {
          "port": 8080,
          "protocol": "TCP"
        },
        {
          "port": 8443,
          "protocol": "TCP"
        }
      ],
      "dnsNameLabel": "rwwwrse-example"
    },
    "restartPolicy": "Always"
  }
}
```

## Bare Metal Deployments

For detailed examples, see [examples/bare-metal/](../examples/bare-metal/).

### systemd Service

```ini
# /etc/systemd/system/rwwwrse.service
[Unit]
Description=rwwwrse Reverse Proxy Server
After=network.target
Wants=network.target

[Service]
Type=simple
User=rwwwrse
Group=rwwwrse
WorkingDirectory=/opt/rwwwrse
ExecStart=/opt/rwwwrse/bin/rwwwrse
ExecReload=/bin/kill -HUP $MAINPID
Restart=always
RestartSec=5
EnvironmentFile=/etc/rwwwrse/rwwwrse.env
StandardOutput=journal
StandardError=journal

# Security settings
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/opt/rwwwrse/certs /var/log/rwwwrse
PrivateTmp=true
ProtectKernelTunables=true
ProtectControlGroups=true
RestrictRealtime=true
RestrictSUIDSGID=true

[Install]
WantedBy=multi-user.target
```

```bash
# Installation steps
sudo useradd -r -s /bin/false rwwwrse
sudo mkdir -p /opt/rwwwrse/{bin,certs} /etc/rwwwrse /var/log/rwwwrse
sudo chown -R rwwwrse:rwwwrse /opt/rwwwrse /var/log/rwwwrse

# Copy binary
sudo cp rwwwrse /opt/rwwwrse/bin/
sudo chmod +x /opt/rwwwrse/bin/rwwwrse

# Create configuration
sudo cat > /etc/rwwwrse/rwwwrse.env << EOF
RWWWRSE_SERVER_HOST=0.0.0.0
RWWWRSE_SERVER_PORT=8080
RWWWRSE_SERVER_HTTPS_PORT=8443
RWWWRSE_TLS_EMAIL=admin@yourdomain.com
RWWWRSE_TLS_DOMAINS=yourdomain.com
RWWWRSE_TLS_CACHE_DIR=/opt/rwwwrse/certs
RWWWRSE_BACKENDS_ROUTES={"yourdomain.com":{"url":"http://localhost:3000"}}
RWWWRSE_LOGGING_LEVEL=info
RWWWRSE_LOGGING_FORMAT=json
EOF

# Install and start service
sudo systemctl daemon-reload
sudo systemctl enable rwwwrse
sudo systemctl start rwwwrse

# Check status
sudo systemctl status rwwwrse
sudo journalctl -fu rwwwrse
```

### Behind Nginx/Apache

#### Nginx Configuration

```nginx
# /etc/nginx/sites-available/rwwwrse
upstream rwwwrse_backend {
    server 127.0.0.1:8080;
    server 127.0.0.1:8081 backup;
    keepalive 32;
}

server {
    listen 80;
    server_name yourdomain.com;
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name yourdomain.com;

    ssl_certificate /etc/letsencrypt/live/yourdomain.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/yourdomain.com/privkey.pem;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers ECDHE-RSA-AES256-GCM-SHA512:DHE-RSA-AES256-GCM-SHA512;
    ssl_prefer_server_ciphers off;

    location / {
        proxy_pass http://rwwwrse_backend;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_cache_bypass $http_upgrade;
        proxy_connect_timeout 5s;
        proxy_send_timeout 60s;
        proxy_read_timeout 60s;
    }

    location /health {
        proxy_pass http://rwwwrse_backend/health;
        access_log off;
    }

    location /metrics {
        proxy_pass http://127.0.0.1:9090/metrics;
        access_log off;
        allow 127.0.0.1;
        allow 10.0.0.0/8;
        deny all;
    }
}
```

## Development Deployments

For detailed examples, see [examples/docker-compose/development/](../examples/docker-compose/development/).

### Local Development with Hot Reload

```bash
# Use Air for hot reloading
go install github.com/cosmtrek/air@latest

# Create .air.toml
cat > .air.toml << EOF
root = "."
testdata_dir = "testdata"
tmp_dir = "tmp"

[build]
  args_bin = []
  bin = "./tmp/main"
  cmd = "go build -o ./tmp/main ./cmd/rwwwrse"
  delay = 1000
  exclude_dir = ["assets", "tmp", "vendor", "testdata"]
  exclude_file = []
  exclude_regex = ["_test.go"]
  exclude_unchanged = false
  follow_symlink = false
  full_bin = ""
  include_dir = []
  include_ext = ["go", "tpl", "tmpl", "html"]
  kill_delay = "0s"
  log = "build-errors.log"
  send_interrupt = false
  stop_on_root = false

[color]
  app = ""
  build = "yellow"
  main = "magenta"
  runner = "green"
  watcher = "cyan"

[log]
  time = false

[misc]
  clean_on_exit = false
EOF

# Set development environment
export RWWWRSE_TLS_AUTO_CERT=false
export RWWWRSE_LOGGING_LEVEL=debug
export RWWWRSE_BACKENDS_ROUTES='{"localhost":{"url":"http://localhost:3000"}}'

# Start with hot reload
air
```

### Testing with Multiple Backends

```yaml
# docker-compose.dev.yml
version: '3.8'

services:
  rwwwrse:
    build: .
    ports:
      - "8080:8080"
      - "9090:9090"
    environment:
      RWWWRSE_TLS_AUTO_CERT: false
      RWWWRSE_LOGGING_LEVEL: debug
      RWWWRSE_BACKENDS_ROUTES: |
        {
          "api.localhost": {"url":"http://api:8080","health_path":"/health"},
          "web.localhost": {"url":"http://web:3000"},
          "admin.localhost": {"url":"http://admin:8080","health_path":"/status"}
        }
    depends_on:
      - api
      - web
      - admin
    networks:
      - dev-net

  api:
    image: kennethreitz/httpbin
    networks:
      - dev-net

  web:
    image: nginx:alpine
    networks:
      - dev-net

  admin:
    image: adminer
    networks:
      - dev-net

  db:
    image: postgres:15-alpine
    environment:
      POSTGRES_DB: testdb
      POSTGRES_USER: test
      POSTGRES_PASSWORD: test
    networks:
      - dev-net

networks:
  dev-net:
    driver: bridge
```

## Production Considerations

### Security Checklist

- [ ] TLS/SSL properly configured with valid certificates
- [ ] Security headers enabled (HSTS, CSP, etc.)
- [ ] Rate limiting configured appropriately
- [ ] CORS policies properly configured
- [ ] Access logs enabled and monitored
- [ ] Metrics collection enabled
- [ ] Health checks configured
- [ ] Resource limits set
- [ ] Non-root user configured
- [ ] Secrets managed securely

### Performance Tuning

```bash
# Environment variables for production
export RWWWRSE_SERVER_READ_TIMEOUT=30s
export RWWWRSE_SERVER_WRITE_TIMEOUT=30s
export RWWWRSE_SERVER_IDLE_TIMEOUT=60s
export RWWWRSE_RATE_LIMIT_REQUESTS_PER_SECOND=100
export RWWWRSE_RATE_LIMIT_BURST_SIZE=200
export RWWWRSE_HEALTH_INTERVAL=30s
export RWWWRSE_BACKENDS_ROUTES_DEFAULT_TIMEOUT=30s
export RWWWRSE_BACKENDS_ROUTES_DEFAULT_MAX_IDLE_CONNS=100
export RWWWRSE_BACKENDS_ROUTES_DEFAULT_MAX_IDLE_PER_HOST=10
```

### Monitoring Setup

```yaml
# monitoring/prometheus.yml
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: 'rwwwrse'
    static_configs:
      - targets: ['rwwwrse:9090']
    metrics_path: /metrics
    scrape_interval: 5s
    
  - job_name: 'node-exporter'
    static_configs:
      - targets: ['node-exporter:9100']
```

### Backup and Recovery

```bash
# Backup TLS certificates
tar -czf rwwwrse-certs-$(date +%Y%m%d).tar.gz /opt/rwwwrse/certs/

# Backup configuration
cp /etc/rwwwrse/rwwwrse.env rwwwrse-config-$(date +%Y%m%d).env

# Restore certificates
tar -xzf rwwwrse-certs-20240101.tar.gz -C /
sudo chown -R rwwwrse:rwwwrse /opt/rwwwrse/certs/
sudo systemctl restart rwwwrse
```

## Troubleshooting

### Common Issues

#### Certificate Issues

```bash
# Check certificate status
curl -v https://yourdomain.com

# Check Let's Encrypt staging
export RWWWRSE_TLS_STAGING=true

# Manual certificate verification
openssl s_client -connect yourdomain.com:443 -servername yourdomain.com
```

#### Backend Connection Issues

```bash
# Check backend health from proxy
curl http://localhost:8080/health

# Test backend directly
curl http://backend-host:8080/health

# Check network connectivity
docker exec rwwwrse wget -qO- http://backend:8080/health
```

#### Performance Issues

```bash
# Check metrics
curl http://localhost:9090/metrics | grep rwwwrse

# Monitor logs
docker logs -f rwwwrse 2>&1 | grep -E "(ERROR|WARN|latency)"

# Resource usage
docker stats rwwwrse
```

### Debug Mode

```bash
# Enable debug logging
export RWWWRSE_LOGGING_LEVEL=debug

# Restart service
sudo systemctl restart rwwwrse

# Follow logs
sudo journalctl -fu rwwwrse
```

### Health Check Verification

```bash
# Basic health check
curl http://localhost:8080/health

# Detailed health information
curl -s http://localhost:8080/health | jq .

# Backend-specific health
curl http://localhost:8080/health/api.example.com
```

For more troubleshooting information, see [OPERATIONS.md](OPERATIONS.md#troubleshooting).

---

This deployment guide provides comprehensive instructions for all major deployment scenarios. For specific examples and configuration files, see the [examples/](../examples/) directory.
