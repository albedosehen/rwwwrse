# AWS ECS Deployment Guide for rwwwrse

Amazon Elastic Container Service (ECS) is a fully managed container orchestration service. This guide shows how to deploy rwwwrse on ECS with both EC2 and Fargate launch types.

## Overview

AWS ECS provides:
- **Fully managed container orchestration** without Kubernetes complexity
- **Fargate serverless** or **EC2 managed** compute options
- **Built-in load balancing** with Application Load Balancer (ALB)
- **Auto-scaling** based on CloudWatch metrics
- **AWS ecosystem integration** (IAM, CloudWatch, Systems Manager, etc.)
- **Service discovery** with AWS Cloud Map

## Prerequisites

- AWS CLI installed and configured
- Docker image of rwwwrse in Amazon ECR
- VPC with public/private subnets configured
- IAM permissions for ECS, ALB, and CloudWatch

## Quick Start

### 1. Setup AWS Environment

```bash
# Set environment variables
export AWS_REGION="us-east-1"
export CLUSTER_NAME="rwwwrse-cluster"
export SERVICE_NAME="rwwwrse-service"
export REPOSITORY_URI="123456789012.dkr.ecr.us-east-1.amazonaws.com/rwwwrse"

# Create ECS cluster
aws ecs create-cluster \
  --cluster-name $CLUSTER_NAME \
  --capacity-providers FARGATE EC2 \
  --default-capacity-provider-strategy capacityProvider=FARGATE,weight=1
```

### 2. Push Image to ECR

```bash
# Create ECR repository
aws ecr create-repository --repository-name rwwwrse

# Get login token
aws ecr get-login-password --region $AWS_REGION | docker login --username AWS --password-stdin $REPOSITORY_URI

# Build and push image
docker build -t rwwwrse .
docker tag rwwwrse:latest $REPOSITORY_URI:latest
docker push $REPOSITORY_URI:latest
```

### 3. Deploy with Fargate

```bash
# Register task definition
aws ecs register-task-definition --cli-input-json file://task-definition-fargate.json

# Create service
aws ecs create-service \
  --cluster $CLUSTER_NAME \
  --service-name $SERVICE_NAME \
  --task-definition rwwwrse-fargate:1 \
  --desired-count 2 \
  --launch-type FARGATE \
  --network-configuration "awsvpcConfiguration={subnets=[subnet-12345,subnet-67890],securityGroups=[sg-12345],assignPublicIp=ENABLED}"
```

## Fargate Deployment

### Task Definition

The [`task-definition-fargate.json`](task-definition-fargate.json) configures:
- **CPU and Memory**: Right-sized for rwwwrse workload
- **Networking**: awsvpc mode for Fargate
- **Logging**: CloudWatch Logs integration
- **Environment Variables**: Configuration management
- **Health Checks**: Container and load balancer health checks

### Service Configuration

```bash
# Create service with load balancer
aws ecs create-service \
  --cluster $CLUSTER_NAME \
  --service-name $SERVICE_NAME \
  --task-definition rwwwrse-fargate \
  --desired-count 2 \
  --launch-type FARGATE \
  --platform-version LATEST \
  --network-configuration file://network-config.json \
  --load-balancers file://load-balancer-config.json \
  --enable-execute-command
```

### Auto Scaling

```bash
# Register scalable target
aws application-autoscaling register-scalable-target \
  --service-namespace ecs \
  --resource-id service/$CLUSTER_NAME/$SERVICE_NAME \
  --scalable-dimension ecs:service:DesiredCount \
  --min-capacity 1 \
  --max-capacity 10

# Create scaling policy
aws application-autoscaling put-scaling-policy \
  --service-namespace ecs \
  --resource-id service/$CLUSTER_NAME/$SERVICE_NAME \
  --scalable-dimension ecs:service:DesiredCount \
  --policy-name rwwwrse-scale-out \
  --policy-type TargetTrackingScaling \
  --target-tracking-scaling-policy-configuration file://scaling-policy.json
```

## EC2 Launch Type

### Cluster Setup

