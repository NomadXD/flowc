package handlers

import (
	"net/http"

	"github.com/flowc-labs/flowc/internal/flowc/profile"
	"github.com/flowc-labs/flowc/internal/flowc/resource"
	"github.com/flowc-labs/flowc/internal/flowc/resource/store"
	"github.com/flowc-labs/flowc/pkg/logger"
)

// BootstrapHandler generates Envoy bootstrap configurations for gateways.
type BootstrapHandler struct {
	typedStore       *store.TypedStore
	logger           *logger.EnvoyLogger
	controlPlaneHost string
	controlPlanePort int
}

// NewBootstrapHandler creates a new bootstrap handler.
func NewBootstrapHandler(s store.Store, controlPlaneHost string, controlPlanePort int, log *logger.EnvoyLogger) *BootstrapHandler {
	return &BootstrapHandler{
		typedStore:       store.NewTypedStore(s),
		logger:           log,
		controlPlaneHost: controlPlaneHost,
		controlPlanePort: controlPlanePort,
	}
}

// HandleBootstrap generates an Envoy bootstrap YAML for a gateway.
// GET /api/v1/gateways/{name}/bootstrap
func (h *BootstrapHandler) HandleBootstrap(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")

	gw, err := h.typedStore.GetGateway(r.Context(), name)
	if err != nil {
		if err == resource.ErrNotFound {
			writeError(w, http.StatusNotFound, "gateway not found")
		} else {
			writeError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	// Load the referenced profile (optional).
	var prof *resource.GatewayProfileResource
	if gw.Spec.ProfileRef != "" {
		prof, _ = h.typedStore.GetGatewayProfile(r.Context(), gw.Spec.ProfileRef)
	}

	bootstrapYAML, err := profile.GenerateBootstrapYAML(gw, prof, h.controlPlaneHost, h.controlPlanePort)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate bootstrap: "+err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/x-yaml")
	w.Header().Set("Content-Disposition", "attachment; filename=envoy-bootstrap.yaml")
	w.WriteHeader(http.StatusOK)
	w.Write(bootstrapYAML)
}
