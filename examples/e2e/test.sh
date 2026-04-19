#!/bin/bash
set -e

# Test the deployed API through the Envoy proxy.
# Prerequisites: FlowC + Envoy running, setup.sh already executed.

ENVOY_URL="${ENVOY_URL:-http://localhost:9095}"
FLOWC_URL="${FLOWC_URL:-http://localhost:8080}"
HOST_HEADER="api.example.com"

PASS=0
FAIL=0

check() {
  local desc="$1"
  local url="$2"
  local expected_status="$3"

  status=$(curl -s -o /dev/null -w "%{http_code}" -H "Host: $HOST_HEADER" "$url" 2>/dev/null || echo "000")
  if [ "$status" = "$expected_status" ]; then
    echo "  PASS  $desc (HTTP $status)"
    PASS=$((PASS + 1))
  else
    echo "  FAIL  $desc (expected $expected_status, got $status)"
    FAIL=$((FAIL + 1))
  fi
}

echo "=== FlowC E2E Tests ==="
echo ""

# Control plane health
echo "-- Control Plane --"
check "Health check" "$FLOWC_URL/health" "200"
echo ""

# Envoy admin
echo "-- Envoy Admin --"
check "Envoy admin ready" "http://localhost:9901/ready" "200"
echo ""

# Proxy tests through Envoy
echo "-- Proxy (via Envoy) --"
check "GET /httpbin/get" "$ENVOY_URL/httpbin/get" "200"
check "GET /httpbin/headers" "$ENVOY_URL/httpbin/headers" "200"
check "GET /httpbin/status/200" "$ENVOY_URL/httpbin/status/200" "200"
check "GET /httpbin/status/404" "$ENVOY_URL/httpbin/status/404" "404"
echo ""

# Verbose output for one request
echo "-- Sample Response --"
curl -s -H "Host: $HOST_HEADER" "$ENVOY_URL/httpbin/get" 2>/dev/null | jq . 2>/dev/null || echo "(no response or invalid JSON)"
echo ""

# Summary
TOTAL=$((PASS + FAIL))
echo "=== Results: $PASS/$TOTAL passed ==="
if [ "$FAIL" -gt 0 ]; then
  exit 1
fi
