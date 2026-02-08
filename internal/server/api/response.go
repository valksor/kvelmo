// Package api provides standardized API request/response handling for the web server.
package api

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
)

// Response is the standard API response format.
// All API endpoints should return this structure for consistency.
type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *ErrorInfo  `json:"error,omitempty"`
}

// ErrorInfo contains error details for API responses.
type ErrorInfo struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// Common error codes - use these constants for consistent error handling.
const (
	ErrCodeBadRequest     = "BAD_REQUEST"
	ErrCodeUnauthorized   = "UNAUTHORIZED"
	ErrCodeForbidden      = "FORBIDDEN"
	ErrCodeNotFound       = "NOT_FOUND"
	ErrCodeConflict       = "CONFLICT"
	ErrCodeInternal       = "INTERNAL_ERROR"
	ErrCodeValidation     = "VALIDATION_ERROR"
	ErrCodeNoActiveTask   = "NO_ACTIVE_TASK"
	ErrCodeInvalidState   = "INVALID_STATE"
	ErrCodeProviderError  = "PROVIDER_ERROR"
	ErrCodeAgentError     = "AGENT_ERROR"
	ErrCodeBudgetExceeded = "BUDGET_EXCEEDED"
	ErrCodeNotConfigured  = "NOT_CONFIGURED"
)

// WriteSuccess writes a successful JSON response with the given data.
func WriteSuccess(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(Response{
		Success: true,
		Data:    data,
	}); err != nil {
		slog.Error("failed to encode response", "error", err)
	}
}

// WriteSuccessMessage writes a successful response with just a message.
func WriteSuccessMessage(w http.ResponseWriter, message string) {
	WriteSuccess(w, map[string]string{"message": message})
}

// WriteError writes an error JSON response with the given status, code, and message.
func WriteError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(Response{
		Success: false,
		Error: &ErrorInfo{
			Code:    code,
			Message: message,
		},
	}); err != nil {
		slog.Error("failed to encode error response", "error", err)
	}
}

// WriteErrorWithDetails writes an error response with additional details.
func WriteErrorWithDetails(w http.ResponseWriter, status int, code, message, details string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(Response{
		Success: false,
		Error: &ErrorInfo{
			Code:    code,
			Message: message,
			Details: details,
		},
	}); err != nil {
		slog.Error("failed to encode error response", "error", err)
	}
}

// Convenience functions for common error types

// WriteBadRequest writes a 400 Bad Request error.
func WriteBadRequest(w http.ResponseWriter, message string) {
	WriteError(w, http.StatusBadRequest, ErrCodeBadRequest, message)
}

// WriteValidationError writes a 400 Bad Request with validation error code.
func WriteValidationError(w http.ResponseWriter, message string) {
	WriteError(w, http.StatusBadRequest, ErrCodeValidation, message)
}

// WriteUnauthorized writes a 401 Unauthorized error.
func WriteUnauthorized(w http.ResponseWriter) {
	WriteError(w, http.StatusUnauthorized, ErrCodeUnauthorized, "authentication required")
}

// WriteForbidden writes a 403 Forbidden error.
func WriteForbidden(w http.ResponseWriter, message string) {
	WriteError(w, http.StatusForbidden, ErrCodeForbidden, message)
}

// WriteNotFound writes a 404 Not Found error.
func WriteNotFound(w http.ResponseWriter, message string) {
	WriteError(w, http.StatusNotFound, ErrCodeNotFound, message)
}

// WriteConflict writes a 409 Conflict error.
func WriteConflict(w http.ResponseWriter, message string) {
	WriteError(w, http.StatusConflict, ErrCodeConflict, message)
}

// WriteInternal writes a 500 Internal Server Error.
// The full error is logged but not exposed to the client.
func WriteInternal(w http.ResponseWriter, err error) {
	slog.Error("internal server error", "error", err)
	WriteError(w, http.StatusInternalServerError, ErrCodeInternal, "internal server error")
}

// WriteInternalWithContext writes a 500 error with additional context for logging.
func WriteInternalWithContext(w http.ResponseWriter, err error, context string) {
	slog.Error("internal server error", "error", err, "context", context)
	WriteError(w, http.StatusInternalServerError, ErrCodeInternal, "internal server error")
}

// Domain-specific error helpers

// WriteNoActiveTask writes an error indicating no active task exists.
func WriteNoActiveTask(w http.ResponseWriter) {
	WriteError(w, http.StatusBadRequest, ErrCodeNoActiveTask, "no active task")
}

// WriteInvalidState writes an error indicating the task is in the wrong state.
func WriteInvalidState(w http.ResponseWriter, current, required string) {
	WriteError(w, http.StatusConflict, ErrCodeInvalidState,
		fmt.Sprintf("task is in '%s' state, requires '%s'", current, required))
}

// WriteInvalidStateForAction writes an error for an invalid state transition.
func WriteInvalidStateForAction(w http.ResponseWriter, action, current string) {
	WriteError(w, http.StatusConflict, ErrCodeInvalidState,
		fmt.Sprintf("cannot %s: task is in '%s' state", action, current))
}

// WriteProviderError writes an error from a task provider.
func WriteProviderError(w http.ResponseWriter, err error) {
	slog.Error("provider error", "error", err)
	WriteError(w, http.StatusBadGateway, ErrCodeProviderError, "provider error: "+err.Error())
}

// WriteAgentError writes an error from an AI agent.
func WriteAgentError(w http.ResponseWriter, err error) {
	slog.Error("agent error", "error", err)
	WriteError(w, http.StatusBadGateway, ErrCodeAgentError, "agent error: "+err.Error())
}

// WriteBudgetExceeded writes an error indicating budget has been exceeded.
func WriteBudgetExceeded(w http.ResponseWriter, budgetType string) {
	WriteError(w, http.StatusPaymentRequired, ErrCodeBudgetExceeded,
		budgetType+" budget exceeded")
}

// WriteNotConfigured writes an error indicating a feature is not configured.
func WriteNotConfigured(w http.ResponseWriter, feature string) {
	WriteError(w, http.StatusNotFound, ErrCodeNotConfigured,
		feature+" is not configured")
}
