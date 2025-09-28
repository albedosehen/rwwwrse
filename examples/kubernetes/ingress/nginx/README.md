# rwwwrse with NGINX Ingress Controller

This example demonstrates deploying rwwwrse with NGINX Ingress Controller for advanced traffic management, SSL termination, and load balancing.

## 🏗️ Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                    Kubernetes Cluster                      │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐   │
│  │              NGINX Ingress Controller               │   │
│  │                                                     │   │
│  │  ┌─────────────┐    ┌─────────────────────────┐    │   │
│  │  │   External  │    │      SSL Termination   │    │   │
│  │  │Load Balancer│────│     & Traffic Routing  │    │   │
│  │  │   (Cloud)   │    │                         │    │   │
│  │  └─────────────┘    └─────────────────────────┘    │   │
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
│  │                                │                    │   │
│  │                                ▼                    │   │
│  │  ┌─────────────────────────────────────────────┐    │   │
│  │  │          Backend Services                   │    │   │
│  │  │  • API Service (port 3000)                 │    │   │
│  │  │  • Frontend Service (port 80)              │    │   │
│  │  │  • Web Service (port 8080)                 │    │   │
│  │  │  • Admin Service (port 9000)               │    │   │
│  │  └─────────────────────────────────────────────┘    │   │
│  └─────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
```

## 🎯 NGINX Ingress Features

- **SSL/TLS Termination**: Automatic HTTPS with cert-manager integration
- **Advanced Load Balancing**: Multiple algorithms (round-robin, IP hash, etc.)
- **Rate Limiting**: Per-IP and per-endpoint rate limiting
- **CORS Support**: Cross-origin resource sharing configuration
- **Security Headers**: Automatic security header injection
- **Compression**: Gzip compression for improved performance
- **Custom Error Pages**: Branded error pages for better UX
- **WebSocket Support**: Full WebSocket proxy capabilities
- **Path-based Routing**: Route traffic based on URL paths
- **Host-based Routing**: Route traffic based on host headers

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

### 1. Install NGINX Ingress Controller

#### Option A: Using Helm (Recommended)

```bash
# Add NGINX Ingress Helm repository
helm repo add ingress-nginx https://kubernetes.github.io/ingress-nginx
helm repo update

# Install NGINX Ingress Controller
helm install ingress-nginx ingress-nginx/ingress-nginx \
    --namespace ingress-nginx \
    --create-namespace \
    --set controller.service.type=LoadBalancer \
    --set controller.metrics.enabled=true \
    --set controller.metrics.serviceMonitor.enabled=true \
    --set controller.podSecurityContext.runAsUser=101 \
    --set controller.podSecurityContext.runAsGroup=82 \
    --set controller.podSecurityContext.fsGroup=82

# Verify installation
kubectl get pods -n ingress-nginx
kubectl get services -n ingress-nginx
```

#### Option B: Using Kubectl

```bash
# Apply NGINX Ingress Controller manifests
kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/controller-v1.8.2/deploy/static/provider/cloud/deploy.yaml

# Wait for deployment
kubectl wait --namespace ingress-nginx \
  --for=condition=ready pod \
  --selector=app.kubernetes.io/component=controller \
  --timeout=300s
```

### 2. Install cert-manager (Optional but Recommended)

```bash
# Install cert-manager for automatic SSL certificates
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.13.0/cert-manager.yaml

# Wait for cert-manager to be ready
kubectl wait --for=condition=ready pod -l app=cert-manager -n cert-manager --timeout=300s
kubectl wait --for=condition=ready pod -l app=webhook -n cert-manager --timeout=300s
kubectl wait --for=condition=ready pod -l app=cainjector -n cert-manager --timeout=300s
```

### 3. Create ClusterIssuer for Let's Encrypt

```bash
# Create Let's Encrypt ClusterIssuer
cat <<EOF | kubectl apply -f -
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: letsencrypt-prod
spec:
  acme:
    server: https://acme-v02.api.letsencrypt.org/directory
    email: admin@your-domain.com  # Replace with your email
    privateKeySecretRef:
      name: letsencrypt-prod
    solvers:
    - http01:
        ingress:
          class: nginx
