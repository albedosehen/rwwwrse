# Migration Guide for rwwwrse

This guide provides comprehensive migration strategies for upgrading to rwwwrse from legacy reverse proxy implementations and between different versions of rwwwrse itself. It covers scenario-specific upgrade paths, breaking changes, and rollback procedures.

## Overview

Migration to rwwwrse involves several considerations:

- **Source system assessment** - Understanding current proxy configuration and dependencies
- **Target deployment strategy** - Choosing the appropriate rwwwrse deployment model
- **Migration approach** - Blue-green, canary, or rolling upgrade strategies
- **Configuration conversion** - Translating existing configurations to rwwwrse format
- **Certificate migration** - Moving SSL/TLS certificates and automation
- **Monitoring transition** - Ensuring observability throughout the migration
- **Rollback preparation** - Planning for potential migration issues

## Migration Assessment

### Pre-Migration Checklist

```bash
#!/bin/bash
# pre-migration-assessment.sh

echo "=== rwwwrse Migration Assessment ==="

# Current system information
echo "1. Current Reverse Proxy System:"
if command -v nginx >/dev/null 2>&1; then
    echo "   - NGINX version: $(nginx -v 2>&1 | cut -d' ' -f3)"
    echo "   - Config location: /etc/nginx/"
fi

if command -v haproxy >/dev/null 2>&1; then
    echo "   - HAProxy version: $(haproxy -v | head -1)"
    echo "   - Config location: /etc/haproxy/"
fi

if command -v httpd >/dev/null 2>&1; then
    echo "   - Apache version: $(httpd -v | head -1)"
    echo "   - Config location: /etc/httpd/ or /etc/apache2/"
fi

# Traffic analysis
echo "2. Traffic Analysis:"
echo "   - Current connections: $(netstat -an | grep :80 | wc -l)"
echo "   - SSL connections: $(netstat -an | grep :443 | wc -l)"
echo "   - Backend services: $(netstat -an | grep ESTABLISHED | wc -l)"

# Certificate inventory
echo "3. SSL Certificate Inventory:"
find /etc/ssl /etc/pki /etc/letsencrypt -name "*.crt" -o -name "*.pem" 2>/dev/null | head -10

# Configuration complexity
echo "4. Configuration Complexity:"
if [ -f /etc/nginx/nginx.conf ]; then
    echo "   - NGINX config lines: $(wc -l /etc/nginx/nginx.conf | cut -d' ' -f1)"
    echo "   - Virtual hosts: $(find /etc/nginx -name "*.conf" | wc -l)"
fi

echo "5. Dependencies:"
systemctl list-dependencies --reverse nginx 2>/dev/null || echo "   - No systemd dependencies found"

echo "Assessment complete. Review output before proceeding with migration."
```

### Migration Compatibility Matrix

| Source System | Target Deployment | Complexity | Migration Time | Rollback Risk |
|---------------|-------------------|------------|----------------|---------------|
| NGINX → Docker Compose | Medium | 2-4 hours | Low | Low |
| NGINX → Kubernetes | High | 4-8 hours | Medium | Medium |
| HAProxy → Cloud Run | Medium | 3-6 hours | Low | Low |
| Apache → ECS | High | 6-12 hours | Medium | Medium |
| Traefik → rwwwrse | Low | 1-2 hours | Very Low | Very Low |
| Legacy rwwwrse → Latest | Low | 30-60 minutes | Very Low | Very Low |

## Migration Strategies

### Blue-Green Deployment Migration

#### Strategy Overview

Run both old and new systems simultaneously, then switch traffic.

#### Preparation

```yaml
# docker-compose-blue-green.yml
version: '3.8'

services:
  # Blue environment (current NGINX)
  nginx-blue:
    image: nginx:alpine
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf
      - ./ssl:/etc/ssl
    ports:
      - "80:80"
      - "443:443"
    networks:
      - blue-network

  # Green environment (new rwwwrse)
  rwwwrse-green:
    image: rwwwrse:latest
    environment:
      - RWWWRSE_BACKENDS=http://backend1:8080,http://backend2:8080
      - RWWWRSE_ENABLE_TLS=true
      - RWWWRSE_TLS_CERT_FILE=/etc/ssl/cert.pem
      - RWWWRSE_TLS_KEY_FILE=/etc/ssl/key.pem
    volumes:
      - ./ssl:/etc/ssl
    ports:
      - "8080:8080"
      - "8443:8443"
    networks:
      - green-network

  # Load balancer for traffic switching
  traefik:
    image: traefik:v3.0
    command:
      - "--providers.docker=true"
      - "--entrypoints.web.address=:80"
      - "--entrypoints.websecure.address=:443"
    volumes:
      - "/var/run/docker.sock:/var/run/docker.sock:ro"
    ports:
      - "80:80"
      - "443:443"
    labels:
      - "traefik.http.routers.api.rule=Host(`proxy.example.com`) && PathPrefix(`/api`)"

networks:
  blue-network:
  green-network:
```

#### Traffic Switch Script

