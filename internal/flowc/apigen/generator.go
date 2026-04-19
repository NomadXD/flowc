package apigen

import "github.com/getkin/kin-openapi/openapi3"

// Generate builds the complete OpenAPI 3.0 specification for the flowc REST API.
func Generate() *openapi3.T {
	return &openapi3.T{
		OpenAPI: "3.0.3",
		Info: &openapi3.Info{
			Title: "FlowC Control Plane API",
			Description: "Declarative Envoy xDS control plane with reconciliation-based architecture. " +
				"Resources (Gateways, Listeners, APIs, Deployments, Policies) are declared " +
				"via this REST API and stored in a desired-state store.",
			Version: "3.0.0",
			License: &openapi3.License{
				Name: "Apache 2.0",
			},
		},
		Servers: openapi3.Servers{
			{URL: "http://localhost:8080", Description: "Local development server"},
		},
		Tags: openapi3.Tags{
			{Name: "Health", Description: "Health check endpoint"},
			{Name: "Gateways", Description: "Gateway resource operations"},
			{Name: "Listeners", Description: "Listener resource operations"},
			{Name: "APIs", Description: "API resource operations"},
			{Name: "Deployments", Description: "Deployment resource operations"},
			{Name: "GatewayPolicies", Description: "Gateway policy operations"},
			{Name: "APIPolicies", Description: "API policy operations"},
			{Name: "BackendPolicies", Description: "Backend policy operations"},
			{Name: "Apply", Description: "Bulk resource apply"},
			{Name: "Upload", Description: "ZIP bundle upload"},
			{Name: "Bootstrap", Description: "Envoy bootstrap configuration generation"},
			{Name: "Deploy", Description: "Gateway deployment instructions (Docker and Kubernetes)"},
		},
		Paths: buildPaths(),
		Components: &openapi3.Components{
			Schemas: buildSchemas(),
		},
	}
}
