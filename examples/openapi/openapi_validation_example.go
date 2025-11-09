package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/flowc-labs/flowc/pkg/logger"
	"github.com/flowc-labs/flowc/pkg/openapi"
)

// Example demonstrating how to use kin-openapi for validation in flowc

func main() {
	// Initialize logger
	logger := logger.NewDefaultEnvoyLogger()

	// Create OpenAPI manager
	openAPIManager := openapi.NewOpenAPIManager()

	// Load OpenAPI specification from file
	ctx := context.Background()
	doc, err := openAPIManager.LoadFromFile(ctx, "examples/api-deployment/openapi.yaml")
	if err != nil {
		log.Fatalf("Failed to load OpenAPI spec: %v", err)
	}

	// Create router for validation
	router, err := openAPIManager.CreateRouter(doc)
	if err != nil {
		log.Fatalf("Failed to create router: %v", err)
	}

	// Extract routes from the specification
	routes, err := openAPIManager.ExtractRoutes(doc)
	if err != nil {
		log.Fatalf("Failed to extract routes: %v", err)
	}

	fmt.Printf("Loaded OpenAPI spec with %d routes:\n", len(routes))
	for _, route := range routes {
		fmt.Printf("  %s %s - %s\n", route.Get.OperationID, route.Get.Summary, route.Get.Description)
	}

	// Example HTTP handler with validation
	http.HandleFunc("/validate", func(w http.ResponseWriter, r *http.Request) {
		// Validate the request against OpenAPI spec
		if err := openAPIManager.ValidateRequest(ctx, r, router); err != nil {
			logger.WithError(err).Error("Request validation failed")
			http.Error(w, fmt.Sprintf("Request validation failed: %v", err), http.StatusBadRequest)
			return
		}

		// Request is valid, process it
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Request is valid according to OpenAPI spec"))
	})

	// Example of finding an operation by ID
	operation, path, method, err := openAPIManager.GetOperationByID(doc, "listPets")
	if err != nil {
		log.Printf("Operation not found: %v", err)
	} else {
		fmt.Printf("Found operation 'listPets': %s %s - %s\n", method, path, operation.Summary)
	}

	// Example of getting a schema by reference
	schema, err := openAPIManager.GetSchemaByRef(doc, "#/components/schemas/Pet")
	if err != nil {
		log.Printf("Schema not found: %v", err)
	} else {
		fmt.Printf("Found Pet schema with type: %s\n", schema.Type)
	}

	fmt.Println("OpenAPI validation example completed successfully!")
}
