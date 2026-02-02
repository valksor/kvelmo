package conductor

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/valksor/go-mehrhof/internal/provider"
	"github.com/valksor/go-mehrhof/internal/storage"
)

func TestCodeDir_WithWorkspace(t *testing.T) {
	// Create temporary directory structure
	tmpDir := t.TempDir()
	codeDir := filepath.Join(tmpDir, "code")
	if err := os.MkdirAll(codeDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create .mehrhof directory for workspace
	if err := os.MkdirAll(filepath.Join(tmpDir, ".mehrhof"), 0o755); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name     string
		codeDir  string // code_dir config value
		wantDir  string // expected CodeDir() result
		wantRoot string // expected Root() (should be hub root always)
	}{
		{
			name:     "no code_dir configured",
			codeDir:  "",
			wantDir:  tmpDir,
			wantRoot: tmpDir,
		},
		{
			name:     "code_dir set to absolute path",
			codeDir:  codeDir,
			wantDir:  codeDir,
			wantRoot: tmpDir,
		},
		{
			name:     "code_dir set to relative path",
			codeDir:  "code",
			wantDir:  codeDir,
			wantRoot: tmpDir,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create config with code_dir and open workspace with it
			ctx := context.Background()
			cfg := storage.NewDefaultWorkspaceConfig()
			cfg.Project.CodeDir = tt.codeDir

			ws, err := storage.OpenWorkspace(ctx, tmpDir, cfg)
			if err != nil {
				t.Fatalf("OpenWorkspace: %v", err)
			}

			c := &Conductor{
				workspace: ws,
				opts:      Options{WorkDir: tmpDir},
			}

			got := c.CodeDir()
			if got != tt.wantDir {
				t.Errorf("CodeDir() = %q, want %q", got, tt.wantDir)
			}

			// Root should always be the hub root
			if ws.Root() != tt.wantRoot {
				t.Errorf("Root() = %q, want %q", ws.Root(), tt.wantRoot)
			}
		})
	}
}

func TestCodeDir_FallbackToWorkDir(t *testing.T) {
	c := &Conductor{
		opts: Options{WorkDir: "/some/dir"},
	}

	got := c.CodeDir()
	if got != "/some/dir" {
		t.Errorf("CodeDir() = %q, want %q", got, "/some/dir")
	}
}

func TestMergeLocalMetadata(t *testing.T) {
	tests := []struct {
		name            string
		workUnit        *provider.WorkUnit
		localTask       *storage.QueuedTask
		wantDescription string
		wantMetadata    map[string]any
	}{
		{
			name: "local description longer replaces provider",
			workUnit: &provider.WorkUnit{
				Description: "Short desc",
				Metadata:    map[string]any{"from_provider": true},
			},
			localTask: &storage.QueuedTask{
				Description: "A much longer local description with details and code examples",
			},
			wantDescription: "A much longer local description with details and code examples",
			wantMetadata:    map[string]any{"from_provider": true},
		},
		{
			name: "provider description longer preserved",
			workUnit: &provider.WorkUnit{
				Description: "A detailed provider description with lots of info",
			},
			localTask: &storage.QueuedTask{
				Description: "Short local",
			},
			wantDescription: "A detailed provider description with lots of info",
		},
		{
			name: "local metadata fills gaps",
			workUnit: &provider.WorkUnit{
				Description: "Test",
				Metadata:    map[string]any{"provider_key": "value"},
			},
			localTask: &storage.QueuedTask{
				Metadata: map[string]any{
					"local_key":    "local_value",
					"provider_key": "should_not_overwrite",
				},
			},
			wantDescription: "Test",
			wantMetadata: map[string]any{
				"provider_key": "value",
				"local_key":    "local_value",
			},
		},
		{
			name: "nil provider metadata initialized",
			workUnit: &provider.WorkUnit{
				Description: "Test",
				Metadata:    nil,
			},
			localTask: &storage.QueuedTask{
				Metadata: map[string]any{"key": "value"},
			},
			wantDescription: "Test",
			wantMetadata:    map[string]any{"key": "value"},
		},
		{
			name: "source path stored in metadata",
			workUnit: &provider.WorkUnit{
				Description: "Test",
			},
			localTask: &storage.QueuedTask{
				SourcePath: "/path/to/local/task.md",
			},
			wantDescription: "Test",
			wantMetadata:    map[string]any{"source_path": "/path/to/local/task.md"},
		},
		{
			name: "empty local task changes nothing",
			workUnit: &provider.WorkUnit{
				Description: "Original",
				Metadata:    map[string]any{"existing": true},
			},
			localTask:       &storage.QueuedTask{},
			wantDescription: "Original",
			wantMetadata:    map[string]any{"existing": true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Conductor{}
			c.mergeLocalMetadata(tt.workUnit, tt.localTask)

			if tt.workUnit.Description != tt.wantDescription {
				t.Errorf("Description = %q, want %q", tt.workUnit.Description, tt.wantDescription)
			}

			if tt.wantMetadata != nil {
				if tt.workUnit.Metadata == nil {
					t.Fatal("Metadata is nil, want non-nil")
				}
				for k, want := range tt.wantMetadata {
					got, exists := tt.workUnit.Metadata[k]
					if !exists {
						t.Errorf("Metadata[%q] not found, want %v", k, want)
					} else if got != want {
						t.Errorf("Metadata[%q] = %v, want %v", k, got, want)
					}
				}
			}
		})
	}
}

