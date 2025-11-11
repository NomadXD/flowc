package main

import (
	"context"
	"fmt"

	"github.com/flowc-labs/flowc/internal/flowc/xds/translator"
	"github.com/flowc-labs/flowc/pkg/logger"
	"github.com/flowc-labs/flowc/pkg/types"
	"github.com/getkin/kin-openapi/openapi3"
)

// This example demonstrates the complete composite translator architecture
// showing how different strategies are composed together
func main() {
	// Create logger
	log := logger.NewDefaultEnvoyLogger()

	fmt.Println("=== FlowC Composite Translator Example ===\n")

	// =========================================================================
	// SCENARIO 1: Payment API with Custom Strategy Configuration
	// =========================================================================
	fmt.Println("--- Scenario 1: Payment API (Canary + Custom Strategies) ---")

	paymentMetadata := &types.FlowCMetadata{
		Name:    "payment-api",
		Version: "v2.0.0",
		Context: "/api/payments",
		Upstream: types.UpstreamConfig{
			Host:   "payment-service.internal",
			Port:   8080,
			Scheme: "https",
		},
	}

	paymentSpec := &openapi3.T{
		OpenAPI: "3.0.0",
		Info: &openapi3.Info{
			Title:   "Payment API",
			Version: "2.0.0",
		},
		Paths: openapi3.NewPaths(
			openapi3.WithPath("/process", &openapi3.PathItem{
				Post: &openapi3.Operation{Summary: "Process payment"},
			}),
			openapi3.WithPath("/refund", &openapi3.PathItem{
				Post: &openapi3.Operation{Summary: "Process refund"},
			}),
		),
	}

	// Configure strategies for payment API
	paymentConfig := &translator.XDSStrategyConfig{
		// Canary deployment - gradual rollout
		Deployment: &translator.DeploymentStrategyConfig{
			Type: "canary",
			Canary: &translator.CanaryConfig{
				BaselineVersion: "v1.0.0",
				CanaryVersion:   "v2.0.0",
				CanaryWeight:    10, // Start with 10% traffic
			},
		},
		// Exact path matching for security
		RouteMatching: &translator.RouteMatchStrategyConfig{
			Type:          "exact",
			CaseSensitive: true,
		},
		// Session affinity for payment flows
		LoadBalancing: &translator.LoadBalancingStrategyConfig{
			Type:       "consistent-hash",
			HashOn:     "header",
			HeaderName: "x-session-id",
		},
		// NO retry for payments (avoid double-charging!)
		Retry: &translator.RetryStrategyConfig{
			Type: "none",
		},
	}

	// Create deployment model with strategy config
	paymentModel := translator.NewDeploymentModel(paymentMetadata, paymentSpec, "deploy-payment-001")
	paymentModel.WithNodeID("envoy-gateway-1").
		WithStrategyConfig(paymentConfig)

	// Create translator using config resolver and factory
	paymentTranslator, err := createCompositeTranslator(paymentConfig, paymentModel, log)
	if err != nil {
		fmt.Printf("❌ Failed to create translator: %v\n", err)
		return
	}

	// Translate
	paymentResources, err := paymentTranslator.Translate(context.Background(), paymentModel)
	if err != nil {
		fmt.Printf("❌ Translation failed: %v\n", err)
		return
	}

	printTranslationResults("Payment API", paymentTranslator, paymentResources)
	fmt.Println()

	// =========================================================================
	// SCENARIO 2: User API with Default Strategies
	// =========================================================================
	fmt.Println("--- Scenario 2: User API (Basic + Aggressive Retry) ---")

	userMetadata := &types.FlowCMetadata{
		Name:    "user-api",
		Version: "v1.0.0",
		Context: "/api/users",
		Upstream: types.UpstreamConfig{
			Host:   "user-service.internal",
			Port:   8080,
			Scheme: "http",
		},
	}

	userSpec := &openapi3.T{
		OpenAPI: "3.0.0",
		Info: &openapi3.Info{
			Title:   "User API",
			Version: "1.0.0",
		},
		Paths: openapi3.NewPaths(
			openapi3.WithPath("/list", &openapi3.PathItem{
				Get: &openapi3.Operation{Summary: "List users"},
			}),
			openapi3.WithPath("/{id}", &openapi3.PathItem{
				Get:    &openapi3.Operation{Summary: "Get user"},
				Put:    &openapi3.Operation{Summary: "Update user"},
				Delete: &openapi3.Operation{Summary: "Delete user"},
			}),
		),
	}

	// Simpler config for user API
	userConfig := &translator.XDSStrategyConfig{
		// Basic deployment
		Deployment: &translator.DeploymentStrategyConfig{
			Type: "basic",
		},
		// Prefix matching (default)
		RouteMatching: &translator.RouteMatchStrategyConfig{
			Type: "prefix",
		},
		// Round-robin LB
		LoadBalancing: &translator.LoadBalancingStrategyConfig{
			Type: "round-robin",
		},
		// Aggressive retry OK for read-heavy API
		Retry: &translator.RetryStrategyConfig{
			Type: "aggressive",
		},
	}

	userModel := translator.NewDeploymentModel(userMetadata, userSpec, "deploy-user-001")
	userModel.WithNodeID("envoy-gateway-1").
		WithStrategyConfig(userConfig)

	userTranslator, err := createCompositeTranslator(userConfig, userModel, log)
	if err != nil {
		fmt.Printf("❌ Failed to create translator: %v\n", err)
		return
	}

	userResources, err := userTranslator.Translate(context.Background(), userModel)
	if err != nil {
		fmt.Printf("❌ Translation failed: %v\n", err)
		return
	}

	printTranslationResults("User API", userTranslator, userResources)
	fmt.Println()

	// =========================================================================
	// SCENARIO 3: Order API with Blue-Green Deployment
	// =========================================================================
	fmt.Println("--- Scenario 3: Order API (Blue-Green + Conservative Retry) ---")

	orderMetadata := &types.FlowCMetadata{
		Name:    "order-api",
		Version: "v2.0.0",
		Context: "/api/orders",
		Upstream: types.UpstreamConfig{
			Host:   "order-service.internal",
			Port:   8080,
			Scheme: "http",
		},
	}

	orderSpec := &openapi3.T{
		OpenAPI: "3.0.0",
		Info: &openapi3.Info{
			Title:   "Order API",
			Version: "2.0.0",
		},
		Paths: openapi3.NewPaths(
			openapi3.WithPath("/create", &openapi3.PathItem{
				Post: &openapi3.Operation{Summary: "Create order"},
			}),
			openapi3.WithPath("/{id}", &openapi3.PathItem{
				Get: &openapi3.Operation{Summary: "Get order"},
			}),
		),
	}

	orderConfig := &translator.XDSStrategyConfig{
		// Blue-green deployment
		Deployment: &translator.DeploymentStrategyConfig{
			Type: "blue-green",
			BlueGreen: &translator.BlueGreenConfig{
				ActiveVersion:  "v1.0.0",
				StandbyVersion: "v2.0.0",
				AutoPromote:    false,
			},
		},
		RouteMatching: &translator.RouteMatchStrategyConfig{
			Type: "prefix",
		},
		LoadBalancing: &translator.LoadBalancingStrategyConfig{
			Type:        "least-request",
			ChoiceCount: 2,
		},
		Retry: &translator.RetryStrategyConfig{
			Type: "conservative",
		},
	}

	orderModel := translator.NewDeploymentModel(orderMetadata, orderSpec, "deploy-order-001")
	orderModel.WithNodeID("envoy-gateway-1").
		WithStrategyConfig(orderConfig)

	orderTranslator, err := createCompositeTranslator(orderConfig, orderModel, log)
	if err != nil {
		fmt.Printf("❌ Failed to create translator: %v\n", err)
		return
	}

	orderResources, err := orderTranslator.Translate(context.Background(), orderModel)
	if err != nil {
		fmt.Printf("❌ Translation failed: %v\n", err)
		return
	}

	printTranslationResults("Order API", orderTranslator, orderResources)
	fmt.Println()

	// =========================================================================
	// SUMMARY
	// =========================================================================
	fmt.Println("=== Summary ===")
	fmt.Println("✅ Successfully demonstrated composite translator architecture")
	fmt.Println("✅ Each API uses different strategy combinations:")
	fmt.Println("   • Payment: Canary + Exact Match + Consistent Hash + No Retry")
	fmt.Println("   • User:    Basic + Prefix Match + Round Robin + Aggressive Retry")
	fmt.Println("   • Order:   Blue-Green + Prefix Match + Least Request + Conservative Retry")
	fmt.Println("\n✅ All strategies are independently configurable and composable!")
}

