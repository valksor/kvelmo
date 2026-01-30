package apiclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNew(t *testing.T) {
	client := New("https://api.example.com", BearerAuth("token123"))

	if client == nil {
		t.Fatal("New() returned nil")
	}
	if client.baseURL != "https://api.example.com" {
		t.Errorf("baseURL = %q, want %q", client.baseURL, "https://api.example.com")
	}
	if client.authFn == nil {
		t.Error("authFn is nil")
	}
	if client.httpClient == nil {
		t.Error("httpClient is nil")
	}
}

func TestBearerAuth(t *testing.T) {
	authFn := BearerAuth("test-token")
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	authFn(req)

	got := req.Header.Get("Authorization")
	want := "Bearer test-token"
	if got != want {
		t.Errorf("Authorization header = %q, want %q", got, want)
	}
}

func TestBasicAuth(t *testing.T) {
	authFn := BasicAuth("user", "pass")
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	authFn(req)

	user, pass, ok := req.BasicAuth()
	if !ok {
		t.Error("BasicAuth() returned false")
	}
	if user != "user" || pass != "pass" {
		t.Errorf("BasicAuth credentials = (%q, %q), want (%q, %q)", user, pass, "user", "pass")
	}
}

func TestHeaderAuth(t *testing.T) {
	authFn := HeaderAuth(map[string]string{
		"X-API-Key":     "key123",
		"X-API-Version": "v2",
	})
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	authFn(req)

	if got := req.Header.Get("X-Api-Key"); got != "key123" {
		t.Errorf("X-API-Key = %q, want %q", got, "key123")
	}
	if got := req.Header.Get("X-Api-Version"); got != "v2" {
		t.Errorf("X-API-Version = %q, want %q", got, "v2")
	}
}

func TestClient_Do(t *testing.T) {
	type response struct {
		Message string `json:"message"`
		Count   int    `json:"count"`
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify headers
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("Authorization header = %q, want %q", r.Header.Get("Authorization"), "Bearer test-token")
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Content-Type = %q, want %q", r.Header.Get("Content-Type"), "application/json")
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response{Message: "success", Count: 42})
	}))
	defer server.Close()

	client := New(server.URL, BearerAuth("test-token"))

	var result response
	err := client.Do(context.Background(), http.MethodGet, "/test", nil, &result)
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}

	if result.Message != "success" {
		t.Errorf("result.Message = %q, want %q", result.Message, "success")
	}
	if result.Count != 42 {
		t.Errorf("result.Count = %d, want %d", result.Count, 42)
	}
}

func TestClient_Do_WithBody(t *testing.T) {
	type request struct {
		Name string `json:"name"`
	}
	type response struct {
		ID string `json:"id"`
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("Failed to decode request body: %v", err)
		}
		if req.Name != "test" {
			t.Errorf("request.Name = %q, want %q", req.Name, "test")
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response{ID: "123"})
	}))
	defer server.Close()

	client := New(server.URL, BearerAuth("token"))

	var result response
	err := client.Do(context.Background(), http.MethodPost, "/create", request{Name: "test"}, &result)
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}

	if result.ID != "123" {
		t.Errorf("result.ID = %q, want %q", result.ID, "123")
	}
}

func TestClient_Do_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error": "not found"}`))
	}))
	defer server.Close()

	client := New(server.URL, BearerAuth("token"))

	var result map[string]string
	err := client.Do(context.Background(), http.MethodGet, "/missing", nil, &result)
	if err == nil {
		t.Fatal("Do() expected error for 404 response")
	}

	// Check that error contains HTTP status code
	if httpErr, ok := err.(interface{ HTTPStatusCode() int }); ok {
		if httpErr.HTTPStatusCode() != http.StatusNotFound {
			t.Errorf("HTTPStatusCode() = %d, want %d", httpErr.HTTPStatusCode(), http.StatusNotFound)
		}
	}
}

func TestClient_DoRaw(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`raw response data`))
	}))
	defer server.Close()

	client := New(server.URL, BearerAuth("token"))

	body, err := client.DoRaw(context.Background(), http.MethodGet, "/raw", nil)
	if err != nil {
		t.Fatalf("DoRaw() error = %v", err)
	}

	if string(body) != "raw response data" {
		t.Errorf("body = %q, want %q", string(body), "raw response data")
	}
}

func TestClient_DoWithHeaders(t *testing.T) {
	type response struct {
		Version string `json:"version"`
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify custom header
		if r.Header.Get("X-Api-Version") != "2022-06-28" {
			t.Errorf("X-API-Version = %q, want %q", r.Header.Get("X-Api-Version"), "2022-06-28")
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response{Version: "v2"})
	}))
	defer server.Close()

	client := New(server.URL, BearerAuth("token"))

	var result response
	err := client.DoWithHeaders(context.Background(), http.MethodGet, "/test", nil, &result, map[string]string{
		"X-API-Version": "2022-06-28",
	})
	if err != nil {
		t.Fatalf("DoWithHeaders() error = %v", err)
	}

	if result.Version != "v2" {
		t.Errorf("result.Version = %q, want %q", result.Version, "v2")
	}
}

func TestClient_NilAuthFunc(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Should have no Authorization header
		if r.Header.Get("Authorization") != "" {
			t.Error("Expected no Authorization header")
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := New(server.URL, nil)

	err := client.Do(context.Background(), http.MethodGet, "/test", nil, nil)
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
}
