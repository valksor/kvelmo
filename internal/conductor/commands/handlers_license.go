package commands

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/valksor/go-toolkit/licensing"

	"github.com/valksor/go-mehrhof/internal/conductor"
	"github.com/valksor/go-mehrhof/internal/server/static"
)

func init() {
	Register(Command{
		Info: CommandInfo{
			Name:        "license",
			Description: "Show project license",
			Category:    "info",
		},
		Handler: handleLicense,
	})

	Register(Command{
		Info: CommandInfo{
			Name:        "license-info",
			Description: "Show dependency license information",
			Category:    "info",
		},
		Handler: handleLicenseInfo,
	})
}

func handleLicense(_ context.Context, _ *conductor.Conductor, _ Invocation) (*Result, error) {
	text := licensing.GetProjectLicense()

	return NewResult("License loaded").WithData(map[string]any{
		"license": text,
		"type":    "BSD-3-Clause",
	}), nil
}

func handleLicenseInfo(_ context.Context, _ *conductor.Conductor, _ Invocation) (*Result, error) {
	data, err := static.FS.ReadFile("licenses.json")
	if err != nil {
		return nil, fmt.Errorf("failed to read embedded licenses: %w", err)
	}

	var response struct {
		Licenses []struct {
			Path    string `json:"path"`
			License string `json:"license"`
			Unknown bool   `json:"unknown"`
		} `json:"licenses"`
		Count int `json:"count"`
	}
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse licenses: %w", err)
	}

	return NewResult(fmt.Sprintf("%d dependency license(s)", response.Count)).WithData(map[string]any{
		"licenses": response.Licenses,
		"count":    response.Count,
	}), nil
}
