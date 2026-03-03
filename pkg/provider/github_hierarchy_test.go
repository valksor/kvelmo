package provider

import "testing"

func TestGitHubProvider_ImplementsHierarchyProvider(t *testing.T) {
	var _ HierarchyProvider = (*GitHubProvider)(nil)
}

func TestGitHubProvider_ImplementsMergeProvider(t *testing.T) {
	var _ MergeProvider = (*GitHubProvider)(nil)
}
