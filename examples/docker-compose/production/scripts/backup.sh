#!/bin/bash

# Production Backup Script for rwwwrse Environment
# This script creates comprehensive backups of databases, configurations, and logs

set -euo pipefail

# Configuration
BACKUP_DIR="${BACKUP_DIR:-/backup}"
DATE=$(date +%Y%m%d_%H%M%S)
RETENTION_DAYS="${RETENTION_DAYS:-30}"
S3_BUCKET="${S3_BACKUP_BUCKET:-}"
POSTGRES_HOST="${POSTGRES_HOST:-postgres-primary}"
POSTGRES_DB="${POSTGRES_DB:-rwwwrse}"
POSTGRES_USER="${POSTGRES_USER:-rwwwrse}"
REDIS_HOST="${REDIS_HOST:-redis}"

# Logging
LOG_FILE="${BACKUP_DIR}/backup_${DATE}.log"
mkdir -p "$(dirname "$LOG_FILE")"

log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1" | tee -a "$LOG_FILE"
}

error_exit() {
    log "ERROR: $1"
    exit 1
}

# Create backup directories
mkdir -p "${BACKUP_DIR}/postgres"
mkdir -p "${BACKUP_DIR}/redis"
mkdir -p "${BACKUP_DIR}/configs"
mkdir -p "${BACKUP_DIR}/logs"

log "Starting backup process..."

# PostgreSQL Backup
log "Backing up PostgreSQL database..."
if pg_dump -h "$POSTGRES_HOST" -U "$POSTGRES_USER" -d "$POSTGRES_DB" \
    --no-password --verbose --format=custom \
    --file="${BACKUP_DIR}/postgres/postgres_${DATE}.backup"; then
    log "PostgreSQL backup completed successfully"
    
    # Compress the backup
    gzip "${BACKUP_DIR}/postgres/postgres_${DATE}.backup"
    log "PostgreSQL backup compressed"
else
    error_exit "PostgreSQL backup failed"
fi

# Redis Backup
log "Backing up Redis data..."
if redis-cli -h "$REDIS_HOST" --rdb "${BACKUP_DIR}/redis/redis_${DATE}.rdb"; then
    log "Redis backup completed successfully"
    
    # Compress the backup
    gzip "${BACKUP_DIR}/redis/redis_${DATE}.rdb"
    log "Redis backup compressed"
else
    error_exit "Redis backup failed"
fi

# Configuration Backup
log "Backing up configuration files..."
tar -czf "${BACKUP_DIR}/configs/configs_${DATE}.tar.gz" \
    -C / \
    etc/rwwwrse \
    etc/nginx \
    etc/prometheus \
    etc/alertmanager \
    etc/grafana \
    2>/dev/null || log "Warning: Some config directories may not exist"

log "Configuration backup completed"

# Application Logs Backup (last 7 days)
log "Backing up recent application logs..."
find /var/log -name "*.log" -mtime -7 -type f | \
tar -czf "${BACKUP_DIR}/logs/logs_${DATE}.tar.gz" \
    --files-from=- 2>/dev/null || log "Warning: Some log files may not be accessible"

log "Log backup completed"

# Docker Volumes Backup (if any persistent volumes exist)
log "Backing up Docker volumes..."
docker run --rm \
    -v rwwwrse_prometheus_data:/volume \
    -v "${BACKUP_DIR}:/backup" \
    alpine tar -czf "/backup/volumes/prometheus_data_${DATE}.tar.gz" -C /volume . 2>/dev/null || true

docker run --rm \
    -v rwwwrse_grafana_data:/volume \
    -v "${BACKUP_DIR}:/backup" \
    alpine tar -czf "/backup/volumes/grafana_data_${DATE}.tar.gz" -C /volume . 2>/dev/null || true

log "Docker volumes backup completed"

