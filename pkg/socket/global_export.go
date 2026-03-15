package socket

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/valksor/kvelmo/pkg/activitylog"
	"github.com/valksor/kvelmo/pkg/metrics"
)

// handleExport returns task and metrics data for external analysis.
func (g *GlobalSocket) handleExport(_ context.Context, req *Request) (*Response, error) {
	var params struct {
		Format  string `json:"format"`
		Since   string `json:"since"`
		Include string `json:"include"`
	}
	if req.Params != nil {
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return NewErrorResponse(req.ID, ErrCodeInvalidParams, "invalid params"), nil //nolint:nilerr // JSON-RPC error response
		}
	}

	result := map[string]any{
		"format": params.Format,
	}

	// Include metrics snapshot
	result["metrics"] = metrics.Global().Snapshot()

	// Include active tasks from all worktrees
	g.mu.RLock()
	tasks := make([]map[string]any, 0, len(g.worktrees))
	for _, wt := range g.worktrees {
		tasks = append(tasks, map[string]any{
			"id":    wt.ID,
			"path":  wt.Path,
			"state": wt.State,
		})
	}
	g.mu.RUnlock()

	result["tasks"] = tasks

	// Include activity log entries if available
	if g.server.activityLogger != nil {
		if adapter, ok := g.server.activityLogger.(*activityLogAdapter); ok {
			since := 7 * 24 * time.Hour // default 7 days
			if params.Since != "" {
				if d, err := parseSinceDuration(params.Since); err == nil {
					since = d
				}
			}
			entries, err := adapter.log.Query(activitylog.QueryOptions{
				Since: since,
			})
			if err == nil {
				result["activity"] = entries
			}
		}
	}

	return NewResultResponse(req.ID, result)
}

// parseSinceDuration parses a duration string supporting both Go duration syntax
// and shorthand "Nd" for days.
func parseSinceDuration(s string) (time.Duration, error) {
	if d, err := time.ParseDuration(s); err == nil {
		return d, nil
	}
	if strings.HasSuffix(s, "d") {
		numStr := strings.TrimSuffix(s, "d")
		var days int
		if _, err := fmt.Sscanf(numStr, "%d", &days); err == nil {
			return time.Duration(days) * 24 * time.Hour, nil
		}
	}

	return 0, fmt.Errorf("invalid duration: %s", s)
}
