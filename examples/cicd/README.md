# CI/CD Pipeline Examples for rwwwrse

This directory contains comprehensive CI/CD pipeline examples for automated deployment of rwwwrse across different platforms and environments. These pipelines integrate with the deployment examples provided in other directories.

## Overview

CI/CD pipelines automate:
- **Code compilation and testing** for every commit
- **Container image building** and vulnerability scanning
- **Deployment to multiple environments** (dev, staging, production)
- **Health checks and rollback** on deployment failures
- **Security scanning** and compliance checks
- **Notifications** and reporting

## Pipeline Examples

### GitHub Actions
- **[Basic Pipeline](.github/workflows/basic.yml)** - Simple build, test, and deploy
- **[Multi-Environment Pipeline](.github/workflows/multi-env.yml)** - Deploy to dev/staging/prod
- **[Docker Pipeline](.github/workflows/docker.yml)** - Container-focused deployment
- **[Kubernetes Pipeline](.github/workflows/kubernetes.yml)** - K8s deployment with Helm
- **[Cloud-Specific Pipelines](.github/workflows/)** - AWS ECS, Google Cloud Run, Azure ACI

### GitLab CI/CD
- **[Basic Pipeline](.gitlab-ci.yml)** - GitLab CI/CD configuration
- **[Multi-Stage Pipeline](gitlab/multi-stage.yml)** - Complex deployment stages
- **[Auto DevOps](gitlab/auto-devops.yml)** - GitLab's automated pipeline
- **[Container Registry](gitlab/container-registry.yml)** - GitLab Container Registry integration

### Jenkins
- **[Declarative Pipeline](jenkins/Jenkinsfile.declarative)** - Modern Jenkins syntax
- **[Scripted Pipeline](jenkins/Jenkinsfile.scripted)** - Traditional Jenkins approach
- **[Multi-Branch Pipeline](jenkins/Jenkinsfile.multibranch)** - Branch-specific deployments
- **[Blue-Green Deployment](jenkins/blue-green.groovy)** - Zero-downtime deployments

### Azure DevOps
- **[Build Pipeline](azure-devops/azure-pipelines.yml)** - Azure Pipelines configuration
- **[Release Pipeline](azure-devops/release-pipeline.yml)** - Multi-stage releases
- **[Container Pipeline](azure-devops/container-pipeline.yml)** - ACR and ACI deployment

### CircleCI
- **[Basic Configuration](circleci/config.yml)** - CircleCI 2.1 configuration
- **[Orb-Based Pipeline](circleci/orb-config.yml)** - Using CircleCI Orbs
- **[Docker Workflow](circleci/docker-workflow.yml)** - Container-focused pipeline

## Feature Matrix

| Platform | Docker Build | K8s Deploy | Cloud Deploy | Security Scan | Multi-Env | Rollback |
|----------|-------------|------------|--------------|---------------|-----------|----------|
| GitHub Actions | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| GitLab CI/CD | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Jenkins | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Azure DevOps | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| CircleCI | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |

## Pipeline Stages

### 1. Source Control
- **Trigger**: Push to main/develop branches, pull requests
- **Actions**: Checkout code, validate syntax
- **Artifacts**: Source code

### 2. Build & Test
- **Go Build**: Compile rwwwrse binary
- **Unit Tests**: Run Go test suite with coverage
- **Integration Tests**: Test with real backends
- **Linting**: golangci-lint, go vet, gofmt
- **Security**: gosec, dependency scanning

### 3. Container Build
- **Docker Build**: Multi-stage container build
- **Image Scanning**: Vulnerability assessment
- **Registry Push**: Push to container registry
- **Image Signing**: Container image signing (optional)

### 4. Deployment
- **Environment Promotion**: Dev → Staging → Production
- **Infrastructure**: Terraform/CloudFormation (if needed)
- **Application Deploy**: Rolling updates, blue-green, canary
- **Configuration**: Environment-specific settings

### 5. Verification
- **Health Checks**: Application health endpoints
- **Smoke Tests**: Basic functionality verification
- **Performance Tests**: Load testing (optional)
- **Monitoring**: Deploy monitoring configuration

### 6. Notification
- **Success/Failure**: Slack, email, Teams notifications
- **Metrics**: Deployment metrics and dashboards
- **Documentation**: Auto-generated deployment notes

## Environment Strategy

