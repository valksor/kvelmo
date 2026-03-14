package socket

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"time"

	"github.com/valksor/kvelmo/pkg/agent/recorder"
	"github.com/valksor/kvelmo/pkg/paths"
)

type recordingsListParams struct {
	Job   string `json:"job,omitempty"`
	Since string `json:"since,omitempty"`
}

type recordingsViewParams struct {
	File string `json:"file"`
}

type recordingViewResult struct {
	Header  *recorder.Header  `json:"header"`
	Records []recorder.Record `json:"records"`
}

func recordingsDir() string {
	return filepath.Join(paths.Paths().BaseDir(), "recordings")
}

func (g *GlobalSocket) handleRecordingsList(_ context.Context, req *Request) (*Response, error) {
	var params recordingsListParams
	if req.Params != nil {
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return NewErrorResponse(req.ID, ErrCodeInvalidParams, "invalid params"), nil //nolint:nilerr // JSON-RPC error response
		}
	}

	dir := recordingsDir()
	infos, err := recorder.ListRecordings(dir)
	if err != nil {
		return NewErrorResponse(req.ID, ErrCodeInternal, err.Error()), nil
	}

	// Filter by job if specified
	if params.Job != "" {
		var filtered []recorder.RecordingInfo
		for _, info := range infos {
			if info.JobID == params.Job {
				filtered = append(filtered, info)
			}
		}
		infos = filtered
	}

	// Filter by time if specified
	if params.Since != "" {
		since, err := parseDurationString(params.Since)
		if err != nil {
			return NewErrorResponse(req.ID, ErrCodeInvalidParams, "invalid since duration: "+err.Error()), nil //nolint:nilerr // JSON-RPC error response
		}
		cutoff := time.Now().Add(-since)
		var filtered []recorder.RecordingInfo
		for _, info := range infos {
			t, err := time.Parse(time.RFC3339, info.StartedAt)
			if err != nil {
				continue
			}
			if t.After(cutoff) {
				filtered = append(filtered, info)
			}
		}
		infos = filtered
	}

	if infos == nil {
		infos = []recorder.RecordingInfo{}
	}

	return NewResultResponse(req.ID, map[string]any{
		"recordings": infos,
	})
}

func (g *GlobalSocket) handleRecordingsView(_ context.Context, req *Request) (*Response, error) {
	var params recordingsViewParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return NewErrorResponse(req.ID, ErrCodeInvalidParams, "invalid params"), nil
	}

	if params.File == "" {
		return NewErrorResponse(req.ID, ErrCodeInvalidParams, "file is required"), nil
	}

	path := params.File
	if !filepath.IsAbs(path) {
		path = filepath.Join(recordingsDir(), path)
	}

	reader, err := recorder.OpenReader(path)
	if err != nil {
		return NewErrorResponse(req.ID, ErrCodeInternal, "open recording: "+err.Error()), nil
	}
	defer func() { _ = reader.Close() }()

	var records []recorder.Record
	for {
		rec, err := reader.Next()
		if err != nil {
			return NewErrorResponse(req.ID, ErrCodeInternal, "read record: "+err.Error()), nil
		}
		if rec == nil {
			break
		}
		records = append(records, *rec)
	}

	if records == nil {
		records = []recorder.Record{}
	}

	result := recordingViewResult{
		Header:  reader.Header(),
		Records: records,
	}

	return NewResultResponse(req.ID, result)
}

// parseDurationString parses duration strings like "24h", "7d", "30d".
func parseDurationString(s string) (time.Duration, error) {
	if len(s) > 1 && s[len(s)-1] == 'd' {
		var days int
		if _, err := fmt.Sscanf(s, "%dd", &days); err != nil {
			return 0, fmt.Errorf("invalid day duration: %s", s)
		}

		return time.Duration(days) * 24 * time.Hour, nil
	}

	return time.ParseDuration(s)
}
