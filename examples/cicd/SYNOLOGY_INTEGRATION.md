# Synology NAS CI/CD Integration

This document describes the integration of Synology NAS deployment as the final stage in the modular CI/CD architecture for rwwwrse.

## Overview

The Synology integration provides automated deployment to a self-hosted Synology NAS environment as part of the complete CI/CD pipeline. This enables a full deployment flow from development through production to a personal/homelab environment.

## Architecture

### Pipeline Flow

```
CI Pipeline → CD Pipeline → Synology Deployment
     ↓             ↓              ↓
   Build        Deploy to      Deploy to
   Test         Cloud Envs     Self-hosted
   Security     (Dev/Staging/   Synology NAS
   Scan         Production)
```

### Workflow Dependencies

1. **CI Pipeline** ([`ci.yml`](.github/workflows/ci.yml))
   - Builds and tests the application
   - Creates validated container images
   - Runs security scans

2. **CD Pipeline** ([`cd.yml`](.github/workflows/cd.yml))
   - Deploys to development environment
   - Deploys to staging environment (with approval)
   - Deploys to production environment (with approval)
   - **Triggers Synology deployment after production**

3. **Synology Deployment** ([`syno.yaml`](.github/workflows/syno.yaml))
   - Deploys to self-hosted Synology NAS
   - Uses validated container images from CD pipeline
   - Provides comprehensive health checks

## Components

### 1. Synology Composite Action

**Location**: [`.github/actions/deploy-synology/action.yml`](.github/actions/deploy-synology/action.yml)

**Purpose**: Reusable deployment logic for Synology NAS environments.

**Key Features**:
- Input validation and repository structure verification
- Docker image building (optional) or use of pre-built images
- Docker Compose deployment with proper environment setup
- Comprehensive health checks and verification
- Detailed logging and error reporting

**Inputs**:
- `image-tag`: Docker image tag to deploy (required)
- `image-digest`: Container image digest for verification (optional)
- `skip-build`: Skip local docker build step (default: false)
- `compose-file`: Docker Compose file to use (default: docker-compose.deploy.yml)
- `health-check-host`: Host for health checks (default: localhost)
- `health-check-port`: Port for health checks (default: 8080)
- `health-check-timeout`: Health check timeout in seconds (default: 180)

### 2. Updated Synology Workflow

**Location**: [`.github/workflows/syno.yaml`](.github/workflows/syno.yaml)

**Trigger Methods**:
1. **Workflow Call** (Primary): Called by CD pipeline after successful production deployment
2. **Manual Dispatch**: For emergency deployments or testing

**Key Changes from Original**:
- Converted from standalone workflow to reusable workflow
- Accepts validated container images from CD pipeline
- Integrated with notification system
- Added deployment metadata tracking
- Improved error handling and rollback capabilities

**Jobs**:
1. **load-synology-config**: Loads environment-specific configuration
2. **pre-deployment**: Validates container images and deployment configuration
3. **deploy-synology**: Main deployment using the composite action
4. **notify-deployment**: Sends notifications and creates deployment summary

### 3. CD Pipeline Integration

**Location**: [`.github/workflows/cd.yml`](.github/workflows/cd.yml)

**Integration Points**:
- Synology deployment depends on production deployment completion
- Passes validated container images and metadata to Synology workflow
- Includes Synology deployment status in overall pipeline reporting

**Deployment Strategy**:
```yaml
deploy-synology:
  needs: [validate-ci, deploy-production]
  if: needs.validate-ci.outputs.deploy-synology == 'true' && 
      (success() || needs.validate-ci.outputs.deploy-production == 'false')
  uses: ./.github/workflows/syno.yaml
  with:
    image_tag: ${{ needs.validate-ci.outputs.image-tag }}
    image_digest: ${{ needs.validate-ci.outputs.image-digest }}
    skip_build: true
    deployment_metadata: |
      {
        "trigger": "${{ github.event_name }}",
        "branch": "${{ github.ref_name }}",
        "commit": "${{ github.sha }}",
        "workflow_run_id": "${{ github.run_id }}",
        "deployment_time": "${{ github.event.head_commit.timestamp }}",
        "validated_by_cd": true,
        "coverage": "${{ needs.validate-ci.outputs.coverage }}"
      }
```

## Configuration

### Environment Configuration

**Location**: [`.github/config/environments.yml`](.github/config/environments.yml)

The Synology environment is configured with:
- **Type**: `docker-compose` (vs. Kubernetes for other environments)
- **Auto Deploy**: `true` (automatic deployment after production)
- **Self-hosted Runner**: Uses `[self-hosted, synology]` labels
- **Docker Compose**: Uses `docker-compose.deploy.yml` file
- **Health Checks**: Configured for localhost:8080
- **Volume Mapping**: Proper host path mapping for Synology filesystem

### Notification Configuration

**Location**: [`.github/config/notifications.yml`](.github/config/notifications.yml)

