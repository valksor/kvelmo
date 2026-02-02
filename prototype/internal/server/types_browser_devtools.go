package server

// browserNetworkRequest is the request body for POST /api/v1/browser/network.
type browserNetworkRequest struct {
	TabID       string `json:"tab_id,omitempty"`
	Duration    int    `json:"duration"`      // seconds, default 5
	CaptureBody bool   `json:"capture_body"`  // capture request/response bodies
	MaxBodySize int    `json:"max_body_size"` // max body size in bytes, default 1MB
}

// browserConsoleRequest is the request body for POST /api/v1/browser/console.
type browserConsoleRequest struct {
	TabID    string `json:"tab_id,omitempty"`
	Duration int    `json:"duration"` // seconds, default 5
	Level    string `json:"level"`    // filter by level: "error", "warn", etc.
}

// browserWebSocketRequest is the request body for POST /api/v1/browser/websocket.
type browserWebSocketRequest struct {
	TabID    string `json:"tab_id,omitempty"`
	Duration int    `json:"duration"` // seconds, default 5
}

// browserSourceRequest is the request body for POST /api/v1/browser/source.
type browserSourceRequest struct {
	TabID string `json:"tab_id,omitempty"`
}

// browserScriptsRequest is the request body for POST /api/v1/browser/scripts.
type browserScriptsRequest struct {
	TabID string `json:"tab_id,omitempty"`
}

// browserStylesRequest is the request body for POST /api/v1/browser/styles.
type browserStylesRequest struct {
	TabID    string `json:"tab_id,omitempty"`
	Selector string `json:"selector"`
	Computed bool   `json:"computed"` // return computed styles (default true)
	Matched  bool   `json:"matched"`  // return matched CSS rules
}

// browserCoverageRequest is the request body for POST /api/v1/browser/coverage.
type browserCoverageRequest struct {
	TabID    string `json:"tab_id,omitempty"`
	Duration int    `json:"duration"`  // seconds, default 5
	TrackJS  bool   `json:"track_js"`  // track JS coverage (default true)
	TrackCSS bool   `json:"track_css"` // track CSS coverage (default true)
}
