package wrike

import (
	"context"
	"testing"

	"github.com/valksor/go-mehrhof/internal/provider"
)

func TestInfo_DependencyCapabilities(t *testing.T) {
	info := Info()

	expectedCaps := []provider.Capability{
		provider.CapCreateDependency,
		provider.CapFetchDependencies,
	}

	for _, cap := range expectedCaps {
		if !info.Capabilities.Has(cap) {
			t.Errorf("Capabilities missing %q", cap)
		}
	}
}

func TestDependencyInterfaceImplementation(t *testing.T) {
	// Verify Provider implements the dependency interfaces
	var _ provider.DependencyCreator = (*Provider)(nil)
	var _ provider.DependencyFetcher = (*Provider)(nil)
}

func TestProviderNotInitialized(t *testing.T) {
	p := &Provider{client: nil}

	// Test CreateDependency with nil client
	err := p.CreateDependency(context.Background(), "task123", "task456")
	if err == nil {
		t.Error("CreateDependency with nil client should return error")
	}

	// Test GetDependencies with nil client
	_, err = p.GetDependencies(context.Background(), "task123")
	if err == nil {
		t.Error("GetDependencies with nil client should return error")
	}
}

func TestDependencyType(t *testing.T) {
	// Wrike uses native dependencies via dependencyIds field
	dep := Dependency{
		ID:            "dep123",
		PredecessorID: "task100",
		SuccessorID:   "task200",
		RelationType:  "FinishToStart",
	}

	if dep.PredecessorID != "task100" {
		t.Errorf("PredecessorID = %q, want %q", dep.PredecessorID, "task100")
	}
	if dep.SuccessorID != "task200" {
		t.Errorf("SuccessorID = %q, want %q", dep.SuccessorID, "task200")
	}
	if dep.RelationType != "FinishToStart" {
		t.Errorf("RelationType = %q, want %q", dep.RelationType, "FinishToStart")
	}
}

func TestCreateTaskOptionsWithDependencies(t *testing.T) {
	opts := CreateTaskOptions{
		Title:         "Test Task",
		Description:   "Test Description",
		DependencyIDs: []string{"task1", "task2", "task3"},
	}

	if len(opts.DependencyIDs) != 3 {
		t.Errorf("DependencyIDs length = %d, want 3", len(opts.DependencyIDs))
	}

	expected := []string{"task1", "task2", "task3"}
	for i, id := range opts.DependencyIDs {
		if id != expected[i] {
			t.Errorf("DependencyIDs[%d] = %q, want %q", i, id, expected[i])
		}
	}
}
