package commands

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/valksor/go-mehrhof/internal/agent"
	"github.com/valksor/go-mehrhof/internal/conductor"
	"github.com/valksor/go-mehrhof/internal/library"
	"github.com/valksor/go-mehrhof/internal/memory"
	"github.com/valksor/go-toolkit/display"
)

// handleChat sends a chat message to the agent.
func (s *InteractiveSession) handleChat(ctx context.Context, message string) error {
	if message == "" {
		return errors.New("message cannot be empty")
	}

	// Check if we have an active agent
	activeAgent := s.cond.GetActiveAgent()
	if activeAgent == nil {
		return errors.New("no agent available")
	}

	s.printf(true, "\n%s %s\n", display.Bold("You:"), message)
	s.printf(true, "%s\n", display.Bold("Agent:"))

	// Build prompt with context
	prompt := s.buildChatPrompt(message)

	// Run agent with streaming
	response, err := activeAgent.RunWithCallback(ctx, prompt, func(event agent.Event) error {
		return s.handleAgentEvent(event)
	})
	if err != nil {
		return fmt.Errorf("agent error: %w", err)
	}

	fmt.Println() // New line after response

	// Handle if the agent asked a question
	if response != nil && response.Question != nil {
		return s.handleAgentQuestion(response.Question)
	}

	return nil
}

// handleFind performs AI-powered code search.
func (s *InteractiveSession) handleFind(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: find <query>")
	}

	query := strings.Join(args, " ")
	s.printf(true, "Searching for: %s\n", display.Cyan(query))

	findOpts := conductor.FindOptions{
		Query:     query,
		Path:      "",
		Pattern:   "",
		Context:   3,
		Workspace: s.cond.GetWorkspace(),
	}

	resultChan, err := s.cond.Find(ctx, findOpts)
	if err != nil {
		return fmt.Errorf("find: %w", err)
	}

	var results []conductor.FindResult
	for result := range resultChan {
		if result.File == "__error__" {
			return fmt.Errorf("search error: %s", result.Snippet)
		}
		results = append(results, result)
	}

	if len(results) == 0 {
		s.printf(true, "No matches found.\n")

		return nil
	}

	s.printf(true, "\n%s\n", display.Bold(fmt.Sprintf("Found %d match(es):", len(results))))
	for i, r := range results {
		s.printf(true, "%d. %s:%d\n", i+1, r.File, r.Line)
		if r.Snippet != "" {
			s.printf(true, "   %s\n", display.Muted(r.Snippet))
		}
		if r.Reason != "" {
			s.printf(true, "   %s %s\n", display.Cyan("->"), r.Reason)
		}
	}
	s.printf(true, "\n")

	return nil
}

// handleSimplify simplifies code based on the current workflow The handleSimplify function optimizes code according to the current workflow status.
//
//nolint:unparam // args are kept for consistent signature with other handlers
func (s *InteractiveSession) handleSimplify(ctx context.Context, args []string) error {
	if s.cond.GetActiveTask() == nil {
		return errors.New("no active task")
	}

	s.printf(true, "Simplifying...\n")

	if err := s.cond.Simplify(ctx, "", true); err != nil {
		return fmt.Errorf("simplify: %w", err)
	}

	s.printf(true, "%s Simplification complete\n", display.SuccessMsg("OK"))

	return nil
}

// handleLabel manages task labels.
func (s *InteractiveSession) handleLabel(ctx context.Context, args []string) error {
	if len(args) == 0 {
		s.listLabels(ctx)

		return nil
	}

	subcommand := args[0]
	subArgs := args[1:]

	taskID := ""
	if s.cond.GetActiveTask() != nil {
		taskID = s.cond.GetActiveTask().ID
	}

	switch subcommand {
	case "add":
		return s.handleLabelAdd(ctx, taskID, subArgs)
	case "remove", "rm":
		return s.handleLabelRemove(ctx, taskID, subArgs)
	case "set":
		return s.handleLabelSet(ctx, taskID, subArgs)
	case "clear":
		return s.handleLabelSet(ctx, taskID, []string{})
	case "list", "ls":
		s.listLabels(ctx)

		return nil
	default:
		return s.handleLabelAdd(ctx, taskID, args)
	}
}

// handleLabelAdd adds labels to the active task.
//
//nolint:unparam // ctx is kept for consistent signature with other handlers
func (s *InteractiveSession) handleLabelAdd(ctx context.Context, taskID string, labels []string) error {
	if taskID == "" {
		return errors.New("no active task")
	}
	if len(labels) == 0 {
		return errors.New("usage: label add <label...>")
	}
	ws := s.cond.GetWorkspace()
	for _, label := range labels {
		if err := ws.AddLabel(taskID, label); err != nil {
			return fmt.Errorf("add label %q: %w", label, err)
		}
	}
	s.printf(true, "%s Added %d label(s)\n", display.SuccessMsg("OK"), len(labels))

	return nil
}

