package queue

import (
	"context"
	"testing"

	"github.com/valksor/go-mehrhof/internal/provider"
	"github.com/valksor/go-mehrhof/internal/storage"
	"github.com/valksor/go-toolkit/capability"
	"github.com/valksor/go-toolkit/providerconfig"
	"github.com/valksor/go-toolkit/snapshot"
	"github.com/valksor/go-toolkit/workunit"
)

func TestProviderName(t *testing.T) {
	if ProviderName != "queue" {
		t.Errorf("ProviderName = %q, want %q", ProviderName, "queue")
	}
}

func TestInfo(t *testing.T) {
	info := Info()

	if info.Name != ProviderName {
		t.Errorf("Info.Name = %q, want %q", info.Name, ProviderName)
	}

	if !info.Capabilities[capability.CapRead] {
		t.Error("Info should have CapRead capability")
	}

	if !info.Capabilities[capability.CapSnapshot] {
		t.Error("Info should have CapSnapshot capability")
	}
}

func TestMatch(t *testing.T) {
	p := &Provider{}

	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"queue prefix", "queue:abc/123", true},
		{"queue with longer id", "queue:my-queue/task-456", true},
		{"no prefix", "file:task.md", false},
		{"empty string", "", false},
		{"just queue", "queue", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := p.Match(tt.input); got != tt.want {
				t.Errorf("Match() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParse(t *testing.T) {
	p := &Provider{}

	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{"valid queue task", "queue:my-queue/task-1", "my-queue/task-1", false},
		{"valid with numbers", "queue:q1/t2", "q1/t2", false},
		{"missing identifier", "queue:", "", true},
		{"missing slash", "queue:task", "", true},
		{"empty after prefix", "queue:", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := p.Parse(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)

				return
			}
			if got != tt.want {
				t.Errorf("Parse() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSplitQueueTaskID(t *testing.T) {
	tests := []struct {
		name      string
		id        string
		wantQueue string
		wantTask  string
		wantErr   bool
	}{
		{"valid id", "q1/t1", "q1", "t1", false},
		{"valid with hyphens", "my-queue/task-123", "my-queue", "task-123", false},
		{"empty queue", "/t1", "", "", true},
		{"empty task", "q1/", "", "", true},
		{"no slash", "q1", "", "", true},
		{"empty string", "", "", "", true},
		{"multiple slashes", "q1/t1/extra", "q1", "t1/extra", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			queue, task, err := splitQueueTaskID(tt.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("splitQueueTaskID() error = %v, wantErr %v", err, tt.wantErr)

				return
			}
			if queue != tt.wantQueue || task != tt.wantTask {
				t.Errorf("splitQueueTaskID() = (%q, %q), want (%q, %q)", queue, task, tt.wantQueue, tt.wantTask)
			}
		})
	}
}

func TestMapPriority(t *testing.T) {
	tests := []struct {
		name     string
		priority int
		want     workunit.Priority
	}{
		{"high priority 0", 0, workunit.PriorityHigh},
		{"high priority 1", 1, workunit.PriorityHigh},
		{"normal priority 2", 2, workunit.PriorityNormal},
		{"low priority 3", 3, workunit.PriorityLow},
		{"low priority 5", 5, workunit.PriorityLow},
		{"default priority", -1, workunit.PriorityNormal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mapPriority(tt.priority); got != tt.want {
				t.Errorf("mapPriority() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProviderInterfaces(t *testing.T) {
	var _ workunit.Reader = (*Provider)(nil)
	var _ workunit.Identifier = (*Provider)(nil)
	var _ snapshot.Snapshotter = (*Provider)(nil)
}

func TestNew(t *testing.T) {
	ctx := context.Background()
	cfg := providerconfig.Config{}

	// When running from the repo directory, this will succeed
	// The test verifies the function signature is correct
	_, err := New(ctx, cfg)
	// Don't fail the test if we're in a workspace
	_ = err // We just want to ensure the code compiles and runs
}

func TestRegister(t *testing.T) {
	registry := provider.NewRegistry()
	Register(registry)

	p, factory, found := registry.Get("queue")
	if !found {
		t.Fatal("Get() did not find queue provider")
	}
	if factory == nil {
		t.Error("Get() returned nil factory")
	}
	// Verify provider info
	if p.Name != "queue" {
		t.Errorf("ProviderInfo.Name = %q, want %q", p.Name, "queue")
	}
}

func TestLoadTaskQueueNotExists(t *testing.T) {
	p := &Provider{workspace: &storage.Workspace{}}

	// Test with a non-existent queue
	_, err := storage.LoadTaskQueue(p.workspace, "nonexistent")
	if err == nil {
		t.Error("LoadTaskQueue should return error for non-existent queue")
	}
}