```bash
#!/bin/bash
# blue-green-switch.sh

ENVIRONMENT=${1:-green}
HEALTH_CHECK_URL="http://localhost:8080/health"

case $ENVIRONMENT in
  green)
    echo "Switching to GREEN environment (rwwwrse)"
    
    # Health check
    if curl -sf $HEALTH_CHECK_URL >/dev/null; then
      echo "Health check passed, switching traffic"
      
      # Update load balancer to point to rwwwrse
      docker-compose exec traefik \
        curl -X PUT -H "Content-Type: application/json" \
        -d '{"loadBalancer":{"servers":[{"url":"http://rwwwrse-green:8080"}]}}' \
        http://localhost:8080/api/http/services/app/loadBalancer
      
      echo "Traffic switched to GREEN (rwwwrse)"
    else
      echo "Health check failed, staying on BLUE (nginx)"
      exit 1
    fi
    ;;
    
  blue)
    echo "Rolling back to BLUE environment (nginx)"
    
    # Switch back to nginx
    docker-compose exec traefik \
      curl -X PUT -H "Content-Type: application/json" \
      -d '{"loadBalancer":{"servers":[{"url":"http://nginx-blue:80"}]}}' \
      http://localhost:8080/api/http/services/app/loadBalancer
    
    echo "Traffic rolled back to BLUE (nginx)"
    ;;
    
  *)
    echo "Usage: $0 {green|blue}"
    exit 1
    ;;
esac
```

### Canary Migration

#### Gradual Traffic Shift

```yaml
# kubernetes-canary-migration.yaml
apiVersion: argoproj.io/v1alpha1
kind: Rollout
metadata:
  name: rwwwrse-migration
spec:
  replicas: 5
  strategy:
    canary:
      steps:
      - setWeight: 10
      - pause: {duration: 5m}
      - setWeight: 25
      - pause: {duration: 10m}
      - setWeight: 50
      - pause: {duration: 15m}
      - setWeight: 75
      - pause: {duration: 10m}
      canaryService: rwwwrse-canary
      stableService: nginx-stable
      trafficRouting:
        nginx:
          stableIngress: nginx-stable-ingress
          annotationPrefix: nginx.ingress.kubernetes.io
  selector:
    matchLabels:
      app: reverse-proxy
  template:
    metadata:
      labels:
        app: reverse-proxy
    spec:
      containers:
      - name: rwwwrse
        image: rwwwrse:latest
        ports:
        - containerPort: 8080
```

### Rolling Update Migration

#### Kubernetes Rolling Update

```bash
#!/bin/bash
# kubernetes-rolling-migration.sh

set -e

echo "Starting Kubernetes rolling migration to rwwwrse"

# Backup current configuration
kubectl get deployment nginx-proxy -o yaml > nginx-backup.yaml

# Apply new rwwwrse deployment
kubectl apply -f - <<EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: rwwwrse-proxy
spec:
  replicas: 3
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxUnavailable: 1
      maxSurge: 1
  selector:
    matchLabels:
      app: rwwwrse-proxy
  template:
    metadata:
      labels:
        app: rwwwrse-proxy
    spec:
      containers:
      - name: rwwwrse
        image: rwwwrse:latest
        ports:
        - containerPort: 8080
        env:
        - name: RWWWRSE_BACKENDS
          value: "http://backend-service:8080"
        readinessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 10
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 15
          periodSeconds: 20
EOF

# Wait for rollout
kubectl rollout status deployment/rwwwrse-proxy

# Switch service to new deployment
kubectl patch service proxy-service -p '{"spec":{"selector":{"app":"rwwwrse-proxy"}}}'

echo "Rolling migration completed successfully"
```

## Configuration Migration

### NGINX to rwwwrse

#### Configuration Converter Script

```bash
#!/bin/bash
# nginx-to-rwwwrse-converter.sh

NGINX_CONF=${1:-/etc/nginx/nginx.conf}
OUTPUT_FILE=${2:-rwwwrse-config.env}

echo "Converting NGINX configuration to rwwwrse environment variables"

# Extract upstream backends
echo "# Extracted backends from NGINX upstream configuration" > $OUTPUT_FILE
grep -A 10 "upstream" $NGINX_CONF | grep "server" | sed 's/.*server \([^;]*\);.*/\1/' | \
  tr '\n' ',' | sed 's/,$//' | sed 's/^/RWWWRSE_BACKENDS=/' >> $OUTPUT_FILE

# Extract SSL configuration
if grep -q "ssl_certificate" $NGINX_CONF; then
  CERT_PATH=$(grep "ssl_certificate[^_]" $NGINX_CONF | head -1 | awk '{print $2}' | sed 's/;//')
  KEY_PATH=$(grep "ssl_certificate_key" $NGINX_CONF | head -1 | awk '{print $2}' | sed 's/;//')
  
  echo "" >> $OUTPUT_FILE
  echo "# SSL Configuration" >> $OUTPUT_FILE
  echo "RWWWRSE_ENABLE_TLS=true" >> $OUTPUT_FILE
  echo "RWWWRSE_TLS_CERT_FILE=$CERT_PATH" >> $OUTPUT_FILE
  echo "RWWWRSE_TLS_KEY_FILE=$KEY_PATH" >> $OUTPUT_FILE
fi

# Extract rate limiting
if grep -q "limit_req" $NGINX_CONF; then
  echo "" >> $OUTPUT_FILE
  echo "# Rate Limiting" >> $OUTPUT_FILE
  echo "RWWWRSE_ENABLE_RATE_LIMITING=true" >> $OUTPUT_FILE
  
  RATE=$(grep "limit_req_zone" $NGINX_CONF | sed -n 's/.*rate=\([^[:space:]]*\).*/\1/p')
  if [ ! -z "$RATE" ]; then
    echo "RWWWRSE_RATE_LIMIT=$RATE" >> $OUTPUT_FILE
  fi
fi

# Extract health check paths
if grep -q "location.*health" $NGINX_CONF; then
  HEALTH_PATH=$(grep "location.*health" $NGINX_CONF | sed -n 's/.*location \([^[:space:]]*\).*/\1/p')
  echo "" >> $OUTPUT_FILE
  echo "# Health Check" >> $OUTPUT_FILE
  echo "RWWWRSE_HEALTH_CHECK_PATH=$HEALTH_PATH" >> $OUTPUT_FILE
fi

echo "Configuration converted to $OUTPUT_FILE"
echo "Review and adjust the generated configuration before use."
```

