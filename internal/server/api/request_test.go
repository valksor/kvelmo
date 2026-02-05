package api

import (
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestParseJSON(t *testing.T) {
	tests := []struct {
		name        string
		body        string
		want        interface{}
		wantSuccess bool
	}{
		{
			name:        "valid JSON",
			body:        `{"foo":"bar"}`,
			want:        map[string]any{"foo": "bar"},
			wantSuccess: true,
		},
		{
			name:        "empty body - uses defaults",
			body:        "",
			want:        map[string]any{},
			wantSuccess: true,
		},
		{
			name:        "nil body",
			body:        "",
			want:        map[string]any{},
			wantSuccess: false, // nil Body returns error
		},
		{
			name:        "invalid JSON",
			body:        `{invalid}`,
			want:        map[string]any{},
			wantSuccess: false,
		},
		{
			name:        "valid JSON array",
			body:        `["a","b","c"]`,
			want:        []any{"a", "b", "c"},
			wantSuccess: true,
		},
		{
			name:        "valid JSON with numbers",
			body:        `{"count":42,"price":3.14}`,
			want:        map[string]any{"count": float64(42), "price": 3.14},
			wantSuccess: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &fakeResponseWriter{}

			var r *http.Request
			if tt.name == "nil body" {
				r = &http.Request{Body: nil}
			} else {
				r = &http.Request{
					Body: io.NopCloser(strings.NewReader(tt.body)),
				}
			}

			// Create appropriate value type based on expected result
			var v interface{}
			switch tt.want.(type) {
			case map[string]any:
				m := make(map[string]any)
				v = &m
			case []any:
				s := make([]any, 0)
				v = &s
			default:
				m := make(map[string]any)
				v = &m
			}

			result := ParseJSON(w, r, v)

			if result != tt.wantSuccess {
				t.Errorf("ParseJSON() success = %v, want %v", result, tt.wantSuccess)
			}

			// For successful parses with non-empty body, verify the value was parsed correctly
			if tt.wantSuccess && tt.body != "" {
				// Dereference to get the actual value
				var actual interface{}
				switch ptr := v.(type) {
				case *map[string]any:
					actual = *ptr
				case *[]any:
					actual = *ptr
				}
				if !deepEqual(actual, tt.want) {
					t.Errorf("ParseJSON() value = %v, want %v", actual, tt.want)
				}
			}
		})
	}
}

func TestRequireJSON(t *testing.T) {
	tests := []struct {
		name        string
		body        string
		wantSuccess bool
	}{
		{
			name:        "valid JSON",
			body:        `{"foo":"bar"}`,
			wantSuccess: true,
		},
		{
			name:        "empty body - should fail",
			body:        "",
			wantSuccess: false,
		},
		{
			name:        "nil body - should fail",
			body:        "",
			wantSuccess: false,
		},
		{
			name:        "invalid JSON",
			body:        `{invalid}`,
			wantSuccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &fakeResponseWriter{}
			var r *http.Request
			if tt.name == "nil body - should fail" {
				r = &http.Request{Body: nil}
			} else {
				r = &http.Request{
					Body: io.NopCloser(strings.NewReader(tt.body)),
				}
			}
			m := make(map[string]any)
			v := &m

			result := RequireJSON(w, r, v)

			if result != tt.wantSuccess {
				t.Errorf("RequireJSON() success = %v, want %v", result, tt.wantSuccess)
			}
		})
	}
}