// createCompositeTranslator creates a composite translator from configuration
func createCompositeTranslator(config *translator.XDSStrategyConfig, model *translator.DeploymentModel, log *logger.EnvoyLogger) (*translator.CompositeTranslator, error) {
	// Resolve configuration (apply gateway defaults if needed)
	// For this example, we're using API-specific config directly
	resolver := translator.NewConfigResolver(nil, log)
	resolvedConfig := resolver.Resolve(config)

	// Create strategy factory
	factory := translator.NewStrategyFactory(translator.DefaultTranslatorOptions(), log)

	// Create strategy set from resolved config
	strategies, err := factory.CreateStrategySet(resolvedConfig, model)
	if err != nil {
		return nil, fmt.Errorf("failed to create strategy set: %w", err)
	}

	// Create composite translator
	return translator.NewCompositeTranslator(strategies, translator.DefaultTranslatorOptions(), log)
}

// printTranslationResults prints the results in a nice format
func printTranslationResults(apiName string, t *translator.CompositeTranslator, resources *translator.XDSResources) {
	fmt.Printf("✅ %s Translation Complete\n", apiName)
	fmt.Printf("   Translator: %s\n", t.Name())
	fmt.Printf("   Resources Generated:\n")
	fmt.Printf("     • Clusters:  %d\n", len(resources.Clusters))
	for _, cluster := range resources.Clusters {
		fmt.Printf("       - %s\n", cluster.Name)
	}
	fmt.Printf("     • Routes:    %d\n", len(resources.Routes))
	for _, route := range resources.Routes {
		fmt.Printf("       - %s (%d virtual hosts)\n", route.Name, len(route.VirtualHosts))
	}
	fmt.Printf("     • Listeners: %d\n", len(resources.Listeners))
	fmt.Printf("     • Endpoints: %d\n", len(resources.Endpoints))
}
