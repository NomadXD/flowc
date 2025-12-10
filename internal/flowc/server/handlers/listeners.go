package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/flowc-labs/flowc/internal/flowc/server/models"
)

// CreateListener handles listener creation within a gateway
func (h *Handlers) CreateListener(w http.ResponseWriter, r *http.Request) {
	gatewayID := r.PathValue("gateway_id")
	if gatewayID == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "Gateway ID is required")
		return
	}

	var req models.CreateListenerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	// Validate required fields
	if req.Port == 0 {
		h.writeErrorResponse(w, http.StatusBadRequest, "port is required")
		return
	}

	// Validate that at least one environment is provided
	if len(req.Environments) == 0 {
		h.writeErrorResponse(w, http.StatusBadRequest, "at least one environment is required")
		return
	}

	listener, err := h.services.ListenerService.CreateListener(r.Context(), gatewayID, &req)
	if err != nil {
		h.logger.WithError(err).Error("Failed to create listener")
		h.writeErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	response := models.ListenerResponse{
		Success:  true,
		Listener: listener,
	}
	h.WriteJSONResponse(w, http.StatusCreated, response)
}

// GetListener retrieves a listener by gateway ID and port
func (h *Handlers) GetListener(w http.ResponseWriter, r *http.Request) {
	gatewayID := r.PathValue("gateway_id")
	if gatewayID == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "Gateway ID is required")
		return
	}

	portStr := r.PathValue("port")
	if portStr == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "Port is required")
		return
	}

	port, err := strconv.ParseUint(portStr, 10, 32)
	if err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid port number")
		return
	}

	listener, err := h.services.ListenerService.GetListenerByGatewayAndPort(r.Context(), gatewayID, uint32(port))
	if err != nil {
		h.writeErrorResponse(w, http.StatusNotFound, err.Error())
		return
	}

	response := models.ListenerResponse{
		Success:  true,
		Listener: listener,
	}
	h.WriteJSONResponse(w, http.StatusOK, response)
}

// ListListeners retrieves all listeners for a gateway
func (h *Handlers) ListListeners(w http.ResponseWriter, r *http.Request) {
	gatewayID := r.PathValue("gateway_id")
	if gatewayID == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "Gateway ID is required")
		return
	}

	listeners, err := h.services.ListenerService.ListListenersByGateway(r.Context(), gatewayID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to list listeners")
		h.writeErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	response := models.ListListenersResponse{
		Success:   true,
		Listeners: listeners,
		Total:     len(listeners),
	}
	h.WriteJSONResponse(w, http.StatusOK, response)
}

// UpdateListener updates an existing listener
func (h *Handlers) UpdateListener(w http.ResponseWriter, r *http.Request) {
	gatewayID := r.PathValue("gateway_id")
	if gatewayID == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "Gateway ID is required")
		return
	}

	portStr := r.PathValue("port")
	if portStr == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "Port is required")
		return
	}

	port, err := strconv.ParseUint(portStr, 10, 32)
	if err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid port number")
		return
	}

	var req models.UpdateListenerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	// Get listener by gateway and port first
	existingListener, err := h.services.ListenerService.GetListenerByGatewayAndPort(r.Context(), gatewayID, uint32(port))
	if err != nil {
		h.writeErrorResponse(w, http.StatusNotFound, err.Error())
		return
	}

	listener, err := h.services.ListenerService.UpdateListener(r.Context(), existingListener.ID, &req)
	if err != nil {
		h.logger.WithError(err).Error("Failed to update listener")
		h.writeErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	response := models.ListenerResponse{
		Success:  true,
		Listener: listener,
	}
	h.WriteJSONResponse(w, http.StatusOK, response)
}

// DeleteListener removes a listener from a gateway
func (h *Handlers) DeleteListener(w http.ResponseWriter, r *http.Request) {
	gatewayID := r.PathValue("gateway_id")
	if gatewayID == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "Gateway ID is required")
		return
	}

	portStr := r.PathValue("port")
	if portStr == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "Port is required")
		return
	}

	port, err := strconv.ParseUint(portStr, 10, 32)
	if err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid port number")
		return
	}

	// Check for force parameter
	force := r.URL.Query().Get("force") == "true"

	// Get listener by gateway and port first
	existingListener, err := h.services.ListenerService.GetListenerByGatewayAndPort(r.Context(), gatewayID, uint32(port))
	if err != nil {
		h.writeErrorResponse(w, http.StatusNotFound, err.Error())
		return
	}

	if err := h.services.ListenerService.DeleteListener(r.Context(), existingListener.ID, force); err != nil {
		h.logger.WithError(err).WithField("force", force).Error("Failed to delete listener")
		h.writeErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	response := models.DeleteListenerResponse{
		Success: true,
		Message: "Listener deleted successfully",
	}
	h.WriteJSONResponse(w, http.StatusOK, response)
}
