# SSL/TLS and Certificate Management Guide for rwwwrse

This guide provides comprehensive SSL/TLS certificate management strategies for rwwwrse across all deployment scenarios. It covers certificate acquisition, automation, renewal, and security best practices.

## Overview

SSL/TLS certificate management for rwwwrse involves:

- **Certificate acquisition** from various Certificate Authorities (CAs)
- **Automated renewal** to prevent service interruptions
- **Secure storage** and distribution of certificates
- **Platform-specific integration** for different deployment environments
- **Monitoring and alerting** for certificate expiration
- **Security best practices** for certificate lifecycle management

## Certificate Management Strategies

### Deployment-Specific Approaches

| Deployment Type | SSL Termination | Certificate Source | Renewal Method | Complexity |
|----------------|-----------------|-------------------|----------------|------------|
| Docker Compose | Load Balancer/Proxy | Let's Encrypt, Custom CA | Automated scripts | Medium |
| Kubernetes | Ingress Controller | cert-manager | Automated CRDs | Low |
| AWS ECS | Application Load Balancer | AWS Certificate Manager | Automatic | Low |
| Google Cloud Run | Built-in | Google-managed SSL | Automatic | Low |
| Azure Container Instances | Application Gateway | Azure Key Vault | Semi-automatic | Medium |
| Bare Metal | NGINX/Apache | Let's Encrypt, Custom CA | Certbot, scripts | High |
| Cloud-Specific | Platform Load Balancer | Platform-managed | Automatic | Low |

## Let's Encrypt Integration

### Certbot for Bare Metal and Docker

#### Installation and Setup

```bash
# Install Certbot
# Ubuntu/Debian
sudo apt install certbot python3-certbot-nginx python3-certbot-apache

# CentOS/RHEL
sudo yum install certbot python3-certbot-nginx python3-certbot-apache

# macOS
brew install certbot
```

#### Certificate Acquisition

```bash
# Standalone mode (requires port 80/443 to be free)
sudo certbot certonly --standalone \
  -d api.example.com \
  -d app.example.com \
  -d web.example.com \
  --email admin@example.com \
  --agree-tos \
  --non-interactive

# Webroot mode (for running web servers)
sudo certbot certonly --webroot \
  -w /var/www/html \
  -d api.example.com \
  -d app.example.com \
  --email admin@example.com \
  --agree-tos \
  --non-interactive

# DNS challenge (for wildcard certificates)
sudo certbot certonly --manual \
  --preferred-challenges dns \
  -d "*.example.com" \
  --email admin@example.com \
  --agree-tos
```

#### Automated Renewal

```bash
# Create renewal script
sudo tee /usr/local/bin/renew-certs.sh > /dev/null << 'EOF'
#!/bin/bash
set -e

# Renew certificates
/usr/bin/certbot renew --quiet

# Reload web server
if systemctl is-active --quiet nginx; then
    systemctl reload nginx
fi

if systemctl is-active --quiet apache2; then
    systemctl reload apache2
fi

# Restart rwwwrse if using built-in TLS
if systemctl is-active --quiet rwwwrse; then
    systemctl reload rwwwrse
fi

# Log renewal
echo "$(date): Certificate renewal completed" >> /var/log/cert-renewal.log
EOF

sudo chmod +x /usr/local/bin/renew-certs.sh

# Add to crontab for automatic renewal
sudo crontab -e
# Add: 0 12 * * * /usr/local/bin/renew-certs.sh
```

### Docker Compose with Let's Encrypt

#### Using Traefik with Automatic HTTPS

```yaml
# docker-compose.yml
version: '3.8'

services:
  traefik:
    image: traefik:v3.0
    command:
      - "--api.insecure=true"
      - "--providers.docker=true"
      - "--providers.docker.exposedbydefault=false"
      - "--entrypoints.web.address=:80"
      - "--entrypoints.websecure.address=:443"
      - "--certificatesresolvers.letsencrypt.acme.tlschallenge=true"
      - "--certificatesresolvers.letsencrypt.acme.email=admin@example.com"
      - "--certificatesresolvers.letsencrypt.acme.storage=/letsencrypt/acme.json"
    ports:
      - "80:80"
      - "443:443"
      - "8080:8080"
    volumes:
      - "/var/run/docker.sock:/var/run/docker.sock:ro"
      - "./letsencrypt:/letsencrypt"

  rwwwrse:
    image: rwwwrse:latest
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.rwwwrse.rule=Host(`api.example.com`,`app.example.com`)"
      - "traefik.http.routers.rwwwrse.entrypoints=websecure"
      - "traefik.http.routers.rwwwrse.tls.certresolver=letsencrypt"
      - "traefik.http.services.rwwwrse.loadbalancer.server.port=8080"
    environment:
      - RWWWRSE_ENABLE_TLS=false  # TLS handled by Traefik
```

