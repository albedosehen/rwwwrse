# rwwwrse on AWS EKS

This example demonstrates deploying rwwwrse on Amazon Elastic Kubernetes Service (EKS) with AWS-specific integrations and best practices.

## 🏗️ Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                        AWS EKS Cluster                     │
│  ┌─────────────────────────────────────────────────────┐   │
│  │                rwwwrse Namespace                   │   │
│  │                                                     │   │
│  │  ┌─────────────┐    ┌─────────────────────────┐    │   │
│  │  │  Network    │    │      rwwwrse Pods      │    │   │
│  │  │ Load Balancer│────│   (Auto-scaled 3-50)   │    │   │
│  │  │   (NLB)     │    │                         │    │   │
│  │  └─────────────┘    └─────────────────────────┘    │   │
│  │                                                     │   │
│  │  ┌─────────────┐    ┌─────────────────────────┐    │   │
│  │  │  ECR Image  │    │    Sample Applications  │    │   │
│  │  │  Registry   │    │   (Auto-scaled 2-20)   │    │   │
│  │  └─────────────┘    └─────────────────────────┘    │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐   │
│  │                AWS Integrations                     │   │
│  │  • IAM Roles & Service Accounts (IRSA)             │   │
│  │  • Certificate Manager (ACM)                       │   │
│  │  • External DNS                                     │   │
│  │  • Cluster Autoscaler                              │   │
│  │  • CloudWatch Logging                              │   │
│  └─────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
```

## 🎯 AWS-Specific Features

- **Network Load Balancer (NLB)**: High-performance Layer 4 load balancing
- **IAM Roles for Service Accounts (IRSA)**: Secure AWS API access
- **ECR Integration**: Container image registry
- **Certificate Manager**: Automatic SSL/TLS certificates
- **External DNS**: Automatic DNS record management
- **Cluster Autoscaler**: Automatic node scaling
- **CloudWatch Integration**: Comprehensive logging and monitoring

## 📋 Prerequisites

### AWS Resources

```bash
# Required AWS CLI and tools
aws --version  # >= 2.0.0
eksctl version  # >= 0.100.0
kubectl version --client  # >= 1.20.0
helm version  # >= 3.7.0 (optional)
```

### AWS Permissions

The deploying user needs permissions for:
- EKS cluster management
- IAM role creation
- ECR repository access
- VPC and networking
- Certificate Manager
- Route 53 (for External DNS)

### Domain and SSL

- Domain registered in Route 53 or delegated to Route 53
- Certificate issued/imported in AWS Certificate Manager

## 🚀 Quick Start

### 1. Create EKS Cluster

```bash
# Create cluster with eksctl
eksctl create cluster \
  --name rwwwrse-cluster \
  --region us-west-2 \
  --version 1.28 \
  --nodegroup-name standard-workers \
  --node-type m5.large \
  --nodes 3 \
  --nodes-min 1 \
  --nodes-max 10 \
  --managed \
  --with-oidc \
  --ssh-access \
  --ssh-public-key my-key

# Verify cluster
kubectl get nodes
```

### 2. Set Up AWS Load Balancer Controller

```bash
# Create IAM policy and role
curl -o iam_policy.json https://raw.githubusercontent.com/kubernetes-sigs/aws-load-balancer-controller/v2.5.4/docs/install/iam_policy.json

aws iam create-policy \
    --policy-name AWSLoadBalancerControllerIAMPolicy \
    --policy-document file://iam_policy.json

eksctl create iamserviceaccount \
  --cluster=rwwwrse-cluster \
  --namespace=kube-system \
  --name=aws-load-balancer-controller \
  --role-name AmazonEKSLoadBalancerControllerRole \
  --attach-policy-arn=arn:aws:iam::ACCOUNT_ID:policy/AWSLoadBalancerControllerIAMPolicy \
  --approve

# Install AWS Load Balancer Controller
helm repo add eks https://aws.github.io/eks-charts
helm repo update

helm install aws-load-balancer-controller eks/aws-load-balancer-controller \
  -n kube-system \
  --set clusterName=rwwwrse-cluster \
  --set serviceAccount.create=false \
  --set serviceAccount.name=aws-load-balancer-controller
```

### 3. Set Up External DNS (Optional)

```bash
# Create IAM policy for External DNS
cat <<EOF > external-dns-policy.json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "route53:ChangeResourceRecordSets"
      ],
      "Resource": [
        "arn:aws:route53:::hostedzone/*"
      ]
    },
    {
      "Effect": "Allow",
      "Action": [
        "route53:ListHostedZones",
        "route53:ListResourceRecordSets"
      ],
      "Resource": [
        "*"
      ]
    }
  ]
}
EOF

aws iam create-policy \
    --policy-name ExternalDNSIAMPolicy \
    --policy-document file://external-dns-policy.json

eksctl create iamserviceaccount \
  --cluster=rwwwrse-cluster \
  --namespace=kube-system \
  --name=external-dns \
  --role-name ExternalDNSRole \
  --attach-policy-arn=arn:aws:iam::ACCOUNT_ID:policy/ExternalDNSIAMPolicy \
  --approve