```bash
# Create ECS cluster with EC2 capacity provider
aws ecs create-cluster \
  --cluster-name $CLUSTER_NAME-ec2 \
  --capacity-providers EC2 \
  --default-capacity-provider-strategy capacityProvider=EC2,weight=1

# Create Auto Scaling Group (using CloudFormation template)
aws cloudformation create-stack \
  --stack-name ecs-cluster-ec2 \
  --template-body file://ecs-cluster-cloudformation.yaml \
  --parameters ParameterKey=ClusterName,ParameterValue=$CLUSTER_NAME-ec2 \
  --capabilities CAPABILITY_IAM
```

### Task Definition for EC2

```bash
# Register EC2 task definition
aws ecs register-task-definition --cli-input-json file://task-definition-ec2.json

# Create service
aws ecs create-service \
  --cluster $CLUSTER_NAME-ec2 \
  --service-name $SERVICE_NAME-ec2 \
  --task-definition rwwwrse-ec2 \
  --desired-count 3 \
  --launch-type EC2 \
  --load-balancers file://load-balancer-config.json \
  --placement-strategy file://placement-strategy.json
```

## Load Balancer Configuration

### Application Load Balancer

```bash
# Create ALB
aws elbv2 create-load-balancer \
  --name rwwwrse-alb \
  --subnets subnet-12345 subnet-67890 \
  --security-groups sg-12345 \
  --scheme internet-facing \
  --type application \
  --ip-address-type ipv4

# Create target group
aws elbv2 create-target-group \
  --name rwwwrse-targets \
  --protocol HTTP \
  --port 8080 \
  --vpc-id vpc-12345 \
  --target-type ip \
  --health-check-path /health \
  --health-check-interval-seconds 30 \
  --health-check-timeout-seconds 5 \
  --healthy-threshold-count 2 \
  --unhealthy-threshold-count 3

# Create listener
aws elbv2 create-listener \
  --load-balancer-arn arn:aws:elasticloadbalancing:us-east-1:123456789012:loadbalancer/app/rwwwrse-alb/1234567890123456 \
  --protocol HTTPS \
  --port 443 \
  --certificates CertificateArn=arn:aws:acm:us-east-1:123456789012:certificate/12345678-1234-1234-1234-123456789012 \
  --default-actions Type=forward,TargetGroupArn=arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/rwwwrse-targets/1234567890123456
```

### SSL/TLS Configuration

```bash
# Request certificate from ACM
aws acm request-certificate \
  --domain-name api.example.com \
  --subject-alternative-names app.example.com web.example.com admin.example.com \
  --validation-method DNS \
  --tags Key=Name,Value=rwwwrse-cert

# Add certificate to listener
aws elbv2 modify-listener \
  --listener-arn arn:aws:elasticloadbalancing:us-east-1:123456789012:listener/app/rwwwrse-alb/1234567890123456/1234567890123456 \
  --certificates CertificateArn=arn:aws:acm:us-east-1:123456789012:certificate/12345678-1234-1234-1234-123456789012
```

## Configuration Management

### Systems Manager Parameter Store

```bash
# Store configuration parameters
aws ssm put-parameter \
  --name "/rwwwrse/api-target" \
  --value "http://api-service.local:3001" \
  --type "String" \
  --description "API backend target URL"

aws ssm put-parameter \
  --name "/rwwwrse/database-url" \
  --value "postgresql://username:password@rds-endpoint:5432/database" \
  --type "SecureString" \
  --description "Database connection URL"
```

### Secrets Manager

```bash
# Store sensitive data in Secrets Manager
aws secretsmanager create-secret \
  --name "rwwwrse/api-key" \
  --description "API key for backend services" \
  --secret-string "your-secret-api-key"

aws secretsmanager create-secret \
  --name "rwwwrse/jwt-secret" \
  --description "JWT signing secret" \
  --secret-string "your-jwt-secret"
```

## Service Discovery

### AWS Cloud Map

```bash
# Create namespace
aws servicediscovery create-private-dns-namespace \
  --name rwwwrse.local \
  --vpc vpc-12345 \
  --description "Private namespace for rwwwrse services"

# Create service
aws servicediscovery create-service \
  --name api-service \
  --namespace-id ns-12345 \
  --dns-config NamespaceId=ns-12345,DnsRecords=[{Type=A,TTL=60}] \
  --health-check-custom-config FailureThreshold=1

# Update ECS service to use service discovery
aws ecs update-service \
  --cluster $CLUSTER_NAME \
  --service $SERVICE_NAME \
  --service-registries registryArn=arn:aws:servicediscovery:us-east-1:123456789012:service/srv-12345
```