// handleLabelRemove removes labels from the active task.
//
//nolint:unparam // ctx is kept for consistent signature with other handlers
func (s *InteractiveSession) handleLabelRemove(ctx context.Context, taskID string, labels []string) error {
	if taskID == "" {
		return errors.New("no active task")
	}
	if len(labels) == 0 {
		return errors.New("usage: label remove <label...>")
	}
	ws := s.cond.GetWorkspace()
	for _, label := range labels {
		if err := ws.RemoveLabel(taskID, label); err != nil {
			return fmt.Errorf("remove label %q: %w", label, err)
		}
	}
	s.printf(true, "%s Removed %d label(s)\n", display.SuccessMsg("OK"), len(labels))

	return nil
}

// handleLabelSet sets labels on the active task.
//
//nolint:unparam // ctx is kept for consistent signature with other handlers
func (s *InteractiveSession) handleLabelSet(ctx context.Context, taskID string, labels []string) error {
	if taskID == "" {
		return errors.New("no active task")
	}
	ws := s.cond.GetWorkspace()
	if err := ws.SetLabels(taskID, labels); err != nil {
		return fmt.Errorf("set labels: %w", err)
	}
	if len(labels) == 0 {
		s.printf(true, "%s Cleared all labels\n", display.SuccessMsg("OK"))
	} else {
		s.printf(true, "%s Set %d label(s)\n", display.SuccessMsg("OK"), len(labels))
	}

	return nil
}

// listLabels lists labels for the active task.
func (s *InteractiveSession) listLabels(context.Context) {
	task := s.cond.GetActiveTask()
	if task == nil {
		s.printf(true, "No active task\n")

		return
	}
	ws := s.cond.GetWorkspace()
	labels, err := ws.GetLabels(task.ID)
	if err != nil {
		s.printf(false, "%s %v\n", display.ErrorMsg("Error:"), err)

		return
	}
	s.printf(true, "\n%s\n", display.Bold("Labels:"))
	if len(labels) == 0 {
		s.printf(true, "  (no labels)\n")

		return
	}
	for _, label := range labels {
		s.printf(true, "  - %s\n", display.Cyan(label))
	}
	s.printf(true, "\n")
}

// handleMemory searches semantic memory.
func (s *InteractiveSession) handleMemory(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: memory <query>")
	}

	mem := s.cond.GetMemory()
	if mem == nil {
		return errors.New("memory system is not enabled")
	}

	query := strings.Join(args, " ")

	s.printf(true, "Searching memory for: %s\n", display.Cyan(query))

	results, err := mem.Search(ctx, query, memory.SearchOptions{
		Limit:    5,
		MinScore: 0.65,
	})
	if err != nil {
		return fmt.Errorf("memory search: %w", err)
	}

	if len(results) == 0 {
		s.printf(true, "No similar tasks found.\n")

		return nil
	}

	s.printf(true, "\n%s\n", display.Bold(fmt.Sprintf("Found %d similar task(s):", len(results))))
	for i, result := range results {
		doc := result.Document
		s.printf(true, "%d. Task %s (%.0f%% similar)\n", i+1, doc.TaskID, result.Score*100)
		s.printf(true, "   Type: %s\n", doc.Type)
		preview := doc.Content
		if len(preview) > 200 {
			preview = preview[:200] + "..."
		}
		s.printf(true, "   %s\n\n", display.Muted(preview))
	}

	return nil
}

// handleLibrary manages the documentation library.
func (s *InteractiveSession) handleLibrary(ctx context.Context, args []string) error {
	lib := s.cond.GetLibrary()
	if lib == nil {
		// Check if there was an initialization error
		if initErr := s.cond.GetLibraryError(); initErr != nil {
			return initErr
		}

		return errors.New("library system is not enabled. Enable in .mehrhof/config.yaml under 'library:'")
	}

	// Default to list if no subcommand
	subcommand := "list"
	if len(args) > 0 {
		subcommand = args[0]
		args = args[1:]
	}

	switch subcommand {
	case "list", "ls":
		return s.handleLibraryList(ctx, lib)
	case "show":
		if len(args) == 0 {
			return errors.New("usage: library show <name>")
		}

		return s.handleLibraryShow(ctx, lib, args[0])
	case "search":
		if len(args) == 0 {
			return errors.New("usage: library search <query>")
		}

		return s.handleLibrarySearch(ctx, lib, strings.Join(args, " "))
	default:
		// Treat unknown subcommand as collection name for show
		return s.handleLibraryShow(ctx, lib, subcommand)
	}
}

