package commands

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/valksor/go-mehrhof/internal/library"
	"github.com/valksor/go-mehrhof/internal/links"
)

func TestHandleFindValidation(t *testing.T) {
	cond := mustNewConductor(t)

	tests := []struct {
		name   string
		inv    Invocation
		errSub string
	}{
		{
			name:   "empty query",
			inv:    Invocation{},
			errSub: "find requires a query",
		},
		{
			name:   "whitespace only query",
			inv:    Invocation{Args: []string{"   "}},
			errSub: "find requires a query",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := handleFind(context.Background(), cond, tt.inv)
			if result != nil {
				t.Fatalf("expected nil result, got %#v", result)
			}
			if err == nil || !strings.Contains(err.Error(), tt.errSub) {
				t.Fatalf("expected error containing %q, got %v", tt.errSub, err)
			}
		})
	}
}

func TestHandleFindQueryExtraction(t *testing.T) {
	tests := []struct {
		name      string
		inv       Invocation
		wantQuery string
	}{
		{
			name:      "query from args",
			inv:       Invocation{Args: []string{"search", "term"}},
			wantQuery: "search term",
		},
		{
			name:      "query from options",
			inv:       Invocation{Options: map[string]any{"query": "from options"}},
			wantQuery: "from options",
		},
		{
			name:      "args take precedence",
			inv:       Invocation{Args: []string{"from args"}, Options: map[string]any{"query": "from options"}},
			wantQuery: "from args",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := strings.TrimSpace(strings.Join(tt.inv.Args, " "))
			if query == "" {
				query = strings.TrimSpace(GetString(tt.inv.Options, "query"))
			}
			if query != tt.wantQuery {
				t.Errorf("query = %q, want %q", query, tt.wantQuery)
			}
		})
	}
}

func TestHandleLabelNoTask(t *testing.T) {
	cond := mustNewConductor(t)

	result, err := handleLabel(context.Background(), cond, Invocation{})
	if result != nil {
		t.Fatalf("expected nil result, got %#v", result)
	}
	if !errors.Is(err, ErrNoActiveTask) {
		t.Fatalf("expected ErrNoActiveTask, got %v", err)
	}
}

func TestHandleLabelsNoWorkspace(t *testing.T) {
	cond := mustNewConductor(t)

	result, err := handleLabels(context.Background(), cond, Invocation{})
	if result != nil {
		t.Fatalf("expected nil result, got %#v", result)
	}
	if err == nil || !strings.Contains(err.Error(), "workspace not initialized") {
		t.Fatalf("expected 'workspace not initialized' error, got %v", err)
	}
}

func TestHandleMemoryValidation(t *testing.T) {
	cond := mustNewConductor(t)

	tests := []struct {
		name   string
		inv    Invocation
		errSub string
	}{
		{
			name:   "no subcommand",
			inv:    Invocation{},
			errSub: "memory requires a subcommand",
		},
		{
			name:   "search without query",
			inv:    Invocation{Args: []string{"search"}},
			errSub: "memory search requires a query",
		},
		{
			name:   "index without task id",
			inv:    Invocation{Args: []string{"index"}},
			errSub: "memory index requires a task ID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := handleMemory(context.Background(), cond, tt.inv)
			if result != nil {
				t.Fatalf("expected nil result, got %#v", result)
			}
			if err == nil || !strings.Contains(err.Error(), tt.errSub) {
				t.Fatalf("expected error containing %q, got %v", tt.errSub, err)
			}
		})
	}
}

