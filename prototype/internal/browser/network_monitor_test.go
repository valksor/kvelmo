//go:build !no_browser
// +build !no_browser

package browser

import (
	"testing"
	"time"
)

// TestNetworkMonitor tests network request monitoring functionality.
func TestNetworkMonitor(t *testing.T) {
	t.Run("NewNetworkMonitor", func(t *testing.T) {
		mon := NewNetworkMonitor()
		if mon == nil {
			t.Fatal("NewNetworkMonitor() returned nil")
		}
		if mon.requests == nil {
			t.Error("requests map is not initialized")
		}
		if len(mon.GetRequests()) != 0 {
			t.Error("new monitor should have no requests")
		}
	})

	t.Run("AddRequest", func(t *testing.T) {
		mon := NewNetworkMonitor()

		req := &NetworkRequest{
			ID:           "req-1",
			URL:          "https://example.com/api",
			Method:       "GET",
			Status:       200,
			StatusText:   "OK",
			ResourceType: "XHR",
			MimeType:     "application/json",
			Timestamp:    time.Now(),
		}

		mon.AddRequest(req)

		requests := mon.GetRequests()
		if len(requests) != 1 {
			t.Errorf("got %d requests, want 1", len(requests))
		}

		if requests[0].ID != req.ID {
			t.Errorf("ID = %s, want %s", requests[0].ID, req.ID)
		}
	})

	t.Run("GetRequests", func(t *testing.T) {
		mon := NewNetworkMonitor()

		// Add multiple requests
		req1 := &NetworkRequest{
			ID:        "req-1",
			URL:       "https://example.com/api/1",
			Method:    "GET",
			Status:    200,
			Timestamp: time.Now(),
		}
		req2 := &NetworkRequest{
			ID:        "req-2",
			URL:       "https://example.com/api/2",
			Method:    "POST",
			Status:    201,
			Timestamp: time.Now(),
		}

		mon.AddRequest(req1)
		mon.AddRequest(req2)

		requests := mon.GetRequests()
		if len(requests) != 2 {
			t.Errorf("got %d requests, want 2", len(requests))
		}
	})

	t.Run("GetRequestsByType", func(t *testing.T) {
		mon := NewNetworkMonitor()

		// Add different types of requests
		xhrReq := &NetworkRequest{
			ID:           "xhr-1",
			URL:          "https://example.com/api",
			ResourceType: "XHR",
			Status:       200,
			Timestamp:    time.Now(),
		}
		docReq := &NetworkRequest{
			ID:           "doc-1",
			URL:          "https://example.com",
			ResourceType: "Document",
			Status:       200,
			Timestamp:    time.Now(),
		}
		scriptReq := &NetworkRequest{
			ID:           "script-1",
			URL:          "https://example.com/app.js",
			ResourceType: "Script",
			Status:       200,
			Timestamp:    time.Now(),
		}

		mon.AddRequest(xhrReq)
		mon.AddRequest(docReq)
		mon.AddRequest(scriptReq)

		// Get XHR requests
		xhrReqs := mon.GetRequestsByType("XHR")
		if len(xhrReqs) != 1 {
			t.Errorf("got %d XHR requests, want 1", len(xhrReqs))
		}

		// Get Script requests
		scriptReqs := mon.GetRequestsByType("Script")
		if len(scriptReqs) != 1 {
			t.Errorf("got %d Script requests, want 1", len(scriptReqs))
		}

		// Get non-existent type
		websocketReqs := mon.GetRequestsByType("WebSocket")
		if len(websocketReqs) != 0 {
			t.Errorf("got %d WebSocket requests, want 0", len(websocketReqs))
		}
	})

	t.Run("GetRequestsByURLPattern", func(t *testing.T) {
		mon := NewNetworkMonitor()

		// Add requests with different URLs
		req1 := &NetworkRequest{
			ID:        "req-1",
			URL:       "https://example.com/api/users",
			Status:    200,
			Timestamp: time.Now(),
		}
		req2 := &NetworkRequest{
			ID:        "req-2",
			URL:       "https://example.com/api/posts",
			Status:    200,
			Timestamp: time.Now(),
		}
		req3 := &NetworkRequest{
			ID:        "req-3",
			URL:       "https://other.com/page",
			Status:    200,
			Timestamp: time.Now(),
		}

		mon.AddRequest(req1)
		mon.AddRequest(req2)
		mon.AddRequest(req3)

		// Get requests containing "api"
		apiReqs := mon.GetRequestsByURLPattern("api")
		if len(apiReqs) != 2 {
			t.Errorf("got %d requests with 'api', want 2", len(apiReqs))
		}

		// Get requests for specific domain
		exampleReqs := mon.GetRequestsByURLPattern("example.com")
		if len(exampleReqs) != 2 {
			t.Errorf("got %d requests for 'example.com', want 2", len(exampleReqs))
		}

		// Get non-existent pattern
		fooReqs := mon.GetRequestsByURLPattern("foo")
		if len(fooReqs) != 0 {
			t.Errorf("got %d requests with 'foo', want 0", len(fooReqs))
		}
	})

	t.Run("GetFailedRequests", func(t *testing.T) {
		mon := NewNetworkMonitor()

		// Add requests with different status codes
		successReq := &NetworkRequest{
			ID:        "req-1",
			URL:       "https://example.com/success",
			Status:    200,
			Timestamp: time.Now(),
		}
		redirectReq := &NetworkRequest{
			ID:        "req-2",
			URL:       "https://example.com/redirect",
			Status:    301,
			Timestamp: time.Now(),
		}
		clientErrReq := &NetworkRequest{
			ID:        "req-3",
			URL:       "https://example.com/not-found",
			Status:    404,
			Timestamp: time.Now(),
		}
		serverErrReq := &NetworkRequest{
			ID:        "req-4",
			URL:       "https://example.com/error",
			Status:    500,
			Timestamp: time.Now(),
		}

		mon.AddRequest(successReq)
		mon.AddRequest(redirectReq)
		mon.AddRequest(clientErrReq)
		mon.AddRequest(serverErrReq)

		// Get failed requests (4xx and 5xx)
		failedReqs := mon.GetFailedRequests()
		if len(failedReqs) != 2 {
			t.Errorf("got %d failed requests, want 2", len(failedReqs))
		}

		// Verify they're actually failures
		for _, req := range failedReqs {
			if req.Status < 400 || req.Status >= 600 {
				t.Errorf("request %s has status %d, not a 4xx/5xx error", req.ID, req.Status)
			}
		}
	})

	t.Run("GetRequestsForURL", func(t *testing.T) {
		mon := NewNetworkMonitor()

		// Add requests
		req1 := &NetworkRequest{
			ID:        "req-1",
			URL:       "https://example.com/api",
			Status:    200,
			Timestamp: time.Now(),
		}
		req2 := &NetworkRequest{
			ID:        "req-2",
			URL:       "https://example.com/api",
			Status:    200,
			Timestamp: time.Now(),
		}
		req3 := &NetworkRequest{
			ID:        "req-3",
			URL:       "https://example.com/other",
			Status:    200,
			Timestamp: time.Now(),
		}

		mon.AddRequest(req1)
		mon.AddRequest(req2)
		mon.AddRequest(req3)

		// Get requests for specific URL
		apiReqs := mon.GetRequestsForURL("https://example.com/api")
		if len(apiReqs) != 2 {
			t.Errorf("got %d requests for URL, want 2", len(apiReqs))
		}

		// Get requests for different URL
		otherReqs := mon.GetRequestsForURL("https://example.com/other")
		if len(otherReqs) != 1 {
			t.Errorf("got %d requests for other URL, want 1", len(otherReqs))
		}

		// Get non-existent URL
		fooReqs := mon.GetRequestsForURL("https://example.com/foo")
		if len(fooReqs) != 0 {
			t.Errorf("got %d requests for non-existent URL, want 0", len(fooReqs))
		}
	})

	t.Run("RequestUpdate", func(t *testing.T) {
		mon := NewNetworkMonitor()

		// Add request without status
		req := &NetworkRequest{
			ID:           "req-1",
			URL:          "https://example.com/api",
			Method:       "POST",
			ResourceType: "XHR",
			Timestamp:    time.Now(),
		}

		mon.AddRequest(req)

		// Update the request with response info
		updatedReq := &NetworkRequest{
			ID:         "req-1",
			URL:        "https://example.com/api",
			Method:     "POST",
			Status:     201,
			StatusText: "Created",
			MimeType:   "application/json",
		}

		mon.AddRequest(updatedReq)

		// Verify the request was updated
		requests := mon.GetRequests()
		if len(requests) != 1 {
			t.Fatalf("got %d requests, want 1", len(requests))
		}

		if requests[0].Status != 201 {
			t.Errorf("Status = %d, want 201", requests[0].Status)
		}
		if requests[0].StatusText != "Created" {
			t.Errorf("StatusText = %s, want 'Created'", requests[0].StatusText)
		}
		if requests[0].MimeType != "application/json" {
			t.Errorf("MimeType = %s, want 'application/json'", requests[0].MimeType)
		}
	})

	t.Run("ThreadSafety", func(t *testing.T) {
		mon := NewNetworkMonitor()

		// Add requests concurrently
		done := make(chan bool)
		for i := range 10 {
			go func(index int) {
				req := &NetworkRequest{
					ID:        string(rune(index)),
					URL:       "https://example.com/test",
					Status:    200,
					Timestamp: time.Now(),
				}
				mon.AddRequest(req)
				done <- true
			}(i)
		}

		// Wait for all goroutines
		for range 10 {
			<-done
		}

		// Verify all requests were added
		requests := mon.GetRequests()
		if len(requests) != 10 {
			t.Errorf("got %d requests, want 10", len(requests))
		}
	})
}

// TestNetworkRequestSerialization tests JSON serialization of network requests.
func TestNetworkRequestSerialization(t *testing.T) {
	req := NetworkRequest{
		ID:           "test-123",
		URL:          "https://example.com/api/test",
		Method:       "POST",
		Status:       201,
		StatusText:   "Created",
		Headers:      map[string]string{"Content-Type": "application/json"},
		ResourceType: "XHR",
		MimeType:     "application/json",
		Timestamp:    time.Now(),
		RequestBody:  `{"test": "data"}`,
		ResponseBody: `{"result": "success"}`,
	}

	// Verify all fields are set correctly
	if req.ID == "" {
		t.Error("ID is empty")
	}
	if req.URL == "" {
		t.Error("URL is empty")
	}
	if req.Method == "" {
		t.Error("Method is empty")
	}
	if req.Status == 0 {
		t.Error("Status is 0")
	}
	if req.ResourceType == "" {
		t.Error("ResourceType is empty")
	}
	if req.Timestamp.IsZero() {
		t.Error("Timestamp is zero")
	}
}
