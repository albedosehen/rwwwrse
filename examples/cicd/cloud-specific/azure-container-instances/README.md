# Azure Container Instances Deployment Guide for rwwwrse

Azure Container Instances (ACI) provides the fastest and simplest way to run containers in Azure without managing virtual machines. This guide shows how to deploy rwwwrse using ACI with production-ready configuration.

## Overview

Azure Container Instances provides:
- **Serverless containers** with pay-per-second billing
- **Fast startup** with containers starting in seconds
- **Built-in networking** with public IP and FQDN
- **Persistent storage** with Azure Files integration
- **Container groups** for multi-container applications
- **Azure ecosystem integration** (Key Vault, Monitor, etc.)

## Prerequisites

- Azure CLI installed and authenticated
- Container image of rwwwrse in Azure Container Registry (ACR)
- Azure subscription with appropriate permissions
- Resource group for deployment

## Quick Start

### 1. Setup Azure Environment

```bash
# Set environment variables
export RESOURCE_GROUP="rwwwrse-rg"
export LOCATION="eastus"
export ACR_NAME="rwwwrseacr"
export CONTAINER_GROUP_NAME="rwwwrse-cg"

# Create resource group
az group create \
  --name $RESOURCE_GROUP \
  --location $LOCATION

# Create Azure Container Registry
az acr create \
  --resource-group $RESOURCE_GROUP \
  --name $ACR_NAME \
  --sku Basic \
  --admin-enabled true
```

### 2. Push Image to ACR

```bash
# Build and tag image
docker build -t rwwwrse .
docker tag rwwwrse $ACR_NAME.azurecr.io/rwwwrse:latest

# Login to ACR
az acr login --name $ACR_NAME

# Push image
docker push $ACR_NAME.azurecr.io/rwwwrse:latest
```

### 3. Deploy Container Instance

```bash
# Get ACR credentials
ACR_PASSWORD=$(az acr credential show --name $ACR_NAME --query "passwords[0].value" --output tsv)

# Create container instance
az container create \
  --resource-group $RESOURCE_GROUP \
  --name rwwwrse \
  --image $ACR_NAME.azurecr.io/rwwwrse:latest \
  --registry-login-server $ACR_NAME.azurecr.io \
  --registry-username $ACR_NAME \
  --registry-password $ACR_PASSWORD \
  --dns-name-label rwwwrse-demo \
  --ports 8080 \
  --cpu 1 \
  --memory 1.5 \
  --environment-variables \
    RWWWRSE_PORT=8080 \
    RWWWRSE_LOG_LEVEL=info \
    RWWWRSE_LOG_FORMAT=json
```

## Advanced Configuration

### Container Group with YAML

Create [`container-group.yaml`](container-group.yaml) for advanced configuration:

```bash
# Deploy using YAML configuration
az container create \
  --resource-group $RESOURCE_GROUP \
  --file container-group.yaml
```

### Environment Variables and Secrets

```bash
# Create container with environment variables
az container create \
  --resource-group $RESOURCE_GROUP \
  --name rwwwrse \
  --image $ACR_NAME.azurecr.io/rwwwrse:latest \
  --registry-login-server $ACR_NAME.azurecr.io \
  --registry-username $ACR_NAME \
  --registry-password $ACR_PASSWORD \
  --environment-variables \
    RWWWRSE_PORT=8080 \
    RWWWRSE_HOST=0.0.0.0 \
    RWWWRSE_LOG_LEVEL=info \
    RWWWRSE_LOG_FORMAT=json \
    RWWWRSE_HEALTH_PATH=/health \
    RWWWRSE_METRICS_PATH=/metrics \
    RWWWRSE_ROUTES_API_TARGET=https://api-backend.com \
    RWWWRSE_ROUTES_APP_TARGET=https://app-backend.com \
  --secure-environment-variables \
    DATABASE_URL=postgresql://user:pass@host:5432/db \
    API_KEY=your-secret-api-key
```

## Multi-Container Deployment

### Container Group with Multiple Services

```bash
# Deploy container group with multiple containers
az container create \
  --resource-group $RESOURCE_GROUP \
  --name rwwwrse-multi \
  --file multi-container-group.yaml
```

