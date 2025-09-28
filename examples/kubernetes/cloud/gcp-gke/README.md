# rwwwrse on Google Kubernetes Engine (GKE)

This example demonstrates deploying rwwwrse on Google Kubernetes Engine with GCP-specific integrations and best practices.

## 🏗️ Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                      Google Cloud Platform                 │
│  ┌─────────────────────────────────────────────────────┐   │
│  │                  GKE Cluster                       │   │
│  │                                                     │   │
│  │  ┌─────────────┐    ┌─────────────────────────┐    │   │
│  │  │  Google     │    │      rwwwrse Pods      │    │   │
│  │  │Cloud Load   │────│   (Auto-scaled 3-100)  │    │   │
│  │  │Balancer     │    │                         │    │   │
│  │  │  (GCLB)     │    └─────────────────────────┘    │   │
│  │  └─────────────┘                                    │   │
│  │                     ┌─────────────────────────┐    │   │
│  │  ┌─────────────┐    │    Sample Applications  │    │   │
│  │  │  Container  │    │   (Auto-scaled 2-50)   │    │   │
│  │  │  Registry   │    │                         │    │   │
│  │  │   (GCR)     │    └─────────────────────────┘    │   │
│  │  └─────────────┘                                    │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐   │
│  │                GCP Integrations                     │   │
│  │  • Workload Identity                               │   │
│  │  • Cloud DNS                                       │   │
│  │  • Certificate Manager                             │   │
│  │  • Cloud Monitoring & Logging                      │   │
│  │  • Cloud CDN                                       │   │
│  │  • VPC Native Networking                           │   │
│  └─────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
```

## 🎯 GCP-Specific Features

- **Google Cloud Load Balancer (GCLB)**: Global HTTP(S) load balancing
- **Workload Identity**: Secure service-to-service authentication
- **Container Registry (GCR)**: Private container image storage
- **Cloud DNS**: Managed DNS service
- **Cloud Monitoring**: Advanced metrics and alerting
- **Cloud Logging**: Centralized log management
- **VPC Native Networking**: Advanced networking features
- **Vertical Pod Autoscaler**: Automatic resource optimization

## 📋 Prerequisites

### GCP Setup

```bash
# Required tools
gcloud version  # >= 400.0.0
kubectl version --client  # >= 1.20.0
docker --version  # >= 20.10.0
```

### GCP APIs and Permissions

```bash
# Enable required APIs
gcloud services enable container.googleapis.com
gcloud services enable dns.googleapis.com
gcloud services enable monitoring.googleapis.com
gcloud services enable logging.googleapis.com
gcloud services enable containerregistry.googleapis.com
gcloud services enable certificatemanager.googleapis.com

# Set project
gcloud config set project PROJECT_ID
```

### Required IAM Roles

The deploying user needs:
- `roles/container.admin`
- `roles/iam.serviceAccountAdmin`
- `roles/dns.admin`
- `roles/monitoring.editor`
- `roles/logging.admin`

## 🚀 Quick Start

### 1. Create GKE Cluster

```bash
# Set variables
export PROJECT_ID="your-project-id"
export CLUSTER_NAME="rwwwrse-cluster"
export REGION="us-central1"

# Create GKE cluster with recommended settings
gcloud container clusters create $CLUSTER_NAME \
    --region=$REGION \
    --machine-type=e2-medium \
    --num-nodes=1 \
    --min-nodes=1 \
    --max-nodes=10 \
    --enable-autoscaling \
    --enable-autorepair \
    --enable-autoupgrade \
    --enable-ip-alias \
    --network=default \
    --subnetwork=default \
    --enable-workload-identity \
    --enable-shielded-nodes \
    --disk-type=pd-ssd \
    --disk-size=50GB \
    --enable-network-policy \
    --maintenance-window-start=2023-01-01T09:00:00Z \
    --maintenance-window-end=2023-01-01T17:00:00Z \
    --maintenance-window-recurrence="FREQ=WEEKLY;BYDAY=SA,SU" \
    --addons=HorizontalPodAutoscaling,HttpLoadBalancing,NetworkPolicy

# Get cluster credentials
gcloud container clusters get-credentials $CLUSTER_NAME --region=$REGION
```

### 2. Set Up Workload Identity

```bash
# Create Google Service Account
gcloud iam service-accounts create rwwwrse-sa \
    --display-name="rwwwrse Service Account"

