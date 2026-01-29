//go:build ignore

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/valksor/go-toolkit/licensing"
)

func main() {
	ctx := context.Background()

	// Get dependency licenses from current directory (where go.mod is)
	libs, err := licensing.GetDependencyLicenses(ctx, ".")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting dependency licenses: %v\n", err)
		os.Exit(1)
	}

	// Create output data structure matching the API response
	type licenseInfo struct {
		Path    string `json:"path"`
		License string `json:"license"`
		Unknown bool   `json:"unknown"`
	}

	licenses := make([]licenseInfo, len(libs))
	for i, lib := range libs {
		licenses[i] = licenseInfo{
			Path:    lib.Path,
			License: lib.License,
			Unknown: lib.Unknown,
		}
	}

	output := struct {
		Licenses []licenseInfo `json:"licenses"`
		Count    int           `json:"count"`
	}{
		Licenses: licenses,
		Count:    len(licenses),
	}

	// Encode as JSON
	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
		os.Exit(1)
	}

	// Write to output file
	outputPath := "internal/server/static/licenses.json"
	if err := os.WriteFile(outputPath, data, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing %s: %v\n", outputPath, err)
		os.Exit(1)
	}

	fmt.Printf("Generated %s with %d dependencies\n", outputPath, len(licenses))
}
