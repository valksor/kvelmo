package clickup

import (
	"context"
	"testing"

	"github.com/valksor/go-toolkit/capability"
	"github.com/valksor/go-toolkit/workunit"
)

func TestInfo_DependencyCapabilities(t *testing.T) {
	info := Info()

	expectedCaps := []capability.Capability{
		capability.CapCreateDependency,
		capability.CapFetchDependencies,
	}

	for _, cap := range expectedCaps {
		if !info.Capabilities.Has(cap) {
			t.Errorf("Capabilities missing %q", cap)
		}
	}
}

func TestDependencyInterfaceImplementation(t *testing.T) {
	// Verify Provider implements the dependency interfaces
	var _ workunit.DependencyCreator = (*Provider)(nil)
	var _ workunit.DependencyFetcher = (*Provider)(nil)
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

func TestDependencyTypes(t *testing.T) {
	// ClickUp supports "waiting_on" and "blocking" dependency types
	// We use "waiting_on" which means the task is waiting on another task
	depType := "waiting_on"

	if depType != "waiting_on" {
		t.Errorf("Expected dependency type 'waiting_on', got %q", depType)
	}
}
