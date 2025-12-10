package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/flowc-labs/flowc/internal/flowc/server/models"
)

// CreateGateway handles gateway registration
func (h *Handlers) CreateGateway(w http.ResponseWriter, r *http.Request) {
	var req models.CreateGatewayRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	// Validate required fields
	if req.NodeID == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "node_id is required")
		return
	}
	if req.Name == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "name is required")
		return
	}

	gateway, err := h.services.GatewayService.CreateGateway(r.Context(), &req)
	if err != nil {
		h.logger.WithError(err).Error("Failed to create gateway")
		h.writeErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	response := models.GatewayResponse{
		Success: true,
		Gateway: gateway,
	}
	h.WriteJSONResponse(w, http.StatusCreated, response)
}

// GetGateway retrieves a gateway by ID
func (h *Handlers) GetGateway(w http.ResponseWriter, r *http.Request) {
	gatewayID := r.PathValue("id")
	if gatewayID == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "Gateway ID is required")
		return
	}

	gateway, err := h.services.GatewayService.GetGateway(r.Context(), gatewayID)
	if err != nil {
		h.writeErrorResponse(w, http.StatusNotFound, err.Error())
		return
	}

	response := models.GatewayResponse{
		Success: true,
		Gateway: gateway,
	}
	h.WriteJSONResponse(w, http.StatusOK, response)
}

// ListGateways retrieves all gateways
func (h *Handlers) ListGateways(w http.ResponseWriter, r *http.Request) {
	gateways, err := h.services.GatewayService.ListGateways(r.Context())
	if err != nil {
		h.logger.WithError(err).Error("Failed to list gateways")
		h.writeErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	response := models.ListGatewaysResponse{
		Success:  true,
		Gateways: gateways,
		Total:    len(gateways),
	}
	h.WriteJSONResponse(w, http.StatusOK, response)
}

// UpdateGateway updates an existing gateway
func (h *Handlers) UpdateGateway(w http.ResponseWriter, r *http.Request) {
	gatewayID := r.PathValue("id")
	if gatewayID == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "Gateway ID is required")
		return
	}

	var req models.UpdateGatewayRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	gateway, err := h.services.GatewayService.UpdateGateway(r.Context(), gatewayID, &req)
	if err != nil {
		h.logger.WithError(err).Error("Failed to update gateway")
		h.writeErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	response := models.GatewayResponse{
		Success: true,
		Gateway: gateway,
	}
	h.WriteJSONResponse(w, http.StatusOK, response)
}

// DeleteGateway removes a gateway
func (h *Handlers) DeleteGateway(w http.ResponseWriter, r *http.Request) {
	gatewayID := r.PathValue("id")
	if gatewayID == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "Gateway ID is required")
		return
	}

	// Check for force parameter
	force := r.URL.Query().Get("force") == "true"

	if err := h.services.GatewayService.DeleteGateway(r.Context(), gatewayID, force); err != nil {
		h.logger.WithError(err).WithField("force", force).Error("Failed to delete gateway")
		h.writeErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	response := models.DeleteGatewayResponse{
		Success: true,
		Message: "Gateway deleted successfully",
	}
	h.WriteJSONResponse(w, http.StatusOK, response)
}

// GetGatewayAPIs retrieves APIs deployed to a gateway
func (h *Handlers) GetGatewayAPIs(w http.ResponseWriter, r *http.Request) {
	gatewayID := r.PathValue("id")
	if gatewayID == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "Gateway ID is required")
		return
	}

	deployments, err := h.services.GatewayService.GetGatewayAPIs(r.Context(), gatewayID)
	if err != nil {
		h.writeErrorResponse(w, http.StatusNotFound, err.Error())
		return
	}

	response := models.GatewayAPIsResponse{
		Success:     true,
		GatewayID:   gatewayID,
		Deployments: deployments,
		Total:       len(deployments),
	}
	h.WriteJSONResponse(w, http.StatusOK, response)
}
