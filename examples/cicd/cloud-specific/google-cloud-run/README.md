# Google Cloud Run Deployment Guide for rwwwrse

Google Cloud Run is a fully managed serverless platform that automatically scales your containerized applications. This guide shows how to deploy rwwwrse on Cloud Run with production-ready configuration.

## Overview

Cloud Run provides:
- **Serverless scaling** from 0 to 1000+ instances
- **Pay-per-request pricing** with no idle costs
- **Built-in HTTPS** and custom domain support
- **Integrated monitoring** and logging
- **Traffic splitting** for blue-green deployments
- **No infrastructure management** required

## Prerequisites

- Google Cloud Platform account with billing enabled
- `gcloud` CLI installed and authenticated
- Docker image of rwwwrse in Google Container Registry or Artifact Registry
- Domain name for custom hostname (optional)

## Quick Start

### 1. Setup Google Cloud Environment

```bash
# Set your project ID
export PROJECT_ID="your-project-id"
export REGION="us-central1"

# Configure gcloud
gcloud config set project $PROJECT_ID
gcloud config set run/region $REGION

# Enable required APIs
gcloud services enable run.googleapis.com
gcloud services enable containerregistry.googleapis.com
gcloud services enable cloudbuild.googleapis.com
```

### 2. Build and Push Docker Image

```bash
# Build rwwwrse image
docker build -t gcr.io/$PROJECT_ID/rwwwrse:latest .

# Push to Google Container Registry
docker push gcr.io/$PROJECT_ID/rwwwrse:latest

# Alternative: Build with Cloud Build
gcloud builds submit --tag gcr.io/$PROJECT_ID/rwwwrse:latest
```

### 3. Deploy to Cloud Run

```bash
# Deploy with basic configuration
gcloud run deploy rwwwrse \
  --image gcr.io/$PROJECT_ID/rwwwrse:latest \
  --platform managed \
  --region $REGION \
  --allow-unauthenticated \
  --set-env-vars "RWWWRSE_PORT=8080,RWWWRSE_LOG_LEVEL=info"

# Get the service URL
gcloud run services describe rwwwrse --region $REGION --format 'value(status.url)'
```

## Advanced Configuration

### Production Deployment with YAML

Create [`service.yaml`](service.yaml) for production configuration:

```bash
# Deploy using configuration file
gcloud run services replace service.yaml --region $REGION
```

### Environment Variables Configuration

Set up environment variables for your backends:

```bash
gcloud run services update rwwwrse \
  --region $REGION \
  --set-env-vars \
    RWWWRSE_PORT=8080,\
    RWWWRSE_HOST=0.0.0.0,\
    RWWWRSE_LOG_LEVEL=info,\
    RWWWRSE_LOG_FORMAT=json,\
    RWWWRSE_HEALTH_PATH=/health,\
    RWWWRSE_METRICS_PATH=/metrics,\
    RWWWRSE_ENABLE_TLS=false,\
    RWWWRSE_ROUTES_API_TARGET=https://api-backend-service.com,\
    RWWWRSE_ROUTES_API_HOST=api.example.com,\
    RWWWRSE_ROUTES_APP_TARGET=https://app-backend-service.com,\
    RWWWRSE_ROUTES_APP_HOST=app.example.com
```

### Secrets Management

Use Google Secret Manager for sensitive configuration:

```bash
# Create secrets
gcloud secrets create rwwwrse-api-key --data-file=api-key.txt
gcloud secrets create rwwwrse-database-url --data-file=database-url.txt

# Grant Cloud Run access to secrets
gcloud secrets add-iam-policy-binding rwwwrse-api-key \
  --member=serviceAccount:$(gcloud projects describe $PROJECT_ID --format="value(projectNumber)")-compute@developer.gserviceaccount.com \
  --role=roles/secretmanager.secretAccessor

# Update service to use secrets
gcloud run services update rwwwrse \
  --region $REGION \
  --update-secrets API_KEY=rwwwrse-api-key:latest,DATABASE_URL=rwwwrse-database-url:latest
```

### Resource Configuration

Configure CPU, memory, and scaling:

```bash
gcloud run services update rwwwrse \
  --region $REGION \
  --cpu 1 \
  --memory 512Mi \
  --max-instances 100 \
  --min-instances 0 \
  --concurrency 80 \
  --timeout 300
```

### Health Checks and Startup Probes

