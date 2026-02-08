package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWriteSuccess(t *testing.T) {
	tests := []struct {
		name        string
		data        interface{}
		wantSuccess bool
	}{
		{
			name:        "string data",
			data:        "hello",
			wantSuccess: true,
		},
		{
			name:        "map data",
			data:        map[string]any{"foo": "bar"},
			wantSuccess: true,
		},
		{
			name:        "slice data",
			data:        []string{"a", "b", "c"},
			wantSuccess: true,
		},
		{
			name:        "nil data",
			data:        nil,
			wantSuccess: true,
		},
		{
			name:        "struct data",
			data:        struct{ Name string }{Name: "test"},
			wantSuccess: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			WriteSuccess(w, tt.data)

			// Check status code
			if w.Code != http.StatusOK {
				t.Errorf("WriteSuccess() status = %d, want %d", w.Code, http.StatusOK)
			}

			// Check content type
			ct := w.Header().Get("Content-Type")
			if !strings.Contains(ct, "application/json") {
				t.Errorf("WriteSuccess() content-type = %s, want application/json", ct)
			}

			// Parse response
			var resp Response
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Fatalf("failed to parse response: %v", err)
			}

			if resp.Success != tt.wantSuccess {
				t.Errorf("WriteSuccess() success = %v, want %v", resp.Success, tt.wantSuccess)
			}

			if resp.Error != nil {
				t.Errorf("WriteSuccess() error = %v, want nil", resp.Error)
			}
		})
	}
}

func TestWriteSuccessMessage(t *testing.T) {
	w := httptest.NewRecorder()
	message := "operation completed successfully"

	WriteSuccessMessage(w, message)

	if w.Code != http.StatusOK {
		t.Errorf("WriteSuccessMessage() status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if !resp.Success {
		t.Errorf("WriteSuccessMessage() success = false, want true")
	}

	// Data should contain the message
	if resp.Data == nil {
		t.Errorf("WriteSuccessMessage() data = nil, want map with message")
	}

	data, ok := resp.Data.(map[string]any)
	if !ok {
		t.Fatalf("WriteSuccessMessage() data type = %T, want map[string]any", resp.Data)
	}

	if data["message"] != message {
		t.Errorf("WriteSuccessMessage() message = %v, want %s", data["message"], message)
	}
}

func TestWriteError(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		code       string
		message    string
		wantStatus int
	}{
		{
			name:       "bad request",
			status:     http.StatusBadRequest,
			code:       ErrCodeBadRequest,
			message:    "invalid input",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "not found",
			status:     http.StatusNotFound,
			code:       ErrCodeNotFound,
			message:    "resource not found",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "internal error",
			status:     http.StatusInternalServerError,
			code:       ErrCodeInternal,
			message:    "something went wrong",
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			WriteError(w, tt.status, tt.code, tt.message)

			if w.Code != tt.wantStatus {
				t.Errorf("WriteError() status = %d, want %d", w.Code, tt.wantStatus)
			}

			ct := w.Header().Get("Content-Type")
			if !strings.Contains(ct, "application/json") {
				t.Errorf("WriteError() content-type = %s, want application/json", ct)
			}

			var resp Response
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Fatalf("failed to parse response: %v", err)
			}

			if resp.Success {
				t.Errorf("WriteError() success = true, want false")
			}

			if resp.Error == nil {
				t.Fatal("WriteError() error = nil, want error info")
			}

			if resp.Error.Code != tt.code {
				t.Errorf("WriteError() code = %s, want %s", resp.Error.Code, tt.code)
			}

			if resp.Error.Message != tt.message {
				t.Errorf("WriteError() message = %s, want %s", resp.Error.Message, tt.message)
			}
		})
	}
}

func TestWriteErrorWithDetails(t *testing.T) {
	w := httptest.NewRecorder()
	status := http.StatusBadRequest
	code := ErrCodeValidation
	message := "validation failed"
	details := "field 'name' is required"

	WriteErrorWithDetails(w, status, code, message, details)

	if w.Code != status {
		t.Errorf("WriteErrorWithDetails() status = %d, want %d", w.Code, status)
	}

	var resp Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp.Success {
		t.Errorf("WriteErrorWithDetails() success = true, want false")
	}

	if resp.Error == nil {
		t.Fatal("WriteErrorWithDetails() error = nil, want error info")
	}

	if resp.Error.Code != code {
		t.Errorf("WriteErrorWithDetails() code = %s, want %s", resp.Error.Code, code)
	}

	if resp.Error.Message != message {
		t.Errorf("WriteErrorWithDetails() message = %s, want %s", resp.Error.Message, message)
	}

	if resp.Error.Details != details {
		t.Errorf("WriteErrorWithDetails() details = %s, want %s", resp.Error.Details, details)
	}
}

