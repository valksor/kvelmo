//go:build !no_mcp
// +build !no_mcp

package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"

	"golang.org/x/time/rate"
)

const (
	maxRequestSize    = 10 * 1024 * 1024 // 10MB max request size
	maxConcurrentReqs = 10               // Max concurrent requests
	defaultRateLimit  = 10               // Default requests per second
	defaultRateBurst  = 20               // Default burst size
	ioBufferSize      = 32 * 1024        // 32KB buffer for stdio I/O
)

// ServerOption configures the MCP server.
type ServerOption func(*Server)

// WithRateLimit sets a custom rate limit (requests per second) and burst size.
func WithRateLimit(ratePerSec float64, burst int) ServerOption {
	return func(s *Server) {
		if ratePerSec > 0 && burst > 0 {
			s.rateLimiter = rate.NewLimiter(rate.Limit(ratePerSec), burst)
		}
	}
}

// Server implements an MCP server over stdio.
type Server struct {
	toolRegistry *ToolRegistry
	initialized  atomic.Bool
	serverInfo   ServerInfo
	semaphore    chan struct{} // Concurrency limiter
	shutdownChan chan struct{} // Shutdown signal
	shutdownOnce sync.Once     // Ensure shutdownChan is only closed once
	rateLimiter  *rate.Limiter // Rate limiter for tool calls
}

// NewServer creates a new MCP server.
func NewServer(toolRegistry *ToolRegistry, opts ...ServerOption) *Server {
	s := &Server{
		toolRegistry: toolRegistry,
		serverInfo: ServerInfo{
			Name:    "go-mehrhof",
			Version: "1.0.0",
		},
		semaphore:    make(chan struct{}, maxConcurrentReqs),
		shutdownChan: make(chan struct{}),
		rateLimiter:  rate.NewLimiter(rate.Limit(defaultRateLimit), defaultRateBurst),
	}

	// Apply options
	for _, opt := range opts {
		opt(s)
	}

	return s
}

// Serve starts the MCP server, reading requests from stdin and writing responses to stdout.
func (s *Server) Serve(ctx context.Context) error {
	// Use buffered I/O for performance
	// Note: Buffer size is for reading individual JSON-RPC lines, not entire requests.
	// MCP protocol sends one JSON-RPC request per line, so the buffer only needs to handle
	// the longest single line. The maxRequestSize (10MB) limit is applied after reading.
	reader := bufio.NewReaderSize(os.Stdin, ioBufferSize)
	writer := bufio.NewWriter(os.Stdout)
	defer func() {
		_ = writer.Flush()
	}()

	slog.Info("MCP server started on stdio", "max_request_size", maxRequestSize, "max_concurrent", maxConcurrentReqs)

	for {
		select {
		case <-s.shutdownChan:
			slog.Info("MCP server shutting down (shutdown request)")

			return nil
		case <-ctx.Done():
			slog.Info("MCP server shutting down (context canceled)")

			return ctx.Err()
		default:
		}

		// Acquire semaphore (concurrency limit, blocking)
		select {
		case s.semaphore <- struct{}{}:
			// Got semaphore slot - check for immediate shutdown
		case <-s.shutdownChan:
			slog.Info("MCP server shutting down (shutdown request)")

			return nil
		case <-ctx.Done():
			slog.Info("MCP server shutting down (context canceled)")

			return ctx.Err()
		}

		// After acquiring semaphore, double-check we weren't shut down while waiting
		select {
		case <-s.shutdownChan:
			<-s.semaphore // Release before returning
			slog.Info("MCP server shutting down (shutdown request)")

			return nil
		default:
			// Continue processing
		}

		// Read a line (JSON-RPC request)
		// Note: This is a blocking read. For stdio-based MCP, we rely on the client
		// to be responsive. Timeouts are not supported for stdio reads.
		line, err := reader.ReadString('\n')
		if err != nil {
			<-s.semaphore // Release semaphore
			if errors.Is(err, io.EOF) {
				slog.Info("MCP server received EOF, shutting down")

				return nil
			}

			return fmt.Errorf("read stdin: %w", err)
		}

		if len(line) == 0 || line == "\n" {
			<-s.semaphore // Release semaphore

			continue
		}

		// Check request size
		if len(line) > maxRequestSize {
			<-s.semaphore // Release semaphore
			slog.Warn("Request too large, rejecting", "size", len(line), "max", maxRequestSize)
			// Write error response
			_ = s.writeResponse(writer, &Response{
				JSONRPC: "2.0",
				ID:      nil,
				Error: &Error{
					Code:    InvalidRequest,
					Message: fmt.Sprintf("Request too large: %d bytes (max %d)", len(line), maxRequestSize),
				},
			})

			continue
		}

		// Parse and handle request
		response := s.handleRequest(ctx, line)

		// Write response
		if response != nil {
			if err := s.writeResponse(writer, response); err != nil {
				<-s.semaphore // Release semaphore

				return err
			}
		}

		<-s.semaphore // Release semaphore
	}
}

