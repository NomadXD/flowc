package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/flowc-labs/flowc/internal/flowc/resource"
	"github.com/flowc-labs/flowc/internal/flowc/resource/store"
	"github.com/flowc-labs/flowc/pkg/logger"
)

// ResourceHandler is the unified HTTP handler for all declarative resource operations.
type ResourceHandler struct {
	store  store.Store
	logger *logger.EnvoyLogger
}

// NewResourceHandler creates a new resource handler.
func NewResourceHandler(s store.Store, log *logger.EnvoyLogger) *ResourceHandler {
	return &ResourceHandler{store: s, logger: log}
}

// HandlePut handles PUT /api/v1/{kind-plural}/{name}
// Creates or updates a resource. Returns 201 for create, 200 for update.
func (h *ResourceHandler) HandlePut(kind resource.ResourceKind) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")

		body, err := io.ReadAll(r.Body)
		if err != nil {
			writeError(w, http.StatusBadRequest, "failed to read request body")
			return
		}

		// Parse the spec from the body
		var envelope struct {
			Spec   json.RawMessage `json:"spec"`
			Status json.RawMessage `json:"status,omitempty"`
		}
		if err := json.Unmarshal(body, &envelope); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
			return
		}
		if envelope.Spec == nil {
			// Allow full resource body without wrapper
			envelope.Spec = body
		}

		// Validate the typed resource
		if err := validateResource(kind, name, envelope.Spec); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}

		// Build stored resource
		meta := resource.ResourceMeta{
			Kind:   kind,
			Name:   name,
			Labels: extractLabels(body),
		}

		// Extract conflict policy from body
		var metaOverrides struct {
			Metadata struct {
				ConflictPolicy resource.ConflictPolicy `json:"conflictPolicy"`
			} `json:"metadata"`
		}
		json.Unmarshal(body, &metaOverrides)
		if metaOverrides.Metadata.ConflictPolicy != "" {
			meta.ConflictPolicy = metaOverrides.Metadata.ConflictPolicy
		}

		stored := &store.StoredResource{
			Meta:       meta,
			SpecJSON:   envelope.Spec,
			StatusJSON: envelope.Status,
		}

		opts := store.PutOptions{
			ManagedBy: r.Header.Get("X-Managed-By"),
		}

		// If-Match header for optimistic concurrency
		if ifMatch := r.Header.Get("If-Match"); ifMatch != "" {
			rev, err := strconv.ParseInt(ifMatch, 10, 64)
			if err == nil {
				opts.ExpectedRevision = rev
			}
		}

		out, err := h.store.Put(r.Context(), stored, opts)
		if err != nil {
			handleStoreError(w, err)
			return
		}

		status := http.StatusOK
		if out.Meta.Revision == 1 {
			status = http.StatusCreated
		}

		writeResourceResponse(w, status, out)
	}
}

// HandleGet handles GET /api/v1/{kind-plural}/{name}
func (h *ResourceHandler) HandleGet(kind resource.ResourceKind) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")

		key := resource.ResourceKey{Kind: kind, Name: name}
		res, err := h.store.Get(r.Context(), key)
		if err != nil {
			handleStoreError(w, err)
			return
		}

		writeResourceResponse(w, http.StatusOK, res)
	}
}

// HandleList handles GET /api/v1/{kind-plural}
// Supports query params: labels (metadata labels), gatewayRef, listenerRef, virtualHostRef (spec fields).
func (h *ResourceHandler) HandleList(kind resource.ResourceKind) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		filter := store.ListFilter{
			Kind:   kind,
			Labels: parseLabelsQuery(r),
		}

		items, err := h.store.List(r.Context(), filter)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		// Apply spec-field filters (gatewayRef, listenerRef, virtualHostRef).
		// These are post-filters applied after the store list since the store
		// only supports kind+label filtering.
		specFilters := parseSpecFilters(r)
		if len(specFilters) > 0 {
			items = filterBySpec(items, specFilters)
		}

		responses := make([]resource.Response, 0, len(items))
		for _, item := range items {
			responses = append(responses, resource.Response{
				Kind:     item.Meta.Kind,
				Metadata: item.Meta,
				Spec:     item.SpecJSON,
				Status:   item.StatusJSON,
			})
		}

		writeJSON(w, http.StatusOK, resource.ListResponse{
			Kind:  string(kind) + "List",
			Items: responses,
			Total: len(responses),
		})
	}
}