# Grant necessary permissions
gcloud projects add-iam-policy-binding $PROJECT_ID \
    --member="serviceAccount:rwwwrse-sa@$PROJECT_ID.iam.gserviceaccount.com" \
    --role="roles/monitoring.metricWriter"

gcloud projects add-iam-policy-binding $PROJECT_ID \
    --member="serviceAccount:rwwwrse-sa@$PROJECT_ID.iam.gserviceaccount.com" \
    --role="roles/logging.logWriter"

# Enable Workload Identity on cluster
gcloud container clusters update $CLUSTER_NAME \
    --region=$REGION \
    --workload-pool=$PROJECT_ID.svc.id.goog

# Create namespace
kubectl create namespace rwwwrse

# Create Kubernetes Service Account
kubectl create serviceaccount rwwwrse-ksa -n rwwwrse

# Bind Kubernetes SA to Google SA
gcloud iam service-accounts add-iam-policy-binding \
    rwwwrse-sa@$PROJECT_ID.iam.gserviceaccount.com \
    --role="roles/iam.workloadIdentityUser" \
    --member="serviceAccount:$PROJECT_ID.svc.id.goog[rwwwrse/rwwwrse-ksa]"

# Annotate Kubernetes SA
kubectl annotate serviceaccount rwwwrse-ksa \
    -n rwwwrse \
    iam.gke.io/gcp-service-account=rwwwrse-sa@$PROJECT_ID.iam.gserviceaccount.com
```

### 3. Build and Push Container Image

```bash
# Configure Docker for GCR
gcloud auth configure-docker

# Build and push image
docker build -t gcr.io/$PROJECT_ID/rwwwrse:latest .
docker push gcr.io/$PROJECT_ID/rwwwrse:latest
```

### 4. Deploy rwwwrse

```bash
# Navigate to GKE example directory
cd examples/kubernetes/cloud/gcp-gke

# Update configuration files with your values
# - Replace PROJECT_ID with your actual project ID
# - Update domain names and other configurations

# Create secrets
kubectl create secret generic rwwwrse-secrets \
    --from-literal=tls-email=admin@your-domain.com \
    -n rwwwrse

# Apply manifests
kubectl apply -f namespace.yaml
kubectl apply -f configmap.yaml
kubectl apply -f deployment.yaml

# Verify deployment
kubectl get pods -n rwwwrse
kubectl get services -n rwwwrse
```

### 5. Set Up Managed SSL Certificate

```bash
# Create managed SSL certificate
gcloud compute ssl-certificates create rwwwrse-ssl-cert \
    --domains=your-domain.com,www.your-domain.com \
    --global

# Update ingress to use the certificate
kubectl apply -f ingress.yaml
```

## ⚙️ Configuration

### Container Registry Integration

Update the image reference in [`deployment.yaml`](deployment.yaml):

```yaml
image: gcr.io/PROJECT_ID/rwwwrse:latest
```

### Workload Identity Configuration

Service account annotation in deployment:

```yaml
serviceAccountName: rwwwrse-ksa
annotations:
  iam.gke.io/gcp-service-account: rwwwrse-sa@PROJECT_ID.iam.gserviceaccount.com
```

### Auto Scaling Configuration

#### Horizontal Pod Autoscaler (HPA)
- **Min replicas**: 3
- **Max replicas**: 100
- **CPU target**: 70%
- **Memory target**: 80%

#### Vertical Pod Autoscaler (VPA)
GKE-specific feature for automatic resource optimization:

```yaml
apiVersion: autoscaling.k8s.io/v1
kind: VerticalPodAutoscaler
metadata:
  name: rwwwrse-vpa
spec:
  updatePolicy:
    updateMode: "Auto"
```

#### Cluster Autoscaler
Automatically managed by GKE based on cluster creation settings.

## 📊 Monitoring and Observability

### Cloud Monitoring Integration

rwwwrse metrics are automatically collected by Cloud Monitoring:

```bash
# View metrics in Cloud Console
gcloud alpha monitoring dashboards list

# Create custom dashboard
gcloud alpha monitoring dashboards create --config-from-file=dashboard.json
```

### Cloud Logging

Logs are automatically sent to Cloud Logging:

```bash
# View logs
gcloud logging read "resource.type=k8s_container AND resource.labels.container_name=rwwwrse" \
    --limit=50 \
    --format=json

