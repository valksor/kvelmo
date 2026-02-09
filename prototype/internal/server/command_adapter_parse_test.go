package server

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/valksor/go-mehrhof/internal/conductor/commands"
)

// --- Memory parse tests ---

func TestParseMemorySearchInvocation(t *testing.T) {
	tests := []struct {
		name      string
		query     url.Values
		wantErr   bool
		wantQuery string
		wantLimit int
	}{
		{
			name:    "missing q parameter",
			query:   url.Values{},
			wantErr: true,
		},
		{
			name:      "valid query only",
			query:     url.Values{"q": {"test query"}},
			wantQuery: "test query",
			wantLimit: 5, // default
		},
		{
			name:      "valid query with limit",
			query:     url.Values{"q": {"test query"}, "limit": {"10"}},
			wantQuery: "test query",
			wantLimit: 10,
		},
		{
			name:      "invalid limit uses default",
			query:     url.Values{"q": {"test query"}, "limit": {"invalid"}},
			wantQuery: "test query",
			wantLimit: 5,
		},
		{
			name:      "negative limit uses default",
			query:     url.Values{"q": {"test query"}, "limit": {"-5"}},
			wantQuery: "test query",
			wantLimit: 5,
		},
		{
			name:      "zero limit uses default",
			query:     url.Values{"q": {"test query"}, "limit": {"0"}},
			wantQuery: "test query",
			wantLimit: 5,
		},
		{
			name:      "with types parameter",
			query:     url.Values{"q": {"test"}, "types": {"code,spec"}},
			wantQuery: "test",
			wantLimit: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/memory/search?"+tt.query.Encode(), nil)

			inv, err := parseMemorySearchInvocation(req)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}

				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if inv.Source != commands.SourceAPI {
				t.Errorf("Source = %v, want %v", inv.Source, commands.SourceAPI)
			}
			if len(inv.Args) < 2 || inv.Args[0] != "search" || inv.Args[1] != tt.wantQuery {
				t.Errorf("Args = %v, want [search %s]", inv.Args, tt.wantQuery)
			}
			if limit, ok := inv.Options["limit"].(int); !ok || limit != tt.wantLimit {
				t.Errorf("Options[limit] = %v, want %d", inv.Options["limit"], tt.wantLimit)
			}
		})
	}
}

func TestParseMemoryIndexInvocation(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		body        string
		wantErr     bool
		wantTaskID  string
	}{
		{
			name:    "missing task_id json",
			body:    `{}`,
			wantErr: true,
		},
		{
			name:       "valid json",
			body:       `{"task_id":"task-123"}`,
			wantTaskID: "task-123",
		},
		{
			name:        "valid form data",
			contentType: "application/x-www-form-urlencoded",
			body:        "task_id=task-456",
			wantTaskID:  "task-456",
		},
		{
			name:        "empty form task_id",
			contentType: "application/x-www-form-urlencoded",
			body:        "task_id=",
			wantErr:     true,
		},
		{
			name:    "invalid json",
			body:    `{invalid`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/memory/index", strings.NewReader(tt.body))
			if tt.contentType != "" {
				req.Header.Set("Content-Type", tt.contentType)
			}

			inv, err := parseMemoryIndexInvocation(req)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}

				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if inv.Source != commands.SourceAPI {
				t.Errorf("Source = %v, want %v", inv.Source, commands.SourceAPI)
			}
			if len(inv.Args) < 2 || inv.Args[0] != "index" || inv.Args[1] != tt.wantTaskID {
				t.Errorf("Args = %v, want [index %s]", inv.Args, tt.wantTaskID)
			}
		})
	}
}

func TestParseMemoryStatsInvocation(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/memory/stats", nil)

	inv, err := parseMemoryStatsInvocation(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if inv.Source != commands.SourceAPI {
		t.Errorf("Source = %v, want %v", inv.Source, commands.SourceAPI)
	}
	if len(inv.Args) != 1 || inv.Args[0] != "stats" {
		t.Errorf("Args = %v, want [stats]", inv.Args)
	}
}

// --- Library parse tests ---

func TestParseLibraryListInvocation(t *testing.T) {
	tests := []struct {
		name            string
		query           url.Values
		wantSharedOnly  bool
		wantProjectOnly bool
		wantTag         string
	}{
		{
			name:  "no parameters",
			query: url.Values{},
		},
		{
			name:           "shared only",
			query:          url.Values{"shared": {"true"}},
			wantSharedOnly: true,
		},
		{
			name:            "project only",
			query:           url.Values{"project": {"true"}},
			wantProjectOnly: true,
		},
		{
			name:    "with tag",
			query:   url.Values{"tag": {"docs"}},
			wantTag: "docs",
		},
		{
			name:            "all parameters",
			query:           url.Values{"shared": {"true"}, "project": {"true"}, "tag": {"api"}},
			wantSharedOnly:  true,
			wantProjectOnly: true,
			wantTag:         "api",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/library?"+tt.query.Encode(), nil)

			inv, err := parseLibraryListInvocation(req)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if inv.Source != commands.SourceAPI {
				t.Errorf("Source = %v, want %v", inv.Source, commands.SourceAPI)
			}
			if len(inv.Args) != 1 || inv.Args[0] != "list" {
				t.Errorf("Args = %v, want [list]", inv.Args)
			}

			if shared, ok := inv.Options["shared_only"].(bool); !ok || shared != tt.wantSharedOnly {
				t.Errorf("Options[shared_only] = %v, want %v", inv.Options["shared_only"], tt.wantSharedOnly)
			}
			if project, ok := inv.Options["project_only"].(bool); !ok || project != tt.wantProjectOnly {
				t.Errorf("Options[project_only] = %v, want %v", inv.Options["project_only"], tt.wantProjectOnly)
			}
			if tag, ok := inv.Options["tag"].(string); !ok || tag != tt.wantTag {
				t.Errorf("Options[tag] = %v, want %q", inv.Options["tag"], tt.wantTag)
			}
		})
	}
}

func TestParseLibraryShowInvocation(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		wantErr  bool
		wantName string
	}{
		{
			name:    "empty name",
			path:    "/api/v1/library/",
			wantErr: true,
		},
		{
			name:     "valid name",
			path:     "/api/v1/library/my-collection",
			wantName: "my-collection",
		},
		{
			name:     "name with hyphen",
			path:     "/api/v1/library/react-docs",
			wantName: "react-docs",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)

			inv, err := parseLibraryShowInvocation(req)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}

				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(inv.Args) < 2 || inv.Args[0] != "show" || inv.Args[1] != tt.wantName {
				t.Errorf("Args = %v, want [show %s]", inv.Args, tt.wantName)
			}
		})
	}
}