# Create backup manifest
log "Creating backup manifest..."
cat > "${BACKUP_DIR}/manifest_${DATE}.json" << EOF
{
  "backup_date": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "backup_id": "${DATE}",
  "environment": "production",
  "components": {
    "postgres": {
      "file": "postgres/postgres_${DATE}.backup.gz",
      "size": "$(du -h "${BACKUP_DIR}/postgres/postgres_${DATE}.backup.gz" 2>/dev/null | cut -f1 || echo 'unknown')"
    },
    "redis": {
      "file": "redis/redis_${DATE}.rdb.gz",
      "size": "$(du -h "${BACKUP_DIR}/redis/redis_${DATE}.rdb.gz" 2>/dev/null | cut -f1 || echo 'unknown')"
    },
    "configs": {
      "file": "configs/configs_${DATE}.tar.gz",
      "size": "$(du -h "${BACKUP_DIR}/configs/configs_${DATE}.tar.gz" 2>/dev/null | cut -f1 || echo 'unknown')"
    },
    "logs": {
      "file": "logs/logs_${DATE}.tar.gz",
      "size": "$(du -h "${BACKUP_DIR}/logs/logs_${DATE}.tar.gz" 2>/dev/null | cut -f1 || echo 'unknown')"
    }
  },
  "total_size": "$(du -sh "${BACKUP_DIR}" | cut -f1)",
  "retention_policy": "${RETENTION_DAYS} days"
}
EOF

log "Backup manifest created"

# Upload to S3 if configured
if [[ -n "$S3_BUCKET" ]]; then
    log "Uploading backup to S3..."
    if command -v aws >/dev/null 2>&1; then
        aws s3 sync "${BACKUP_DIR}" "s3://${S3_BUCKET}/backups/$(hostname)/${DATE}/" \
            --exclude "*.log" || log "Warning: S3 upload had issues"
        log "Backup uploaded to S3"
    else
        log "Warning: AWS CLI not found, skipping S3 upload"
    fi
fi

# Cleanup old backups
log "Cleaning up old backups (older than ${RETENTION_DAYS} days)..."
find "${BACKUP_DIR}" -type f -name "*.backup.gz" -mtime +${RETENTION_DAYS} -delete 2>/dev/null || true
find "${BACKUP_DIR}" -type f -name "*.rdb.gz" -mtime +${RETENTION_DAYS} -delete 2>/dev/null || true
find "${BACKUP_DIR}" -type f -name "*.tar.gz" -mtime +${RETENTION_DAYS} -delete 2>/dev/null || true
find "${BACKUP_DIR}" -type f -name "manifest_*.json" -mtime +${RETENTION_DAYS} -delete 2>/dev/null || true
find "${BACKUP_DIR}" -type f -name "backup_*.log" -mtime +${RETENTION_DAYS} -delete 2>/dev/null || true

log "Cleanup completed"

# Backup verification
log "Verifying backup integrity..."
BACKUP_COUNT=$(find "${BACKUP_DIR}" -name "*_${DATE}.*" -type f | wc -l)
if [[ $BACKUP_COUNT -lt 3 ]]; then
    error_exit "Backup verification failed - insufficient backup files created"
fi

# Test PostgreSQL backup
if gunzip -t "${BACKUP_DIR}/postgres/postgres_${DATE}.backup.gz" 2>/dev/null; then
    log "PostgreSQL backup integrity verified"
else
    error_exit "PostgreSQL backup integrity check failed"
fi

# Test Redis backup
if gunzip -t "${BACKUP_DIR}/redis/redis_${DATE}.rdb.gz" 2>/dev/null; then
    log "Redis backup integrity verified"
else
    error_exit "Redis backup integrity check failed"
fi

log "Backup process completed successfully"
log "Backup location: ${BACKUP_DIR}"
log "Backup ID: ${DATE}"

# Send notification if configured
if [[ -n "${WEBHOOK_URL:-}" ]]; then
    curl -X POST "$WEBHOOK_URL" \
        -H "Content-Type: application/json" \
        -d "{
            \"text\": \"✅ Production backup completed successfully\",
            \"attachments\": [{
                \"color\": \"good\",
                \"fields\": [{
                    \"title\": \"Backup ID\",
                    \"value\": \"${DATE}\",
                    \"short\": true
                }, {
                    \"title\": \"Environment\",
                    \"value\": \"Production\",
                    \"short\": true
                }, {
                    \"title\": \"Total Size\",
                    \"value\": \"$(du -sh "${BACKUP_DIR}" | cut -f1)\",
                    \"short\": true
                }]
            }]
        }" 2>/dev/null || log "Warning: Failed to send notification"
fi

exit 0