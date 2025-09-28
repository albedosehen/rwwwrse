# rwwwrse Production Deployment with Docker Compose

This example demonstrates a complete production-ready deployment of rwwwrse with comprehensive monitoring, logging, high availability, and backup solutions.

## 🏗️ Architecture Overview

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Load Balancer │────│     rwwwrse     │────│   Applications  │
│    (nginx)      │    │  Reverse Proxy  │    │                 │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                                │
                    ┌───────────┼───────────┐
                    │           │           │
            ┌───────▼───┐  ┌────▼────┐  ┌──▼────┐
            │PostgreSQL │  │  Redis  │  │ Logs  │
            │Primary+   │  │ Cache   │  │ ELK   │
            │Replica    │  │         │  │Stack  │
            └───────────┘  └─────────┘  └───────┘
                    │
            ┌───────▼───────────────────────────┐
            │        Monitoring Stack          │
            │ Prometheus + Grafana + AlertMgr  │
            └─────────────────────────────────┘
```

## 🎯 Features

- **High Availability**: PostgreSQL primary-replica setup, Redis clustering
- **Comprehensive Monitoring**: Prometheus metrics, Grafana dashboards, Alertmanager notifications
- **Centralized Logging**: ELK stack with Fluentd for log aggregation
- **Automated Backups**: Database backups with S3 integration
- **Security**: TLS encryption, security headers, network isolation
- **Performance**: Optimized configurations for production workloads
- **Observability**: Health checks, metrics collection, distributed tracing

## 📋 Prerequisites

### System Requirements

- **CPU**: Minimum 4 cores, recommended 8+ cores
- **Memory**: Minimum 8GB RAM, recommended 16GB+ RAM
- **Storage**: Minimum 100GB SSD, recommended 500GB+ SSD
- **Network**: Stable internet connection for certificate management

### Software Dependencies

```bash
# Docker and Docker Compose
docker --version  # >= 20.10.0
docker-compose --version  # >= 1.29.0

# Optional: AWS CLI for S3 backups
aws --version  # >= 2.0.0
```

### Domain and SSL Requirements

- Domain name with DNS control
- Email address for Let's Encrypt certificates
- Firewall rules allowing ports 80, 443, and any custom application ports

## 🚀 Quick Start

### 1. Clone and Setup

```bash
# Navigate to production example
cd examples/docker-compose/production

# Copy environment template
cp .env.example .env

# Edit configuration
nano .env
```

### 2. Configure Environment Variables

Edit the `.env` file with your specific values:

```bash
# Domain and SSL
DOMAIN=your-domain.com
ACME_EMAIL=admin@your-domain.com

# Database
POSTGRES_PASSWORD=your-secure-postgres-password
POSTGRES_REPLICATION_PASSWORD=your-replication-password

# Redis
REDIS_PASSWORD=your-redis-password

# Monitoring
GRAFANA_ADMIN_PASSWORD=your-grafana-password

# Backup (optional)
S3_BACKUP_BUCKET=your-backup-bucket
AWS_ACCESS_KEY_ID=your-aws-access-key
AWS_SECRET_ACCESS_KEY=your-aws-secret-key

# Notifications (optional)
SLACK_WEBHOOK_URL=your-slack-webhook
SMTP_PASSWORD=your-smtp-password
PAGERDUTY_INTEGRATION_KEY=your-pagerduty-key
```

### 3. Deploy the Stack

```bash
# Create necessary directories
mkdir -p {data,logs,backup,ssl}

# Start the production stack
docker-compose up -d

# Verify deployment
docker-compose ps
docker-compose logs rwwwrse
```

### 4. Initial Configuration

```bash
# Wait for services to be ready (2-3 minutes)
./scripts/wait-for-services.sh

# Import Grafana dashboards
./scripts/import-dashboards.sh

# Run initial backup
./scripts/backup.sh
```

## ⚙️ Configuration

### rwwwrse Configuration

The rwwwrse proxy is configured via environment variables:

```yaml
environment:
  RWWWRSE_LOG_LEVEL: info
  RWWWRSE_LOG_FORMAT: json
  RWWWRSE_METRICS_ENABLED: "true"
  RWWWRSE_METRICS_PORT: "9090"
  RWWWRSE_TLS_AUTO: "true"
  RWWWRSE_TLS_EMAIL: "${ACME_EMAIL}"
  RWWWRSE_RATE_LIMIT: "100"
  RWWWRSE_TIMEOUT: "30s"