```bash
# Configure health check
gcloud run services update rwwwrse \
  --region $REGION \
  --port 8080 \
  --http2

# The service.yaml includes proper health check configuration
```

## Custom Domain Setup

### 1. Domain Mapping

```bash
# Map custom domain
gcloud run domain-mappings create \
  --service rwwwrse \
  --domain api.example.com \
  --region $REGION

# Get DNS configuration
gcloud run domain-mappings describe api.example.com \
  --region $REGION \
  --format "export"
```

### 2. DNS Configuration

Configure your DNS provider with the CNAME record provided by the domain mapping command.

### 3. SSL Certificate

Cloud Run automatically provisions SSL certificates for custom domains.

## Multi-Service Architecture

### Backend Services on Cloud Run

```bash
# Deploy API service
gcloud run deploy api-service \
  --image gcr.io/$PROJECT_ID/api-service:latest \
  --region $REGION \
  --no-allow-unauthenticated \
  --set-env-vars "PORT=3001"

# Deploy App service
gcloud run deploy app-service \
  --image gcr.io/$PROJECT_ID/app-service:latest \
  --region $REGION \
  --no-allow-unauthenticated \
  --set-env-vars "PORT=3000"

# Get service URLs for rwwwrse configuration
API_URL=$(gcloud run services describe api-service --region $REGION --format 'value(status.url)')
APP_URL=$(gcloud run services describe app-service --region $REGION --format 'value(status.url)')

# Update rwwwrse with backend URLs
gcloud run services update rwwwrse \
  --region $REGION \
  --set-env-vars \
    RWWWRSE_ROUTES_API_TARGET=$API_URL,\
    RWWWRSE_ROUTES_APP_TARGET=$APP_URL
```

### Service-to-Service Authentication

```bash
# Create service account for rwwwrse
gcloud iam service-accounts create rwwwrse-sa \
  --display-name "rwwwrse Service Account"

# Grant Cloud Run Invoker role for backend services
gcloud run services add-iam-policy-binding api-service \
  --region $REGION \
  --member serviceAccount:rwwwrse-sa@$PROJECT_ID.iam.gserviceaccount.com \
  --role roles/run.invoker

# Update rwwwrse service to use the service account
gcloud run services update rwwwrse \
  --region $REGION \
  --service-account rwwwrse-sa@$PROJECT_ID.iam.gserviceaccount.com
```

## Traffic Management

### Blue-Green Deployment

```bash
# Deploy new version without traffic
gcloud run deploy rwwwrse \
  --image gcr.io/$PROJECT_ID/rwwwrse:v2 \
  --region $REGION \
  --no-traffic \
  --tag blue

# Split traffic between versions
gcloud run services update-traffic rwwwrse \
  --region $REGION \
  --to-revisions LATEST=50,blue=50

# Route all traffic to new version
gcloud run services update-traffic rwwwrse \
  --region $REGION \
  --to-latest
```

### Canary Deployment

```bash
# Deploy canary version
gcloud run deploy rwwwrse \
  --image gcr.io/$PROJECT_ID/rwwwrse:canary \
  --region $REGION \
  --no-traffic \
  --tag canary

# Route 10% traffic to canary
gcloud run services update-traffic rwwwrse \
  --region $REGION \
  --to-revisions LATEST=90,canary=10
```

## Monitoring and Logging

### Cloud Monitoring Integration

Cloud Run automatically integrates with Google Cloud's observability stack:

```bash
# View metrics
gcloud logging read "resource.type=cloud_run_revision" --limit 50

# Create monitoring dashboard
gcloud alpha monitoring dashboards create --config-from-file=dashboard.json
```

### Custom Metrics

rwwwrse exposes metrics at `/metrics` endpoint. Set up monitoring:

```bash
# Create uptime check
gcloud alpha monitoring uptime create \
  --display-name "rwwwrse Health Check" \
  --http-check-path /health \
  --http-check-port 443 \
  --http-check-use-ssl \
  --monitored-resource-type gce_instance \
  --monitored-resource-labels project_id=$PROJECT_ID

# Create alerting policy
gcloud alpha monitoring policies create --policy-from-file=alert-policy.yaml
```

### Structured Logging

Configure structured logging for better observability:

```bash
gcloud run services update rwwwrse \
  --region $REGION \
  --set-env-vars \
    RWWWRSE_LOG_FORMAT=json,\
    RWWWRSE_LOG_LEVEL=info
```