EOF
```

### 4. Deploy rwwwrse

```bash
# Navigate to NGINX ingress example directory
cd examples/kubernetes/ingress/nginx

# Update configuration
# - Edit configmap.yaml to set your domain names
# - Edit deployment.yaml to set your container image
# - Edit ingress rules in deployment.yaml for your domains

# Deploy rwwwrse
kubectl apply -f namespace.yaml
kubectl apply -f configmap.yaml
kubectl apply -f deployment.yaml

# Verify deployment
kubectl get pods -n rwwwrse
kubectl get services -n rwwwrse
kubectl get ingress -n rwwwrse
```

### 5. Verify SSL Certificate

```bash
# Check certificate status
kubectl get certificates -n rwwwrse
kubectl describe certificate rwwwrse-tls-cert -n rwwwrse

# Check certificate order
kubectl get certificaterequests -n rwwwrse
kubectl get orders -n rwwwrse
```

## ⚙️ Configuration

### Ingress Configuration

The main ingress configuration is in [`deployment.yaml`](deployment.yaml):

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: rwwwrse-ingress
  annotations:
    kubernetes.io/ingress.class: "nginx"
    cert-manager.io/cluster-issuer: "letsencrypt-prod"
    nginx.ingress.kubernetes.io/ssl-redirect: "true"
    nginx.ingress.kubernetes.io/rate-limit: "100"
spec:
  tls:
  - hosts:
    - api.example.com
    - app.example.com
    secretName: rwwwrse-tls-cert
  rules:
  - host: api.example.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: rwwwrse
            port:
              number: 80
```

### Key Annotations

#### SSL/TLS Configuration
```yaml
nginx.ingress.kubernetes.io/ssl-redirect: "true"
nginx.ingress.kubernetes.io/force-ssl-redirect: "true"
nginx.ingress.kubernetes.io/ssl-protocols: "TLSv1.2 TLSv1.3"
cert-manager.io/cluster-issuer: "letsencrypt-prod"
```

#### Rate Limiting
```yaml
nginx.ingress.kubernetes.io/rate-limit: "100"
nginx.ingress.kubernetes.io/rate-limit-window: "1m"
nginx.ingress.kubernetes.io/rate-limit-connections: "10"
```

#### CORS Configuration
```yaml
nginx.ingress.kubernetes.io/enable-cors: "true"
nginx.ingress.kubernetes.io/cors-allow-origin: "https://app.example.com"
nginx.ingress.kubernetes.io/cors-allow-methods: "GET,POST,PUT,DELETE,OPTIONS"
nginx.ingress.kubernetes.io/cors-allow-headers: "Origin,Content-Type,Accept,Authorization"
```

#### Custom Headers
```yaml
nginx.ingress.kubernetes.io/configuration-snippet: |
  more_set_headers "X-Forwarded-Proto $scheme";
  more_set_headers "X-Real-IP $remote_addr";
  more_set_headers "X-Request-ID $request_id";
```

### Backend Protocol
```yaml
nginx.ingress.kubernetes.io/backend-protocol: "HTTP"
nginx.ingress.kubernetes.io/upstream-hash-by: "$request_uri"
```

## 📊 Monitoring and Observability

### NGINX Ingress Metrics

```bash
# Enable metrics collection
helm upgrade ingress-nginx ingress-nginx/ingress-nginx \
    --namespace ingress-nginx \
    --set controller.metrics.enabled=true \
    --set controller.metrics.serviceMonitor.enabled=true

# View ingress controller metrics
kubectl port-forward -n ingress-nginx service/ingress-nginx-controller-metrics 10254:10254
curl http://localhost:10254/metrics
```

### Prometheus Integration

The deployment includes ServiceMonitor for Prometheus:

```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: rwwwrse-metrics
spec:
  selector:
    matchLabels:
      app: rwwwrse
  endpoints:
  - port: metrics
    path: /metrics
    interval: 30s
```

### Key Metrics Available

- **Request Rate**: `nginx_ingress_controller_requests_total`
- **Request Duration**: `nginx_ingress_controller_request_duration_seconds`
- **Response Size**: `nginx_ingress_controller_response_size`
- **SSL Certificate Expiry**: `nginx_ingress_controller_ssl_expire_time_seconds`
- **Backend Connections**: `nginx_ingress_controller_nginx_process_connections`

