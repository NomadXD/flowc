# API Deployment Example

This directory contains example files for deploying an API using FlowC.

## Files

- `flowc.yaml` - FlowC metadata configuration
- `openapi.yaml` - OpenAPI 3.0 specification
- `create-deployment.sh` - Script to create a deployment zip and upload it

## Usage

1. Create a zip file containing both YAML files:
```bash
zip api-deployment.zip flowc.yaml openapi.yaml
```

2. Deploy the API using curl:
```bash
curl -X POST http://localhost:8080/api/v1/deployments \
  -F "file=@api-deployment.zip" \
  -F "node_id=test-envoy-node" \
  -F "description=Pet Store API deployment"
```

3. Check deployment status:
```bash
curl http://localhost:8080/api/v1/deployments
```

## FlowC Configuration

The `flowc.yaml` file contains:
- **name**: API name
- **version**: API version
- **context**: URL path context (e.g., `/petstore`)
- **gateway**: Gateway configuration (host, port, TLS)
- **upstream**: Upstream service configuration
- **labels**: Optional metadata labels

## OpenAPI Specification

The `openapi.yaml` file contains the standard OpenAPI 3.0 specification with:
- API metadata (title, version, description)
- Server configurations
- Path definitions with operations
- Component schemas

## API Endpoints

Once deployed, the FlowC REST API provides:

- `POST /api/v1/deployments` - Deploy new API
- `GET /api/v1/deployments` - List all deployments
- `GET /api/v1/deployments/{id}` - Get specific deployment
- `PUT /api/v1/deployments/{id}` - Update deployment
- `DELETE /api/v1/deployments/{id}` - Delete deployment
- `GET /api/v1/deployments/stats` - Get deployment statistics
- `POST /api/v1/validate` - Validate zip file
- `GET /health` - Health check