#### NGINX Virtual Host Conversion

```bash
#!/bin/bash
# convert-nginx-vhosts.sh

VHOSTS_DIR="/etc/nginx/sites-enabled"
OUTPUT_DIR="./rwwwrse-configs"

mkdir -p $OUTPUT_DIR

for vhost in $VHOSTS_DIR/*.conf; do
  DOMAIN=$(basename "$vhost" .conf)
  CONFIG_FILE="$OUTPUT_DIR/$DOMAIN.env"
  
  echo "Converting $vhost to $CONFIG_FILE"
  
  # Extract server_name
  SERVER_NAME=$(grep "server_name" "$vhost" | head -1 | sed 's/.*server_name \([^;]*\);.*/\1/')
  echo "RWWWRSE_HOST_FILTER=$SERVER_NAME" > $CONFIG_FILE
  
  # Extract proxy_pass targets
  BACKENDS=$(grep "proxy_pass" "$vhost" | sed 's/.*proxy_pass \([^;]*\);.*/\1/' | tr '\n' ',' | sed 's/,$//')
  if [ ! -z "$BACKENDS" ]; then
    echo "RWWWRSE_BACKENDS=$BACKENDS" >> $CONFIG_FILE
  fi
  
  # Extract SSL configuration
  if grep -q "ssl_certificate" "$vhost"; then
    echo "RWWWRSE_ENABLE_TLS=true" >> $CONFIG_FILE
    CERT_PATH=$(grep "ssl_certificate[^_]" "$vhost" | head -1 | awk '{print $2}' | sed 's/;//')
    KEY_PATH=$(grep "ssl_certificate_key" "$vhost" | head -1 | awk '{print $2}' | sed 's/;//')
    echo "RWWWRSE_TLS_CERT_FILE=$CERT_PATH" >> $CONFIG_FILE
    echo "RWWWRSE_TLS_KEY_FILE=$KEY_PATH" >> $CONFIG_FILE
  fi
  
  echo "Converted $DOMAIN configuration"
done

echo "All virtual hosts converted to $OUTPUT_DIR/"
```

### HAProxy to rwwwrse

#### HAProxy Configuration Parser

```python
#!/usr/bin/env python3
# haproxy-to-rwwwrse.py

import re
import sys
import os

def parse_haproxy_config(config_file):
    """Parse HAProxy configuration and extract relevant settings."""
    
    with open(config_file, 'r') as f:
        content = f.read()
    
    # Extract backends
    backend_pattern = r'backend\s+(\S+).*?(?=\n(?:backend|frontend|listen|\s*$))'
    backends = re.findall(backend_pattern, content, re.DOTALL | re.MULTILINE)
    
    backend_servers = {}
    for backend_block in re.finditer(backend_pattern, content, re.DOTALL | re.MULTILINE):
        backend_name = backend_block.group(1)
        backend_content = backend_block.group(0)
        
        # Extract server lines
        server_pattern = r'server\s+\S+\s+([^:\s]+):(\d+)'
        servers = re.findall(server_pattern, backend_content)
        
        if servers:
            backend_servers[backend_name] = [f"http://{host}:{port}" for host, port in servers]
    
    # Extract frontends
    frontend_pattern = r'frontend\s+(\S+).*?(?=\n(?:backend|frontend|listen|\s*$))'
    frontends = {}
    
    for frontend_block in re.finditer(frontend_pattern, content, re.DOTALL | re.MULTILINE):
        frontend_name = frontend_block.group(1)
        frontend_content = frontend_block.group(0)
        
        # Extract bind addresses
        bind_pattern = r'bind\s+([^:\s]+):(\d+)(?:\s+ssl)?'
        binds = re.findall(bind_pattern, frontend_content)
        
        # Extract ACLs and use_backend rules
        acl_pattern = r'acl\s+(\S+)\s+(.+)'
        use_backend_pattern = r'use_backend\s+(\S+)\s+if\s+(.+)'
        
        acls = re.findall(acl_pattern, frontend_content)
        use_backends = re.findall(use_backend_pattern, frontend_content)
        
        frontends[frontend_name] = {
            'binds': binds,
            'acls': acls,
            'use_backends': use_backends
        }
    
    return backend_servers, frontends

def generate_rwwwrse_config(backend_servers, frontends, output_dir):
    """Generate rwwwrse configuration files."""
    
    os.makedirs(output_dir, exist_ok=True)
    
    for frontend_name, frontend_config in frontends.items():
        config_file = os.path.join(output_dir, f"{frontend_name}.env")
        
        with open(config_file, 'w') as f:
            f.write(f"# Converted from HAProxy frontend: {frontend_name}\n\n")
            
            # Determine if SSL is enabled
            ssl_enabled = any('ssl' in str(bind) for bind in frontend_config['binds'])
            if ssl_enabled:
                f.write("RWWWRSE_ENABLE_TLS=true\n")
                f.write("RWWWRSE_TLS_CERT_FILE=/etc/ssl/certs/server.crt\n")
                f.write("RWWWRSE_TLS_KEY_FILE=/etc/ssl/private/server.key\n\n")
            
            # Handle backend mapping
            if frontend_config['use_backends']:
                for backend_name, condition in frontend_config['use_backends']:
                    if backend_name in backend_servers:
                        servers = ','.join(backend_servers[backend_name])
                        f.write(f"# Backend: {backend_name} (condition: {condition})\n")
                        f.write(f"RWWWRSE_BACKENDS={servers}\n\n")
            
            # Add host filtering based on ACLs
            host_acls = [acl for acl in frontend_config['acls'] if 'hdr(host)' in acl[1]]
            if host_acls:
                hosts = []
                for acl_name, acl_condition in host_acls:
                    # Extract hostname from ACL condition
                    host_match = re.search(r'hdr\(host\)\s+-i\s+(\S+)', acl_condition)
                    if host_match:
                        hosts.append(host_match.group(1))
                
                if hosts:
                    f.write(f"RWWWRSE_HOST_FILTER={','.join(hosts)}\n")

if __name__ == "__main__":
    if len(sys.argv) != 3:
        print("Usage: python3 haproxy-to-rwwwrse.py <haproxy.cfg> <output_dir>")
        sys.exit(1)
    
    config_file = sys.argv[1]
    output_dir = sys.argv[2]
    
    try:
        backend_servers, frontends = parse_haproxy_config(config_file)
        generate_rwwwrse_config(backend_servers, frontends, output_dir)
        print(f"HAProxy configuration converted successfully to {output_dir}/")
    except Exception as e:
        print(f"Error converting configuration: {e}")
        sys.exit(1)
```

