package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/flowc-labs/flowc/internal/flowc/server/models"
	services "github.com/flowc-labs/flowc/internal/flowc/server/services"
	bundle "github.com/flowc-labs/flowc/pkg/bundle"
	"github.com/flowc-labs/flowc/pkg/logger"
)

// DeploymentHandlers handles HTTP requests for API deployments
type DeploymentHandlers struct {
	deploymentService *services.DeploymentService
	logger            *logger.EnvoyLogger
	startTime         time.Time
}

// NewDeploymentHandlers creates a new deployment handlers instance
func NewDeploymentHandlers(deploymentService *services.DeploymentService, logger *logger.EnvoyLogger) *DeploymentHandlers {
	return &DeploymentHandlers{
		deploymentService: deploymentService,
		logger:            logger,
		startTime:         time.Now(),
	}
}

// DeployAPI handles API deployment from zip file upload
func (h *Handlers) DeployAPI(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("Received API deployment request")

	// Parse multipart form
	if err := r.ParseMultipartForm(32 << 20); err != nil { // 32MB max
		h.logger.WithError(err).Error("Failed to parse multipart form")
		h.writeErrorResponse(w, http.StatusBadRequest, "Failed to parse multipart form")
		return
	}

	// Get optional description
	description := r.FormValue("description")

	// Get uploaded file
	file, header, err := r.FormFile("file")
	if err != nil {
		h.logger.WithError(err).Error("Failed to get uploaded file")
		h.writeErrorResponse(w, http.StatusBadRequest, "Failed to get uploaded file")
		return
	}
	defer file.Close()

	// Validate file type - check both content type and filename extension
	contentType := header.Header.Get("Content-Type")
	filename := header.Filename
	isZip := contentType == "application/zip" ||
		contentType == "application/x-zip-compressed" ||
		strings.HasSuffix(strings.ToLower(filename), ".zip")

	if !isZip {
		h.writeErrorResponse(w, http.StatusBadRequest, "File must be a zip archive")
		return
	}

	// Read file data
	zipData, err := io.ReadAll(file)
	if err != nil {
		h.logger.WithError(err).Error("Failed to read uploaded file")
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to read uploaded file")
		return
	}

	h.logger.WithFields(map[string]interface{}{
		"filename": header.Filename,
		"size":     len(zipData),
	}).Info("Processing API deployment")

	// Deploy API
	deployment, err := h.services.DeploymentService.DeployAPI(zipData, description)
	if err != nil {
		h.logger.WithError(err).Error("Failed to deploy API")
		h.writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to deploy API: %v", err))
		return
	}

	response := models.DeploymentResponse{
		Success:    true,
		Message:    "API deployed successfully",
		Deployment: deployment,
	}

	h.WriteJSONResponse(w, http.StatusCreated, response)
}

// GetDeployment retrieves a specific deployment
func (h *Handlers) GetDeployment(w http.ResponseWriter, r *http.Request) {

	deploymentID := r.PathValue("id")
	if deploymentID == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "Deployment ID is required")
		return
	}

	deployment, err := h.services.DeploymentService.GetDeployment(deploymentID)
	if err != nil {
		response := models.GetDeploymentResponse{
			Success: false,
			Error:   err.Error(),
		}
		h.WriteJSONResponse(w, http.StatusNotFound, response)
		return
	}

	response := models.GetDeploymentResponse{
		Success:    true,
		Deployment: deployment,
	}
	h.WriteJSONResponse(w, http.StatusOK, response)
}

// ListDeployments retrieves all deployments
func (h *Handlers) ListDeployments(w http.ResponseWriter, r *http.Request) {
	deployments := h.services.DeploymentService.ListDeployments()

	response := models.ListDeploymentsResponse{
		Success:     true,
		Deployments: deployments,
		Total:       len(deployments),
	}

	h.WriteJSONResponse(w, http.StatusOK, response)
}

// DeleteDeployment removes a deployment
func (h *Handlers) DeleteDeployment(w http.ResponseWriter, r *http.Request) {

	deploymentID := r.PathValue("id")
	if deploymentID == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "Deployment ID is required")
		return
	}

	if err := h.services.DeploymentService.DeleteDeployment(deploymentID); err != nil {
		response := models.DeleteDeploymentResponse{
			Success: false,
			Error:   err.Error(),
		}
		h.WriteJSONResponse(w, http.StatusNotFound, response)
		return
	}

	response := models.DeleteDeploymentResponse{
		Success: true,
		Message: "Deployment deleted successfully",
	}
	h.WriteJSONResponse(w, http.StatusOK, response)
}