func TestMergeLocalSourceIntoSnapshot(t *testing.T) {
	t.Run("single file", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "task.md")
		if err := os.WriteFile(filePath, []byte("# Task\nSome content"), 0o644); err != nil {
			t.Fatal(err)
		}

		snapshot := &provider.Snapshot{
			Type:  "wrike",
			Ref:   "wrike:12345",
			Files: []provider.SnapshotFile{{Path: "original.md", Content: "original"}},
		}

		c := &Conductor{}
		c.mergeLocalSourceIntoSnapshot(snapshot, filePath)

		if len(snapshot.Files) != 2 {
			t.Fatalf("Files count = %d, want 2", len(snapshot.Files))
		}

		localFile := snapshot.Files[1]
		if localFile.Path != "local/task.md" {
			t.Errorf("local file Path = %q, want %q", localFile.Path, "local/task.md")
		}
		if localFile.Content != "# Task\nSome content" {
			t.Errorf("local file Content = %q, want %q", localFile.Content, "# Task\nSome content")
		}
	})

	t.Run("directory with multiple files", func(t *testing.T) {
		tmpDir := t.TempDir()
		sourceDir := filepath.Join(tmpDir, "source")
		if err := os.MkdirAll(filepath.Join(sourceDir, "sub"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(sourceDir, "a.md"), []byte("file a"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(sourceDir, "sub", "b.md"), []byte("file b"), 0o644); err != nil {
			t.Fatal(err)
		}

		snapshot := &provider.Snapshot{Type: "test", Ref: "test:1"}

		c := &Conductor{}
		c.mergeLocalSourceIntoSnapshot(snapshot, sourceDir)

		if len(snapshot.Files) < 2 {
			t.Fatalf("Files count = %d, want >= 2", len(snapshot.Files))
		}

		// Check that files are under local/ prefix
		for _, f := range snapshot.Files {
			if !strings.HasPrefix(f.Path, "local/") {
				t.Errorf("file %q not under local/ prefix", f.Path)
			}
		}
	})

	t.Run("nonexistent path skips gracefully", func(t *testing.T) {
		snapshot := &provider.Snapshot{Type: "test", Ref: "test:1"}

		c := &Conductor{}
		c.mergeLocalSourceIntoSnapshot(snapshot, "/nonexistent/path/file.md")

		if len(snapshot.Files) != 0 {
			t.Fatalf("Files count = %d, want 0 (nothing merged)", len(snapshot.Files))
		}
	})

	t.Run("nil snapshot is safe", func(t *testing.T) {
		c := &Conductor{}
		// Should not panic
		c.mergeLocalSourceIntoSnapshot(nil, "/some/path")
	})

	t.Run("skips hidden directories", func(t *testing.T) {
		tmpDir := t.TempDir()
		sourceDir := filepath.Join(tmpDir, "source")
		if err := os.MkdirAll(filepath.Join(sourceDir, ".hidden"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(sourceDir, "visible.md"), []byte("visible"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(sourceDir, ".hidden", "secret.md"), []byte("secret"), 0o644); err != nil {
			t.Fatal(err)
		}

		snapshot := &provider.Snapshot{Type: "test", Ref: "test:1"}

		c := &Conductor{}
		c.mergeLocalSourceIntoSnapshot(snapshot, sourceDir)

		for _, f := range snapshot.Files {
			if filepath.Base(f.Path) == "secret.md" {
				t.Error("hidden directory file should be skipped")
			}
		}
	})
}
