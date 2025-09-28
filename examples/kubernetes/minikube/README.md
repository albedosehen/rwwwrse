# rwwwrse on Kubernetes (Minikube)

This example demonstrates deploying rwwwrse on a local Kubernetes cluster using Minikube. It includes a complete setup with sample applications, monitoring, and service discovery.

## 🏗️ Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                    Minikube Cluster                        │
│  ┌─────────────────────────────────────────────────────┐   │
│  │                rwwwrse Namespace                   │   │
│  │                                                     │   │
│  │  ┌─────────────┐    ┌─────────────────────────┐    │   │
│  │  │  rwwwrse    │    │    Sample Applications  │    │   │
│  │  │  Reverse    │────│                         │    │   │
│  │  │  Proxy      │    │  ┌─────┐ ┌─────┐ ┌────┐ │    │   │
│  │  │             │    │  │App1 │ │App2 │ │API │ │    │   │
│  │  └─────────────┘    │  └─────┘ └─────┘ └────┘ │    │   │
│  │                     │                         │    │   │
│  │  ┌─────────────┐    │  ┌─────────────────────┐ │    │   │
│  │  │ Prometheus  │    │  │  Static Content     │ │    │   │
│  │  │ Monitoring  │    │  │  (nginx)           │ │    │   │
│  │  └─────────────┘    │  └─────────────────────┘ │    │   │
│  └─────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
```

## 🎯 Features

- **Multi-Service Routing**: Host-based routing to different sample applications
- **Service Discovery**: Automatic discovery of Kubernetes services
- **Health Monitoring**: Comprehensive health checks and readiness probes
- **Metrics Collection**: Prometheus monitoring with service discovery
- **Resource Management**: CPU/memory limits and requests
- **Configuration Management**: ConfigMaps and Secrets
- **Development Ready**: Self-signed certificates and local development setup

## 📋 Prerequisites

### Software Requirements

```bash
# Minikube
minikube version  # >= 1.25.0

# kubectl
kubectl version --client  # >= 1.20.0

# Docker (for building images)
docker --version  # >= 20.10.0
```

### System Requirements

- **CPU**: 2+ cores available to Minikube
- **Memory**: 4GB+ RAM available to Minikube
- **Disk**: 10GB+ free space

## 🚀 Quick Start

### 1. Start Minikube

```bash
# Start Minikube with adequate resources
minikube start --cpus=4 --memory=4096 --disk-size=20g

# Enable required addons
minikube addons enable metrics-server
minikube addons enable ingress

# Verify cluster is running
kubectl cluster-info
```

### 2. Build rwwwrse Image (if needed)

```bash
# Build the rwwwrse image and load into Minikube
# (Run from the root of the rwwwrse project)
eval $(minikube docker-env)
docker build -t rwwwrse:latest .

# Or pull from registry if available
# docker pull your-registry/rwwwrse:latest
# docker tag your-registry/rwwwrse:latest rwwwrse:latest
```

### 3. Deploy the Stack

```bash
# Navigate to the minikube example directory
cd examples/kubernetes/minikube

# Apply all manifests in order
kubectl apply -f namespace.yaml
kubectl apply -f secrets.yaml
kubectl apply -f configmap.yaml
kubectl apply -f content.yaml
kubectl apply -f deployment.yaml

# Verify deployment
kubectl get pods -n rwwwrse
kubectl get services -n rwwwrse
```

### 4. Access the Applications

```bash
# Get Minikube IP
MINIKUBE_IP=$(minikube ip)

# Add entries to /etc/hosts for local development
echo "$MINIKUBE_IP app1.minikube.local" | sudo tee -a /etc/hosts
echo "$MINIKUBE_IP app2.minikube.local" | sudo tee -a /etc/hosts
echo "$MINIKUBE_IP api.minikube.local" | sudo tee -a /etc/hosts
echo "$MINIKUBE_IP static.minikube.local" | sudo tee -a /etc/hosts

# Access applications through rwwwrse
curl http://app1.minikube.local:30080
curl http://app2.minikube.local:30080
curl http://api.minikube.local:30080/api/health
curl http://static.minikube.local:30080

# Access monitoring
kubectl port-forward -n rwwwrse service/prometheus 9090:9090 &
# Open http://localhost:9090 in browser
```

## ⚙️ Configuration

### rwwwrse Configuration

The rwwwrse proxy is configured via the [`configmap.yaml`](configmap.yaml) file:

```yaml
# Core configuration
RWWWRSE_LOG_LEVEL: "info"
RWWWRSE_LOG_FORMAT: "json"
RWWWRSE_METRICS_ENABLED: "true"
RWWWRSE_RATE_LIMIT: "100"