// UpdateDeployment updates an existing deployment
func (h *Handlers) UpdateDeployment(w http.ResponseWriter, r *http.Request) {

	deploymentID := r.PathValue("id")
	if deploymentID == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "Deployment ID is required")
		return
	}

	// Parse multipart form
	if err := r.ParseMultipartForm(32 << 20); err != nil { // 32MB max
		h.logger.WithError(err).Error("Failed to parse multipart form")
		h.writeErrorResponse(w, http.StatusBadRequest, "Failed to parse multipart form")
		return
	}

	// Get uploaded file
	file, header, err := r.FormFile("file")
	if err != nil {
		h.logger.WithError(err).Error("Failed to get uploaded file")
		h.writeErrorResponse(w, http.StatusBadRequest, "Failed to get uploaded file")
		return
	}
	defer file.Close()

	// Validate file type - check both content type and filename extension
	contentType := header.Header.Get("Content-Type")
	filename := header.Filename
	isZip := contentType == "application/zip" ||
		contentType == "application/x-zip-compressed" ||
		strings.HasSuffix(strings.ToLower(filename), ".zip")

	if !isZip {
		h.writeErrorResponse(w, http.StatusBadRequest, "File must be a zip archive")
		return
	}

	// Read file data
	zipData, err := io.ReadAll(file)
	if err != nil {
		h.logger.WithError(err).Error("Failed to read uploaded file")
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to read uploaded file")
		return
	}

	// Update deployment
	deployment, err := h.services.DeploymentService.UpdateDeployment(deploymentID, zipData)
	if err != nil {
		h.logger.WithError(err).Error("Failed to update deployment")
		h.writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to update deployment: %v", err))
		return
	}

	response := models.DeploymentResponse{
		Success:    true,
		Message:    "Deployment updated successfully",
		Deployment: deployment,
	}

	h.WriteJSONResponse(w, http.StatusOK, response)
}

// GetDeploymentStats returns deployment statistics
func (h *Handlers) GetDeploymentStats(w http.ResponseWriter, r *http.Request) {

	stats := h.services.DeploymentService.GetDeploymentStats()
	response := map[string]interface{}{
		"success": true,
		"stats":   stats,
	}

	h.WriteJSONResponse(w, http.StatusOK, response)
}

// HealthCheck returns the health status of the API
func (h *Handlers) HealthCheck(w http.ResponseWriter, r *http.Request) {

	uptime := time.Since(h.startTime)

	response := models.HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now(),
		Version:   "1.0.0",
		Uptime:    uptime.String(),
	}

	h.WriteJSONResponse(w, http.StatusOK, response)
}

// ValidateZip validates a zip file without deploying it
func (h *Handlers) ValidateZip(w http.ResponseWriter, r *http.Request) {

	// Parse multipart form
	if err := r.ParseMultipartForm(32 << 20); err != nil { // 32MB max
		h.writeErrorResponse(w, http.StatusBadRequest, "Failed to parse multipart form")
		return
	}

	// Get uploaded file
	file, _, err := r.FormFile("file")
	if err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Failed to get uploaded file")
		return
	}
	defer file.Close()

	// Read file data
	zipData, err := io.ReadAll(file)
	if err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to read uploaded file")
		return
	}

	// Validate zip file
	if err := bundle.ValidateZip(zipData); err != nil {
		response := map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		}
		h.WriteJSONResponse(w, http.StatusBadRequest, response)
		return
	}

	// Get file list
	files, err := bundle.ListFiles(zipData)
	if err != nil {
		response := map[string]interface{}{
			"success": false,
			"error":   "Failed to get file list",
		}
		h.WriteJSONResponse(w, http.StatusInternalServerError, response)
		return
	}

	response := map[string]interface{}{
		"success": true,
		"message": "Zip file is valid",
		"files":   files,
	}
	h.WriteJSONResponse(w, http.StatusOK, response)
}

// Helper methods

// WriteJSONResponse writes a JSON response with the given status code
func (h *Handlers) WriteJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.WithError(err).Error("Failed to encode JSON response")
	}
}

// writeErrorResponse writes an error response in JSON format
func (h *Handlers) writeErrorResponse(w http.ResponseWriter, statusCode int, message string) {
	response := map[string]interface{}{
		"success": false,
		"error":   message,
	}
	h.WriteJSONResponse(w, statusCode, response)
}
