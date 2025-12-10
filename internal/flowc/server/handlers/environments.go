package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/flowc-labs/flowc/internal/flowc/server/models"
)

// CreateEnvironment handles environment creation within a listener
func (h *Handlers) CreateEnvironment(w http.ResponseWriter, r *http.Request) {
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

	var req models.CreateEnvironmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	// Validate required fields
	if req.Name == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "name is required")
		return
	}
	if req.Hostname == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "hostname is required")
		return
	}

	// Get listener by gateway and port
	listener, err := h.services.ListenerService.GetListenerByGatewayAndPort(r.Context(), gatewayID, uint32(port))
	if err != nil {
		h.writeErrorResponse(w, http.StatusNotFound, err.Error())
		return
	}

	env, err := h.services.EnvironmentService.CreateEnvironment(r.Context(), listener.ID, &req)
	if err != nil {
		h.logger.WithError(err).Error("Failed to create environment")
		h.writeErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	response := models.EnvironmentResponse{
		Success:     true,
		Environment: env,
	}
	h.WriteJSONResponse(w, http.StatusCreated, response)
}

// GetEnvironment retrieves an environment by listener and name
func (h *Handlers) GetEnvironment(w http.ResponseWriter, r *http.Request) {
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

	envName := r.PathValue("name")
	if envName == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "Environment name is required")
		return
	}

	// Get listener by gateway and port
	listener, err := h.services.ListenerService.GetListenerByGatewayAndPort(r.Context(), gatewayID, uint32(port))
	if err != nil {
		h.writeErrorResponse(w, http.StatusNotFound, err.Error())
		return
	}

	env, err := h.services.EnvironmentService.GetEnvironmentByListenerAndName(r.Context(), listener.ID, envName)
	if err != nil {
		h.writeErrorResponse(w, http.StatusNotFound, err.Error())
		return
	}

	response := models.EnvironmentResponse{
		Success:     true,
		Environment: env,
	}
	h.WriteJSONResponse(w, http.StatusOK, response)
}

// ListEnvironments retrieves all environments for a listener
func (h *Handlers) ListEnvironments(w http.ResponseWriter, r *http.Request) {
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

	// Get listener by gateway and port
	listener, err := h.services.ListenerService.GetListenerByGatewayAndPort(r.Context(), gatewayID, uint32(port))
	if err != nil {
		h.writeErrorResponse(w, http.StatusNotFound, err.Error())
		return
	}

	envs, err := h.services.EnvironmentService.ListEnvironmentsByListener(r.Context(), listener.ID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to list environments")
		h.writeErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	response := models.ListEnvironmentsResponse{
		Success:      true,
		Environments: envs,
		Total:        len(envs),
	}
	h.WriteJSONResponse(w, http.StatusOK, response)
}

// UpdateEnvironment updates an existing environment
func (h *Handlers) UpdateEnvironment(w http.ResponseWriter, r *http.Request) {
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

	envName := r.PathValue("name")
	if envName == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "Environment name is required")
		return
	}

	var req models.UpdateEnvironmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	// Get listener by gateway and port
	listener, err := h.services.ListenerService.GetListenerByGatewayAndPort(r.Context(), gatewayID, uint32(port))
	if err != nil {
		h.writeErrorResponse(w, http.StatusNotFound, err.Error())
		return
	}

	// Get environment by name
	existingEnv, err := h.services.EnvironmentService.GetEnvironmentByListenerAndName(r.Context(), listener.ID, envName)
	if err != nil {
		h.writeErrorResponse(w, http.StatusNotFound, err.Error())
		return
	}

	env, err := h.services.EnvironmentService.UpdateEnvironment(r.Context(), existingEnv.ID, &req)
	if err != nil {
		h.logger.WithError(err).Error("Failed to update environment")
		h.writeErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	response := models.EnvironmentResponse{
		Success:     true,
		Environment: env,
	}
	h.WriteJSONResponse(w, http.StatusOK, response)
}

// DeleteEnvironment removes an environment from a listener
func (h *Handlers) DeleteEnvironment(w http.ResponseWriter, r *http.Request) {
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

	envName := r.PathValue("name")
	if envName == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "Environment name is required")
		return
	}

	// Check for force parameter
	force := r.URL.Query().Get("force") == "true"

	// Get listener by gateway and port
	listener, err := h.services.ListenerService.GetListenerByGatewayAndPort(r.Context(), gatewayID, uint32(port))
	if err != nil {
		h.writeErrorResponse(w, http.StatusNotFound, err.Error())
		return
	}

	// Get environment by name
	existingEnv, err := h.services.EnvironmentService.GetEnvironmentByListenerAndName(r.Context(), listener.ID, envName)
	if err != nil {
		h.writeErrorResponse(w, http.StatusNotFound, err.Error())
		return
	}

	if err := h.services.EnvironmentService.DeleteEnvironment(r.Context(), existingEnv.ID, force); err != nil {
		h.logger.WithError(err).WithField("force", force).Error("Failed to delete environment")
		h.writeErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	response := models.DeleteEnvironmentResponse{
		Success: true,
		Message: "Environment deleted successfully",
	}
	h.WriteJSONResponse(w, http.StatusOK, response)
}

// GetEnvironmentAPIs retrieves APIs deployed to an environment
func (h *Handlers) GetEnvironmentAPIs(w http.ResponseWriter, r *http.Request) {
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

	envName := r.PathValue("name")
	if envName == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "Environment name is required")
		return
	}

	// Get listener by gateway and port
	listener, err := h.services.ListenerService.GetListenerByGatewayAndPort(r.Context(), gatewayID, uint32(port))
	if err != nil {
		h.writeErrorResponse(w, http.StatusNotFound, err.Error())
		return
	}

	// Get environment by name
	env, err := h.services.EnvironmentService.GetEnvironmentByListenerAndName(r.Context(), listener.ID, envName)
	if err != nil {
		h.writeErrorResponse(w, http.StatusNotFound, err.Error())
		return
	}

	deployments, err := h.services.EnvironmentService.GetEnvironmentAPIs(r.Context(), env.ID)
	if err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	response := models.EnvironmentAPIsResponse{
		Success:       true,
		EnvironmentID: env.ID,
		Deployments:   deployments,
		Total:         len(deployments),
	}
	h.WriteJSONResponse(w, http.StatusOK, response)
}