## Security Best Practices

### IAM and Service Accounts

```bash
# Create minimal IAM role
cat > rwwwrse-role.yaml << 'EOF'
title: "rwwwrse Custom Role"
description: "Minimal permissions for rwwwrse service"
stage: "GA"
includedPermissions:
- run.services.get
- secretmanager.versions.access
EOF

gcloud iam roles create rwwwrse.custom \
  --project $PROJECT_ID \
  --file rwwwrse-role.yaml

# Assign custom role to service account
gcloud projects add-iam-policy-binding $PROJECT_ID \
  --member serviceAccount:rwwwrse-sa@$PROJECT_ID.iam.gserviceaccount.com \
  --role projects/$PROJECT_ID/roles/rwwwrse.custom
```

### VPC Connector (Optional)

For private backend connectivity:

```bash
# Create VPC connector
gcloud compute networks vpc-access connectors create rwwwrse-connector \
  --region $REGION \
  --subnet your-subnet \
  --subnet-project $PROJECT_ID \
  --min-instances 2 \
  --max-instances 10

# Update service to use VPC connector
gcloud run services update rwwwrse \
  --region $REGION \
  --vpc-connector rwwwrse-connector \
  --vpc-egress private-ranges-only
```

### Binary Authorization

```bash
# Enable Binary Authorization
gcloud container binauthz policy import policy.yaml

# Update service to require attestation
gcloud run services update rwwwrse \
  --region $REGION \
  --binary-authorization default
```

## Cost Optimization

### Resource Right-Sizing

```bash
# Monitor resource usage
gcloud logging read "resource.type=cloud_run_revision AND severity>=WARNING" \
  --format="table(timestamp,severity,textPayload)" \
  --limit 50

# Adjust resources based on metrics
gcloud run services update rwwwrse \
  --region $REGION \
  --cpu 0.5 \
  --memory 256Mi \
  --max-instances 50
```

### Minimum Instances

For consistent performance (increases cost):

```bash
gcloud run services update rwwwrse \
  --region $REGION \
  --min-instances 1
```

## Automation Scripts

### Deployment Script

Create [`deploy.sh`](scripts/deploy.sh):

```bash
#!/bin/bash
set -e

# Source configuration
source config.env

# Build and deploy
gcloud builds submit --tag gcr.io/$PROJECT_ID/rwwwrse:$VERSION
gcloud run services replace service.yaml --region $REGION

echo "Deployment completed successfully"
echo "Service URL: $(gcloud run services describe rwwwrse --region $REGION --format 'value(status.url)')"
```

### Health Check Script

Create [`health-check.sh`](scripts/health-check.sh):

```bash
#!/bin/bash

SERVICE_URL=$(gcloud run services describe rwwwrse --region $REGION --format 'value(status.url)')
HEALTH_URL="$SERVICE_URL/health"

if curl -f -s "$HEALTH_URL" > /dev/null; then
  echo "Service is healthy"
  exit 0
else
  echo "Health check failed"
  exit 1
fi
```

## Troubleshooting

### Common Issues

1. **Cold Start Latency**
   ```bash
   # Set minimum instances
   gcloud run services update rwwwrse --min-instances 1
   
   # Optimize container image size
   # Use multi-stage Docker builds
   # Implement health check caching
   ```

2. **Memory Issues**
   ```bash
   # Check memory usage
   gcloud logging read "resource.type=cloud_run_revision AND textPayload:'memory'" --limit 10
   
   # Increase memory allocation
   gcloud run services update rwwwrse --memory 1Gi
   ```

3. **Timeout Issues**
   ```bash
   # Increase request timeout
   gcloud run services update rwwwrse --timeout 900
   
   # Check for long-running requests
   gcloud logging read "resource.type=cloud_run_revision AND httpRequest.latency>10s"
   ```

4. **Authentication Errors**
   ```bash
   # Check service account permissions
   gcloud iam service-accounts get-iam-policy rwwwrse-sa@$PROJECT_ID.iam.gserviceaccount.com
   
   # Verify invoker permissions
   gcloud run services get-iam-policy rwwwrse --region $REGION
   ```

### Debugging Commands

```bash
# View service configuration
gcloud run services describe rwwwrse --region $REGION

# Check recent logs
gcloud logging read "resource.type=cloud_run_revision" --limit 50 --format json

# List all revisions
gcloud run revisions list --service rwwwrse --region $REGION

# Get revision details
gcloud run revisions describe REVISION_NAME --region $REGION
```

