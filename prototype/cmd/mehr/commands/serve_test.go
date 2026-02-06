//go:build !testbinary

package commands

import (
	"bytes"
	"os"
	"testing"
)

func TestServeCommand_Properties(t *testing.T) {
	if serveCmd.Use != "serve" {
		t.Errorf("Use = %q, want %q", serveCmd.Use, "serve")
	}

	if serveCmd.Short == "" {
		t.Error("Short description is empty")
	}

	if serveCmd.Long == "" {
		t.Error("Long description is empty")
	}

	if serveCmd.RunE == nil {
		t.Error("RunE not set")
	}
}

func TestServeCommand_Flags(t *testing.T) {
	tests := []struct {
		name         string
		flagName     string
		shorthand    string
		defaultValue string
	}{
		{
			name:         "port flag",
			flagName:     "port",
			shorthand:    "p",
			defaultValue: "0",
		},
		{
			name:         "global flag",
			flagName:     "global",
			shorthand:    "",
			defaultValue: "false",
		},
		{
			name:         "open flag",
			flagName:     "open",
			shorthand:    "",
			defaultValue: "false",
		},
		{
			name:         "api flag",
			flagName:     "api",
			shorthand:    "",
			defaultValue: "false",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag := serveCmd.Flags().Lookup(tt.flagName)
			if flag == nil {
				t.Errorf("flag %q not found", tt.flagName)

				return
			}

			if flag.DefValue != tt.defaultValue {
				t.Errorf("flag %q default value = %q, want %q", tt.flagName, flag.DefValue, tt.defaultValue)
			}

			if tt.shorthand != "" {
				shorthand := serveCmd.Flags().ShorthandLookup(tt.shorthand)
				if shorthand == nil {
					t.Errorf("shorthand %q not found for flag %q", tt.shorthand, tt.flagName)
				}
			}
		})
	}
}

func TestServeCommand_Subcommands(t *testing.T) {
	subcommands := serveCmd.Commands()

	// Register/unregister should be present (needed for global mode)
	expectedNames := []string{"register", "unregister"}
	for _, name := range expectedNames {
		found := false
		for _, cmd := range subcommands {
			if cmd.Name() == name {
				found = true

				break
			}
		}
		if !found {
			t.Errorf("serve command missing %q subcommand", name)
		}
	}

	// Auth subcommand is disabled (remote serve)
	for _, cmd := range subcommands {
		if cmd.Name() == "auth" {
			t.Error("serve command should not have \"auth\" subcommand (remote serve disabled)")
		}
	}
}

func TestServeAuthCommand_Subcommands(t *testing.T) {
	t.Skip("remote serve temporarily disabled")

	subcommands := serveAuthCmd.Commands()

	expectedNames := []string{"add", "list", "remove", "passwd", "role"}
	for _, name := range expectedNames {
		found := false
		for _, cmd := range subcommands {
			if cmd.Name() == name {
				found = true

				break
			}
		}
		if !found {
			t.Errorf("serve auth command missing %q subcommand", name)
		}
	}
}

func TestServeCommand_RegisteredInRoot(t *testing.T) {
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "serve" {
			found = true

			break
		}
	}
	if !found {
		t.Error("serve command not registered in root command")
	}
}

func TestExtractRepoName(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{
			name:     "HTTPS URL with .git",
			url:      "https://github.com/user/repo.git",
			expected: "repo",
		},
		{
			name:     "HTTPS URL without .git",
			url:      "https://github.com/user/repo",
			expected: "repo",
		},
		{
			name:     "SSH URL",
			url:      "git@github.com:user/repo.git",
			expected: "repo",
		},
		{
			name:     "bare name",
			url:      "myrepo",
			expected: "myrepo",
		},
		{
			name:     "bare name with .git",
			url:      "myrepo.git",
			expected: "myrepo",
		},
		{
			name:     "nested path",
			url:      "https://gitlab.com/group/subgroup/repo.git",
			expected: "repo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractRepoName(tt.url)
			if got != tt.expected {
				t.Errorf("extractRepoName(%q) = %q, want %q", tt.url, got, tt.expected)
			}
		})
	}
}

