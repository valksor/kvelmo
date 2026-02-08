package commands

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/valksor/go-mehrhof/internal/conductor"
	"github.com/valksor/go-mehrhof/internal/library"
	"github.com/valksor/go-mehrhof/internal/links"
	"github.com/valksor/go-mehrhof/internal/memory"
	"github.com/valksor/go-mehrhof/internal/storage"
)

type libraryCommandOptions struct {
	Shared        bool     `json:"shared,omitempty"`
	SharedOnly    bool     `json:"shared_only,omitempty"`
	ProjectOnly   bool     `json:"project_only,omitempty"`
	Tag           string   `json:"tag,omitempty"`
	Name          string   `json:"name,omitempty"`
	Mode          string   `json:"mode,omitempty"`
	Paths         []string `json:"paths,omitempty"`
	Tags          []string `json:"tags,omitempty"`
	MaxDepth      int      `json:"max_depth,omitempty"`
	MaxPages      int      `json:"max_pages,omitempty"`
	Continue      bool     `json:"continue,omitempty"`
	Restart       bool     `json:"restart,omitempty"`
	DomainScope   string   `json:"domain_scope,omitempty"`
	VersionFilter bool     `json:"version_filter,omitempty"`
	Version       string   `json:"version,omitempty"`
	DryRun        bool     `json:"dry_run,omitempty"`
}

func init() {
	Register(Command{
		Info: CommandInfo{
			Name:         "find",
			Aliases:      []string{"search"},
			Description:  "Search codebase for patterns",
			Category:     "exploration",
			RequiresTask: false,
		},
		Handler: handleFind,
	})

	Register(Command{
		Info: CommandInfo{
			Name:         "label",
			Description:  "Manage task labels",
			Category:     "task",
			RequiresTask: true,
			MutatesState: true,
		},
		Handler: handleLabel,
	})

	Register(Command{
		Info: CommandInfo{
			Name:         "labels",
			Description:  "List labels across all tasks",
			Category:     "task",
			RequiresTask: false,
		},
		Handler: handleLabels,
	})

	Register(Command{
		Info: CommandInfo{
			Name:         "memory",
			Aliases:      []string{"mem"},
			Description:  "Search and manage semantic memory",
			Category:     "exploration",
			RequiresTask: false,
		},
		Handler: handleMemory,
	})

	Register(Command{
		Info: CommandInfo{
			Name:         "library",
			Aliases:      []string{"lib"},
			Description:  "Search and manage library collections",
			Category:     "exploration",
			RequiresTask: false,
			MutatesState: true,
		},
		Handler: handleLibrary,
	})

	Register(Command{
		Info: CommandInfo{
			Name:         "links",
			Description:  "Explore workspace link graph",
			Category:     "exploration",
			RequiresTask: false,
			MutatesState: true,
		},
		Handler: handleLinks,
	})

	Register(Command{
		Info: CommandInfo{
			Name:         "notes",
			Description:  "Read notes for a task",
			Category:     "task",
			RequiresTask: false,
		},
		Handler: handleNotes,
	})
}

func handleFind(ctx context.Context, cond *conductor.Conductor, inv Invocation) (*Result, error) {
	query := strings.TrimSpace(strings.Join(inv.Args, " "))
	if query == "" {
		query = strings.TrimSpace(GetString(inv.Options, "query"))
	}
	if query == "" {
		return nil, errors.New("find requires a query")
	}

	findOpts := conductor.FindOptions{
		Query:     query,
		Path:      GetString(inv.Options, "path"),
		Pattern:   GetString(inv.Options, "pattern"),
		Context:   GetInt(inv.Options, "context"),
		Workspace: cond.GetWorkspace(),
	}
	resultChan, err := cond.Find(ctx, findOpts)
	if err != nil {
		return nil, fmt.Errorf("find: %w", err)
	}

	results := make([]conductor.FindResult, 0)
	for result := range resultChan {
		if result.File == "__error__" {
			return nil, errors.New(result.Snippet)
		}
		results = append(results, result)
	}

	return NewResult(fmt.Sprintf("Found %d match(es)", len(results))).WithData(map[string]any{
		"query":   query,
		"count":   len(results),
		"matches": results,
	}), nil
}

