package resource

import "encoding/json"

// Response is the standard single-resource API response.
type Response struct {
	Kind     ResourceKind    `json:"kind"`
	Metadata ResourceMeta    `json:"metadata"`
	Spec     json.RawMessage `json:"spec"`
	Status   json.RawMessage `json:"status,omitempty"`
}

// ErrorResponse is the standard error response.
type ErrorResponse struct {
	Error   string            `json:"error"`
	Code    int               `json:"code"`
	Details map[string]string `json:"details,omitempty"`
}

// ListResponse is the standard list response.
type ListResponse struct {
	Kind  string     `json:"kind"` // e.g., "GatewayList"
	Items []Response `json:"items"`
	Total int        `json:"total"`
}

// ApplyRequest is the bulk-apply request body.
type ApplyRequest struct {
	Resources []json.RawMessage `json:"resources"`
}

// ApplyResultItem describes the outcome of applying one resource.
type ApplyResultItem struct {
	Kind   ResourceKind `json:"kind"`
	Name   string       `json:"name"`
	Action string       `json:"action"` // "created", "updated", "unchanged", "failed"
	Error  string       `json:"error,omitempty"`
}

// ApplyResult is the response for a bulk-apply request.
type ApplyResult struct {
	Results []ApplyResultItem `json:"results"`
}