### Apache to rwwwrse

#### Apache Virtual Host Converter

```bash
#!/bin/bash
# apache-to-rwwwrse.sh

APACHE_SITES_DIR="/etc/apache2/sites-enabled"
OUTPUT_DIR="./rwwwrse-apache-configs"

mkdir -p $OUTPUT_DIR

for site in $APACHE_SITES_DIR/*.conf; do
  SITE_NAME=$(basename "$site" .conf)
  CONFIG_FILE="$OUTPUT_DIR/$SITE_NAME.env"
  
  echo "Converting Apache site $site to $CONFIG_FILE"
  
  # Extract ServerName
  SERVER_NAME=$(grep "ServerName" "$site" | head -1 | awk '{print $2}')
  if [ ! -z "$SERVER_NAME" ]; then
    echo "RWWWRSE_HOST_FILTER=$SERVER_NAME" > $CONFIG_FILE
  fi
  
  # Extract ProxyPass directives
  PROXY_BACKENDS=$(grep "ProxyPass" "$site" | grep -v "ProxyPassReverse" | \
    sed 's/.*ProxyPass \S\+ \(http[^[:space:]]*\).*/\1/' | tr '\n' ',' | sed 's/,$//')
  
  if [ ! -z "$PROXY_BACKENDS" ]; then
    echo "RWWWRSE_BACKENDS=$PROXY_BACKENDS" >> $CONFIG_FILE
  fi
  
  # Extract SSL configuration
  if grep -q "SSLEngine on" "$site"; then
    echo "RWWWRSE_ENABLE_TLS=true" >> $CONFIG_FILE
    
    CERT_FILE=$(grep "SSLCertificateFile" "$site" | head -1 | awk '{print $2}')
    KEY_FILE=$(grep "SSLCertificateKeyFile" "$site" | head -1 | awk '{print $2}')
    
    if [ ! -z "$CERT_FILE" ]; then
      echo "RWWWRSE_TLS_CERT_FILE=$CERT_FILE" >> $CONFIG_FILE
    fi
    if [ ! -z "$KEY_FILE" ]; then
      echo "RWWWRSE_TLS_KEY_FILE=$KEY_FILE" >> $CONFIG_FILE
    fi
  fi
  
  echo "Converted $SITE_NAME"
done

echo "Apache configuration conversion completed in $OUTPUT_DIR/"
```

## SSL/TLS Certificate Migration

### Certificate Migration Script