// handleLibraryList lists all library collections.
func (s *InteractiveSession) handleLibraryList(ctx context.Context, lib *library.Manager) error {
	collections, err := lib.List(ctx, &library.ListOptions{})
	if err != nil {
		return fmt.Errorf("list collections: %w", err)
	}

	if len(collections) == 0 {
		s.printf(true, "No library collections. Use 'mehr library pull <source>' to add documentation.\n")

		return nil
	}

	s.printf(true, "\n%s\n", display.Bold(fmt.Sprintf("%d Collection(s):", len(collections))))
	for _, c := range collections {
		mode := string(c.IncludeMode)
		location := c.Location
		s.printf(true, "  %s [%s, %s]\n", display.Cyan(c.Name), mode, location)
		s.printf(true, "    Source: %s\n", display.Muted(c.Source))
		s.printf(true, "    Pages: %d  Size: %s\n", c.PageCount, formatSize(c.TotalSize))
	}
	s.printf(true, "\n")

	return nil
}

// handleLibraryShow shows details of a collection.
func (s *InteractiveSession) handleLibraryShow(ctx context.Context, lib *library.Manager, name string) error {
	coll, err := lib.Show(ctx, name)
	if err != nil {
		return fmt.Errorf("show collection: %w", err)
	}

	s.printf(true, "\n%s\n", display.Bold("Collection: "+coll.Name))
	s.printf(true, "  ID:          %s\n", coll.ID)
	s.printf(true, "  Source:      %s\n", coll.Source)
	s.printf(true, "  Type:        %s\n", coll.SourceType)
	s.printf(true, "  Mode:        %s\n", coll.IncludeMode)
	s.printf(true, "  Location:    %s\n", coll.Location)
	s.printf(true, "  Pages:       %d\n", coll.PageCount)
	s.printf(true, "  Total Size:  %s\n", formatSize(coll.TotalSize))

	if len(coll.Paths) > 0 {
		s.printf(true, "  Paths:       %s\n", strings.Join(coll.Paths, ", "))
	}
	if len(coll.Tags) > 0 {
		s.printf(true, "  Tags:        %s\n", strings.Join(coll.Tags, ", "))
	}

	// List pages
	pages, err := lib.ListPages(ctx, coll.ID)
	if err == nil && len(pages) > 0 {
		s.printf(true, "\n%s\n", display.Bold("Pages:"))
		limit := 10
		for i, page := range pages {
			if i >= limit {
				s.printf(true, "  ... and %d more\n", len(pages)-limit)

				break
			}
			s.printf(true, "  - %s\n", page)
		}
	}
	s.printf(true, "\n")

	return nil
}

// handleLibrarySearch searches library documentation.
func (s *InteractiveSession) handleLibrarySearch(ctx context.Context, lib *library.Manager, query string) error {
	s.printf(true, "Searching library for: %s\n", display.Cyan(query))

	// Use the library context search
	docCtx, err := lib.GetDocsForQuery(ctx, query, 10000)
	if err != nil {
		return fmt.Errorf("search library: %w", err)
	}

	if docCtx == nil || len(docCtx.Pages) == 0 {
		s.printf(true, "No matching documentation found.\n")

		return nil
	}

	// Extract unique collection names from pages
	collectionSet := make(map[string]bool)
	for _, p := range docCtx.Pages {
		collectionSet[p.CollectionName] = true
	}
	var collNames []string
	for name := range collectionSet {
		collNames = append(collNames, name)
	}

	s.printf(true, "\n%s\n", display.Bold(fmt.Sprintf("Found %d page(s) from %d collection(s):", len(docCtx.Pages), len(collNames))))
	for _, name := range collNames {
		s.printf(true, "  - %s\n", display.Cyan(name))
	}

	// Show preview of first page
	if len(docCtx.Pages) > 0 {
		page := docCtx.Pages[0]
		s.printf(true, "\n%s\n", display.Bold("First match: "+page.Title))
		preview := page.Content
		if len(preview) > 500 {
			preview = preview[:500] + "\n... (truncated)"
		}
		s.printf(true, "%s\n", display.Muted(preview))
	}
	s.printf(true, "\n")

	return nil
}

// formatSize formats bytes as human-readable string.
func formatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// capitalizeFirst capitalizes the first letter of a string.
func capitalizeFirst(s string) string {
	if s == "" {
		return s
	}

	return strings.ToUpper(s[:1]) + s[1:]
}