func TestParseLibraryItemsInvocation(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		wantErr bool
	}{
		{
			name:    "missing id",
			id:      "",
			wantErr: true,
		},
		{
			name: "valid id",
			id:   "coll-123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/library/"+tt.id+"/items", nil)
			req.SetPathValue("id", tt.id)

			inv, err := parseLibraryItemsInvocation(req)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}

				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(inv.Args) < 2 || inv.Args[0] != "items" || inv.Args[1] != tt.id {
				t.Errorf("Args = %v, want [items %s]", inv.Args, tt.id)
			}
		})
	}
}

func TestParseLibraryRemoveInvocation(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		wantErr  bool
		wantName string
	}{
		{
			name:    "empty name",
			path:    "/api/v1/library/",
			wantErr: true,
		},
		{
			name:     "valid name",
			path:     "/api/v1/library/old-collection",
			wantName: "old-collection",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodDelete, tt.path, nil)

			inv, err := parseLibraryRemoveInvocation(req)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}

				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(inv.Args) < 2 || inv.Args[0] != "remove" || inv.Args[1] != tt.wantName {
				t.Errorf("Args = %v, want [remove %s]", inv.Args, tt.wantName)
			}
		})
	}
}

func TestParseLibraryStatsInvocation(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/library/stats", nil)

	inv, err := parseLibraryStatsInvocation(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(inv.Args) != 1 || inv.Args[0] != "stats" {
		t.Errorf("Args = %v, want [stats]", inv.Args)
	}
}

func TestParseLibraryPullInvocation(t *testing.T) {
	tests := []struct {
		name       string
		formData   url.Values
		wantErr    bool
		wantSource string
	}{
		{
			name:     "missing source",
			formData: url.Values{},
			wantErr:  true,
		},
		{
			name:       "valid source",
			formData:   url.Values{"source": {"https://docs.example.com"}},
			wantSource: "https://docs.example.com",
		},
		{
			name:       "source with paths",
			formData:   url.Values{"source": {"https://example.com"}, "paths": {"/docs,/api"}},
			wantSource: "https://example.com",
		},
		{
			name:       "source with whitespace trimmed",
			formData:   url.Values{"source": {"  https://example.com  "}},
			wantSource: "https://example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := bytes.NewBufferString(tt.formData.Encode())
			req := httptest.NewRequest(http.MethodPost, "/api/v1/library/pull", body)
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			inv, err := parseLibraryPullInvocation(req)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}

				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(inv.Args) < 2 || inv.Args[0] != "pull" || inv.Args[1] != tt.wantSource {
				t.Errorf("Args = %v, want [pull %s]", inv.Args, tt.wantSource)
			}
		})
	}
}

// --- Links parse tests ---

func TestParseLinksListInvocation(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/links", nil)

	inv, err := parseLinksListInvocation(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(inv.Args) != 1 || inv.Args[0] != "list" {
		t.Errorf("Args = %v, want [list]", inv.Args)
	}
}

func TestParseLinksEntityInvocation(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		wantErr bool
	}{
		{
			name:    "missing id",
			id:      "",
			wantErr: true,
		},
		{
			name: "valid id",
			id:   "spec:task-1:auth",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/links/"+tt.id, nil)
			req.SetPathValue("id", tt.id)

			inv, err := parseLinksEntityInvocation(req)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}

				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(inv.Args) != 1 || inv.Args[0] != tt.id {
				t.Errorf("Args = %v, want [%s]", inv.Args, tt.id)
			}
		})
	}
}

func TestParseLinksSearchInvocation(t *testing.T) {
	tests := []struct {
		name      string
		query     url.Values
		wantErr   bool
		wantQuery string
	}{
		{
			name:    "missing query",
			query:   url.Values{},
			wantErr: true,
		},
		{
			name:      "valid query",
			query:     url.Values{"q": {"auth"}},
			wantQuery: "auth",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/links/search?"+tt.query.Encode(), nil)

			inv, err := parseLinksSearchInvocation(req)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}

				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(inv.Args) < 2 || inv.Args[0] != "search" || inv.Args[1] != tt.wantQuery {
				t.Errorf("Args = %v, want [search %s]", inv.Args, tt.wantQuery)
			}
		})
	}
}

func TestParseLinksStatsInvocation(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/links/stats", nil)

	inv, err := parseLinksStatsInvocation(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(inv.Args) != 1 || inv.Args[0] != "stats" {
		t.Errorf("Args = %v, want [stats]", inv.Args)
	}
}

func TestParseLinksRebuildInvocation(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/links/rebuild", nil)

	inv, err := parseLinksRebuildInvocation(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(inv.Args) != 1 || inv.Args[0] != "rebuild" {
		t.Errorf("Args = %v, want [rebuild]", inv.Args)
	}
}

// --- Labels parse tests ---

func TestParseLabelsGetInvocation(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tasks/task-1/labels", nil)
	req.SetPathValue("id", "task-1")

	inv, err := parseLabelsGetInvocation(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// parseLabelsGetInvocation returns Args: []string{"list"} - doesn't use task_id from path
	if len(inv.Args) != 1 || inv.Args[0] != "list" {
		t.Errorf("Args = %v, want [list]", inv.Args)
	}
}

func TestParseLabelsPostInvocation(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		wantErr    bool
		wantAction string
	}{
		{
			name:    "invalid action",
			body:    `{"action":"invalid","labels":["bug"]}`,
			wantErr: true,
		},
		{
			name:       "add action",
			body:       `{"action":"add","labels":["bug","feature"]}`,
			wantAction: "add",
		},
		{
			name:       "remove action",
			body:       `{"action":"remove","labels":["old"]}`,
			wantAction: "remove",
		},
		{
			name:       "set action",
			body:       `{"action":"set","labels":["new"]}`,
			wantAction: "set",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks/task-1/labels", strings.NewReader(tt.body))
			req.SetPathValue("id", "task-1")

			inv, err := parseLabelsPostInvocation(req)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}

				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if inv.Args[0] != tt.wantAction {
				t.Errorf("Args[0] = %v, want %s", inv.Args[0], tt.wantAction)
			}
		})
	}
}

// --- Template parse tests ---

