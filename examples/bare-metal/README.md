# Bare-Metal Deployment Guide for rwwwrse

This guide covers deploying rwwwrse directly on bare-metal servers or virtual machines without containerization. This deployment method provides maximum performance and control but requires more manual configuration.

## Overview

Bare-metal deployment involves:
- Running rwwwrse as a system service using systemd
- Optional: Using NGINX or Apache as a reverse proxy for SSL termination and caching
- Manual SSL certificate management
- System-level monitoring and logging configuration

## Prerequisites

- Linux system with systemd support (Ubuntu 18.04+, CentOS 7+, etc.)
- Go 1.21+ installed for building rwwwrse
- Root or sudo access for system configuration
- Domain names configured with DNS A records pointing to your server

## Quick Start

1. **Build and install rwwwrse:**
```bash
# Clone and build
git clone <repository-url>
cd rwwwrse
go build -o /usr/local/bin/rwwwrse ./cmd/rwwwrse

# Create rwwwrse user
sudo useradd --system --shell /bin/false --home /var/lib/rwwwrse rwwwrse
sudo mkdir -p /var/lib/rwwwrse /var/log/rwwwrse
sudo chown rwwwrse:rwwwrse /var/lib/rwwwrse /var/log/rwwwrse
```

2. **Install systemd service:**
```bash
sudo cp examples/bare-metal/systemd/rwwwrse.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable rwwwrse
```

3. **Configure environment:**
```bash
sudo tee /etc/rwwwrse/config.env > /dev/null << 'EOF'
RWWWRSE_PORT=8080
RWWWRSE_HOST=0.0.0.0
RWWWRSE_LOG_LEVEL=info
RWWWRSE_LOG_FORMAT=json
RWWWRSE_HEALTH_PATH=/health
RWWWRSE_METRICS_PATH=/metrics
RWWWRSE_ENABLE_TLS=false
RWWWRSE_ROUTES_API_TARGET=http://localhost:3001
RWWWRSE_ROUTES_APP_TARGET=http://localhost:3000
RWWWRSE_ROUTES_WEB_TARGET=http://localhost:3002
EOF

sudo chown rwwwrse:rwwwrse /etc/rwwwrse/config.env
sudo chmod 640 /etc/rwwwrse/config.env
```

4. **Start service:**
```bash
sudo systemctl start rwwwrse
sudo systemctl status rwwwrse
```

## Systemd Service Configuration

### Service File Features

The provided [`rwwwrse.service`](systemd/rwwwrse.service) includes:

- **Security hardening** with multiple systemd security features
- **Resource limits** to prevent resource exhaustion
- **Automatic restart** on failure with exponential backoff
- **Proper user isolation** with dedicated service account
- **Environment file** support for configuration management

### Key Security Features

```ini
# Process isolation
User=rwwwrse
Group=rwwwrse
DynamicUser=false

# Filesystem security
ReadOnlyDirectories=/
ReadWriteDirectories=/var/lib/rwwwrse /var/log/rwwwrse /tmp
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true

# Network security
PrivateNetwork=false
RestrictAddressFamilies=AF_INET AF_INET6

# System call filtering
SystemCallFilter=@system-service
SystemCallErrorNumber=EPERM
```

### Configuration Management

Environment variables are loaded from `/etc/rwwwrse/config.env`:

```bash
# Core configuration
RWWWRSE_PORT=8080
RWWWRSE_HOST=0.0.0.0
RWWWRSE_LOG_LEVEL=info
RWWWRSE_LOG_FORMAT=json

# Health and metrics
RWWWRSE_HEALTH_PATH=/health
RWWWRSE_METRICS_PATH=/metrics

# TLS configuration (disabled when using reverse proxy)
RWWWRSE_ENABLE_TLS=false

# Route configuration
RWWWRSE_ROUTES_API_TARGET=http://localhost:3001
RWWWRSE_ROUTES_API_HOST=api.example.com
RWWWRSE_ROUTES_APP_TARGET=http://localhost:3000
RWWWRSE_ROUTES_APP_HOST=app.example.com
RWWWRSE_ROUTES_WEB_TARGET=http://localhost:3002
RWWWRSE_ROUTES_WEB_HOST=web.example.com
RWWWRSE_ROUTES_ADMIN_TARGET=http://localhost:3003
RWWWRSE_ROUTES_ADMIN_HOST=admin.example.com
```

### Service Management