```

### 4. Build and Push rwwwrse Image

```bash
# Create ECR repository
aws ecr create-repository \
    --repository-name rwwwrse \
    --region us-west-2

# Get login token
aws ecr get-login-password --region us-west-2 | \
    docker login --username AWS --password-stdin ACCOUNT_ID.dkr.ecr.us-west-2.amazonaws.com

# Build and push image
docker build -t rwwwrse:latest .
docker tag rwwwrse:latest ACCOUNT_ID.dkr.ecr.us-west-2.amazonaws.com/rwwwrse:latest
docker push ACCOUNT_ID.dkr.ecr.us-west-2.amazonaws.com/rwwwrse:latest
```

### 5. Create IAM Role for rwwwrse

```bash
# Create service account with IAM role
eksctl create iamserviceaccount \
  --cluster=rwwwrse-cluster \
  --namespace=rwwwrse \
  --name=rwwwrse-service-account \
  --role-name=rwwwrse-service-role \
  --attach-policy-arn=arn:aws:iam::aws:policy/CloudWatchAgentServerPolicy \
  --approve
```

### 6. Deploy rwwwrse

```bash
# Navigate to AWS EKS example directory
cd examples/kubernetes/cloud/aws-eks

# Update configuration files with your values
# - Replace ACCOUNT_ID with your AWS account ID
# - Replace CERTIFICATE_ID with your ACM certificate ID
# - Replace your-domain.com with your actual domain

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

## ⚙️ Configuration

### ECR Image Repository

Update the image reference in [`deployment.yaml`](deployment.yaml):

```yaml
image: YOUR_ACCOUNT.dkr.ecr.us-west-2.amazonaws.com/rwwwrse:latest
```

### Certificate Manager Integration

Configure SSL certificate in the service annotation:

```yaml
annotations:
  service.beta.kubernetes.io/aws-load-balancer-ssl-cert: arn:aws:acm:us-west-2:ACCOUNT_ID:certificate/CERTIFICATE_ID
```

### External DNS Configuration

Add hostname annotation for automatic DNS management:

```yaml
annotations:
  external-dns.alpha.kubernetes.io/hostname: rwwwrse.your-domain.com
```

### Auto Scaling Configuration

The deployment includes both:

#### Horizontal Pod Autoscaler (HPA)
- **Min replicas**: 3
- **Max replicas**: 50
- **CPU target**: 70%
- **Memory target**: 80%

#### Cluster Autoscaler
Configured via node group settings and tolerations.

### Resource Requests and Limits

```yaml
resources:
  requests:
    cpu: 200m
    memory: 256Mi
  limits:
    cpu: 1000m
    memory: 1Gi
```

## 📊 Monitoring and Observability

### CloudWatch Integration

```bash
# Install CloudWatch Container Insights
curl https://raw.githubusercontent.com/aws-samples/amazon-cloudwatch-container-insights/latest/k8s-deployment-manifest-templates/deployment-mode/daemonset/container-insights-monitoring/quickstart/cwagent-fluentd-quickstart.yaml | \
  sed "s/{{cluster_name}}/rwwwrse-cluster/;s/{{region_name}}/us-west-2/" | \
  kubectl apply -f -
```

### Prometheus Integration

rwwwrse exposes metrics on port 9090 that can be scraped by Prometheus:

```yaml
annotations:
  prometheus.io/scrape: "true"
  prometheus.io/port: "9090"
  prometheus.io/path: "/metrics"
```

### Key Metrics Available

- Request rate and response times
- Error rates by status code
- Active connections
- Resource utilization
- Load balancer metrics

## 🔐 Security

### IAM Roles for Service Accounts (IRSA)

```yaml
serviceAccountName: rwwwrse-service-account
annotations:
  eks.amazonaws.com/role-arn: arn:aws:iam::ACCOUNT_ID:role/rwwwrse-service-role
```

### Security Context

```yaml
securityContext:
  runAsNonRoot: true
  runAsUser: 65534
  allowPrivilegeEscalation: false
  readOnlyRootFilesystem: true
  capabilities:
    drop:
    - ALL
```

### Network Policies

Network policies are configured to restrict traffic between pods and namespaces.

### Pod Security Standards

The namespace is configured with restricted pod security standards.

## 🔧 Operations

### Scaling Operations

```bash
# Manual scaling
kubectl scale deployment rwwwrse --replicas=5 -n rwwwrse

# Check HPA status
kubectl get hpa -n rwwwrse

# Check node autoscaling
kubectl get nodes
```

### Rolling Updates

```bash
# Update image
kubectl set image deployment/rwwwrse rwwwrse=ACCOUNT_ID.dkr.ecr.us-west-2.amazonaws.com/rwwwrse:v2.0.0 -n rwwwrse

# Monitor rollout
kubectl rollout status deployment/rwwwrse -n rwwwrse

# Rollback if needed
kubectl rollout undo deployment/rwwwrse -n rwwwrse
```

### Certificate Management

