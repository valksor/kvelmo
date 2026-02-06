package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/valksor/go-mehrhof/internal/conductor"
)

func decodeBodyMap(t *testing.T, rr *httptest.ResponseRecorder) map[string]any {
	t.Helper()

	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response body: %v", err)
	}

	return body
}

func TestWriteHelpers(t *testing.T) {
	tests := []struct {
		name       string
		call       func(http.ResponseWriter)
		wantStatus int
	}{
		{
			name: "WriteSuccess",
			call: func(w http.ResponseWriter) {
				WriteSuccess(w, map[string]string{"ok": "true"})
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "WriteBadRequest",
			call: func(w http.ResponseWriter) {
				WriteBadRequest(w, "bad input")
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "WriteNotFound",
			call: func(w http.ResponseWriter) {
				WriteNotFound(w, "missing")
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name: "WriteUnauthorized",
			call: func(w http.ResponseWriter) {
				WriteUnauthorized(w)
			},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "WriteNoActiveTask",
			call: func(w http.ResponseWriter) {
				WriteNoActiveTask(w)
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "WriteInvalidState",
			call: func(w http.ResponseWriter) {
				WriteInvalidState(w, "idle", "planning")
			},
			wantStatus: http.StatusConflict,
		},
		{
			name: "WriteInternal",
			call: func(w http.ResponseWriter) {
				WriteInternal(w, assertError("boom"))
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			tt.call(rr)
			if rr.Code != tt.wantStatus {
				t.Fatalf("%s status = %d, want %d", tt.name, rr.Code, tt.wantStatus)
			}
			if rr.Body.Len() == 0 {
				t.Fatalf("%s should write a response body", tt.name)
			}
		})
	}
}

func TestWriteError(t *testing.T) {
	rr := httptest.NewRecorder()
	WriteError(rr, http.StatusTeapot, "teapot", "short and stout")
	if rr.Code != http.StatusTeapot {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusTeapot)
	}
	body := decodeBodyMap(t, rr)
	errorInfo, ok := body["error"].(map[string]any)
	if !ok {
		t.Fatalf("expected error object, got %#v", body["error"])
	}
	if errorInfo["code"] != "teapot" {
		t.Fatalf("code = %v, want teapot", errorInfo["code"])
	}
}

func TestRequireHelpers(t *testing.T) {
	rr := httptest.NewRecorder()
	if ok := RequireConductor(rr, nil); ok {
		t.Fatalf("RequireConductor(nil) should return false")
	}
	if rr.Code != http.StatusNotFound {
		t.Fatalf("RequireConductor(nil) status = %d, want %d", rr.Code, http.StatusNotFound)
	}

	rr = httptest.NewRecorder()
	if ok := RequireWorkspace(rr, nil); ok {
		t.Fatalf("RequireWorkspace(nil) should return false")
	}
	if rr.Code != http.StatusNotFound {
		t.Fatalf("RequireWorkspace(nil) status = %d, want %d", rr.Code, http.StatusNotFound)
	}

	cond, err := conductor.New()
	if err != nil {
		t.Fatalf("conductor.New() failed: %v", err)
	}

	rr = httptest.NewRecorder()
	if ok := RequireConductor(rr, cond); !ok {
		t.Fatalf("RequireConductor(non-nil) should return true")
	}

	rr = httptest.NewRecorder()
	if ok := RequireActiveTask(rr, cond); ok {
		t.Fatalf("RequireActiveTask(with nil active task) should return false")
	}
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("RequireActiveTask status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
}

type assertError string

func (e assertError) Error() string { return string(e) }