```bash
# Start/stop/restart service
sudo systemctl start rwwwrse
sudo systemctl stop rwwwrse
sudo systemctl restart rwwwrse

# Enable/disable automatic startup
sudo systemctl enable rwwwrse
sudo systemctl disable rwwwrse

# Check service status
sudo systemctl status rwwwrse

# View logs
sudo journalctl -u rwwwrse -f
sudo journalctl -u rwwwrse --since "1 hour ago"

# Reload configuration (restart required for env changes)
sudo systemctl reload-or-restart rwwwrse
```

## NGINX Reverse Proxy Integration

### Installation and Setup

1. **Install NGINX:**
```bash
# Ubuntu/Debian
sudo apt update && sudo apt install nginx

# CentOS/RHEL
sudo yum install nginx
# or for newer versions:
sudo dnf install nginx
```

2. **Install the configuration:**
```bash
sudo cp examples/bare-metal/nginx-proxy/nginx.conf /etc/nginx/nginx.conf
sudo nginx -t  # Test configuration
sudo systemctl restart nginx
sudo systemctl enable nginx
```

### Configuration Features

The [`nginx.conf`](nginx-proxy/nginx.conf) provides:

- **SSL termination** with modern TLS configuration
- **HTTP/2 support** for improved performance
- **Rate limiting** with different zones for different endpoints
- **Caching** with configurable cache zones
- **Security headers** including HSTS, CSP, X-Frame-Options
- **Load balancing** between multiple rwwwrse instances
- **Gzip compression** for better bandwidth utilization
- **Access logging** in JSON format for structured analysis

### Key Features

**SSL Configuration:**
```nginx
ssl_protocols TLSv1.2 TLSv1.3;
ssl_ciphers ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256...;
ssl_session_cache shared:SSL:10m;
ssl_stapling on;
```

**Rate Limiting:**
```nginx
limit_req_zone $binary_remote_addr zone=api:10m rate=10r/s;
limit_req_zone $binary_remote_addr zone=general:10m rate=5r/s;
limit_req_zone $binary_remote_addr zone=login:10m rate=1r/s;
```

**Upstream Configuration:**
```nginx
upstream rwwwrse_backend {
    server 127.0.0.1:8080 max_fails=3 fail_timeout=30s weight=1;
    server 127.0.0.1:8081 max_fails=3 fail_timeout=30s weight=1 backup;
    keepalive 32;
}
```

### NGINX Management

```bash
# Test configuration
sudo nginx -t

# Reload configuration (graceful)
sudo nginx -s reload

# Restart service
sudo systemctl restart nginx

# Check status
sudo systemctl status nginx

# View access logs
sudo tail -f /var/log/nginx/access.log

# View error logs
sudo tail -f /var/log/nginx/error.log
```

## Apache HTTP Server Integration

### Installation and Setup

1. **Install Apache:**
```bash
# Ubuntu/Debian
sudo apt update && sudo apt install apache2

# CentOS/RHEL
sudo yum install httpd
# or for newer versions:
sudo dnf install httpd

# Install additional modules
sudo a2enmod ssl rewrite headers proxy proxy_http proxy_balancer lbmethod_byrequests
```

2. **Install the configuration:**
```bash
sudo cp examples/bare-metal/apache-proxy/httpd.conf /etc/httpd/conf/httpd.conf
# or on Debian/Ubuntu:
sudo cp examples/bare-metal/apache-proxy/httpd.conf /etc/apache2/apache2.conf

sudo httpd -t  # Test configuration
sudo systemctl restart httpd  # or apache2 on Debian/Ubuntu
sudo systemctl enable httpd   # or apache2 on Debian/Ubuntu
```

### Configuration Features

The [`httpd.conf`](apache-proxy/httpd.conf) provides:

- **SSL/TLS termination** with modern cipher suites
- **HTTP/2 support** for improved performance
- **DDoS protection** using mod_evasive
- **Web Application Firewall** using mod_security
- **Load balancing** with health checks
- **Caching** with mod_cache_disk
- **Security headers** and hardening
- **Structured logging** in JSON format

### Key Features

**Load Balancer Configuration:**
```apache
<Proxy balancer://rwwwrse-cluster>
    BalancerMember http://127.0.0.1:8080 route=1 retry=300
    BalancerMember http://127.0.0.1:8081 route=2 retry=300 status=+H
    ProxySet connectiontimeout=30
    ProxySet retry=300
    ProxySet timeout=60
</Proxy>
```

**Security Module Configuration:**
```apache
<IfModule mod_security2.c>
    SecRuleEngine On
    SecRequestBodyAccess On
    SecResponseBodyAccess On
    SecRequestBodyLimit 13107200
    SecRequestBodyNoFilesLimit 131072
</IfModule>
```