func handleLabel(_ context.Context, cond *conductor.Conductor, inv Invocation) (*Result, error) {
	task := cond.GetActiveTask()
	if task == nil {
		return nil, ErrNoActiveTask
	}
	ws := cond.GetWorkspace()
	if ws == nil {
		return nil, errors.New("workspace not initialized")
	}

	if len(inv.Args) == 0 {
		labels, err := ws.GetLabels(task.ID)
		if err != nil {
			return nil, fmt.Errorf("get labels: %w", err)
		}

		return NewResult("labels loaded").WithData(map[string]any{
			"task_id": task.ID,
			"labels":  labels,
		}), nil
	}

	subCmd := inv.Args[0]
	subArgs := inv.Args[1:]
	switch subCmd {
	case "add":
		for _, label := range subArgs {
			if err := ws.AddLabel(task.ID, label); err != nil {
				return nil, fmt.Errorf("add label %q: %w", label, err)
			}
		}
	case "remove", "rm":
		for _, label := range subArgs {
			if err := ws.RemoveLabel(task.ID, label); err != nil {
				return nil, fmt.Errorf("remove label %q: %w", label, err)
			}
		}
	case "set":
		if err := ws.SetLabels(task.ID, subArgs); err != nil {
			return nil, fmt.Errorf("set labels: %w", err)
		}
	case "clear":
		if err := ws.SetLabels(task.ID, []string{}); err != nil {
			return nil, fmt.Errorf("clear labels: %w", err)
		}
	case "list", "ls":
		labels, err := ws.GetLabels(task.ID)
		if err != nil {
			return nil, fmt.Errorf("get labels: %w", err)
		}

		return NewResult("labels loaded").WithData(map[string]any{
			"task_id": task.ID,
			"labels":  labels,
		}), nil
	default:
		for _, label := range inv.Args {
			if err := ws.AddLabel(task.ID, label); err != nil {
				return nil, fmt.Errorf("add label %q: %w", label, err)
			}
		}
	}

	labels, err := ws.GetLabels(task.ID)
	if err != nil {
		return nil, fmt.Errorf("get labels: %w", err)
	}

	action := subCmd
	if action == "rm" {
		action = "remove"
	}
	if action != "add" && action != "remove" && action != "set" && action != "clear" {
		action = "add"
	}

	return NewResult("labels updated").WithData(map[string]any{
		"success": true,
		"task_id": task.ID,
		"action":  action,
		"labels":  labels,
	}), nil
}

func handleLabels(_ context.Context, cond *conductor.Conductor, _ Invocation) (*Result, error) {
	ws := cond.GetWorkspace()
	if ws == nil {
		return nil, errors.New("workspace not initialized")
	}

	taskIDs, err := ws.ListWorks()
	if err != nil {
		return nil, fmt.Errorf("failed to list tasks: %w", err)
	}

	labelCounts := make(map[string]int)
	for _, taskID := range taskIDs {
		work, workErr := ws.LoadWork(taskID)
		if workErr != nil || work == nil {
			continue
		}
		for _, label := range work.Metadata.Labels {
			labelCounts[label]++
		}
	}

	type labelInfo struct {
		Label string `json:"label"`
		Count int    `json:"count"`
	}

	labels := make([]labelInfo, 0, len(labelCounts))
	for label, count := range labelCounts {
		labels = append(labels, labelInfo{Label: label, Count: count})
	}
	slices.SortFunc(labels, func(a, b labelInfo) int {
		if a.Label < b.Label {
			return -1
		}
		if a.Label > b.Label {
			return 1
		}

		return 0
	})

	return NewResult("labels loaded").WithData(map[string]any{
		"labels": labels,
		"count":  len(labels),
	}), nil
}

