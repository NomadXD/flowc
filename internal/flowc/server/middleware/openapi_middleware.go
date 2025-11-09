package middleware

import (
	"context"
	"fmt"
	"net/http"

	"github.com/flowc-labs/flowc/pkg/logger"
	"github.com/flowc-labs/flowc/pkg/openapi"
	"github.com/getkin/kin-openapi/routers"
)

// OpenAPIValidationMiddleware provides request/response validation against OpenAPI specs
type OpenAPIValidationMiddleware struct {
	openAPIManager *openapi.OpenAPIManager
	logger         *logger.EnvoyLogger
	router         routers.Router
}

// NewOpenAPIValidationMiddleware creates a new OpenAPI validation middleware
func NewOpenAPIValidationMiddleware(logger *logger.EnvoyLogger) *OpenAPIValidationMiddleware {
	return &OpenAPIValidationMiddleware{
		openAPIManager: openapi.NewOpenAPIManager(),
		logger:         logger,
	}
}

// SetRouter sets the OpenAPI router for validation
func (m *OpenAPIValidationMiddleware) SetRouter(router routers.Router) {
	m.router = router
}

// ValidateRequest validates incoming HTTP requests against the OpenAPI specification
func (m *OpenAPIValidationMiddleware) ValidateRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip validation if no router is set
		if m.router == nil {
			next.ServeHTTP(w, r)
			return
		}

		// Validate the request
		ctx := context.Background()
		if err := m.openAPIManager.ValidateRequest(ctx, r, m.router); err != nil {
			m.logger.WithError(err).Error("Request validation failed")
			http.Error(w, fmt.Sprintf("Request validation failed: %v", err), http.StatusBadRequest)
			return
		}

		// Request is valid, continue to next handler
		next.ServeHTTP(w, r)
	})
}

// ValidateResponse validates outgoing HTTP responses against the OpenAPI specification
func (m *OpenAPIValidationMiddleware) ValidateResponse(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip validation if no router is set
		if m.router == nil {
			next.ServeHTTP(w, r)
			return
		}

		// Create a response recorder to capture the response
		recorder := &responseRecorder{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		// Call the next handler
		next.ServeHTTP(recorder, r)

		// Validate the response
		ctx := context.Background()
		resp := &http.Response{
			StatusCode: recorder.statusCode,
			Header:     recorder.Header(),
		}

		if err := m.openAPIManager.ValidateResponse(ctx, r, resp, m.router); err != nil {
			m.logger.WithError(err).Warn("Response validation failed")
			// Note: We don't return an error here as the response has already been sent
			// This is mainly for logging/monitoring purposes
		}
	})
}

// responseRecorder captures response data for validation
type responseRecorder struct {
	http.ResponseWriter
	statusCode int
}

// WriteHeader captures the status code
func (r *responseRecorder) WriteHeader(statusCode int) {
	r.statusCode = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

// Write delegates to the underlying ResponseWriter
func (r *responseRecorder) Write(data []byte) (int, error) {
	return r.ResponseWriter.Write(data)
}
