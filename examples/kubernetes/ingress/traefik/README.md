# rwwwrse with Traefik Ingress Controller

This example demonstrates deploying rwwwrse with Traefik using Custom Resource Definitions (CRDs) for advanced traffic management, dynamic configuration, and comprehensive middleware pipeline.

## 🏗️ Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                    Kubernetes Cluster                      │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐   │
│  │               Traefik Controller                    │   │
│  │                                                     │   │
│  │  ┌─────────────┐    ┌─────────────────────────┐    │   │
│  │  │   External  │    │    Dynamic Routing &   │    │   │
│  │  │Load Balancer│────│   Middleware Pipeline  │    │   │
│  │  │   (Cloud)   │    │                         │    │   │
│  │  └─────────────┘    └─────────────────────────┘    │   │
│  │                                │                    │   │
│  │                                ▼                    │   │
│  │  ┌─────────────────────────────────────────────┐    │   │
│  │  │            IngressRoutes                    │    │   │
│  │  │  • api.example.com → Security + Rate Limit │    │   │
│  │  │  • app.example.com → CORS + Compression    │    │   │
│  │  │  • web.example.com → Circuit Breaker       │    │   │
│  │  │  • admin.example.com → Auth + Rate Limit   │    │   │
│  │  └─────────────────────────────────────────────┘    │   │
│  │                                │                    │   │
│  │                                ▼                    │   │
│  │  ┌─────────────────────────────────────────────┐    │   │
│  │  │            rwwwrse Pods                     │    │   │
│  │  │          (Auto-scaled 2-20)                │    │   │
│  │  │                                             │    │   │
│  │  │  ┌─────────┐ ┌─────────┐ ┌─────────┐      │    │   │
│  │  │  │  Pod 1  │ │  Pod 2  │ │  Pod 3  │      │    │   │
│  │  │  └─────────┘ └─────────┘ └─────────┘      │    │   │
│  │  └─────────────────────────────────────────────┘    │   │
│  └─────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
```

## 🎯 Traefik CRD Features

- **IngressRoute CRDs**: Native Kubernetes resources for routing configuration
- **Middleware Pipeline**: Composable middleware for request/response processing
- **Dynamic Configuration**: Real-time updates without restarts
- **TLS Options**: Fine-grained SSL/TLS configuration
- **Circuit Breaker**: Built-in failure detection and recovery
- **Rate Limiting**: Per-IP and per-service rate limiting
- **CORS Support**: Complete cross-origin resource sharing setup
- **Automatic HTTPS**: Let's Encrypt integration with TLS challenge
- **Observability**: Prometheus metrics, Jaeger tracing, and access logs
- **Dashboard**: Real-time monitoring and configuration visualization

## 📋 Prerequisites

### Kubernetes Cluster

```bash
# Verify cluster access
kubectl cluster-info
kubectl get nodes
```

### Required Tools

```bash
# Required versions
kubectl version --client  # >= 1.20.0
helm version              # >= 3.7.0 (optional)
```

## 🚀 Quick Start

### 1. Install Traefik CRDs

```bash
# Install Traefik CRDs first
kubectl apply -f https://raw.githubusercontent.com/traefik/traefik/v3.0/docs/content/reference/dynamic-configuration/kubernetes-crd-definition-v1.yml

# Verify CRDs are installed
kubectl get crd | grep traefik
```

### 2. Install Traefik Controller

#### Option A: Using Helm (Recommended)

```bash
# Add Traefik Helm repository
helm repo add traefik https://traefik.github.io/charts
helm repo update

# Create namespace
kubectl create namespace traefik-system

# Install Traefik with custom values
cat <<EOF > traefik-values.yaml
# Traefik configuration
deployment:
  replicas: 2

# Service configuration
service:
  type: LoadBalancer