// HandleDelete handles DELETE /api/v1/{kind-plural}/{name}
func (h *ResourceHandler) HandleDelete(kind resource.ResourceKind) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")

		key := resource.ResourceKey{Kind: kind, Name: name}

		opts := store.DeleteOptions{}
		if ifMatch := r.Header.Get("If-Match"); ifMatch != "" {
			rev, err := strconv.ParseInt(ifMatch, 10, 64)
			if err == nil {
				opts.ExpectedRevision = rev
			}
		}

		if err := h.store.Delete(r.Context(), key, opts); err != nil {
			handleStoreError(w, err)
			return
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"message": fmt.Sprintf("%s %q deleted", kind, name),
		})
	}
}

// HandleApply handles POST /api/v1/apply — bulk create-or-update.
func (h *ResourceHandler) HandleApply(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to read request body")
		return
	}

	var req resource.ApplyRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	managedBy := r.Header.Get("X-Managed-By")
	var results []resource.ApplyResultItem

	for _, raw := range req.Resources {
		var envelope struct {
			Kind     resource.ResourceKind `json:"kind"`
			Metadata struct {
				Name           string                  `json:"name"`
				Labels         map[string]string       `json:"labels,omitempty"`
				ConflictPolicy resource.ConflictPolicy `json:"conflictPolicy,omitempty"`
			} `json:"metadata"`
			Spec   json.RawMessage `json:"spec"`
			Status json.RawMessage `json:"status,omitempty"`
		}
		if err := json.Unmarshal(raw, &envelope); err != nil {
			results = append(results, resource.ApplyResultItem{
				Action: "failed",
				Error:  "invalid resource: " + err.Error(),
			})
			continue
		}

		meta := resource.ResourceMeta{
			Kind:           envelope.Kind,
			Name:           envelope.Metadata.Name,
			Labels:         envelope.Metadata.Labels,
			ConflictPolicy: envelope.Metadata.ConflictPolicy,
		}

		stored := &store.StoredResource{
			Meta:       meta,
			SpecJSON:   envelope.Spec,
			StatusJSON: envelope.Status,
		}

		out, err := h.store.Put(r.Context(), stored, store.PutOptions{ManagedBy: managedBy})
		if err != nil {
			results = append(results, resource.ApplyResultItem{
				Kind:   envelope.Kind,
				Name:   envelope.Metadata.Name,
				Action: "failed",
				Error:  err.Error(),
			})
			continue
		}

		action := "updated"
		if out.Meta.Revision == 1 {
			action = "created"
		}
		results = append(results, resource.ApplyResultItem{
			Kind:   envelope.Kind,
			Name:   out.Meta.Name,
			Action: action,
		})
	}

	writeJSON(w, http.StatusOK, resource.ApplyResult{Results: results})
}

// HealthCheck handles GET /health
func (h *ResourceHandler) HealthCheck(startTime time.Time) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"status":    "healthy",
			"timestamp": time.Now(),
			"version":   "3.0.0",
			"uptime":    time.Since(startTime).String(),
		})
	}
}

// --- Helpers ---

func validateResource(kind resource.ResourceKind, name string, specJSON json.RawMessage) error {
	switch kind {
	case resource.KindGatewayProfile:
		var r resource.GatewayProfileResource
		r.Meta = resource.ResourceMeta{Name: name}
		if err := json.Unmarshal(specJSON, &r.Spec); err != nil {
			return fmt.Errorf("invalid gateway profile spec: %w", err)
		}
		return r.Validate()

	case resource.KindGateway:
		var r resource.GatewayResource
		r.Meta = resource.ResourceMeta{Name: name}
		if err := json.Unmarshal(specJSON, &r.Spec); err != nil {
			return fmt.Errorf("invalid gateway spec: %w", err)
		}
		return r.Validate()

	case resource.KindListener:
		var r resource.ListenerResource
		r.Meta = resource.ResourceMeta{Name: name}
		if err := json.Unmarshal(specJSON, &r.Spec); err != nil {
			return fmt.Errorf("invalid listener spec: %w", err)
		}
		return r.Validate()

	case resource.KindVirtualHost:
		var r resource.VirtualHostResource
		r.Meta = resource.ResourceMeta{Name: name}
		if err := json.Unmarshal(specJSON, &r.Spec); err != nil {
			return fmt.Errorf("invalid virtual host spec: %w", err)
		}
		return r.Validate()

	case resource.KindAPI:
		var r resource.APIResource
		r.Meta = resource.ResourceMeta{Name: name}
		if err := json.Unmarshal(specJSON, &r.Spec); err != nil {
			return fmt.Errorf("invalid api spec: %w", err)
		}
		return r.Validate()

	case resource.KindDeployment:
		var r resource.DeploymentResource
		r.Meta = resource.ResourceMeta{Name: name}
		if err := json.Unmarshal(specJSON, &r.Spec); err != nil {
			return fmt.Errorf("invalid deployment spec: %w", err)
		}
		return r.Validate()
	}
	return fmt.Errorf("unknown kind: %s", kind)
}

