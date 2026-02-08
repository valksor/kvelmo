package commands

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/valksor/go-mehrhof/internal/conductor"
	"github.com/valksor/go-mehrhof/internal/template"
)

func init() {
	Register(Command{
		Info: CommandInfo{
			Name:        "template",
			Description: "Template management commands",
			Category:    "tools",
			Subcommands: []string{"list", "get", "apply"},
		},
		Handler: handleTemplate,
	})
}

func handleTemplate(_ context.Context, _ *conductor.Conductor, inv Invocation) (*Result, error) {
	subcommand := "list"
	subArgs := inv.Args
	if len(subArgs) > 0 {
		subcommand = subArgs[0]
		subArgs = subArgs[1:]
	}

	switch subcommand {
	case "list":
		return handleTemplateList()
	case "get":
		return handleTemplateGet(inv, subArgs)
	case "apply":
		return handleTemplateApply(inv)
	default:
		return nil, fmt.Errorf("unknown template subcommand: %s (available: list, get, apply)", subcommand)
	}
}

func handleTemplateList() (*Result, error) {
	names := template.BuiltInTemplates()

	type templateItem struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}

	items := make([]templateItem, 0, len(names))
	for _, name := range names {
		tpl, err := template.LoadBuiltIn(name)
		if err != nil {
			continue
		}
		items = append(items, templateItem{
			Name:        name,
			Description: tpl.GetDescription(),
		})
	}

	return NewResult(fmt.Sprintf("%d template(s) available", len(items))).WithData(map[string]any{
		"templates": items,
		"count":     len(items),
	}), nil
}

func handleTemplateGet(inv Invocation, subArgs []string) (*Result, error) {
	name := GetString(inv.Options, "name")
	if name == "" && len(subArgs) > 0 {
		name = strings.TrimSpace(subArgs[0])
	}
	if name == "" {
		return nil, errors.New("template name is required")
	}

	tpl, err := template.LoadBuiltIn(name)
	if err != nil {
		return nil, fmt.Errorf("template not found: %w", err)
	}

	agentSteps := make(map[string]any, len(tpl.AgentSteps))
	for step, cfg := range tpl.AgentSteps {
		agentSteps[step] = cfg
	}

	workflowCfg := make(map[string]any, len(tpl.Workflow))
	for k, v := range tpl.Workflow {
		workflowCfg[k] = v
	}

	return NewResult("Template: " + tpl.Name).WithData(map[string]any{
		"name":        tpl.Name,
		"description": tpl.Description,
		"frontmatter": tpl.Frontmatter,
		"agent":       tpl.Agent,
		"agent_steps": agentSteps,
		"git":         tpl.Git,
		"workflow":    workflowCfg,
	}), nil
}

func handleTemplateApply(inv Invocation) (*Result, error) {
	name := GetString(inv.Options, "name")
	if name == "" {
		return nil, errors.New("template name is required (options.name)")
	}

	filePath := GetString(inv.Options, "path")
	if filePath == "" {
		return nil, errors.New("file path is required (options.path)")
	}

	tpl, err := template.LoadBuiltIn(name)
	if err != nil {
		return nil, fmt.Errorf("template not found: %w", err)
	}

	// Read existing content or use default
	var content string
	data, err := os.ReadFile(filePath)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to read file: %w", err)
		}
		content = "# Task Title\n\nDescribe your task here.\n"
	} else {
		content = string(data)
	}

	// Apply template
	newContent := tpl.ApplyToContent(content)

	// Write back
	if err := os.WriteFile(filePath, []byte(newContent), 0o644); err != nil {
		return nil, fmt.Errorf("failed to write file: %w", err)
	}

	return NewResult(fmt.Sprintf("Applied template '%s' to %s", tpl.Name, filePath)).WithData(map[string]any{
		"success":     true,
		"frontmatter": tpl.Frontmatter,
		"message":     fmt.Sprintf("applied template '%s' to %s", tpl.Name, filePath),
	}), nil
}