Synology-specific notifications include:
- **Slack**: `#synology-deployments` channel with `@homelab-team` mentions
- **Email**: Notifications to homelab administrators
- **GitHub Issues**: Automatic issue creation for deployment failures
- **Additional Fields**: Deployment metadata and container status

## Deployment Process

### Automatic Deployment (Main Branch)

1. **Code Push** to `main` branch
2. **CI Pipeline** runs automatically
   - Builds application
   - Runs tests and security scans
   - Creates validated container images
3. **CD Pipeline** triggered after CI success
   - Deploys to development environment
   - Deploys to staging environment
   - Deploys to production environment (with approval)
   - **Deploys to Synology NAS** using validated images
4. **Notifications** sent for all deployment results

### Manual Deployment

```bash
# Deploy specific image tag to Synology only
gh workflow run cd.yml \
  --field environment=synology \
  --field image_tag=v1.2.3

# Emergency deployment with custom image
gh workflow run syno.yaml \
  --field image_tag=hotfix-v1.2.4 \
  --field skip_build=false
```

## Self-hosted Runner Setup

### Prerequisites

1. **Synology NAS** with Docker and Docker Compose installed
2. **GitHub Actions Runner** installed and configured
3. **Runner Labels**: `self-hosted` and `synology`
4. **Network Access**: Ability to pull from GitHub Container Registry

### Runner Configuration

```bash
# On Synology NAS
./config.sh --url https://github.com/your-org/rwwwrse \
            --token YOUR_RUNNER_TOKEN \
            --labels self-hosted,synology \
            --name synology-runner
```

### Directory Structure

```
/volume1/docker/github-runner/
├── _work/                    # GitHub Actions workspace
├── actions-runner/           # Runner installation
└── rwwwrse/                 # Project workspace
    ├── docker-data/         # Persistent data
    │   ├── certs/           # SSL certificates
    │   └── logs/            # Application logs
    └── docker-compose.deploy.yml
```

## Security Considerations

### Secrets Management

Required secrets for Synology deployment:
- `DOPPLER_TOKEN`: For configuration management (optional)
- `SLACK_WEBHOOK_URL`: For notifications
- `HOSTNAME`: Synology NAS hostname (default: localhost)

### Network Security

- Synology deployment runs on internal network
- Health checks use localhost endpoints
- No external exposure required for deployment process
- Container registry access through authenticated pulls

### Access Control

- Self-hosted runner has limited scope to Synology environment only
- No access to production cloud environments
- Separate notification channels for homelab vs. production

## Monitoring and Troubleshooting

### Health Checks

The deployment includes comprehensive health checks:
1. **Container Status**: Verify containers are running
2. **Application Health**: HTTP health endpoint checks
3. **Metrics Endpoint**: Prometheus metrics availability
4. **Network Connectivity**: Internal service communication

### Logging

Deployment logs are available in:
- **GitHub Actions**: Workflow run logs
- **Synology NAS**: `/volume1/docker/github-runner/_work/.../docker-data/logs/`
- **Container Logs**: `docker logs rwwwrse-app`

### Common Issues

1. **Container Build Failures**
   ```bash
   # Check Docker daemon status
   docker version
   docker system info
   ```

2. **Health Check Failures**
   ```bash
   # Manual health check
   curl http://localhost:8080/health
   
   # Check container status
   docker ps --filter "name=rwwwrse"
   ```

3. **Volume Mount Issues**
   ```bash
   # Verify directory permissions
   ls -la /volume1/docker/github-runner/_work/
   
   # Check mount points
   docker inspect rwwwrse-app | grep Mounts -A 10
   ```

## Rollback Procedures

### Automatic Rollback

The deployment includes automatic rollback on health check failures:
- Health checks run for 3 minutes with 10-second intervals
- Failure triggers container restart and re-verification
- Persistent failures result in deployment failure status

### Manual Rollback

```bash
# Rollback to previous version
docker compose -f docker-compose.deploy.yml down
docker compose -f docker-compose.deploy.yml up -d

# Or deploy specific version
gh workflow run syno.yaml --field image_tag=v1.2.2
```

## Integration Benefits

1. **Consistent Images**: Same validated container images across all environments
2. **Automated Flow**: No manual intervention required for standard deployments
3. **Comprehensive Monitoring**: Full visibility into deployment status
4. **Error Handling**: Proper rollback and notification on failures
5. **Modular Design**: Reusable components for future self-hosted environments

## Future Enhancements

1. **Blue-Green Deployment**: Support for zero-downtime deployments
2. **Backup Integration**: Automatic backup before deployment
3. **Multi-NAS Support**: Deploy to multiple Synology devices
4. **Performance Monitoring**: Integration with monitoring stack
5. **Automated Testing**: Post-deployment integration tests

## Related Documentation

- [Main CI/CD README](README.md)
- [Environment Configuration](.github/config/environments.yml)
- [Notification Configuration](.github/config/notifications.yml)
- [Docker Compose Examples](../docker-compose/)
- [Operations Guide](../../docs/OPERATIONS.md)