```

### Database Configuration

#### PostgreSQL Primary

- **File**: `database/postgres-primary.conf`
- **Features**: WAL replication, performance tuning, monitoring
- **Memory**: Optimized for production workloads
- **Security**: SSL enabled, restricted connections

#### PostgreSQL Replica

- **File**: `database/postgres-replica.conf`
- **Features**: Hot standby, read scaling, automatic failover
- **Performance**: Optimized for read-heavy workloads
- **Recovery**: Streaming replication with archive recovery

### Monitoring Configuration

#### Prometheus

- **File**: `monitoring/prometheus.yml`
- **Retention**: 30 days of metrics data
- **Targets**: All services with service discovery
- **Rules**: Custom alerting rules for rwwwrse

#### Alertmanager

- **File**: `monitoring/alertmanager.yml`
- **Channels**: Email, Slack, PagerDuty
- **Routing**: Severity-based alert routing
- **Inhibition**: Smart alert suppression

### Logging Configuration

#### Fluentd

- **File**: `logging/fluentd.conf`
- **Sources**: Application logs, system logs, audit logs
- **Parsing**: JSON and structured log parsing
- **Outputs**: Elasticsearch, S3 backup

## 📊 Monitoring and Observability

### Grafana Dashboards

Access Grafana at `https://your-domain.com/grafana`:

1. **rwwwrse Overview**: Request rates, response times, error rates
2. **Infrastructure**: CPU, memory, disk, network metrics
3. **Database**: PostgreSQL performance, replication lag
4. **Security**: Failed logins, suspicious activity
5. **Business**: Custom application metrics

### Key Metrics to Monitor

#### rwwwrse Metrics

```prometheus
# Request rate
rate(http_requests_total[5m])

# Response time percentiles
histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))

# Error rate
rate(http_requests_total{status=~"5.."}[5m]) / rate(http_requests_total[5m])

# Active connections
rwwwrse_active_connections
```

#### Database Metrics

```prometheus
# Connection usage
pg_stat_database_numbackends / pg_settings_max_connections * 100

# Replication lag
pg_replication_lag

# Query performance
rate(pg_stat_database_tup_returned[5m])
```

### Alert Thresholds

- **High response time**: 95th percentile > 1s for 2 minutes
- **High error rate**: Error rate > 5% for 1 minute
- **Database down**: Connection failure for 30 seconds
- **Replication lag**: Lag > 30 seconds for 1 minute
- **High memory usage**: Memory usage > 90% for 5 minutes

## 🔐 Security

### Network Security

```yaml
# Network isolation
networks:
  frontend:
    driver: bridge
  backend:
    driver: bridge
    internal: true  # No external access
```

### SSL/TLS Configuration

- **Automatic certificates**: Let's Encrypt with auto-renewal
- **Strong ciphers**: Modern TLS configuration
- **HSTS**: HTTP Strict Transport Security enabled
- **Security headers**: CSP, X-Frame-Options, etc.

### Access Controls

```yaml
# Database access
- Host-based authentication (pg_hba.conf)
- SSL-only connections
- Limited user privileges

# Monitoring access
- Grafana authentication
- Prometheus access restrictions
- Alert manager routing
```

## 💾 Backup and Recovery

### Automated Backups

The backup script (`scripts/backup.sh`) runs automated backups:

```bash
# Daily backup schedule
0 2 * * * /path/to/backup.sh

# Backup components
- PostgreSQL dumps (compressed)
- Redis snapshots
- Configuration files
- Application logs (last 7 days)
- Docker volumes
```

### Backup Verification

```bash
# Check backup integrity
./scripts/verify-backup.sh

# Test restore procedure
./scripts/test-restore.sh
```

### Disaster Recovery

#### Database Recovery

```bash
# Stop services
docker-compose stop postgres-primary postgres-replica

# Restore from backup
gunzip < backup/postgres/postgres_YYYYMMDD_HHMMSS.backup.gz | \
  pg_restore -h localhost -U rwwwrse -d rwwwrse --clean --if-exists

# Restart replication
docker-compose up -d postgres-primary postgres-replica
```

#### Complete Environment Recovery

```bash
# Clone repository
git clone <repo-url>
cd examples/docker-compose/production

# Restore configurations
tar -xzf backup/configs/configs_YYYYMMDD_HHMMSS.tar.gz -C /

# Deploy stack
cp .env.backup .env
docker-compose up -d

# Restore data
./scripts/restore-data.sh backup/YYYYMMDD_HHMMSS/
```

## 🔧 Maintenance

### Regular Maintenance Tasks

#### Daily

```bash
# Check service status
docker-compose ps

# Review logs for errors
docker-compose logs --tail=100 rwwwrse | grep ERROR

# Verify backup completion
ls -la backup/ | tail -5
```

#### Weekly

```bash
# Update certificates (if needed)
docker-compose exec rwwwrse rwwwrse renew-certs

# Clean old logs
find logs/ -name "*.log" -mtime +7 -delete

# Database maintenance
docker-compose exec postgres-primary vacuumdb -U rwwwrse -d rwwwrse --analyze
```

#### Monthly

```bash
# Update container images
docker-compose pull
docker-compose up -d

# Review and rotate secrets
./scripts/rotate-secrets.sh

# Performance review
./scripts/performance-report.sh
```

### Scaling Procedures

#### Horizontal Scaling