#### Using NGINX with Certbot

```yaml
# docker-compose.yml
version: '3.8'

services:
  nginx:
    image: nginx:alpine
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf
      - ./ssl:/etc/ssl
      - letsencrypt:/etc/letsencrypt
      - letsencrypt-www:/var/www/letsencrypt
    depends_on:
      - rwwwrse

  rwwwrse:
    image: rwwwrse:latest
    environment:
      - RWWWRSE_ENABLE_TLS=false  # TLS handled by NGINX

  certbot:
    image: certbot/certbot
    volumes:
      - letsencrypt:/etc/letsencrypt
      - letsencrypt-www:/var/www/letsencrypt
    command: certonly --webroot --webroot-path=/var/www/letsencrypt --email admin@example.com --agree-tos --no-eff-email -d api.example.com -d app.example.com

volumes:
  letsencrypt:
  letsencrypt-www:
```

#### Certificate Renewal with Docker

```bash
#!/bin/bash
# docker-cert-renewal.sh

# Renew certificates
docker-compose run --rm certbot renew

# Reload NGINX
docker-compose exec nginx nginx -s reload

# Log renewal
echo "$(date): Docker certificate renewal completed" >> /var/log/docker-cert-renewal.log
```

## Kubernetes with cert-manager

### cert-manager Installation

```bash
# Install cert-manager
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.13.0/cert-manager.yaml

# Verify installation
kubectl get pods --namespace cert-manager
```

### ClusterIssuer Configuration

#### Let's Encrypt HTTP Challenge

```yaml
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: letsencrypt-prod
spec:
  acme:
    server: https://acme-v02.api.letsencrypt.org/directory
    email: admin@example.com
    privateKeySecretRef:
      name: letsencrypt-prod
    solvers:
    - http01:
        ingress:
          class: nginx
          podTemplate:
            spec:
              nodeSelector:
                "kubernetes.io/os": linux
```

#### Let's Encrypt DNS Challenge (for wildcard certificates)

```yaml
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: letsencrypt-dns
spec:
  acme:
    server: https://acme-v02.api.letsencrypt.org/directory
    email: admin@example.com
    privateKeySecretRef:
      name: letsencrypt-dns
    solvers:
    - dns01:
        cloudflare:
          email: admin@example.com
          apiTokenSecretRef:
            name: cloudflare-api-token
            key: api-token
```

### Certificate Automation

#### Ingress with Automatic Certificate

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: rwwwrse-ingress
  annotations:
    kubernetes.io/ingress.class: nginx
    cert-manager.io/cluster-issuer: letsencrypt-prod
    nginx.ingress.kubernetes.io/ssl-redirect: "true"
spec:
  tls:
  - hosts:
    - api.example.com
    - app.example.com
    - web.example.com
    secretName: rwwwrse-tls
  rules:
  - host: api.example.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: rwwwrse
            port:
              number: 80
```

#### Certificate Resource

```yaml
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: rwwwrse-cert
  namespace: rwwwrse
spec:
  secretName: rwwwrse-tls
  issuerRef:
    name: letsencrypt-prod
    kind: ClusterIssuer
  dnsNames:
  - api.example.com
  - app.example.com
  - web.example.com
  - admin.example.com
```

### Monitoring Certificate Status

```bash
# Check certificate status
kubectl get certificates -A

# Describe certificate for details
kubectl describe certificate rwwwrse-cert -n rwwwrse

# Check certificate expiry
kubectl get secret rwwwrse-tls -n rwwwrse -o yaml | \
  yq '.data."tls.crt"' | base64 -d | \
  openssl x509 -noout -dates
```

## Cloud Platform Certificate Management

### AWS Certificate Manager (ACM)

#### Certificate Request

```bash
# Request certificate
aws acm request-certificate \
  --domain-name example.com \
  --subject-alternative-names api.example.com app.example.com web.example.com \
  --validation-method DNS \
  --tags Key=Name,Value=rwwwrse-cert

