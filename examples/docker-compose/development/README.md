# Development Environment with rwwwrse

This example demonstrates a complete local development environment using rwwwrse as the reverse proxy. It's optimized for rapid development with hot reloading, debugging tools, real-time monitoring, and comprehensive development utilities.

## Development Features

- **Hot Reload**: Automatic file watching and application restart
- **Debug Mode**: Verbose logging and error reporting
- **Development Tools**: Real-time monitoring dashboard
- **API Testing**: Mock APIs with comprehensive endpoints
- **Email Testing**: Mailhog for email development
- **File Watcher**: Real-time file change notifications
- **Profiling**: Go pprof integration for performance analysis
- **Development Databases**: PostgreSQL and Redis with sample data

## Architecture

```plaintext
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   Developer     │───▶│   rwwwrse Proxy  │───▶│  Dev Services   │
│   Browser       │    │  (Debug Mode)    │    │                 │
└─────────────────┘    └──────────────────┘    └─────────────────┘
                                │                        │
                                ▼                        ▼
    ┌─────────────────────────────────────────────────────────────┐
    │                    Development Network                      │
    │  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐ ┌────────┐ │
    │  │  Frontend   │ │     API     │ │    Docs     │ │  Dev   │ │
    │  │    Dev      │ │    Dev      │ │    Server   │ │ Tools  │ │
    │  │ (Node.js)   │ │ (Node.js)   │ │  (Nginx)    │ │(Node.js)│ │
    │  └─────────────┘ └─────────────┘ └─────────────┘ └────────┘ │
    └─────────────────────────────────────────────────────────────┘
                                │
                                ▼
    ┌─────────────────────────────────────────────────────────────┐
    │                  Development Infrastructure                 │
    │  ┌─────────────┐ ┌─────────────┐ ┌─────────────────────────┐ │
    │  │ PostgreSQL  │ │    Redis    │ │       Mailhog           │ │
    │  │   DevDB     │ │   DevCache  │ │   Email Testing         │ │
    │  └─────────────┘ └─────────────┘ └─────────────────────────┘ │
    └─────────────────────────────────────────────────────────────┘
```

## Quick Start

### Prerequisites

- Docker and Docker Compose
- Node.js 18+ (for local development)
- At least 2GB RAM available

### 1. Clone and Setup

```bash
git clone <repository-url>
cd examples/docker-compose/development
```

### 2. Generate Development Certificates

```bash
# Create self-signed certificates for development
mkdir -p certs
openssl req -x509 -newkey rsa:4096 -keyout certs/dev.key -out certs/dev.crt \
  -days 365 -nodes -subj "/CN=localhost"
```

### 3. Start Development Environment

```bash
# Start all services
docker-compose up -d

# View logs
docker-compose logs -f
```

### 4. Configure Local Hosts

Add these entries to your `/etc/hosts` file (or `C:\Windows\System32\drivers\etc\hosts` on Windows):

```plaintext
127.0.0.1 app.localhost
127.0.0.1 api.localhost
127.0.0.1 docs.localhost
127.0.0.1 tools.localhost
```

### 5. Access Development Services

| Service | URL | Description |
|---------|-----|-------------|
| **Frontend** | <http://app.localhost> or <http://localhost> | React/Vue development server |
| **API** | <http://api.localhost> | Backend API with mock data |
| **Documentation** | <http://docs.localhost> | Project documentation |
| **Dev Tools** | <http://tools.localhost> | Real-time monitoring dashboard |
| **Mailhog** | <http://localhost:8025> | Email testing interface |
| **PostgreSQL** | localhost:5432 | Database (devuser/devpass) |
| **Redis** | localhost:6379 | Cache (password: devredispass) |
| **Metrics** | <http://localhost:9090/metrics> | rwwwrse metrics |
| **Profiling** | <http://localhost:6060/debug/pprof/> | Go performance profiling |

## Development Tools

### Real-Time Dashboard

Access the development dashboard at <http://tools.localhost> to monitor:

- **System Statistics**: CPU, memory, uptime, load average
- **Container Status**: All services with health checks
- **Live Logs**: Real-time log streaming from all services
- **File Watcher**: Notifications when files change
- **Quick Actions**: Restart services, clear data, hot reload

### API Development