### Logging

```bash
# View ingress controller logs
kubectl logs -n ingress-nginx deployment/ingress-nginx-controller -f

# View rwwwrse logs
kubectl logs -n rwwwrse deployment/rwwwrse -f

# View access logs (JSON format)
kubectl logs -n ingress-nginx deployment/ingress-nginx-controller -f | jq .
```

## 🔐 Security

### SSL/TLS Best Practices

- **TLS 1.2 and 1.3 Only**: Configured in NGINX
- **Strong Cipher Suites**: ECDHE with AES-GCM
- **HSTS Headers**: Automatically added
- **Certificate Auto-renewal**: Handled by cert-manager

### Rate Limiting

```yaml
# Per-IP rate limiting
nginx.ingress.kubernetes.io/rate-limit: "100"
nginx.ingress.kubernetes.io/rate-limit-window: "1m"

# Connection limiting
nginx.ingress.kubernetes.io/rate-limit-connections: "10"
```

### Security Headers

Automatically applied security headers:
- `X-Frame-Options: SAMEORIGIN`
- `X-Content-Type-Options: nosniff`
- `X-XSS-Protection: 1; mode=block`
- `Strict-Transport-Security: max-age=31536000`
- `Referrer-Policy: strict-origin-when-cross-origin`

### Authentication

```yaml
# Basic Auth example
nginx.ingress.kubernetes.io/auth-type: basic
nginx.ingress.kubernetes.io/auth-secret: basic-auth
nginx.ingress.kubernetes.io/auth-realm: 'Authentication Required'

# OAuth2 integration
nginx.ingress.kubernetes.io/auth-url: "https://oauth2.example.com/auth"
nginx.ingress.kubernetes.io/auth-signin: "https://oauth2.example.com/start"
```

## 🔧 Operations

### Scaling

#### Horizontal Pod Autoscaler
```bash
# View HPA status
kubectl get hpa -n rwwwrse
kubectl describe hpa rwwwrse-hpa -n rwwwrse

# Manual scaling
kubectl scale deployment rwwwrse --replicas=5 -n rwwwrse
```

#### NGINX Ingress Controller Scaling
```bash
# Scale ingress controller
kubectl scale deployment ingress-nginx-controller --replicas=3 -n ingress-nginx

# Using Helm
helm upgrade ingress-nginx ingress-nginx/ingress-nginx \
    --namespace ingress-nginx \
    --set controller.replicaCount=3
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

### Certificate Management

```bash
# Check certificate status
kubectl get certificates -n rwwwrse
kubectl describe certificate rwwwrse-tls-cert -n rwwwrse

# Force certificate renewal
kubectl delete secret rwwwrse-tls-cert -n rwwwrse
# cert-manager will automatically recreate it
```

## 🔍 Troubleshooting

### Common Issues

#### 1. Ingress Not Getting External IP

```bash
# Check service status
kubectl get service ingress-nginx-controller -n ingress-nginx

# Check cloud provider load balancer
# AWS
aws elbv2 describe-load-balancers

# GCP
gcloud compute forwarding-rules list

# Azure
az network lb list
```

#### 2. SSL Certificate Issues

```bash
# Check certificate order
kubectl get certificaterequests -n rwwwrse
kubectl describe certificaterequest -n rwwwrse

# Check ACME challenge
kubectl get challenges -n rwwwrse
kubectl describe challenge -n rwwwrse

# Common issues:
# - DNS not pointing to ingress IP
# - Firewall blocking port 80 (needed for HTTP-01 challenge)
# - Rate limiting from Let's Encrypt
```

#### 3. Backend Connection Errors

```bash
# Check service endpoints
kubectl get endpoints rwwwrse -n rwwwrse

# Check pod readiness
kubectl get pods -n rwwwrse
kubectl describe pod -l app=rwwwrse -n rwwwrse

# Check service selector
kubectl get service rwwwrse -n rwwwrse -o yaml
```

#### 4. Rate Limiting Issues

```bash
# Check rate limiting logs
kubectl logs -n ingress-nginx deployment/ingress-nginx-controller | grep "limiting"

