#!/bin/bash
set -e

# Start Envoy proxy connected to the FlowC xDS control plane.
# Prerequisite: FlowC must be running (go run ./cmd/flowc or ./flowc).

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

echo "Starting Envoy proxy"
echo "  xDS control plane: localhost:18000"
echo "  Node ID:           e2e-gateway"
echo "  Admin:             http://localhost:9901"
echo "  Proxy:             http://localhost:9095"
echo ""

docker run --rm -it \
  --name flowc-e2e-envoy \
  -p 9901:9901 \
  -p 9095:9095 \
  -v "$SCRIPT_DIR/envoy-bootstrap.yaml:/etc/envoy/envoy.yaml:ro" \
  --add-host host.docker.internal:host-gateway \
  envoyproxy/envoy:v1.37-latest \
  -c /etc/envoy/envoy.yaml \
  --log-level info