### Development Environment
```yaml
# Trigger: Every push to feature branches
environment:
  name: development
  auto_deploy: true
  approval: false
  variables:
    - RWWWRSE_LOG_LEVEL=debug
    - RWWWRSE_ROUTES_API_TARGET=http://dev-api.internal
```

### Staging Environment
```yaml
# Trigger: Push to develop branch
environment:
  name: staging
  auto_deploy: true
  approval: false
  variables:
    - RWWWRSE_LOG_LEVEL=info
    - RWWWRSE_ROUTES_API_TARGET=http://staging-api.internal
```

### Production Environment
```yaml
# Trigger: Manual deployment or release tags
environment:
  name: production
  auto_deploy: false
  approval: true
  variables:
    - RWWWRSE_LOG_LEVEL=warn
    - RWWWRSE_ROUTES_API_TARGET=http://prod-api.internal
```

## Security Best Practices

### Secret Management
```yaml
# Use platform-native secret management
secrets:
  - DATABASE_URL          # Database connection string
  - DOCKER_REGISTRY_TOKEN # Container registry authentication
  - KUBERNETES_TOKEN      # K8s cluster access
  - CLOUD_CREDENTIALS     # Cloud provider credentials
  - NOTIFICATION_WEBHOOKS # Slack/Teams webhook URLs
```

### Compliance and Scanning
- **SAST**: Static Application Security Testing
- **DAST**: Dynamic Application Security Testing
- **Container Scanning**: Vulnerability assessment
- **Dependency Scanning**: Known CVE detection
- **License Scanning**: Open source license compliance

### Access Control
- **RBAC**: Role-based access control
- **Environment Protection**: Production deployment approvals
- **Branch Protection**: Require PR reviews
- **Audit Logging**: Track all deployment activities

## Deployment Strategies

### Rolling Deployment
```yaml
strategy:
  type: rolling
  max_unavailable: 25%
  max_surge: 25%
  health_check:
    path: /health
    timeout: 30s
    retries: 3
```

### Blue-Green Deployment
```yaml
strategy:
  type: blue_green
  blue_weight: 0
  green_weight: 100
  switch_traffic: manual
  rollback_on_failure: true
```

### Canary Deployment
```yaml
strategy:
  type: canary
  canary_weight: 10
  stable_weight: 90
  success_threshold: 95%
  analysis_duration: 10m
```

## Monitoring Integration

### Metrics Collection
- **Deployment Frequency**: DORA metrics
- **Lead Time**: Code to production time
- **MTTR**: Mean time to recovery
- **Change Failure Rate**: Failed deployment percentage

### Alerting
```yaml
alerts:
  - name: deployment_failure
    condition: deployment_status == "failed"
    channels: [slack, email]
  - name: high_error_rate
    condition: error_rate > 5%
    channels: [pagerduty]
```

## Getting Started

### 1. Choose Your Platform

**For GitHub Projects:**
```bash
# Copy GitHub Actions workflows
cp -r examples/cicd/.github/workflows/ .github/workflows/
```

**For GitLab Projects:**
```bash
# Copy GitLab CI configuration
cp examples/cicd/.gitlab-ci.yml .gitlab-ci.yml
```

**For Jenkins:**
```bash
# Copy Jenkinsfile to your repository root
cp examples/cicd/jenkins/Jenkinsfile .
```

### 2. Configure Secrets

Each platform requires specific secrets configuration:

**GitHub Actions:**
```bash
# Set repository secrets in GitHub UI or using GitHub CLI
gh secret set DOCKER_USERNAME --body="your-username"
gh secret set DOCKER_PASSWORD --body="your-password"
gh secret set KUBECONFIG --body-file="kubeconfig.yaml"
```

**GitLab CI/CD:**
```bash
# Set CI/CD variables in GitLab UI or using GitLab CLI
glab variable set DOCKER_USERNAME "your-username"
glab variable set DOCKER_PASSWORD "your-password" --masked
glab variable set KUBECONFIG "$(cat kubeconfig.yaml)" --masked
```

### 3. Customize Configuration

Update pipeline files with your specific:
- **Container registry** URLs and credentials
- **Deployment targets** (K8s clusters, cloud services)
- **Environment variables** and configuration
- **Notification channels** (Slack, Teams, email)

### 4. Test Pipeline

```bash
# Create a test branch and push changes
git checkout -b test-pipeline
git add .github/workflows/
git commit -m "Add CI/CD pipeline"
git push origin test-pipeline

# Create pull request to trigger pipeline
# Monitor pipeline execution in platform UI
```

## Advanced Patterns