func handleMemory(ctx context.Context, cond *conductor.Conductor, inv Invocation) (*Result, error) {
	mem := cond.GetMemory()
	if len(inv.Args) == 0 {
		return nil, errors.New("memory requires a subcommand: search <query>, index <task-id>, stats")
	}

	subcommand := inv.Args[0]
	subArgs := inv.Args[1:]
	switch subcommand {
	case "search":
		query := strings.TrimSpace(strings.Join(subArgs, " "))
		if query == "" {
			query = strings.TrimSpace(GetString(inv.Options, "query"))
		}
		if query == "" {
			return nil, errors.New("memory search requires a query")
		}
		if mem == nil {
			return NewResult("Found 0 similar task(s)").WithData(map[string]any{
				"results": []map[string]any{},
				"count":   0,
			}), nil
		}

		documentTypes := make([]memory.DocumentType, 0)
		for _, raw := range strings.Split(strings.ToLower(strings.TrimSpace(GetString(inv.Options, "types"))), ",") {
			switch strings.TrimSpace(raw) {
			case "code_change", "code":
				documentTypes = append(documentTypes, memory.TypeCodeChange)
			case "specification", "spec":
				documentTypes = append(documentTypes, memory.TypeSpecification)
			case "session":
				documentTypes = append(documentTypes, memory.TypeSession)
			case "solution":
				documentTypes = append(documentTypes, memory.TypeSolution)
			case "decision":
				documentTypes = append(documentTypes, memory.TypeDecision)
			case "error":
				documentTypes = append(documentTypes, memory.TypeError)
			}
		}

		limit := GetInt(inv.Options, "limit")
		if limit <= 0 {
			limit = 5
		}

		results, err := mem.Search(ctx, query, memory.SearchOptions{
			Limit:         limit,
			MinScore:      0.65,
			DocumentTypes: documentTypes,
		})
		if err != nil {
			return nil, fmt.Errorf("memory search: %w", err)
		}

		memResults := make([]map[string]any, 0, len(results))
		for _, result := range results {
			memResults = append(memResults, map[string]any{
				"task_id":  result.Document.TaskID,
				"type":     string(result.Document.Type),
				"score":    float64(result.Score),
				"content":  result.Document.Content,
				"metadata": result.Document.Metadata,
			})
		}

		return NewResult(fmt.Sprintf("Found %d similar task(s)", len(results))).WithData(map[string]any{
			"results": memResults,
			"count":   len(memResults),
		}), nil
	case "index":
		if len(subArgs) == 0 {
			return nil, errors.New("memory index requires a task ID")
		}
		if mem == nil {
			return nil, errors.New("memory system is not available")
		}

		ws := cond.GetWorkspace()
		if ws == nil {
			return nil, errors.New("workspace not initialized")
		}
		taskID := subArgs[0]
		if _, err := ws.LoadWork(taskID); err != nil {
			return nil, fmt.Errorf("task not found: %w", err)
		}

		indexer := memory.NewIndexer(mem, ws, nil)
		if err := indexer.IndexTask(ctx, taskID); err != nil {
			return nil, fmt.Errorf("failed to index task: %w", err)
		}

		return NewResult("task indexed successfully").WithData(map[string]any{
			"success": true,
			"message": "task indexed successfully",
			"task_id": taskID,
		}), nil
	case "stats":
		if mem == nil {
			return NewResult("Memory disabled").WithData(map[string]any{
				"total_documents": 0,
				"by_type":         map[string]int{},
				"enabled":         false,
			}), nil
		}

		ws := cond.GetWorkspace()
		if ws == nil {
			return nil, errors.New("workspace not initialized")
		}

		indexer := memory.NewIndexer(mem, ws, nil)
		stats, err := indexer.GetStats(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get stats: %w", err)
		}

		return NewResult(fmt.Sprintf("Total documents: %d", stats.TotalDocuments)).WithData(map[string]any{
			"total_documents": stats.TotalDocuments,
			"by_type":         stats.ByType,
			"enabled":         true,
		}), nil
	default:
		if mem == nil {
			return NewResult("Found 0 similar task(s)").WithData(map[string]any{
				"results": []map[string]any{},
				"count":   0,
			}), nil
		}

		query := strings.Join(inv.Args, " ")
		results, err := mem.Search(ctx, query, memory.SearchOptions{
			Limit:    5,
			MinScore: 0.65,
		})
		if err != nil {
			return nil, fmt.Errorf("memory search: %w", err)
		}

		memResults := make([]map[string]any, 0, len(results))
		for _, result := range results {
			memResults = append(memResults, map[string]any{
				"task_id":  result.Document.TaskID,
				"type":     string(result.Document.Type),
				"score":    float64(result.Score),
				"content":  result.Document.Content,
				"metadata": result.Document.Metadata,
			})
		}

		return NewResult(fmt.Sprintf("Found %d similar task(s)", len(results))).WithData(map[string]any{
			"results": memResults,
			"count":   len(memResults),
		}), nil
	}
}

