#!/bin/bash
# Script to validate Doppler integration with rwwwrse

set -e  # Exit on any error

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[0;33m'
NC='\033[0m' # No Color

echo "=== rwwwrse Doppler Integration Validation ==="
echo

# Check if .env file exists
if [ ! -f .env ]; then
    echo -e "${YELLOW}Warning: .env file not found. Creating from example...${NC}"
    cp .env.example .env
    echo -e "${RED}Please edit .env file to add your Doppler token!${NC}"
    exit 1
fi

# Source .env file
source .env

# Check if DOPPLER_TOKEN is set
if [ -z "$DOPPLER_TOKEN" ]; then
    echo -e "${RED}Error: DOPPLER_TOKEN is not set in .env file${NC}"
    exit 1
fi

# Check if Docker is running
if ! docker info > /dev/null 2>&1; then
    echo -e "${RED}Error: Docker is not running. Please start Docker first.${NC}"
    exit 1
fi

# Check if Doppler CLI is installed
if ! command -v doppler &> /dev/null; then
    echo -e "${RED}Error: Doppler CLI is not installed.${NC}"
    echo "Please install Doppler CLI: https://docs.doppler.com/docs/cli"
    exit 1
fi

echo "Checking Doppler authentication..."
if ! doppler me &> /dev/null; then
    echo -e "${RED}Error: Not authenticated with Doppler. Please run 'doppler login' first.${NC}"
    exit 1
fi
echo -e "${GREEN}✓ Doppler authentication successful${NC}"

# Set project and config
PROJECT=${DOPPLER_PROJECT:-rwwwrse}
CONFIG=${DOPPLER_CONFIG:-dev}

echo "Using Doppler project: $PROJECT, config: $CONFIG"

# Check if project exists, create if not
if ! doppler projects get $PROJECT &> /dev/null; then
    echo "Project $PROJECT does not exist. Creating it..."
    doppler projects create $PROJECT
    echo -e "${GREEN}✓ Created project $PROJECT${NC}"
else
    echo -e "${GREEN}✓ Project $PROJECT exists${NC}"
fi

# Check if config exists, create if not
if ! doppler configs get $CONFIG --project $PROJECT &> /dev/null; then
    echo "Config $CONFIG does not exist in project $PROJECT. Creating it..."
    doppler configs create $CONFIG --project $PROJECT
    echo -e "${GREEN}✓ Created config $CONFIG${NC}"
else
    echo -e "${GREEN}✓ Config $CONFIG exists${NC}"
fi

# Add test secret
echo "Adding test secret to Doppler..."
doppler secrets set RWWWRSE_TEST_SECRET="doppler-validation-$(date +%s)" \
    --project $PROJECT --config $CONFIG > /dev/null
echo -e "${GREEN}✓ Test secret added${NC}"

echo
echo "Starting Docker Compose environment..."
echo

# Create logs directory if it doesn't exist
mkdir -p logs

# Start Docker Compose in detached mode
docker-compose up -d

echo
echo "Waiting for services to start..."
sleep 5

# Check if containers are running
if ! docker-compose ps | grep "Up" > /dev/null; then
    echo -e "${RED}Error: Containers failed to start.${NC}"
    docker-compose logs
    docker-compose down
    exit 1
fi

echo -e "${GREEN}✓ Services started successfully${NC}"
echo

# Check logs for successful Doppler integration
echo "Checking container logs for Doppler secrets..."
CONTAINER_NAME=$(docker-compose ps -q rwwwrse-doppler-test)
if docker logs $CONTAINER_NAME 2>&1 | grep "RWWWRSE_TEST_SECRET=doppler-validation" > /dev/null; then
    echo -e "${GREEN}✓ Doppler secrets loaded successfully!${NC}"
    TEST_SUCCESS=true
else
    echo -e "${RED}✗ Doppler secrets not found in container logs${NC}"
    echo "Container logs:"
    docker logs $CONTAINER_NAME
    TEST_SUCCESS=false
fi

echo
echo "Testing API endpoint..."
RESPONSE=$(curl -s -o /dev/null -w "%{http_code}" -H "Host: localhost" http://localhost:8080/)
if [ "$RESPONSE" = "200" ]; then
    echo -e "${GREEN}✓ API endpoint working correctly${NC}"
else
    echo -e "${RED}✗ API endpoint returned status $RESPONSE${NC}"
    TEST_SUCCESS=false
fi

echo
if [ "$TEST_SUCCESS" = true ]; then
    echo -e "${GREEN}==== Validation SUCCESSFUL! ====${NC}"
    echo "Doppler integration is working correctly."
else
    echo -e "${RED}==== Validation FAILED! ====${NC}"
    echo "Doppler integration is not working correctly. Check the logs for details."
fi

echo
echo "Cleaning up..."
docker-compose down

echo
echo "Done."