# Route configuration in routes.yaml
routes:
  - host: "app1.minikube.local"
    target: "http://sample-app1:8080"
    health_check: "/health"
    timeout: "30s"
```

### Service Discovery

Services are automatically discovered through Kubernetes DNS:

- **sample-app1** → `http://sample-app1:8080`
- **sample-app2** → `http://sample-app2:8080`
- **sample-api** → `http://sample-api:3000`
- **sample-static** → `http://sample-static:80`

### Resource Limits

Each service has defined resource requests and limits:

```yaml
resources:
  requests:
    cpu: 100m
    memory: 128Mi
  limits:
    cpu: 500m
    memory: 512Mi
```

## 📊 Monitoring

### Prometheus Metrics

Access Prometheus at `http://localhost:9090` (after port-forwarding):

```bash
kubectl port-forward -n rwwwrse service/prometheus 9090:9090
```

### Key Metrics to Monitor

```prometheus
# rwwwrse request rate
rate(http_requests_total[5m])

# Response time percentiles
histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))

# Error rate
rate(http_requests_total{status=~"5.."}[5m]) / rate(http_requests_total[5m])

# Pod resource usage
container_memory_usage_bytes{namespace="rwwwrse"}
container_cpu_usage_seconds_total{namespace="rwwwrse"}
```

### Service Discovery Monitoring

Prometheus automatically discovers services with annotations:

```yaml
annotations:
  prometheus.io/scrape: "true"
  prometheus.io/port: "9090"
  prometheus.io/path: "/metrics"
```

## 🔧 Development

### Updating rwwwrse Configuration

```bash
# Edit the ConfigMap
kubectl edit configmap rwwwrse-config -n rwwwrse

# Restart rwwwrse pods to pick up changes
kubectl rollout restart deployment rwwwrse -n rwwwrse

# Monitor rollout
kubectl rollout status deployment rwwwrse -n rwwwrse
```

### Adding New Routes

1. Update the routes in [`configmap.yaml`](configmap.yaml):

```yaml
routes:
  - host: "newapp.minikube.local"
    target: "http://new-service:8080"
    health_check: "/health"
    timeout: "30s"
```

2. Apply the changes:

```bash
kubectl apply -f configmap.yaml
kubectl rollout restart deployment rwwwrse -n rwwwrse
```

3. Add the hostname to `/etc/hosts`:

```bash
echo "$(minikube ip) newapp.minikube.local" | sudo tee -a /etc/hosts
```

### Scaling Services

```bash
# Scale rwwwrse replicas
kubectl scale deployment rwwwrse --replicas=3 -n rwwwrse

# Scale sample applications
kubectl scale deployment sample-app1 --replicas=2 -n rwwwrse
kubectl scale deployment sample-app2 --replicas=2 -n rwwwrse

# Verify scaling
kubectl get pods -n rwwwrse
```

## 🔍 Troubleshooting

### Common Issues

#### rwwwrse Pods Not Starting

```bash
# Check pod status
kubectl get pods -n rwwwrse

# View pod logs
kubectl logs -l app=rwwwrse -n rwwwrse

# Describe pod for events
kubectl describe pod -l app=rwwwrse -n rwwwrse

# Common fixes:
# 1. Ensure rwwwrse:latest image is available in Minikube
# 2. Check resource limits
# 3. Verify ConfigMap syntax
```

#### Cannot Access Applications

```bash
# Check service endpoints
kubectl get endpoints -n rwwwrse

# Verify NodePort services
kubectl get services -n rwwwrse -o wide

# Test internal connectivity
kubectl exec -it deployment/rwwwrse -n rwwwrse -- wget -qO- http://sample-app1:8080/health

# Check /etc/hosts entries
grep minikube.local /etc/hosts
```

#### Prometheus Not Scraping Metrics

```bash
# Check Prometheus configuration
kubectl get configmap prometheus-config -n rwwwrse -o yaml

# View Prometheus logs
kubectl logs deployment/prometheus -n rwwwrse

# Check service discovery
kubectl get pods -n rwwwrse --show-labels
```

### Debugging Commands

```bash
# View all resources in namespace
kubectl get all -n rwwwrse

# Check resource usage
kubectl top pods -n rwwwrse
kubectl top nodes

# View events
kubectl get events -n rwwwrse --sort-by='.lastTimestamp'

# Port forward for debugging
kubectl port-forward -n rwwwrse deployment/rwwwrse 8080:8080
kubectl port-forward -n rwwwrse deployment/sample-api 3000:3000
```

### Logs Analysis

```bash
# View rwwwrse logs
kubectl logs -f deployment/rwwwrse -n rwwwrse

# View application logs
kubectl logs -f deployment/sample-app1 -n rwwwrse
kubectl logs -f deployment/sample-api -n rwwwrse

# View all logs with labels
kubectl logs -l app=rwwwrse -n rwwwrse --tail=100
```