func TestWriteBadRequest(t *testing.T) {
	w := httptest.NewRecorder()
	message := "bad request"

	WriteBadRequest(w, message)

	if w.Code != http.StatusBadRequest {
		t.Errorf("WriteBadRequest() status = %d, want %d", w.Code, http.StatusBadRequest)
	}

	var resp Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp.Error == nil {
		t.Fatal("WriteBadRequest() error = nil")
	}

	if resp.Error.Code != ErrCodeBadRequest {
		t.Errorf("WriteBadRequest() code = %s, want %s", resp.Error.Code, ErrCodeBadRequest)
	}
}

func TestWriteValidationError(t *testing.T) {
	w := httptest.NewRecorder()
	message := "validation error"

	WriteValidationError(w, message)

	if w.Code != http.StatusBadRequest {
		t.Errorf("WriteValidationError() status = %d, want %d", w.Code, http.StatusBadRequest)
	}

	var resp Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp.Error == nil {
		t.Fatal("WriteValidationError() error = nil")
	}

	if resp.Error.Code != ErrCodeValidation {
		t.Errorf("WriteValidationError() code = %s, want %s", resp.Error.Code, ErrCodeValidation)
	}
}

func TestWriteUnauthorized(t *testing.T) {
	w := httptest.NewRecorder()

	WriteUnauthorized(w)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("WriteUnauthorized() status = %d, want %d", w.Code, http.StatusUnauthorized)
	}

	var resp Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp.Error == nil {
		t.Fatal("WriteUnauthorized() error = nil")
	}

	if resp.Error.Code != ErrCodeUnauthorized {
		t.Errorf("WriteUnauthorized() code = %s, want %s", resp.Error.Code, ErrCodeUnauthorized)
	}

	if resp.Error.Message != "authentication required" {
		t.Errorf("WriteUnauthorized() message = %s, want 'authentication required'", resp.Error.Message)
	}
}

func TestWriteForbidden(t *testing.T) {
	w := httptest.NewRecorder()
	message := "access denied"

	WriteForbidden(w, message)

	if w.Code != http.StatusForbidden {
		t.Errorf("WriteForbidden() status = %d, want %d", w.Code, http.StatusForbidden)
	}

	var resp Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp.Error == nil {
		t.Fatal("WriteForbidden() error = nil")
	}

	if resp.Error.Code != ErrCodeForbidden {
		t.Errorf("WriteForbidden() code = %s, want %s", resp.Error.Code, ErrCodeForbidden)
	}
}

func TestWriteNotFound(t *testing.T) {
	w := httptest.NewRecorder()
	message := "resource not found"

	WriteNotFound(w, message)

	if w.Code != http.StatusNotFound {
		t.Errorf("WriteNotFound() status = %d, want %d", w.Code, http.StatusNotFound)
	}

	var resp Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp.Error == nil {
		t.Fatal("WriteNotFound() error = nil")
	}

	if resp.Error.Code != ErrCodeNotFound {
		t.Errorf("WriteNotFound() code = %s, want %s", resp.Error.Code, ErrCodeNotFound)
	}
}

func TestWriteConflict(t *testing.T) {
	w := httptest.NewRecorder()
	message := "resource already exists"

	WriteConflict(w, message)

	if w.Code != http.StatusConflict {
		t.Errorf("WriteConflict() status = %d, want %d", w.Code, http.StatusConflict)
	}

	var resp Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp.Error == nil {
		t.Fatal("WriteConflict() error = nil")
	}

	if resp.Error.Code != ErrCodeConflict {
		t.Errorf("WriteConflict() code = %s, want %s", resp.Error.Code, ErrCodeConflict)
	}
}

func TestWriteInternal(t *testing.T) {
	w := httptest.NewRecorder()
	err := &testError{msg: "something went wrong"}

	WriteInternal(w, err)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("WriteInternal() status = %d, want %d", w.Code, http.StatusInternalServerError)
	}

	var resp Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp.Error == nil {
		t.Fatal("WriteInternal() error = nil")
	}

	if resp.Error.Code != ErrCodeInternal {
		t.Errorf("WriteInternal() code = %s, want %s", resp.Error.Code, ErrCodeInternal)
	}

	if resp.Error.Message != "internal server error" {
		t.Errorf("WriteInternal() message = %s, want 'internal server error'", resp.Error.Message)
	}
}

