package schema

import "strings"

// SectionRegistry maps field path prefixes to section metadata.
// The key is the first component of the field path (e.g., "git" for "git.commit_prefix").
var SectionRegistry = map[string]SectionMeta{
	// Core settings
	"git": {
		Title:       "Git",
		Description: "Version control settings",
		Icon:        "git-branch",
		Category:    "core",
	},
	"agent": {
		Title:       "Agent",
		Description: "AI agent configuration",
		Icon:        "bot",
		Category:    "core",
	},
	"workflow": {
		Title:       "Workflow",
		Description: "Task workflow settings",
		Icon:        "git-pull-request",
		Category:    "core",
	},
	"budget": {
		Title:       "Budget",
		Description: "Cost and budget controls",
		Icon:        "wallet",
		Category:    "core",
	},
	"project": {
		Title:       "Project",
		Description: "Project configuration",
		Icon:        "folder",
		Category:    "core",
	},
	"storage": {
		Title:       "Storage",
		Description: "Data storage settings",
		Icon:        "database",
		Category:    "core",
	},
	"update": {
		Title:       "Updates",
		Description: "Automatic update settings",
		Icon:        "download",
		Category:    "core",
	},
	"specification": {
		Title:       "Specifications",
		Description: "Spec file naming settings",
		Icon:        "file-text",
		Category:    "core",
	},
	"review": {
		Title:       "Reviews",
		Description: "Review file naming settings",
		Icon:        "message-square",
		Category:    "core",
	},

	// Providers
	"providers": {
		Title:       "Default Provider",
		Description: "Default task source settings",
		Icon:        "plug",
		Category:    "providers",
	},
	"github": {
		Title:       "GitHub",
		Description: "GitHub integration",
		Icon:        "github",
		Category:    "providers",
	},
	"gitlab": {
		Title:       "GitLab",
		Description: "GitLab integration",
		Icon:        "gitlab",
		Category:    "providers",
	},
	"jira": {
		Title:       "Jira",
		Description: "Jira integration",
		Icon:        "layout-kanban",
		Category:    "providers",
	},
	"linear": {
		Title:       "Linear",
		Description: "Linear integration",
		Icon:        "layers",
		Category:    "providers",
	},
	"notion": {
		Title:       "Notion",
		Description: "Notion integration",
		Icon:        "book-open",
		Category:    "providers",
	},
	"bitbucket": {
		Title:       "Bitbucket",
		Description: "Bitbucket integration",
		Icon:        "bitbucket",
		Category:    "providers",
	},
	"asana": {
		Title:       "Asana",
		Description: "Asana integration",
		Icon:        "check-square",
		Category:    "providers",
	},
	"clickup": {
		Title:       "ClickUp",
		Description: "ClickUp integration",
		Icon:        "check-circle",
		Category:    "providers",
	},
	"azure_devops": {
		Title:       "Azure DevOps",
		Description: "Azure DevOps integration",
		Icon:        "cloud",
		Category:    "providers",
	},
	"trello": {
		Title:       "Trello",
		Description: "Trello integration",
		Icon:        "trello",
		Category:    "providers",
	},
	"wrike": {
		Title:       "Wrike",
		Description: "Wrike integration",
		Icon:        "briefcase",
		Category:    "providers",
	},
	"youtrack": {
		Title:       "YouTrack",
		Description: "YouTrack integration",
		Icon:        "target",
		Category:    "providers",
	},

	// Features
	"browser": {
		Title:       "Browser Automation",
		Description: "Chrome DevTools Protocol settings",
		Icon:        "globe",
		Category:    "features",
	},
	"mcp": {
		Title:       "MCP",
		Description: "Model Context Protocol settings",
		Icon:        "cpu",
		Category:    "features",
	},
	"security": {
		Title:       "Security",
		Description: "Security scanning configuration",
		Icon:        "shield",
		Category:    "features",
	},
	"memory": {
		Title:       "Memory",
		Description: "Semantic memory settings",
		Icon:        "brain",
		Category:    "features",
	},
	"library": {
		Title:       "Library",
		Description: "Shared library settings",
		Icon:        "library",
		Category:    "features",
	},
	"orchestration": {
		Title:       "Orchestration",
		Description: "Agent orchestration settings",
		Icon:        "network",
		Category:    "features",
	},
	"ml": {
		Title:       "ML",
		Description: "Machine learning settings",
		Icon:        "sparkles",
		Category:    "features",
	},
	"sandbox": {
		Title:       "Sandbox",
		Description: "Code sandbox settings",
		Icon:        "box",
		Category:    "features",
	},
	"labels": {
		Title:       "Labels",
		Description: "Label configuration",
		Icon:        "tag",
		Category:    "features",
	},
	"quality": {
		Title:       "Quality",
		Description: "Code quality settings",
		Icon:        "check",
		Category:    "features",
	},
	"links": {
		Title:       "Links",
		Description: "Bidirectional linking settings",
		Icon:        "link",
		Category:    "features",
	},
	"context": {
		Title:       "Context",
		Description: "Context window settings",
		Icon:        "file-text",
		Category:    "features",
	},
	"stack": {
		Title:       "Stack",
		Description: "Tech stack configuration",
		Icon:        "layers",
		Category:    "features",
	},
	"display": {
		Title:       "Display",
		Description: "Display settings",
		Icon:        "monitor",
		Category:    "features",
	},
	"plugins": {
		Title:       "Plugins",
		Description: "External plugin configuration",
		Icon:        "puzzle",
		Category:    "features",
	},
}

// GetSectionMeta returns the metadata for a section by its ID.
// Returns a default SectionMeta if not found.
func GetSectionMeta(sectionID string) SectionMeta {
	if meta, ok := SectionRegistry[sectionID]; ok {
		return meta
	}
	// Default: capitalize the section ID as title
	return SectionMeta{
		Title:    capitalizeFirst(sectionID),
		Category: "features",
	}
}

// capitalizeFirst capitalizes the first letter of a string.
func capitalizeFirst(s string) string {
	if s == "" {
		return s
	}

	return strings.ToUpper(s[:1]) + s[1:]
}