## Performance Optimization

### Container Optimization

```dockerfile
# Multi-stage build for smaller images
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o rwwwrse ./cmd/rwwwrse

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/rwwwrse .
EXPOSE 8080
CMD ["./rwwwrse"]
```

### Configuration Tuning

```yaml
# service.yaml optimizations
spec:
  template:
    metadata:
      annotations:
        autoscaling.knative.dev/maxScale: "100"
        autoscaling.knative.dev/minScale: "0"
        run.googleapis.com/cpu-throttling: "false"
        run.googleapis.com/execution-environment: gen2
    spec:
      containerConcurrency: 100
      timeoutSeconds: 300
      containers:
      - image: gcr.io/PROJECT_ID/rwwwrse:latest
        resources:
          limits:
            cpu: 1000m
            memory: 512Mi
        startupProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 10
          timeoutSeconds: 5
          periodSeconds: 10
          failureThreshold: 3
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          periodSeconds: 30
          timeoutSeconds: 5
```

## Backup and Disaster Recovery

### Configuration Backup

```bash
# Export service configuration
gcloud run services describe rwwwrse --region $REGION --format export > rwwwrse-backup.yaml

# Export IAM policies
gcloud run services get-iam-policy rwwwrse --region $REGION --format json > rwwwrse-iam-backup.json
```

### Multi-Region Deployment

```bash
# Deploy to multiple regions
REGIONS=("us-central1" "us-east1" "europe-west1")

for region in "${REGIONS[@]}"; do
  gcloud run services replace service.yaml --region $region
done

# Set up global load balancer
gcloud compute url-maps create rwwwrse-lb \
  --default-backend-bucket=rwwwrse-bucket

gcloud compute backend-services create rwwwrse-backend \
  --global \
  --load-balancing-scheme=EXTERNAL \
  --protocol=HTTPS
```

## Migration from Other Platforms

### From Kubernetes

1. **Extract configuration:**
   ```bash
   kubectl get deployment rwwwrse -o yaml > k8s-deployment.yaml
   kubectl get service rwwwrse -o yaml > k8s-service.yaml
   kubectl get configmap rwwwrse-config -o yaml > k8s-configmap.yaml
   ```

2. **Convert to Cloud Run:**
   ```bash
   # Extract environment variables from ConfigMap
   # Create service.yaml with equivalent configuration
   # Deploy using gcloud run services replace
   ```

### From Docker Compose

```bash
# Extract environment variables from docker-compose.yml
grep -E "RWWWRSE_" docker-compose.yml > cloud-run.env

# Convert to gcloud command
gcloud run services update rwwwrse --env-vars-file cloud-run.env
```

## CI/CD Integration

### Cloud Build Integration

Create [`cloudbuild.yaml`](cloudbuild.yaml):

```yaml
steps:
- name: 'gcr.io/cloud-builders/docker'
  args: ['build', '-t', 'gcr.io/$PROJECT_ID/rwwwrse:$COMMIT_SHA', '.']
- name: 'gcr.io/cloud-builders/docker'
  args: ['push', 'gcr.io/$PROJECT_ID/rwwwrse:$COMMIT_SHA']
- name: 'gcr.io/cloud-builders/gcloud'
  args: ['run', 'services', 'update', 'rwwwrse', '--image', 'gcr.io/$PROJECT_ID/rwwwrse:$COMMIT_SHA', '--region', 'us-central1']
```

### GitHub Actions Integration

See [`../../cicd/github-actions/`](../../cicd/github-actions/) for complete CI/CD examples.

## Next Steps

1. **Set up monitoring and alerting** using Google Cloud Monitoring
2. **Implement CI/CD pipeline** for automated deployments
3. **Configure multi-region deployment** for high availability
4. **Optimize costs** by monitoring usage patterns and adjusting resources
5. **Set up backup and disaster recovery** procedures

## Related Documentation

- [Google Cloud Run Documentation](https://cloud.google.com/run/docs)
- [Cloud Build Documentation](https://cloud.google.com/build/docs)
- [Secret Manager Documentation](https://cloud.google.com/secret-manager/docs)
- [Cloud Monitoring Documentation](https://cloud.google.com/monitoring/docs)
- [Docker Examples](../../docker-compose/) - Local development
- [CI/CD Examples](../../cicd/) - Automated deployment pipelines