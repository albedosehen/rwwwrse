# rwwwrse on Azure Kubernetes Service (AKS)

This example demonstrates deploying rwwwrse on Azure Kubernetes Service with Azure-specific integrations and best practices.

## Architecture Overview

```bash
┌─────────────────────────────────────────────────────────────┐
│                       Microsoft Azure                      │
│  ┌─────────────────────────────────────────────────────┐   │
│  │                  AKS Cluster                       │   │
│  │                                                     │   │
│  │  ┌─────────────┐    ┌─────────────────────────┐    │   │
│  │  │   Azure     │    │      rwwwrse Pods      │    │   │
│  │  │Load Balancer│────│   (Auto-scaled 3-100)  │    │   │
│  │  │   (ALB)     │    │                         │    │   │
│  │  └─────────────┘    └─────────────────────────┘    │   │
│  │                                                     │   │
│  │  ┌─────────────┐    ┌─────────────────────────┐    │   │
│  │  │   Azure     │    │    Sample Applications  │    │   │
│  │  │ Container   │    │   (Auto-scaled 2-50)   │    │   │
│  │  │ Registry    │    │                         │    │   │
│  │  │   (ACR)     │    └─────────────────────────┘    │   │
│  │  └─────────────┘                                    │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐   │
│  │                Azure Integrations                   │   │
│  │  • Workload Identity                               │   │
│  │  • Azure Monitor & Application Insights            │   │
│  │  • Key Vault CSI Driver                            │   │
│  │  • Azure DNS                                       │   │
│  │  • Application Gateway                             │   │
│  │  • Virtual Network Integration                     │   │
│  └─────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
```

## Azure-Specific Features

- **Azure Load Balancer**: Standard SKU with health probes and static IP
- **Workload Identity**: Secure pod-to-Azure service authentication
- **Azure Container Registry (ACR)**: Private container image storage
- **Azure Monitor & Application Insights**: Comprehensive observability
- **Azure Key Vault CSI Driver**: Secure secret management
- **Azure DNS**: Managed DNS with automatic record creation
- **Application Gateway**: Advanced load balancing with WAF
- **Virtual Network Integration**: Secure networking with Network Security Groups

## Prerequisites

### Azure Setup

```bash
# Required tools
az version  # >= 2.40.0
kubectl version --client  # >= 1.20.0
docker --version  # >= 20.10.0
```

### Azure CLI and Extensions

```bash
# Install required extensions
az extension add --name aks-preview
az extension add --name application-insights
az extension add --name keyvault-preview

# Register required providers
az provider register --namespace Microsoft.ContainerService
az provider register --namespace Microsoft.Insights
az provider register --namespace Microsoft.KeyVault
az provider register --namespace Microsoft.Network
```

### Required Azure Roles

The deploying user needs:

- `Contributor` or `Owner` on the subscription
- `Azure Kubernetes Service Cluster Admin Role`
- `Key Vault Administrator`
- `Monitoring Contributor`

## Quick Start

### 1. Set Environment Variables

```bash
# Set deployment variables
export SUBSCRIPTION_ID="your-subscription-id"
export RESOURCE_GROUP="rwwwrse-rg"
export CLUSTER_NAME="rwwwrse-aks"
export LOCATION="eastus"
export ACR_NAME="rwwwrseacr"
export KEYVAULT_NAME="rwwwrse-kv"

# Set Azure subscription
az account set --subscription $SUBSCRIPTION_ID
```

### 2. Create Resource Group and ACR

```bash
# Create resource group
az group create \
    --name $RESOURCE_GROUP \
    --location $LOCATION

# Create Azure Container Registry
az acr create \
    --resource-group $RESOURCE_GROUP \
    --name $ACR_NAME \
    --sku Standard \
    --admin-enabled false

# Get ACR login server
ACR_LOGIN_SERVER=$(az acr show --name $ACR_NAME --resource-group $RESOURCE_GROUP --query "loginServer" --output tsv)
```

### 3. Create AKS Cluster with Workload Identity

