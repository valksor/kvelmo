package commands

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/valksor/go-mehrhof/internal/conductor"
)

func init() {
	Register(Command{
		Info: CommandInfo{
			Name:         "agent-logs-history",
			Description:  "Get recent agent log history for a task",
			Category:     "info",
			RequiresTask: false,
		},
		Handler: handleAgentLogsHistory,
	})
}

// handleAgentLogsHistory returns transcript-based agent log history.
func handleAgentLogsHistory(_ context.Context, cond *conductor.Conductor, inv Invocation) (*Result, error) {
	taskID := GetString(inv.Options, "task_id")

	// Fall back to active task if no task ID provided.
	if taskID == "" && cond != nil {
		if task := cond.GetActiveTask(); task != nil {
			taskID = task.ID
		}
	}

	if taskID == "" {
		return NewResult("No task").WithData(map[string]any{
			"logs": []map[string]any{},
		}), nil
	}

	if cond == nil {
		return NewResult("No conductor").WithData(map[string]any{
			"logs":  []map[string]any{},
			"error": "conductor not initialized",
		}), nil
	}

	ws := cond.GetWorkspace()
	if ws == nil {
		return nil, errors.New("workspace not initialized")
	}

	transcripts, err := ws.ListTranscripts(taskID)
	if err != nil {
		return nil, errors.New("failed to load transcripts")
	}

	sort.Strings(transcripts)

	var logs []map[string]any
	lineIndex := 0

	for _, transcriptFile := range transcripts {
		content, loadErr := ws.LoadTranscript(taskID, transcriptFile)
		if loadErr != nil {
			continue
		}

		kind, startedAt := parseTranscriptMeta(transcriptFile)
		lines := strings.Split(content, "\n")

		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}

			logs = append(logs, map[string]any{
				"index":      lineIndex,
				"kind":       kind,
				"started_at": startedAt,
				"file":       transcriptFile,
				"type":       "output",
				"message":    line,
			})
			lineIndex++
		}
	}

	return NewResult("Logs loaded").WithData(map[string]any{
		"logs":    logs,
		"task_id": taskID,
		"count":   len(logs),
	}), nil
}

// parseTranscriptMeta extracts kind and started_at from a transcript filename.
// Format: "2006-01-02T15-04-05-kind.log".
func parseTranscriptMeta(filename string) (string, string) {
	kind := "unknown"
	startedAt := ""

	trimmed := strings.TrimSuffix(filename, ".log")
	lastDash := strings.LastIndex(trimmed, "-")
	if lastDash <= 0 || lastDash >= len(trimmed)-1 {
		return kind, startedAt
	}

	timestampPart := trimmed[:lastDash]
	kind = trimmed[lastDash+1:]

	if parsed, err := time.Parse("2006-01-02T15-04-05", timestampPart); err == nil {
		startedAt = parsed.Format(time.RFC3339)
	}

	return kind, startedAt
}