### Multi-Cloud Deployment
```yaml
# Deploy to multiple cloud providers
deploy:
  parallel:
    - aws_ecs
    - google_cloud_run
    - azure_container_instances
  strategy: all_or_nothing
```

### Feature Flag Integration
```yaml
# Deploy with feature flags
deploy:
  feature_flags:
    - new_routing_algorithm: 10%
    - enhanced_logging: 100%
    - experimental_cache: 0%
```

### Database Migrations
```yaml
# Handle database schema changes
pre_deploy:
  - run: migrate up
    rollback: migrate down
  - run: seed test data
    condition: environment == "staging"
```

## Troubleshooting

### Common Pipeline Issues

1. **Build Failures:**
```bash
# Check Go module dependencies
go mod tidy
go mod verify

# Run tests locally
go test ./...
go test -race ./...
```

2. **Container Build Issues:**
```bash
# Test Docker build locally
docker build -t rwwwrse:test .
docker run --rm rwwwrse:test --version

# Check multi-platform builds
docker buildx build --platform linux/amd64,linux/arm64 .
```

3. **Deployment Failures:**
```bash
# Check Kubernetes resources
kubectl get pods -l app=rwwwrse
kubectl describe deployment rwwwrse
kubectl logs -l app=rwwwrse

# Verify service connectivity
kubectl port-forward svc/rwwwrse 8080:80
curl http://localhost:8080/health
```

### Pipeline Debugging

Enable debug logging in your pipelines:

```yaml
# GitHub Actions
- name: Debug
  run: echo "::debug::Debug message"
  env:
    ACTIONS_STEP_DEBUG: true

# GitLab CI
script:
  - set -x  # Enable bash debug mode
  - echo "Debug information"

# Jenkins
pipeline {
  options {
    parallelsAlwaysFailFast()
    timestamps()
    timeout(time: 1, unit: 'HOURS')
  }
}
```

## Platform-Specific Features

### GitHub Actions
- **Marketplace Actions**: Reusable community actions
- **Matrix Builds**: Test multiple Go versions/platforms
- **Environments**: Built-in environment management
- **Security**: Dependabot, CodeQL scanning

### GitLab CI/CD
- **Auto DevOps**: Automatic pipeline generation
- **Container Registry**: Built-in registry integration
- **Review Apps**: Temporary deployment environments
- **Compliance**: Built-in security and compliance tools

### Jenkins
- **Plugins**: Extensive plugin ecosystem
- **Blue Ocean**: Modern UI for pipelines
- **Pipeline as Code**: Jenkinsfile in repository
- **Distributed Builds**: Master-agent architecture

### Azure DevOps
- **Azure Integration**: Native Azure service integration
- **Boards**: Work item tracking
- **Artifacts**: Package management
- **Test Plans**: Comprehensive testing tools

## Migration Guides

### From Jenkins to GitHub Actions
1. **Convert Jenkinsfile** to workflow YAML
2. **Migrate credentials** to GitHub secrets
3. **Update triggers** and branch strategies
4. **Test pipeline** with sample deployment

### From GitLab CI to Azure DevOps
1. **Export GitLab configuration** and variables
2. **Create Azure DevOps project** and service connections
3. **Convert pipeline syntax** to Azure Pipelines YAML
4. **Migrate container registry** integration

## Best Practices

### Pipeline Design
- **Keep pipelines fast** (<10 minutes for basic builds)
- **Fail fast** on obvious errors
- **Parallel execution** where possible
- **Caching** for dependencies and build artifacts
- **Idempotent deployments** that can be run multiple times

### Security
- **Minimal secrets exposure** only where needed
- **Least privilege** access for service accounts
- **Regular secret rotation** and audit
- **Signed commits** and verified deployments

### Monitoring
- **Pipeline metrics** and alerting
- **Deployment tracking** and history
- **Performance monitoring** of builds
- **Cost optimization** for cloud resources

## Related Documentation

- [Docker Compose Examples](../docker-compose/) - Local development and testing
- [Kubernetes Examples](../kubernetes/) - Container orchestration deployment
- [Cloud-Specific Examples](../cloud-specific/) - Cloud platform deployments
- [Bare-Metal Examples](../bare-metal/) - Traditional server deployment
- [Operations Guide](../../docs/OPERATIONS.md) - Monitoring and troubleshooting
- [SSL/TLS Guide](../../docs/SSL-TLS.md) - Certificate management
- [Configuration Guide](../../docs/CONFIGURATION.md) - Environment setup