**DDoS Protection:**
```apache
<IfModule mod_evasive24.c>
    DOSHashTableSize    2048
    DOSPageCount        2
    DOSSiteCount        50
    DOSPageInterval     1
    DOSSiteInterval     1
    DOSBlockingPeriod   600
</IfModule>
```

### Apache Management

```bash
# Test configuration
sudo httpd -t
# or on Debian/Ubuntu:
sudo apache2ctl configtest

# Reload configuration (graceful)
sudo httpd -k graceful
# or on Debian/Ubuntu:
sudo apache2ctl graceful

# Restart service
sudo systemctl restart httpd
# or on Debian/Ubuntu:
sudo systemctl restart apache2

# Check status
sudo systemctl status httpd
# or on Debian/Ubuntu:
sudo systemctl status apache2

# View access logs
sudo tail -f /var/log/httpd/access_log
# or on Debian/Ubuntu:
sudo tail -f /var/log/apache2/access.log

# View error logs
sudo tail -f /var/log/httpd/error_log
# or on Debian/Ubuntu:
sudo tail -f /var/log/apache2/error.log
```

## SSL Certificate Management

### Let's Encrypt with Certbot

1. **Install Certbot:**
```bash
# Ubuntu/Debian
sudo apt install certbot

# CentOS/RHEL
sudo yum install certbot
# or:
sudo dnf install certbot
```

2. **Obtain certificates:**
```bash
# For NGINX
sudo certbot certonly --webroot -w /var/www/letsencrypt \
  -d api.example.com \
  -d app.example.com \
  -d web.example.com \
  -d admin.example.com

# For Apache
sudo certbot certonly --apache \
  -d api.example.com \
  -d app.example.com \
  -d web.example.com \
  -d admin.example.com
```

3. **Setup automatic renewal:**
```bash
# Add to crontab
sudo crontab -e

# Add this line for automatic renewal
0 12 * * * /usr/bin/certbot renew --quiet && systemctl reload nginx
# or for Apache:
0 12 * * * /usr/bin/certbot renew --quiet && systemctl reload httpd
```

### Manual Certificate Installation

For custom certificates:

```bash
# Create certificate directory
sudo mkdir -p /etc/ssl/certs /etc/ssl/private

# Install certificates
sudo cp your-domain.crt /etc/ssl/certs/
sudo cp your-domain.key /etc/ssl/private/
sudo chmod 644 /etc/ssl/certs/your-domain.crt
sudo chmod 600 /etc/ssl/private/your-domain.key

# Update configuration files with correct paths
```

## Monitoring and Logging

### System Monitoring

1. **Install monitoring tools:**
```bash
# Install htop, iotop, netstat
sudo apt install htop iotop net-tools
# or:
sudo yum install htop iotop net-tools
```

2. **Monitor rwwwrse process:**
```bash
# Check process status
ps aux | grep rwwwrse

# Monitor resource usage
htop -p $(pgrep rwwwrse)

# Check network connections
netstat -tlnp | grep :8080

# Monitor file descriptors
ls -la /proc/$(pgrep rwwwrse)/fd/ | wc -l
```

### Log Management

1. **Configure log rotation:**
```bash
sudo tee /etc/logrotate.d/rwwwrse > /dev/null << 'EOF'
/var/log/rwwwrse/*.log {
    daily
    missingok
    rotate 52
    compress
    delaycompress
    notifempty
    create 644 rwwwrse rwwwrse
    postrotate
        systemctl reload rwwwrse
    endscript
}
EOF
```

2. **Configure rsyslog for structured logging:**
```bash
sudo tee /etc/rsyslog.d/50-rwwwrse.conf > /dev/null << 'EOF'
# rwwwrse application logs
if $programname == 'rwwwrse' then /var/log/rwwwrse/app.log
& stop
EOF

sudo systemctl restart rsyslog
```

### Health Checks

Create health check scripts:

```bash
sudo tee /usr/local/bin/rwwwrse-health-check > /dev/null << 'EOF'
#!/bin/bash
# rwwwrse health check script

HEALTH_URL="http://localhost:8080/health"
TIMEOUT=10

if curl -f -s --max-time $TIMEOUT "$HEALTH_URL" > /dev/null; then
    echo "rwwwrse is healthy"
    exit 0
else
    echo "rwwwrse health check failed"
    exit 1
fi
EOF

sudo chmod +x /usr/local/bin/rwwwrse-health-check

# Test the health check
/usr/local/bin/rwwwrse-health-check
```

## Performance Tuning

### System-Level Optimization

