# Docker Compose with Doppler for Secrets Management

This example demonstrates how to integrate [Doppler](https://www.doppler.com/) for secrets management with the rwwwrse reverse proxy in a Docker Compose environment.

## Overview

Doppler is a SecretOps platform that helps manage secrets and configuration across environments. This example shows how to:

1. Configure the rwwwrse container to use Doppler for secrets
2. Set up environment variables for Doppler authentication
3. Override default configuration with secrets from Doppler

## Prerequisites

- Docker and Docker Compose installed
- A Doppler account and project created
- Doppler CLI installed locally (for setup)
- Git repository cloned locally (for building from source)

## Setup Doppler

1. Create a Doppler service token for your project:

   ```bash
   doppler configs tokens create rwwwrse-token --project rwwwrse --config dev --plain
   ```

2. Create a `.env` file in this directory with your Doppler token:

   ```bash
   DOPPLER_TOKEN=dp.st.your-token-here
   DOPPLER_PROJECT=rwwwrse
   DOPPLER_CONFIG=dev
   ```

3. Add your secrets to Doppler. Example secrets you might want to configure:

   ```bash
   RWWWRSE_SECURITY_API_KEYS=["secret-key-1", "secret-key-2"]
   RWWWRSE_TLS_CERT_EMAIL=your-email@example.com
   RWWWRSE_SECURITY_BASIC_AUTH_USERS={"admin": "hashed-password"}
   ```

## Building and Running Locally

This example now supports building the image locally instead of pulling from a repository. This is useful for development or when you don't have access to the container registry.

### Setup

1. Copy the provided `.env.example` file to `.env`:

   ```bash
   cp .env.example .env
   ```

2. Edit the `.env` file to add your Doppler token:

   ```bash
   # Open in your editor
   nano .env
   # Or
   vim .env
   ```

### Running the Example

Start the services with Docker Compose:

```bash
docker-compose up -d
```

The command will:
1. Build the rwwwrse image from the local Dockerfile
2. Start the containers with Doppler integration
3. Load secrets from Doppler at startup

To rebuild the image after code changes:

```bash
docker-compose build
docker-compose up -d
```

## How It Works

1. The rwwwrse Docker image includes the Doppler CLI.
2. When the container starts, it uses the Doppler token to fetch secrets.
3. These secrets are loaded as environment variables before the application starts.
4. Any configurations set in Doppler will override the default values in the Docker Compose file.

## Notes

- For production use, consider using a more secure method to provide the Doppler token, such as Docker secrets or a secrets management solution.
- Always use the appropriate Doppler configuration (dev, staging, prod) for each environment.
- Monitor your Doppler dashboard for activity and changes to secrets.
