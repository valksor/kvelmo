package security

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// SARIFReport represents a SARIF log file.
type SARIFReport struct {
	Version string     `json:"version"`
	Schema  string     `json:"$schema"`
	Runs    []SARIFRun `json:"runs"`
}

// SARIFRun represents a single run in the SARIF report.
type SARIFRun struct {
	Tool    SARIFTool     `json:"tool"`
	Results []SARIFResult `json:"results"`
}

// SARIFTool represents the tool that generated the results.
type SARIFTool struct {
	Driver SARIFDriver `json:"driver"`
}

// SARIFDriver represents the driver (scanner) information.
type SARIFDriver struct {
	Name           string      `json:"name"`
	Version        string      `json:"version,omitempty"`
	InformationURI string      `json:"informationURI,omitempty"`
	Rules          []SARIFRule `json:"rules,omitempty"`
}

// SARIFRule represents a rule in the tool driver.
type SARIFRule struct {
	ID               string                 `json:"id"`
	Name             string                 `json:"name,omitempty"`
	ShortDescription SARIFMessage           `json:"shortDescription"`
	FullDescription  SARIFMessage           `json:"fullDescription,omitempty"`
	HelpURI          string                 `json:"helpUri,omitempty"`
	Properties       map[string]interface{} `json:"properties,omitempty"`
}

// SARIFMessage represents a message in SARIF.
type SARIFMessage struct {
	Text string `json:"text"`
}

// SARIFResult represents a single result in the SARIF report.
type SARIFResult struct {
	RuleID    string          `json:"ruleId"`
	Message   SARIFMessage    `json:"message"`
	Level     string          `json:"level"`
	Locations []SARIFLocation `json:"locations"`
	Fixes     []SARIFFix      `json:"fixes,omitempty"`
}

// SARIFLocation represents a physical location in the SARIF report.
type SARIFLocation struct {
	PhysicalLocation SARIFPhysicalLocation `json:"physicalLocation"`
}

// SARIFPhysicalLocation represents a physical location.
type SARIFPhysicalLocation struct {
	ArtifactLocation SARIFArtifactLocation `json:"artifactLocation"`
	Region           SARIFRegion           `json:"region"`
}

// SARIFArtifactLocation represents an artifact location.
type SARIFArtifactLocation struct {
	URI string `json:"uri"`
}

// SARIFRegion represents a region in the artifact.
type SARIFRegion struct {
	StartLine   int           `json:"startLine"`
	StartColumn int           `json:"startColumn,omitempty"`
	EndLine     int           `json:"endLine,omitempty"`
	EndColumn   int           `json:"endColumn,omitempty"`
	Snippet     *SARIFSnippet `json:"snippet,omitempty"`
}

// SARIFSnippet represents a code snippet.
type SARIFSnippet struct {
	Text string `json:"text"`
}

// SARIFFix represents a fix suggestion.
type SARIFFix struct {
	Description     SARIFMessage          `json:"description"`
	ArtifactChanges []SARIFArtifactChange `json:"artifactChanges"`
}

// SARIFArtifactChange represents a change to an artifact.
type SARIFArtifactChange struct {
	ArtifactLocation SARIFArtifactLocation `json:"artifactLocation"`
	Replacements     []SARIFReplacement    `json:"replacements"`
}

// SARIFReplacement represents a replacement in the artifact.
type SARIFReplacement struct {
	DeletedRegion   SARIFRegion            `json:"deletedRegion"`
	InsertedContent *SARIFInsertionContext `json:"insertedContent,omitempty"`
}

// SARIFInsertionContext represents content to be inserted.
type SARIFInsertionContext struct {
	Text string `json:"text"`
}

// GenerateSARIF generates a SARIF report from scan results.
func GenerateSARIF(results []*ScanResult) (*SARIFReport, error) {
	report := &SARIFReport{
		Version: "2.1.0",
		Schema:  "https://json.schemastore.org/sarif-2.1.0.json",
		Runs:    make([]SARIFRun, 0, len(results)),
	}

	for _, result := range results {
		if result.Error != nil {
			// Skip scanners that errored
			continue
		}

		run := SARIFRun{
			Tool: SARIFTool{
				Driver: SARIFDriver{
					Name:    result.Scanner,
					Version: "1.0.0",
				},
			},
			Results: make([]SARIFResult, 0, len(result.Findings)),
		}

		// Build unique rules from findings
		ruleMap := make(map[string]*SARIFRule)
		for _, finding := range result.Findings {
			ruleID := finding.ID
			if _, exists := ruleMap[ruleID]; !exists {
				ruleMap[ruleID] = &SARIFRule{
					ID:   ruleID,
					Name: finding.Title,
					ShortDescription: SARIFMessage{
						Text: finding.Description,
					},
				}

				// Add CVE info to properties
				if finding.CVE != "" {
					if ruleMap[ruleID].Properties == nil {
						ruleMap[ruleID].Properties = make(map[string]interface{})
					}
					ruleMap[ruleID].Properties["cve"] = finding.CVE
				}
			}
		}

		// Convert rules to slice
		for _, rule := range ruleMap {
			run.Tool.Driver.Rules = append(run.Tool.Driver.Rules, *rule)
		}

		// Convert findings to results
		for _, finding := range result.Findings {
			result := convertFindingToSARIF(finding)
			run.Results = append(run.Results, result)
		}

		report.Runs = append(report.Runs, run)
	}

	return report, nil
}