func TestParseTemplateListInvocation(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/templates", nil)

	inv, err := parseTemplateListInvocation(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(inv.Args) != 1 || inv.Args[0] != "list" {
		t.Errorf("Args = %v, want [list]", inv.Args)
	}
}

func TestParseTemplateGetInvocation(t *testing.T) {
	tests := []struct {
		name     string
		tmplName string
		wantErr  bool
		wantName string
	}{
		{
			name:     "missing name",
			tmplName: "",
			wantErr:  true,
		},
		{
			name:     "valid name",
			tmplName: "feature",
			wantName: "feature",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/templates/"+tt.tmplName, nil)
			req.SetPathValue("name", tt.tmplName) // Uses "name" not "id"

			inv, err := parseTemplateGetInvocation(req)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}

				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Returns Args: []string{"get"} and Options["name"] = name
			if len(inv.Args) != 1 || inv.Args[0] != "get" {
				t.Errorf("Args = %v, want [get]", inv.Args)
			}
			if inv.Options["name"] != tt.wantName {
				t.Errorf("Options[name] = %v, want %q", inv.Options["name"], tt.wantName)
			}
		})
	}
}

// --- Stack parse tests ---
// Note: parseStackSubcommand is a factory function that returns a parse function.
// Tests for this are in TestParseStackSubcommandFactory below.

// --- Agent alias parse tests ---

func TestParseAgentAliasListInvocation(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/agents/aliases", nil)

	inv, err := parseAgentAliasListInvocation(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Returns Options["subcommand"] = "list", no Args
	if inv.Options["subcommand"] != "list" {
		t.Errorf("Options[subcommand] = %v, want list", inv.Options["subcommand"])
	}
}

func TestParseAgentAliasDeleteInvocation(t *testing.T) {
	tests := []struct {
		name     string
		alias    string
		wantErr  bool
		wantName string
	}{
		{
			name:    "missing alias",
			alias:   "",
			wantErr: true,
		},
		{
			name:     "valid alias",
			alias:    "my-agent",
			wantName: "my-agent",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Uses URL path parsing: strings.TrimPrefix(r.URL.Path, "/api/v1/agents/aliases/")
			req := httptest.NewRequest(http.MethodDelete, "/api/v1/agents/aliases/"+tt.alias, nil)

			inv, err := parseAgentAliasDeleteInvocation(req)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}

				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Returns Options["subcommand"] = "delete", Options["name"] = name
			if inv.Options["subcommand"] != "delete" {
				t.Errorf("Options[subcommand] = %v, want delete", inv.Options["subcommand"])
			}
			if inv.Options["name"] != tt.wantName {
				t.Errorf("Options[name] = %v, want %q", inv.Options["name"], tt.wantName)
			}
		})
	}
}

// --- Settings parse tests ---

func TestParseSettingsGetInvocation(t *testing.T) {
	tests := []struct {
		name        string
		query       url.Values
		wantProject string
	}{
		{
			name:  "no project parameter",
			query: url.Values{},
		},
		{
			name:        "with project parameter",
			query:       url.Values{"project": {"my-project"}},
			wantProject: "my-project",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/settings?"+tt.query.Encode(), nil)

			inv, err := parseSettingsGetInvocation(req)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if project, ok := inv.Options["project"].(string); !ok || project != tt.wantProject {
				t.Errorf("Options[project] = %v, want %q", inv.Options["project"], tt.wantProject)
			}
		})
	}
}

// --- Library pull preview tests ---

func TestParseLibraryPullPreviewInvocation(t *testing.T) {
	tests := []struct {
		name       string
		formData   url.Values
		wantErr    bool
		wantSource string
		wantDryRun bool
	}{
		{
			name:     "missing source",
			formData: url.Values{},
			wantErr:  true,
		},
		{
			name:       "valid source",
			formData:   url.Values{"source": {"https://docs.example.com"}},
			wantSource: "https://docs.example.com",
			wantDryRun: true,
		},
		{
			name:       "source with options",
			formData:   url.Values{"source": {"https://example.com"}, "max_depth": {"3"}, "max_pages": {"50"}},
			wantSource: "https://example.com",
			wantDryRun: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := bytes.NewBufferString(tt.formData.Encode())
			req := httptest.NewRequest(http.MethodPost, "/api/v1/library/pull/preview", body)
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			inv, err := parseLibraryPullPreviewInvocation(req)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}

				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(inv.Args) < 2 || inv.Args[0] != "pull" || inv.Args[1] != tt.wantSource {
				t.Errorf("Args = %v, want [pull %s]", inv.Args, tt.wantSource)
			}
			if dryRun, ok := inv.Options["dry_run"].(bool); !ok || dryRun != tt.wantDryRun {
				t.Errorf("Options[dry_run] = %v, want %v", inv.Options["dry_run"], tt.wantDryRun)
			}
		})
	}
}

// --- Sync/Simplify tests ---

func TestParseSyncInvocation(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		wantErr    bool
		wantTaskID string
	}{
		{
			name:    "invalid json",
			body:    `{invalid`,
			wantErr: true,
		},
		{
			name:    "missing task_id",
			body:    `{}`,
			wantErr: true,
		},
		{
			name:    "empty task_id",
			body:    `{"task_id":""}`,
			wantErr: true,
		},
		{
			name:       "valid task_id",
			body:       `{"task_id":"task-123"}`,
			wantTaskID: "task-123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/sync", strings.NewReader(tt.body))

			inv, err := parseSyncInvocation(req)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}

				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(inv.Args) != 1 || inv.Args[0] != tt.wantTaskID {
				t.Errorf("Args = %v, want [%s]", inv.Args, tt.wantTaskID)
			}
			if taskID, ok := inv.Options["task_id"].(string); !ok || taskID != tt.wantTaskID {
				t.Errorf("Options[task_id] = %v, want %q", inv.Options["task_id"], tt.wantTaskID)
			}
		})
	}
}

func TestParseSimplifyInvocation(t *testing.T) {
	tests := []struct {
		name             string
		body             string
		wantErr          bool
		wantAgent        string
		wantNoCheckpoint bool
	}{
		{
			name: "empty body",
			body: "",
		},
		{
			name:    "invalid json",
			body:    `{invalid`,
			wantErr: true,
		},
		{
			name:      "with agent",
			body:      `{"agent":"claude-sonnet"}`,
			wantAgent: "claude-sonnet",
		},
		{
			name:             "with no_checkpoint",
			body:             `{"no_checkpoint":true}`,
			wantNoCheckpoint: true,
		},
		{
			name:             "all options",
			body:             `{"agent":"opus","no_checkpoint":true}`,
			wantAgent:        "opus",
			wantNoCheckpoint: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/simplify", strings.NewReader(tt.body))

			inv, err := parseSimplifyInvocation(req)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}

				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if agent, ok := inv.Options["agent"].(string); !ok || agent != tt.wantAgent {
				t.Errorf("Options[agent] = %v, want %q", inv.Options["agent"], tt.wantAgent)
			}
			if noChkpt, ok := inv.Options["no_checkpoint"].(bool); !ok || noChkpt != tt.wantNoCheckpoint {
				t.Errorf("Options[no_checkpoint] = %v, want %v", inv.Options["no_checkpoint"], tt.wantNoCheckpoint)
			}
		})
	}
}