The [`multi-container-group.yaml`](multi-container-group.yaml) includes:
- rwwwrse proxy container
- Backend API container
- Shared network and storage
- Service discovery configuration

## Load Balancer Integration

### Azure Application Gateway

```bash
# Create virtual network
az network vnet create \
  --resource-group $RESOURCE_GROUP \
  --name rwwwrse-vnet \
  --address-prefix 10.0.0.0/16 \
  --subnet-name default \
  --subnet-prefix 10.0.0.0/24

# Create Application Gateway subnet
az network vnet subnet create \
  --resource-group $RESOURCE_GROUP \
  --vnet-name rwwwrse-vnet \
  --name appgateway-subnet \
  --address-prefix 10.0.1.0/24

# Create public IP for Application Gateway
az network public-ip create \
  --resource-group $RESOURCE_GROUP \
  --name rwwwrse-appgw-pip \
  --allocation-method Static \
  --sku Standard

# Create Application Gateway
az network application-gateway create \
  --resource-group $RESOURCE_GROUP \
  --name rwwwrse-appgw \
  --location $LOCATION \
  --capacity 2 \
  --sku Standard_v2 \
  --vnet-name rwwwrse-vnet \
  --subnet appgateway-subnet \
  --public-ip-address rwwwrse-appgw-pip \
  --http-settings-cookie-based-affinity Disabled \
  --frontend-port 80 \
  --http-settings-port 8080 \
  --http-settings-protocol Http
```

### Azure Front Door

```bash
# Create Front Door profile
az afd profile create \
  --resource-group $RESOURCE_GROUP \
  --profile-name rwwwrse-frontdoor \
  --sku Premium_AzureFrontDoor

# Create endpoint
az afd endpoint create \
  --resource-group $RESOURCE_GROUP \
  --profile-name rwwwrse-frontdoor \
  --endpoint-name rwwwrse \
  --enabled-state Enabled

# Create origin group
az afd origin-group create \
  --resource-group $RESOURCE_GROUP \
  --profile-name rwwwrse-frontdoor \
  --origin-group-name rwwwrse-origins \
  --probe-request-type GET \
  --probe-protocol Http \
  --probe-interval-in-seconds 120 \
  --probe-path /health

# Add origin
az afd origin create \
  --resource-group $RESOURCE_GROUP \
  --profile-name rwwwrse-frontdoor \
  --origin-group-name rwwwrse-origins \
  --origin-name rwwwrse-origin \
  --origin-host-header rwwwrse-demo.eastus.azurecontainer.io \
  --hostname rwwwrse-demo.eastus.azurecontainer.io \
  --http-port 8080 \
  --priority 1 \
  --weight 1000 \
  --enabled-state Enabled
```

## Storage Integration

### Azure Files Mount

```bash
# Create storage account
az storage account create \
  --resource-group $RESOURCE_GROUP \
  --name rwwwrsestorage \
  --location $LOCATION \
  --sku Standard_LRS

# Create file share
az storage share create \
  --name rwwwrse-data \
  --account-name rwwwrsestorage

# Get storage key
STORAGE_KEY=$(az storage account keys list \
  --resource-group $RESOURCE_GROUP \
  --account-name rwwwrsestorage \
  --query "[0].value" --output tsv)

# Deploy container with mounted storage
az container create \
  --resource-group $RESOURCE_GROUP \
  --name rwwwrse-with-storage \
  --image $ACR_NAME.azurecr.io/rwwwrse:latest \
  --azure-file-volume-account-name rwwwrsestorage \
  --azure-file-volume-account-key $STORAGE_KEY \
  --azure-file-volume-share-name rwwwrse-data \
  --azure-file-volume-mount-path /data
```

## Security Configuration

### Azure Key Vault Integration

