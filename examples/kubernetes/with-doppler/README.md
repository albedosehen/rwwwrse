# Kubernetes with Doppler Secrets Management

This example demonstrates how to use [Doppler](https://www.doppler.com/) for secrets management with rwwwrse in a Kubernetes environment.

## Overview

This example shows a Kubernetes deployment of rwwwrse that:

1. Uses an init container to fetch secrets from Doppler
2. Makes secrets available to the main container
3. Securely manages Doppler tokens in Kubernetes secrets
4. Automates the secrets loading process

## Prerequisites

- A Kubernetes cluster (e.g., Minikube, EKS, GKE, AKS)
- `kubectl` configured to connect to your cluster
- A Doppler account and project created
- Doppler CLI installed locally (for setup)

## Setup

### 1. Create a Doppler Service Token

First, create a service token in your Doppler account that has access to your project:

```bash
doppler configs tokens create rwwwrse-k8s --project rwwwrse --config dev --plain
```

### 2. Create the Kubernetes Secret for Doppler Token

Create a Kubernetes secret containing your Doppler token:

```bash
kubectl create secret generic doppler-token \
  --namespace rwwwrse \
  --from-literal=token=dp.st.your-token-here
```

Replace `dp.st.your-token-here` with your actual Doppler service token.

### 3. Configure Your Secrets in Doppler

Add your application secrets to Doppler. For example:

- `RWWWRSE_SECURITY_API_KEYS`
- `RWWWRSE_SECURITY_BASIC_AUTH_USERS`
- `RWWWRSE_TLS_CERT_EMAIL`

### 4. Apply the Kubernetes Configuration

```bash
kubectl apply -f namespace.yaml
kubectl apply -f configmap.yaml
kubectl apply -f deployment.yaml
```

## How It Works

1. **Init Container**: An init container runs before the main rwwwrse container and:
   - Uses the Doppler CLI to download secrets
   - Stores secrets in a shared volume
   - Ensures secrets are available before the main container starts

2. **Secrets Management**:
   - The Doppler token is stored as a Kubernetes Secret
   - Secrets are fetched at pod startup
   - Secrets are mounted to the main container

3. **Benefits**:
   - Centralized secrets management
   - No secrets in Git/YAML files
   - Easy secret rotation
   - Environment-specific configuration

## Security Considerations

- Use role-based access control (RBAC) to restrict access to the Doppler secret
- Consider using a secrets management solution like HashiCorp Vault for the Doppler token
- Rotate Doppler service tokens regularly
- Use service accounts with minimal permissions

## Alternative Approaches

1. **Kubernetes Operator**: For production use, consider implementing a Kubernetes operator for Doppler that can automatically sync secrets.

2. **Kubernetes External Secrets**: The [External Secrets Operator](https://external-secrets.io/) can be configured to pull secrets from Doppler and create Kubernetes secrets.

3. **Sidecar Container**: Instead of an init container, you could use a sidecar container to periodically refresh secrets.

## Troubleshooting

If you encounter issues:

1. Check the init container logs:
   ```bash
   kubectl logs -n rwwwrse pod/rwwwrse-pod-name -c doppler-secrets
   ```

2. Verify the Doppler token is correctly stored:
   ```bash
   kubectl get secret -n rwwwrse doppler-token -o jsonpath='{.data.token}' | base64 --decode
   ```

3. Check that the secret file exists in the container:
   ```bash
   kubectl exec -n rwwwrse rwwwrse-pod-name -- cat /etc/secrets/doppler-secrets.env