# Get certificate ARN
CERT_ARN=$(aws acm list-certificates \
  --query 'CertificateSummaryList[?DomainName==`example.com`].CertificateArn' \
  --output text)

echo "Certificate ARN: $CERT_ARN"
```

#### DNS Validation

```bash
# Get DNS validation records
aws acm describe-certificate \
  --certificate-arn $CERT_ARN \
  --query 'Certificate.DomainValidationOptions' \
  --output table

# Validation records need to be added to DNS
# ACM will automatically validate and issue the certificate
```

#### Integration with Application Load Balancer

```bash
# Create ALB with SSL certificate
aws elbv2 create-load-balancer \
  --name rwwwrse-alb \
  --subnets subnet-12345 subnet-67890 \
  --security-groups sg-12345

# Create HTTPS listener
aws elbv2 create-listener \
  --load-balancer-arn $ALB_ARN \
  --protocol HTTPS \
  --port 443 \
  --certificates CertificateArn=$CERT_ARN \
  --default-actions Type=forward,TargetGroupArn=$TARGET_GROUP_ARN
```

### Google Cloud Managed SSL

#### Automatic Certificate for Cloud Run

```bash
# Deploy Cloud Run service (automatic HTTPS)
gcloud run deploy rwwwrse \
  --image gcr.io/project-id/rwwwrse:latest \
  --platform managed \
  --region us-central1 \
  --allow-unauthenticated

# Map custom domain (triggers automatic SSL)
gcloud run domain-mappings create \
  --service rwwwrse \
  --domain api.example.com \
  --region us-central1

# Check domain mapping status
gcloud run domain-mappings describe api.example.com \
  --region us-central1
```

#### Google-managed SSL for Load Balancer

```yaml
# google-managed-ssl.yaml
apiVersion: networking.gke.io/v1
kind: ManagedCertificate
metadata:
  name: rwwwrse-ssl-cert
spec:
  domains:
    - api.example.com
    - app.example.com
    - web.example.com
---
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: rwwwrse-ingress
  annotations:
    kubernetes.io/ingress.global-static-ip-name: rwwwrse-ip
    networking.gke.io/managed-certificates: rwwwrse-ssl-cert
    kubernetes.io/ingress.class: gce
