#!/bin/bash

# Setup Gateway Hierarchy Script
# This script creates the gateway -> listener -> environment hierarchy
# required for deploying APIs to FlowC.
#
# The new API allows creating the entire hierarchy in a single request.

set -e

# Configuration
API_HOST="${FLOWC_API_HOST:-localhost}"
API_PORT="${FLOWC_API_PORT:-8080}"
BASE_URL="http://${API_HOST}:${API_PORT}/api/v1"

# Gateway configuration
GATEWAY_NODE_ID="${GATEWAY_NODE_ID:-test-envoy-node}"
GATEWAY_NAME="${GATEWAY_NAME:-Production Gateway}"
GATEWAY_DESCRIPTION="${GATEWAY_DESCRIPTION:-Main production gateway}"

# Listener configuration
LISTENER_PORT="${LISTENER_PORT:-9095}"
LISTENER_ADDRESS="${LISTENER_ADDRESS:-0.0.0.0}"
LISTENER_HTTP2="${LISTENER_HTTP2:-true}"

# Environment configuration
ENV_NAME="${ENV_NAME:-production}"
ENV_HOSTNAME="${ENV_HOSTNAME:-*}"
ENV_DESCRIPTION="${ENV_DESCRIPTION:-Production environment}"

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}FlowC Gateway Hierarchy Setup${NC}"
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

# Create Gateway with Listener and Environment in a single request
echo -e "${YELLOW}Creating Gateway with Listener and Environment...${NC}"
echo -e "${BLUE}Gateway Configuration:${NC}"
echo -e "  Node ID: ${GATEWAY_NODE_ID}"
echo -e "  Name: ${GATEWAY_NAME}"
echo -e "${BLUE}Listener Configuration:${NC}"
echo -e "  Port: ${LISTENER_PORT}"
echo -e "  Address: ${LISTENER_ADDRESS}"
echo -e "  HTTP/2: ${LISTENER_HTTP2}"
echo -e "${BLUE}Environment Configuration:${NC}"
echo -e "  Name: ${ENV_NAME}"
echo -e "  Hostname: ${ENV_HOSTNAME}"
echo ""

GATEWAY_RESPONSE=$(curl -s -X POST "${BASE_URL}/gateways" \
  -H "Content-Type: application/json" \
  -d "{
    \"node_id\": \"${GATEWAY_NODE_ID}\",
    \"name\": \"${GATEWAY_NAME}\",
    \"description\": \"${GATEWAY_DESCRIPTION}\",
    \"listeners\": [
      {
        \"port\": ${LISTENER_PORT},
        \"address\": \"${LISTENER_ADDRESS}\",
        \"http2\": ${LISTENER_HTTP2},
        \"environments\": [
          {
            \"name\": \"${ENV_NAME}\",
            \"hostname\": \"${ENV_HOSTNAME}\",
            \"description\": \"${ENV_DESCRIPTION}\"
          }
        ]
      }
    ]
  }")

# Check if gateway creation was successful
if echo "$GATEWAY_RESPONSE" | grep -q '"success":true'; then
    GATEWAY_ID=$(echo "$GATEWAY_RESPONSE" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
    echo -e "${GREEN}✓ Gateway hierarchy created successfully!${NC}"
    echo -e "  Gateway ID: ${GATEWAY_ID}"
elif echo "$GATEWAY_RESPONSE" | grep -q "already exists"; then
    echo -e "${YELLOW}⚠ Gateway already exists with node_id '${GATEWAY_NODE_ID}'${NC}"
    echo -e "${YELLOW}  The gateway may already have listeners and environments configured.${NC}"

    # Get existing gateway by listing all gateways
    GATEWAYS_LIST=$(curl -s "${BASE_URL}/gateways")
    GATEWAY_ID=$(echo "$GATEWAYS_LIST" | grep -B5 "\"node_id\":\"${GATEWAY_NODE_ID}\"" | grep -o "\"id\":\"[^\"]*\"" | head -1 | cut -d'"' -f4)

    if [ -z "$GATEWAY_ID" ]; then
        echo -e "${RED}Error: Failed to get existing gateway ID${NC}"
        exit 1
    fi
    echo -e "${GREEN}✓ Using existing gateway${NC}"
    echo -e "  Gateway ID: ${GATEWAY_ID}"
else
    echo -e "${RED}Error creating gateway hierarchy:${NC}"
    echo "$GATEWAY_RESPONSE" | jq '.' 2>/dev/null || echo "$GATEWAY_RESPONSE"
    exit 1
fi
echo ""

# Summary
echo -e "${BLUE}========================================${NC}"
echo -e "${GREEN}Gateway Hierarchy Setup Complete!${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""
echo -e "${BLUE}Created Hierarchy:${NC}"
echo -e "  Gateway: ${GATEWAY_NAME} (${GATEWAY_ID})"
echo -e "    └── Listener: ${LISTENER_ADDRESS}:${LISTENER_PORT} (HTTP/2: ${LISTENER_HTTP2})"
echo -e "          └── Environment: ${ENV_NAME} (hostname: ${ENV_HOSTNAME})"
echo ""
echo -e "${BLUE}What was created:${NC}"
echo -e "  • 1 Gateway with node_id: ${GATEWAY_NODE_ID}"
echo -e "  • 1 Listener on port ${LISTENER_PORT}"
echo -e "  • 1 Environment named '${ENV_NAME}' matching hostname '${ENV_HOSTNAME}'"
echo ""
echo -e "${BLUE}Next Steps:${NC}"
echo ""
echo -e "1. ${YELLOW}Update your flowc.yaml${NC} to reference this hierarchy:"
echo ""
echo -e "${YELLOW}gateway:"
echo -e "  gateway_id: \"${GATEWAY_ID}\""
echo -e "  port: ${LISTENER_PORT}"
echo -e "  environment: \"${ENV_NAME}\""
echo -e "  virtual_host:"
echo -e "    domains: [\"${ENV_HOSTNAME}\"]${NC}"
echo ""
echo -e "2. ${YELLOW}Deploy your API:${NC}"
echo -e "   curl -X POST http://${API_HOST}:${API_PORT}/api/v1/deployments \\"
echo -e "     -F \"file=@your-api.zip\""
echo ""
echo -e "3. ${YELLOW}Verify gateway configuration:${NC}"
echo -e "   curl http://${API_HOST}:${API_PORT}/api/v1/gateways/${GATEWAY_ID}"
echo ""
echo -e "4. ${YELLOW}Check Envoy configuration:${NC}"
echo -e "   curl http://localhost:9901/config_dump"
echo ""
echo -e "${BLUE}Additional Operations:${NC}"
echo ""
echo -e "• ${YELLOW}Add another environment:${NC}"
echo -e "  curl -X POST http://${API_HOST}:${API_PORT}/api/v1/gateways/${GATEWAY_ID}/listeners/${LISTENER_PORT}/environments \\"
echo -e "    -H \"Content-Type: application/json\" \\"
echo -e "    -d '{\"name\": \"staging\", \"hostname\": \"staging.example.com\"}'"
echo ""
echo -e "• ${YELLOW}Add another listener:${NC}"
echo -e "  curl -X POST http://${API_HOST}:${API_PORT}/api/v1/gateways/${GATEWAY_ID}/listeners \\"
echo -e "    -H \"Content-Type: application/json\" \\"
echo -e "    -d '{\"port\": 8443, \"environments\": [{\"name\": \"secure\", \"hostname\": \"*\"}]}'"
echo ""
