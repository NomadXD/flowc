#!/bin/bash
set -e

# Create the full resource hierarchy in FlowC:
#   Gateway -> Listener -> VirtualHost -> API + Deployment
#
# Prerequisites:
#   - FlowC running on localhost:8080
#   - Envoy running via run-envoy.sh (optional, for proxy testing)

FLOWC_URL="${FLOWC_URL:-http://localhost:8080}"

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

echo "=== FlowC E2E Setup ==="
echo "API: $FLOWC_URL"
echo ""

# 1. Create Gateway (matches Envoy node ID in envoy-bootstrap.yaml)
echo "--- Creating Gateway ---"
curl -s -X PUT "$FLOWC_URL/api/v1/gateways/e2e-gateway" \
  -H "Content-Type: application/json" \
  -d '{
    "spec": {
      "nodeId": "e2e-gateway"
    }
  }' | jq .
echo ""

# 2. Create Listener (port 9095 to match Envoy's exposed port)
echo "--- Creating Listener ---"
curl -s -X PUT "$FLOWC_URL/api/v1/listeners/e2e-listener" \
  -H "Content-Type: application/json" \
  -d '{
    "spec": {
      "gatewayRef": "e2e-gateway",
      "port": 9095,
      "address": "0.0.0.0"
    }
  }' | jq .
echo ""

# 3. Create VirtualHost
echo "--- Creating VirtualHost ---"
curl -s -X PUT "$FLOWC_URL/api/v1/virtualhosts/e2e-vhost" \
  -H "Content-Type: application/json" \
  -d '{
    "spec": {
      "gatewayRef": "e2e-gateway",
      "listenerRef": "e2e-listener",
      "hostname": "api.example.com"
    }
  }' | jq .
echo ""

# 4. Create API with inline OpenAPI spec
echo "--- Creating API ---"
SPEC_CONTENT=$(cat "$SCRIPT_DIR/openapi.yaml")
# Build JSON payload with jq to safely embed the YAML string
PAYLOAD=$(jq -n --arg spec "$SPEC_CONTENT" '{
  spec: {
    version: "1.0.0",
    context: "/httpbin",
    apiType: "rest",
    specContent: $spec,
    upstream: {
      host: "httpbin.org",
      port: 443,
      scheme: "https",
      timeout: "30s"
    }
  }
}')
curl -s -X PUT "$FLOWC_URL/api/v1/apis/httpbin-api" \
  -H "Content-Type: application/json" \
  -d "$PAYLOAD" | jq .
echo ""

# 5. Create Deployment (binds API to Gateway — listenerRef and virtualHostRef
#    are optional and auto-resolved when the gateway has exactly one of each)
echo "--- Creating Deployment ---"
curl -s -X PUT "$FLOWC_URL/api/v1/deployments/httpbin-deploy" \
  -H "Content-Type: application/json" \
  -d '{
    "spec": {
      "apiRef": "httpbin-api",
      "gatewayRef": "e2e-gateway"
    }
  }' | jq .
echo ""

# Wait for reconciliation
echo "Waiting for reconciliation (200ms)..."
sleep 0.5

# 6. Check deployment status
echo "--- Deployment Status ---"
curl -s "$FLOWC_URL/api/v1/deployments/httpbin-deploy" | jq .
echo ""

echo "=== Setup Complete ==="
echo ""
echo "If Envoy is running (./run-envoy.sh), test with:"
echo "  curl -H 'Host: api.example.com' http://localhost:9095/httpbin/get"
echo "  curl -H 'Host: api.example.com' http://localhost:9095/httpbin/headers"