```bash
# Create AKS cluster
az aks create \
    --resource-group $RESOURCE_GROUP \
    --name $CLUSTER_NAME \
    --location $LOCATION \
    --node-count 3 \
    --min-count 1 \
    --max-count 10 \
    --enable-cluster-autoscaler \
    --node-vm-size Standard_D2s_v3 \
    --enable-workload-identity \
    --enable-oidc-issuer \
    --enable-managed-identity \
    --attach-acr $ACR_NAME \
    --enable-addons monitoring \
    --generate-ssh-keys \
    --network-plugin azure \
    --network-policy azure \
    --enable-node-public-ip false \
    --enable-blob-driver \
    --enable-file-driver \
    --kubernetes-version "1.28"

# Get cluster credentials
az aks get-credentials \
    --resource-group $RESOURCE_GROUP \
    --name $CLUSTER_NAME \
    --overwrite-existing
```

### 4. Set Up Azure Key Vault

```bash
# Create Key Vault
az keyvault create \
    --name $KEYVAULT_NAME \
    --resource-group $RESOURCE_GROUP \
    --location $LOCATION \
    --enable-rbac-authorization

# Get current user object ID
USER_OBJECT_ID=$(az ad signed-in-user show --query id --output tsv)

# Assign Key Vault Administrator role
az role assignment create \
    --role "Key Vault Administrator" \
    --assignee $USER_OBJECT_ID \
    --scope "/subscriptions/$SUBSCRIPTION_ID/resourceGroups/$RESOURCE_GROUP/providers/Microsoft.KeyVault/vaults/$KEYVAULT_NAME"

# Create secrets
az keyvault secret set \
    --vault-name $KEYVAULT_NAME \
    --name "rwwwrse-tls-email" \
    --value "admin@your-domain.com"

az keyvault secret set \
    --vault-name $KEYVAULT_NAME \
    --name "rwwwrse-jwt-secret" \
    --value "your-jwt-secret-key"
```

### 5. Install Key Vault CSI Driver

```bash
# Install Key Vault CSI driver
az aks enable-addons \
    --addons azure-keyvault-secrets-provider \
    --name $CLUSTER_NAME \
    --resource-group $RESOURCE_GROUP

# Verify installation
kubectl get pods -n kube-system -l app=secrets-store-csi-driver
kubectl get pods -n kube-system -l app=secrets-store-provider-azure
```

### 6. Set Up Workload Identity

```bash
# Get OIDC issuer URL
OIDC_ISSUER=$(az aks show --name $CLUSTER_NAME --resource-group $RESOURCE_GROUP --query "oidcIssuerProfile.issuerUrl" --output tsv)

# Create managed identity
az identity create \
    --name "rwwwrse-identity" \
    --resource-group $RESOURCE_GROUP \
    --location $LOCATION

# Get identity details
IDENTITY_CLIENT_ID=$(az identity show --name "rwwwrse-identity" --resource-group $RESOURCE_GROUP --query "clientId" --output tsv)
IDENTITY_PRINCIPAL_ID=$(az identity show --name "rwwwrse-identity" --resource-group $RESOURCE_GROUP --query "principalId" --output tsv)

# Assign Key Vault Secrets User role
az role assignment create \
    --role "Key Vault Secrets User" \
    --assignee $IDENTITY_PRINCIPAL_ID \
    --scope "/subscriptions/$SUBSCRIPTION_ID/resourceGroups/$RESOURCE_GROUP/providers/Microsoft.KeyVault/vaults/$KEYVAULT_NAME"

# Create federated identity credential
az identity federated-credential create \
    --name "rwwwrse-fedcred" \
    --identity-name "rwwwrse-identity" \
    --resource-group $RESOURCE_GROUP \
    --issuer $OIDC_ISSUER \
    --subject "system:serviceaccount:rwwwrse:rwwwrse-sa" \
    --audience "api://AzureADTokenExchange"
```

### 7. Build and Push Container Image

