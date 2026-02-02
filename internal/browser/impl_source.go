package browser

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/go-rod/rod/lib/proto"
)

// GetPageSource returns the full HTML source of the current page.
func (c *controller) GetPageSource(ctx context.Context, tabID string) (string, error) {
	c.mu.RLock()
	page, err := c.getPage(tabID)
	c.mu.RUnlock()
	if err != nil {
		return "", errNotFound("tab " + tabID)
	}

	html, err := page.Context(ctx).HTML()
	if err != nil {
		return "", fmt.Errorf("browser: get page source: %w", err)
	}

	return html, nil
}

// GetScriptSources returns all JavaScript sources loaded in the page.
// Uses the Debugger domain to discover parsed scripts and retrieve their source.
func (c *controller) GetScriptSources(ctx context.Context, tabID string) ([]ScriptSource, error) {
	c.mu.RLock()
	page, err := c.getPage(tabID)
	c.mu.RUnlock()
	if err != nil {
		return nil, errNotFound("tab " + tabID)
	}

	ctxPage := page.Context(ctx)

	// Enable the Debugger domain to discover scripts
	_, _ = proto.DebuggerEnable{}.Call(ctxPage)
	defer func() {
		_ = proto.DebuggerDisable{}.Call(ctxPage)
	}()

	// Collect ScriptParsed events with a short timeout
	// Scripts already loaded fire immediately when Debugger is enabled
	var scripts []ScriptSource

	scriptCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	// Use page.EachEvent to collect ScriptParsed events
	done := make(chan struct{})
	go func() {
		defer close(done)
		ctxPage.Context(scriptCtx).EachEvent(
			func(e *proto.DebuggerScriptParsed) {
				// Skip internal/extension scripts
				if e.URL == "" {
					return
				}

				source, err := proto.DebuggerGetScriptSource{
					ScriptID: e.ScriptID,
				}.Call(ctxPage)
				if err != nil {
					slog.Debug("failed to get script source", "scriptID", e.ScriptID, "url", e.URL, "error", err)

					return
				}

				scripts = append(scripts, ScriptSource{
					ScriptID: string(e.ScriptID),
					URL:      e.URL,
					Source:   source.ScriptSource,
					Length:   len(source.ScriptSource),
				})
			},
		)()
		<-scriptCtx.Done()
	}()

	<-done

	return scripts, nil
}