1. **Network tuning:**
```bash
sudo tee -a /etc/sysctl.conf > /dev/null << 'EOF'
# Network performance tuning for rwwwrse
net.core.somaxconn = 65535
net.core.netdev_max_backlog = 5000
net.core.rmem_max = 134217728
net.core.wmem_max = 134217728
net.ipv4.tcp_wmem = 4096 65536 134217728
net.ipv4.tcp_rmem = 4096 65536 134217728
net.ipv4.tcp_congestion_control = bbr
net.ipv4.tcp_slow_start_after_idle = 0
net.ipv4.tcp_tw_reuse = 1
EOF

sudo sysctl -p
```

2. **File descriptor limits:**
```bash
sudo tee -a /etc/security/limits.conf > /dev/null << 'EOF'
# File descriptor limits for rwwwrse
rwwwrse soft nofile 65535
rwwwrse hard nofile 65535
EOF
```

3. **CPU affinity (optional):**
```bash
# Pin rwwwrse to specific CPU cores
sudo taskset -cp 0-3 $(pgrep rwwwrse)
```

### rwwwrse Configuration Tuning

```bash
# Add to /etc/rwwwrse/config.env
RWWWRSE_READ_TIMEOUT=30s
RWWWRSE_WRITE_TIMEOUT=30s
RWWWRSE_IDLE_TIMEOUT=120s
RWWWRSE_MAX_HEADER_BYTES=1048576
RWWWRSE_READ_HEADER_TIMEOUT=10s
```

## Security Best Practices

### Firewall Configuration

```bash
# Using UFW (Ubuntu)
sudo ufw default deny incoming
sudo ufw default allow outgoing
sudo ufw allow ssh
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
sudo ufw enable

# Using firewalld (CentOS/RHEL)
sudo firewall-cmd --permanent --add-service=ssh
sudo firewall-cmd --permanent --add-service=http
sudo firewall-cmd --permanent --add-service=https
sudo firewall-cmd --reload
```

### AppArmor/SELinux Configuration

For Ubuntu with AppArmor:
```bash
sudo tee /etc/apparmor.d/usr.local.bin.rwwwrse > /dev/null << 'EOF'
#include <tunables/global>

/usr/local/bin/rwwwrse {
  #include <abstractions/base>
  #include <abstractions/nameservice>
  
  capability net_bind_service,
  
  /usr/local/bin/rwwwrse mr,
  /etc/rwwwrse/config.env r,
  /var/lib/rwwwrse/ rw,
  /var/lib/rwwwrse/** rw,
  /var/log/rwwwrse/ w,
  /var/log/rwwwrse/** w,
  
  network inet stream,
  network inet6 stream,
}
EOF

sudo apparmor_parser -r /etc/apparmor.d/usr.local.bin.rwwwrse
```

### Fail2ban Integration

```bash
sudo tee /etc/fail2ban/filter.d/rwwwrse.conf > /dev/null << 'EOF'
[Definition]
failregex = ^.*"remote_addr":"<HOST>".*"status":"4[0-9]{2}".*$
ignoreregex =
EOF

sudo tee /etc/fail2ban/jail.d/rwwwrse.conf > /dev/null << 'EOF'
[rwwwrse]
enabled = true
port = http,https
filter = rwwwrse
logpath = /var/log/nginx/access.log
maxretry = 5
bantime = 3600
findtime = 600
EOF

sudo systemctl restart fail2ban
```

## Troubleshooting

### Common Issues

1. **Service won't start:**
```bash
# Check service status
sudo systemctl status rwwwrse

# Check logs
sudo journalctl -u rwwwrse -n 50

# Verify configuration
/usr/local/bin/rwwwrse --config-check
```

2. **Port binding issues:**
```bash
# Check if port is in use
sudo netstat -tlnp | grep :8080

# Check firewall
sudo ufw status
# or:
sudo firewall-cmd --list-all
```

3. **Permission issues:**
```bash
# Check file permissions
ls -la /usr/local/bin/rwwwrse
ls -la /etc/rwwwrse/config.env
ls -la /var/lib/rwwwrse/
ls -la /var/log/rwwwrse/

# Fix permissions if needed
sudo chown rwwwrse:rwwwrse /var/lib/rwwwrse /var/log/rwwwrse
sudo chmod 755 /var/lib/rwwwrse /var/log/rwwwrse
```

4. **SSL certificate issues:**
```bash
# Check certificate validity
openssl x509 -in /etc/letsencrypt/live/example.com/fullchain.pem -text -noout

# Test SSL configuration
openssl s_client -connect example.com:443 -servername example.com

# Check certificate renewal
sudo certbot certificates
```

### Performance Issues