func TestTruncatePath(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		maxLen   int
		expected string
	}{
		{
			name:     "short path within limit",
			path:     "/home/user",
			maxLen:   20,
			expected: "/home/user",
		},
		{
			name:     "exact length",
			path:     "12345",
			maxLen:   5,
			expected: "12345",
		},
		{
			name:     "long path truncated",
			path:     "/very/long/path/to/some/directory",
			maxLen:   20,
			expected: "...to/some/directory",
		},
		{
			name:     "empty path",
			path:     "",
			maxLen:   10,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncatePath(tt.path, tt.maxLen)
			if got != tt.expected {
				t.Errorf("truncatePath(%q, %d) = %q, want %q", tt.path, tt.maxLen, got, tt.expected)
			}
		})
	}
}

func TestTrimSuffix(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		suffix   string
		expected string
	}{
		{
			name:     "matching suffix",
			s:        "repo.git",
			suffix:   ".git",
			expected: "repo",
		},
		{
			name:     "no matching suffix",
			s:        "repo",
			suffix:   ".git",
			expected: "repo",
		},
		{
			name:     "empty string",
			s:        "",
			suffix:   ".git",
			expected: "",
		},
		{
			name:     "empty suffix",
			s:        "repo.git",
			suffix:   "",
			expected: "repo.git",
		},
		{
			name:     "suffix equals string",
			s:        ".git",
			suffix:   ".git",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := trimSuffix(tt.s, tt.suffix)
			if got != tt.expected {
				t.Errorf("trimSuffix(%q, %q) = %q, want %q", tt.s, tt.suffix, got, tt.expected)
			}
		})
	}
}

func TestSplitAny(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		sep      string
		expected []string
	}{
		{
			name:     "split by slash",
			s:        "a/b/c",
			sep:      "/",
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "split by slash and colon",
			s:        "git@github.com:user/repo",
			sep:      "/:",
			expected: []string{"git@github.com", "user", "repo"},
		},
		{
			name:     "no separator found",
			s:        "nosep",
			sep:      "/:",
			expected: []string{"nosep"},
		},
		{
			name:     "leading separator",
			s:        "/a/b",
			sep:      "/",
			expected: []string{"a", "b"},
		},
		{
			name:     "trailing separator",
			s:        "a/b/",
			sep:      "/",
			expected: []string{"a", "b"},
		},
		{
			name:     "empty string",
			s:        "",
			sep:      "/",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := splitAny(tt.s, tt.sep)
			if len(got) != len(tt.expected) {
				t.Errorf("splitAny(%q, %q) = %v (len %d), want %v (len %d)",
					tt.s, tt.sep, got, len(got), tt.expected, len(tt.expected))

				return
			}

			for i := range got {
				if got[i] != tt.expected[i] {
					t.Errorf("splitAny(%q, %q)[%d] = %q, want %q",
						tt.s, tt.sep, i, got[i], tt.expected[i])
				}
			}
		})
	}
}

func TestRunServe_TunnelInfo(t *testing.T) {
	t.Skip("remote serve temporarily disabled")

	// Save and restore flag
	origTunnelInfo := serveTunnelInfo
	origPort := servePort
	defer func() {
		serveTunnelInfo = origTunnelInfo
		servePort = origPort
	}()

	serveTunnelInfo = true
	servePort = 6337

	// Capture stdout
	r, w, _ := os.Pipe()
	oldStdout := os.Stdout
	os.Stdout = w

	err := runServe(serveCmd, nil)

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("runServe() with --tunnel-info returned error: %v", err)
	}

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	expectedContents := []string{
		"SSH Tunnel",
		"localhost:6337",
	}

	for _, want := range expectedContents {
		if !containsString(output, want) {
			t.Errorf("tunnel-info output does not contain %q\nGot:\n%s", want, output)
		}
	}
}
