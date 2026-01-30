package conductor

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// SourceTaskOptions configures creating a single queue task from a source.
type SourceTaskOptions struct {
	QueueID      string
	Title        string
	Instructions string
	Notes        []string
	Provider     string
	Priority     int
	Labels       []string
}

// SourceTaskResult holds the result of creating a task from a source.
type SourceTaskResult struct {
	QueueID string
	TaskID  string
	Draft   *draftTask
}

type draftTask struct {
	Title       string
	Description string
	Labels      []string
	Priority    int
}

// CreateQueueTaskFromSource creates a single queue task from a source file or directory.
func (c *Conductor) CreateQueueTaskFromSource(ctx context.Context, source string, opts SourceTaskOptions) (*SourceTaskResult, error) {
	if strings.TrimSpace(source) == "" {
		return nil, errors.New("source is required")
	}

	normalizedSource, err := normalizeSourceRef(source)
	if err != nil {
		return nil, err
	}

	notes := append([]string{}, opts.Notes...)
	if opts.Provider != "" && !hasProviderNote(notes) {
		notes = append(notes, "Target provider: "+opts.Provider)
	}

	var prompt string
	if strings.HasPrefix(normalizedSource, "research:") {
		dirPath := strings.TrimPrefix(normalizedSource, "research:")
		manifest, err := c.readResearchSource(dirPath)
		if err != nil {
			return nil, fmt.Errorf("read research source: %w", err)
		}
		prompt = buildSingleTaskResearchPrompt(opts.Title, manifest, opts.Instructions, notes)
	} else {
		sourceContent, err := c.readProjectSource(ctx, normalizedSource)
		if err != nil {
			return nil, fmt.Errorf("read source: %w", err)
		}
		prompt = buildSingleTaskPrompt(opts.Title, sourceContent, opts.Instructions, notes)
	}

	ag, err := c.GetAgentForStep(ctx, "planning")
	if err != nil {
		return nil, fmt.Errorf("get agent: %w", err)
	}

	resp, err := ag.Run(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("execute planning: %w", err)
	}

	response := resp.Summary
	if response == "" && len(resp.Messages) > 0 {
		response = strings.Join(resp.Messages, "\n")
	}

	draft, err := parseDraftTask(response)
	if err != nil {
		return nil, fmt.Errorf("parse draft task: %w", err)
	}

	applyDraftOverrides(draft, normalizedSource, opts)

	result, err := c.CreateQuickTask(ctx, QuickTaskOptions{
		Description: draft.Description,
		Title:       draft.Title,
		Priority:    draft.Priority,
		Labels:      draft.Labels,
		QueueID:     opts.QueueID,
	})
	if err != nil {
		return nil, fmt.Errorf("create quick task: %w", err)
	}

	ws := c.workspace
	for _, note := range notes {
		if strings.TrimSpace(note) == "" {
			continue
		}
		if err := ws.AppendQueueNote(result.QueueID, result.TaskID, note); err != nil {
			return nil, fmt.Errorf("append note: %w", err)
		}
	}

	return &SourceTaskResult{
		QueueID: result.QueueID,
		TaskID:  result.TaskID,
		Draft:   draft,
	}, nil
}

func normalizeSourceRef(source string) (string, error) {
	trimmed := strings.TrimSpace(source)
	if trimmed == "" {
		return "", errors.New("source is required")
	}

	if isKnownSourceRef(trimmed) {
		return trimmed, nil
	}

	info, err := os.Stat(trimmed)
	if err != nil {
		if os.IsNotExist(err) {
			return trimmed, nil
		}

		return "", fmt.Errorf("stat source: %w", err)
	}

	if info.IsDir() {
		return "research:" + trimmed, nil
	}

	return "file:" + trimmed, nil
}

func isKnownSourceRef(source string) bool {
	prefixes := []string{
		"dir:", "file:", "research:",
		"github:", "gitlab:", "jira:", "wrike:", "linear:", "asana:",
		"notion:", "trello:", "youtrack:", "bitbucket:", "clickup:", "azuredevops:",
	}
	for _, prefix := range prefixes {
		if strings.HasPrefix(source, prefix) {
			return true
		}
	}

	return false
}

func hasProviderNote(notes []string) bool {
	for _, note := range notes {
		if strings.Contains(strings.ToLower(note), "target provider") {
			return true
		}
	}

	return false
}

func applyDraftOverrides(draft *draftTask, source string, opts SourceTaskOptions) {
	if opts.Title != "" {
		draft.Title = opts.Title
	} else if draft.Title == "" {
		draft.Title = fallbackTitleFromSource(source)
	}

	if opts.Priority > 0 {
		draft.Priority = opts.Priority
	}
	if draft.Priority == 0 {
		draft.Priority = 2
	}

	if len(opts.Labels) > 0 {
		draft.Labels = mergeLabels(draft.Labels, opts.Labels)
	}
}

func fallbackTitleFromSource(source string) string {
	raw := strings.TrimPrefix(source, "file:")
	raw = strings.TrimPrefix(raw, "dir:")
	raw = strings.TrimPrefix(raw, "research:")

	base := filepath.Base(raw)
	if base == "." || base == string(filepath.Separator) || base == "" {
		return "Untitled Task"
	}

	return strings.TrimSuffix(base, filepath.Ext(base))
}

