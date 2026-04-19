#!/bin/bash

# Deploy Example API Script
# This script prepares and deploys the example petstore API to FlowC
# It automatically discovers the gateway hierarchy (Gateway → Listener → Environment)
# and updates the flowc.yaml accordingly before deployment.

set -e

# Configuration
API_HOST="${FLOWC_API_HOST:-localhost}"
API_PORT="${FLOWC_API_PORT:-8080}"
BASE_URL="http://${API_HOST}:${API_PORT}/api/v1"

EXAMPLE_DIR="examples/api-deployment"
TEMP_DIR="/tmp/flowc-deploy-$$"

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}FlowC Example API Deployment${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# Check if FlowC API is running
echo -e "${YELLOW}Checking if FlowC API is running...${NC}"
if ! curl -s -f "${BASE_URL%/api/v1}/health" > /dev/null 2>&1; then
    echo -e "${RED}Error: FlowC API is not running at ${BASE_URL%/api/v1}${NC}"
    echo -e "${YELLOW}Please start FlowC first: ./flowc${NC}"
    exit 1
fi
echo -e "${GREEN}✓ FlowC API is running${NC}"
echo ""

# Step 1: Get Gateway Information
echo -e "${YELLOW}Step 1: Fetching gateway information...${NC}"
GATEWAYS_RESPONSE=$(curl -s "${BASE_URL}/gateways")

if ! echo "$GATEWAYS_RESPONSE" | grep -q '"success":true'; then
    echo -e "${RED}Error: Failed to fetch gateways${NC}"
    echo "$GATEWAYS_RESPONSE"
    exit 1
fi

GATEWAY_COUNT=$(echo "$GATEWAYS_RESPONSE" | grep -o '"total":[0-9]*' | cut -d':' -f2)

if [ "$GATEWAY_COUNT" -eq "0" ]; then
    echo -e "${RED}Error: No gateways found${NC}"
    echo -e "${YELLOW}Please create a gateway first using: ./scripts/setup-gateway.sh${NC}"
    exit 1
fi

# Extract gateway details
GATEWAY_ID=$(echo "$GATEWAYS_RESPONSE" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
GATEWAY_NAME=$(echo "$GATEWAYS_RESPONSE" | grep -o '"name":"[^"]*"' | head -1 | cut -d'"' -f4)
NODE_ID=$(echo "$GATEWAYS_RESPONSE" | grep -o '"node_id":"[^"]*"' | head -1 | cut -d'"' -f4)

echo -e "${GREEN}✓ Found gateway: ${GATEWAY_NAME}${NC}"
echo -e "  Gateway ID: ${GATEWAY_ID}"
echo -e "  Node ID: ${NODE_ID}"
echo ""

# Step 2: Get Listener Information
echo -e "${YELLOW}Step 2: Fetching listener information from gateway...${NC}"
LISTENERS_RESPONSE=$(curl -s "${BASE_URL}/gateways/${GATEWAY_ID}/listeners")

if ! echo "$LISTENERS_RESPONSE" | grep -q '"success":true'; then
    echo -e "${RED}Error: Failed to fetch listeners for gateway ${GATEWAY_ID}${NC}"
    echo "$LISTENERS_RESPONSE"
    exit 1
fi

LISTENER_COUNT=$(echo "$LISTENERS_RESPONSE" | grep -o '"total":[0-9]*' | cut -d':' -f2)

if [ "$LISTENER_COUNT" -eq "0" ]; then
    echo -e "${RED}Error: No listeners found on gateway ${GATEWAY_NAME}${NC}"
    echo -e "${YELLOW}Please create a listener first using: ./scripts/setup-gateway.sh${NC}"
    exit 1
fi

# Extract listener details
LISTENER_PORT=$(echo "$LISTENERS_RESPONSE" | grep -o '"port":[0-9]*' | head -1 | cut -d':' -f2)
LISTENER_ID=$(echo "$LISTENERS_RESPONSE" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)

echo -e "${GREEN}✓ Found listener${NC}"
echo -e "  Listener ID: ${LISTENER_ID}"
echo -e "  Port: ${LISTENER_PORT}"
echo ""

# Step 3: Get Environment Information
echo -e "${YELLOW}Step 3: Fetching environment information from listener...${NC}"
ENVIRONMENTS_RESPONSE=$(curl -s "${BASE_URL}/gateways/${GATEWAY_ID}/listeners/${LISTENER_PORT}/environments")

if ! echo "$ENVIRONMENTS_RESPONSE" | grep -q '"success":true'; then
    echo -e "${RED}Error: Failed to fetch environments for listener ${LISTENER_PORT}${NC}"
    echo "$ENVIRONMENTS_RESPONSE"
    exit 1
fi

ENV_COUNT=$(echo "$ENVIRONMENTS_RESPONSE" | grep -o '"total":[0-9]*' | cut -d':' -f2)

if [ "$ENV_COUNT" -eq "0" ]; then
    echo -e "${RED}Error: No environments found on listener (port ${LISTENER_PORT})${NC}"
    echo -e "${YELLOW}Please create an environment first using: ./scripts/setup-gateway.sh${NC}"
    exit 1
fi

# Extract environment details
ENV_ID=$(echo "$ENVIRONMENTS_RESPONSE" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
ENV_NAME=$(echo "$ENVIRONMENTS_RESPONSE" | grep -o '"name":"[^"]*"' | head -1 | cut -d'"' -f4)
ENV_HOSTNAME=$(echo "$ENVIRONMENTS_RESPONSE" | grep -o '"hostname":"[^"]*"' | head -1 | cut -d'"' -f4)

echo -e "${GREEN}✓ Found environment${NC}"
echo -e "  Environment ID: ${ENV_ID}"
echo -e "  Name: ${ENV_NAME}"
echo -e "  Hostname: ${ENV_HOSTNAME}"
echo ""

echo -e "${BLUE}Gateway Hierarchy:${NC}"
echo -e "  Gateway: ${GATEWAY_NAME} (${GATEWAY_ID})"
echo -e "    └─ Listener: Port ${LISTENER_PORT} (${LISTENER_ID})"
echo -e "       └─ Environment: ${ENV_NAME} @ ${ENV_HOSTNAME} (${ENV_ID})"
echo ""

# Step 4: Prepare Deployment Bundle
echo -e "${YELLOW}Step 4: Preparing deployment bundle...${NC}"

# Create temp directory
mkdir -p "$TEMP_DIR"
trap "rm -rf $TEMP_DIR" EXIT

# Copy files to temp directory
cp "${EXAMPLE_DIR}/openapi.yaml" "$TEMP_DIR/"
cp "${EXAMPLE_DIR}/flowc.yaml" "$TEMP_DIR/flowc.yaml.template"

# Update flowc.yaml with actual gateway hierarchy details
sed -e "s/gateway_id: \"REPLACE_WITH_GATEWAY_ID\"/gateway_id: \"${GATEWAY_ID}\"/" \
    -e "s/port: 9095/port: ${LISTENER_PORT}/" \
    -e "s/environment: \"production\"/environment: \"${ENV_NAME}\"/" \
    -e "s/domains: \[\"api.example.com\"\]/domains: [\"${ENV_HOSTNAME}\"]/" \
    "$TEMP_DIR/flowc.yaml.template" > "$TEMP_DIR/flowc.yaml"

rm "$TEMP_DIR/flowc.yaml.template"

# Create zip bundle
cd "$TEMP_DIR"
ZIP_FILE="petstore-api.zip"
zip -q "$ZIP_FILE" flowc.yaml openapi.yaml

echo -e "${GREEN}✓ Bundle created: ${ZIP_FILE}${NC}"
echo ""

# Display the updated flowc.yaml (gateway section)
echo -e "${BLUE}Updated flowc.yaml (gateway section):${NC}"
echo -e "${YELLOW}---${NC}"
sed -n '/^gateway:/,/^upstream:/p' flowc.yaml | head -n -1 | sed 's/^/  /'
echo -e "${YELLOW}---${NC}"
echo ""

# Step 5: Deploy the API
echo -e "${YELLOW}Step 5: Deploying API to FlowC...${NC}"
DEPLOY_RESPONSE=$(curl -s -X POST "${BASE_URL}/deployments" \
  -F "file=@${ZIP_FILE}" \
  -F "description=Petstore API example deployment")

# Check deployment result
if echo "$DEPLOY_RESPONSE" | grep -q '"success":true'; then
    DEPLOYMENT_ID=$(echo "$DEPLOY_RESPONSE" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
    DEPLOYMENT_NAME=$(echo "$DEPLOY_RESPONSE" | grep -o '"name":"[^"]*"' | head -1 | cut -d'"' -f4)
    DEPLOYMENT_VERSION=$(echo "$DEPLOY_RESPONSE" | grep -o '"version":"[^"]*"' | head -1 | cut -d'"' -f4)
    DEPLOYMENT_STATUS=$(echo "$DEPLOY_RESPONSE" | grep -o '"status":"[^"]*"' | head -1 | cut -d'"' -f4)

    echo -e "${GREEN}✓ API deployed successfully!${NC}"
    echo ""
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}Deployment Details${NC}"
    echo -e "${BLUE}========================================${NC}"
    echo -e "  ID: ${DEPLOYMENT_ID}"
    echo -e "  Name: ${DEPLOYMENT_NAME}"
    echo -e "  Version: ${DEPLOYMENT_VERSION}"
    echo -e "  Status: ${DEPLOYMENT_STATUS}"
    echo ""
    echo -e "${BLUE}Deployment Target:${NC}"
    echo -e "  Gateway: ${GATEWAY_NAME} (${NODE_ID})"
    echo -e "  Listener: Port ${LISTENER_PORT}"
    echo -e "  Environment: ${ENV_NAME}"
    echo -e "  Hostname: ${ENV_HOSTNAME}"
    echo ""
    echo -e "${BLUE}API Endpoint:${NC}"
    echo -e "  Base URL: http://${ENV_HOSTNAME}/petstore"
    echo ""
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}Test the API${NC}"
    echo -e "${BLUE}========================================${NC}"
    echo ""
    echo -e "${YELLOW}# Test with Host header (recommended):${NC}"
    echo -e "  curl -H \"Host: ${ENV_HOSTNAME}\" http://localhost:${LISTENER_PORT}/petstore/v2/pet/1"
    echo ""
    echo -e "${YELLOW}# Get pet by ID:${NC}"
    echo -e "  curl -H \"Host: ${ENV_HOSTNAME}\" http://localhost:${LISTENER_PORT}/petstore/v2/pet/123"
    echo ""
    echo -e "${YELLOW}# List pets:${NC}"
    echo -e "  curl -H \"Host: ${ENV_HOSTNAME}\" http://localhost:${LISTENER_PORT}/petstore/v2/pet/findByStatus?status=available"
    echo ""
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}Management Commands${NC}"
    echo -e "${BLUE}========================================${NC}"
    echo ""
    echo -e "${YELLOW}# Check deployment:${NC}"
    echo -e "  curl http://localhost:${API_PORT}/api/v1/deployments/${DEPLOYMENT_ID}"
    echo ""
    echo -e "${YELLOW}# List all deployments:${NC}"
    echo -e "  curl http://localhost:${API_PORT}/api/v1/deployments"
    echo ""
    echo -e "${YELLOW}# List APIs in this environment:${NC}"
    echo -e "  curl http://localhost:${API_PORT}/api/v1/gateways/${GATEWAY_ID}/listeners/${LISTENER_PORT}/environments/${ENV_NAME}/deployments"
    echo ""
    echo -e "${YELLOW}# Update deployment:${NC}"
    echo -e "  curl -X PUT http://localhost:${API_PORT}/api/v1/deployments/${DEPLOYMENT_ID} -F \"file=@new-version.zip\""
    echo ""
    echo -e "${YELLOW}# Delete deployment:${NC}"
    echo -e "  curl -X DELETE http://localhost:${API_PORT}/api/v1/deployments/${DEPLOYMENT_ID}"
    echo ""
else
    echo -e "${RED}Error: Deployment failed${NC}"
    echo ""
    echo -e "${YELLOW}Response:${NC}"
    echo "$DEPLOY_RESPONSE" | jq '.' 2>/dev/null || echo "$DEPLOY_RESPONSE"
    exit 1
fi

echo -e "${BLUE}========================================${NC}"
echo -e "${GREEN}Deployment Complete!${NC}"
echo -e "${BLUE}========================================${NC}"