1. **High CPU usage:**
```bash
# Check top processes
top -p $(pgrep rwwwrse)

# Check for CPU throttling
dmesg | grep -i throttl

# Profile the application
go tool pprof http://localhost:8080/debug/pprof/profile
```

2. **High memory usage:**
```bash
# Check memory usage
cat /proc/$(pgrep rwwwrse)/status | grep Vm

# Check for memory leaks
go tool pprof http://localhost:8080/debug/pprof/heap
```

3. **Network issues:**
```bash
# Check network connections
ss -tuln | grep :8080

# Check network statistics
cat /proc/net/netstat

# Monitor network traffic
sudo nethogs
```

## Backup and Recovery

### Configuration Backup

```bash
#!/bin/bash
# Backup script for rwwwrse configuration

BACKUP_DIR="/var/backups/rwwwrse"
DATE=$(date +%Y%m%d_%H%M%S)

mkdir -p "$BACKUP_DIR"

# Backup configuration
tar -czf "$BACKUP_DIR/rwwwrse-config-$DATE.tar.gz" \
  /etc/rwwwrse/ \
  /etc/systemd/system/rwwwrse.service \
  /etc/nginx/nginx.conf \
  /etc/ssl/

# Keep only last 30 days of backups
find "$BACKUP_DIR" -name "*.tar.gz" -mtime +30 -delete

echo "Backup completed: $BACKUP_DIR/rwwwrse-config-$DATE.tar.gz"
```

### Disaster Recovery

```bash
#!/bin/bash
# Disaster recovery script

# Stop services
sudo systemctl stop rwwwrse nginx

# Restore from backup
tar -xzf /var/backups/rwwwrse/rwwwrse-config-latest.tar.gz -C /

# Restore permissions
sudo chown -R rwwwrse:rwwwrse /etc/rwwwrse/
sudo chmod 640 /etc/rwwwrse/config.env

# Reload systemd
sudo systemctl daemon-reload

# Start services
sudo systemctl start rwwwrse nginx

# Verify services
sudo systemctl status rwwwrse nginx
```

## Migration from Other Deployments

### From Docker to Bare-Metal

1. **Export configuration:**
```bash
# From Docker Compose environment
docker-compose exec rwwwrse env | grep RWWWRSE_ > bare-metal-config.env
```

2. **Adapt configuration:**
```bash
# Convert Docker internal hostnames to localhost
sed -i 's/api:3001/localhost:3001/g' bare-metal-config.env
sed -i 's/app:3000/localhost:3000/g' bare-metal-config.env
```

3. **Deploy and test:**
```bash
sudo cp bare-metal-config.env /etc/rwwwrse/config.env
sudo systemctl restart rwwwrse
```

### From Kubernetes to Bare-Metal

1. **Extract ConfigMap:**
```bash
kubectl get configmap rwwwrse-config -o yaml > k8s-config.yaml
```

2. **Convert to environment file:**
```bash
# Extract data section and format as env file
yq eval '.data | to_entries | .[] | .key + "=" + .value' k8s-config.yaml > /etc/rwwwrse/config.env
```

## Scaling Considerations

### Horizontal Scaling

Run multiple rwwwrse instances:

```bash
# Create additional service instances
sudo cp /etc/systemd/system/rwwwrse.service /etc/systemd/system/rwwwrse@.service

# Modify the template service
sudo sed -i 's/config.env/config-%i.env/g' /etc/systemd/system/rwwwrse@.service

# Create instance-specific configs
sudo cp /etc/rwwwrse/config.env /etc/rwwwrse/config-1.env
sudo cp /etc/rwwwrse/config.env /etc/rwwwrse/config-2.env

# Modify ports
sudo sed -i 's/RWWWRSE_PORT=8080/RWWWRSE_PORT=8081/g' /etc/rwwwrse/config-2.env

# Start instances
sudo systemctl enable rwwwrse@1 rwwwrse@2
sudo systemctl start rwwwrse@1 rwwwrse@2
```

### Load Balancer Integration

Update NGINX/Apache configuration to include additional instances in the upstream configuration.

## Next Steps

- Consider implementing centralized logging with ELK stack or similar
- Set up monitoring with Prometheus and Grafana
- Implement automated deployment with CI/CD pipelines
- Consider moving to containerized deployment for easier management

## Related Documentation

- [Docker Compose Examples](../docker-compose/)
- [Kubernetes Examples](../kubernetes/)
- [SSL/TLS Configuration Guide](../../docs/SSL-TLS.md)
- [Operations Guide](../../docs/OPERATIONS.md)
- [Performance Tuning Guide](../../docs/PERFORMANCE.md)