```bash
#!/bin/bash
# migrate-certificates.sh

SOURCE_CERT_DIR="/etc/ssl"
TARGET_CERT_DIR="/opt/rwwwrse/ssl"
BACKUP_DIR="/var/backups/ssl-migration-$(date +%Y%m%d_%H%M%S)"

set -e

echo "Starting SSL certificate migration"

# Create backup
mkdir -p "$BACKUP_DIR"
cp -r "$SOURCE_CERT_DIR" "$BACKUP_DIR/"
echo "Certificates backed up to $BACKUP_DIR"

# Create target directory
mkdir -p "$TARGET_CERT_DIR"

# Migrate certificates
migrate_certificate() {
  local cert_name=$1
  local source_cert="$SOURCE_CERT_DIR/certs/${cert_name}.crt"
  local source_key="$SOURCE_CERT_DIR/private/${cert_name}.key"
  local target_cert="$TARGET_CERT_DIR/${cert_name}.crt"
  local target_key="$TARGET_CERT_DIR/${cert_name}.key"
  
  if [ -f "$source_cert" ] && [ -f "$source_key" ]; then
    echo "Migrating certificate: $cert_name"
    
    # Validate certificate before migration
    if openssl x509 -in "$source_cert" -noout -checkend 86400; then
      cp "$source_cert" "$target_cert"
      cp "$source_key" "$target_key"
      
      # Set proper permissions
      chmod 644 "$target_cert"
      chmod 600 "$target_key"
      
      # Verify certificate chain
      if openssl verify -CAfile /etc/ssl/certs/ca-certificates.crt "$target_cert" >/dev/null 2>&1; then
        echo "  ✓ Certificate $cert_name migrated and verified"
      else
        echo "  ⚠ Certificate $cert_name migrated but verification failed"
      fi
    else
      echo "  ✗ Certificate $cert_name is expired or invalid, skipping"
    fi
  else
    echo "  ✗ Certificate files not found for $cert_name"
  fi
}

# Detect and migrate certificates
for cert_file in "$SOURCE_CERT_DIR/certs"/*.crt; do
  if [ -f "$cert_file" ]; then
    cert_name=$(basename "$cert_file" .crt)
    migrate_certificate "$cert_name"
  fi
done

# Migrate Let's Encrypt certificates
if [ -d "/etc/letsencrypt/live" ]; then
  echo "Migrating Let's Encrypt certificates"
  
  for domain_dir in /etc/letsencrypt/live/*/; do
    domain=$(basename "$domain_dir")
    echo "Migrating Let's Encrypt certificate for $domain"
    
    if [ -f "$domain_dir/fullchain.pem" ] && [ -f "$domain_dir/privkey.pem" ]; then
      cp "$domain_dir/fullchain.pem" "$TARGET_CERT_DIR/${domain}.crt"
      cp "$domain_dir/privkey.pem" "$TARGET_CERT_DIR/${domain}.key"
      
      chmod 644 "$TARGET_CERT_DIR/${domain}.crt"
      chmod 600 "$TARGET_CERT_DIR/${domain}.key"
      
      echo "  ✓ Let's Encrypt certificate for $domain migrated"
    fi
  done
fi

echo "Certificate migration completed"
echo "Target directory: $TARGET_CERT_DIR"
echo "Backup directory: $BACKUP_DIR"
```

### Let's Encrypt Integration Migration

```bash
#!/bin/bash
# migrate-letsencrypt-to-rwwwrse.sh

DOMAINS=(api.example.com app.example.com web.example.com)
RWWWRSE_SSL_DIR="/opt/rwwwrse/ssl"
CERTBOT_HOOK_DIR="/etc/letsencrypt/renewal-hooks/deploy"

echo "Setting up Let's Encrypt integration for rwwwrse"

# Create SSL directory
mkdir -p "$RWWWRSE_SSL_DIR"

# Create renewal hook for rwwwrse
cat > "$CERTBOT_HOOK_DIR/rwwwrse-reload.sh" << 'EOF'
#!/bin/bash
# Certbot renewal hook for rwwwrse

RWWWRSE_SSL_DIR="/opt/rwwwrse/ssl"
RENEWED_DOMAINS="${RENEWED_DOMAINS}"

for domain in $RENEWED_DOMAINS; do
    echo "Processing renewed certificate for $domain"
    
    # Copy new certificates
    if [ -f "/etc/letsencrypt/live/$domain/fullchain.pem" ]; then
        cp "/etc/letsencrypt/live/$domain/fullchain.pem" "$RWWWRSE_SSL_DIR/$domain.crt"
        cp "/etc/letsencrypt/live/$domain/privkey.pem" "$RWWWRSE_SSL_DIR/$domain.key"
        
        chmod 644 "$RWWWRSE_SSL_DIR/$domain.crt"
        chmod 600 "$RWWWRSE_SSL_DIR/$domain.key"
        
        echo "Certificate for $domain updated"
    fi
done

# Reload rwwwrse
if systemctl is-active --quiet rwwwrse; then
    systemctl reload rwwwrse
    echo "rwwwrse reloaded with new certificates"
elif docker ps | grep -q rwwwrse; then
    docker restart rwwwrse
    echo "rwwwrse container restarted with new certificates"
fi
EOF

chmod +x "$CERTBOT_HOOK_DIR/rwwwrse-reload.sh"

# Initial certificate setup
for domain in "${DOMAINS[@]}"; do
    if [ -f "/etc/letsencrypt/live/$domain/fullchain.pem" ]; then
        echo "Setting up existing certificate for $domain"
        cp "/etc/letsencrypt/live/$domain/fullchain.pem" "$RWWWRSE_SSL_DIR/$domain.crt"
        cp "/etc/letsencrypt/live/$domain/privkey.pem" "$RWWWRSE_SSL_DIR/$domain.key"
        
        chmod 644 "$RWWWRSE_SSL_DIR/$domain.crt"
        chmod 600 "$RWWWRSE_SSL_DIR/$domain.key"
    else
        echo "No existing certificate found for $domain, requesting new one"
        certbot certonly --nginx -d "$domain" --non-interactive --agree-tos --email admin@example.com
    fi
done

echo "Let's Encrypt integration setup completed"
```