# Dashboard configuration
ingressRoute:
  dashboard:
    enabled: true
    matchRule: Host(\`traefik.example.com\`)
    entryPoints: ["websecure"]
    tls:
      secretName: traefik-dashboard-tls

# Certificate resolvers
certificatesResolvers:
  letsencrypt:
    acme:
      tlsChallenge: {}
      email: admin@example.com
      storage: /data/acme.json
      caServer: https://acme-v02.api.letsencrypt.org/directory

# Prometheus metrics
metrics:
  prometheus:
    addEntryPointsLabels: true
    addServicesLabels: true

# Access logs
logs:
  general:
    level: INFO
  access:
    enabled: true

# Additional arguments
additionalArguments:
  - "--certificatesresolvers.letsencrypt.acme.tlschallenge=true"
  - "--certificatesresolvers.letsencrypt.acme.email=admin@example.com"
  - "--certificatesresolvers.letsencrypt.acme.storage=/data/acme.json"
  - "--certificatesresolvers.letsencrypt.acme.caserver=https://acme-v02.api.letsencrypt.org/directory"

# Ports configuration
ports:
  web:
    port: 80
    exposedPort: 80
    protocol: TCP
  websecure:
    port: 443
    exposedPort: 443
    protocol: TCP
    tls:
      enabled: true
  traefik:
    port: 9000
    exposedPort: 9000
    protocol: TCP

# Persistence for ACME
persistence:
  enabled: true
  size: 128Mi
  path: /data
EOF

# Install Traefik
helm install traefik traefik/traefik \
    --namespace traefik-system \
    --values traefik-values.yaml

# Verify installation
kubectl get pods -n traefik-system
kubectl get services -n traefik-system
```

#### Option B: Using Kubectl

```bash
# Apply Traefik manifests
kubectl apply -f https://raw.githubusercontent.com/traefik/traefik/v3.0/docs/content/getting-started/install-traefik.yml

# Wait for deployment
kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=traefik -n traefik-system --timeout=300s
```

### 3. Deploy rwwwrse

```bash
# Navigate to Traefik example directory
cd examples/kubernetes/ingress/traefik

# Update configuration
# - Edit configmap.yaml to set your domain names and email
# - Edit deployment.yaml to set your container image
# - Update IngressRoute hosts for your domains

# Deploy rwwwrse
kubectl apply -f namespace.yaml
kubectl apply -f configmap.yaml
kubectl apply -f deployment.yaml

# Verify deployment
kubectl get pods -n rwwwrse
kubectl get services -n rwwwrse
kubectl get ingressroutes -n rwwwrse
```

### 4. Verify SSL Certificates

```bash
# Check certificate status (Traefik manages certificates automatically)
kubectl logs -n traefik-system deployment/traefik | grep -i certificate

# Check IngressRoute status
kubectl describe ingressroute -n rwwwrse

# Test HTTPS endpoints
curl -k https://api.example.com/health
curl -k https://app.example.com/
```

## ⚙️ Configuration

### IngressRoute Configuration

Traefik uses IngressRoute CRDs instead of standard Ingress resources:

```yaml
apiVersion: traefik.containo.us/v1alpha1
kind: IngressRoute
metadata:
  name: rwwwrse-api
spec:
  entryPoints:
    - websecure
  routes:
  - match: Host(`api.example.com`)
    kind: Rule
    services:
    - name: rwwwrse
      port: 80
    middlewares:
    - name: security-headers
    - name: rate-limit
    - name: compression
    - name: circuit-breaker
  tls:
    secretName: rwwwrse-api-tls
    options:
      name: modern-tls
```

### Middleware Configuration

Traefik middleware provides composable request/response processing:

#### Security Headers Middleware
```yaml
apiVersion: traefik.containo.us/v1alpha1
kind: Middleware
metadata:
  name: security-headers
spec:
  headers:
    frameDeny: true
    contentTypeNosniff: true
    browserXssFilter: true
    referrerPolicy: "strict-origin-when-cross-origin"
    forceSTSHeader: true
    stsIncludeSubdomains: true
    stsPreload: true
    stsSeconds: 31536000
```

#### Rate Limiting Middleware
```yaml
apiVersion: traefik.containo.us/v1alpha1
kind: Middleware
metadata:
  name: rate-limit
spec:
  rateLimit:
    burst: 100
    average: 100
    period: "1m"
    sourceCriterion:
      ipStrategy:
        depth: 1
```

#### CORS Middleware
```yaml
apiVersion: traefik.containo.us/v1alpha1
kind: Middleware
metadata:
  name: cors
spec:
  headers:
    accessControlAllowMethods:
      - "GET"
      - "POST"
      - "PUT"
      - "DELETE"
      - "OPTIONS"
    accessControlAllowOriginList:
      - "https://app.example.com"
      - "https://web.example.com"
    accessControlAllowCredentials: true
```

#### Circuit Breaker Middleware
```yaml
apiVersion: traefik.containo.us/v1alpha1
kind: Middleware
metadata:
  name: circuit-breaker
spec:
  circuitBreaker:
    expression: "NetworkErrorRatio() > 0.10 || ResponseCodeRatio(500, 600, 0, 600) > 0.25"
    checkPeriod: "10s"
    fallbackDuration: "60s"
    recoveryDuration: "30s"
```

### TLS Options

Fine-grained TLS configuration:

```yaml
apiVersion: traefik.containo.us/v1alpha1
kind: TLSOption
metadata:
  name: modern-tls
spec:
  minVersion: "VersionTLS12"
  maxVersion: "VersionTLS13"
  cipherSuites:
    - "TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256"
    - "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256"
    - "TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384"
    - "TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384"
  sniStrict: true
```

## 📊 Monitoring and Observability

### Traefik Dashboard

Access the Traefik dashboard:

```bash
# Port forward to access dashboard
kubectl port-forward -n traefik-system service/traefik 9000:9000

# Open browser to http://localhost:9000/dashboard/
```

### Prometheus Metrics

```bash
# Enable metrics in Helm values
helm upgrade traefik traefik/traefik \
    --namespace traefik-system \
    --set metrics.prometheus.addEntryPointsLabels=true \
    --set metrics.prometheus.addServicesLabels=true

# Access metrics
kubectl port-forward -n traefik-system service/traefik 9000:9000
curl http://localhost:9000/metrics
```

### Key Metrics Available

- **Request Duration**: `traefik_service_request_duration_seconds`
- **Request Count**: `traefik_service_requests_total`
- **Response Size**: `traefik_service_response_size_bytes`
- **Circuit Breaker**: `traefik_service_circuit_breaker_transitions_total`
- **Rate Limit**: `traefik_service_rate_limiter_requests_total`

### Jaeger Tracing

Enable distributed tracing:

```yaml
# Add to Traefik configuration
tracing:
  jaeger:
    samplingServerURL: http://jaeger:5778/sampling
    localAgentHostPort: jaeger:6831
```

### Access Logs

```bash
# View Traefik access logs
kubectl logs -n traefik-system deployment/traefik -f

# View rwwwrse logs
kubectl logs -n rwwwrse deployment/rwwwrse -f

# Access logs are in JSON format for easy parsing
kubectl logs -n traefik-system deployment/traefik -f | jq .
```

## 🔐 Security

### Automatic HTTPS

Traefik automatically obtains and renews SSL certificates:

```yaml
certificatesResolvers:
  letsencrypt:
    acme:
      tlsChallenge: {}
      email: admin@example.com
      storage: /data/acme.json
```

### Security Headers

Applied automatically via middleware:
- `X-Frame-Options: DENY`
- `X-Content-Type-Options: nosniff`
- `X-XSS-Protection: 1; mode=block`
- `Strict-Transport-Security: max-age=31536000`

### Rate Limiting

```yaml
# Per-service rate limiting
rateLimit:
  burst: 100
  average: 100
  period: "1m"
```

### IP Whitelisting

```yaml
apiVersion: traefik.containo.us/v1alpha1
kind: Middleware
metadata:
  name: ip-whitelist
spec:
  ipWhiteList:
    sourceRange:
      - "10.0.0.0/8"
      - "172.16.0.0/12"
      - "192.168.0.0/16"
```

## 🔧 Operations

### Dynamic Configuration Updates

Traefik supports real-time configuration updates:

```bash
# Update middleware
kubectl apply -f updated-middleware.yaml

# Configuration is automatically reloaded
# No restart required!

# Check configuration status
kubectl get middlewares -n rwwwrse
kubectl describe middleware security-headers -n rwwwrse
```

### Scaling

#### Horizontal Pod Autoscaler
```bash
# View HPA status
kubectl get hpa -n rwwwrse
kubectl describe hpa rwwwrse-hpa -n rwwwrse

# Manual scaling
kubectl scale deployment rwwwrse --replicas=5 -n rwwwrse
```

#### Traefik Controller Scaling
```bash
# Scale Traefik controller
kubectl scale deployment traefik --replicas=3 -n traefik-system

# Using Helm
helm upgrade traefik traefik/traefik \
    --namespace traefik-system \
    --set deployment.replicas=3
```

### Certificate Management

```bash
# Check certificate status
kubectl logs -n traefik-system deployment/traefik | grep -i acme

# Force certificate renewal (delete and let Traefik recreate)
kubectl delete secret rwwwrse-api-tls -n rwwwrse

# Monitor certificate creation
kubectl logs -n traefik-system deployment/traefik -f | grep -i certificate
```

### Updates and Rollouts

```bash
# Update rwwwrse deployment
kubectl set image deployment/rwwwrse rwwwrse=rwwwrse:v2.0.0 -n rwwwrse

# Monitor rollout
kubectl rollout status deployment/rwwwrse -n rwwwrse

# Rollback if needed
kubectl rollout undo deployment/rwwwrse -n rwwwrse
```

## 🔍 Troubleshooting

### Common Issues

#### 1. CRDs Not Found

```bash
# Check if Traefik CRDs are installed
kubectl get crd | grep traefik

# If missing, install CRDs
kubectl apply -f https://raw.githubusercontent.com/traefik/traefik/v3.0/docs/content/reference/dynamic-configuration/kubernetes-crd-definition-v1.yml
```

#### 2. IngressRoute Not Working

```bash
# Check IngressRoute status
kubectl get ingressroutes -n rwwwrse
kubectl describe ingressroute rwwwrse-api -n rwwwrse

# Check service endpoints
kubectl get endpoints rwwwrse -n rwwwrse

# Check Traefik logs
kubectl logs -n traefik-system deployment/traefik | grep -i error
```

#### 3. SSL Certificate Issues

```bash
# Check ACME logs
kubectl logs -n traefik-system deployment/traefik | grep -i acme

# Common issues:
# - DNS not pointing to Traefik IP
# - Firewall blocking port 80 (needed for TLS challenge)
# - Rate limiting from Let's Encrypt
# - Invalid email address in certificate resolver
```

#### 4. Middleware Not Applied

```bash
# Check middleware status
kubectl get middlewares -n rwwwrse
kubectl describe middleware security-headers -n rwwwrse

# Check if middleware is referenced in IngressRoute
kubectl get ingressroute rwwwrse-api -n rwwwrse -o yaml | grep -A5 middlewares
```

### Debugging Commands

```bash
# Check Traefik configuration
kubectl logs -n traefik-system deployment/traefik | grep -i "configuration received"

# Check service discovery
kubectl logs -n traefik-system deployment/traefik | grep -i "kubernetes"

# Test connectivity
kubectl run debug --rm -i --tty --image=curlimages/curl -- sh
# From inside pod: curl http://rwwwrse.rwwwrse.svc.cluster.local

# Check Traefik API
kubectl port-forward -n traefik-system service/traefik 8080:8080
curl http://localhost:8080/api/rawdata
```

### Performance Testing

```bash
# Install hey (HTTP load testing tool)
go install github.com/rakyll/hey@latest

# Test through Traefik
hey -z 30s -c 10 https://api.example.com/health

# Test backend directly (for comparison)
kubectl port-forward -n rwwwrse service/rwwwrse 8080:80
hey -z 30s -c 10 http://localhost:8080/health
```

## 🎨 Advanced Configuration

### Custom Error Pages

```yaml
apiVersion: traefik.containo.us/v1alpha1
kind: Middleware
metadata:
  name: error-pages
spec:
  errors:
    status:
      - "404"
      - "500-599"
    service:
      name: error-pages
      port: 80
    query: "/{status}.html"
```

### Request/Response Modification

```yaml
apiVersion: traefik.containo.us/v1alpha1
kind: Middleware
metadata:
  name: headers-modifier
spec:
  headers:
    customRequestHeaders:
      X-Custom-Header: "custom-value"
    customResponseHeaders:
      X-API-Version: "v1.0.0"
```

### Sticky Sessions

```yaml
apiVersion: traefik.containo.us/v1alpha1
kind: TraefikService
metadata:
  name: rwwwrse-sticky
spec:
  weighted:
    sticky:
      cookie:
        name: "server"
        secure: true
        httpOnly: true
    services:
    - name: rwwwrse
      port: 80
      weight: 1
```

### A/B Testing

```yaml
apiVersion: traefik.containo.us/v1alpha1
kind: IngressRoute
metadata:
  name: rwwwrse-ab-test
spec:
  entryPoints:
    - websecure
  routes:
  - match: Host(`app.example.com`) && Headers(`X-Version`, `beta`)
    kind: Rule
    services:
    - name: rwwwrse-beta
      port: 80
  - match: Host(`app.example.com`)
    kind: Rule
    services:
    - name: rwwwrse
      port: 80
```

## 🧹 Cleanup

### Remove rwwwrse

```bash
kubectl delete -f deployment.yaml
kubectl delete -f configmap.yaml
kubectl delete -f namespace.yaml
```

### Remove Traefik

```bash
# If installed with Helm
helm uninstall traefik -n traefik-system
kubectl delete namespace traefik-system

# If installed with kubectl
kubectl delete -f https://raw.githubusercontent.com/traefik/traefik/v3.0/docs/content/getting-started/install-traefik.yml
```

### Remove CRDs

```bash
kubectl delete -f https://raw.githubusercontent.com/traefik/traefik/v3.0/docs/content/reference/dynamic-configuration/kubernetes-crd-definition-v1.yml
```

## 📚 Additional Resources

- [Traefik Documentation](https://doc.traefik.io/traefik/)
- [Traefik Kubernetes CRD Reference](https://doc.traefik.io/traefik/routing/providers/kubernetes-crd/)
- [Traefik Middleware Reference](https://doc.traefik.io/traefik/middlewares/overview/)
- [Helm Chart Documentation](https://github.com/traefik/traefik-helm-chart)
- [Let's Encrypt Rate Limits](https://letsencrypt.org/docs/rate-limits/)

## 🆘 Getting Help

For Traefik-specific issues:

1. Check the [troubleshooting section](#-troubleshooting)
2. Review Traefik controller logs
3. Consult the [Traefik documentation](https://doc.traefik.io/traefik/)
4. Verify CRD installation and configuration
5. Check DNS configuration and certificate status

For rwwwrse-specific issues, refer to the [main documentation](../../../docs/DEPLOYMENT.md).

Remember to replace placeholder values (domain names, email addresses) with your actual configuration before deployment.