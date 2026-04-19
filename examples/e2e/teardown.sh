#!/bin/bash
set -e

# Delete all resources created by setup.sh (reverse order).

FLOWC_URL="${FLOWC_URL:-http://localhost:8080}"

echo "=== FlowC E2E Teardown ==="
echo ""

echo "Deleting deployment..."
curl -s -X DELETE "$FLOWC_URL/api/v1/deployments/httpbin-deploy" | jq .

echo "Deleting API..."
curl -s -X DELETE "$FLOWC_URL/api/v1/apis/httpbin-api" | jq .

echo "Deleting virtual host..."
curl -s -X DELETE "$FLOWC_URL/api/v1/virtualhosts/e2e-vhost" | jq .

echo "Deleting listener..."
curl -s -X DELETE "$FLOWC_URL/api/v1/listeners/e2e-listener" | jq .

echo "Deleting gateway..."
curl -s -X DELETE "$FLOWC_URL/api/v1/gateways/e2e-gateway" | jq .

echo ""
echo "=== Teardown Complete ==="