## Monitoring and Logging

### CloudWatch Integration

```bash
# Create log group
aws logs create-log-group \
  --log-group-name /ecs/rwwwrse \
  --retention-in-days 30

# Create custom metrics
aws cloudwatch put-metric-alarm \
  --alarm-name "rwwwrse-high-cpu" \
  --alarm-description "rwwwrse high CPU utilization" \
  --metric-name CPUUtilization \
  --namespace AWS/ECS \
  --statistic Average \
  --period 300 \
  --threshold 80 \
  --comparison-operator GreaterThanThreshold \
  --dimensions Name=ServiceName,Value=$SERVICE_NAME Name=ClusterName,Value=$CLUSTER_NAME \
  --evaluation-periods 2
```

### Container Insights

```bash
# Enable Container Insights for the cluster
aws ecs put-account-setting \
  --name containerInsights \
  --value enabled

# Update cluster to enable Container Insights
aws ecs put-cluster-capacity-providers \
  --cluster $CLUSTER_NAME \
  --capacity-providers FARGATE \
  --default-capacity-provider-strategy capacityProvider=FARGATE,weight=1 \
  --include-container-insights
```

## Security Best Practices

### IAM Roles and Policies

```bash
# Create task execution role
aws iam create-role \
  --role-name ecsTaskExecutionRole \
  --assume-role-policy-document file://ecs-task-execution-role-trust-policy.json

# Attach managed policy
aws iam attach-role-policy \
  --role-name ecsTaskExecutionRole \
  --policy-arn arn:aws:iam::aws:policy/service-role/AmazonECSTaskExecutionRolePolicy

# Create task role for application
aws iam create-role \
  --role-name rwwwrseTaskRole \
  --assume-role-policy-document file://ecs-task-role-trust-policy.json

# Create and attach custom policy
aws iam create-policy \
  --policy-name rwwwrseTaskPolicy \
  --policy-document file://rwwwrse-task-policy.json

aws iam attach-role-policy \
  --role-name rwwwrseTaskRole \
  --policy-arn arn:aws:iam::123456789012:policy/rwwwrseTaskPolicy
```

### Network Security

```bash
# Create security group for ECS tasks
aws ec2 create-security-group \
  --group-name rwwwrse-ecs-sg \
  --description "Security group for rwwwrse ECS tasks" \
  --vpc-id vpc-12345

# Allow inbound HTTP traffic from ALB
aws ec2 authorize-security-group-ingress \
  --group-id sg-12345 \
  --protocol tcp \
  --port 8080 \
  --source-group sg-67890

# Create security group for ALB
aws ec2 create-security-group \
  --group-name rwwwrse-alb-sg \
  --description "Security group for rwwwrse ALB" \
  --vpc-id vpc-12345

# Allow inbound HTTPS traffic
aws ec2 authorize-security-group-ingress \
  --group-id sg-67890 \
  --protocol tcp \
  --port 443 \
  --cidr 0.0.0.0/0
```

## Cost Optimization

### Fargate Spot

```bash
# Update service to use Fargate Spot
aws ecs update-service \
  --cluster $CLUSTER_NAME \
  --service $SERVICE_NAME \
  --capacity-provider-strategy capacityProvider=FARGATE_SPOT,weight=1,base=1

# Mixed capacity provider strategy
aws ecs update-service \
  --cluster $CLUSTER_NAME \
  --service $SERVICE_NAME \
  --capacity-provider-strategy \
    capacityProvider=FARGATE,weight=1,base=1 \
    capacityProvider=FARGATE_SPOT,weight=4
```

### Resource Right-Sizing