// --- Specification diff tests ---

func TestParseSpecificationDiffInvocation(t *testing.T) {
	tests := []struct {
		name        string
		taskID      string
		specNumber  string
		query       url.Values
		wantErr     bool
		errContains string
	}{
		{
			name:        "missing task ID",
			taskID:      "",
			specNumber:  "1",
			query:       url.Values{"file": {"main.go"}},
			wantErr:     true,
			errContains: "task ID is required",
		},
		{
			name:        "missing spec number",
			taskID:      "task-1",
			specNumber:  "",
			query:       url.Values{"file": {"main.go"}},
			wantErr:     true,
			errContains: "specification number is required",
		},
		{
			name:        "invalid spec number",
			taskID:      "task-1",
			specNumber:  "invalid",
			query:       url.Values{"file": {"main.go"}},
			wantErr:     true,
			errContains: "specification number must be a positive integer",
		},
		{
			name:        "zero spec number",
			taskID:      "task-1",
			specNumber:  "0",
			query:       url.Values{"file": {"main.go"}},
			wantErr:     true,
			errContains: "specification number must be a positive integer",
		},
		{
			name:        "missing file parameter",
			taskID:      "task-1",
			specNumber:  "1",
			query:       url.Values{},
			wantErr:     true,
			errContains: "file query parameter is required",
		},
		{
			name:       "valid request",
			taskID:     "task-1",
			specNumber: "2",
			query:      url.Values{"file": {"main.go"}},
		},
		{
			name:        "invalid context parameter",
			taskID:      "task-1",
			specNumber:  "1",
			query:       url.Values{"file": {"main.go"}, "context": {"invalid"}},
			wantErr:     true,
			errContains: "context must be a non-negative integer",
		},
		{
			name:       "valid context parameter",
			taskID:     "task-1",
			specNumber: "1",
			query:      url.Values{"file": {"main.go"}, "context": {"5"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/tasks/"+tt.taskID+"/specifications/"+tt.specNumber+"/diff?"+tt.query.Encode(), nil)
			req.SetPathValue("id", tt.taskID)
			req.SetPathValue("number", tt.specNumber)

			inv, err := parseSpecificationDiffInvocation(req)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("error = %v, want containing %q", err, tt.errContains)
				}

				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if inv.Options["task_id"] != tt.taskID {
				t.Errorf("Options[task_id] = %v, want %q", inv.Options["task_id"], tt.taskID)
			}
		})
	}
}

// --- Security scan tests ---

func TestParseSecurityScanInvocation(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		body        string
		wantDir     string
	}{
		{
			name:        "json with dir",
			contentType: "application/json",
			body:        `{"dir":"/path/to/scan"}`,
			wantDir:     "/path/to/scan",
		},
		{
			name:        "json with scanners",
			contentType: "application/json",
			body:        `{"scanners":["gosec","gitleaks"]}`,
		},
		{
			name:        "form with sast scanner",
			contentType: "application/x-www-form-urlencoded",
			body:        "scanners=sast",
		},
		{
			name:        "form with secrets scanner",
			contentType: "application/x-www-form-urlencoded",
			body:        "scanners=secrets",
		},
		{
			name:        "form with vulns scanner",
			contentType: "application/x-www-form-urlencoded",
			body:        "scanners=vulns",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/security/scan", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", tt.contentType)

			inv, err := parseSecurityScanInvocation(req)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.wantDir != "" {
				if dir, ok := inv.Options["dir"].(string); !ok || dir != tt.wantDir {
					t.Errorf("Options[dir] = %v, want %q", inv.Options["dir"], tt.wantDir)
				}
			}
		})
	}
}

// --- Config explain tests ---

func TestParseConfigExplainInvocation(t *testing.T) {
	tests := []struct {
		name     string
		query    url.Values
		wantErr  bool
		wantStep string
	}{
		{
			name:    "missing step",
			query:   url.Values{},
			wantErr: true,
		},
		{
			name:     "planning step",
			query:    url.Values{"step": {"planning"}},
			wantStep: "planning",
		},
		{
			name:     "implementing step",
			query:    url.Values{"step": {"implementing"}},
			wantStep: "implementing",
		},
		{
			name:     "reviewing step",
			query:    url.Values{"step": {"reviewing"}},
			wantStep: "reviewing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/config/explain?"+tt.query.Encode(), nil)

			inv, err := parseConfigExplainInvocation(req)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}

				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if step, ok := inv.Options["step"].(string); !ok || step != tt.wantStep {
				t.Errorf("Options[step] = %v, want %q", inv.Options["step"], tt.wantStep)
			}
		})
	}
}

// --- Quick task tests ---

func TestParseQuickCreateInvocation(t *testing.T) {
	tests := []struct {
		name            string
		contentType     string
		body            string
		wantErr         bool
		wantDescription string
		wantTitle       string
	}{
		{
			name:    "missing description json",
			body:    `{}`,
			wantErr: true,
		},
		{
			name:            "valid json",
			body:            `{"description":"Fix bug in login"}`,
			wantDescription: "Fix bug in login",
		},
		{
			name:            "json with title",
			body:            `{"description":"Fix bug","title":"Login Bug"}`,
			wantDescription: "Fix bug",
			wantTitle:       "Login Bug",
		},
		{
			name:            "form data",
			contentType:     "application/x-www-form-urlencoded",
			body:            "description=Add+feature",
			wantDescription: "Add feature",
		},
		{
			name:        "form data missing description",
			contentType: "application/x-www-form-urlencoded",
			body:        "title=something",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/quick-tasks", strings.NewReader(tt.body))
			if tt.contentType != "" {
				req.Header.Set("Content-Type", tt.contentType)
			}

			inv, err := parseQuickCreateInvocation(req)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}

				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(inv.Args) != 1 || inv.Args[0] != tt.wantDescription {
				t.Errorf("Args = %v, want [%s]", inv.Args, tt.wantDescription)
			}
			if title, ok := inv.Options["title"].(string); !ok || title != tt.wantTitle {
				t.Errorf("Options[title] = %v, want %q", inv.Options["title"], tt.wantTitle)
			}
		})
	}
}