```bash
# Login to ACR
az acr login --name $ACR_NAME

# Build and push image
docker build -t $ACR_LOGIN_SERVER/rwwwrse:latest .
docker push $ACR_LOGIN_SERVER/rwwwrse:latest
```

### 8. Deploy rwwwrse

```bash
# Navigate to AKS example directory
cd examples/kubernetes/cloud/azure-aks

# Update configuration files with your values
# - Replace subscription-id, resource-group, and other placeholders
# - Update ACR references and domain names

# Apply manifests
kubectl apply -f namespace.yaml
kubectl apply -f configmap.yaml

# Update deployment with actual values
sed -i "s/your-acr.azurecr.io/$ACR_LOGIN_SERVER/g" deployment.yaml
sed -i "s/your-subscription-id/$SUBSCRIPTION_ID/g" deployment.yaml
sed -i "s/your-resource-group/$RESOURCE_GROUP/g" deployment.yaml

kubectl apply -f deployment.yaml

# Verify deployment
kubectl get pods -n rwwwrse
kubectl get services -n rwwwrse
```

## Configuration

### Azure Container Registry Integration

Update the image reference in [`deployment.yaml`](deployment.yaml):

```yaml
image: your-acr.azurecr.io/rwwwrse:latest
```

### Workload Identity Configuration

Service account annotation in deployment:

```yaml
serviceAccountName: rwwwrse-sa
annotations:
  azure.workload.identity/client-id: YOUR_IDENTITY_CLIENT_ID
```

### Auto Scaling Configuration

#### Horizontal Pod Autoscaler (HPA)

- **Min replicas**: 3
- **Max replicas**: 100
- **CPU target**: 70%
- **Memory target**: 80%

#### Cluster Autoscaler

Automatically scales nodes based on pod resource requirements:

```bash
# View autoscaler status
kubectl describe configmap cluster-autoscaler-status -n kube-system
```

## 📊 Monitoring and Observability

### Azure Monitor Integration

Application metrics are automatically collected by Azure Monitor:

```bash
# View cluster metrics
az monitor metrics list \
    --resource "/subscriptions/$SUBSCRIPTION_ID/resourceGroups/$RESOURCE_GROUP/providers/Microsoft.ContainerService/managedClusters/$CLUSTER_NAME" \
    --metric "node_cpu_usage_percentage"

# Create action group for alerts
az monitor action-group create \
    --resource-group $RESOURCE_GROUP \
    --name "rwwwrse-alerts" \
    --short-name "rwwwrse"
```

### Application Insights

Detailed application telemetry:

```bash
# Create Application Insights
az monitor app-insights component create \
    --app "rwwwrse-insights" \
    --location $LOCATION \
    --resource-group $RESOURCE_GROUP \
    --kind web

# Get connection string
APPINSIGHTS_CONNECTION_STRING=$(az monitor app-insights component show \
    --app "rwwwrse-insights" \
    --resource-group $RESOURCE_GROUP \
    --query "connectionString" \
    --output tsv)

# Store in Key Vault
az keyvault secret set \
    --vault-name $KEYVAULT_NAME \
    --name "appinsights-connection-string" \
    --value "$APPINSIGHTS_CONNECTION_STRING"
```

### Key Metrics Available

- **AKS cluster metrics**: CPU, memory, disk, network
- **Application metrics**: Request rates, response times, error rates
- **Custom metrics**: Business logic and application-specific metrics
- **Infrastructure metrics**: Load balancer, storage, network security groups

## Security

### Workload Identity

Secure authentication to Azure services without storing credentials:

```yaml
annotations:
  azure.workload.identity/client-id: YOUR_MANAGED_IDENTITY_CLIENT_ID
```

### Azure Key Vault CSI Driver

Secure secret management:

```yaml
volumes:
- name: azure-keyvault-secrets
  csi:
    driver: secrets-store.csi.k8s.io
    readOnly: true
    volumeAttributes:
      secretProviderClass: "rwwwrse-secrets"
```

### Network Security

- **Azure Network Policies**: Pod-to-pod communication control
- **Network Security Groups**: Traffic filtering at subnet level
- **Private cluster option**: API server accessible only from private network

