// Package workflow provides utilities for generating delta specifications.
package workflow

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/valksor/go-mehrhof/internal/provider"
)

// Generator creates delta specifications for updated work units.
type Generator struct {
	workDir string
}

// NewGenerator creates a new delta specification generator for the given work directory.
func NewGenerator(workDir string) *Generator {
	return &Generator{
		workDir: workDir,
	}
}

// GenerateDeltaSpecification creates a new specification file for updates to a work unit.
// Returns the path to the generated specification file.
func (g *Generator) GenerateDeltaSpecification(ctx context.Context, changes provider.ChangeSet, oldContent, newContent string) (string, error) {
	if !changes.HasChanges {
		return "", errors.New("no changes detected")
	}

	// Find the next specification filename
	specificationPath, err := g.nextSpecificationFilename()
	if err != nil {
		return "", fmt.Errorf("find next specification filename: %w", err)
	}

	// Build the specification content
	content := g.buildUpdatePrompt(changes, oldContent, newContent)

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(specificationPath), 0o755); err != nil {
		return "", fmt.Errorf("create directory: %w", err)
	}

	// Write specification file
	if err := os.WriteFile(specificationPath, []byte(content), 0o644); err != nil {
		return "", fmt.Errorf("write specification file: %w", err)
	}

	return specificationPath, nil
}

// nextSpecificationFilename finds the next available specification filename.
// Returns paths like "specification-2.md", "specification-3.md", etc.
// Uses atomic file creation to avoid TOCTOU race conditions.
func (g *Generator) nextSpecificationFilename() (string, error) {
	specificationDir := filepath.Join(g.workDir, "specifications")

	// Ensure directory exists
	if err := os.MkdirAll(specificationDir, 0o755); err != nil {
		return "", fmt.Errorf("create directory: %w", err)
	}

	// Try incrementing numbers until we find an available file
	// Use O_EXCL to atomically create the file and avoid race conditions
	const maxSpecificationFiles = 10000
	for num := 1; num <= maxSpecificationFiles; num++ {
		path := filepath.Join(specificationDir, fmt.Sprintf("specification-%d.md", num))
		fd, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
		if err == nil {
			// File created successfully - close and return path
			if closeErr := fd.Close(); closeErr != nil {
				return "", fmt.Errorf("close file: %w", closeErr)
			}

			return path, nil
		}
		if !os.IsExist(err) {
			return "", fmt.Errorf("create specification file: %w", err)
		}
		// File exists, try next number
	}

	return "", fmt.Errorf("too many specification files (max: %d)", maxSpecificationFiles)
}

// buildUpdatePrompt creates the content for a delta specification file.
func (g *Generator) buildUpdatePrompt(changes provider.ChangeSet, oldContent, newContent string) string {
	var builder strings.Builder

	// Header
	builder.WriteString("# Update Specification\n\n")
	builder.WriteString("The following changes have been detected in the work unit:\n\n")
	builder.WriteString("## Summary of Changes\n\n")
	builder.WriteString(changes.Summary() + "\n\n")

	// Detailed changes
	if changes.DescriptionChanged {
		builder.WriteString("## Description Changes\n\n")
		builder.WriteString("The task description has been updated.\n\n")
	}

	if changes.StatusChanged {
		builder.WriteString("## Status Changes\n\n")
		builder.WriteString(fmt.Sprintf("Status changed from `%s` to `%s`\n\n", changes.OldStatus, changes.NewStatus))
	}

	if len(changes.NewComments) > 0 {
		builder.WriteString(fmt.Sprintf("## New Comments (%d)\n\n", len(changes.NewComments)))
		for _, comment := range changes.NewComments {
			author := comment.Author.Name
			if author == "" {
				author = comment.Author.ID
			}
			builder.WriteString("### " + author + "\n\n")
			builder.WriteString(comment.Body + "\n\n")
		}
	}

	if len(changes.UpdatedComments) > 0 {
		builder.WriteString(fmt.Sprintf("## Updated Comments (%d)\n\n", len(changes.UpdatedComments)))
		for _, comment := range changes.UpdatedComments {
			author := comment.Author.Name
			if author == "" {
				author = comment.Author.ID
			}
			builder.WriteString("### " + author + "\n\n")
			builder.WriteString(comment.Body + "\n\n")
		}
	}

	if len(changes.NewAttachments) > 0 {
		builder.WriteString(fmt.Sprintf("## New Attachments (%d)\n\n", len(changes.NewAttachments)))
		for _, att := range changes.NewAttachments {
			builder.WriteString("- " + att.Name + "\n")
		}
		builder.WriteString("\n")
	}

	if len(changes.RemovedAttachments) > 0 {
		builder.WriteString(fmt.Sprintf("## Removed Attachments (%d)\n\n", len(changes.RemovedAttachments)))
		for _, att := range changes.RemovedAttachments {
			builder.WriteString("- " + att.Name + "\n")
		}
		builder.WriteString("\n")
	}

	// Content comparison
	builder.WriteString("## Content Comparison\n\n")
	if oldContent != "" {
		builder.WriteString("### Previous Content\n\n")
		builder.WriteString("```\n")
		builder.WriteString(oldContent)
		builder.WriteString("\n```\n\n")
	}

	if newContent != "" {
		builder.WriteString("### Updated Content\n\n")
		builder.WriteString("```\n")
		builder.WriteString(newContent)
		builder.WriteString("\n```\n\n")
	}

	// Instructions for agent
	builder.WriteString("## Instructions\n\n")
	builder.WriteString("Based on the changes above, please:\n\n")
	builder.WriteString("1. Review the updated requirements and content\n")
	builder.WriteString("2. Identify what needs to be implemented or modified\n")
	builder.WriteString("3. Create a detailed implementation plan\n")
	builder.WriteString("4. Update the existing specifications as needed\n\n")

	return builder.String()
}

// BackupSourceFile backs up the original source file before updating.
// Uses timestamp to preserve all backup versions.
func (g *Generator) BackupSourceFile(sourcePath string) error {
	// Use timestamp for unique backup names
	timestamp := time.Now().Format("20060102-150405")
	backupPath := sourcePath + "." + timestamp + ".backup"

	// Read source
	content, err := os.ReadFile(sourcePath)
	if err != nil {
		return err
	}

	// Write backup
	return os.WriteFile(backupPath, content, 0o644)
}

// WriteDiffFile writes a human-readable diff summary to a file.
func (g *Generator) WriteDiffFile(changes provider.ChangeSet) error {
	diffPath := filepath.Join(g.workDir, "source", "changes.txt")
	content := changes.FormatDiff()

	return os.WriteFile(diffPath, []byte(content), 0o644)
}