# Adjust rate limits
kubectl annotate ingress rwwwrse-ingress -n rwwwrse \
    nginx.ingress.kubernetes.io/rate-limit="200"
```

### Debugging Commands

```bash
# Check ingress status
kubectl get ingress -n rwwwrse
kubectl describe ingress rwwwrse-ingress -n rwwwrse

# Check ingress controller logs
kubectl logs -n ingress-nginx deployment/ingress-nginx-controller -f

# Check ingress controller events
kubectl get events -n ingress-nginx --sort-by='.lastTimestamp'

# Test connectivity
kubectl run debug --rm -i --tty --image=curlimages/curl -- sh
# From inside pod: curl http://rwwwrse.rwwwrse.svc.cluster.local
```

### Performance Testing

```bash
# Install hey (HTTP load testing tool)
go install github.com/rakyll/hey@latest

# Test ingress performance
hey -z 30s -c 10 https://api.example.com/health

# Test backend directly (for comparison)
kubectl port-forward -n rwwwrse service/rwwwrse 8080:80
hey -z 30s -c 10 http://localhost:8080/health
```

## 🎨 Customization

### Custom Error Pages

```bash
# Create custom error page ConfigMap
kubectl create configmap custom-error-pages \
    --from-file=404.html \
    --from-file=500.html \
    -n ingress-nginx

# Update ingress controller
helm upgrade ingress-nginx ingress-nginx/ingress-nginx \
    --namespace ingress-nginx \
    --set controller.defaultBackend.enabled=true \
    --set controller.defaultBackend.image.repository=custom-error-pages \
    --set controller.defaultBackend.image.tag=latest
```

### Custom NGINX Configuration

Edit the nginx-configuration ConfigMap in [`configmap.yaml`](configmap.yaml):

```yaml
data:
  proxy-body-size: "50m"
  proxy-connect-timeout: "60"
  proxy-send-timeout: "120"
  proxy-read-timeout: "120"
  ssl-session-cache: "shared:SSL:20m"
  ssl-session-timeout: "15m"
```

### WebSocket Support

```yaml
# Add WebSocket annotations
nginx.ingress.kubernetes.io/proxy-read-timeout: "3600"
nginx.ingress.kubernetes.io/proxy-send-timeout: "3600"
nginx.ingress.kubernetes.io/configuration-snippet: |
  proxy_set_header Upgrade $http_upgrade;
  proxy_set_header Connection "upgrade";
```

## 🧹 Cleanup

### Remove rwwwrse

```bash
kubectl delete -f deployment.yaml
kubectl delete -f configmap.yaml
kubectl delete -f namespace.yaml
```

### Remove NGINX Ingress Controller

```bash
# If installed with Helm
helm uninstall ingress-nginx -n ingress-nginx
kubectl delete namespace ingress-nginx

# If installed with kubectl
kubectl delete -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/controller-v1.8.2/deploy/static/provider/cloud/deploy.yaml
```

### Remove cert-manager

```bash
kubectl delete -f https://github.com/cert-manager/cert-manager/releases/download/v1.13.0/cert-manager.yaml
```

## 📚 Additional Resources

- [NGINX Ingress Controller Documentation](https://kubernetes.github.io/ingress-nginx/)
- [cert-manager Documentation](https://cert-manager.io/docs/)
- [Kubernetes Ingress Documentation](https://kubernetes.io/docs/concepts/services-networking/ingress/)
- [NGINX Configuration Reference](https://nginx.org/en/docs/)
- [Let's Encrypt Rate Limits](https://letsencrypt.org/docs/rate-limits/)

## 🆘 Getting Help

For NGINX Ingress specific issues:

1. Check the [troubleshooting section](#-troubleshooting)
2. Review NGINX Ingress Controller logs
3. Consult the [NGINX Ingress documentation](https://kubernetes.github.io/ingress-nginx/)
4. Check certificate status if using HTTPS
5. Verify DNS configuration and firewall rules

For rwwwrse-specific issues, refer to the [main documentation](../../../docs/DEPLOYMENT.md).

Remember to replace placeholder values (domain names, email addresses) with your actual configuration before deployment.