// convertFindingToSARIF converts a Finding to a SARIFResult.
func convertFindingToSARIF(finding Finding) SARIFResult {
	result := SARIFResult{
		RuleID: finding.ID,
		Message: SARIFMessage{
			Text: finding.Description,
		},
		Level: severityToLevel(finding.Severity),
		Locations: []SARIFLocation{
			{
				PhysicalLocation: SARIFPhysicalLocation{
					ArtifactLocation: SARIFArtifactLocation{
						URI: finding.Location.File,
					},
					Region: SARIFRegion{
						StartLine:   finding.Location.Line,
						StartColumn: finding.Location.Column,
						EndLine:     finding.Location.EndLine,
						EndColumn:   finding.Location.EndColumn,
					},
				},
			},
		},
	}

	// Add code snippet if available
	if finding.Code != nil && finding.Code.Before != "" {
		result.Locations[0].PhysicalLocation.Region.Snippet = &SARIFSnippet{
			Text: finding.Code.Before,
		}
	}

	// Add fix suggestion if available
	if finding.Fix != nil {
		result.Fixes = []SARIFFix{
			{
				Description: SARIFMessage{
					Text: finding.Fix.Description,
				},
			},
		}

		// If we have a patch, add it as a replacement
		if finding.Fix.Patch != "" && finding.Code != nil {
			result.Fixes[0].ArtifactChanges = []SARIFArtifactChange{
				{
					ArtifactLocation: SARIFArtifactLocation{
						URI: finding.Location.File,
					},
					Replacements: []SARIFReplacement{
						{
							DeletedRegion: SARIFRegion{
								StartLine:   finding.Location.Line,
								StartColumn: finding.Location.Column,
							},
							InsertedContent: &SARIFInsertionContext{
								Text: finding.Fix.Patch,
							},
						},
					},
				},
			}
		}
	}

	return result
}

// severityToLevel converts a Severity to a SARIF level.
func severityToLevel(severity Severity) string {
	switch severity {
	case SeverityCritical:
		return "error"
	case SeverityHigh:
		return "error"
	case SeverityMedium:
		return "warning"
	case SeverityLow:
		return "note"
	case SeverityInfo:
		return "note"
	default:
		return "note"
	}
}

// WriteSARIF writes a SARIF report to a file with path validation and atomic write.
func WriteSARIF(report *SARIFReport, filePath string) error {
	// Validate output path for directory traversal attempts
	abs, err := filepath.Abs(filePath)
	if err != nil {
		return fmt.Errorf("invalid SARIF output path: %w", err)
	}

	// Check for directory traversal attempts
	parts := strings.Split(abs, string(filepath.Separator))
	for _, part := range parts {
		if part == ".." {
			return fmt.Errorf("directory traversal detected in SARIF output path: %s", filePath)
		}
	}

	// Ensure directory exists
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create SARIF output directory: %w", err)
	}

	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal SARIF report: %w", err)
	}

	// Use atomic write with temp file
	tmpPath := filePath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o600); err != nil {
		return fmt.Errorf("failed to write SARIF report: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tmpPath, filePath); err != nil {
		// Clean up temp file on error (ignore error since we're already in an error state)
		_ = os.Remove(tmpPath)

		return fmt.Errorf("failed to finalize SARIF report: %w", err)
	}

	return nil
}

// GenerateAndWriteSARIF generates a SARIF report and writes it to a file.
func GenerateAndWriteSARIF(results []*ScanResult, filepath string) error {
	report, err := GenerateSARIF(results)
	if err != nil {
		return fmt.Errorf("failed to generate SARIF report: %w", err)
	}

	if err := WriteSARIF(report, filepath); err != nil {
		return err
	}

	return nil
}

// MarshalJSON marshals the SARIF report to JSON.
func (r *SARIFReport) MarshalJSON() ([]byte, error) {
	return json.Marshal(r)
}

// MarshalJSONResults marshals scan results to JSON.
func MarshalJSONResults(results []*ScanResult) ([]byte, error) {
	return json.MarshalIndent(results, "", "  ")
}