func TestWriteInternalWithContext(t *testing.T) {
	w := httptest.NewRecorder()
	err := &testError{msg: "database error"}
	context := "failed to save user"

	WriteInternalWithContext(w, err, context)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("WriteInternalWithContext() status = %d, want %d", w.Code, http.StatusInternalServerError)
	}

	var resp Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp.Error == nil {
		t.Fatal("WriteInternalWithContext() error = nil")
	}

	if resp.Error.Code != ErrCodeInternal {
		t.Errorf("WriteInternalWithContext() code = %s, want %s", resp.Error.Code, ErrCodeInternal)
	}

	// The user-facing message should still be generic
	if resp.Error.Message != "internal server error" {
		t.Errorf("WriteInternalWithContext() message = %s, want 'internal server error'", resp.Error.Message)
	}
}

func TestWriteNoActiveTask(t *testing.T) {
	w := httptest.NewRecorder()

	WriteNoActiveTask(w)

	if w.Code != http.StatusBadRequest {
		t.Errorf("WriteNoActiveTask() status = %d, want %d", w.Code, http.StatusBadRequest)
	}

	var resp Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp.Error == nil {
		t.Fatal("WriteNoActiveTask() error = nil")
	}

	if resp.Error.Code != ErrCodeNoActiveTask {
		t.Errorf("WriteNoActiveTask() code = %s, want %s", resp.Error.Code, ErrCodeNoActiveTask)
	}

	if resp.Error.Message != "no active task" {
		t.Errorf("WriteNoActiveTask() message = %s, want 'no active task'", resp.Error.Message)
	}
}

func TestWriteInvalidState(t *testing.T) {
	w := httptest.NewRecorder()
	current := "done"
	required := "planning"

	WriteInvalidState(w, current, required)

	if w.Code != http.StatusConflict {
		t.Errorf("WriteInvalidState() status = %d, want %d", w.Code, http.StatusConflict)
	}

	var resp Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp.Error == nil {
		t.Fatal("WriteInvalidState() error = nil")
	}

	if resp.Error.Code != ErrCodeInvalidState {
		t.Errorf("WriteInvalidState() code = %s, want %s", resp.Error.Code, ErrCodeInvalidState)
	}

	expectedMsg := "task is in 'done' state, requires 'planning'"
	if resp.Error.Message != expectedMsg {
		t.Errorf("WriteInvalidState() message = %s, want %s", resp.Error.Message, expectedMsg)
	}
}

func TestWriteInvalidStateForAction(t *testing.T) {
	w := httptest.NewRecorder()
	action := "plan"
	current := "implementing"

	WriteInvalidStateForAction(w, action, current)

	if w.Code != http.StatusConflict {
		t.Errorf("WriteInvalidStateForAction() status = %d, want %d", w.Code, http.StatusConflict)
	}

	var resp Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp.Error == nil {
		t.Fatal("WriteInvalidStateForAction() error = nil")
	}

	if resp.Error.Code != ErrCodeInvalidState {
		t.Errorf("WriteInvalidStateForAction() code = %s, want %s", resp.Error.Code, ErrCodeInvalidState)
	}

	expectedMsg := "cannot plan: task is in 'implementing' state"
	if resp.Error.Message != expectedMsg {
		t.Errorf("WriteInvalidStateForAction() message = %s, want %s", resp.Error.Message, expectedMsg)
	}
}

func TestWriteProviderError(t *testing.T) {
	w := httptest.NewRecorder()
	err := &testError{msg: "github API rate limit exceeded"}

	WriteProviderError(w, err)

	if w.Code != http.StatusBadGateway {
		t.Errorf("WriteProviderError() status = %d, want %d", w.Code, http.StatusBadGateway)
	}

	var resp Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp.Error == nil {
		t.Fatal("WriteProviderError() error = nil")
	}

	if resp.Error.Code != ErrCodeProviderError {
		t.Errorf("WriteProviderError() code = %s, want %s", resp.Error.Code, ErrCodeProviderError)
	}

	// Message should include the error
	if !strings.Contains(resp.Error.Message, "provider error") {
		t.Errorf("WriteProviderError() message = %s, want 'provider error: ...'", resp.Error.Message)
	}
}