## Deployment-Specific Migrations

### Docker Compose Migration

#### From NGINX to rwwwrse

```yaml
# docker-compose-migration.yml
version: '3.8'

services:
  # Legacy NGINX (will be removed after migration)
  nginx-legacy:
    image: nginx:alpine
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf
      - ./ssl:/etc/ssl
    ports:
      - "8080:80"   # Moved to alternate port
      - "8443:443"  # Moved to alternate port
    networks:
      - legacy

  # New rwwwrse service
  rwwwrse:
    image: rwwwrse:latest
    environment:
      - RWWWRSE_BACKENDS=http://backend1:8080,http://backend2:8080
      - RWWWRSE_ENABLE_TLS=true
      - RWWWRSE_TLS_CERT_FILE=/etc/ssl/server.crt
      - RWWWRSE_TLS_KEY_FILE=/etc/ssl/server.key
      - RWWWRSE_LOG_LEVEL=info
    volumes:
      - ./ssl:/etc/ssl:ro
    ports:
      - "80:8080"   # Main HTTP port
      - "443:8443"  # Main HTTPS port
    networks:
      - frontend
      - backend
    depends_on:
      - backend1
      - backend2
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3

  # Backend services (unchanged)
  backend1:
    image: backend-app:latest
    networks:
      - backend

  backend2:
    image: backend-app:latest
    networks:
      - backend

networks:
  legacy:
  frontend:
  backend:
```

#### Migration Script

```bash
#!/bin/bash
# docker-compose-migration.sh

set -e

echo "Starting Docker Compose migration from NGINX to rwwwrse"

# Backup current configuration
docker-compose config > docker-compose-backup.yml

# Health check function
health_check() {
  local service=$1
  local health_url=$2
  local max_attempts=30
  local attempt=1
  
  while [ $attempt -le $max_attempts ]; do
    if curl -sf "$health_url" >/dev/null 2>&1; then
      echo "Health check passed for $service"
      return 0
    fi
    
    echo "Health check attempt $attempt/$max_attempts for $service"
    sleep 10
    ((attempt++))
  done
  
  echo "Health check failed for $service"
  return 1
}

# Start migration
echo "1. Starting rwwwrse service alongside NGINX"
docker-compose -f docker-compose-migration.yml up -d rwwwrse

# Wait for rwwwrse to be healthy
echo "2. Waiting for rwwwrse to become healthy"
if health_check "rwwwrse" "http://localhost:80/health"; then
  echo "rwwwrse is healthy, proceeding with migration"
else
  echo "rwwwrse health check failed, rolling back"
  docker-compose -f docker-compose-migration.yml down rwwwrse
  exit 1
fi

# Switch traffic by updating ports
echo "3. Switching traffic to rwwwrse"
docker-compose -f docker-compose-migration.yml stop nginx-legacy

# Final verification
echo "4. Final verification"
if health_check "rwwwrse" "http://localhost:80/health"; then
  echo "Migration successful, cleaning up legacy services"
  docker-compose -f docker-compose-migration.yml rm -f nginx-legacy
  echo "Migration completed successfully"
else
  echo "Final verification failed, rolling back"
  docker-compose -f docker-compose-migration.yml up -d nginx-legacy
  docker-compose -f docker-compose-migration.yml stop rwwwrse
  exit 1
fi
```

### Kubernetes Migration

#### Deployment Migration

```yaml
# kubernetes-migration.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: migration-scripts
data:
  migrate.sh: |
    #!/bin/bash
    set -e
    
    echo "Starting Kubernetes migration"
    
    # Scale down old deployment gradually
    kubectl scale deployment nginx-proxy --replicas=2
    sleep 30
    
    # Deploy rwwwrse
    kubectl apply -f rwwwrse-deployment.yaml
    kubectl rollout status deployment/rwwwrse-proxy
    
    # Health check
    kubectl wait --for=condition=ready pod -l app=rwwwrse-proxy --timeout=300s
    
    # Switch service
    kubectl patch service proxy-service -p '{"spec":{"selector":{"app":"rwwwrse-proxy"}}}'
    
    # Scale down old deployment
    kubectl scale deployment nginx-proxy --replicas=0
    
    echo "Migration completed"
---
apiVersion: batch/v1
kind: Job
metadata:
  name: migration-job
spec:
  template:
    spec:
      containers:
      - name: migration
        image: kubectl:latest
        command: ["/bin/bash"]
        args: ["/scripts/migrate.sh"]
        volumeMounts:
        - name: scripts
          mountPath: /scripts
      volumes:
      - name: scripts
        configMap:
          name: migration-scripts
          defaultMode: 0755
      restartPolicy: Never
```

### Cloud Platform Migrations

#### AWS ECS Migration

