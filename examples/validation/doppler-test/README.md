# Doppler Integration Validation Tests

This directory contains configurations to test and validate the Doppler integration with rwwwrse.

## Overview

These test configurations allow you to verify that:

1. The rwwwrse container can authenticate with Doppler
2. Secrets are properly loaded from Doppler at startup
3. The application correctly uses secrets from Doppler
4. The fallback to environment variables works as expected

## Prerequisites

- Docker and Docker Compose installed
- A Doppler account and project set up
- A Doppler service token with appropriate permissions

## Setup

1. First, set up your Doppler project and configuration:

```bash
# Create a new project in Doppler (if not already created)
doppler projects create rwwwrse-test

# Create configs for different environments
doppler configs create dev --project rwwwrse-test
```

2. Add some test secrets to your Doppler project:

```bash
# Add test secrets to Doppler
doppler secrets set RWWWRSE_TEST_SECRET="this-is-from-doppler" \
  RWWWRSE_SECURITY_API_KEYS='["secret-key-1", "secret-key-2"]' \
  RWWWRSE_TLS_CERT_EMAIL="secure@example.com" \
  --project rwwwrse-test --config dev
```

3. Create a service token for testing:

```bash
# Create a service token for this test
doppler configs tokens create test-token \
  --project rwwwrse-test \
  --config dev \
  --plain
```

4. Copy the `.env.example` file to `.env` and add your token:

```bash
cp .env.example .env
# Edit the .env file to add your Doppler token
```

## Running the Tests

Start the Docker Compose environment:

```bash
docker-compose up
```

This will:
1. Start an echo server for testing backend routing
2. Start rwwwrse with Doppler integration
3. Display environment variables to verify Doppler secrets were loaded

## Validation Checks

### 1. Verify Doppler Secrets Are Loaded

Look for output like:

```
rwwwrse-doppler-test | RWWWRSE_TEST_SECRET=this-is-from-doppler
```

This confirms that secrets were successfully loaded from Doppler.

### 2. Test API Functionality

Once the containers are running, you can test the API by making requests:

```bash
curl -H "Host: localhost" http://localhost:8080/
```

This should route to the echo server, confirming the proxy works with the loaded configuration.

### 3. Check Metrics Endpoint

Verify metrics are working:

```bash
curl http://localhost:8080/metrics
```

## Common Issues

### Authentication Failure

If you see errors like:

```
Failed to fetch secrets from Doppler: authentication failed
```

Check:
- Your Doppler token is correct
- The token has permissions for the project and config
- The project and config names match what's in your `.env` file

### Missing Secrets

If the application starts but certain secrets are missing:

- Verify they're set in Doppler with the exact names expected
- Check if environment variable overrides might be taking precedence
- Ensure the secrets were set in the correct project and config

## Clean Up

Stop the containers when done:

```bash
docker-compose down
```

Optionally, revoke the test token:

```bash
doppler configs tokens revoke [TOKEN_ID] --project rwwwrse-test