## 🧪 Testing

### Health Checks

```bash
# Test rwwwrse health
curl http://$(minikube ip):30080/health

# Test individual services
kubectl exec -it deployment/rwwwrse -n rwwwrse -- \
  wget -qO- http://sample-app1:8080/health

kubectl exec -it deployment/rwwwrse -n rwwwrse -- \
  wget -qO- http://sample-api:3000/api/health
```

### Load Testing

```bash
# Simple load test with kubectl exec
kubectl exec -it deployment/rwwwrse -n rwwwrse -- \
  sh -c 'for i in $(seq 1 100); do wget -qO- http://sample-app1:8080/ && echo; done'

# Using hey (if available)
hey -n 1000 -c 10 http://app1.minikube.local:30080
```

### Metrics Validation

```bash
# Check metrics endpoint
curl http://$(minikube ip):30090/metrics

# Query Prometheus
kubectl port-forward -n rwwwrse service/prometheus 9090:9090 &
curl 'http://localhost:9090/api/v1/query?query=up'
```

## 🔄 Updates and Maintenance

### Updating rwwwrse

```bash
# Build new image
eval $(minikube docker-env)
docker build -t rwwwrse:v2.0.0 .
docker tag rwwwrse:v2.0.0 rwwwrse:latest

# Update deployment
kubectl set image deployment/rwwwrse rwwwrse=rwwwrse:latest -n rwwwrse

# Monitor rollout
kubectl rollout status deployment/rwwwrse -n rwwwrse
```

### Resource Cleanup

```bash
# Clean up the namespace
kubectl delete namespace rwwwrse

# Or delete individual resources
kubectl delete -f deployment.yaml
kubectl delete -f configmap.yaml
kubectl delete -f content.yaml
kubectl delete -f secrets.yaml
kubectl delete -f namespace.yaml

# Clean up /etc/hosts entries
sudo sed -i '/minikube.local/d' /etc/hosts
```

### Minikube Cleanup

```bash
# Stop Minikube
minikube stop

# Delete Minikube cluster
minikube delete

# Clean up Minikube profiles
minikube profile list
minikube delete -p <profile-name>
```

## 📚 File Structure

```
examples/kubernetes/minikube/
├── README.md              # This documentation
├── namespace.yaml          # Namespace, ResourceQuota, LimitRange
├── secrets.yaml           # TLS certificates and secrets
├── configmap.yaml         # rwwwrse config, Prometheus config, app configs
├── content.yaml           # HTML content for sample applications
├── deployment.yaml        # All deployments and services
└── scripts/
    ├── deploy.sh          # Complete deployment script
    ├── clean.sh           # Cleanup script
    └── port-forward.sh    # Port forwarding helper
```

## 🔗 Useful Commands

### Quick Access URLs

```bash
# Store Minikube IP
export MINIKUBE_IP=$(minikube ip)

# Application URLs
echo "App 1: http://app1.minikube.local:30080"
echo "App 2: http://app2.minikube.local:30080"
echo "API: http://api.minikube.local:30080/api"
echo "Static: http://static.minikube.local:30080"
echo "Prometheus: http://localhost:9090 (after port-forward)"
```

### Port Forwarding

```bash
# rwwwrse proxy
kubectl port-forward -n rwwwrse service/rwwwrse 8080:80

# Prometheus monitoring
kubectl port-forward -n rwwwrse service/prometheus 9090:9090

# Direct service access
kubectl port-forward -n rwwwrse service/sample-app1 8081:8080
kubectl port-forward -n rwwwrse service/sample-api 3000:3000
```

### Development Shortcuts

```bash
# Quick restart all services
kubectl rollout restart deployment -n rwwwrse

# Watch pod status
kubectl get pods -n rwwwrse -w

# Stream logs from all rwwwrse pods
kubectl logs -f -l app=rwwwrse -n rwwwrse

# Quick status check
kubectl get all -n rwwwrse
```

## 🆘 Getting Help

For issues with this Minikube deployment:

1. Check the [troubleshooting section](#-troubleshooting)
2. Review pod logs: `kubectl logs -l app=rwwwrse -n rwwwrse`
3. Verify Minikube status: `minikube status`
4. Check resource usage: `kubectl top pods -n rwwwrse`
5. Consult the [main documentation](../../docs/DEPLOYMENT.md)

For Minikube-specific issues:
- [Minikube Documentation](https://minikube.sigs.k8s.io/docs/)
- [Kubernetes Troubleshooting](https://kubernetes.io/docs/tasks/debug-application-cluster/)

Remember to sanitize sensitive information before sharing logs or configurations.