// handleRequest processes a single JSON-RPC request.
func (s *Server) handleRequest(ctx context.Context, line string) *Response {
	var req Request
	if err := json.Unmarshal([]byte(line), &req); err != nil {
		return &Response{
			JSONRPC: "2.0",
			ID:      nil,
			Error: &Error{
				Code:    ParseError,
				Message: fmt.Sprintf("Parse error: %v", err),
			},
		}
	}

	// Validate JSON-RPC version
	if req.JSONRPC != "2.0" {
		return &Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &Error{
				Code:    InvalidRequest,
				Message: "Invalid JSON-RPC version (expected 2.0)",
			},
		}
	}

	// Route method
	switch req.Method {
	case MethodInitialize:
		return s.handleInitialize(ctx, &req)
	case MethodToolsList:
		return s.handleToolsList(ctx, &req)
	case MethodToolsCall:
		return s.handleToolsCall(ctx, &req)
	case MethodShutdown:
		return s.handleShutdown(ctx, &req)
	default:
		return &Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &Error{
				Code:    MethodNotFound,
				Message: "Method not found: " + req.Method,
			},
		}
	}
}

// handleInitialize handles the initialize request.
func (s *Server) handleInitialize(_ context.Context, req *Request) *Response {
	var params InitializeParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return &Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &Error{
				Code:    InvalidParams,
				Message: fmt.Sprintf("Invalid params: %v", err),
			},
		}
	}

	// Validate protocol version
	if params.ProtocolVersion != ProtocolVersion {
		slog.Warn("MCP client protocol version mismatch",
			"client_version", params.ProtocolVersion,
			"server_version", ProtocolVersion,
			"request_id", req.ID)

		return &Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &Error{
				Code: InvalidParams,
				Message: fmt.Sprintf("Protocol version mismatch: client sent %s, server expects %s",
					params.ProtocolVersion, ProtocolVersion),
			},
		}
	}

	// Validate required fields
	if params.ClientInfo.Name == "" {
		return &Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &Error{
				Code:    InvalidParams,
				Message: "Missing required field: clientInfo.name",
			},
		}
	}

	s.initialized.Store(true)

	slog.Info("MCP client initialized",
		"client", params.ClientInfo.Name,
		"version", params.ClientInfo.Version,
		"request_id", req.ID)

	result := InitializeResult{
		ProtocolVersion: ProtocolVersion,
		Capabilities: ServerCapabilities{
			Tools: &ToolsCapabilities{
				ListChanged: false,
			},
		},
		ServerInfo: s.serverInfo,
	}

	resultData, err := json.Marshal(result)
	if err != nil {
		slog.Error("Failed to marshal initialize result", "error", err, "request_id", req.ID)

		return &Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &Error{
				Code:    InternalError,
				Message: fmt.Sprintf("Failed to marshal response: %v", err),
			},
		}
	}

	return &Response{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  resultData,
	}
}