func TestParseQuickTaskIDInvocation(t *testing.T) {
	tests := []struct {
		name    string
		taskID  string
		wantErr bool
	}{
		{
			name:    "missing task ID",
			taskID:  "",
			wantErr: true,
		},
		{
			name:   "valid task ID",
			taskID: "task-123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/quick-tasks/"+tt.taskID, nil)
			req.SetPathValue("taskId", tt.taskID)

			inv, err := parseQuickTaskIDInvocation(req)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}

				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if taskID, ok := inv.Options["task_id"].(string); !ok || taskID != tt.taskID {
				t.Errorf("Options[task_id] = %v, want %q", inv.Options["task_id"], tt.taskID)
			}
		})
	}
}

func TestParseQuickNoteInvocation(t *testing.T) {
	tests := []struct {
		name    string
		taskID  string
		body    string
		wantErr bool
	}{
		{
			name:    "missing task ID",
			taskID:  "",
			body:    `{"note":"test"}`,
			wantErr: true,
		},
		{
			name:    "missing note",
			taskID:  "task-1",
			body:    `{}`,
			wantErr: true,
		},
		{
			name:   "valid request",
			taskID: "task-1",
			body:   `{"note":"Important note"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/quick-tasks/"+tt.taskID+"/notes", strings.NewReader(tt.body))
			req.SetPathValue("taskId", tt.taskID)

			inv, err := parseQuickNoteInvocation(req)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}

				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if taskID, ok := inv.Options["task_id"].(string); !ok || taskID != tt.taskID {
				t.Errorf("Options[task_id] = %v, want %q", inv.Options["task_id"], tt.taskID)
			}
		})
	}
}

func TestParseQuickOptimizeInvocation(t *testing.T) {
	tests := []struct {
		name      string
		taskID    string
		body      string
		wantErr   bool
		wantAgent string
	}{
		{
			name:    "missing task ID",
			taskID:  "",
			wantErr: true,
		},
		{
			name:   "valid without body",
			taskID: "task-1",
		},
		{
			name:      "valid with agent",
			taskID:    "task-1",
			body:      `{"agent":"claude-opus"}`,
			wantAgent: "claude-opus",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var bodyReader *strings.Reader
			if tt.body != "" {
				bodyReader = strings.NewReader(tt.body)
			} else {
				bodyReader = strings.NewReader("")
			}
			req := httptest.NewRequest(http.MethodPost, "/api/v1/quick-tasks/"+tt.taskID+"/optimize", bodyReader)
			req.SetPathValue("taskId", tt.taskID)
			if tt.body != "" {
				req.ContentLength = int64(len(tt.body))
			}

			inv, err := parseQuickOptimizeInvocation(req)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}

				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if agent, ok := inv.Options["agent"].(string); !ok || agent != tt.wantAgent {
				t.Errorf("Options[agent] = %v, want %q", inv.Options["agent"], tt.wantAgent)
			}
		})
	}
}

func TestParseQuickExportInvocation(t *testing.T) {
	tests := []struct {
		name    string
		taskID  string
		wantErr bool
	}{
		{
			name:    "missing task ID",
			taskID:  "",
			wantErr: true,
		},
		{
			name:   "valid task ID",
			taskID: "task-1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/quick-tasks/"+tt.taskID+"/export", nil)
			req.SetPathValue("taskId", tt.taskID)

			inv, err := parseQuickExportInvocation(req)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}

				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(inv.Args) != 1 || inv.Args[0] != "quick-tasks/"+tt.taskID {
				t.Errorf("Args = %v, want [quick-tasks/%s]", inv.Args, tt.taskID)
			}
		})
	}
}

func TestParseQuickSubmitInvocation(t *testing.T) {
	tests := []struct {
		name    string
		taskID  string
		body    string
		wantErr bool
	}{
		{
			name:    "missing task ID",
			taskID:  "",
			body:    `{"provider":"github"}`,
			wantErr: true,
		},
		{
			name:    "missing provider",
			taskID:  "task-1",
			body:    `{}`,
			wantErr: true,
		},
		{
			name:   "valid request",
			taskID: "task-1",
			body:   `{"provider":"github"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/quick-tasks/"+tt.taskID+"/submit", strings.NewReader(tt.body))
			req.SetPathValue("taskId", tt.taskID)

			inv, err := parseQuickSubmitInvocation(req)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}

				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(inv.Args) < 2 || inv.Args[0] != "quick-tasks/"+tt.taskID {
				t.Errorf("Args[0] = %v, want quick-tasks/%s", inv.Args[0], tt.taskID)
			}
		})
	}
}

func TestParseQuickStartInvocation(t *testing.T) {
	tests := []struct {
		name    string
		taskID  string
		wantErr bool
	}{
		{
			name:    "missing task ID",
			taskID:  "",
			wantErr: true,
		},
		{
			name:   "valid task ID",
			taskID: "task-1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/quick-tasks/"+tt.taskID+"/start", nil)
			req.SetPathValue("taskId", tt.taskID)

			inv, err := parseQuickStartInvocation(req)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}

				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			expectedRef := "queue:quick-tasks/" + tt.taskID
			if inv.Options["ref"] != expectedRef {
				t.Errorf("Options[ref] = %v, want %q", inv.Options["ref"], expectedRef)
			}
		})
	}
}

func TestParseQuickDeleteInvocation(t *testing.T) {
	tests := []struct {
		name    string
		taskID  string
		wantErr bool
	}{
		{
			name:    "missing task ID",
			taskID:  "",
			wantErr: true,
		},
		{
			name:   "valid task ID",
			taskID: "task-1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodDelete, "/api/v1/quick-tasks/"+tt.taskID, nil)
			req.SetPathValue("taskId", tt.taskID)

			inv, err := parseQuickDeleteInvocation(req)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}

				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(inv.Args) != 1 || inv.Args[0] != "quick-tasks/"+tt.taskID {
				t.Errorf("Args = %v, want [quick-tasks/%s]", inv.Args, tt.taskID)
			}
		})
	}
}

// --- Submit source tests ---

