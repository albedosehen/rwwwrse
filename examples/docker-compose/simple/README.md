# Simple Docker Compose Setup

This example demonstrates the basic usage of rwwwrse as a reverse proxy with a single backend service using Docker Compose.

## Overview

This setup includes:
- **rwwwrse**: Reverse proxy server
- **nginx-backend**: Simple nginx server serving static content

## Quick Start

1. **Clone and navigate to this directory**:
   ```bash
   git clone https://github.com/albedosehen/rwwwrse.git
   cd rwwwrse/examples/docker-compose/simple
   ```

2. **Start the services**:
   ```bash
   docker-compose up -d
   ```

3. **Test the proxy**:
   ```bash
   # Test the proxy (requests to localhost:8080 are routed to nginx backend)
   curl http://localhost:8080
   
   # Check health status
   curl http://localhost:8080/health
   
   # View metrics
   curl http://localhost:9090/metrics
   ```

4. **Stop the services**:
   ```bash
   docker-compose down
   ```

## Configuration

### Backend Routing

The proxy is configured to route requests for `localhost` to the nginx backend:

```json
{
  "localhost": {
    "url": "http://nginx-backend:80",
    "health_path": "/",
    "timeout": "30s",
    "max_idle_conns": 100,
    "max_idle_per_host": 10,
    "dial_timeout": "10s"
  }
}
```

### TLS Configuration

For this simple example, TLS auto-certificate is disabled to avoid requiring real domain names. In production, you would:

1. Set `RWWWRSE_TLS_AUTO_CERT=true`
2. Configure `RWWWRSE_TLS_EMAIL` with your email
3. Set `RWWWRSE_TLS_DOMAINS` with your actual domains

## Testing

### Basic Functionality
```bash
# Start services
docker-compose up -d

# Test proxy response
curl -v http://localhost:8080

# Expected: HTML response from nginx backend
```

### Health Checks
```bash
# Check overall proxy health
curl http://localhost:8080/health

# Expected response:
{
  "status": "healthy",
  "checks": {
    "backends": "healthy"
  }
}
```

### Metrics
```bash
# View Prometheus metrics
curl http://localhost:9090/metrics | grep rwwwrse

# Key metrics to look for:
# - rwwwrse_requests_total
# - rwwwrse_request_duration_seconds
# - rwwwrse_backend_health_status
```

## Customization

### Adding Custom Content

1. **Create custom HTML content**:
   ```bash
   mkdir -p nginx/html
   echo "<h1>Hello from rwwwrse!</h1>" > nginx/html/index.html
   ```

2. **Restart services**:
   ```bash
   docker-compose restart nginx-backend
   ```

### Environment Variables

You can override any configuration by setting environment variables:

```bash
# Use different ports
export RWWWRSE_SERVER_PORT=8090
export RWWWRSE_SERVER_HTTPS_PORT=8491

# Enable debug logging
export RWWWRSE_LOGGING_LEVEL=debug

# Restart with new configuration
docker-compose down
docker-compose up -d
```

### Custom Backend

Replace the nginx backend with your own service:

```yaml
# In docker-compose.yml, replace nginx-backend with:
my-backend:
  image: my-org/my-app:latest
  ports:
    - "3000:3000"
  networks:
    - proxy-network
```

Then update the routing configuration:
```json
{
  "localhost": {
    "url": "http://my-backend:3000",
    "health_path": "/health"
  }
}
```

## Production Considerations

This simple setup is great for:
- ✅ Local development
- ✅ Testing rwwwrse functionality
- ✅ Learning the configuration

For production, consider:
- 🔒 Enable TLS with real certificates
- 📊 Add monitoring stack (see [production example](../production/))
- 🔐 Configure proper security headers
- 🚀 Set up multiple replicas
- 💾 Configure persistent storage for certificates

## Troubleshooting

### Services Won't Start
```bash
# Check logs
docker-compose logs rwwwrse
docker-compose logs nginx-backend

# Check if ports are available
netstat -tulpn | grep -E ':(8080|8443|9090)'
```

### Backend Connection Issues
```bash
# Test backend directly
docker-compose exec rwwwrse wget -qO- http://nginx-backend:80

# Check network connectivity
docker network ls
docker network inspect simple-proxy-net
```

### Configuration Issues
```bash
# Validate environment variables
docker-compose exec rwwwrse env | grep RWWWRSE

# Test configuration syntax
docker-compose config
```

## Next Steps

- **Microservices**: Try the [microservices example](../microservices/) for multiple backend routing
- **Development**: Use the [development example](../development/) for local development with hot reload
- **Production**: Deploy with the [production example](../production/) for monitoring and security
- **Kubernetes**: Scale with [Kubernetes examples](../../kubernetes/)

## Files Structure

```
simple/
├── docker-compose.yml     # Main compose configuration
├── README.md             # This file
├── nginx/
│   ├── default.conf      # Nginx configuration
│   └── html/
│       └── index.html    # Custom content (optional)
└── .env.example          # Environment variables template