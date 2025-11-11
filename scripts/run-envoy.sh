#!/bin/bash

# Run Envoy proxy with xDS configuration
# Make sure your control plane is running on localhost:18000

echo "Starting Envoy proxy connected to control plane at localhost:18000"
echo "Node ID: test-envoy-node"
echo "Admin interface will be available at: http://localhost:9901"
echo "Proxy will listen on port 10000 (as configured in your control plane)"
echo ""

docker run --rm -it \
  --name envoy-test \
  -p 9901:9901 \
  -p 9095:9095 \
  -v "$(pwd)/envoy-bootstrap.yaml:/etc/envoy/envoy.yaml:ro" \
  --add-host host.docker.internal:host-gateway \
  envoyproxy/envoy:v1.28-latest \
  -c /etc/envoy/envoy.yaml \
  --log-level info
