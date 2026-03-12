package commands

import (
	"testing"
	"time"

	"github.com/spf13/cobra"
)

// findSubcommand finds a subcommand by Use prefix (first word).
func findSubcommand(parent *cobra.Command, name string) *cobra.Command {
	for _, sub := range parent.Commands() {
		// Use field may include args like "view <file>", match first word
		if len(sub.Use) >= len(name) && sub.Use[:len(name)] == name {
			return sub
		}
	}

	return nil
}

func TestParseDuration(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    time.Duration
		wantErr bool
	}{
		{"hours", "24h", 24 * time.Hour, false},
		{"days", "7d", 7 * 24 * time.Hour, false},
		{"30 days", "30d", 30 * 24 * time.Hour, false},
		{"minutes", "30m", 30 * time.Minute, false},
		{"invalid days", "xd", 0, true},
		{"invalid format", "abc", 0, true},
		{"zero duration", "0h", 0, true},
		{"negative days", "-1d", 0, true},
		{"1 day", "1d", 24 * time.Hour, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseDuration(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseDuration(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)

				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("parseDuration(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestRecordingsCommand(t *testing.T) {
	if RecordingsCmd.Use != "recordings" {
		t.Errorf("Use = %s, want recordings", RecordingsCmd.Use)
	}
	// Verify subcommands are registered
	subs := RecordingsCmd.Commands()
	subNames := make(map[string]bool)
	for _, sub := range subs {
		subNames[sub.Use] = true
	}
	for _, want := range []string{"list", "view <file>", "replay <file>", "clean"} {
		if !subNames[want] {
			t.Errorf("missing subcommand %q", want)
		}
	}
	// Verify persistent flags
	if f := RecordingsCmd.PersistentFlags().Lookup("dir"); f == nil {
		t.Error("--dir persistent flag should exist")
	}
}

func TestRecordingsListFlags(t *testing.T) {
	listCmd := findSubcommand(RecordingsCmd, "list")
	if listCmd == nil {
		t.Fatal("list subcommand not found")
	}
	if f := listCmd.Flags().Lookup("job"); f == nil {
		t.Error("--job flag should exist")
	}
	if f := listCmd.Flags().Lookup("since"); f == nil {
		t.Error("--since flag should exist")
	}
	if f := listCmd.Flags().Lookup("json"); f == nil {
		t.Error("--json flag should exist")
	}
}

func TestRecordingsCleanFlags(t *testing.T) {
	cleanCmd := findSubcommand(RecordingsCmd, "clean")
	if cleanCmd == nil {
		t.Fatal("clean subcommand not found")
	}
	if f := cleanCmd.Flags().Lookup("older-than"); f == nil {
		t.Error("--older-than flag should exist")
	}
}

func TestRecordingsReplayFlags(t *testing.T) {
	replayCmd := findSubcommand(RecordingsCmd, "replay")
	if replayCmd == nil {
		t.Fatal("replay subcommand not found")
	}
	if f := replayCmd.Flags().Lookup("filter"); f == nil {
		t.Error("--filter flag should exist")
	}
}