```bash
# Create Key Vault
az keyvault create \
  --resource-group $RESOURCE_GROUP \
  --name rwwwrse-kv \
  --location $LOCATION \
  --enable-soft-delete true \
  --enable-purge-protection true

# Store secrets
az keyvault secret set \
  --vault-name rwwwrse-kv \
  --name database-url \
  --value "postgresql://user:pass@host:5432/db"

az keyvault secret set \
  --vault-name rwwwrse-kv \
  --name api-key \
  --value "your-secret-api-key"

# Create managed identity
az identity create \
  --resource-group $RESOURCE_GROUP \
  --name rwwwrse-identity

# Get identity details
IDENTITY_ID=$(az identity show \
  --resource-group $RESOURCE_GROUP \
  --name rwwwrse-identity \
  --query clientId --output tsv)

IDENTITY_RESOURCE_ID=$(az identity show \
  --resource-group $RESOURCE_GROUP \
  --name rwwwrse-identity \
  --query id --output tsv)

# Grant Key Vault access
az keyvault set-policy \
  --name rwwwrse-kv \
  --object-id $(az identity show --resource-group $RESOURCE_GROUP --name rwwwrse-identity --query principalId --output tsv) \
  --secret-permissions get list

# Deploy container with managed identity
az container create \
  --resource-group $RESOURCE_GROUP \
  --name rwwwrse-secure \
  --image $ACR_NAME.azurecr.io/rwwwrse:latest \
  --assign-identity $IDENTITY_RESOURCE_ID \
  --environment-variables \
    AZURE_CLIENT_ID=$IDENTITY_ID \
    KEY_VAULT_NAME=rwwwrse-kv
```

### Network Security

```bash
# Create network profile for subnet deployment
az network vnet subnet create \
  --resource-group $RESOURCE_GROUP \
  --vnet-name rwwwrse-vnet \
  --name container-subnet \
  --address-prefix 10.0.2.0/24

# Delegate subnet to container instances
az network vnet subnet update \
  --resource-group $RESOURCE_GROUP \
  --vnet-name rwwwrse-vnet \
  --name container-subnet \
  --delegations Microsoft.ContainerInstance/containerGroups

# Create network security group
az network nsg create \
  --resource-group $RESOURCE_GROUP \
  --name rwwwrse-nsg

# Add security rules
az network nsg rule create \
  --resource-group $RESOURCE_GROUP \
  --nsg-name rwwwrse-nsg \
  --name allow-http \
  --protocol tcp \
  --priority 1000 \
  --destination-port-range 8080 \
  --access allow

# Associate NSG with subnet
az network vnet subnet update \
  --resource-group $RESOURCE_GROUP \
  --vnet-name rwwwrse-vnet \
  --name container-subnet \
  --network-security-group rwwwrse-nsg

# Deploy container in subnet
az container create \
  --resource-group $RESOURCE_GROUP \
  --name rwwwrse-private \
  --image $ACR_NAME.azurecr.io/rwwwrse:latest \
  --vnet rwwwrse-vnet \
  --subnet container-subnet
```

## Monitoring and Logging

### Azure Monitor Integration

```bash
# Create Log Analytics workspace
az monitor log-analytics workspace create \
  --resource-group $RESOURCE_GROUP \
  --workspace-name rwwwrse-logs \
  --location $LOCATION

# Get workspace ID and key
WORKSPACE_ID=$(az monitor log-analytics workspace show \
  --resource-group $RESOURCE_GROUP \
  --workspace-name rwwwrse-logs \
  --query customerId --output tsv)

WORKSPACE_KEY=$(az monitor log-analytics workspace get-shared-keys \
  --resource-group $RESOURCE_GROUP \
  --workspace-name rwwwrse-logs \
  --query primarySharedKey --output tsv)

# Deploy container with Log Analytics
az container create \
  --resource-group $RESOURCE_GROUP \
  --name rwwwrse-monitored \
  --image $ACR_NAME.azurecr.io/rwwwrse:latest \
  --log-analytics-workspace $WORKSPACE_ID \
  --log-analytics-workspace-key $WORKSPACE_KEY
```

### Application Insights