func extractLabels(body []byte) map[string]string {
	var wrapper struct {
		Metadata struct {
			Labels map[string]string `json:"labels"`
		} `json:"metadata"`
	}
	json.Unmarshal(body, &wrapper)
	return wrapper.Metadata.Labels
}

func parseLabelsQuery(r *http.Request) map[string]string {
	labelStr := r.URL.Query().Get("labels")
	if labelStr == "" {
		return nil
	}
	labels := make(map[string]string)
	for _, pair := range strings.Split(labelStr, ",") {
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) == 2 {
			labels[parts[0]] = parts[1]
		}
	}
	return labels
}

// parseSpecFilters extracts spec-field query params (gatewayRef, listenerRef, etc.).
// The same query param name is used across resource kinds; matchesSpecFilters resolves
// both flat fields (e.g., spec.gatewayRef on Listeners) and nested fields
// (e.g., spec.gateway.name on Deployments) via fallback logic.
func parseSpecFilters(r *http.Request) map[string]string {
	filters := make(map[string]string)
	for _, key := range []string{"gatewayRef", "listenerRef", "virtualHostRef", "apiRef"} {
		if v := r.URL.Query().Get(key); v != "" {
			filters[key] = v
		}
	}
	return filters
}

// filterBySpec post-filters stored resources by spec JSON fields.
func filterBySpec(items []*store.StoredResource, filters map[string]string) []*store.StoredResource {
	var result []*store.StoredResource
	for _, item := range items {
		if matchesSpecFilters(item.SpecJSON, filters) {
			result = append(result, item)
		}
	}
	return result
}

// specFilterAliases maps query param names to nested JSON paths for resources
// that use nested structures (e.g., Deployment.spec.gateway.name).
var specFilterAliases = map[string]string{
	"gatewayRef":     "gateway.name",
	"listenerRef":    "gateway.listener",
	"virtualHostRef": "gateway.virtualHost",
}

// matchesSpecFilters checks if a resource's spec JSON contains all the
// specified field values. Tries the flat key first (e.g., spec.gatewayRef),
// then falls back to a nested alias (e.g., spec.gateway.name) for resources
// like Deployments that use nested structures.
func matchesSpecFilters(specJSON json.RawMessage, filters map[string]string) bool {
	var spec map[string]interface{}
	if err := json.Unmarshal(specJSON, &spec); err != nil {
		return false
	}
	for key, expected := range filters {
		actual := resolveNestedField(spec, key)
		if actual == nil {
			if alias, ok := specFilterAliases[key]; ok {
				actual = resolveNestedField(spec, alias)
			}
		}
		if actual == nil {
			return false
		}
		if fmt.Sprintf("%v", actual) != expected {
			return false
		}
	}
	return true
}

// resolveNestedField resolves a dot-notation key (e.g., "gateway.name") against a map.
func resolveNestedField(m map[string]interface{}, key string) interface{} {
	parts := strings.Split(key, ".")
	var current interface{} = m
	for _, part := range parts {
		obj, ok := current.(map[string]interface{})
		if !ok {
			return nil
		}
		current, ok = obj[part]
		if !ok {
			return nil
		}
	}
	return current
}

func writeResourceResponse(w http.ResponseWriter, status int, res *store.StoredResource) {
	writeJSON(w, status, resource.Response{
		Kind:     res.Meta.Kind,
		Metadata: res.Meta,
		Spec:     res.SpecJSON,
		Status:   res.StatusJSON,
	})
}

func handleStoreError(w http.ResponseWriter, err error) {
	switch {
	case isNotFound(err):
		writeError(w, http.StatusNotFound, err.Error())
	case isRevisionConflict(err):
		writeError(w, http.StatusConflict, err.Error())
	case isOwnershipConflict(err):
		writeError(w, http.StatusConflict, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, err.Error())
	}
}

func isNotFound(err error) bool {
	return err == resource.ErrNotFound
}

func isRevisionConflict(err error) bool {
	_, ok := err.(*resource.RevisionConflictError)
	return ok || err == resource.ErrRevisionConflict
}

func isOwnershipConflict(err error) bool {
	_, ok := err.(*resource.OwnershipConflictError)
	return ok || err == resource.ErrOwnershipConflict
}

func writeError(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, resource.ErrorResponse{Error: msg, Code: code})
}

func writeJSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}