func TestParseSubmitSourceInvocation(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		wantErr bool
	}{
		{
			name:    "missing source",
			body:    `{"provider":"github"}`,
			wantErr: true,
		},
		{
			name:    "missing provider",
			body:    `{"source":"github:123"}`,
			wantErr: true,
		},
		{
			name: "valid request",
			body: `{"source":"github:123","provider":"linear"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/submit-source", strings.NewReader(tt.body))

			inv, err := parseSubmitSourceInvocation(req)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}

				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if inv.Options["source"] == nil {
				t.Error("Options[source] should not be nil")
			}
		})
	}
}

// --- Commit parse tests ---

func TestParseCommitChangesInvocation(t *testing.T) {
	tests := []struct {
		name               string
		query              url.Values
		wantIncludeUnstage bool
	}{
		{
			name:  "no parameters",
			query: url.Values{},
		},
		{
			name:               "include_unstaged true",
			query:              url.Values{"include_unstaged": {"true"}},
			wantIncludeUnstage: true,
		},
		{
			name:  "include_unstaged false",
			query: url.Values{"include_unstaged": {"false"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/commit/changes?"+tt.query.Encode(), nil)

			inv, err := parseCommitChangesInvocation(req)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(inv.Args) != 1 || inv.Args[0] != "changes" {
				t.Errorf("Args = %v, want [changes]", inv.Args)
			}
			if unstaged, ok := inv.Options["include_unstaged"].(bool); !ok || unstaged != tt.wantIncludeUnstage {
				t.Errorf("Options[include_unstaged] = %v, want %v", inv.Options["include_unstaged"], tt.wantIncludeUnstage)
			}
		})
	}
}

func TestParseCommitPlanInvocation(t *testing.T) {
	tests := []struct {
		name               string
		query              url.Values
		wantIncludeUnstage bool
	}{
		{
			name:  "no parameters",
			query: url.Values{},
		},
		{
			name:               "all true",
			query:              url.Values{"all": {"true"}},
			wantIncludeUnstage: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/commit/plan?"+tt.query.Encode(), nil)

			inv, err := parseCommitPlanInvocation(req)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(inv.Args) != 1 || inv.Args[0] != "plan" {
				t.Errorf("Args = %v, want [plan]", inv.Args)
			}
		})
	}
}

func TestParseCommitExecuteInvocation(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		wantErr bool
	}{
		{
			name:    "invalid json",
			body:    `{invalid`,
			wantErr: true,
		},
		{
			name: "valid request",
			body: `{"groups":[{"message":"feat: add feature","files":["main.go"]}]}`,
		},
		{
			name: "with push",
			body: `{"groups":[{"message":"fix: bug","files":["fix.go"]}],"push":true}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/commit/execute", strings.NewReader(tt.body))

			inv, err := parseCommitExecuteInvocation(req)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}

				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(inv.Args) != 1 || inv.Args[0] != "execute" {
				t.Errorf("Args = %v, want [execute]", inv.Args)
			}
		})
	}
}

// --- Stack subcommand factory tests ---

func TestParseStackSubcommandFactory(t *testing.T) {
	tests := []struct {
		name       string
		subcommand string
	}{
		{"list", "list"},
		{"sync", "sync"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parseFn := parseStackSubcommand(tt.subcommand)
			req := httptest.NewRequest(http.MethodGet, "/api/v1/stack", nil)

			inv, err := parseFn(req)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if inv.Options["subcommand"] != tt.subcommand {
				t.Errorf("Options[subcommand] = %v, want %q", inv.Options["subcommand"], tt.subcommand)
			}
		})
	}
}

func TestParseStackRebasePreviewInvocation(t *testing.T) {
	tests := []struct {
		name           string
		query          url.Values
		wantStackID    string
		wantTaskID     string
		wantPreviewAll bool
	}{
		{
			name:           "no parameters - preview all",
			query:          url.Values{},
			wantPreviewAll: true,
		},
		{
			name:        "with stack_id",
			query:       url.Values{"stack_id": {"stack-1"}},
			wantStackID: "stack-1",
		},
		{
			name:       "with task_id",
			query:      url.Values{"task_id": {"task-1"}},
			wantTaskID: "task-1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/stack/rebase/preview?"+tt.query.Encode(), nil)

			inv, err := parseStackRebasePreviewInvocation(req)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if inv.Options["subcommand"] != "rebase-preview" {
				t.Errorf("Options[subcommand] = %v, want rebase-preview", inv.Options["subcommand"])
			}

			if tt.wantStackID != "" {
				if inv.Options["stack_id"] != tt.wantStackID {
					t.Errorf("Options[stack_id] = %v, want %q", inv.Options["stack_id"], tt.wantStackID)
				}
			}
			if tt.wantTaskID != "" {
				if inv.Options["task_id"] != tt.wantTaskID {
					t.Errorf("Options[task_id] = %v, want %q", inv.Options["task_id"], tt.wantTaskID)
				}
			}
			if tt.wantPreviewAll {
				if inv.Options["preview_all"] != true {
					t.Errorf("Options[preview_all] = %v, want true", inv.Options["preview_all"])
				}
			}
		})
	}
}

func TestParseStackRebaseInvocation(t *testing.T) {
	tests := []struct {
		name          string
		body          string
		wantRebaseAll bool
		wantTaskID    string
		wantStackID   string
	}{
		{
			name:          "empty body - rebase all",
			body:          "",
			wantRebaseAll: true,
		},
		{
			name:       "with task_id",
			body:       `{"task_id":"task-1"}`,
			wantTaskID: "task-1",
		},
		{
			name:        "with stack_id",
			body:        `{"stack_id":"stack-1"}`,
			wantStackID: "stack-1",
		},
		{
			name:          "empty json - rebase all",
			body:          `{}`,
			wantRebaseAll: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var bodyReader *strings.Reader
			if tt.body != "" {
				bodyReader = strings.NewReader(tt.body)
			} else {
				bodyReader = strings.NewReader("")
			}
			req := httptest.NewRequest(http.MethodPost, "/api/v1/stack/rebase", bodyReader)
			if tt.body != "" {
				req.ContentLength = int64(len(tt.body))
			}

			inv, err := parseStackRebaseInvocation(req)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if inv.Options["subcommand"] != "rebase" {
				t.Errorf("Options[subcommand] = %v, want rebase", inv.Options["subcommand"])
			}

			if tt.wantRebaseAll {
				if inv.Options["rebase_all"] != true {
					t.Errorf("Options[rebase_all] = %v, want true", inv.Options["rebase_all"])
				}
			}
			if tt.wantTaskID != "" {
				if inv.Options["task_id"] != tt.wantTaskID {
					t.Errorf("Options[task_id] = %v, want %q", inv.Options["task_id"], tt.wantTaskID)
				}
			}
			if tt.wantStackID != "" {
				if inv.Options["stack_id"] != tt.wantStackID {
					t.Errorf("Options[stack_id] = %v, want %q", inv.Options["stack_id"], tt.wantStackID)
				}
			}
		})
	}
}

// --- Project parse tests ---

