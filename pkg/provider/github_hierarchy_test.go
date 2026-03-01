package provider

import "testing"

func TestGitHubProvider_ImplementsHierarchyProvider(t *testing.T) {
	var _ HierarchyProvider = (*GitHubProvider)(nil)
}
