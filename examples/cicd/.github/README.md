# Synology Deployment for rwwwrse

This directory contains GitHub Actions workflows and configuration for deploying rwwwrse to a Synology NAS using Docker Compose.

## Files

- `workflows/syno.yaml` - GitHub Actions workflow for Synology deployment
- `syno.example.yaml` - Example workflow from another project (reference)

## Setup Instructions

### 1. Synology NAS Setup

1. **Install Docker on your Synology NAS**
   - Open Package Center
   - Install Docker package
   - Enable SSH access in Control Panel > Terminal & SNMP

2. **Set up GitHub Actions Runner**
   - Follow GitHub's self-hosted runner setup guide
   - Install the runner on your Synology NAS
   - Tag the runner with `synology` label

### 2. GitHub Repository Configuration

1. **Configure Repository Secrets**
   
   Go to your repository Settings > Secrets and variables > Actions and add:

   **Required Secrets:**
   - `DOPPLER_TOKEN` - Your Doppler project token (if using Doppler for secrets)
   - `HOSTNAME` - Your Synology NAS hostname or IP address

   **Optional Configuration Secrets:**
   - `TLS_AUTO_CERT` - Enable automatic TLS certificates (true/false, default: false)
   - `TLS_EMAIL` - Email for Let's Encrypt certificates
   - `TLS_DOMAINS` - Comma-separated list of domains for TLS
   - `BACKENDS_ROUTES` - JSON configuration for backend routing
   - `LOG_LEVEL` - Logging level (debug, info, warn, error, default: info)
   - `RATE_LIMIT_RPS` - Rate limit requests per second (default: 100)
   - `RATE_LIMIT_BURST` - Rate limit burst size (default: 200)
   - `CORS_ORIGINS` - CORS allowed origins (default: *)

2. **Example Backend Routes Configuration**
   
   For `BACKENDS_ROUTES` secret, use JSON format:
   ```json
   {
     "example.com": {
       "url": "http://backend-service:8080",
       "health_path": "/health",
       "timeout": "30s",
       "max_idle_conns": 100,
       "max_idle_per_host": 10,
       "dial_timeout": "10s"
     },
     "api.example.com": {
       "url": "http://api-service:8080",
       "health_path": "/api/health",
       "timeout": "30s",
       "max_idle_conns": 200,
       "max_idle_per_host": 20,
       "dial_timeout": "10s"
     }
   }
   ```

### 3. Deployment Process

The workflow will:

1. **Build Phase:**
   - Checkout the repository
   - Verify project structure
   - Build Docker image locally on Synology NAS
   - Tag image with specified version

2. **Deploy Phase:**
   - Prepare Docker environment and directories
   - Stop existing containers
   - Deploy using Docker Compose
   - Verify deployment health

3. **Verification:**
   - Check container status
   - Perform health checks on endpoints
   - Verify metrics endpoint accessibility

### 4. Manual Deployment

You can also trigger deployment manually:

1. Go to Actions tab in your GitHub repository
2. Select "Build & deploy rwwwrse to Synology" workflow
3. Click "Run workflow"
4. Optionally specify:
   - `image_tag`: Docker image tag (default: latest)
   - `no_build`: Skip build step if image already exists

### 5. Monitoring

After deployment, you can monitor:

- **Application:** `http://your-nas-ip:8080`
- **Health Check:** `http://your-nas-ip:8080/health`
- **Metrics:** `http://your-nas-ip:9090/metrics`

### 6. File Structure

```
rwwwrse/
├── docker-compose.deploy.yml    # Deployment Docker Compose file
├── Dockerfile                   # Multi-stage Docker build
├── examples/cicd/.github/
│   └── workflows/
│       └── syno.yaml           # Synology deployment workflow
└── cmd/rwwwrse/
    └── main.go                 # Application entry point
```

### 7. Troubleshooting

**Common Issues:**

1. **Runner Connection Issues:**
   - Verify runner is online in GitHub repository settings
   - Check runner logs on Synology NAS
   - Ensure proper network connectivity

2. **Docker Build Failures:**
   - Check Docker daemon is running on NAS
   - Verify sufficient disk space
   - Review build logs in workflow output

3. **Deployment Failures:**
   - Check Docker Compose configuration
   - Verify environment variables are set
   - Review container logs: `docker logs rwwwrse-app`

4. **Health Check Failures:**
   - Verify application is listening on correct port
   - Check firewall settings on NAS
   - Review application logs for startup errors

**Useful Commands:**

```bash
# Check container status
docker ps --filter "name=rwwwrse"

# View application logs
docker logs rwwwrse-app --tail 50

# Restart deployment
docker compose -f docker-compose.deploy.yml restart

# Stop deployment
docker compose -f docker-compose.deploy.yml down

# View Docker Compose logs
docker compose -f docker-compose.deploy.yml logs
```

## Security Considerations

1. **Network Security:**
   - Configure firewall rules appropriately
   - Use HTTPS in production with proper certificates
   - Restrict access to management ports

2. **Secrets Management:**
   - Use GitHub Secrets for sensitive configuration
   - Consider using Doppler or similar for secret management
   - Rotate secrets regularly

3. **Container Security:**
   - Application runs as non-root user
   - Minimal base image (Alpine Linux)
   - Regular security updates

## Support

For issues specific to this deployment setup, check:
1. GitHub Actions workflow logs
2. Synology Docker logs
3. Application health endpoints
4. rwwwrse project documentation