// handleToolsList handles the tools/list request.
func (s *Server) handleToolsList(_ context.Context, req *Request) *Response {
	if !s.initialized.Load() {
		return &Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &Error{
				Code:    InvalidRequest,
				Message: "Server not initialized",
			},
		}
	}

	tools := s.toolRegistry.ListTools()
	result := ToolsListResult{
		Tools: tools,
	}

	resultData, err := json.Marshal(result)
	if err != nil {
		slog.Error("Failed to marshal tools list result", "error", err, "request_id", req.ID)

		return &Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &Error{
				Code:    InternalError,
				Message: fmt.Sprintf("Failed to marshal response: %v", err),
			},
		}
	}

	return &Response{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  resultData,
	}
}

// handleToolsCall handles the tools/call request.
func (s *Server) handleToolsCall(ctx context.Context, req *Request) *Response {
	if !s.initialized.Load() {
		return &Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &Error{
				Code:    InvalidRequest,
				Message: "Server not initialized",
			},
		}
	}

	// Check rate limit
	if err := s.rateLimiter.Wait(ctx); err != nil {
		return &Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &Error{
				Code:    InternalError,
				Message: "Rate limit exceeded",
			},
		}
	}

	var params ToolCallParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return &Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &Error{
				Code:    InvalidParams,
				Message: fmt.Sprintf("Invalid params: %v", err),
			},
		}
	}

	result, err := s.toolRegistry.CallTool(ctx, params.Name, params.Arguments)
	if err != nil {
		slog.Error("Tool execution failed", "error", err, "tool", params.Name, "request_id", req.ID)

		return &Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &Error{
				Code:    InternalError,
				Message: fmt.Sprintf("Tool execution failed: %v", err),
			},
		}
	}

	resultData, err := json.Marshal(result)
	if err != nil {
		slog.Error("Failed to marshal tool call result", "error", err, "tool", params.Name, "request_id", req.ID)

		return &Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &Error{
				Code:    InternalError,
				Message: fmt.Sprintf("Failed to marshal response: %v", err),
			},
		}
	}

	return &Response{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  resultData,
	}
}

// handleShutdown handles the shutdown request.
func (s *Server) handleShutdown(_ context.Context, req *Request) *Response {
	s.initialized.Store(false)

	// Signal shutdown (only close once)
	s.shutdownOnce.Do(func() {
		close(s.shutdownChan)
	})

	slog.Info("MCP server shutting down (shutdown request)", "request_id", req.ID)

	resultData, err := json.Marshal(EmptyResult{})
	if err != nil {
		slog.Error("Failed to marshal shutdown result", "error", err, "request_id", req.ID)

		return &Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &Error{
				Code:    InternalError,
				Message: fmt.Sprintf("Failed to marshal response: %v", err),
			},
		}
	}

	return &Response{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  resultData,
	}
}

// isBrokenPipe checks if an error is a broken pipe error.
// This is needed because on some systems the error might be wrapped.
func isBrokenPipe(err error) bool {
	if err == nil {
		return false
	}
	// Check if the error string contains "broken pipe" for cross-platform support
	return errors.Is(err, syscall.EPIPE) ||
		errors.Is(err, io.ErrClosedPipe) ||
		strings.Contains(strings.ToLower(err.Error()), "broken pipe")
}

// writeResponse writes a JSON-RPC response to stdout, handling broken pipes.
func (s *Server) writeResponse(writer *bufio.Writer, response *Response) error {
	responseData, err := json.Marshal(response)
	if err != nil {
		slog.Error("Failed to marshal response", "error", err)

		return err
	}

	if _, err := writer.Write(responseData); err != nil {
		return s.checkWriteError(err)
	}
	if _, err := writer.WriteString("\n"); err != nil {
		return s.checkWriteError(err)
	}
	if err := writer.Flush(); err != nil {
		return s.checkWriteError(err)
	}

	return nil
}

// checkWriteError checks if a write error is a broken pipe (client disconnect).
// Returns an error to trigger server shutdown on client disconnect.
func (s *Server) checkWriteError(err error) error {
	if errors.Is(err, syscall.EPIPE) || isBrokenPipe(err) {
		slog.Info("MCP client disconnected (broken pipe)")

		// Return error to trigger server shutdown
		// This is important so the Serve() loop can exit cleanly
		return fmt.Errorf("client disconnected: %w", err)
	}

	return err
}