### Pod Security Standards

```yaml
securityContext:
  runAsNonRoot: true
  runAsUser: 65534
  allowPrivilegeEscalation: false
  readOnlyRootFilesystem: true
  capabilities:
    drop: ["ALL"]
```

## Operations

### Cluster Management

```bash
# Scale cluster
az aks scale \
    --resource-group $RESOURCE_GROUP \
    --name $CLUSTER_NAME \
    --node-count 5

# Upgrade cluster
az aks upgrade \
    --resource-group $RESOURCE_GROUP \
    --name $CLUSTER_NAME \
    --kubernetes-version "1.28"

# Update node pools
az aks nodepool upgrade \
    --resource-group $RESOURCE_GROUP \
    --cluster-name $CLUSTER_NAME \
    --name nodepool1 \
    --kubernetes-version "1.28"
```

### Application Management

```bash
# Update application
kubectl set image deployment/rwwwrse \
    rwwwrse=$ACR_LOGIN_SERVER/rwwwrse:v2.0.0 \
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

# View Azure Load Balancer details
az network lb list --resource-group MC_${RESOURCE_GROUP}_${CLUSTER_NAME}_${LOCATION}
```

## Troubleshooting

### Common Issues

#### Pods Not Starting

```bash
# Check pod status
kubectl get pods -n rwwwrse
kubectl describe pod -l app=rwwwrse -n rwwwrse

# Check logs
kubectl logs -l app=rwwwrse -n rwwwrse

# Common causes:
# - ACR authentication issues
# - Workload Identity misconfiguration
# - Resource constraints
# - Key Vault access issues
```

#### Load Balancer Issues

```bash
# Check service events
kubectl describe service rwwwrse -n rwwwrse

# Check Azure Load Balancer
az network lb list --resource-group MC_${RESOURCE_GROUP}_${CLUSTER_NAME}_${LOCATION}

# Check Network Security Groups
az network nsg list --resource-group MC_${RESOURCE_GROUP}_${CLUSTER_NAME}_${LOCATION}
```

#### Workload Identity Issues

```bash
# Test Workload Identity
kubectl run -it --rm debug \
    --image=mcr.microsoft.com/azure-cli:latest \
    --serviceaccount=rwwwrse-sa \
    --namespace=rwwwrse \
    -- az account show

# Check federated credential
az identity federated-credential list \
    --identity-name "rwwwrse-identity" \
    --resource-group $RESOURCE_GROUP
```

#### Key Vault Issues

```bash
# Test Key Vault access
kubectl exec -it deployment/rwwwrse -n rwwwrse -- ls -la /mnt/secrets-store

# Check CSI driver logs
kubectl logs -l app=secrets-store-csi-driver -n kube-system
kubectl logs -l app=secrets-store-provider-azure -n kube-system
```

### Log Analysis

```bash
# Application logs
kubectl logs -f deployment/rwwwrse -n rwwwrse

# All rwwwrse pods
kubectl logs -f -l app=rwwwrse -n rwwwrse

# Azure Monitor logs
az monitor log-analytics query \
    --workspace "your-workspace-id" \
    --analytics-query "ContainerLog | where ContainerName == 'rwwwrse'"
```

## Cost Optimization

### Azure Spot Instances

```bash
# Create spot node pool
az aks nodepool add \
    --resource-group $RESOURCE_GROUP \
    --cluster-name $CLUSTER_NAME \
    --name spotnodes \
    --priority Spot \
    --eviction-policy Delete \
    --spot-max-price -1 \
    --enable-cluster-autoscaler \
    --min-count 0 \
    --max-count 10 \
    --node-vm-size Standard_D2s_v3
```

### Reserved Instances

Consider Azure Reserved VM Instances for production workloads with predictable usage patterns.

### Resource Optimization

```bash
# Monitor resource usage
kubectl top pods -n rwwwrse
kubectl top nodes

# Use Azure Advisor recommendations
az advisor recommendation list --category Cost
```