```json
{
  "family": "rwwwrse-migration",
  "networkMode": "awsvpc",
  "requiresCompatibilities": ["FARGATE"],
  "cpu": "512",
  "memory": "1024",
  "executionRoleArn": "arn:aws:iam::account:role/ecsTaskExecutionRole",
  "taskRoleArn": "arn:aws:iam::account:role/ecsTaskRole",
  "containerDefinitions": [
    {
      "name": "rwwwrse",
      "image": "rwwwrse:latest",
      "portMappings": [
        {
          "containerPort": 8080,
          "protocol": "tcp"
        }
      ],
      "environment": [
        {
          "name": "RWWWRSE_BACKENDS",
          "value": "http://backend.internal:8080"
        },
        {
          "name": "RWWWRSE_ENABLE_TLS",
          "value": "false"
        }
      ],
      "logConfiguration": {
        "logDriver": "awslogs",
        "options": {
          "awslogs-group": "/ecs/rwwwrse",
          "awslogs-region": "us-west-2",
          "awslogs-stream-prefix": "ecs"
        }
      },
      "healthCheck": {
        "command": ["CMD-SHELL", "curl -f http://localhost:8080/health || exit 1"],
        "interval": 30,
        "timeout": 5,
        "retries": 3,
        "startPeriod": 60
      }
    }
  ]
}
```

#### ECS Migration Script

```bash
#!/bin/bash
# ecs-migration.sh

CLUSTER_NAME="production-cluster"
SERVICE_NAME="proxy-service"
OLD_TASK_DEFINITION="nginx-proxy:1"
NEW_TASK_DEFINITION="rwwwrse-migration:1"

echo "Starting ECS migration"

# Register new task definition
aws ecs register-task-definition --cli-input-json file://rwwwrse-task-definition.json

# Update service with blue-green deployment
echo "Updating service to use new task definition"
aws ecs update-service \
  --cluster $CLUSTER_NAME \
  --service $SERVICE_NAME \
  --task-definition $NEW_TASK_DEFINITION \
  --deployment-configuration "maximumPercent=200,minimumHealthyPercent=50"

# Wait for deployment
echo "Waiting for deployment to complete"
aws ecs wait services-stable \
  --cluster $CLUSTER_NAME \
  --services $SERVICE_NAME

echo "ECS migration completed successfully"
```

## Version-Specific Migrations

### rwwwrse v1.x to v2.x

#### Breaking Changes

- Environment variable prefix changed from `REVERSE_` to `RWWWRSE_`
- Configuration file format changed from YAML to environment variables
- Health check endpoint moved from `/status` to `/health`

#### A Migration Script

```bash
#!/bin/bash
# v1-to-v2-migration.sh

CONFIG_FILE="/etc/rwwwrse/config.yml"
NEW_CONFIG_FILE="/etc/rwwwrse/config.env"

echo "Migrating rwwwrse from v1.x to v2.x"

if [ ! -f "$CONFIG_FILE" ]; then
  echo "No v1.x configuration found at $CONFIG_FILE"
  exit 1
fi

# Convert YAML configuration to environment variables
python3 << 'EOF'
import yaml
import sys

config_file = '/etc/rwwwrse/config.yml'
output_file = '/etc/rwwwrse/config.env'

try:
    with open(config_file, 'r') as f:
        config = yaml.safe_load(f)
    
    with open(output_file, 'w') as f:
        f.write("# Migrated from v1.x configuration\n\n")
        
        # Map old configuration to new environment variables
        mapping = {
            'listen_port': 'RWWWRSE_PORT',
            'backends': 'RWWWRSE_BACKENDS',
            'ssl.enabled': 'RWWWRSE_ENABLE_TLS',
            'ssl.cert_file': 'RWWWRSE_TLS_CERT_FILE',
            'ssl.key_file': 'RWWWRSE_TLS_KEY_FILE',
            'rate_limit.enabled': 'RWWWRSE_ENABLE_RATE_LIMITING',
            'rate_limit.requests_per_minute': 'RWWWRSE_RATE_LIMIT',
            'health_check.path': 'RWWWRSE_HEALTH_CHECK_PATH',
            'logging.level': 'RWWWRSE_LOG_LEVEL'
        }
        
        def get_nested_value(obj, key):
            keys = key.split('.')
            value = obj
            for k in keys:
                if isinstance(value, dict) and k in value:
                    value = value[k]
                else:
                    return None
            return value
        
        for old_key, new_key in mapping.items():
            value = get_nested_value(config, old_key)
            if value is not None:
                if isinstance(value, list):
                    value = ','.join(str(v) for v in value)
                f.write(f"{new_key}={value}\n")
    
    print(f"Configuration migrated to {output_file}")
    
except Exception as e:
    print(f"Migration failed: {e}")
    sys.exit(1)
EOF

# Update systemd service file
if [ -f "/etc/systemd/system/rwwwrse.service" ]; then
  echo "Updating systemd service file"
  
  # Backup current service file
  cp /etc/systemd/system/rwwwrse.service /etc/systemd/system/rwwwrse.service.backup
  
  # Update service file to use new configuration
  sed -i 's|--config=/etc/rwwwrse/config.yml|--env-file=/etc/rwwwrse/config.env|g' \
    /etc/systemd/system/rwwwrse.service
  
  # Reload systemd
  systemctl daemon-reload
fi

echo "v1.x to v2.x migration completed"
echo "Please review $NEW_CONFIG_FILE before restarting the service"
```

## Rollback Procedures

### Quick Rollback Script