func handleLibrary(ctx context.Context, cond *conductor.Conductor, inv Invocation) (*Result, error) {
	opts, err := DecodeOptions[libraryCommandOptions](inv)
	if err != nil {
		return nil, err
	}

	lib := cond.GetLibrary()

	subcommand := "list"
	subArgs := inv.Args
	if len(subArgs) > 0 {
		subcommand = subArgs[0]
		subArgs = subArgs[1:]
	}

	switch subcommand {
	case "list", "ls":
		if lib == nil {
			return NewResult("library disabled").WithData(map[string]any{
				"enabled":     false,
				"collections": []map[string]any{},
				"count":       0,
			}), nil
		}

		listOpts := &library.ListOptions{
			SharedOnly:  opts.SharedOnly || opts.Shared,
			ProjectOnly: opts.ProjectOnly,
			Tag:         strings.TrimSpace(opts.Tag),
		}
		collections, err := lib.List(ctx, listOpts)
		if err != nil {
			return nil, fmt.Errorf("list collections: %w", err)
		}

		items := make([]map[string]any, 0, len(collections))
		for _, collection := range collections {
			if collection == nil {
				continue
			}
			items = append(items, libraryCollectionToMap(*collection))
		}

		return NewResult(fmt.Sprintf("Found %d collection(s)", len(collections))).WithData(map[string]any{
			"enabled":     true,
			"collections": items,
			"count":       len(items),
		}), nil
	case "show":
		if lib == nil {
			return nil, errors.New("library system not available")
		}
		if len(subArgs) == 0 {
			return nil, errors.New("usage: library show <name>")
		}
		coll, err := lib.Show(ctx, strings.TrimSpace(subArgs[0]))
		if err != nil {
			return nil, fmt.Errorf("show collection: %w", err)
		}
		pages, _ := lib.ListPages(ctx, coll.ID)

		return NewResult(fmt.Sprintf("Collection: %s (%d pages)", coll.Name, coll.PageCount)).WithData(map[string]any{
			"collection": libraryCollectionToMap(*coll),
			"pages":      pages,
		}), nil
	case "items":
		if lib == nil {
			return nil, errors.New("library system not available")
		}
		if len(subArgs) == 0 {
			return nil, errors.New("usage: library items <id>")
		}
		coll, err := lib.Show(ctx, strings.TrimSpace(subArgs[0]))
		if err != nil {
			return nil, fmt.Errorf("show collection: %w", err)
		}
		pagePaths, err := lib.ListPages(ctx, coll.ID)
		if err != nil {
			return nil, fmt.Errorf("list pages: %w", err)
		}

		items := make([]map[string]any, 0, len(pagePaths))
		for _, pagePath := range pagePaths {
			page, content, showErr := lib.ShowPage(ctx, coll.ID, pagePath)
			if showErr != nil {
				continue
			}
			title := pagePath
			if page != nil && page.Title != "" {
				title = page.Title
			}
			entry := map[string]any{
				"id":         pagePath,
				"title":      title,
				"content":    content,
				"collection": coll.ID,
			}
			items = append(items, entry)
		}

		return NewResult(fmt.Sprintf("Loaded %d item(s)", len(items))).WithData(map[string]any{
			"collection": coll.ID,
			"items":      items,
			"count":      len(items),
		}), nil
	case "search":
		if lib == nil {
			return nil, errors.New("library system not available")
		}
		if len(subArgs) == 0 {
			return nil, errors.New("usage: library search <query>")
		}
		searchQuery := strings.Join(subArgs, " ")
		docCtx, err := lib.GetDocsForQuery(ctx, searchQuery, 10000)
		if err != nil {
			return nil, fmt.Errorf("search library: %w", err)
		}
		if docCtx == nil || len(docCtx.Pages) == 0 {
			return NewResult("No matching documentation found").WithData(map[string]any{
				"query":   searchQuery,
				"results": []map[string]any{},
				"count":   0,
			}), nil
		}

		collectionSet := make(map[string]bool)
		for _, p := range docCtx.Pages {
			collectionSet[p.CollectionName] = true
		}

		return NewResult(fmt.Sprintf("Found %d page(s) from %d collection(s)", len(docCtx.Pages), len(collectionSet))).WithData(map[string]any{
			"query":   searchQuery,
			"results": docCtx.Pages,
			"count":   len(docCtx.Pages),
		}), nil
	case "pull":
		if lib == nil {
			return nil, errors.New("library system not available")
		}
		if len(subArgs) == 0 {
			return nil, errors.New("usage: library pull <source>")
		}
		source := subArgs[0]
		pullOpts := &library.PullOptions{}
		for i := 1; i < len(subArgs); i++ {
			if subArgs[i] == "--name" && i+1 < len(subArgs) {
				pullOpts.Name = subArgs[i+1]
				i++
			} else if subArgs[i] == "--shared" {
				pullOpts.Shared = true
			}
		}
		if strings.TrimSpace(opts.Name) != "" {
			pullOpts.Name = strings.TrimSpace(opts.Name)
		}
		pullOpts.Shared = pullOpts.Shared || opts.Shared
		pullOpts.IncludeMode = libraryIncludeModeFromString(opts.Mode)
		pullOpts.Paths = append(pullOpts.Paths, opts.Paths...)
		pullOpts.Tags = append(pullOpts.Tags, opts.Tags...)
		pullOpts.MaxDepth = opts.MaxDepth
		pullOpts.MaxPages = opts.MaxPages
		pullOpts.Continue = opts.Continue
		pullOpts.ForceRestart = opts.Restart
		pullOpts.DomainScope = strings.TrimSpace(opts.DomainScope)
		pullOpts.VersionFilter = opts.VersionFilter
		pullOpts.VersionPath = strings.TrimSpace(opts.Version)
		pullOpts.DryRun = opts.DryRun

		pullResult, err := lib.Pull(ctx, source, pullOpts)
		if err != nil {
			var incompleteErr *library.IncompleteCrawlError
			if errors.As(err, &incompleteErr) {
				return &Result{
					Type:    ResultConflict,
					Message: incompleteErr.Error(),
					Data: map[string]any{
						"error":         "incomplete_crawl",
						"message":       incompleteErr.Error(),
						"collection_id": incompleteErr.CollectionID,
						"total":         incompleteErr.Total,
						"success":       incompleteErr.Success,
						"failed":        incompleteErr.Failed,
						"pending":       incompleteErr.Pending,
						"started_at":    incompleteErr.StartedAt,
					},
				}, nil
			}

			return nil, fmt.Errorf("pull library: %w", err)
		}
		if pullOpts.DryRun {
			return NewResult("library pull preview complete").WithData(map[string]any{
				"urls":  pullResult.DryRunURLs,
				"count": len(pullResult.DryRunURLs),
			}), nil
		}

		return NewResult(fmt.Sprintf("Pulled collection: %s (%d pages)", pullResult.Collection.Name, pullResult.Collection.PageCount)).WithData(map[string]any{
			"success":       true,
			"collection_id": pullResult.Collection.ID,
			"name":          pullResult.Collection.Name,
			"pages_written": pullResult.PagesWritten,
			"source":        source,
		}), nil
	case "remove", "rm":
		if lib == nil {
			return nil, errors.New("library system not available")
		}
		if len(subArgs) == 0 {
			return nil, errors.New("usage: library remove <name>")
		}
		name := strings.TrimSpace(subArgs[0])
		if err := lib.Remove(ctx, name, false); err != nil {
			return nil, fmt.Errorf("remove collection: %w", err)
		}

		return NewResult("collection removed successfully").WithData(map[string]any{
			"success": true,
			"message": "collection removed successfully",
		}), nil
	case "stats":
		if lib == nil {
			return NewResult("library disabled").WithData(map[string]any{
				"enabled": false,
			}), nil
		}

		collections, err := lib.List(ctx, &library.ListOptions{
			SharedOnly: opts.SharedOnly,
		})
		if err != nil {
			return nil, fmt.Errorf("list collections: %w", err)
		}
		totalPages := 0
		totalSize := int64(0)
		projectCount := 0
		sharedCount := 0
		byMode := make(map[string]int)
		for _, collection := range collections {
			totalPages += collection.PageCount
			totalSize += collection.TotalSize
			byMode[string(collection.IncludeMode)]++
			if collection.Location == "shared" {
				sharedCount++
			} else {
				projectCount++
			}
		}

		return NewResult(fmt.Sprintf("%d collections, %d total pages", len(collections), totalPages)).WithData(map[string]any{
			"total_collections": len(collections),
			"total_pages":       totalPages,
			"total_size":        totalSize,
			"project_count":     projectCount,
			"shared_count":      sharedCount,
			"by_mode":           byMode,
			"enabled":           true,
		}), nil
	default:
		if lib == nil {
			return nil, errors.New("library system not available")
		}

		coll, err := lib.Show(ctx, subcommand)
		if err != nil {
			return nil, fmt.Errorf("show collection: %w", err)
		}
		pages, _ := lib.ListPages(ctx, coll.ID)

		return NewResult(fmt.Sprintf("Collection: %s (%d pages)", coll.Name, coll.PageCount)).WithData(map[string]any{
			"collection": libraryCollectionToMap(*coll),
			"pages":      pages,
		}), nil
	}
}