```bash
# Monitor resource utilization
aws cloudwatch get-metric-statistics \
  --namespace AWS/ECS \
  --metric-name CPUUtilization \
  --dimensions Name=ServiceName,Value=$SERVICE_NAME Name=ClusterName,Value=$CLUSTER_NAME \
  --start-time 2023-01-01T00:00:00Z \
  --end-time 2023-01-02T00:00:00Z \
  --period 3600 \
  --statistics Average Maximum

# Update task definition with optimized resources
aws ecs register-task-definition \
  --family rwwwrse-fargate \
  --task-role-arn arn:aws:iam::123456789012:role/rwwwrseTaskRole \
  --execution-role-arn arn:aws:iam::123456789012:role/ecsTaskExecutionRole \
  --network-mode awsvpc \
  --requires-compatibilities FARGATE \
  --cpu 256 \
  --memory 512 \
  --container-definitions file://optimized-container-definitions.json
```

## Deployment Strategies

### Blue-Green Deployment

```bash
# Create new task definition revision
aws ecs register-task-definition --cli-input-json file://task-definition-v2.json

# Update service with new task definition
aws ecs update-service \
  --cluster $CLUSTER_NAME \
  --service $SERVICE_NAME \
  --task-definition rwwwrse-fargate:2 \
  --deployment-configuration "maximumPercent=200,minimumHealthyPercent=50"
```

### Rolling Updates

```bash
# Configure rolling update strategy
aws ecs update-service \
  --cluster $CLUSTER_NAME \
  --service $SERVICE_NAME \
  --deployment-configuration "maximumPercent=200,minimumHealthyPercent=50" \
  --desired-count 4
```

## Troubleshooting

### Common Issues

1. **Task failing to start:**
```bash
# Check task definition
aws ecs describe-task-definition --task-definition rwwwrse-fargate

# Check service events
aws ecs describe-services --cluster $CLUSTER_NAME --services $SERVICE_NAME

# Check task logs
aws logs filter-log-events \
  --log-group-name /ecs/rwwwrse \
  --start-time 1640995200000
```

2. **Load balancer health check failures:**
```bash
# Check target group health
aws elbv2 describe-target-health \
  --target-group-arn arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/rwwwrse-targets/1234567890123456

# Update health check configuration
aws elbv2 modify-target-group \
  --target-group-arn arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/rwwwrse-targets/1234567890123456 \
  --health-check-path /health \
  --health-check-interval-seconds 30 \
  --health-check-timeout-seconds 5
```

3. **Permission issues:**
```bash
# Check task role permissions
aws iam simulate-principal-policy \
  --policy-source-arn arn:aws:iam::123456789012:role/rwwwrseTaskRole \
  --action-names ssm:GetParameter \
  --resource-arns arn:aws:ssm:us-east-1:123456789012:parameter/rwwwrse/*
```

## Automation Scripts

### Deployment Script

```bash
#!/bin/bash
set -e

# Deploy script for ECS
source config.env

echo "Building and pushing image..."
docker build -t $REPOSITORY_URI:$VERSION .
docker push $REPOSITORY_URI:$VERSION

echo "Updating task definition..."
sed "s/IMAGE_URI/$REPOSITORY_URI:$VERSION/g" task-definition-template.json > task-definition.json
aws ecs register-task-definition --cli-input-json file://task-definition.json

echo "Updating service..."
aws ecs update-service \
  --cluster $CLUSTER_NAME \
  --service $SERVICE_NAME \
  --task-definition rwwwrse-fargate:LATEST

echo "Deployment completed successfully"
```

## Next Steps

1. **Set up CI/CD pipeline** with AWS CodePipeline or GitHub Actions
2. **Implement monitoring and alerting** with CloudWatch and SNS
3. **Configure backup and disaster recovery** procedures
4. **Optimize costs** with Spot instances and resource right-sizing
5. **Implement security scanning** with Amazon Inspector

## Related Documentation

- [AWS ECS Documentation](https://docs.aws.amazon.com/ecs/)
- [AWS Fargate Documentation](https://docs.aws.amazon.com/AmazonECS/latest/developerguide/AWS_Fargate.html)
- [Application Load Balancer Documentation](https://docs.aws.amazon.com/elasticloadbalancing/latest/application/)
- [CI/CD Examples](../../cicd/) - Automated deployment pipelines
- [Docker Examples](../../docker-compose/) - Local development and testing