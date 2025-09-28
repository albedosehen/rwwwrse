# Microservices Architecture with rwwwrse

This example demonstrates a complete microservices architecture using rwwwrse as the reverse proxy. It includes multiple backend services, databases, message queues, and comprehensive service discovery.

## Architecture Overview

```plaintext
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   Web Browser   │───▶│   rwwwrse Proxy  │───▶│  Microservices  │
└─────────────────┘    │  (Load Balancer) │    └─────────────────┘
                       └──────────────────┘              │
                                │                        │
                                ▼                        ▼
    ┌─────────────────────────────────────────────────────────────┐
    │                    Service Mesh                             │
    │  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐ ┌────────┐ │
    │  │     Web     │ │    Auth     │ │     API     │ │ Admin  │ │
    │  │  Frontend   │ │   Service   │ │   Gateway   │ │ Panel  │ │
    │  └─────────────┘ └─────────────┘ └─────────────┘ └────────┘ │
    └─────────────────────────────────────────────────────────────┘
                                │
                                ▼
    ┌─────────────────────────────────────────────────────────────┐
    │                 Infrastructure Layer                        │
    │  ┌─────────────┐ ┌─────────────┐ ┌─────────────────────────┐ │
    │  │ PostgreSQL  │ │    Redis    │ │       RabbitMQ          │ │
    │  │  Database   │ │    Cache    │ │   Message Queue         │ │
    │  └─────────────┘ └─────────────┘ └─────────────────────────┘ │
    └─────────────────────────────────────────────────────────────┘
```

## Services Included

### Frontend Services

- **Web Frontend** (`web.example.com`) - React/Vue.js application
- **Admin Panel** (`admin.example.com`) - Administrative interface (Adminer)

### Backend Services  

- **Authentication Service** (`auth.example.com`) - User auth & session management
- **API Gateway** (`api.example.com`) - RESTful API endpoints (httpbin for demo)

### Infrastructure Services

- **PostgreSQL** - Primary database with sample schema
- **Redis** - Caching and session storage
- **RabbitMQ** - Message queue with management UI

### Reverse Proxy

- **rwwwrse** - Handles routing, load balancing, health checks, SSL termination

## Prerequisites

- Docker and Docker Compose
- At least 4GB RAM available for containers
- Ports 80, 443, 9090, and 15672 available

## Quick Start

### 1. Clone and Navigate

```bash
git clone <repository-url>
cd examples/docker-compose/microservices
```

### 2. Start All Services

```bash
docker-compose up -d
```

### 3. Verify Services

```bash
# Check all containers are running
docker-compose ps

# Check rwwwrse logs
docker-compose logs rwwwrse

# Check service health
curl http://localhost/health
```

### 4. Access Services

For local testing, add these entries to your `/etc/hosts` file:

```plaintext
127.0.0.1 web.example.com
127.0.0.1 auth.example.com  
127.0.0.1 api.example.com
127.0.0.1 admin.example.com
```

Then access:

- **Web Frontend**: <http://web.example.com> or <http://localhost>
- **Authentication**: <http://auth.example.com>
- **API Gateway**: <http://api.example.com>
- **Admin Panel**: <http://admin.example.com>
- **RabbitMQ Management**: <http://localhost:15672> (mquser/mqpass)
- **Metrics**: <http://localhost:9090/metrics>

## 🔐 Authentication Testing

### Sample Accounts

The database is pre-populated with test accounts:

- **Admin User**: <admin@example.com> / demo123
- **Demo User**: <demo@example.com> / demo123
- **API Service**: <api@example.com> / demo123
- **Test User**: <test@example.com> / demo123

### Authentication Flow

1. Visit <http://auth.example.com>
2. Use demo credentials to test login functionality
3. Check session management and token validation

## API Testing

### Health Checks

```bash
# rwwwrse health
curl http://localhost/health

# Auth service health
curl http://auth.example.com/status

# API service health  
curl http://api.example.com/health
```

### API Endpoints

```bash
# Test httpbin API functionality
curl http://api.example.com/get
curl http://api.example.com/status/200
curl -X POST http://api.example.com/post -d '{"test": "data"}'

# Test authentication endpoints
curl http://auth.example.com/api/v1/auth
curl -X POST http://auth.example.com/api/v1/login \
  -H "Content-Type: application/json" \
  -d '{"username": "demo@example.com", "password": "demo123"}'
```

## 🗄️ Database Access

### Connect to PostgreSQL

```bash
# Via container
docker-compose exec postgres-db psql -U dbuser -d microservices

# Via admin panel
# Visit http://admin.example.com
# Server: postgres-db, Username: dbuser, Password: dbpass
```

### Sample Queries

```sql
-- View users
SELECT * FROM users;

-- Check API endpoints
SELECT * FROM api_endpoints;

-- View request logs
SELECT * FROM request_logs ORDER BY created_at DESC LIMIT 10;

-- User statistics
SELECT * FROM user_stats;
```

## Monitoring & Metrics

### Prometheus Metrics

```bash
# View all metrics
curl http://localhost:9090/metrics

# rwwwrse specific metrics
curl http://localhost:9090/metrics | grep rwwwrse
```

### RabbitMQ Management

- **URL**: <http://localhost:15672>
- **Username**: mquser
- **Password**: mqpass

### Logs Monitoring

```bash
# Follow all logs
docker-compose logs -f

# Specific service logs
docker-compose logs -f rwwwrse
docker-compose logs -f api-service
docker-compose logs -f auth-service
```