func handleLinks(ctx context.Context, cond *conductor.Conductor, inv Invocation) (*Result, error) {
	ws := cond.GetWorkspace()
	if ws == nil {
		return nil, errors.New("workspace not initialized")
	}

	linkMgr := storage.GetLinkManager(ctx, ws)

	subcommand := "list"
	subArgs := inv.Args
	if len(subArgs) > 0 {
		subcommand = subArgs[0]
		subArgs = subArgs[1:]
	}

	switch subcommand {
	case "list", "ls":
		if linkMgr == nil {
			return NewResult("0 links from 0 sources").WithData(map[string]any{
				"links": []map[string]any{},
				"count": 0,
			}), nil
		}

		linkIndex := linkMgr.GetIndex()
		if linkIndex == nil {
			return NewResult("0 links from 0 sources").WithData(map[string]any{
				"links": []map[string]any{},
				"count": 0,
			}), nil
		}

		allLinks := make([]map[string]any, 0)
		for _, forwardLinks := range linkIndex.Forward {
			for _, link := range forwardLinks {
				allLinks = append(allLinks, linkToMap(link))
			}
		}

		return NewResult(fmt.Sprintf("%d links from %d sources", len(allLinks), len(linkIndex.Forward))).WithData(map[string]any{
			"links": allLinks,
			"count": len(allLinks),
		}), nil
	case "backlinks":
		if len(subArgs) == 0 {
			return nil, errors.New("usage: links backlinks <entity-id>")
		}
		if linkMgr == nil {
			return NewResult("0 backlinks").WithData(map[string]any{
				"entity_id": subArgs[0],
				"backlinks": []map[string]any{},
				"total":     0,
			}), nil
		}
		incoming := linkMgr.GetIncoming(subArgs[0])
		backlinks := make([]map[string]any, 0, len(incoming))
		for _, link := range incoming {
			backlinks = append(backlinks, linkToMap(link))
		}

		return NewResult(fmt.Sprintf("%d backlinks to %s", len(incoming), subArgs[0])).WithData(map[string]any{
			"entity_id": subArgs[0],
			"backlinks": backlinks,
			"total":     len(backlinks),
		}), nil
	case "search":
		query := strings.TrimSpace(strings.Join(subArgs, " "))
		if query == "" {
			query = strings.TrimSpace(GetString(inv.Options, "query"))
		}
		if query == "" {
			return nil, errors.New("usage: links search <query>")
		}
		if linkMgr == nil {
			return NewResult("Found 0 matching entities").WithData(map[string]any{
				"query":   query,
				"results": []map[string]any{},
				"count":   0,
			}), nil
		}

		queryLower := strings.ToLower(query)
		names := linkMgr.GetNames()
		if names == nil {
			return NewResult("Found 0 matching entities").WithData(map[string]any{
				"query":   query,
				"results": []map[string]any{},
				"count":   0,
			}), nil
		}

		results := make([]map[string]any, 0)
		searchLinkRegistry(names.Specs, queryLower, "spec", &results)
		searchLinkRegistry(names.Sessions, queryLower, "session", &results)
		searchLinkRegistry(names.Decisions, queryLower, "decision", &results)
		searchLinkRegistry(names.Tasks, queryLower, "task", &results)
		searchLinkRegistry(names.Notes, queryLower, "note", &results)

		return NewResult(fmt.Sprintf("Found %d matching entities", len(results))).WithData(map[string]any{
			"query":   query,
			"results": results,
			"count":   len(results),
		}), nil
	case "stats":
		if linkMgr == nil {
			return NewResult("links disabled").WithData(map[string]any{
				"total_links":     0,
				"total_sources":   0,
				"total_targets":   0,
				"orphan_entities": 0,
				"most_linked":     []map[string]any{},
				"enabled":         false,
			}), nil
		}

		stats := linkMgr.GetStats()
		if stats == nil {
			return nil, errors.New("failed to get link stats")
		}

		linkIndex := linkMgr.GetIndex()
		mostLinked := make([]map[string]any, 0)
		if linkIndex != nil {
			for source, forwardLinks := range linkIndex.Forward {
				typ, taskID, id := links.ParseEntityID(source)
				totalLinks := len(forwardLinks) + len(linkIndex.Backward[source])
				mostLinked = append(mostLinked, map[string]any{
					"entity_id":   source,
					"type":        string(typ),
					"task_id":     taskID,
					"id":          id,
					"total_links": totalLinks,
				})
			}
		}
		slices.SortFunc(mostLinked, func(a, b map[string]any) int {
			left, _ := a["total_links"].(int)
			right, _ := b["total_links"].(int)
			switch {
			case left > right:
				return -1
			case left < right:
				return 1
			default:
				return 0
			}
		})
		if len(mostLinked) > 10 {
			mostLinked = mostLinked[:10]
		}

		return NewResult(fmt.Sprintf("%d links, %d sources, %d targets", stats.TotalLinks, stats.TotalSources, stats.TotalTargets)).WithData(map[string]any{
			"total_links":     stats.TotalLinks,
			"total_sources":   stats.TotalSources,
			"total_targets":   stats.TotalTargets,
			"orphan_entities": stats.OrphanEntities,
			"most_linked":     mostLinked,
			"enabled":         true,
		}), nil
	case "rebuild":
		if linkMgr == nil {
			return nil, errors.New("links not available")
		}
		if err := linkMgr.Rebuild(); err != nil {
			return nil, fmt.Errorf("rebuild failed: %w", err)
		}
		stats := linkMgr.GetStats()

		return NewResult(fmt.Sprintf("Index rebuilt: %d links", stats.TotalLinks)).WithData(map[string]any{
			"success":       true,
			"message":       "index rebuilt successfully",
			"total_links":   stats.TotalLinks,
			"total_sources": stats.TotalSources,
			"total_targets": stats.TotalTargets,
		}), nil
	default:
		entityID := strings.TrimSpace(subcommand)
		if entityID == "" {
			return nil, errors.New("entity ID is required")
		}
		if linkMgr == nil {
			return NewResult("0 links").WithData(map[string]any{
				"entity_id": entityID,
				"outgoing":  []map[string]any{},
				"incoming":  []map[string]any{},
			}), nil
		}

		outgoing := linkMgr.GetOutgoing(entityID)
		incoming := linkMgr.GetIncoming(entityID)
		outgoingData := make([]map[string]any, 0, len(outgoing))
		for _, link := range outgoing {
			outgoingData = append(outgoingData, linkToMap(link))
		}
		incomingData := make([]map[string]any, 0, len(incoming))
		for _, link := range incoming {
			incomingData = append(incomingData, linkToMap(link))
		}

		return NewResult(fmt.Sprintf("%d outgoing, %d incoming links for %s", len(outgoing), len(incoming), entityID)).WithData(map[string]any{
			"entity_id": entityID,
			"outgoing":  outgoingData,
			"incoming":  incomingData,
		}), nil
	}
}

