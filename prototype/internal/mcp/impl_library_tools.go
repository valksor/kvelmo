package mcp

import (
	"context"
	"fmt"
	"strings"

	"github.com/valksor/go-mehrhof/internal/library"
)

// RegisterLibraryTools registers library documentation tools with the MCP registry.
func RegisterLibraryTools(registry *ToolRegistry) {
	// library_list - List documentation collections
	registry.RegisterDirectTool(
		"library_list",
		"List all documentation collections. Returns collection names, sources, page counts, and include modes.",
		map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
			"required":   []string{},
		},
		func(ctx context.Context, args map[string]interface{}) (*ToolCallResult, error) {
			ws, err := openValidWorkspace(ctx)
			if err != nil {
				return errorResult(err), nil
			}

			lib, err := library.NewManagerFromWorkspace(ctx, ws)
			if err != nil {
				return errorResult(fmt.Errorf("initialize library: %w", err)), nil
			}

			collections, err := lib.List(ctx, nil)
			if err != nil {
				return errorResult(fmt.Errorf("list collections: %w", err)), nil
			}

			if len(collections) == 0 {
				return textResult("No documentation collections found. Use library_pull to add documentation from URLs, files, or git repositories."), nil
			}

			var sb strings.Builder
			sb.WriteString(fmt.Sprintf("Found %d collection(s):\n\n", len(collections)))
			for _, c := range collections {
				sb.WriteString(fmt.Sprintf("- %s\n", c.Name))
				sb.WriteString(fmt.Sprintf("  Source: %s\n", c.Source))
				sb.WriteString(fmt.Sprintf("  Type: %s\n", c.SourceType))
				sb.WriteString(fmt.Sprintf("  Mode: %s\n", c.IncludeMode))
				sb.WriteString(fmt.Sprintf("  Pages: %d\n", c.PageCount))
				if len(c.Tags) > 0 {
					sb.WriteString(fmt.Sprintf("  Tags: %s\n", strings.Join(c.Tags, ", ")))
				}
				sb.WriteString("\n")
			}

			return textResult(sb.String()), nil
		},
	)

	// library_show - Show collection details
	registry.RegisterDirectTool(
		"library_show",
		"Show details for a specific documentation collection including metadata and page list.",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"name": map[string]interface{}{
					"type":        "string",
					"description": "Collection name to show details for",
				},
			},
			"required": []string{"name"},
		},
		func(ctx context.Context, args map[string]interface{}) (*ToolCallResult, error) {
			ws, err := openValidWorkspace(ctx)
			if err != nil {
				return errorResult(err), nil
			}

			lib, err := library.NewManagerFromWorkspace(ctx, ws)
			if err != nil {
				return errorResult(fmt.Errorf("initialize library: %w", err)), nil
			}

			name, ok := args["name"].(string)
			if !ok || name == "" {
				return textResult("Error: name is required"), nil
			}

			collection, err := lib.Show(ctx, name)
			if err != nil {
				return errorResult(fmt.Errorf("show collection: %w", err)), nil
			}

			pages, _ := lib.ListPages(ctx, collection.ID)

			var sb strings.Builder
			sb.WriteString(fmt.Sprintf("Collection: %s\n\n", collection.Name))
			sb.WriteString(fmt.Sprintf("Source: %s\n", collection.Source))
			sb.WriteString(fmt.Sprintf("Type: %s\n", collection.SourceType))
			sb.WriteString(fmt.Sprintf("Mode: %s\n", collection.IncludeMode))
			sb.WriteString(fmt.Sprintf("Pages: %d\n", collection.PageCount))

			if len(pages) > 0 {
				sb.WriteString("\nPages (showing first 50):\n")
				maxPages := 50
				if len(pages) < maxPages {
					maxPages = len(pages)
				}
				for _, p := range pages[:maxPages] {
					sb.WriteString(fmt.Sprintf("  - %s\n", p))
				}
				if len(pages) > 50 {
					sb.WriteString(fmt.Sprintf("  ... and %d more\n", len(pages)-50))
				}
			}

			return textResult(sb.String()), nil
		},
	)

	// library_get_docs - Get relevant documentation for file paths
	registry.RegisterDirectTool(
		"library_get_docs",
		"Get relevant library documentation for given file paths. Auto-includes collections matching path patterns. Returns formatted documentation for AI prompt injection.",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"paths": map[string]interface{}{
					"type":        "string",
					"description": "Comma-separated file paths to find relevant docs for",
				},
			},
			"required": []string{"paths"},
		},
		func(ctx context.Context, args map[string]interface{}) (*ToolCallResult, error) {
			ws, err := openValidWorkspace(ctx)
			if err != nil {
				return errorResult(err), nil
			}

			lib, err := library.NewManagerFromWorkspace(ctx, ws)
			if err != nil {
				return errorResult(fmt.Errorf("initialize library: %w", err)), nil
			}

			pathsStr, ok := args["paths"].(string)
			if !ok || pathsStr == "" {
				return textResult("Error: paths is required"), nil
			}

			paths := strings.Split(pathsStr, ",")
			for i := range paths {
				paths[i] = strings.TrimSpace(paths[i])
			}

			docs, err := lib.GetDocsForPaths(ctx, paths, 8000)
			if err != nil {
				return errorResult(fmt.Errorf("get docs: %w", err)), nil
			}

			formatted := library.FormatDocsForPrompt(docs)
			if formatted == "" {
				return textResult("No relevant documentation found for the given paths. Use library_list to see available collections."), nil
			}

			return textResult(formatted), nil
		},
	)
}
