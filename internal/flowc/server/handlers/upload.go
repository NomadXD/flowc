package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/flowc-labs/flowc/internal/flowc/resource"
	"github.com/flowc-labs/flowc/internal/flowc/resource/store"
	"github.com/flowc-labs/flowc/internal/flowc/server/loader"
	"github.com/flowc-labs/flowc/pkg/bundle"
	"github.com/flowc-labs/flowc/pkg/logger"
)

// UploadHandler handles ZIP bundle uploads and converts them to API + Deployment resources.
type UploadHandler struct {
	store        store.Store
	bundleLoader *loader.BundleLoader
	logger       *logger.EnvoyLogger
}

// NewUploadHandler creates a new upload handler.
func NewUploadHandler(s store.Store, log *logger.EnvoyLogger) *UploadHandler {
	return &UploadHandler{
		store:        s,
		bundleLoader: loader.NewBundleLoader(),
		logger:       log,
	}
}

// HandleUpload handles POST /api/v1/upload
// Accepts a multipart ZIP file, creates an API resource and optionally a Deployment resource.
func (h *UploadHandler) HandleUpload(w http.ResponseWriter, r *http.Request) {
	// Parse multipart form
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "failed to parse multipart form: "+err.Error())
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "file field is required")
		return
	}
	defer file.Close()

	zipData, err := io.ReadAll(file)
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to read file")
		return
	}

	// Validate ZIP
	if err := bundle.ValidateZip(zipData); err != nil {
		writeError(w, http.StatusBadRequest, "invalid zip: "+err.Error())
		return
	}

	// Load bundle
	deploymentBundle, err := h.bundleLoader.LoadBundle(zipData)
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to parse bundle: "+err.Error())
		return
	}

	meta := deploymentBundle.FlowCMetadata

	// Create API resource
	apiName := meta.Name
	apiSpec := resource.APISpec{
		Version:     meta.Version,
		Description: meta.Description,
		Context:     meta.Context,
		APIType:     meta.APIType,
		SpecContent: string(deploymentBundle.Spec),
		Upstream:    meta.Upstream,
	}

	apiSpecJSON, _ := json.Marshal(apiSpec)
	apiStored := &store.StoredResource{
		Meta: resource.ResourceMeta{
			Kind: resource.KindAPI,
			Name: apiName,
		},
		SpecJSON: apiSpecJSON,
	}

	managedBy := r.Header.Get("X-Managed-By")
	if managedBy == "" {
		managedBy = "upload"
	}

	apiOut, err := h.store.Put(r.Context(), apiStored, store.PutOptions{ManagedBy: managedBy})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to store API: "+err.Error())
		return
	}

	result := []resource.ApplyResultItem{
		{
			Kind:   resource.KindAPI,
			Name:   apiOut.Meta.Name,
			Action: actionFromRevision(apiOut.Meta.Revision),
		},
	}

	// If gateway config is present, create a Deployment resource too
	if meta.Gateway.GatewayID != "" || meta.Gateway.NodeID != "" {
		depName := fmt.Sprintf("%s-deploy", apiName)
		depSpec := resource.DeploymentSpec{
			APIRef:         apiName,
			GatewayRef:     coalesce(meta.Gateway.GatewayID, meta.Gateway.NodeID),
			ListenerRef:    fmt.Sprintf("port-%d", meta.Gateway.Port),
			EnvironmentRef: meta.Gateway.Environment,
			Strategy:       meta.Strategy,
		}

		depSpecJSON, _ := json.Marshal(depSpec)
		depStored := &store.StoredResource{
			Meta: resource.ResourceMeta{
				Kind: resource.KindDeployment,
				Name: depName,
			},
			SpecJSON: depSpecJSON,
		}

		depOut, err := h.store.Put(r.Context(), depStored, store.PutOptions{ManagedBy: managedBy})
		if err != nil {
			// API was created but deployment failed
			result = append(result, resource.ApplyResultItem{
				Kind:   resource.KindDeployment,
				Name:   depName,
				Action: "failed",
				Error:  err.Error(),
			})
		} else {
			result = append(result, resource.ApplyResultItem{
				Kind:   resource.KindDeployment,
				Name:   depOut.Meta.Name,
				Action: actionFromRevision(depOut.Meta.Revision),
			})
		}
	}

	writeJSON(w, http.StatusOK, resource.ApplyResult{Results: result})
}

func actionFromRevision(rev int64) string {
	if rev == 1 {
		return "created"
	}
	return "updated"
}

func coalesce(a, b string) string {
	if a != "" {
		return a
	}
	return b
}