func libraryCollectionToMap(collection library.Collection) map[string]any {
	return map[string]any{
		"id":           collection.ID,
		"name":         collection.Name,
		"source":       collection.Source,
		"source_type":  string(collection.SourceType),
		"include_mode": string(collection.IncludeMode),
		"page_count":   collection.PageCount,
		"total_size":   collection.TotalSize,
		"location":     collection.Location,
		"pulled_at":    collection.PulledAt,
		"tags":         collection.Tags,
		"paths":        collection.Paths,
	}
}

func libraryIncludeModeFromString(raw string) library.IncludeMode {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "explicit":
		return library.IncludeModeExplicit
	case "always":
		return library.IncludeModeAlways
	default:
		return library.IncludeModeAuto
	}
}

func linkToMap(link links.Link) map[string]any {
	return map[string]any{
		"source":     link.Source,
		"target":     link.Target,
		"context":    link.Context,
		"created_at": link.CreatedAt.Format(time.RFC3339),
	}
}

func searchLinkRegistry(registry map[string]string, queryLower, entityType string, out *[]map[string]any) {
	for name, entityID := range registry {
		if strings.Contains(strings.ToLower(name), queryLower) {
			typ, taskID, id := links.ParseEntityID(entityID)
			*out = append(*out, map[string]any{
				"entity_id": entityID,
				"type":      entityType,
				"name":      name,
				"task_id":   taskID,
				"id":        id,
				"full_type": string(typ),
			})
		}
	}
}

func handleNotes(_ context.Context, cond *conductor.Conductor, inv Invocation) (*Result, error) {
	ws := cond.GetWorkspace()
	if ws == nil {
		return nil, errors.New("workspace not initialized")
	}

	taskID := GetString(inv.Options, "task_id")
	if taskID == "" && len(inv.Args) > 0 {
		taskID = strings.TrimSpace(inv.Args[0])
	}
	if taskID == "" {
		if task := cond.GetActiveTask(); task != nil {
			taskID = task.ID
		}
	}
	if taskID == "" {
		return nil, errors.New("task ID is required")
	}

	content, err := ws.ReadNotes(taskID)
	if err != nil {
		content = ""
	}

	return NewResult("Notes loaded").WithData(map[string]any{
		"task_id": taskID,
		"content": content,
	}), nil
}