```bash
# List certificates
aws acm list-certificates --region us-west-2

# Check certificate status
aws acm describe-certificate --certificate-arn arn:aws:acm:us-west-2:ACCOUNT_ID:certificate/CERTIFICATE_ID
```

### Load Balancer Management

```bash
# Check load balancer status
kubectl get service rwwwrse -n rwwwrse -o wide

# Get load balancer details
aws elbv2 describe-load-balancers \
  --query 'LoadBalancers[?contains(LoadBalancerName, `rwwwrse`)]'
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
# - ECR access issues
# - IAM role configuration
# - Resource constraints
# - Image pull errors
```

#### Load Balancer Not Accessible

```bash
# Check service status
kubectl get service rwwwrse -n rwwwrse

# Check AWS Load Balancer Controller logs
kubectl logs -n kube-system -l app.kubernetes.io/name=aws-load-balancer-controller

# Verify certificate
aws acm describe-certificate --certificate-arn YOUR_CERT_ARN
```

#### Auto Scaling Issues

```bash
# Check HPA status
kubectl describe hpa rwwwrse-hpa -n rwwwrse

# Check metrics server
kubectl top pods -n rwwwrse

# Check cluster autoscaler logs
kubectl logs -n kube-system -l app=cluster-autoscaler
```

### Log Analysis

```bash
# Application logs
kubectl logs -f deployment/rwwwrse -n rwwwrse

# All rwwwrse pods
kubectl logs -f -l app=rwwwrse -n rwwwrse

# CloudWatch logs
aws logs describe-log-groups --log-group-name-prefix /aws/containerinsights/rwwwrse-cluster
```

## 💰 Cost Optimization

### Resource Optimization

```bash
# Check resource usage
kubectl top pods -n rwwwrse
kubectl top nodes

# Review resource requests vs usage
kubectl describe deployment rwwwrse -n rwwwrse
```

### Node Group Optimization

```bash
# Use spot instances for non-critical workloads
eksctl create nodegroup \
  --cluster=rwwwrse-cluster \
  --name=spot-workers \
  --instance-types=m5.large,m5.xlarge \
  --spot \
  --nodes-min=0 \
  --nodes-max=10
```

### Storage Optimization

- Use gp3 volumes for better cost/performance
- Configure appropriate volume sizes
- Clean up unused resources regularly

## 🔄 CI/CD Integration

### GitHub Actions Example

```yaml
name: Deploy to EKS
on:
  push:
    branches: [main]

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v3
    
    - name: Configure AWS credentials
      uses: aws-actions/configure-aws-credentials@v2
      with:
        aws-access-key-id: ${{ secrets.AWS_ACCESS_KEY_ID }}
        aws-secret-access-key: ${{ secrets.AWS_SECRET_ACCESS_KEY }}
        aws-region: us-west-2
    
    - name: Login to ECR
      uses: aws-actions/amazon-ecr-login@v1
    
    - name: Build and push image
      run: |
        docker build -t rwwwrse:${{ github.sha }} .
        docker tag rwwwrse:${{ github.sha }} ${{ secrets.ECR_REGISTRY }}/rwwwrse:${{ github.sha }}
        docker push ${{ secrets.ECR_REGISTRY }}/rwwwrse:${{ github.sha }}
    
    - name: Deploy to EKS
      run: |
        aws eks update-kubeconfig --name rwwwrse-cluster --region us-west-2
        kubectl set image deployment/rwwwrse rwwwrse=${{ secrets.ECR_REGISTRY }}/rwwwrse:${{ github.sha }} -n rwwwrse
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
# Delete the EKS cluster
eksctl delete cluster --name rwwwrse-cluster --region us-west-2

# Clean up IAM policies (optional)
aws iam delete-policy --policy-arn arn:aws:iam::ACCOUNT_ID:policy/AWSLoadBalancerControllerIAMPolicy
aws iam delete-policy --policy-arn arn:aws:iam::ACCOUNT_ID:policy/ExternalDNSIAMPolicy
```

## 📚 Additional Resources

- [Amazon EKS User Guide](https://docs.aws.amazon.com/eks/latest/userguide/)
- [AWS Load Balancer Controller](https://kubernetes-sigs.github.io/aws-load-balancer-controller/)
- [External DNS on AWS](https://github.com/kubernetes-sigs/external-dns/blob/master/docs/tutorials/aws.md)
- [EKS Best Practices](https://aws.github.io/aws-eks-best-practices/)
- [Container Insights](https://docs.aws.amazon.com/AmazonCloudWatch/latest/monitoring/ContainerInsights.html)

## 🆘 Getting Help

For AWS EKS-specific issues:

1. Check the [troubleshooting section](#-troubleshooting)
2. Review AWS service logs and events
3. Consult the [EKS documentation](https://docs.aws.amazon.com/eks/)
4. Check AWS service health dashboard
5. Review IAM permissions and roles

For rwwwrse-specific issues, refer to the [main documentation](../../../docs/DEPLOYMENT.md).

Remember to replace placeholder values (ACCOUNT_ID, CERTIFICATE_ID, domain names) with your actual AWS resources before deployment.