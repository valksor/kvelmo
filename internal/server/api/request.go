package api

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
)

// ParseJSON parses a JSON request body into the given struct.
// Returns false and writes an error response if parsing fails.
func ParseJSON(w http.ResponseWriter, r *http.Request, v interface{}) bool {
	if r.Body == nil {
		WriteBadRequest(w, "request body is required")

		return false
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		WriteBadRequest(w, "failed to read request body")

		return false
	}

	if len(body) == 0 {
		// Allow empty body - just use default values
		return true
	}

	if err := json.Unmarshal(body, v); err != nil {
		WriteBadRequest(w, "invalid JSON: "+err.Error())

		return false
	}

	return true
}

// RequireJSON parses a JSON request body, requiring it to be non-empty.
// Returns false and writes an error response if parsing fails.
func RequireJSON(w http.ResponseWriter, r *http.Request, v interface{}) bool {
	if r.Body == nil {
		WriteBadRequest(w, "request body is required")

		return false
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		WriteBadRequest(w, "failed to read request body")

		return false
	}

	if len(body) == 0 {
		WriteBadRequest(w, "request body is required")

		return false
	}

	if err := json.Unmarshal(body, v); err != nil {
		WriteBadRequest(w, "invalid JSON: "+err.Error())

		return false
	}

	return true
}

// PathParam extracts a path parameter value.
// This is a placeholder - actual implementation depends on router used.
func PathParam(r *http.Request, name string) string {
	// For chi router, this would be:
	// return chi.URLParam(r, name)
	//
	// For gorilla mux:
	// return mux.Vars(r)[name]
	//
	// For now, extract from path manually based on expected patterns
	return extractPathParam(r.URL.Path, name)
}

// extractPathParam extracts a path parameter from common URL patterns.
// This handles patterns like /api/v1/tasks/{id}/notes.
func extractPathParam(path, name string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")

	// Common patterns where we need to extract IDs
	switch name {
	case "id", "task_id", "taskId":
		// Look for patterns like /tasks/{id} or /tasks/{id}/...
		for i, part := range parts {
			if part == "tasks" && i+1 < len(parts) {
				return parts[i+1]
			}
			if part == "quick" && i+1 < len(parts) {
				return parts[i+1]
			}
		}
	case "name":
		// Look for patterns like /templates/{name}
		for i, part := range parts {
			if (part == "templates" || part == "agents" || part == "queues") && i+1 < len(parts) {
				return parts[i+1]
			}
		}
	}

	return ""
}

// QueryString returns a query parameter value or empty string if not present.
func QueryString(r *http.Request, name string) string {
	return r.URL.Query().Get(name)
}

// QueryStringDefault returns a query parameter value or the default if not present.
func QueryStringDefault(r *http.Request, name, defaultVal string) string {
	val := r.URL.Query().Get(name)
	if val == "" {
		return defaultVal
	}

	return val
}

// QueryInt returns a query parameter as an integer, or 0 if not present/invalid.
func QueryInt(r *http.Request, name string) int {
	val := r.URL.Query().Get(name)
	if val == "" {
		return 0
	}
	n, err := strconv.Atoi(val)
	if err != nil {
		return 0
	}

	return n
}

// QueryIntDefault returns a query parameter as an integer or the default.
func QueryIntDefault(r *http.Request, name string, defaultVal int) int {
	val := r.URL.Query().Get(name)
	if val == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(val)
	if err != nil {
		return defaultVal
	}

	return n
}

// QueryBool returns a query parameter as a boolean.
// Accepts "true", "1", "yes" as true; everything else is false.
func QueryBool(r *http.Request, name string) bool {
	val := strings.ToLower(r.URL.Query().Get(name))

	return val == "true" || val == "1" || val == "yes"
}

// QueryBoolDefault returns a query parameter as a boolean or the default.
func QueryBoolDefault(r *http.Request, name string, defaultVal bool) bool {
	val := r.URL.Query().Get(name)
	if val == "" {
		return defaultVal
	}
	lower := strings.ToLower(val)

	return lower == "true" || lower == "1" || lower == "yes"
}

// QueryStrings returns all values for a query parameter (for arrays).
func QueryStrings(r *http.Request, name string) []string {
	return r.URL.Query()[name]
}

// AcceptsJSON returns true if the request accepts JSON responses.
func AcceptsJSON(r *http.Request) bool {
	accept := r.Header.Get("Accept")

	return strings.Contains(accept, "application/json") || strings.Contains(accept, "*/*")
}

// AcceptsHTML returns true if the request accepts HTML responses.
func AcceptsHTML(r *http.Request) bool {
	accept := r.Header.Get("Accept")

	return strings.Contains(accept, "text/html") || accept == ""
}

// IsHTMXRequest returns true if this is an HTMX request.
func IsHTMXRequest(r *http.Request) bool {
	return r.Header.Get("Hx-Request") == "true"
}

// HTMXTrigger returns the HX-Trigger header value (element that triggered the request).
func HTMXTrigger(r *http.Request) string {
	return r.Header.Get("Hx-Trigger")
}

// HTMXTarget returns the HX-Target header value (target element ID).
func HTMXTarget(r *http.Request) string {
	return r.Header.Get("Hx-Target")
}

// HTMXCurrentURL returns the HX-Current-URL header value.
func HTMXCurrentURL(r *http.Request) string {
	return r.Header.Get("Hx-Current-Url")
}

// ContentType returns the Content-Type header without parameters.
func ContentType(r *http.Request) string {
	ct := r.Header.Get("Content-Type")
	if idx := strings.Index(ct, ";"); idx != -1 {
		ct = ct[:idx]
	}

	return strings.TrimSpace(ct)
}

// IsJSONContent returns true if the request content type is JSON.
func IsJSONContent(r *http.Request) bool {
	return ContentType(r) == "application/json"
}

// IsFormContent returns true if the request is form-encoded.
func IsFormContent(r *http.Request) bool {
	ct := ContentType(r)

	return ct == "application/x-www-form-urlencoded" || ct == "multipart/form-data"
}