The API service provides comprehensive endpoints for development:

```bash
# Health check
curl http://api.localhost/health

# Authentication testing
curl -X POST http://api.localhost/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email": "dev@example.com", "password": "dev123"}'

# User management
curl http://api.localhost/users

# Development endpoints
curl http://api.localhost/dev/sessions  # View active sessions
curl http://api.localhost/dev/logs      # View request logs
curl http://api.localhost/dev/reset     # Reset development data
```

### Hot Reload Development

Both frontend and API services support automatic reloading:

1. **Frontend Hot Reload**: Edit files in `./frontend/` directory
2. **API Hot Reload**: Edit files in `./api/` directory
3. **File Watcher**: Monitor changes via the dev tools dashboard

### Database Development

Access the development database:

```bash
# Connect via Docker
docker-compose exec postgres-dev psql -U devuser -d devdb

# Connect locally
psql -h localhost -p 5432 -U devuser -d devdb
```

Sample development data is automatically loaded including:

- Test users with different roles
- Sample API endpoints
- Request logging tables
- Development metrics

### Email Testing

Use Mailhog for email development:

- **SMTP**: localhost:1025 (for application configuration)
- **Web UI**: <http://localhost:8025> (to view sent emails)

Configure your applications to use Mailhog:

```javascript
// Node.js example
const nodemailer = require('nodemailer');
const transporter = nodemailer.createTransporter({
  host: 'mailhog',
  port: 1025,
  ignoreTLS: true
});
```

## Configuration

### rwwwrse Development Configuration

Key development settings in [`docker-compose.yml`](./docker-compose.yml):

```yaml
environment:
  # Debug logging
  RWWWRSE_LOGGING_LEVEL: "debug"
  RWWWRSE_LOGGING_FORMAT: "console"
  
  # Relaxed health checks
  RWWWRSE_HEALTH_INTERVAL: "10s"
  RWWWRSE_HEALTH_TIMEOUT: "3s"
  
  # Development profiling
  RWWWRSE_PROFILING_ENABLED: "true"
  RWWWRSE_PROFILING_PORT: "6060"
  
  # Permissive CORS for development
  RWWWRSE_SECURITY_CORS_ORIGINS: "*"
  RWWWRSE_SECURITY_RATE_LIMIT_ENABLED: "false"
```

### Service Configuration

Each service is optimized for development:

- **Frontend**: Nodemon auto-restart, hot module replacement ready
- **API**: Verbose logging, mock data, development endpoints
- **Database**: Sample schema with test data pre-loaded
- **Tools**: Real-time monitoring with WebSocket updates

### Environment Variables

Key environment variables for development:

```bash
# Node.js services
NODE_ENV=development
DEBUG=*
HOT_RELOAD=true

# Database
DATABASE_URL=postgresql://devuser:devpass@postgres-dev:5432/devdb
REDIS_URL=redis://redis-dev:6379

# Security (development only)
JWT_SECRET=dev-secret-key-not-for-production
```

## Development Workflows

### Frontend Development

1. **Edit Files**: Modify files in `./frontend/`
2. **Auto Reload**: Services automatically restart on changes
3. **View Changes**: Browser auto-refreshes (if configured)
4. **Debug**: Check logs in development tools dashboard

Example workflow:

```bash
# Edit frontend code
echo "console.log('Hello Dev!');" >> frontend/public/app.js

# Watch logs
docker-compose logs -f frontend-dev

# View in browser
open http://app.localhost
```

### API Development Post Checks

1. **Edit Endpoints**: Modify `./api/server.js`
2. **Test APIs**: Use curl or Postman
3. **Check Logs**: Monitor via development dashboard
4. **Debug Database**: Connect directly to PostgreSQL

Example API testing:

```bash
# Test new endpoint
curl -X POST http://api.localhost/test \
  -H "Content-Type: application/json" \
  -d '{"message": "Hello API"}'

# Check request logs
curl http://api.localhost/dev/logs
```

### Database Development Post Checks

1. **Schema Changes**: Edit `./database/init-dev.sql`
2. **Restart Database**: `docker-compose restart postgres-dev`
3. **Test Changes**: Connect and verify schema
4. **Seed Data**: Add test data to `./database/seed-dev.sql`

### Performance Profiling