func TestWriteAgentError(t *testing.T) {
	w := httptest.NewRecorder()
	err := &testError{msg: "Claude API error"}

	WriteAgentError(w, err)

	if w.Code != http.StatusBadGateway {
		t.Errorf("WriteAgentError() status = %d, want %d", w.Code, http.StatusBadGateway)
	}

	var resp Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp.Error == nil {
		t.Fatal("WriteAgentError() error = nil")
	}

	if resp.Error.Code != ErrCodeAgentError {
		t.Errorf("WriteAgentError() code = %s, want %s", resp.Error.Code, ErrCodeAgentError)
	}

	// Message should include the error
	if !strings.Contains(resp.Error.Message, "agent error") {
		t.Errorf("WriteAgentError() message = %s, want 'agent error: ...'", resp.Error.Message)
	}
}

func TestWriteBudgetExceeded(t *testing.T) {
	tests := []struct {
		name        string
		budgetType  string
		expectedMsg string
	}{
		{
			name:        "monthly budget",
			budgetType:  "monthly",
			expectedMsg: "monthly budget exceeded",
		},
		{
			name:        "task budget",
			budgetType:  "task",
			expectedMsg: "task budget exceeded",
		},
		{
			name:        "daily budget",
			budgetType:  "daily",
			expectedMsg: "daily budget exceeded",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			WriteBudgetExceeded(w, tt.budgetType)

			if w.Code != http.StatusPaymentRequired {
				t.Errorf("WriteBudgetExceeded() status = %d, want %d", w.Code, http.StatusPaymentRequired)
			}

			var resp Response
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Fatalf("failed to parse response: %v", err)
			}

			if resp.Error == nil {
				t.Fatal("WriteBudgetExceeded() error = nil")
			}

			if resp.Error.Code != ErrCodeBudgetExceeded {
				t.Errorf("WriteBudgetExceeded() code = %s, want %s", resp.Error.Code, ErrCodeBudgetExceeded)
			}

			if resp.Error.Message != tt.expectedMsg {
				t.Errorf("WriteBudgetExceeded() message = %s, want %s", resp.Error.Message, tt.expectedMsg)
			}
		})
	}
}

func TestWriteNotConfigured(t *testing.T) {
	tests := []struct {
		name    string
		feature string
		wantMsg string
	}{
		{
			name:    "GitHub provider",
			feature: "GitHub",
			wantMsg: "GitHub is not configured",
		},
		{
			name:    "OpenAI agent",
			feature: "OpenAI",
			wantMsg: "OpenAI is not configured",
		},
		{
			name:    "memory feature",
			feature: "Memory",
			wantMsg: "Memory is not configured",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			WriteNotConfigured(w, tt.feature)

			if w.Code != http.StatusNotFound {
				t.Errorf("WriteNotConfigured() status = %d, want %d", w.Code, http.StatusNotFound)
			}

			var resp Response
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Fatalf("failed to parse response: %v", err)
			}

			if resp.Error == nil {
				t.Fatal("WriteNotConfigured() error = nil")
			}

			if resp.Error.Code != ErrCodeNotConfigured {
				t.Errorf("WriteNotConfigured() code = %s, want %s", resp.Error.Code, ErrCodeNotConfigured)
			}

			if resp.Error.Message != tt.wantMsg {
				t.Errorf("WriteNotConfigured() message = %s, want %s", resp.Error.Message, tt.wantMsg)
			}
		})
	}
}

// Test error codes constants.
func TestErrorCodes(t *testing.T) {
	tests := []struct {
		name string
		code string
		want string
	}{
		{"Bad Request", ErrCodeBadRequest, "BAD_REQUEST"},
		{"Unauthorized", ErrCodeUnauthorized, "UNAUTHORIZED"},
		{"Forbidden", ErrCodeForbidden, "FORBIDDEN"},
		{"Not Found", ErrCodeNotFound, "NOT_FOUND"},
		{"Conflict", ErrCodeConflict, "CONFLICT"},
		{"Internal", ErrCodeInternal, "INTERNAL_ERROR"},
		{"Validation", ErrCodeValidation, "VALIDATION_ERROR"},
		{"No Active Task", ErrCodeNoActiveTask, "NO_ACTIVE_TASK"},
		{"Invalid State", ErrCodeInvalidState, "INVALID_STATE"},
		{"Provider Error", ErrCodeProviderError, "PROVIDER_ERROR"},
		{"Agent Error", ErrCodeAgentError, "AGENT_ERROR"},
		{"Budget Exceeded", ErrCodeBudgetExceeded, "BUDGET_EXCEEDED"},
		{"Not Configured", ErrCodeNotConfigured, "NOT_CONFIGURED"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.code != tt.want {
				t.Errorf("error code = %s, want %s", tt.code, tt.want)
			}
		})
	}
}

// Helper types

type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}