```yaml
# Add more rwwwrse instances
rwwwrse:
  deploy:
    replicas: 3

# Add read replicas
postgres-replica-2:
  extends:
    service: postgres-replica
  environment:
    - POSTGRES_PRIMARY_SLOT_NAME=replica2_slot
```

#### Vertical Scaling

```yaml
# Increase resource limits
rwwwrse:
  deploy:
    resources:
      limits:
        cpus: '2.0'
        memory: 2G
      reservations:
        cpus: '1.0'
        memory: 1G
```

## 🐛 Troubleshooting

### Common Issues

#### rwwwrse Won't Start

```bash
# Check logs
docker-compose logs rwwwrse

# Common causes:
- Invalid environment variables
- Port conflicts
- Certificate issues
- Network connectivity

# Solutions:
docker-compose down
docker-compose up -d rwwwrse
```

#### Database Connection Issues

```bash
# Check database status
docker-compose exec postgres-primary pg_isready

# Check replication
docker-compose exec postgres-primary \
  psql -U rwwwrse -c "SELECT * FROM pg_stat_replication;"

# Reset replication if needed
./scripts/reset-replication.sh
```

#### Monitoring Issues

```bash
# Restart monitoring stack
docker-compose restart prometheus grafana alertmanager

# Check Prometheus targets
curl http://localhost:9090/api/v1/targets

# Import missing dashboards
./scripts/import-dashboards.sh
```

### Performance Issues

#### High Response Times

```bash
# Check system resources
docker stats

# Check database performance
docker-compose exec postgres-primary \
  psql -U rwwwrse -c "SELECT * FROM pg_stat_activity WHERE state = 'active';"

# Check cache hit rates
docker-compose exec redis redis-cli info stats
```

#### Memory Issues

```bash
# Check memory usage by container
docker stats --format "table {{.Container}}\t{{.CPUPerc}}\t{{.MemUsage}}"

# Adjust container limits
# Edit docker-compose.yml and restart
```

### Log Analysis

```bash
# Application errors
docker-compose logs rwwwrse | grep ERROR

# Database slow queries
docker-compose exec postgres-primary \
  tail -f /var/log/postgresql/postgresql-*.log | grep "duration:"

# Security events
docker-compose logs | grep -E "(401|403|failed|unauthorized)"
```

## 📈 Performance Optimization

### Database Optimization

```sql
-- Monitor slow queries
SELECT query, mean_time, calls 
FROM pg_stat_statements 
ORDER BY mean_time DESC 
LIMIT 10;

-- Check index usage
SELECT schemaname, tablename, attname, n_distinct, correlation 
FROM pg_stats 
WHERE tablename = 'your_table';
```

### Cache Optimization

```bash
# Redis performance
redis-cli info memory
redis-cli info stats

# Optimize cache policies
redis-cli config set maxmemory-policy allkeys-lru
```

### Network Optimization

```yaml
# Enable HTTP/2
rwwwrse:
  environment:
    - RWWWRSE_HTTP2_ENABLED=true

# Optimize connection pooling
postgres-primary:
  environment:
    - POSTGRES_MAX_CONNECTIONS=200
```

## 🚨 Emergency Procedures

### Service Recovery

```bash
# Emergency restart
docker-compose restart

# Force recreation
docker-compose down
docker-compose up -d --force-recreate

# Rollback to previous version
git checkout previous-tag
docker-compose up -d
```

### Data Recovery

```bash
# Emergency database recovery
./scripts/emergency-recovery.sh

# Restore from last known good backup
./scripts/restore-backup.sh $(ls backup/ | tail -1)
```

## 📞 Support

### Log Collection

```bash
# Collect all logs for support
./scripts/collect-logs.sh

# Generate system report
./scripts/system-report.sh
```

### Configuration Export

```bash
# Export current configuration
./scripts/export-config.sh

# Create support bundle
./scripts/create-support-bundle.sh
```

## 🔄 Updates and Upgrades

### rwwwrse Updates

```bash
# Update to latest version
docker-compose pull rwwwrse
docker-compose up -d rwwwrse

# Verify update
docker-compose logs rwwwrse
```

### Stack Updates

```bash
# Update all services
docker-compose pull
docker-compose up -d

# Verify all services
./scripts/health-check.sh
```

---

## 📚 Additional Resources

- [rwwwrse Documentation](../../docs/)
- [Docker Compose Reference](https://docs.docker.com/compose/)
- [Prometheus Monitoring](https://prometheus.io/docs/)
- [Grafana Dashboards](https://grafana.com/docs/)
- [PostgreSQL High Availability](https://www.postgresql.org/docs/current/high-availability.html)

## 🆘 Getting Help

If you encounter issues with this production deployment:

1. Check the [troubleshooting section](#-troubleshooting)
2. Review logs with `docker-compose logs`
3. Consult the [main documentation](../../docs/DEPLOYMENT.md)
4. Open an issue with your configuration and logs

Remember to sanitize any sensitive information before sharing logs or configurations.