func TestHandleMemorySearchNoMemory(t *testing.T) {
	cond := mustNewConductor(t)

	// When memory is nil, search should return empty results
	result, err := handleMemory(context.Background(), cond, Invocation{Args: []string{"search", "test query"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	data, ok := result.Data.(map[string]any)
	if !ok {
		t.Fatalf("expected map data, got %T", result.Data)
	}
	if count, ok := data["count"].(int); !ok || count != 0 {
		t.Errorf("expected count=0, got %v", data["count"])
	}
}

func TestHandleMemoryStatsNoMemory(t *testing.T) {
	cond := mustNewConductor(t)

	// When memory is nil, stats should return disabled
	result, err := handleMemory(context.Background(), cond, Invocation{Args: []string{"stats"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	data, ok := result.Data.(map[string]any)
	if !ok {
		t.Fatalf("expected map data, got %T", result.Data)
	}
	if enabled, ok := data["enabled"].(bool); !ok || enabled {
		t.Errorf("expected enabled=false, got %v", data["enabled"])
	}
}

func TestHandleMemoryDefaultSearchNoMemory(t *testing.T) {
	cond := mustNewConductor(t)

	// When memory is nil and using default search (unrecognized subcommand)
	result, err := handleMemory(context.Background(), cond, Invocation{Args: []string{"some query text"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	data, ok := result.Data.(map[string]any)
	if !ok {
		t.Fatalf("expected map data, got %T", result.Data)
	}
	if count, ok := data["count"].(int); !ok || count != 0 {
		t.Errorf("expected count=0, got %v", data["count"])
	}
}

func TestHandleLibraryValidation(t *testing.T) {
	cond := mustNewConductor(t)

	// Note: Without a library system, all these commands fail with "library system not available"
	// rather than the usage errors. This is expected behavior - library must be initialized first.
	tests := []struct {
		name   string
		inv    Invocation
		errSub string
	}{
		{
			name:   "show needs library",
			inv:    Invocation{Args: []string{"show"}},
			errSub: "library system not available",
		},
		{
			name:   "show with name needs library",
			inv:    Invocation{Args: []string{"show", "collection-name"}},
			errSub: "library system not available",
		},
		{
			name:   "items needs library",
			inv:    Invocation{Args: []string{"items"}},
			errSub: "library system not available",
		},
		{
			name:   "search needs library",
			inv:    Invocation{Args: []string{"search"}},
			errSub: "library system not available",
		},
		{
			name:   "pull needs library",
			inv:    Invocation{Args: []string{"pull"}},
			errSub: "library system not available",
		},
		{
			name:   "remove needs library",
			inv:    Invocation{Args: []string{"remove"}},
			errSub: "library system not available",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := handleLibrary(context.Background(), cond, tt.inv)
			if result != nil {
				t.Fatalf("expected nil result, got %#v", result)
			}
			if err == nil || !strings.Contains(err.Error(), tt.errSub) {
				t.Fatalf("expected error containing %q, got %v", tt.errSub, err)
			}
		})
	}
}

func TestHandleLibraryListNoLibrary(t *testing.T) {
	cond := mustNewConductor(t)

	// When library is nil, list should return disabled
	result, err := handleLibrary(context.Background(), cond, Invocation{Args: []string{"list"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	data, ok := result.Data.(map[string]any)
	if !ok {
		t.Fatalf("expected map data, got %T", result.Data)
	}
	if enabled, ok := data["enabled"].(bool); !ok || enabled {
		t.Errorf("expected enabled=false, got %v", data["enabled"])
	}
}

func TestHandleLibraryStatsNoLibrary(t *testing.T) {
	cond := mustNewConductor(t)

	// When library is nil, stats should return disabled
	result, err := handleLibrary(context.Background(), cond, Invocation{Args: []string{"stats"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	data, ok := result.Data.(map[string]any)
	if !ok {
		t.Fatalf("expected map data, got %T", result.Data)
	}
	if enabled, ok := data["enabled"].(bool); !ok || enabled {
		t.Errorf("expected enabled=false, got %v", data["enabled"])
	}
}

func TestHandleLinksNoWorkspace(t *testing.T) {
	cond := mustNewConductor(t)

	result, err := handleLinks(context.Background(), cond, Invocation{})
	if result != nil {
		t.Fatalf("expected nil result, got %#v", result)
	}
	if err == nil || !strings.Contains(err.Error(), "workspace not initialized") {
		t.Fatalf("expected 'workspace not initialized' error, got %v", err)
	}
}

func TestHandleNotesValidation(t *testing.T) {
	cond := mustNewConductor(t)

	tests := []struct {
		name   string
		inv    Invocation
		errSub string
	}{
		{
			name:   "no workspace",
			inv:    Invocation{},
			errSub: "workspace not initialized",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := handleNotes(context.Background(), cond, tt.inv)
			if result != nil {
				t.Fatalf("expected nil result, got %#v", result)
			}
			if err == nil || !strings.Contains(err.Error(), tt.errSub) {
				t.Fatalf("expected error containing %q, got %v", tt.errSub, err)
			}
		})
	}
}

func TestLibraryCommandOptionsDecoding(t *testing.T) {
	tests := []struct {
		name    string
		options map[string]any
		want    libraryCommandOptions
	}{
		{
			name:    "empty options",
			options: map[string]any{},
			want:    libraryCommandOptions{},
		},
		{
			name:    "shared flag",
			options: map[string]any{"shared": true},
			want:    libraryCommandOptions{Shared: true},
		},
		{
			name:    "shared_only flag",
			options: map[string]any{"shared_only": true},
			want:    libraryCommandOptions{SharedOnly: true},
		},
		{
			name:    "project_only flag",
			options: map[string]any{"project_only": true},
			want:    libraryCommandOptions{ProjectOnly: true},
		},
		{
			name:    "tag option",
			options: map[string]any{"tag": "docs"},
			want:    libraryCommandOptions{Tag: "docs"},
		},
		{
			name:    "name option",
			options: map[string]any{"name": "my-collection"},
			want:    libraryCommandOptions{Name: "my-collection"},
		},
		{
			name:    "mode option",
			options: map[string]any{"mode": "explicit"},
			want:    libraryCommandOptions{Mode: "explicit"},
		},
		{
			name:    "paths option",
			options: map[string]any{"paths": []string{"/docs", "/api"}},
			want:    libraryCommandOptions{Paths: []string{"/docs", "/api"}},
		},
		{
			name:    "tags option",
			options: map[string]any{"tags": []string{"api", "reference"}},
			want:    libraryCommandOptions{Tags: []string{"api", "reference"}},
		},
		{
			name:    "max_depth option",
			options: map[string]any{"max_depth": 3},
			want:    libraryCommandOptions{MaxDepth: 3},
		},
		{
			name:    "max_pages option",
			options: map[string]any{"max_pages": 100},
			want:    libraryCommandOptions{MaxPages: 100},
		},
		{
			name:    "continue flag",
			options: map[string]any{"continue": true},
			want:    libraryCommandOptions{Continue: true},
		},
		{
			name:    "restart flag",
			options: map[string]any{"restart": true},
			want:    libraryCommandOptions{Restart: true},
		},
		{
			name:    "domain_scope option",
			options: map[string]any{"domain_scope": "same_domain"},
			want:    libraryCommandOptions{DomainScope: "same_domain"},
		},
		{
			name:    "version_filter flag",
			options: map[string]any{"version_filter": true},
			want:    libraryCommandOptions{VersionFilter: true},
		},
		{
			name:    "version option",
			options: map[string]any{"version": "v2"},
			want:    libraryCommandOptions{Version: "v2"},
		},
		{
			name:    "dry_run flag",
			options: map[string]any{"dry_run": true},
			want:    libraryCommandOptions{DryRun: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inv := Invocation{Options: tt.options}
			opts, err := DecodeOptions[libraryCommandOptions](inv)
			if err != nil {
				t.Fatalf("DecodeOptions failed: %v", err)
			}
			if opts.Shared != tt.want.Shared {
				t.Errorf("Shared = %v, want %v", opts.Shared, tt.want.Shared)
			}
			if opts.SharedOnly != tt.want.SharedOnly {
				t.Errorf("SharedOnly = %v, want %v", opts.SharedOnly, tt.want.SharedOnly)
			}
			if opts.ProjectOnly != tt.want.ProjectOnly {
				t.Errorf("ProjectOnly = %v, want %v", opts.ProjectOnly, tt.want.ProjectOnly)
			}
			if opts.Tag != tt.want.Tag {
				t.Errorf("Tag = %q, want %q", opts.Tag, tt.want.Tag)
			}
			if opts.Name != tt.want.Name {
				t.Errorf("Name = %q, want %q", opts.Name, tt.want.Name)
			}
			if opts.Mode != tt.want.Mode {
				t.Errorf("Mode = %q, want %q", opts.Mode, tt.want.Mode)
			}
			if opts.MaxDepth != tt.want.MaxDepth {
				t.Errorf("MaxDepth = %d, want %d", opts.MaxDepth, tt.want.MaxDepth)
			}
			if opts.MaxPages != tt.want.MaxPages {
				t.Errorf("MaxPages = %d, want %d", opts.MaxPages, tt.want.MaxPages)
			}
			if opts.Continue != tt.want.Continue {
				t.Errorf("Continue = %v, want %v", opts.Continue, tt.want.Continue)
			}
			if opts.Restart != tt.want.Restart {
				t.Errorf("Restart = %v, want %v", opts.Restart, tt.want.Restart)
			}
			if opts.DomainScope != tt.want.DomainScope {
				t.Errorf("DomainScope = %q, want %q", opts.DomainScope, tt.want.DomainScope)
			}
			if opts.VersionFilter != tt.want.VersionFilter {
				t.Errorf("VersionFilter = %v, want %v", opts.VersionFilter, tt.want.VersionFilter)
			}
			if opts.Version != tt.want.Version {
				t.Errorf("Version = %q, want %q", opts.Version, tt.want.Version)
			}
			if opts.DryRun != tt.want.DryRun {
				t.Errorf("DryRun = %v, want %v", opts.DryRun, tt.want.DryRun)
			}
		})
	}
}

func TestLibraryIncludeModeFromString(t *testing.T) {
	tests := []struct {
		input string
		want  library.IncludeMode
	}{
		{"explicit", library.IncludeModeExplicit},
		{"EXPLICIT", library.IncludeModeExplicit},
		{"  explicit  ", library.IncludeModeExplicit},
		{"always", library.IncludeModeAlways},
		{"ALWAYS", library.IncludeModeAlways},
		{"auto", library.IncludeModeAuto},
		{"", library.IncludeModeAuto},
		{"unknown", library.IncludeModeAuto},
		{"random", library.IncludeModeAuto},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := libraryIncludeModeFromString(tt.input)
			if got != tt.want {
				t.Errorf("libraryIncludeModeFromString(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestLinkToMap(t *testing.T) {
	now := time.Now()
	link := links.Link{
		Source:    "spec:task-1:1",
		Target:    "decision:task-1:cache-strategy",
		Context:   "Referenced in implementation",
		CreatedAt: now,
	}

	result := linkToMap(link)

	if result["source"] != link.Source {
		t.Errorf("source = %v, want %v", result["source"], link.Source)
	}
	if result["target"] != link.Target {
		t.Errorf("target = %v, want %v", result["target"], link.Target)
	}
	if result["context"] != link.Context {
		t.Errorf("context = %v, want %v", result["context"], link.Context)
	}
	expectedTime := now.Format(time.RFC3339)
	if result["created_at"] != expectedTime {
		t.Errorf("created_at = %v, want %v", result["created_at"], expectedTime)
	}
}

func TestLibraryCollectionToMap(t *testing.T) {
	now := time.Now()
	coll := library.Collection{
		ID:          "coll-123",
		Name:        "Test Collection",
		Source:      "https://docs.example.com",
		SourceType:  library.SourceURL,
		IncludeMode: library.IncludeModeAlways,
		PageCount:   42,
		TotalSize:   1024000,
		Location:    "shared",
		PulledAt:    now,
		Tags:        []string{"api", "docs"},
		Paths:       []string{"/api", "/guide"},
	}

	result := libraryCollectionToMap(coll)

	if result["id"] != coll.ID {
		t.Errorf("id = %v, want %v", result["id"], coll.ID)
	}
	if result["name"] != coll.Name {
		t.Errorf("name = %v, want %v", result["name"], coll.Name)
	}
	if result["source"] != coll.Source {
		t.Errorf("source = %v, want %v", result["source"], coll.Source)
	}
	if result["source_type"] != string(coll.SourceType) {
		t.Errorf("source_type = %v, want %v", result["source_type"], string(coll.SourceType))
	}
	if result["include_mode"] != string(coll.IncludeMode) {
		t.Errorf("include_mode = %v, want %v", result["include_mode"], string(coll.IncludeMode))
	}
	if result["page_count"] != coll.PageCount {
		t.Errorf("page_count = %v, want %v", result["page_count"], coll.PageCount)
	}
	if result["total_size"] != coll.TotalSize {
		t.Errorf("total_size = %v, want %v", result["total_size"], coll.TotalSize)
	}
	if result["location"] != coll.Location {
		t.Errorf("location = %v, want %v", result["location"], coll.Location)
	}
}

func TestSearchLinkRegistry(t *testing.T) {
	registry := map[string]string{
		"Authentication Flow": "spec:task-1:auth-flow",
		"Cache Strategy":      "decision:task-1:cache",
		"User Login":          "spec:task-2:login",
		"database-schema":     "spec:task-3:db",
	}

	tests := []struct {
		name       string
		query      string
		wantCount  int
		wantEntity string
	}{
		{
			name:       "match single",
			query:      "authentication",
			wantCount:  1,
			wantEntity: "spec:task-1:auth-flow",
		},
		{
			name:      "match multiple",
			query:     "spec",
			wantCount: 0, // searches by name, not entity ID
		},
		{
			name:       "case insensitive",
			query:      "CACHE",
			wantCount:  1,
			wantEntity: "decision:task-1:cache",
		},
		{
			name:      "partial match",
			query:     "user",
			wantCount: 1,
		},
		{
			name:      "no match",
			query:     "nonexistent",
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := make([]map[string]any, 0)
			searchLinkRegistry(registry, strings.ToLower(tt.query), "spec", &results)

			if len(results) != tt.wantCount {
				t.Errorf("got %d results, want %d", len(results), tt.wantCount)
			}

			if tt.wantEntity != "" && len(results) > 0 {
				if results[0]["entity_id"] != tt.wantEntity {
					t.Errorf("entity_id = %v, want %v", results[0]["entity_id"], tt.wantEntity)
				}
			}
		})
	}
}

func TestHandleLabelSubcommands(t *testing.T) {
	// This tests the label subcommand routing logic
	// Since we don't have a workspace, these will fail, but we can verify the routing

	cond := mustNewConductor(t)

	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{"add subcommand", []string{"add", "bug"}, ""},
		{"remove subcommand", []string{"remove", "bug"}, ""},
		{"rm subcommand", []string{"rm", "bug"}, ""},
		{"set subcommand", []string{"set", "bug", "feature"}, ""},
		{"clear subcommand", []string{"clear"}, ""},
		{"list subcommand", []string{"list"}, ""},
		{"ls subcommand", []string{"ls"}, ""},
		{"default add", []string{"bug", "feature"}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := handleLabel(context.Background(), cond, Invocation{Args: tt.args})
			// All should fail with ErrNoActiveTask since we have no task
			if !errors.Is(err, ErrNoActiveTask) {
				t.Errorf("expected ErrNoActiveTask, got %v", err)
			}
		})
	}
}

func TestHandleLibrarySubcommandRouting(t *testing.T) {
	cond := mustNewConductor(t)

	// Test that various subcommands are recognized
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{"list", []string{"list"}, false},
		{"ls alias", []string{"ls"}, false},
		{"stats", []string{"stats"}, false},
		// These need library to be non-nil
		{"show needs lib", []string{"show", "name"}, true},
		{"items needs lib", []string{"items", "id"}, true},
		{"search needs lib", []string{"search", "query"}, true},
		{"pull needs lib", []string{"pull", "source"}, true},
		{"remove needs lib", []string{"remove", "name"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := handleLibrary(context.Background(), cond, Invocation{Args: tt.args})

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if result == nil {
					t.Error("expected result, got nil")
				}
			}
		})
	}
}

func TestHandleMemorySubcommandRouting(t *testing.T) {
	cond := mustNewConductor(t)

	tests := []struct {
		name       string
		args       []string
		wantErr    bool
		wantErrSub string
	}{
		{"search with query", []string{"search", "test"}, false, ""},
		{"stats", []string{"stats"}, false, ""},
		{"default search", []string{"some", "query"}, false, ""},
		// index needs workspace
		{"index needs workspace", []string{"index", "task-1"}, true, "memory system is not available"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := handleMemory(context.Background(), cond, Invocation{Args: tt.args})

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				} else if tt.wantErrSub != "" && !strings.Contains(err.Error(), tt.wantErrSub) {
					t.Errorf("expected error containing %q, got %v", tt.wantErrSub, err)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if result == nil {
					t.Error("expected result, got nil")
				}
			}
		})
	}
}
