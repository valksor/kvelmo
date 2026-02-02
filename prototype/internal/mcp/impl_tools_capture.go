package mcp

import (
	"context"
	"fmt"
	"io"
	"os"
	"sync"
)

// captureStdout redirects os.Stdout during fn() execution and returns any output
// written via fmt.Printf/Println (which bypasses Cobra's cmd.SetOut).
//
// This is critical for MCP tool execution: Cobra commands use fmt.Printf for output,
// but the MCP server captures output via root.SetOut(). Without this function,
// fmt.Printf output goes to the real stdout (corrupting JSON-RPC protocol), while
// the MCP response returns empty text.
//
// Safety: The MCP server's bufio.Writer (impl_server.go) holds the original os.Stdout
// reference from startup. Reassigning os.Stdout here only affects new callers (fmt.Printf),
// not the existing writer. The MCP server also processes requests sequentially, so no
// concurrent tool execution can race on os.Stdout.
//
// A background goroutine drains the pipe read-end to prevent deadlock from pipe buffer
// exhaustion (64KB on Linux).
//
// A mutex serializes access to os.Stdout. The context parameter prevents deadlock:
// if a previous tool execution is hanging and holding the mutex, this function will
// return ctx.Err() instead of blocking forever. Additionally, if fn() hangs, the
// context cancellation forcefully closes the pipe write-end, breaking any blocked
// write and allowing cleanup to proceed.
var stdoutCaptureMu sync.Mutex

func captureStdout(ctx context.Context, fn func()) (string, error) {
	// Try to acquire the lock with context awareness.
	// If a previous tool is hanging and holding the lock, we don't block forever.
	acquired := make(chan struct{})
	go func() {
		stdoutCaptureMu.Lock()
		close(acquired)
	}()

	select {
	case <-acquired:
		// Got the lock, proceed.
	case <-ctx.Done():
		// Context cancelled while waiting for the lock. The goroutine above will
		// eventually acquire and release when the hung tool finishes/times out.
		// We start a cleanup goroutine to unlock when acquired.
		go func() {
			<-acquired
			stdoutCaptureMu.Unlock()
		}()

		return "", ctx.Err()
	}

	defer stdoutCaptureMu.Unlock()

	origStdout := os.Stdout

	pr, pw, err := os.Pipe()
	if err != nil {
		// Pipe failure (fd exhaustion) — return error rather than running fn()
		// without capture, which would corrupt the JSON-RPC protocol stream.
		return "", fmt.Errorf("capture stdout pipe: %w", err)
	}

	os.Stdout = pw

	// Background reader prevents pipe buffer deadlock: if fn() writes >64KB,
	// the write blocks until someone reads. This goroutine continuously drains.
	doneCh := make(chan string, 1)

	go func() {
		data, _ := io.ReadAll(pr)
		doneCh <- string(data)
	}()

	// Use sync.Once to ensure the pipe write-end is closed exactly once,
	// even if both the context-cancellation goroutine and the normal path
	// attempt to close it.
	var closePW sync.Once
	closePipe := func() { _ = pw.Close() }

	// Force-close the pipe write-end if the context is cancelled while fn() is
	// running. This breaks any blocked stdout write inside fn(), allowing the
	// background reader to receive EOF and the function to unwind.
	go func() {
		<-ctx.Done()
		closePW.Do(closePipe)
	}()

	// Ensure stdout restoration on panic. Normal path handles cleanup directly.
	panicked := true

	defer func() {
		if panicked {
			os.Stdout = origStdout
			closePW.Do(closePipe)
			_ = pr.Close()
		}
	}()

	fn()

	panicked = false

	// Normal path: restore stdout and collect captured output
	os.Stdout = origStdout
	closePW.Do(closePipe) // Signal EOF to background reader
	result := <-doneCh    // Wait for reader to finish
	_ = pr.Close()        // Clean up read end

	return result, nil
}
