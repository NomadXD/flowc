package apigen

import (
	"fmt"

	"github.com/getkin/kin-openapi/openapi3"
)

type resourcePathConfig struct {
	pluralName   string
	singularName string
	tag          string
	putReqRef    string
	responseRef  string
	listRef      string
}

// buildPaths returns all API path definitions.
func buildPaths() *openapi3.Paths {
	paths := openapi3.NewPaths()

	// Health
	paths.Set("/health", healthPath())

	// 6 resource kinds — same CRUD pattern
	for _, cfg := range []resourcePathConfig{
		{"gatewayprofiles", "GatewayProfile", "GatewayProfiles", "GatewayProfilePutRequest", "GatewayProfileResponse", "GatewayProfileListResponse"},
		{"gateways", "Gateway", "Gateways", "GatewayPutRequest", "GatewayResponse", "GatewayListResponse"},
		{"listeners", "Listener", "Listeners", "ListenerPutRequest", "ListenerResponse", "ListenerListResponse"},
		{"environments", "Environment", "Environments", "EnvironmentPutRequest", "EnvironmentResponse", "EnvironmentListResponse"},
		{"apis", "API", "APIs", "APIPutRequest", "APIResponse", "APIListResponse"},
		{"deployments", "Deployment", "Deployments", "DeploymentPutRequest", "DeploymentResponse", "DeploymentListResponse"},
	} {
		addResourcePaths(paths, cfg)
	}

	// Gateway bootstrap and deploy instructions
	paths.Set("/api/v1/gateways/{name}/bootstrap", bootstrapPath())
	paths.Set("/api/v1/gateways/{name}/deploy", deployPath())

	// Bulk apply
	paths.Set("/api/v1/apply", applyPath())

	// Upload
	paths.Set("/api/v1/upload", uploadPath())

	return paths
}

func addResourcePaths(paths *openapi3.Paths, cfg resourcePathConfig) {
	nameParam := &openapi3.ParameterRef{
		Value: openapi3.NewPathParameter("name").
			WithSchema(openapi3.NewStringSchema()).
			WithDescription("Resource name"),
	}

	// ── List: GET /api/v1/{plural} ──
	listPath := fmt.Sprintf("/api/v1/%s", cfg.pluralName)
	paths.Set(listPath, &openapi3.PathItem{
		Get: &openapi3.Operation{
			Tags:        []string{cfg.tag},
			Summary:     fmt.Sprintf("List %s", cfg.pluralName),
			OperationID: fmt.Sprintf("list%ss", cfg.singularName),
			Parameters: openapi3.Parameters{
				{Value: &openapi3.Parameter{
					Name:        "labels",
					In:          "query",
					Description: "Label filter (comma-separated key=value pairs)",
					Required:    false,
					Schema:      openapi3.NewSchemaRef("", openapi3.NewStringSchema()),
				}},
			},
			Responses: openapi3.NewResponses(
				openapi3.WithStatus(200, &openapi3.ResponseRef{
					Value: openapi3.NewResponse().
						WithDescription(fmt.Sprintf("List of %s", cfg.pluralName)).
						WithJSONSchemaRef(schemaRef(cfg.listRef)),
				}),
				openapi3.WithStatus(500, errorResponseRef()),
			),
		},
	})

	// ── Item: GET/PUT/DELETE /api/v1/{plural}/{name} ──
	itemPath := fmt.Sprintf("/api/v1/%s/{name}", cfg.pluralName)
	paths.Set(itemPath, &openapi3.PathItem{
		Parameters: openapi3.Parameters{nameParam},

		Get: &openapi3.Operation{
			Tags:        []string{cfg.tag},
			Summary:     fmt.Sprintf("Get a %s", cfg.singularName),
			OperationID: fmt.Sprintf("get%s", cfg.singularName),
			Responses: openapi3.NewResponses(
				openapi3.WithStatus(200, &openapi3.ResponseRef{
					Value: openapi3.NewResponse().
						WithDescription(fmt.Sprintf("The %s resource", cfg.singularName)).
						WithJSONSchemaRef(schemaRef(cfg.responseRef)),
				}),
				openapi3.WithStatus(404, errorResponseRef()),
				openapi3.WithStatus(500, errorResponseRef()),
			),
		},

		Put: &openapi3.Operation{
			Tags:        []string{cfg.tag},
			Summary:     fmt.Sprintf("Create or update a %s", cfg.singularName),
			OperationID: fmt.Sprintf("put%s", cfg.singularName),
			Parameters: openapi3.Parameters{
				ifMatchParam(),
				managedByParam(),
			},
			RequestBody: &openapi3.RequestBodyRef{
				Value: openapi3.NewRequestBody().
					WithRequired(true).
					WithJSONSchemaRef(schemaRef(cfg.putReqRef)),
			},
			Responses: openapi3.NewResponses(
				openapi3.WithStatus(200, &openapi3.ResponseRef{
					Value: openapi3.NewResponse().
						WithDescription("Resource updated").
						WithJSONSchemaRef(schemaRef(cfg.responseRef)),
				}),
				openapi3.WithStatus(201, &openapi3.ResponseRef{
					Value: openapi3.NewResponse().
						WithDescription("Resource created").
						WithJSONSchemaRef(schemaRef(cfg.responseRef)),
				}),
				openapi3.WithStatus(400, errorResponseRef()),
				openapi3.WithStatus(409, errorResponseRef()),
				openapi3.WithStatus(500, errorResponseRef()),
			),
		},

		Delete: &openapi3.Operation{
			Tags:        []string{cfg.tag},
			Summary:     fmt.Sprintf("Delete a %s", cfg.singularName),
			OperationID: fmt.Sprintf("delete%s", cfg.singularName),
			Parameters: openapi3.Parameters{
				ifMatchParam(),
			},
			Responses: openapi3.NewResponses(
				openapi3.WithStatus(200, &openapi3.ResponseRef{
					Value: openapi3.NewResponse().
						WithDescription("Resource deleted").
						WithJSONSchemaRef(schemaRef("DeleteResponse")),
				}),
				openapi3.WithStatus(404, errorResponseRef()),
				openapi3.WithStatus(409, errorResponseRef()),
				openapi3.WithStatus(500, errorResponseRef()),
			),
		},
	})
}