func TestParseProjectSubcommandFactory(t *testing.T) {
	tests := []struct {
		name       string
		subcommand string
	}{
		{"list", "list"},
		{"status", "status"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parseFn := parseProjectSubcommand(tt.subcommand)
			req := httptest.NewRequest(http.MethodGet, "/api/v1/project", nil)

			inv, err := parseFn(req)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(inv.Args) != 1 || inv.Args[0] != tt.subcommand {
				t.Errorf("Args = %v, want [%s]", inv.Args, tt.subcommand)
			}
		})
	}
}

func TestParseProjectPlanInvocation(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		wantErr bool
	}{
		{
			name:    "invalid json",
			body:    `{invalid`,
			wantErr: true,
		},
		{
			name:    "missing source",
			body:    `{}`,
			wantErr: true,
		},
		{
			name: "valid source",
			body: `{"source":"github:123"}`,
		},
		{
			name: "with all options",
			body: `{"source":"file:task.md","title":"New Feature","instructions":"Follow TDD","use_schema":true}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/project/plan", strings.NewReader(tt.body))

			inv, err := parseProjectPlanInvocation(req)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}

				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(inv.Args) < 2 || inv.Args[0] != "plan" {
				t.Errorf("Args[0] = %v, want plan", inv.Args[0])
			}
		})
	}
}

func TestParseProjectQueueInvocation(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
		wantID  string
	}{
		{
			name:    "missing queue ID",
			path:    "/api/v1/project/queue/",
			wantErr: true,
		},
		{
			name:   "valid queue ID",
			path:   "/api/v1/project/queue/backlog",
			wantID: "backlog",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)

			inv, err := parseProjectQueueInvocation(req)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}

				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(inv.Args) < 2 || inv.Args[0] != "queue" || inv.Args[1] != tt.wantID {
				t.Errorf("Args = %v, want [queue %s]", inv.Args, tt.wantID)
			}
		})
	}
}

func TestParseProjectQueueDeleteInvocation(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
		wantID  string
	}{
		{
			name:    "missing queue ID",
			path:    "/api/v1/project/queue/",
			wantErr: true,
		},
		{
			name:   "valid queue ID",
			path:   "/api/v1/project/queue/backlog",
			wantID: "backlog",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodDelete, tt.path, nil)

			inv, err := parseProjectQueueDeleteInvocation(req)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}

				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(inv.Args) < 2 || inv.Args[0] != "queue-delete" || inv.Args[1] != tt.wantID {
				t.Errorf("Args = %v, want [queue-delete %s]", inv.Args, tt.wantID)
			}
		})
	}
}

func TestParseProjectTasksInvocation(t *testing.T) {
	tests := []struct {
		name        string
		query       url.Values
		wantQueueID string
		wantStatus  string
	}{
		{
			name:  "no parameters",
			query: url.Values{},
		},
		{
			name:        "with queue_id",
			query:       url.Values{"queue_id": {"backlog"}},
			wantQueueID: "backlog",
		},
		{
			name:       "with status",
			query:      url.Values{"status": {"pending"}},
			wantStatus: "pending",
		},
		{
			name:        "with both",
			query:       url.Values{"queue_id": {"sprint"}, "status": {"active"}},
			wantQueueID: "sprint",
			wantStatus:  "active",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/project/tasks?"+tt.query.Encode(), nil)

			inv, err := parseProjectTasksInvocation(req)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(inv.Args) != 1 || inv.Args[0] != "tasks" {
				t.Errorf("Args = %v, want [tasks]", inv.Args)
			}
			if queueID, ok := inv.Options["queue_id"].(string); !ok || queueID != tt.wantQueueID {
				t.Errorf("Options[queue_id] = %v, want %q", inv.Options["queue_id"], tt.wantQueueID)
			}
			if status, ok := inv.Options["status"].(string); !ok || status != tt.wantStatus {
				t.Errorf("Options[status] = %v, want %q", inv.Options["status"], tt.wantStatus)
			}
		})
	}
}

func TestParseProjectTaskEditInvocation(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		body    string
		query   url.Values
		wantErr bool
	}{
		{
			name:    "missing task ID",
			path:    "/api/v1/project/tasks/",
			body:    `{"title":"New title"}`,
			wantErr: true,
		},
		{
			name:    "invalid json",
			path:    "/api/v1/project/tasks/task-1",
			body:    `{invalid`,
			wantErr: true,
		},
		{
			name: "valid request",
			path: "/api/v1/project/tasks/task-1",
			body: `{"title":"Updated title"}`,
		},
		{
			name:  "with queue_id",
			path:  "/api/v1/project/tasks/task-1",
			body:  `{"title":"Updated"}`,
			query: url.Values{"queue_id": {"sprint"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqURL := tt.path
			if tt.query != nil {
				reqURL += "?" + tt.query.Encode()
			}
			req := httptest.NewRequest(http.MethodPatch, reqURL, strings.NewReader(tt.body))

			inv, err := parseProjectTaskEditInvocation(req)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}

				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(inv.Args) < 2 || inv.Args[0] != "task-edit" {
				t.Errorf("Args[0] = %v, want task-edit", inv.Args[0])
			}
		})
	}
}

func TestParseProjectReorderInvocation(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		wantErr bool
	}{
		{
			name:    "invalid json",
			body:    `{invalid`,
			wantErr: true,
		},
		{
			name: "auto reorder",
			body: `{"auto":true}`,
		},
		{
			name: "manual reorder",
			body: `{"task_id":"task-1","position":"before","reference_id":"task-2"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/project/reorder", strings.NewReader(tt.body))

			inv, err := parseProjectReorderInvocation(req)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}

				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(inv.Args) != 1 || inv.Args[0] != "reorder" {
				t.Errorf("Args = %v, want [reorder]", inv.Args)
			}
		})
	}
}

func TestParseProjectSubmitInvocation(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		wantErr bool
	}{
		{
			name:    "missing provider",
			body:    `{}`,
			wantErr: true,
		},
		{
			name: "valid request",
			body: `{"provider":"github"}`,
		},
		{
			name: "with all options",
			body: `{"provider":"linear","queue_id":"sprint","create_epic":true,"labels":["bug"],"dry_run":true}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/project/submit", strings.NewReader(tt.body))

			inv, err := parseProjectSubmitInvocation(req)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}

				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(inv.Args) < 2 || inv.Args[0] != "submit" {
				t.Errorf("Args[0] = %v, want submit", inv.Args[0])
			}
		})
	}
}

func TestParseProjectStartInvocation(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		wantErr bool
	}{
		{
			name:    "invalid json",
			body:    `{invalid`,
			wantErr: true,
		},
		{
			name: "empty body",
			body: `{}`,
		},
		{
			name: "with queue and task",
			body: `{"queue_id":"sprint","task_id":"task-1"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/project/start", strings.NewReader(tt.body))

			inv, err := parseProjectStartInvocation(req)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}

				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(inv.Args) != 1 || inv.Args[0] != "start" {
				t.Errorf("Args = %v, want [start]", inv.Args)
			}
		})
	}
}