## CI/CD Integration

### Azure DevOps Pipeline

```yaml
# azure-pipelines.yml
trigger:
- main

pool:
  vmImage: 'ubuntu-latest'

variables:
  acrName: 'rwwwrseacr'
  resourceGroup: 'rwwwrse-rg'
  clusterName: 'rwwwrse-aks'

stages:
- stage: Build
  jobs:
  - job: BuildAndPush
    steps:
    - task: AzureCLI@2
      displayName: 'Build and push image'
      inputs:
        azureSubscription: 'Azure-Subscription'
        scriptType: 'bash'
        scriptLocation: 'inlineScript'
        inlineScript: |
          az acr build --registry $(acrName) --image rwwwrse:$(Build.BuildId) .

- stage: Deploy
  dependsOn: Build
  jobs:
  - job: DeployToAKS
    steps:
    - task: AzureCLI@2
      displayName: 'Deploy to AKS'
      inputs:
        azureSubscription: 'Azure-Subscription'
        scriptType: 'bash'
        scriptLocation: 'inlineScript'
        inlineScript: |
          az aks get-credentials --resource-group $(resourceGroup) --name $(clusterName)
          kubectl set image deployment/rwwwrse rwwwrse=$(acrName).azurecr.io/rwwwrse:$(Build.BuildId) -n rwwwrse
          kubectl rollout status deployment/rwwwrse -n rwwwrse
```

### GitHub Actions with Azure

```yaml
name: Deploy to AKS
on:
  push:
    branches: [main]

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v3
    
    - uses: azure/login@v1
      with:
        creds: ${{ secrets.AZURE_CREDENTIALS }}
    
    - name: Build and push image
      run: |
        az acr build --registry rwwwrseacr --image rwwwrse:${{ github.sha }} .
    
    - name: Deploy to AKS
      run: |
        az aks get-credentials --resource-group rwwwrse-rg --name rwwwrse-aks
        kubectl set image deployment/rwwwrse rwwwrse=rwwwrseacr.azurecr.io/rwwwrse:${{ github.sha }} -n rwwwrse
        kubectl rollout status deployment/rwwwrse -n rwwwrse
```

## Cleanup

### Remove Application

```bash
# Delete rwwwrse resources
kubectl delete -f deployment.yaml
kubectl delete -f configmap.yaml
kubectl delete -f namespace.yaml
```

### Remove Azure Resources

```bash
# Delete AKS cluster
az aks delete \
    --name $CLUSTER_NAME \
    --resource-group $RESOURCE_GROUP \
    --yes --no-wait

# Delete other resources
az keyvault delete --name $KEYVAULT_NAME --resource-group $RESOURCE_GROUP
az acr delete --name $ACR_NAME --resource-group $RESOURCE_GROUP --yes
az identity delete --name "rwwwrse-identity" --resource-group $RESOURCE_GROUP

# Delete resource group (removes all resources)
az group delete --name $RESOURCE_GROUP --yes --no-wait
```

## Additional Resources

- [Azure Kubernetes Service Documentation](https://docs.microsoft.com/azure/aks/)
- [Workload Identity Guide](https://docs.microsoft.com/azure/aks/workload-identity-overview)
- [AKS Best Practices](https://docs.microsoft.com/azure/aks/best-practices)
- [Azure Monitor for Containers](https://docs.microsoft.com/azure/azure-monitor/containers/container-insights-overview)
- [Key Vault CSI Driver](https://docs.microsoft.com/azure/aks/csi-secrets-store-driver)

## Getting Help

For AKS-specific issues:

1. Review Azure Portal logs and monitoring
2. Consult the [AKS documentation](https://docs.microsoft.com/azure/aks/)
3. Check Azure status page for service issues
4. Review RBAC permissions and Workload Identity setup

For rwwwrse-specific issues, refer to the [main documentation](../../../docs/DEPLOYMENT.md).

Remember to replace placeholder values (subscription-id, resource-group, domain names) with your actual Azure resources before deployment.
