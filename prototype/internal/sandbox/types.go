package sandbox

import "runtime"

// Status represents the current sandbox state for API responses.
type Status struct {
	Enabled   bool   `json:"enabled"`   // Is sandbox enabled in config
	Platform  string `json:"platform"`  // "linux", "darwin", "windows", etc.
	Active    bool   `json:"active"`    // Is sandbox currently active for running task
	Network   bool   `json:"network"`   // Network access enabled
	Supported bool   `json:"supported"` // Whether the platform supports sandboxing
	Profile   string `json:"profile"`   // Applied profile name or error message
}

// Config holds sandbox configuration.
type Config struct {
	Enabled    bool     // Enable sandboxing
	ProjectDir string   // Project directory to mount
	HomeDir    string   // Home directory for .claude access
	TmpDir     string   // Path for tmpfs mount (empty = auto)
	Tools      []string // Additional binary paths to bind mount
	Network    bool     // Allow network access (default: true - needed for LLM APIs)
	Profile    string   // macOS: custom SBPL profile (overrides generated)
}

// ToStatus converts Config to Status for API responses.
func (c *Config) ToStatus() Status {
	return Status{
		Enabled:   c.Enabled,
		Platform:  runtime.GOOS,
		Active:    false, // Set when sandbox is actually active
		Network:   c.Network,
		Supported: Supported(),
		Profile:   c.Profile,
	}
}

// NewConfig creates a default Config with the given project and home directories.
func NewConfig(projectDir, homeDir string) *Config {
	return &Config{
		Enabled:    true,
		ProjectDir: projectDir,
		HomeDir:    homeDir,
		TmpDir:     "", // auto-generate
		Tools:      nil,
		Network:    true, // LLM APIs need network
		Profile:    "",
	}
}
