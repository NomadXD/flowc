#!/bin/bash

# FlowC API Deployment Script
# This script creates a deployment zip file and uploads it to FlowC

set -e

FLOWC_API_URL="${FLOWC_API_URL:-http://localhost:8080}"
NODE_ID="${NODE_ID:-test-envoy-node}"
DESCRIPTION="${DESCRIPTION:-Pet Store API deployment example}"

echo "Creating API deployment package..."

# Create zip file with required files
zip -q api-deployment.zip flowc.yaml openapi.yaml

echo "Zip file created: api-deployment.zip"

# Validate zip file first
echo "Validating zip file..."
VALIDATION_RESPONSE=$(curl -s -X POST "$FLOWC_API_URL/api/v1/validate" \
  -F "file=@api-deployment.zip")

echo "Validation response: $VALIDATION_RESPONSE"

# Check if validation was successful
if echo "$VALIDATION_RESPONSE" | grep -q '"success":true'; then
    echo "✓ Zip file validation passed"
else
    echo "✗ Zip file validation failed"
    echo "$VALIDATION_RESPONSE"
    exit 1
fi

# Deploy the API
echo "Deploying API to FlowC..."
DEPLOYMENT_RESPONSE=$(curl -s -X POST "$FLOWC_API_URL/api/v1/deployments" \
  -F "file=@api-deployment.zip" \
  -F "node_id=$NODE_ID" \
  -F "description=$DESCRIPTION")

echo "Deployment response:"
echo "$DEPLOYMENT_RESPONSE" | jq '.' 2>/dev/null || echo "$DEPLOYMENT_RESPONSE"

# Check if deployment was successful
if echo "$DEPLOYMENT_RESPONSE" | grep -q '"success":true'; then
    echo "✓ API deployed successfully"
    
    # Extract deployment ID
    DEPLOYMENT_ID=$(echo "$DEPLOYMENT_RESPONSE" | jq -r '.deployment.id' 2>/dev/null || echo "unknown")
    echo "Deployment ID: $DEPLOYMENT_ID"
    
    # Show deployment status
    echo "Checking deployment status..."
    curl -s "$FLOWC_API_URL/api/v1/deployments/$DEPLOYMENT_ID" | jq '.' 2>/dev/null || echo "Could not fetch deployment status"
    
else
    echo "✗ API deployment failed"
    echo "$DEPLOYMENT_RESPONSE"
    exit 1
fi

# Clean up
rm -f api-deployment.zip

echo "Deployment complete!"
echo ""
echo "You can now:"
echo "- List all deployments: curl $FLOWC_API_URL/api/v1/deployments"
echo "- Get deployment stats: curl $FLOWC_API_URL/api/v1/deployments/stats"
echo "- Check health: curl $FLOWC_API_URL/health"