func TestPathParam(t *testing.T) {
	tests := []struct {
		name  string
		path  string
		param string
		want  string
	}{
		{
			name:  "extract task ID from /tasks/{id}",
			path:  "/api/v1/tasks/abc123",
			param: "id",
			want:  "abc123",
		},
		{
			name:  "extract task ID from /tasks/{id}/notes",
			path:  "/api/v1/tasks/def456/notes",
			param: "id",
			want:  "def456",
		},
		{
			name:  "extract task_id from /tasks/{id}",
			path:  "/api/v1/tasks/ghi789",
			param: "task_id",
			want:  "ghi789",
		},
		{
			name:  "extract quick task ID from /quick/{id}",
			path:  "/quick/jkl012",
			param: "id",
			want:  "jkl012",
		},
		{
			name:  "extract template name",
			path:  "/api/v1/templates/my-template",
			param: "name",
			want:  "my-template",
		},
		{
			name:  "extract agent name",
			path:  "/api/v1/agents/claude",
			param: "name",
			want:  "claude",
		},
		{
			name:  "extract queue name",
			path:  "/api/v1/queues/priority",
			param: "name",
			want:  "priority",
		},
		{
			name:  "no match - empty string",
			path:  "/api/v1/users",
			param: "id",
			want:  "",
		},
		{
			name:  "malformed path",
			path:  "/tasks",
			param: "id",
			want:  "",
		},
		{
			name:  "unknown param type",
			path:  "/api/v1/tasks/abc123",
			param: "unknown",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &http.Request{
				URL: &url.URL{Path: tt.path},
			}
			got := PathParam(r, tt.param)
			if got != tt.want {
				t.Errorf("PathParam() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestQueryString(t *testing.T) {
	tests := []struct {
		name  string
		query string
		key   string
		want  string
	}{
		{
			name:  "existing parameter",
			query: "foo=bar",
			key:   "foo",
			want:  "bar",
		},
		{
			name:  "missing parameter",
			query: "foo=bar",
			key:   "baz",
			want:  "",
		},
		{
			name:  "empty query",
			query: "",
			key:   "foo",
			want:  "",
		},
		{
			name:  "multiple parameters",
			query: "foo=bar&baz=qux",
			key:   "baz",
			want:  "qux",
		},
		{
			name:  "parameter with no value",
			query: "foo=&bar=value",
			key:   "foo",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &http.Request{
				URL: &url.URL{RawQuery: tt.query},
			}
			got := QueryString(r, tt.key)
			if got != tt.want {
				t.Errorf("QueryString() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestQueryStringDefault(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		key        string
		defaultVal string
		want       string
	}{
		{
			name:       "existing parameter",
			query:      "foo=bar",
			key:        "foo",
			defaultVal: "default",
			want:       "bar",
		},
		{
			name:       "missing parameter returns default",
			query:      "foo=bar",
			key:        "baz",
			defaultVal: "default",
			want:       "default",
		},
		{
			name:       "empty query returns default",
			query:      "",
			key:        "foo",
			defaultVal: "default",
			want:       "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &http.Request{
				URL: &url.URL{RawQuery: tt.query},
			}
			got := QueryStringDefault(r, tt.key, tt.defaultVal)
			if got != tt.want {
				t.Errorf("QueryStringDefault() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestQueryInt(t *testing.T) {
	tests := []struct {
		name  string
		query string
		key   string
		want  int
	}{
		{
			name:  "valid integer",
			query: "count=42",
			key:   "count",
			want:  42,
		},
		{
			name:  "negative integer",
			query: "value=-10",
			key:   "value",
			want:  -10,
		},
		{
			name:  "zero",
			query: "value=0",
			key:   "value",
			want:  0,
		},
		{
			name:  "missing parameter",
			query: "foo=bar",
			key:   "count",
			want:  0,
		},
		{
			name:  "invalid integer",
			query: "count=abc",
			key:   "count",
			want:  0,
		},
		{
			name:  "float string",
			query: "count=3.14",
			key:   "count",
			want:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &http.Request{
				URL: &url.URL{RawQuery: tt.query},
			}
			got := QueryInt(r, tt.key)
			if got != tt.want {
				t.Errorf("QueryInt() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestQueryIntDefault(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		key        string
		defaultVal int
		want       int
	}{
		{
			name:       "valid integer",
			query:      "count=42",
			key:        "count",
			defaultVal: 10,
			want:       42,
		},
		{
			name:       "missing parameter returns default",
			query:      "foo=bar",
			key:        "count",
			defaultVal: 10,
			want:       10,
		},
		{
			name:       "invalid integer returns default",
			query:      "count=abc",
			key:        "count",
			defaultVal: 10,
			want:       10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &http.Request{
				URL: &url.URL{RawQuery: tt.query},
			}
			got := QueryIntDefault(r, tt.key, tt.defaultVal)
			if got != tt.want {
				t.Errorf("QueryIntDefault() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestQueryBool(t *testing.T) {
	tests := []struct {
		name  string
		query string
		key   string
		want  bool
	}{
		{
			name:  "true lowercase",
			query: "enabled=true",
			key:   "enabled",
			want:  true,
		},
		{
			name:  "TRUE uppercase",
			query: "enabled=TRUE",
			key:   "enabled",
			want:  true,
		},
		{
			name:  "mixed case True",
			query: "enabled=True",
			key:   "enabled",
			want:  true,
		},
		{
			name:  "1 as boolean",
			query: "enabled=1",
			key:   "enabled",
			want:  true,
		},
		{
			name:  "yes as boolean",
			query: "enabled=yes",
			key:   "enabled",
			want:  true,
		},
		{
			name:  "YES as boolean",
			query: "enabled=YES",
			key:   "enabled",
			want:  true,
		},
		{
			name:  "false",
			query: "enabled=false",
			key:   "enabled",
			want:  false,
		},
		{
			name:  "0",
			query: "enabled=0",
			key:   "enabled",
			want:  false,
		},
		{
			name:  "no",
			query: "enabled=no",
			key:   "enabled",
			want:  false,
		},
		{
			name:  "random string",
			query: "enabled=maybe",
			key:   "enabled",
			want:  false,
		},
		{
			name:  "missing parameter",
			query: "foo=bar",
			key:   "enabled",
			want:  false,
		},
		{
			name:  "empty value",
			query: "enabled=",
			key:   "enabled",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &http.Request{
				URL: &url.URL{RawQuery: tt.query},
			}
			got := QueryBool(r, tt.key)
			if got != tt.want {
				t.Errorf("QueryBool() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestQueryBoolDefault(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		key        string
		defaultVal bool
		want       bool
	}{
		{
			name:       "true value overrides default",
			query:      "enabled=true",
			key:        "enabled",
			defaultVal: false,
			want:       true,
		},
		{
			name:       "missing parameter returns default true",
			query:      "foo=bar",
			key:        "enabled",
			defaultVal: true,
			want:       true,
		},
		{
			name:       "missing parameter returns default false",
			query:      "foo=bar",
			key:        "enabled",
			defaultVal: false,
			want:       false,
		},
		{
			name:       "empty value returns default",
			query:      "enabled=",
			key:        "enabled",
			defaultVal: true,
			want:       true,
		},
		{
			name:       "false value overrides default true",
			query:      "enabled=false",
			key:        "enabled",
			defaultVal: true,
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &http.Request{
				URL: &url.URL{RawQuery: tt.query},
			}
			got := QueryBoolDefault(r, tt.key, tt.defaultVal)
			if got != tt.want {
				t.Errorf("QueryBoolDefault() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestQueryStrings(t *testing.T) {
	tests := []struct {
		name  string
		query string
		key   string
		want  []string
	}{
		{
			name:  "single value",
			query: "tags=foo",
			key:   "tags",
			want:  []string{"foo"},
		},
		{
			name:  "multiple values",
			query: "tags=foo&tags=bar&tags=baz",
			key:   "tags",
			want:  []string{"foo", "bar", "baz"},
		},
		{
			name:  "no values",
			query: "foo=bar",
			key:   "tags",
			want:  nil,
		},
		{
			name:  "empty query",
			query: "",
			key:   "tags",
			want:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &http.Request{
				URL: &url.URL{RawQuery: tt.query},
			}
			got := QueryStrings(r, tt.key)
			if !slicesEqual(got, tt.want) {
				t.Errorf("QueryStrings() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAcceptsJSON(t *testing.T) {
	tests := []struct {
		name   string
		accept string
		want   bool
	}{
		{
			name:   "application/json",
			accept: "application/json",
			want:   true,
		},
		{
			name:   "application/json with charset",
			accept: "application/json; charset=utf-8",
			want:   true,
		},
		{
			name:   "*/* wildcard",
			accept: "*/*",
			want:   true,
		},
		{
			name:   "text/html",
			accept: "text/html",
			want:   false,
		},
		{
			name:   "empty accept",
			accept: "",
			want:   false,
		},
		{
			name:   "multiple types including json",
			accept: "text/html, application/json, */*",
			want:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &http.Request{
				Header: http.Header{"Accept": []string{tt.accept}},
			}
			got := AcceptsJSON(r)
			if got != tt.want {
				t.Errorf("AcceptsJSON() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAcceptsHTML(t *testing.T) {
	tests := []struct {
		name   string
		accept string
		want   bool
	}{
		{
			name:   "text/html",
			accept: "text/html",
			want:   true,
		},
		{
			name:   "text/html with charset",
			accept: "text/html; charset=utf-8",
			want:   true,
		},
		{
			name:   "empty accept header",
			accept: "",
			want:   true,
		},
		{
			name:   "application/json",
			accept: "application/json",
			want:   false,
		},
		{
			name:   "multiple types including html",
			accept: "text/html, application/json",
			want:   true,
		},
		{
			name:   "application/xhtml+xml",
			accept: "application/xhtml+xml",
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &http.Request{
				Header: http.Header{"Accept": []string{tt.accept}},
			}
			got := AcceptsHTML(r)
			if got != tt.want {
				t.Errorf("AcceptsHTML() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestContentType(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		want        string
	}{
		{
			name:        "simple content type",
			contentType: "application/json",
			want:        "application/json",
		},
		{
			name:        "content type with charset",
			contentType: "application/json; charset=utf-8",
			want:        "application/json",
		},
		{
			name:        "content type with boundary",
			contentType: "multipart/form-data; boundary=----WebKitFormBoundary",
			want:        "multipart/form-data",
		},
		{
			name:        "content type with multiple parameters",
			contentType: "text/html; charset=utf-8; version=1",
			want:        "text/html",
		},
		{
			name:        "content type with spaces",
			contentType: " application/json ; charset=utf-8 ",
			want:        "application/json",
		},
		{
			name:        "empty content type",
			contentType: "",
			want:        "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &http.Request{
				Header: http.Header{"Content-Type": []string{tt.contentType}},
			}
			got := ContentType(r)
			if got != tt.want {
				t.Errorf("ContentType() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestIsJSONContent(t *testing.T) {
	tests := []struct {
		name string
		ct   string
		want bool
	}{
		{
			name: "application/json",
			ct:   "application/json",
			want: true,
		},
		{
			name: "application/json with charset",
			ct:   "application/json; charset=utf-8",
			want: true,
		},
		{
			name: "text/html",
			ct:   "text/html",
			want: false,
		},
		{
			name: "application/x-www-form-urlencoded",
			ct:   "application/x-www-form-urlencoded",
			want: false,
		},
		{
			name: "empty",
			ct:   "",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &http.Request{
				Header: http.Header{"Content-Type": []string{tt.ct}},
			}
			got := IsJSONContent(r)
			if got != tt.want {
				t.Errorf("IsJSONContent() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsFormContent(t *testing.T) {
	tests := []struct {
		name string
		ct   string
		want bool
	}{
		{
			name: "application/x-www-form-urlencoded",
			ct:   "application/x-www-form-urlencoded",
			want: true,
		},
		{
			name: "multipart/form-data",
			ct:   "multipart/form-data",
			want: true,
		},
		{
			name: "multipart/form-data with boundary",
			ct:   "multipart/form-data; boundary=----WebKitFormBoundary",
			want: true,
		},
		{
			name: "application/json",
			ct:   "application/json",
			want: false,
		},
		{
			name: "text/html",
			ct:   "text/html",
			want: false,
		},
		{
			name: "empty",
			ct:   "",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &http.Request{
				Header: http.Header{"Content-Type": []string{tt.ct}},
			}
			got := IsFormContent(r)
			if got != tt.want {
				t.Errorf("IsFormContent() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Helper functions

func slicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}

func deepEqual(a, b interface{}) bool {
	switch va := a.(type) {
	case map[string]any:
		vb, ok := b.(map[string]any)
		if !ok || len(va) != len(vb) {
			return false
		}
		for k, av := range va {
			bv, ok := vb[k]
			if !ok || !deepEqual(av, bv) {
				return false
			}
		}

		return true
	case []any:
		vb, ok := b.([]any)
		if !ok || len(va) != len(vb) {
			return false
		}
		for i := range va {
			if !deepEqual(va[i], vb[i]) {
				return false
			}
		}

		return true
	default:
		return a == b
	}
}

// fakeResponseWriter is a minimal http.ResponseWriter for testing.
type fakeResponseWriter struct {
	header http.Header
	body   []byte
	status int
}

func (w *fakeResponseWriter) Header() http.Header {
	if w.header == nil {
		w.header = make(http.Header)
	}

	return w.header
}

func (w *fakeResponseWriter) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	w.body = append(w.body, b...)

	return len(b), nil
}

func (w *fakeResponseWriter) WriteHeader(statusCode int) {
	w.status = statusCode
}
