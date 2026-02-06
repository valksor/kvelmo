package youtrack

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	providererrors "github.com/valksor/go-toolkit/errors"
)

func newYouTrackTestClient(serverURL string) *Client {
	c := NewClient("token-123", "")
	c.baseURL = serverURL

	return c
}

func TestClientDoRequestSuccessAndHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer token-123" {
			t.Fatalf("Authorization = %q", got)
		}
		if got := r.Header.Get("Accept"); got != "application/json" {
			t.Fatalf("Accept = %q", got)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Fatalf("Content-Type = %q", got)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":{"id":"ISSUE-1"}}`))
	}))
	defer server.Close()

	client := newYouTrackTestClient(server.URL)
	var out map[string]any
	err := client.doRequest(context.Background(), http.MethodPost, "/issues", strings.NewReader(`{"x":1}`), &out)
	if err != nil {
		t.Fatalf("doRequest error: %v", err)
	}
	data, ok := out["data"].(map[string]any)
	if !ok || data["id"] != "ISSUE-1" {
		t.Fatalf("unexpected response payload: %#v", out)
	}
}

func TestClientDoRequestErrorMapping(t *testing.T) {
	tests := []struct {
		name string
		want error
		code int
	}{
		{name: "unauthorized", code: http.StatusUnauthorized, want: providererrors.ErrUnauthorized},
		{name: "not found", code: http.StatusNotFound, want: providererrors.ErrNotFound},
		{name: "rate limited", code: http.StatusTooManyRequests, want: providererrors.ErrRateLimited},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.code)
				_, _ = w.Write([]byte(`{"error":"x"}`))
			}))
			defer server.Close()

			client := newYouTrackTestClient(server.URL)
			var out map[string]any
			err := client.doRequest(context.Background(), http.MethodGet, "/issues", nil, &out)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !errors.Is(err, tt.want) {
				t.Fatalf("error = %v, want wrapped %v", err, tt.want)
			}
		})
	}
}

func TestClientDoRequestDecodeError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{not-json`))
	}))
	defer server.Close()

	client := newYouTrackTestClient(server.URL)
	var out map[string]any
	err := client.doRequest(context.Background(), http.MethodGet, "/issues", nil, &out)
	if err == nil || !strings.Contains(err.Error(), "decode response") {
		t.Fatalf("expected decode response error, got %v", err)
	}
}

func TestClientDoRequestWithRetryContextCanceled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"error":"temporary"}`))
	}))
	defer server.Close()

	client := newYouTrackTestClient(server.URL)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var out map[string]any
	err := client.doRequestWithRetry(ctx, http.MethodGet, "/issues", nil, &out)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context canceled, got %v", err)
	}
}

func TestClientGetIssuesByQueryBuildsParameters(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		values, err := url.ParseQuery(r.URL.RawQuery)
		if err != nil {
			t.Fatalf("parse query: %v", err)
		}
		if values.Get("query") != "Unresolved tag: backend" {
			t.Fatalf("query param = %q", values.Get("query"))
		}
		if values.Get("$top") != "10" || values.Get("$skip") != "5" {
			t.Fatalf("pagination params top=%q skip=%q", values.Get("$top"), values.Get("$skip"))
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":[{"id":"i1","idReadable":"ABC-1","summary":"x"}]}`))
	}))
	defer server.Close()

	client := newYouTrackTestClient(server.URL)
	issues, err := client.GetIssuesByQuery(context.Background(), "Unresolved tag: backend", 10, 5)
	if err != nil {
		t.Fatalf("GetIssuesByQuery error: %v", err)
	}
	if len(issues) != 1 || issues[0].ID != "i1" {
		t.Fatalf("unexpected issues: %#v", issues)
	}
}

func TestClientAddCommentAndTagNoReturn(t *testing.T) {
	t.Run("AddComment no comment returned", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				t.Fatalf("method = %s, want POST", r.Method)
			}
			body, _ := io.ReadAll(r.Body)
			var payload map[string]string
			if err := json.Unmarshal(body, &payload); err != nil {
				t.Fatalf("decode payload: %v", err)
			}
			if payload["text"] != "hello" {
				t.Fatalf("payload text = %q", payload["text"])
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"data":[]}`))
		}))
		defer server.Close()

		client := newYouTrackTestClient(server.URL)
		comment, err := client.AddComment(context.Background(), "ABC-1", "hello")
		if err == nil || !strings.Contains(err.Error(), "no comment returned") {
			t.Fatalf("expected no comment returned error, got comment=%#v err=%v", comment, err)
		}
	})

	t.Run("AddTag no tag returned", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				t.Fatalf("method = %s, want POST", r.Method)
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"data":[]}`))
		}))
		defer server.Close()

		client := newYouTrackTestClient(server.URL)
		tag, err := client.AddTag(context.Background(), "ABC-1", "backend")
		if err == nil || !strings.Contains(err.Error(), "no tag returned") {
			t.Fatalf("expected no tag returned error, got tag=%#v err=%v", tag, err)
		}
	})
}

func TestClientDownloadAttachment(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !strings.Contains(r.URL.Path, "/attachments/att-1/content") {
				t.Fatalf("path = %q", r.URL.Path)
			}
			w.Header().Set("Content-Disposition", `attachment; filename="file.txt"`)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("payload"))
		}))
		defer server.Close()

		client := newYouTrackTestClient(server.URL)
		rc, disposition, err := client.DownloadAttachment(context.Background(), "att-1")
		if err != nil {
			t.Fatalf("DownloadAttachment error: %v", err)
		}
		defer func() { _ = rc.Close() }()
		if !strings.Contains(disposition, "file.txt") {
			t.Fatalf("Content-Disposition = %q", disposition)
		}
		data, _ := io.ReadAll(rc)
		if string(data) != "payload" {
			t.Fatalf("downloaded payload = %q", string(data))
		}
	})

	t.Run("error mapping", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte("forbidden"))
		}))
		defer server.Close()

		client := newYouTrackTestClient(server.URL)
		rc, disposition, err := client.DownloadAttachment(context.Background(), "att-2")
		if rc != nil || disposition != "" {
			t.Fatalf("expected nil body and empty disposition on error")
		}
		if !errors.Is(err, providererrors.ErrRateLimited) {
			t.Fatalf("expected rate-limited wrapped error, got %v", err)
		}
	})
}