// --- Template apply tests ---

func TestParseTemplateApplyInvocation(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		wantErr bool
	}{
		{
			name:    "invalid json",
			body:    `{invalid`,
			wantErr: true,
		},
		{
			name:    "missing template_name",
			body:    `{"file_path":"/path/to/file"}`,
			wantErr: true,
		},
		{
			name:    "missing file_path",
			body:    `{"template_name":"feature"}`,
			wantErr: true,
		},
		{
			name: "valid request",
			body: `{"template_name":"feature","file_path":"/path/to/file.md"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/templates/apply", strings.NewReader(tt.body))

			inv, err := parseTemplateApplyInvocation(req)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}

				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(inv.Args) != 1 || inv.Args[0] != "apply" {
				t.Errorf("Args = %v, want [apply]", inv.Args)
			}
		})
	}
}

// --- Interactive answer tests ---

func TestParseInteractiveAnswerInvocation(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		wantErr bool
	}{
		{
			name:    "invalid json",
			body:    `{invalid`,
			wantErr: true,
		},
		{
			name:    "missing response",
			body:    `{}`,
			wantErr: true,
		},
		{
			name:    "empty response",
			body:    `{"response":""}`,
			wantErr: true,
		},
		{
			name: "valid response",
			body: `{"response":"Yes, proceed"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/interactive/answer", strings.NewReader(tt.body))

			inv, err := parseInteractiveAnswerInvocation(req)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}

				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if inv.Options["response"] == nil {
				t.Error("Options[response] should not be nil")
			}
		})
	}
}

// --- Running task cancel tests ---

func TestParseRunningCancelInvocation(t *testing.T) {
	tests := []struct {
		name    string
		taskID  string
		wantErr bool
	}{
		{
			name:    "missing task ID",
			taskID:  "",
			wantErr: true,
		},
		{
			name:   "valid task ID",
			taskID: "task-123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/running/"+tt.taskID+"/cancel", nil)
			req.SetPathValue("id", tt.taskID)

			inv, err := parseRunningCancelInvocation(req)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}

				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if inv.Options["task_id"] != tt.taskID {
				t.Errorf("Options[task_id] = %v, want %q", inv.Options["task_id"], tt.taskID)
			}
		})
	}
}

// --- Agent logs history tests ---

func TestParseAgentLogsHistoryInvocation(t *testing.T) {
	tests := []struct {
		name       string
		query      url.Values
		wantTaskID string
	}{
		{
			name:  "no task_id",
			query: url.Values{},
		},
		{
			name:       "with task_id",
			query:      url.Values{"task_id": {"task-123"}},
			wantTaskID: "task-123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/agent-logs/history?"+tt.query.Encode(), nil)

			inv, err := parseAgentLogsHistoryInvocation(req)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if taskID, ok := inv.Options["task_id"].(string); !ok || taskID != tt.wantTaskID {
				t.Errorf("Options[task_id] = %v, want %q", inv.Options["task_id"], tt.wantTaskID)
			}
		})
	}
}

// --- Standalone review/simplify tests ---

func TestParseStandaloneReviewInvocation(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		wantErr bool
	}{
		{
			name:    "invalid json",
			body:    `{invalid`,
			wantErr: true,
		},
		{
			name: "empty body",
			body: `{}`,
		},
		{
			name: "with options",
			body: `{"agent":"claude-opus","paths":["src/"]}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/standalone/review", strings.NewReader(tt.body))

			inv, err := parseStandaloneReviewInvocation(req)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}

				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if inv.Source != commands.SourceAPI {
				t.Errorf("Source = %v, want %v", inv.Source, commands.SourceAPI)
			}
		})
	}
}

func TestParseStandaloneSimplifyInvocation(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		wantErr bool
	}{
		{
			name:    "invalid json",
			body:    `{invalid`,
			wantErr: true,
		},
		{
			name: "empty body",
			body: `{}`,
		},
		{
			name: "with options",
			body: `{"agent":"claude-sonnet","no_checkpoint":true}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/standalone/simplify", strings.NewReader(tt.body))

			inv, err := parseStandaloneSimplifyInvocation(req)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}

				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if inv.Source != commands.SourceAPI {
				t.Errorf("Source = %v, want %v", inv.Source, commands.SourceAPI)
			}
		})
	}
}

// --- Filesystem browse tests ---

func TestParseFSBrowseInvocation(t *testing.T) {
	tests := []struct {
		name     string
		query    url.Values
		wantPath string
	}{
		{
			name:  "no path parameter",
			query: url.Values{},
		},
		{
			name:     "with path",
			query:    url.Values{"path": {"/home/user/projects"}},
			wantPath: "/home/user/projects",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/fs/browse?"+tt.query.Encode(), nil)

			inv, err := parseFSBrowseInvocation(req)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if path, ok := inv.Options["path"].(string); !ok || path != tt.wantPath {
				t.Errorf("Options[path] = %v, want %q", inv.Options["path"], tt.wantPath)
			}
		})
	}
}

// --- Project path tests ---

func TestParseProjectPathInvocation(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		wantErr bool
	}{
		{
			name:    "invalid json",
			body:    `{invalid`,
			wantErr: true,
		},
		{
			name:    "missing path",
			body:    `{}`,
			wantErr: true,
		},
		{
			name:    "empty path",
			body:    `{"path":""}`,
			wantErr: true,
		},
		{
			name: "valid path",
			body: `{"path":"/home/user/project"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/project/path", strings.NewReader(tt.body))

			inv, err := parseProjectPathInvocation(req)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}

				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if inv.Options["path"] == nil {
				t.Error("Options[path] should not be nil")
			}
		})
	}
}

// --- Agent alias add tests ---

func TestParseAgentAliasAddInvocation(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		wantErr bool
	}{
		{
			name:    "invalid json",
			body:    `{invalid`,
			wantErr: true,
		},
		{
			name: "valid request",
			body: `{"name":"my-agent","extends":"claude","description":"Custom agent"}`,
		},
		{
			name: "with components",
			body: `{"name":"combined","extends":"claude","components":["planner","reviewer"]}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/agents/aliases", strings.NewReader(tt.body))

			inv, err := parseAgentAliasAddInvocation(req)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}

				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if inv.Options["subcommand"] != "add" {
				t.Errorf("Options[subcommand] = %v, want add", inv.Options["subcommand"])
			}
		})
	}
}