Access Go profiling endpoints:

```bash
# CPU profile
curl http://localhost:6060/debug/pprof/profile?seconds=30 > cpu.prof

# Memory profile
curl http://localhost:6060/debug/pprof/heap > mem.prof

# Goroutine analysis
curl http://localhost:6060/debug/pprof/goroutine > goroutine.prof
```

## Troubleshooting

### Common Issues

#### Services Not Starting

```bash
# Check container status
docker-compose ps

# View service logs
docker-compose logs <service-name>

# Restart specific service
docker-compose restart <service-name>
```

#### Port Conflicts

```bash
# Check port usage
netstat -tulpn | grep :80
netstat -tulpn | grep :3000

# Stop conflicting services
sudo systemctl stop nginx  # if nginx is running
```

#### File Permission Issues

```bash
# Fix file permissions
sudo chown -R $USER:$USER .
chmod -R 755 .
```

#### Database Connection Issues

```bash
# Check PostgreSQL status
docker-compose exec postgres-dev pg_isready -U devuser

# Reset database
docker-compose down -v postgres-dev
docker-compose up -d postgres-dev
```

### Health Checks

Monitor service health:

```bash
# rwwwrse health
curl http://localhost/health

# API health
curl http://api.localhost/health

# Frontend health
curl http://app.localhost/health

# Dev tools health
curl http://tools.localhost/health
```

### Performance Issues

If services are slow:

```bash
# Check resource usage
docker stats

# Check logs for errors
docker-compose logs --tail=100

# Restart heavy services
docker-compose restart api-dev frontend-dev
```

### File Watching Issues

If hot reload isn't working:

```bash
# Check file watcher
curl http://tools.localhost/api/watch/start

# Manually trigger reload
curl -X POST http://tools.localhost/api/restart \
  -H "Content-Type: application/json" \
  -d '{"service": "frontend-dev"}'
```

## 📚 Development Tips

### VS Code Integration

For optimal VS Code development:

1. **Extensions**: Install Docker, Node.js, and database extensions
2. **Debugging**: Configure launch.json for remote debugging
3. **Terminal**: Use integrated terminal for docker-compose commands

### Testing Workflows

Recommended testing approach:

```bash
# Run unit tests (when implemented)
docker-compose exec api-dev npm test

# Integration testing
curl http://api.localhost/test
curl http://app.localhost/api/test

# Load testing
ab -n 100 -c 10 http://api.localhost/users
```

### Data Management

Managing development data:

```bash
# Reset all development data
curl http://api.localhost/dev/reset
curl http://tools.localhost/api/logs -X DELETE

# Backup development database
docker-compose exec postgres-dev pg_dump -U devuser devdb > dev-backup.sql

# Restore from backup
docker-compose exec -T postgres-dev psql -U devuser devdb < dev-backup.sql
```

### Security Notes

**Development Only**: This setup includes insecure configurations suitable only for development:

- Self-signed certificates
- Weak passwords
- Permissive CORS
- Disabled rate limiting
- Debug endpoints exposed

**Never use this configuration in production!**

## Advanced Usage

### Custom Services

To add your own service:

1. **Update docker-compose.yml**:

    ```yaml
    my-service:
      image: node:18-alpine
      working_dir: /app
      volumes:
        - ./my-service:/app
      networks:
        - dev-network
    ```

2. **Update rwwwrse routing**:

```yaml
RWWWRSE_BACKENDS_ROUTES: >
  {
    "myapp.localhost": {
      "url": "http://my-service:3000",
      "health_path": "/health"
    }
  }
```

### Multiple Environments

Run multiple development environments:

```bash
# Use different project names
docker-compose -p rwwwrse-dev1 up -d
docker-compose -p rwwwrse-dev2 up -d
```

### Production Testing

Test production configurations:

```bash
# Use production-like settings
RWWWRSE_LOGGING_LEVEL=info \
RWWWRSE_SECURITY_RATE_LIMIT_ENABLED=true \
docker-compose up -d
```

## Related Examples

- [Simple Setup](../simple/) - Basic reverse proxy
- [Microservices](../microservices/) - Complex service architecture  
- [Production Setup](../production/) - Production-ready deployment
- [Kubernetes](../../kubernetes/) - Container orchestration