# Create log-based metrics
gcloud logging metrics create rwwwrse_error_rate \
    --description="rwwwrse error rate" \
    --log-filter='resource.type=k8s_container AND resource.labels.container_name=rwwwrse AND severity>=ERROR'
```

### Key Metrics Available

- Standard Kubernetes metrics via GKE monitoring
- Custom application metrics from rwwwrse
- Load balancer metrics from GCLB
- Network metrics from VPC

## 🔐 Security

### Workload Identity

Secure access to GCP services without storing service account keys:

```yaml
annotations:
  iam.gke.io/gcp-service-account: rwwwrse-sa@PROJECT_ID.iam.gserviceaccount.com
```

### Shielded GKE Nodes

Nodes are protected against rootkits and bootkits:

```bash
# Verify shielded nodes
gcloud container node-pools describe default-pool \
    --cluster=$CLUSTER_NAME \
    --region=$REGION
```

### Pod Security Standards

```yaml
securityContext:
  runAsNonRoot: true
  runAsUser: 65534
  allowPrivilegeEscalation: false
  readOnlyRootFilesystem: true
```

### Network Security

- VPC-native networking for better isolation
- Network policies for pod-to-pod communication control
- Private cluster option available

## 🔧 Operations

### Cluster Management

```bash
# Scale cluster
gcloud container clusters resize $CLUSTER_NAME \
    --num-nodes=5 \
    --region=$REGION

# Upgrade cluster
gcloud container clusters upgrade $CLUSTER_NAME \
    --region=$REGION \
    --master

# Update node pools
gcloud container node-pools upgrade default-pool \
    --cluster=$CLUSTER_NAME \
    --region=$REGION
```

### Application Management

```bash
# Update application
kubectl set image deployment/rwwwrse \
    rwwwrse=gcr.io/$PROJECT_ID/rwwwrse:v2.0.0 \
    -n rwwwrse

# Monitor rollout
kubectl rollout status deployment/rwwwrse -n rwwwrse

# Rollback
kubectl rollout undo deployment/rwwwrse -n rwwwrse
```

### Load Balancer Management

```bash
# Check load balancer status
kubectl get service rwwwrse -n rwwwrse -o wide

# Get external IP
kubectl get service rwwwrse -n rwwwrse -o jsonpath='{.status.loadBalancer.ingress[0].ip}'

# View load balancer details
gcloud compute forwarding-rules list
gcloud compute backend-services list
```

## 🔍 Troubleshooting

### Common Issues

#### Pods Not Starting

```bash
# Check pod status
kubectl get pods -n rwwwrse
kubectl describe pod -l app=rwwwrse -n rwwwrse

# Check logs
kubectl logs -l app=rwwwrse -n rwwwrse

# Common causes:
# - Image pull errors (check GCR permissions)
# - Workload Identity misconfiguration
# - Resource constraints
# - Network policy issues
```

#### Load Balancer Issues

```bash
# Check service events
kubectl describe service rwwwrse -n rwwwrse

# Check backend health
gcloud compute backend-services get-health [BACKEND_SERVICE_NAME] --global

# Check firewall rules
gcloud compute firewall-rules list --filter="name~gke"
```

#### Workload Identity Issues

```bash
# Test Workload Identity
kubectl run -it --rm debug \
    --image=google/cloud-sdk:slim \
    --serviceaccount=rwwwrse-ksa \
    --namespace=rwwwrse \
    -- gcloud auth list

# Check IAM bindings
gcloud iam service-accounts get-iam-policy \
    rwwwrse-sa@$PROJECT_ID.iam.gserviceaccount.com
```

### Log Analysis

```bash
# Application logs
kubectl logs -f deployment/rwwwrse -n rwwwrse

# All rwwwrse pods
kubectl logs -f -l app=rwwwrse -n rwwwrse

# Cloud Logging
gcloud logging read "resource.type=k8s_container" \
    --limit=50 \
    --format="value(textPayload)"