spec:
  rules:
  - host: api.example.com
    http:
      paths:
      - path: /*
        pathType: ImplementationSpecific
        backend:
          service:
            name: rwwwrse
            port:
              number: 80
```

### Azure SSL Certificate Management

#### Azure Application Gateway with Key Vault

```bash
# Create Key Vault
az keyvault create \
  --resource-group rwwwrse-rg \
  --name rwwwrse-kv \
  --location eastus \
  --enable-soft-delete true

# Import certificate to Key Vault
az keyvault certificate import \
  --vault-name rwwwrse-kv \
  --name rwwwrse-cert \
  --file certificate.pfx \
  --password certificate-password

# Create Application Gateway with Key Vault certificate
az network application-gateway create \
  --resource-group rwwwrse-rg \
  --name rwwwrse-appgw \
  --location eastus \
  --capacity 2 \
  --sku Standard_v2 \
  --vnet-name rwwwrse-vnet \
  --subnet appgateway-subnet \
  --public-ip-address rwwwrse-pip \
  --cert-name rwwwrse-cert \
  --cert-password certificate-password
```

#### Azure Front Door with Managed Certificate

```bash
# Create Front Door profile
az afd profile create \
  --resource-group rwwwrse-rg \
  --profile-name rwwwrse-frontdoor \
  --sku Premium_AzureFrontDoor

# Create custom domain with managed certificate
az afd custom-domain create \
  --resource-group rwwwrse-rg \
  --profile-name rwwwrse-frontdoor \
  --custom-domain-name api-example-com \
  --host-name api.example.com \
  --certificate-type ManagedCertificate \
  --minimum-tls-version TLS12
```

## Custom Certificate Authority (CA)

### Internal PKI Setup

#### Root CA Creation

```bash
#!/bin/bash
# create-root-ca.sh

# Create CA directory structure
mkdir -p /etc/ssl/ca/{certs,crl,newcerts,private}
chmod 700 /etc/ssl/ca/private
echo 1000 > /etc/ssl/ca/serial
touch /etc/ssl/ca/index.txt

# Generate root CA private key
openssl genrsa -aes256 -out /etc/ssl/ca/private/ca.key.pem 4096
chmod 400 /etc/ssl/ca/private/ca.key.pem

# Create root CA certificate
openssl req -config /etc/ssl/ca/openssl.cnf \
  -key /etc/ssl/ca/private/ca.key.pem \
  -new -x509 -days 7300 -sha256 -extensions v3_ca \
  -out /etc/ssl/ca/certs/ca.cert.pem

chmod 444 /etc/ssl/ca/certs/ca.cert.pem
```

#### Certificate Signing

```bash
#!/bin/bash
# sign-certificate.sh

DOMAIN=$1
if [ -z "$DOMAIN" ]; then
    echo "Usage: $0 <domain>"
    exit 1
fi

# Generate private key
openssl genrsa -out /etc/ssl/certs/${DOMAIN}.key.pem 2048

# Create certificate signing request
openssl req -config /etc/ssl/ca/openssl.cnf \
  -key /etc/ssl/certs/${DOMAIN}.key.pem \
  -new -sha256 -out /etc/ssl/certs/${DOMAIN}.csr.pem \
  -subj "/C=US/ST=State/L=City/O=Organization/CN=${DOMAIN}"

# Sign certificate
openssl ca -config /etc/ssl/ca/openssl.cnf \
  -extensions server_cert -days 375 -notext -md sha256 \
  -in /etc/ssl/certs/${DOMAIN}.csr.pem \
  -out /etc/ssl/certs/${DOMAIN}.cert.pem

chmod 444 /etc/ssl/certs/${DOMAIN}.cert.pem
```

### Container-Based PKI

```yaml
# docker-compose-pki.yml
version: '3.8'

services:
  step-ca:
    image: smallstep/step-ca
    environment:
      - DOCKER_STEPCA_INIT_NAME=Internal CA
      - DOCKER_STEPCA_INIT_DNS_NAMES=ca.internal,localhost
      - DOCKER_STEPCA_INIT_REMOTE_MANAGEMENT=true
    volumes:
      - step-ca-data:/home/step
    ports:
      - "9000:9000"

  step-cli:
    image: smallstep/step-cli
    volumes:
      - step-ca-data:/home/step
    depends_on:
      - step-ca

volumes:
  step-ca-data:
```

## Certificate Monitoring and Alerting

### Certificate Expiry Monitoring

#### Prometheus Exporter

```yaml
# cert-exporter.yaml
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: cert-exporter
spec:
  selector:
    matchLabels:
      app: cert-exporter
  template:
    metadata:
      labels:
        app: cert-exporter
    spec:
      containers:
      - name: cert-exporter
        image: joenoordhuis/certificate-exporter:latest
        ports:
        - containerPort: 8080
        env:
        - name: CERT_PATHS
          value: "/etc/ssl/certs"
        volumeMounts:
        - name: ssl-certs
          mountPath: /etc/ssl/certs
          readOnly: true
      volumes:
      - name: ssl-certs
        hostPath:
          path: /etc/ssl/certs
```

#### Monitoring Script

```bash
#!/bin/bash
# check-cert-expiry.sh

DOMAINS=("api.example.com" "app.example.com" "web.example.com")
WARNING_DAYS=30
CRITICAL_DAYS=7

for domain in "${DOMAINS[@]}"; do
    # Get certificate expiry date
    expiry_date=$(echo | openssl s_client -servername $domain -connect $domain:443 2>/dev/null | \
                  openssl x509 -noout -dates | grep notAfter | cut -d= -f2)
    
    # Convert to epoch
    expiry_epoch=$(date -d "$expiry_date" +%s)
    current_epoch=$(date +%s)
    days_until_expiry=$(( (expiry_epoch - current_epoch) / 86400 ))
    
    if [ $days_until_expiry -le $CRITICAL_DAYS ]; then
        echo "CRITICAL: Certificate for $domain expires in $days_until_expiry days"
        exit 2
    elif [ $days_until_expiry -le $WARNING_DAYS ]; then
        echo "WARNING: Certificate for $domain expires in $days_until_expiry days"
        exit 1
    else
        echo "OK: Certificate for $domain expires in $days_until_expiry days"
    fi
done
```

### Alerting Rules

#### Prometheus Alerting Rules

```yaml
groups:
- name: ssl-certificates
  rules:
  - alert: CertificateExpiringSoon
    expr: probe_ssl_earliest_cert_expiry - time() < 86400 * 30
    for: 1h
    labels:
      severity: warning
    annotations:
      summary: "SSL certificate expires soon"
      description: "Certificate for {{ $labels.instance }} expires in {{ $value | humanizeDuration }}"

  - alert: CertificateExpiringSoonCritical
    expr: probe_ssl_earliest_cert_expiry - time() < 86400 * 7
    for: 1h
    labels:
      severity: critical
    annotations:
      summary: "SSL certificate expires very soon"
      description: "Certificate for {{ $labels.instance }} expires in {{ $value | humanizeDuration }}"
```

#### Nagios Check

```bash
#!/bin/bash
# check_ssl_cert.sh for Nagios

HOSTNAME=$1
PORT=${2:-443}
WARNING=${3:-30}
CRITICAL=${4:-7}

if [ -z "$HOSTNAME" ]; then
    echo "UNKNOWN - Hostname not specified"
    exit 3
fi

# Get certificate expiry
expiry_epoch=$(echo | openssl s_client -servername $HOSTNAME -connect $HOSTNAME:$PORT 2>/dev/null | \
               openssl x509 -noout -dates | grep notAfter | cut -d= -f2 | xargs -I {} date -d "{}" +%s)

current_epoch=$(date +%s)
days_until_expiry=$(( (expiry_epoch - current_epoch) / 86400 ))

if [ $days_until_expiry -le $CRITICAL ]; then
    echo "CRITICAL - Certificate expires in $days_until_expiry days"
    exit 2
elif [ $days_until_expiry -le $WARNING ]; then
    echo "WARNING - Certificate expires in $days_until_expiry days"
    exit 1
else
    echo "OK - Certificate expires in $days_until_expiry days"
    exit 0
fi
```

## Security Best Practices

### Certificate Storage Security

#### File System Permissions

```bash
# Secure certificate directories
sudo chmod 755 /etc/ssl/certs
sudo chmod 700 /etc/ssl/private
sudo chown root:root /etc/ssl/certs/*
sudo chown root:ssl-cert /etc/ssl/private/*
sudo chmod 644 /etc/ssl/certs/*
sudo chmod 640 /etc/ssl/private/*

# SELinux contexts (if applicable)
sudo restorecon -R /etc/ssl/
```

#### Kubernetes Secret Security

```yaml
# Encrypt secrets at rest
apiVersion: v1
kind: EncryptionConfiguration
resources:
- resources:
  - secrets
  providers:
  - aescbc:
      keys:
      - name: key1
        secret: <base64-encoded-secret>
  - identity: {}
```

#### Docker Secrets

```yaml
# docker-compose.yml with secrets
version: '3.8'

services:
  rwwwrse:
    image: rwwwrse:latest
    secrets:
      - ssl_cert
      - ssl_key
    environment:
      - RWWWRSE_TLS_CERT_FILE=/run/secrets/ssl_cert
      - RWWWRSE_TLS_KEY_FILE=/run/secrets/ssl_key

secrets:
  ssl_cert:
    file: ./ssl/cert.pem
  ssl_key:
    file: ./ssl/key.pem
```

### TLS Configuration Security

#### Strong Cipher Suites

```bash
# Environment variables for secure TLS
RWWWRSE_TLS_MIN_VERSION=1.2
RWWWRSE_TLS_CIPHER_SUITES=TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256
RWWWRSE_TLS_PREFER_SERVER_CIPHER_SUITES=true
RWWWRSE_TLS_CURVE_PREFERENCES=X25519,P-256,P-384
```

#### NGINX SSL Configuration

```nginx
# Strong SSL configuration for NGINX
ssl_protocols TLSv1.2 TLSv1.3;
ssl_ciphers ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256:ECDHE-ECDSA-AES256-GCM-SHA384:ECDHE-RSA-AES256-GCM-SHA384:ECDHE-ECDSA-CHACHA20-POLY1305:ECDHE-RSA-CHACHA20-POLY1305:DHE-RSA-AES128-GCM-SHA256:DHE-RSA-AES256-GCM-SHA384;
ssl_prefer_server_ciphers off;
ssl_session_cache shared:SSL:10m;
ssl_session_timeout 10m;
ssl_session_tickets off;
ssl_stapling on;
ssl_stapling_verify on;

# Security headers
add_header Strict-Transport-Security "max-age=31536000; includeSubDomains; preload" always;
add_header X-Frame-Options "SAMEORIGIN" always;
add_header X-Content-Type-Options "nosniff" always;
```

### Certificate Rotation

#### Automated Rotation Script

```bash
#!/bin/bash
# rotate-certificates.sh

set -euo pipefail

CERT_DIR="/etc/ssl/certs"
BACKUP_DIR="/var/backups/ssl/$(date +%Y%m%d_%H%M%S)"
SERVICE_NAME="rwwwrse"

# Create backup directory
mkdir -p "$BACKUP_DIR"

# Backup current certificates
cp -r "$CERT_DIR" "$BACKUP_DIR/"

# Function to rotate certificate
rotate_cert() {
    local domain=$1
    local cert_file="${CERT_DIR}/${domain}.cert.pem"
    local key_file="${CERT_DIR}/${domain}.key.pem"
    
    echo "Rotating certificate for $domain"
    
    # Check if certificate expires soon
    if ! openssl x509 -checkend $((86400 * 30)) -noout -in "$cert_file" 2>/dev/null; then
        echo "Certificate for $domain expires soon, requesting new certificate"
        
        # Request new certificate (using your preferred method)
        request_new_certificate "$domain"
        
        # Validate new certificate
        if openssl x509 -noout -in "$cert_file" 2>/dev/null; then
            echo "New certificate for $domain is valid"
            
            # Reload service
            systemctl reload "$SERVICE_NAME"
            
            echo "Certificate rotation completed for $domain"
        else
            echo "New certificate for $domain is invalid, restoring backup"
            cp "${BACKUP_DIR}/certs/${domain}.cert.pem" "$cert_file"
            exit 1
        fi
    else
        echo "Certificate for $domain is still valid"
    fi
}

# Rotate certificates for all domains
for domain in api.example.com app.example.com web.example.com; do
    rotate_cert "$domain"
done

echo "Certificate rotation process completed"
```

#### Kubernetes Certificate Rotation

```yaml
# Certificate rotation job
apiVersion: batch/v1
kind: CronJob
metadata:
  name: cert-rotation
spec:
  schedule: "0 2 * * 0"  # Weekly on Sunday at 2 AM
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: cert-rotator
            image: cert-rotator:latest
            command:
            - /bin/sh
            - -c
            - |
              # Check certificate expiry
              for cert in /etc/ssl/certs/*.pem; do
                if ! openssl x509 -checkend $((86400 * 30)) -noout -in "$cert"; then
                  echo "Certificate $cert expires soon, triggering renewal"
                  # Trigger cert-manager renewal
                  kubectl annotate certificate rwwwrse-cert cert-manager.io/force-renew="$(date +%s)"
                fi
              done
            volumeMounts:
            - name: ssl-certs
              mountPath: /etc/ssl/certs
          volumes:
          - name: ssl-certs
            secret:
              secretName: rwwwrse-tls
          restartPolicy: OnFailure
```

## Troubleshooting SSL/TLS Issues

### Common Certificate Problems

#### 1. Certificate Validation Errors

```bash
# Test certificate chain
openssl s_client -connect api.example.com:443 -showcerts

# Verify certificate against CA
openssl verify -CAfile ca-bundle.crt certificate.crt

# Check certificate details
openssl x509 -in certificate.crt -text -noout

# Test specific protocol versions
openssl s_client -connect api.example.com:443 -tls1_2
openssl s_client -connect api.example.com:443 -tls1_3
```

#### 2. Certificate Expiry Issues

```bash
# Check certificate expiry
openssl x509 -in certificate.crt -noout -dates

# Check remote certificate expiry
echo | openssl s_client -servername api.example.com -connect api.example.com:443 2>/dev/null | \
  openssl x509 -noout -dates

# Monitor certificate with timeout
timeout 10 openssl s_client -connect api.example.com:443 2>/dev/null | \
  openssl x509 -noout -subject -dates
```

#### 3. Mixed Certificate Chains

```bash
# Download complete certificate chain
echo | openssl s_client -showcerts -servername api.example.com -connect api.example.com:443 2>/dev/null | \
  awk '/-----BEGIN CERTIFICATE-----/,/-----END CERTIFICATE-----/' > fullchain.pem

# Verify chain order
openssl crl2pkcs7 -nocrl -certfile fullchain.pem | \
  openssl pkcs7 -print_certs -text -noout
```

### Platform-Specific Troubleshooting

#### Docker Compose Issues

```bash
# Check container certificate mounts
docker exec rwwwrse ls -la /etc/ssl/certs/

# Verify certificate in container
docker exec rwwwrse openssl x509 -in /etc/ssl/certs/cert.pem -text -noout

# Check container environment
docker exec rwwwrse env | grep TLS
```

#### Kubernetes Issues

```bash
# Check certificate secret
kubectl get secret rwwwrse-tls -o yaml

# Verify certificate in secret
kubectl get secret rwwwrse-tls -o jsonpath='{.data.tls\.crt}' | \
  base64 -d | openssl x509 -text -noout

# Check cert-manager logs
kubectl logs -n cert-manager deployment/cert-manager

# Check certificate status
kubectl describe certificate rwwwrse-cert
kubectl describe certificaterequest
```

#### Cloud Platform Issues

```bash
# AWS ACM
aws acm describe-certificate --certificate-arn $CERT_ARN

# Google Cloud
gcloud compute ssl-certificates describe certificate-name

# Azure
az network application-gateway ssl-cert show \
  --resource-group rg-name \
  --gateway-name gateway-name \
  --name cert-name
```

### SSL/TLS Testing Tools

#### Online Tools

- **SSL Labs SSL Test**: <https://www.ssllabs.com/ssltest/>
- **SSL Checker**: <https://www.sslshopper.com/ssl-checker.html>
- **DigiCert SSL Installation Checker**: <https://www.digicert.com/help/>

#### Command Line Tools

```bash
# testssl.sh - comprehensive SSL/TLS tester
git clone https://github.com/drwetter/testssl.sh.git
cd testssl.sh
./testssl.sh api.example.com

# nmap SSL enumeration
nmap --script ssl-enum-ciphers -p 443 api.example.com

# sslscan
sslscan api.example.com

# sslyze
sslyze api.example.com
```

## Performance Optimization

### Certificate Performance

#### OCSP Stapling

```nginx
# NGINX OCSP stapling
ssl_stapling on;
ssl_stapling_verify on;
ssl_trusted_certificate /etc/ssl/certs/ca-bundle.crt;
resolver 8.8.8.8 8.8.4.4 valid=300s;
resolver_timeout 5s;
```

#### Session Resumption

```nginx
# SSL session caching
ssl_session_cache shared:SSL:10m;
ssl_session_timeout 10m;
ssl_session_tickets off;  # Disable for security
```

#### Certificate Optimization

```bash
# Use ECDSA certificates for better performance
openssl ecparam -genkey -name prime256v1 -out private-key.pem
openssl req -new -x509 -key private-key.pem -out certificate.pem -days 365

# Optimize certificate chain order
cat certificate.pem intermediate.pem root.pem > fullchain.pem
```

## Compliance and Auditing

### Certificate Compliance

```bash
# PCI DSS compliance check
./testssl.sh --severity HIGH api.example.com

# HIPAA compliance verification
nmap --script ssl-cert,ssl-enum-ciphers api.example.com

# SOC 2 certificate validation
openssl s_client -connect api.example.com:443 -verify_return_error
```

### Audit Logging

```yaml
# Kubernetes audit policy for certificates
apiVersion: audit.k8s.io/v1
kind: Policy
rules:
- level: Metadata
  resources:
  - group: ""
    resources: ["secrets"]
  - group: "cert-manager.io"
    resources: ["certificates", "certificaterequests"]
  namespaces: ["rwwwrse"]
```

## Related Documentation

- [Configuration Guide](CONFIGURATION.md) - Environment-specific configuration
- [Operations Guide](OPERATIONS.md) - Monitoring and troubleshooting
- [Deployment Guide](DEPLOYMENT.md) - Platform-specific deployments
- [Development Guide](DEVELOPMENT.md) - Local development setup
- [Kubernetes Examples](../examples/kubernetes/) - Container orchestration
- [Cloud-Specific Examples](../examples/cloud-specific/) - Cloud platform deployments
- [Docker Compose Examples](../examples/docker-compose/) - Container deployments
- [Bare-Metal Examples](../examples/bare-metal/) - Traditional server deployment