```bash
#!/bin/bash
# quick-rollback.sh

DEPLOYMENT_TYPE=${1:-docker-compose}
BACKUP_TIMESTAMP=${2:-$(date +%Y%m%d_%H%M%S)}

echo "Performing quick rollback for $DEPLOYMENT_TYPE"

case $DEPLOYMENT_TYPE in
  docker-compose)
    echo "Rolling back Docker Compose deployment"
    
    if [ -f "docker-compose-backup.yml" ]; then
      docker-compose -f docker-compose-backup.yml up -d
      docker-compose logs rwwwrse
    else
      echo "No backup configuration found"
      exit 1
    fi
    ;;
    
  kubernetes)
    echo "Rolling back Kubernetes deployment"
    
    # Rollback to previous deployment
    kubectl rollout undo deployment/rwwwrse-proxy
    kubectl rollout status deployment/rwwwrse-proxy
    
    # Restore service selector if needed
    if [ -f "service-backup.yaml" ]; then
      kubectl apply -f service-backup.yaml
    fi
    ;;
    
  systemd)
    echo "Rolling back systemd service"
    
    # Stop current service
    systemctl stop rwwwrse
    
    # Restore backup configuration
    if [ -f "/etc/rwwwrse/config.yml.backup" ]; then
      cp /etc/rwwwrse/config.yml.backup /etc/rwwwrse/config.yml
    fi
    
    # Restore backup binary if available
    if [ -f "/usr/local/bin/rwwwrse.backup" ]; then
      cp /usr/local/bin/rwwwrse.backup /usr/local/bin/rwwwrse
    fi
    
    # Start service
    systemctl start rwwwrse
    systemctl status rwwwrse
    ;;
    
  *)
    echo "Unsupported deployment type: $DEPLOYMENT_TYPE"
    echo "Supported types: docker-compose, kubernetes, systemd"
    exit 1
    ;;
esac

echo "Rollback completed for $DEPLOYMENT_TYPE"
```

### Automated Rollback Triggers

```bash
#!/bin/bash
# automated-rollback-monitor.sh

HEALTH_CHECK_URL="http://localhost:8080/health"
MAX_FAILURES=3
FAILURE_COUNT=0
CHECK_INTERVAL=30

echo "Starting automated rollback monitor"

while true; do
  if curl -sf "$HEALTH_CHECK_URL" >/dev/null 2>&1; then
    FAILURE_COUNT=0
    echo "$(date): Health check passed"
  else
    ((FAILURE_COUNT++))
    echo "$(date): Health check failed ($FAILURE_COUNT/$MAX_FAILURES)"
    
    if [ $FAILURE_COUNT -ge $MAX_FAILURES ]; then
      echo "$(date): Maximum failures reached, triggering rollback"
      ./quick-rollback.sh
      exit 1
    fi
  fi
  
  sleep $CHECK_INTERVAL
done
```

## Post-Migration Validation

### Comprehensive Validation Suite

```bash
#!/bin/bash
# post-migration-validation.sh

set -e

echo "Starting post-migration validation"

# Test basic connectivity
echo "1. Testing basic connectivity"
if curl -sf http://localhost:8080/health >/dev/null; then
  echo "  ✓ HTTP health check passed"
else
  echo "  ✗ HTTP health check failed"
  exit 1
fi

# Test HTTPS if enabled
if curl -sf https://localhost:8443/health >/dev/null 2>&1; then
  echo "  ✓ HTTPS health check passed"
else
  echo "  ⚠ HTTPS health check failed or not configured"
fi

# Test backend connectivity
echo "2. Testing backend connectivity"
BACKENDS=$(curl -s http://localhost:8080/debug/backends 2>/dev/null || echo "")
if [ ! -z "$BACKENDS" ]; then
  echo "  ✓ Backend status endpoint accessible"
  echo "  Backends: $BACKENDS"
else
  echo "  ⚠ Backend status not available"
fi

# Performance comparison
echo "3. Performance testing"
echo "  Running basic load test..."

# Simple load test
for i in {1..100}; do
  curl -s http://localhost:8080/health >/dev/null &
done
wait

echo "  ✓ Basic load test completed"

# SSL certificate validation
echo "4. SSL certificate validation"
if command -v openssl >/dev/null 2>&1; then
  echo | openssl s_client -connect localhost:8443 -servername localhost 2>/dev/null | \
    openssl x509 -noout -dates 2>/dev/null && echo "  ✓ SSL certificate valid" || echo "  ⚠ SSL certificate validation failed"
fi

# Log analysis
echo "5. Log analysis"
if docker ps | grep -q rwwwrse; then
  echo "  Recent logs:"
  docker logs rwwwrse --tail 10
elif systemctl is-active --quiet rwwwrse; then
  echo "  Recent logs:"
  journalctl -u rwwwrse --lines 10 --no-pager
fi

echo "Post-migration validation completed"
```

## Related Documentation

- [Deployment Guide](DEPLOYMENT.md) - Platform-specific deployment strategies
- [Configuration Guide](CONFIGURATION.md) - Environment-specific configuration
- [SSL/TLS Guide](SSL-TLS.md) - Certificate management strategies
- [Operations Guide](OPERATIONS.md) - Monitoring and troubleshooting
- [Development Guide](DEVELOPMENT.md) - Local development setup
- [Docker Compose Examples](../examples/docker-compose/) - Container deployment examples
- [Kubernetes Examples](../examples/kubernetes/) - Orchestration examples
- [Cloud-Specific Examples](../examples/cloud-specific/) - Cloud platform deployments
- [CI/CD Examples](../examples/cicd/) - Automated deployment pipelines