```

## 💰 Cost Optimization

### Preemptible Nodes

```bash
# Create preemptible node pool
gcloud container node-pools create preemptible-pool \
    --cluster=$CLUSTER_NAME \
    --region=$REGION \
    --machine-type=e2-medium \
    --preemptible \
    --num-nodes=1 \
    --enable-autoscaling \
    --min-nodes=0 \
    --max-nodes=10
```

### Resource Optimization

```bash
# Use VPA recommendations
kubectl describe vpa rwwwrse-vpa -n rwwwrse

# Monitor resource usage
kubectl top pods -n rwwwrse
kubectl top nodes
```

### Sustained Use Discounts

GKE automatically applies sustained use discounts for long-running workloads.

## 🔄 CI/CD Integration

### Cloud Build Example

```yaml
# cloudbuild.yaml
steps:
- name: 'gcr.io/cloud-builders/docker'
  args: ['build', '-t', 'gcr.io/$PROJECT_ID/rwwwrse:$COMMIT_SHA', '.']

- name: 'gcr.io/cloud-builders/docker'
  args: ['push', 'gcr.io/$PROJECT_ID/rwwwrse:$COMMIT_SHA']

- name: 'gcr.io/cloud-builders/kubectl'
  args:
  - 'set'
  - 'image'
  - 'deployment/rwwwrse'
  - 'rwwwrse=gcr.io/$PROJECT_ID/rwwwrse:$COMMIT_SHA'
  - '-n'
  - 'rwwwrse'
  env:
  - 'CLOUDSDK_COMPUTE_REGION=us-central1'
  - 'CLOUDSDK_CONTAINER_CLUSTER=rwwwrse-cluster'
```

### GitHub Actions with GKE

```yaml
name: Deploy to GKE
on:
  push:
    branches: [main]

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v3
    
    - uses: google-github-actions/setup-gcloud@v1
      with:
        service_account_key: ${{ secrets.GCP_SA_KEY }}
        project_id: ${{ secrets.GCP_PROJECT_ID }}
    
    - run: gcloud auth configure-docker
    
    - name: Build and push image
      run: |
        docker build -t gcr.io/${{ secrets.GCP_PROJECT_ID }}/rwwwrse:${{ github.sha }} .
        docker push gcr.io/${{ secrets.GCP_PROJECT_ID }}/rwwwrse:${{ github.sha }}
    
    - name: Deploy to GKE
      run: |
        gcloud container clusters get-credentials rwwwrse-cluster --region=us-central1
        kubectl set image deployment/rwwwrse rwwwrse=gcr.io/${{ secrets.GCP_PROJECT_ID }}/rwwwrse:${{ github.sha }} -n rwwwrse
        kubectl rollout status deployment/rwwwrse -n rwwwrse
```

## 🧹 Cleanup

### Remove Application

```bash
# Delete rwwwrse resources
kubectl delete -f deployment.yaml
kubectl delete -f configmap.yaml
kubectl delete -f namespace.yaml
```

### Remove Cluster

```bash
# Delete the GKE cluster
gcloud container clusters delete $CLUSTER_NAME --region=$REGION

# Clean up service accounts
gcloud iam service-accounts delete rwwwrse-sa@$PROJECT_ID.iam.gserviceaccount.com

# Clean up SSL certificates
gcloud compute ssl-certificates delete rwwwrse-ssl-cert --global
```

## 📚 Additional Resources

- [Google Kubernetes Engine Documentation](https://cloud.google.com/kubernetes-engine/docs)
- [Workload Identity Guide](https://cloud.google.com/kubernetes-engine/docs/how-to/workload-identity)
- [GKE Best Practices](https://cloud.google.com/kubernetes-engine/docs/best-practices)
- [Cloud Monitoring for GKE](https://cloud.google.com/monitoring/kubernetes-engine)
- [VPA Documentation](https://cloud.google.com/kubernetes-engine/docs/concepts/verticalpodautoscaler)

## 🆘 Getting Help

For GKE-specific issues:

1. Check the [troubleshooting section](#-troubleshooting)
2. Review GCP Console logs and monitoring
3. Consult the [GKE documentation](https://cloud.google.com/kubernetes-engine/docs)
4. Check GCP status page for service issues
5. Review IAM permissions and Workload Identity setup

For rwwwrse-specific issues, refer to the [main documentation](../../../docs/DEPLOYMENT.md).

Remember to replace placeholder values (PROJECT_ID, domain names) with your actual GCP resources before deployment.