## 🔧 Configuration

### Environment Variables

Key rwwwrse configuration in [`docker-compose.yml`](./docker-compose.yml):

```yaml
environment:
  # Server configuration
  RWWWRSE_SERVER_HOST: "0.0.0.0"
  RWWWRSE_SERVER_PORT: "8080"
  RWWWRSE_SERVER_HTTPS_PORT: "8443"
  
  # TLS configuration
  RWWWRSE_TLS_ENABLED: "true"
  RWWWRSE_TLS_AUTO_CERT: "false"  # Set to true for production
  RWWWRSE_TLS_DOMAINS: "api.example.com,auth.example.com,web.example.com,admin.example.com"
  
  # Complex routing configuration
  RWWWRSE_BACKENDS_ROUTES: >
    {
      "api.example.com": {
        "url": "http://api-service:8080",
        "health_path": "/health",
        "timeout": "30s"
      },
      "auth.example.com": {
        "url": "http://auth-service:8080", 
        "health_path": "/status",
        "timeout": "15s"
      },
      "web.example.com": {
        "url": "http://web-frontend:3000",
        "health_path": "/",
        "timeout": "30s"
      },
      "admin.example.com": {
        "url": "http://admin-panel:8080",
        "health_path": "/health",
        "timeout": "30s"
      }
    }
```

### Custom Service Configuration

To add your own services:

1. **Add service to docker-compose.yml**:

    ```yaml
    my-service:
      image: my-app:latest
      networks:
        - frontend
        - backend
    ```

2. **Update rwwwrse routing**:

    ```yaml
    RWWWRSE_BACKENDS_ROUTES: >
      {
        "myapp.example.com": {
          "url": "http://my-service:8080",
          "health_path": "/health",
          "timeout": "30s"
        }
      }
    ```

3. **Add hosts entry**:

    ```plaintext
    127.0.0.1 myapp.example.com
    ```

## Testing Scenarios

### Load Testing

```bash
# Install Apache Bench
apt-get install apache2-utils

# Test web frontend
ab -n 100 -c 10 http://web.example.com/

# Test API endpoints
ab -n 100 -c 10 http://api.example.com/get

# Test authentication
ab -n 50 -c 5 http://auth.example.com/status
```

### Failover Testing

```bash
# Stop a service to test health checks
docker-compose stop api-service

# Check rwwwrse health detection
docker-compose logs rwwwrse

# Restart service
docker-compose start api-service
```

### Network Testing

```bash
# Test inter-service communication
docker-compose exec web-frontend wget -qO- http://api-service:8080/health
docker-compose exec api-service wget -qO- http://auth-service:8080/status
```

## Troubleshooting

### Common Issues

#### Services Not Starting

```bash
# Check container status
docker-compose ps

# Check specific service logs
docker-compose logs <service-name>

# Restart problematic service
docker-compose restart <service-name>
```

#### Connection Issues

```bash
# Verify network connectivity
docker network ls
docker network inspect microservices-frontend

# Test DNS resolution
docker-compose exec rwwwrse nslookup api-service
```

#### Database Connection Issues

```bash
# Check PostgreSQL health
docker-compose exec postgres-db pg_isready -U dbuser

# Verify database initialization
docker-compose logs postgres-db | grep "database system is ready"
```

#### Performance Issues

```bash
# Check resource usage
docker stats

# Monitor rwwwrse metrics
curl http://localhost:9090/metrics | grep -E "(request_duration|concurrent_connections)"
```

### Health Check Endpoints

| Service | Health Check URL | Expected Response |
|---------|------------------|-------------------|
| rwwwrse | <http://localhost/health> | 200 OK |
| Web Frontend | <http://web.example.com/health> | 200 OK |
| Auth Service | <http://auth.example.com/status> | 200 OK |
| API Gateway | <http://api.example.com/health> | 200 OK |
| Admin Panel | <http://admin.example.com/health> | 200 OK |

## Development Workflow

### Making Changes

```bash
# Rebuild and restart specific service
docker-compose build web-frontend
docker-compose up -d web-frontend

# Update rwwwrse configuration
docker-compose restart rwwwrse
```

### Adding SSL Certificates

For production deployment with real SSL certificates:

1. Update `RWWWRSE_TLS_AUTO_CERT` to `"true"`
2. Set real domain names in `RWWWRSE_TLS_DOMAINS`
3. Ensure DNS points to your server
4. Let's Encrypt will automatically provision certificates

### Database Migrations

```bash
# Connect to database
docker-compose exec postgres-db psql -U dbuser -d microservices

# Run custom migrations
docker-compose exec postgres-db psql -U dbuser -d microservices -f /path/to/migration.sql
```

## Next Steps

1. **Scale Services**: Use `docker-compose up -d --scale api-service=3`
2. **Add Monitoring**: Integrate with Prometheus/Grafana
3. **Implement CI/CD**: See examples in [`examples/cicd/`](../../cicd/)
4. **Production Deployment**: See [`examples/kubernetes/`](../../kubernetes/)
5. **Security Hardening**: Review [`docs/SSL-TLS.md`](../../../docs/SSL-TLS.md)

## Related Examples

- [Simple Setup](../simple/) - Basic reverse proxy
- [Development Environment](../development/) - Development-focused setup
- [Production Setup](../production/) - Production-ready deployment
- [Kubernetes Deployment](../../kubernetes/) - Container orchestration
- [Cloud Deployments](../../cloud-specific/) - Cloud provider examples