```bash
# Create Application Insights
az monitor app-insights component create \
  --resource-group $RESOURCE_GROUP \
  --app rwwwrse-insights \
  --location $LOCATION \
  --kind web

# Get instrumentation key
INSTRUMENTATION_KEY=$(az monitor app-insights component show \
  --resource-group $RESOURCE_GROUP \
  --app rwwwrse-insights \
  --query instrumentationKey --output tsv)

# Deploy container with Application Insights
az container create \
  --resource-group $RESOURCE_GROUP \
  --name rwwwrse-insights \
  --image $ACR_NAME.azurecr.io/rwwwrse:latest \
  --environment-variables \
    APPINSIGHTS_INSTRUMENTATIONKEY=$INSTRUMENTATION_KEY
```

## Scaling and High Availability

### Container Group Scaling

Since ACI doesn't support native auto-scaling, use Azure Logic Apps or Functions:

```bash
# Create scale-out Logic App
az resource create \
  --resource-group $RESOURCE_GROUP \
  --resource-type Microsoft.Logic/workflows \
  --name rwwwrse-scaler \
  --properties @scaler-logic-app.json
```

### Multi-Region Deployment

```bash
# Deploy to multiple regions
REGIONS=("eastus" "westus2" "northeurope")

for region in "${REGIONS[@]}"; do
  az container create \
    --resource-group $RESOURCE_GROUP \
    --name "rwwwrse-$region" \
    --image $ACR_NAME.azurecr.io/rwwwrse:latest \
    --location $region \
    --dns-name-label "rwwwrse-$region"
done

# Configure Traffic Manager for load balancing
az network traffic-manager profile create \
  --resource-group $RESOURCE_GROUP \
  --name rwwwrse-tm \
  --routing-method Performance \
  --unique-dns-name rwwwrse-global
```

## Cost Optimization

### Resource Right-Sizing

```bash
# Monitor resource usage
az monitor metrics list \
  --resource "/subscriptions/{subscription-id}/resourceGroups/$RESOURCE_GROUP/providers/Microsoft.ContainerInstance/containerGroups/rwwwrse" \
  --metric "CpuUsage,MemoryUsage" \
  --start-time 2023-01-01T00:00:00Z \
  --end-time 2023-01-02T00:00:00Z

# Update container with optimized resources
az container create \
  --resource-group $RESOURCE_GROUP \
  --name rwwwrse-optimized \
  --image $ACR_NAME.azurecr.io/rwwwrse:latest \
  --cpu 0.5 \
  --memory 1
```

### Scheduled Scaling

```bash
# Create automation account for scheduled start/stop
az automation account create \
  --resource-group $RESOURCE_GROUP \
  --name rwwwrse-automation \
  --location $LOCATION

# Create runbook for container management
az automation runbook create \
  --resource-group $RESOURCE_GROUP \
  --automation-account-name rwwwrse-automation \
  --name start-stop-containers \
  --type PowerShell \
  --description "Start and stop containers based on schedule"
```

## Backup and Disaster Recovery

### Container Image Backup

```bash
# Enable geo-replication for ACR
az acr replication create \
  --registry $ACR_NAME \
  --location westus2

# Export container configuration
az container export \
  --resource-group $RESOURCE_GROUP \
  --name rwwwrse \
  --file rwwwrse-backup.yaml
```

### Data Backup

```bash
# Backup Azure Files data
az storage file copy start-batch \
  --account-name rwwwrsestorage \
  --destination-share backup-share \
  --source-share rwwwrse-data
```

## Troubleshooting

### Common Issues

1. **Container startup failures:**
```bash
# Check container logs
az container logs \
  --resource-group $RESOURCE_GROUP \
  --name rwwwrse

# Check container events
az container show \
  --resource-group $RESOURCE_GROUP \
  --name rwwwrse \
  --query instanceView.events
```

2. **Networking issues:**
```bash
# Test connectivity
az container exec \
  --resource-group $RESOURCE_GROUP \
  --name rwwwrse \
  --exec-command "/bin/sh"

# Check DNS resolution
nslookup backend-service.com
```

3. **Resource constraints:**
```bash
# Monitor resource usage
az monitor metrics list \
  --resource "/subscriptions/{subscription-id}/resourceGroups/$RESOURCE_GROUP/providers/Microsoft.ContainerInstance/containerGroups/rwwwrse" \
  --metric "CpuUsage,MemoryUsage"
```

### Debugging Commands