func buildSingleTaskPrompt(title, sourceContent, instructions string, notes []string) string {
	currentTime := time.Now().Format("2006-01-02 15:04")

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Current timestamp: %s\n\n", currentTime))
	if title != "" {
		sb.WriteString(fmt.Sprintf("Project/Task Title: %s\n\n", title))
	}
	sb.WriteString("## Source Content\n\n")
	sb.WriteString(sourceContent)
	sb.WriteString("\n\n")

	if len(notes) > 0 {
		sb.WriteString("## Notes\n")
		for i, note := range notes {
			sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, note))
		}
		sb.WriteString("\n")
	}

	if instructions != "" {
		sb.WriteString("## Instructions\n")
		sb.WriteString(instructions)
		sb.WriteString("\n\n")
	}

	sb.WriteString(`## Task Draft Instructions

Create a single task that best captures what needs to be done based on the source content.

## Output Format

Respond with a YAML-like format between --- markers:

---
title: Task Title
priority: 1
labels: label1, label2
description: |
  Task description here.
  Use multiple lines if needed.
---

Use priority 1 (high), 2 (normal), or 3 (low). Do not include any text outside the --- markers.
`)

	return sb.String()
}

func buildSingleTaskResearchPrompt(title string, manifest *ResearchManifest, instructions string, notes []string) string {
	currentTime := time.Now().Format("2006-01-02 15:04")

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Current timestamp: %s\n\n", currentTime))
	if title != "" {
		sb.WriteString(fmt.Sprintf("Project/Task Title: %s\n\n", title))
	}
	sb.WriteString("## Research Base Path\n")
	sb.WriteString(manifest.BasePath)
	sb.WriteString("\n\n")
	sb.WriteString(fmt.Sprintf("This directory contains %d files for you to research.\n\n", manifest.FileCount))

	if len(manifest.EntryPoints) > 0 {
		sb.WriteString("## Detected Entry Points\n\n")
		for i, ep := range manifest.EntryPoints {
			sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, ep))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("## Directory Structure\n\n")
	sb.WriteString("```\n")
	for _, entry := range manifest.Structure {
		indent := strings.Repeat("  ", strings.Count(entry.Path, string(filepath.Separator)))
		if entry.Type == "dir" {
			sb.WriteString(fmt.Sprintf("%s%s/\n", indent, entry.Name))
		} else {
			sb.WriteString(fmt.Sprintf("%s%s (%s, %d bytes)\n", indent, entry.Name, entry.Category, entry.Size))
		}
	}
	sb.WriteString("```\n\n")

	if len(notes) > 0 {
		sb.WriteString("## Notes\n")
		for i, note := range notes {
			sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, note))
		}
		sb.WriteString("\n")
	}

	if instructions != "" {
		sb.WriteString("## Instructions\n")
		sb.WriteString(instructions)
		sb.WriteString("\n\n")
	}

	sb.WriteString(`## Research Instructions

IMPORTANT: You have access to Read, Glob, and Grep tools to explore these files.

1. Start with entry points when present.
2. Explore selectively using Read/Grep/Glob.
3. Focus on the most actionable single task to execute.

## Output Format

Respond with a YAML-like format between --- markers:

---
title: Task Title
priority: 1
labels: label1, label2
description: |
  Task description here.
  Use multiple lines if needed.
---

Use priority 1 (high), 2 (normal), or 3 (low). Do not include any text outside the --- markers.
`)

	return sb.String()
}

func parseDraftTask(response string) (*draftTask, error) {
	content := response
	if strings.Contains(content, "---") {
		parts := strings.Split(content, "---")
		if len(parts) >= 2 {
			content = strings.TrimSpace(parts[1])
		}
	}

	result := &draftTask{}
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		switch {
		case strings.HasPrefix(line, "title:"):
			result.Title = strings.TrimSpace(strings.TrimPrefix(line, "title:"))
		case strings.HasPrefix(line, "labels:"):
			labelsStr := strings.TrimSpace(strings.TrimPrefix(line, "labels:"))
			if labelsStr != "" {
				for _, label := range strings.Split(labelsStr, ",") {
					label = strings.TrimSpace(label)
					if label != "" {
						result.Labels = append(result.Labels, label)
					}
				}
			}
		case strings.HasPrefix(line, "priority:"):
			result.Priority = parseDraftPriority(strings.TrimSpace(strings.TrimPrefix(line, "priority:")))
		case strings.HasPrefix(line, "description:"):
			desc := strings.TrimPrefix(line, "description:")
			if strings.HasPrefix(strings.TrimSpace(desc), "|") {
				result.Description = ""
			} else {
				result.Description = strings.TrimSpace(desc)
			}
		}
	}

	if strings.Contains(response, "description: |") {
		descStart := strings.Index(response, "description: |")
		if descStart >= 0 {
			afterDesc := response[descStart+14:]
			nextField := strings.Index(afterDesc, "\n---")
			if nextField > 0 {
				afterDesc = afterDesc[:nextField]
			} else if nextField = strings.Index(afterDesc, "\nlabels:"); nextField > 0 {
				afterDesc = afterDesc[:nextField]
			} else if nextField = strings.Index(afterDesc, "\npriority:"); nextField > 0 {
				afterDesc = afterDesc[:nextField]
			}
			result.Description = normalizeMultiline(afterDesc)
		}
	}

	if result.Title == "" && result.Description == "" {
		return nil, errors.New("missing task title and description")
	}

	return result, nil
}

func normalizeMultiline(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}

	lines := strings.Split(value, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimLeft(line, " \t")
	}

	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func parseDraftPriority(value string) int {
	lower := strings.ToLower(strings.TrimSpace(value))
	switch lower {
	case "high":
		return 1
	case "normal":
		return 2
	case "low":
		return 3
	}

	if lower == "" {
		return 0
	}

	if num, err := strconv.Atoi(lower); err == nil {
		if num < 1 {
			return 1
		}
		if num > 3 {
			return 3
		}

		return num
	}

	return 0
}