func healthPath() *openapi3.PathItem {
	return &openapi3.PathItem{
		Get: &openapi3.Operation{
			Tags:        []string{"Health"},
			Summary:     "Health check",
			OperationID: "healthCheck",
			Responses: openapi3.NewResponses(
				openapi3.WithStatus(200, &openapi3.ResponseRef{
					Value: openapi3.NewResponse().
						WithDescription("Service is healthy").
						WithJSONSchemaRef(schemaRef("HealthResponse")),
				}),
			),
		},
	}
}

func applyPath() *openapi3.PathItem {
	return &openapi3.PathItem{
		Post: &openapi3.Operation{
			Tags:        []string{"Apply"},
			Summary:     "Bulk apply resources",
			Description: "Creates or updates multiple resources in a single request.",
			OperationID: "applyResources",
			Parameters: openapi3.Parameters{
				managedByParam(),
			},
			RequestBody: &openapi3.RequestBodyRef{
				Value: openapi3.NewRequestBody().
					WithRequired(true).
					WithJSONSchemaRef(schemaRef("ApplyRequest")),
			},
			Responses: openapi3.NewResponses(
				openapi3.WithStatus(200, &openapi3.ResponseRef{
					Value: openapi3.NewResponse().
						WithDescription("Apply results").
						WithJSONSchemaRef(schemaRef("ApplyResult")),
				}),
				openapi3.WithStatus(400, errorResponseRef()),
				openapi3.WithStatus(500, errorResponseRef()),
			),
		},
	}
}

func bootstrapPath() *openapi3.PathItem {
	return &openapi3.PathItem{
		Parameters: openapi3.Parameters{
			{Value: openapi3.NewPathParameter("name").
				WithSchema(openapi3.NewStringSchema()).
				WithDescription("Gateway name")},
		},
		Get: &openapi3.Operation{
			Tags:        []string{"Bootstrap"},
			Summary:     "Generate Envoy bootstrap configuration",
			Description: "Generates an Envoy bootstrap YAML for the gateway, using the referenced profile's settings (admin port, xDS cluster, etc.).",
			OperationID: "getGatewayBootstrap",
			Responses: openapi3.NewResponses(
				openapi3.WithStatus(200, &openapi3.ResponseRef{
					Value: openapi3.NewResponse().
						WithDescription("Envoy bootstrap YAML").
						WithContent(openapi3.Content{
							"application/x-yaml": &openapi3.MediaType{
								Schema: openapi3.NewSchemaRef("", openapi3.NewStringSchema()),
							},
						}),
				}),
				openapi3.WithStatus(404, errorResponseRef()),
				openapi3.WithStatus(500, errorResponseRef()),
			),
		},
	}
}

func deployPath() *openapi3.PathItem {
	return &openapi3.PathItem{
		Parameters: openapi3.Parameters{
			{Value: openapi3.NewPathParameter("name").
				WithSchema(openapi3.NewStringSchema()).
				WithDescription("Gateway name")},
		},
		Get: &openapi3.Operation{
			Tags:        []string{"Deploy"},
			Summary:     "Get gateway deployment instructions",
			Description: "Returns Docker run command, Docker Compose snippet, and Kubernetes manifest for deploying the gateway's Envoy proxy.",
			OperationID: "getGatewayDeployInstructions",
			Responses: openapi3.NewResponses(
				openapi3.WithStatus(200, &openapi3.ResponseRef{
					Value: openapi3.NewResponse().
						WithDescription("Deployment instructions").
						WithJSONSchemaRef(schemaRef("DeployInstructions")),
				}),
				openapi3.WithStatus(404, errorResponseRef()),
				openapi3.WithStatus(500, errorResponseRef()),
			),
		},
	}
}

func uploadPath() *openapi3.PathItem {
	return &openapi3.PathItem{
		Post: &openapi3.Operation{
			Tags:    []string{"Upload"},
			Summary: "Upload API bundle",
			Description: "Uploads a ZIP bundle containing an API spec and flowc.yaml. " +
				"Creates an API resource and optionally a Deployment resource.",
			OperationID: "uploadBundle",
			Parameters: openapi3.Parameters{
				managedByParam(),
			},
			RequestBody: &openapi3.RequestBodyRef{
				Value: &openapi3.RequestBody{
					Required: true,
					Content: openapi3.Content{
						"multipart/form-data": &openapi3.MediaType{
							Schema: newObjectSchema(
								[]string{"file"},
								openapi3.Schemas{
									"file": openapi3.NewSchemaRef("", &openapi3.Schema{
										Type:   &openapi3.Types{openapi3.TypeString},
										Format: "binary",
									}),
								},
							),
						},
					},
				},
			},
			Responses: openapi3.NewResponses(
				openapi3.WithStatus(200, &openapi3.ResponseRef{
					Value: openapi3.NewResponse().
						WithDescription("Upload results").
						WithJSONSchemaRef(schemaRef("ApplyResult")),
				}),
				openapi3.WithStatus(400, errorResponseRef()),
				openapi3.WithStatus(500, errorResponseRef()),
			),
		},
	}
}
