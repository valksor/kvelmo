package provider

import "strings"

// DetectProviderFromURL parses a URL to determine the provider.
// Returns the provider name (github, gitlab, bitbucket, azuredevops) or empty string if unknown.
func DetectProviderFromURL(url string) string {
	switch {
	case strings.Contains(url, "github.com"):
		return "github"
	case strings.Contains(url, "gitlab.com"):
		return "gitlab"
	case strings.Contains(url, "bitbucket.org"):
		return "bitbucket"
	case strings.Contains(url, "dev.azure.com"), strings.Contains(url, "azure.com"), strings.Contains(url, "visualstudio.com"):
		return "azuredevops"
	default:
		return ""
	}
}