```bash
# Get detailed container information
az container show \
  --resource-group $RESOURCE_GROUP \
  --name rwwwrse \
  --output table

# Restart container group
az container restart \
  --resource-group $RESOURCE_GROUP \
  --name rwwwrse

# Delete and recreate container
az container delete \
  --resource-group $RESOURCE_GROUP \
  --name rwwwrse \
  --yes

# Recreate with latest configuration
az container create --file container-group.yaml
```

## Automation Scripts

### Deployment Script

```bash
#!/bin/bash
set -e

# Load configuration
source config.env

echo "Building and pushing image..."
docker build -t $ACR_NAME.azurecr.io/rwwwrse:$VERSION .
az acr login --name $ACR_NAME
docker push $ACR_NAME.azurecr.io/rwwwrse:$VERSION

echo "Updating container configuration..."
sed "s/IMAGE_TAG/$VERSION/g" container-group-template.yaml > container-group.yaml

echo "Deploying container instance..."
az container delete --resource-group $RESOURCE_GROUP --name rwwwrse --yes || true
az container create --resource-group $RESOURCE_GROUP --file container-group.yaml

echo "Deployment completed successfully"
```

### Health Check Script

```bash
#!/bin/bash

FQDN=$(az container show \
  --resource-group $RESOURCE_GROUP \
  --name rwwwrse \
  --query ipAddress.fqdn --output tsv)

HEALTH_URL="http://$FQDN:8080/health"

if curl -f -s "$HEALTH_URL" > /dev/null; then
  echo "Service is healthy"
  exit 0
else
  echo "Health check failed"
  exit 1
fi
```

## Migration Strategies

### From VM to ACI

```bash
# Export VM configuration
az vm show --resource-group vm-rg --name vm-name > vm-config.json

# Create equivalent container configuration
# Convert VM environment to container environment variables
# Deploy using ACI
```

### From App Service to ACI

```bash
# Export App Service configuration
az webapp config appsettings list \
  --resource-group app-rg \
  --name app-name > app-settings.json

# Convert to container environment variables
jq -r '.[] | "\(.name)=\(.value)"' app-settings.json > container.env

# Deploy container with settings
az container create --environment-variables @container.env
```

## CI/CD Integration

### Azure DevOps Pipeline

Create [`azure-pipelines.yml`](azure-pipelines.yml):

```yaml
trigger:
- main

variables:
  containerRegistry: 'rwwwrseacr.azurecr.io'
  imageRepository: 'rwwwrse'
  dockerfilePath: '$(Build.SourcesDirectory)/Dockerfile'
  tag: '$(Build.BuildId)'

stages:
- stage: Build
  jobs:
  - job: Build
    steps:
    - task: Docker@2
      inputs:
        containerRegistry: 'ACR Connection'
        repository: $(imageRepository)
        command: 'buildAndPush'
        Dockerfile: $(dockerfilePath)
        tags: $(tag)

- stage: Deploy
  jobs:
  - job: Deploy
    steps:
    - task: AzureCLI@2
      inputs:
        azureSubscription: 'Azure Subscription'
        scriptType: 'bash'
        scriptLocation: 'inlineScript'
        inlineScript: |
          az container create \
            --resource-group $(resourceGroup) \
            --name rwwwrse \
            --image $(containerRegistry)/$(imageRepository):$(tag)
```

## Next Steps

1. **Set up CI/CD pipeline** with Azure DevOps or GitHub Actions
2. **Implement monitoring and alerting** with Azure Monitor
3. **Configure backup and disaster recovery** procedures
4. **Optimize costs** with scheduled scaling and resource right-sizing
5. **Enhance security** with Azure Key Vault and network isolation

## Related Documentation

- [Azure Container Instances Documentation](https://docs.microsoft.com/en-us/azure/container-instances/)
- [Azure Container Registry Documentation](https://docs.microsoft.com/en-us/azure/container-registry/)
- [Azure Monitor Documentation](https://docs.microsoft.com/en-us/azure/azure-monitor/)
- [CI/CD Examples](../../cicd/) - Automated deployment pipelines
- [Docker Examples](../../docker-compose/) - Local development and testing