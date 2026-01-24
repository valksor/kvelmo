package asana

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
	err := p.CreateDependency(context.Background(), "123", "456")
	if err == nil {
		t.Error("CreateDependency with nil client should return error")
	}

	// Test GetDependencies with nil client
	_, err = p.GetDependencies(context.Background(), "123")
	if err == nil {
		t.Error("GetDependencies with nil client should return error")
	}
}
