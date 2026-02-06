package server

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/valksor/go-mehrhof/internal/conductor"
	"github.com/valksor/go-mehrhof/internal/library"
	"github.com/valksor/go-mehrhof/internal/memory"
	"github.com/valksor/go-mehrhof/internal/storage"
)

// executeInteractiveExploreCommand handles exploration interactive commands.
// Commands: find, memory, library, links.
func (s *Server) executeInteractiveExploreCommand(ctx context.Context, command string, args []string) (string, error) {
	cond := s.config.Conductor

	switch command {
	case "find":
		if len(args) == 0 {
			return "", errors.New("find requires a query")
		}
		findOpts := conductor.FindOptions{
			Query:     strings.Join(args, " "),
			Path:      "",
			Pattern:   "",
			Context:   3,
			Workspace: cond.GetWorkspace(),
		}
		resultChan, findErr := cond.Find(ctx, findOpts)
		if findErr != nil {
			return "", findErr
		}
		var results []conductor.FindResult
		for findResult := range resultChan {
			if findResult.File != "__error__" {
				results = append(results, findResult)
			}
		}
		if len(results) == 0 {
			return "No matches found", nil
		}
		var lines []string
		for _, r := range results {
			lines = append(lines, fmt.Sprintf("• %s:%d - %s", r.File, r.Line, r.Reason))
		}

		return fmt.Sprintf("Found %d match(es):\n%s", len(results), strings.Join(lines, "\n")), nil

	case "memory":
		mem := cond.GetMemory()
		if mem == nil {
			return "", errors.New("memory system is not enabled")
		}
		if len(args) == 0 {
			return "", errors.New("memory requires a subcommand: search <query>, index <task-id>, stats")
		}
		subcommand := args[0]
		subArgs := args[1:]
		switch subcommand {
		case "search":
			if len(subArgs) == 0 {
				return "", errors.New("memory search requires a query")
			}
			query := strings.Join(subArgs, " ")
			memResults, memErr := mem.Search(ctx, query, memory.SearchOptions{
				Limit:    5,
				MinScore: 0.65,
			})
			if memErr != nil {
				return "", memErr
			}
			if len(memResults) == 0 {
				return "No similar tasks found", nil
			}
			var lines []string
			for _, r := range memResults {
				taskID := ""
				if r.Document != nil {
					taskID = r.Document.TaskID
				}
				lines = append(lines, fmt.Sprintf("• %s (%.0f%% similar)", taskID, r.Score*100))
			}

			return fmt.Sprintf("Found %d similar task(s):\n%s", len(memResults), strings.Join(lines, "\n")), nil
		case "index":
			if len(subArgs) == 0 {
				return "", errors.New("memory index requires a task ID")
			}
			ws := cond.GetWorkspace()
			if ws == nil {
				return "", errors.New("workspace not initialized")
			}
			taskID := subArgs[0]
			// Verify task exists
			if _, loadErr := ws.LoadWork(taskID); loadErr != nil {
				return "", fmt.Errorf("task not found: %w", loadErr)
			}
			indexer := memory.NewIndexer(mem, ws, nil)
			if indexErr := indexer.IndexTask(ctx, taskID); indexErr != nil {
				return "", fmt.Errorf("failed to index task: %w", indexErr)
			}

			return fmt.Sprintf("Task %s indexed successfully", taskID), nil
		case "stats":
			ws := cond.GetWorkspace()
			if ws == nil {
				return "", errors.New("workspace not initialized")
			}
			indexer := memory.NewIndexer(mem, ws, nil)
			stats, statsErr := indexer.GetStats(ctx)
			if statsErr != nil {
				return "", fmt.Errorf("failed to get stats: %w", statsErr)
			}
			var lines []string
			lines = append(lines, fmt.Sprintf("Total documents: %d", stats.TotalDocuments))
			if len(stats.ByType) > 0 {
				lines = append(lines, "By type:")
				for docType, count := range stats.ByType {
					lines = append(lines, fmt.Sprintf("  • %s: %d", docType, count))
				}
			}

			return strings.Join(lines, "\n"), nil
		default:
			// Backwards compatibility: treat unknown subcommand as search query
			query := strings.Join(args, " ")
			memResults, memErr := mem.Search(ctx, query, memory.SearchOptions{
				Limit:    5,
				MinScore: 0.65,
			})
			if memErr != nil {
				return "", memErr
			}
			if len(memResults) == 0 {
				return "No similar tasks found", nil
			}
			var lines []string
			for _, r := range memResults {
				taskID := ""
				if r.Document != nil {
					taskID = r.Document.TaskID
				}
				lines = append(lines, fmt.Sprintf("• %s (%.0f%% similar)", taskID, r.Score*100))
			}

			return fmt.Sprintf("Found %d similar task(s):\n%s", len(memResults), strings.Join(lines, "\n")), nil
		}

	case "library":
		lib := cond.GetLibrary()
		if lib == nil {
			// Check if there was an initialization error
			if initErr := cond.GetLibraryError(); initErr != nil {
				return "", initErr
			}

			return "", errors.New("library system is not enabled. Use the Library panel or enable in .mehrhof/config.yaml under 'library:'")
		}
		// Default to list if no subcommand
		subcommand := "list"
		if len(args) > 0 {
			subcommand = args[0]
			args = args[1:]
		}
		switch subcommand {
		case "list", "ls":
			collections, listErr := lib.List(ctx, &library.ListOptions{})
			if listErr != nil {
				return "", listErr
			}
			if len(collections) == 0 {
				return "No library collections. Use the Library panel or run 'mehr library pull <source>' to add documentation.", nil
			}
			var lines []string
			for _, c := range collections {
				lines = append(lines, fmt.Sprintf("• %s [%s, %s] - %d pages",
					c.Name, c.IncludeMode, c.Location, c.PageCount))
			}

			return fmt.Sprintf("%d Collection(s):\n%s", len(collections), strings.Join(lines, "\n")), nil
		case "show":
			if len(args) == 0 {
				return "", errors.New("usage: library show <name>")
			}
			coll, showErr := lib.Show(ctx, args[0])
			if showErr != nil {
				return "", showErr
			}

			return fmt.Sprintf("Collection: %s\nSource: %s\nType: %s\nMode: %s\nPages: %d",
				coll.Name, coll.Source, coll.SourceType, coll.IncludeMode, coll.PageCount), nil
		case "search":
			if len(args) == 0 {
				return "", errors.New("usage: library search <query>")
			}
			query := strings.Join(args, " ")
			docCtx, searchErr := lib.GetDocsForQuery(ctx, query, 10000)
			if searchErr != nil {
				return "", searchErr
			}
			if docCtx == nil || len(docCtx.Pages) == 0 {
				return "No matching documentation found", nil
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

			return fmt.Sprintf("Found %d page(s) from %d collection(s): %s",
				len(docCtx.Pages), len(collNames), strings.Join(collNames, ", ")), nil
		case "pull":
			if len(args) == 0 {
				return "", errors.New("usage: library pull <source> [--name <name>] [--shared]")
			}
			source := args[0]
			opts := &library.PullOptions{}
			// Parse simple flags
			for i := 1; i < len(args); i++ {
				if args[i] == "--name" && i+1 < len(args) {
					opts.Name = args[i+1]
					i++
				} else if args[i] == "--shared" {
					opts.Shared = true
				}
			}
			pullResult, pullErr := lib.Pull(ctx, source, opts)
			if pullErr != nil {
				return "", pullErr
			}

			return fmt.Sprintf("Pulled collection: %s (%d pages)", pullResult.Collection.Name, pullResult.Collection.PageCount), nil
		case "remove", "rm":
			if len(args) == 0 {
				return "", errors.New("usage: library remove <name>")
			}
			if removeErr := lib.Remove(ctx, args[0], false); removeErr != nil {
				return "", removeErr
			}

			return fmt.Sprintf("Collection '%s' removed", args[0]), nil
		case "stats":
			collections, listErr := lib.List(ctx, &library.ListOptions{})
			if listErr != nil {
				return "", listErr
			}
			var totalPages int
			var sharedCount, projectCount int
			for _, c := range collections {
				totalPages += c.PageCount
				if c.Location == "shared" {
					sharedCount++
				} else {
					projectCount++
				}
			}

			return fmt.Sprintf("Library Stats:\n• Collections: %d (%d shared, %d project)\n• Total pages: %d",
				len(collections), sharedCount, projectCount, totalPages), nil
		default:
			// Treat as collection name for show
			coll, showErr := lib.Show(ctx, subcommand)
			if showErr != nil {
				return "", showErr
			}

			return fmt.Sprintf("Collection: %s\nSource: %s\nType: %s\nMode: %s\nPages: %d",
				coll.Name, coll.Source, coll.SourceType, coll.IncludeMode, coll.PageCount), nil
		}

	case "links":
		ws := cond.GetWorkspace()
		if ws == nil {
			return "", errors.New("workspace not initialized")
		}
		linkMgr := storage.GetLinkManager(ctx, ws)
		if linkMgr == nil {
			return "", errors.New("links system is not available")
		}
		subcommand := "list"
		if len(args) > 0 {
			subcommand = args[0]
			args = args[1:]
		}
		switch subcommand {
		case "list", "ls":
			linkIndex := linkMgr.GetIndex()
			var totalLinks int
			for _, forwardLinks := range linkIndex.Forward {
				totalLinks += len(forwardLinks)
			}

			return fmt.Sprintf("Total links: %d (from %d sources)", totalLinks, len(linkIndex.Forward)), nil
		case "backlinks":
			if len(args) == 0 {
				return "", errors.New("usage: links backlinks <entity-id>")
			}
			incoming := linkMgr.GetIncoming(args[0])
			if len(incoming) == 0 {
				return "No backlinks to " + args[0], nil
			}
			var lines []string
			for _, link := range incoming {
				lines = append(lines, "• "+link.Source)
			}

			return fmt.Sprintf("Backlinks to %s:\n%s", args[0], strings.Join(lines, "\n")), nil
		case "search":
			if len(args) == 0 {
				return "", errors.New("usage: links search <query>")
			}
			query := strings.Join(args, " ")
			queryLower := strings.ToLower(query)
			names := linkMgr.GetNames()
			var matches []string
			// Search in specs
			for name := range names.Specs {
				if strings.Contains(strings.ToLower(name), queryLower) {
					matches = append(matches, "spec: "+name)
				}
			}
			// Search in decisions
			for name := range names.Decisions {
				if strings.Contains(strings.ToLower(name), queryLower) {
					matches = append(matches, "decision: "+name)
				}
			}
			if len(matches) == 0 {
				return "No matching entities found", nil
			}

			return fmt.Sprintf("Found %d match(es):\n• %s", len(matches), strings.Join(matches, "\n• ")), nil
		case "stats":
			stats := linkMgr.GetStats()
			if stats == nil {
				return "", errors.New("failed to get link stats")
			}

			return fmt.Sprintf("Link Stats:\n• Total links: %d\n• Sources: %d\n• Targets: %d\n• Orphans: %d",
				stats.TotalLinks, stats.TotalSources, stats.TotalTargets, stats.OrphanEntities), nil
		case "rebuild":
			if rebuildErr := linkMgr.Rebuild(); rebuildErr != nil {
				return "", fmt.Errorf("rebuild failed: %w", rebuildErr)
			}
			stats := linkMgr.GetStats()

			return fmt.Sprintf("Index rebuilt: %d links from %d sources", stats.TotalLinks, stats.TotalSources), nil
		default:
			// Treat as entity ID for getting links
			outgoing := linkMgr.GetOutgoing(subcommand)
			incoming := linkMgr.GetIncoming(subcommand)
			if len(outgoing) == 0 && len(incoming) == 0 {
				return "No links found for " + subcommand, nil
			}
			var lines []string
			if len(outgoing) > 0 {
				lines = append(lines, fmt.Sprintf("Outgoing (%d):", len(outgoing)))
				for _, link := range outgoing {
					lines = append(lines, "  → "+link.Target)
				}
			}
			if len(incoming) > 0 {
				lines = append(lines, fmt.Sprintf("Incoming (%d):", len(incoming)))
				for _, link := range incoming {
					lines = append(lines, "  ← "+link.Source)
				}
			}

			return strings.Join(lines, "\n"), nil
		}

	default:
		return "", fmt.Errorf("unknown explore command: %s", command)
	}
}
