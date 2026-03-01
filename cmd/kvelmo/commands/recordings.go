package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/valksor/kvelmo/pkg/agent/recorder"
)

var recordingsDir string

// RecordingsCmd is the root command for managing agent recordings.
var RecordingsCmd = &cobra.Command{
	Use:   "recordings",
	Short: "Manage agent interaction recordings",
	Long: `View and manage recordings of agent interactions.

Recordings are JSONL files that capture all communication between
kvelmo and AI agents, useful for debugging and auditing.`,
}

var recordingsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all recordings",
	RunE:  runRecordingsList,
}

var recordingsViewCmd = &cobra.Command{
	Use:   "view <file>",
	Short: "View a recording file",
	Args:  cobra.ExactArgs(1),
	RunE:  runRecordingsView,
}

var recordingsReplayCmd = &cobra.Command{
	Use:   "replay <file>",
	Short: "Replay a recording with filtering",
	Args:  cobra.ExactArgs(1),
	RunE:  runRecordingsReplay,
}

var recordingsCleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Remove old recordings",
	RunE:  runRecordingsClean,
}

var (
	recordingsJobFilter   string
	recordingsSinceFilter string
	recordingsTypeFilter  string
	recordingsOlderThan   string
	recordingsOutputJSON  bool
)

func init() {
	// Default recordings directory with robust fallback
	homeDir, err := os.UserHomeDir()
	if err != nil || homeDir == "" {
		homeDir = os.Getenv("HOME")
		if homeDir == "" {
			homeDir = os.TempDir()
		}
	}
	defaultDir := filepath.Join(homeDir, ".valksor", "kvelmo", "recordings")

	RecordingsCmd.PersistentFlags().StringVar(&recordingsDir, "dir", defaultDir, "Recordings directory")

	recordingsListCmd.Flags().StringVar(&recordingsJobFilter, "job", "", "Filter by job ID")
	recordingsListCmd.Flags().StringVar(&recordingsSinceFilter, "since", "", "Show recordings since duration (e.g., 24h, 7d)")
	recordingsListCmd.Flags().BoolVar(&recordingsOutputJSON, "json", false, "Output as JSON")

	recordingsReplayCmd.Flags().StringVar(&recordingsTypeFilter, "filter", "", "Filter by event type (comma-separated)")

	recordingsCleanCmd.Flags().StringVar(&recordingsOlderThan, "older-than", "30d", "Remove recordings older than duration")

	RecordingsCmd.AddCommand(recordingsListCmd)
	RecordingsCmd.AddCommand(recordingsViewCmd)
	RecordingsCmd.AddCommand(recordingsReplayCmd)
	RecordingsCmd.AddCommand(recordingsCleanCmd)
}

func runRecordingsList(_ *cobra.Command, _ []string) error {
	infos, err := recorder.ListRecordings(recordingsDir)
	if err != nil {
		return fmt.Errorf("list recordings: %w", err)
	}

	// Filter by job if specified
	if recordingsJobFilter != "" {
		var filtered []recorder.RecordingInfo
		for _, info := range infos {
			if info.JobID == recordingsJobFilter {
				filtered = append(filtered, info)
			}
		}
		infos = filtered
	}

	// Filter by time if specified
	if recordingsSinceFilter != "" {
		since, err := parseDuration(recordingsSinceFilter)
		if err != nil {
			return fmt.Errorf("invalid --since duration: %w", err)
		}
		cutoff := time.Now().Add(-since)
		var filtered []recorder.RecordingInfo
		for _, info := range infos {
			// Parse the StartedAt string back to time (RFC3339 format)
			t, err := time.Parse(time.RFC3339, info.StartedAt)
			if err != nil {
				fmt.Fprintf(os.Stderr, "warning: skipping recording %s: invalid timestamp %q: %v\n",
					filepath.Base(info.Path), info.StartedAt, err)

				continue
			}
			if t.After(cutoff) {
				filtered = append(filtered, info)
			}
		}
		infos = filtered
	}

	// Check for empty results after filtering
	if len(infos) == 0 {
		fmt.Println("No recordings found")

		return nil
	}

	if recordingsOutputJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")

		return enc.Encode(infos)
	}

	// Table output
	fmt.Printf("%-12s %-8s %-8s %-20s %s\n", "JOB", "AGENT", "LINES", "STARTED", "FILE")
	fmt.Println(strings.Repeat("-", 80))
	for _, info := range infos {
		// Truncate job ID for display
		jobDisplay := info.JobID
		if len(jobDisplay) > 12 {
			jobDisplay = jobDisplay[:12]
		}
		fmt.Printf("%-12s %-8s %-8d %-20s %s\n",
			jobDisplay,
			info.Agent,
			info.Lines,
			info.StartedAt,
			filepath.Base(info.Path),
		)
	}

	fmt.Printf("\nTotal: %d recording(s)\n", len(infos))

	return nil
}

func runRecordingsView(_ *cobra.Command, args []string) error {
	path := args[0]

	// If not absolute, assume it's in the recordings dir
	if !filepath.IsAbs(path) {
		path = filepath.Join(recordingsDir, path)
	}

	reader, err := recorder.OpenReader(path)
	if err != nil {
		return fmt.Errorf("open recording: %w", err)
	}
	defer func() { _ = reader.Close() }()

	// Print header
	if h := reader.Header(); h != nil {
		fmt.Printf("Recording: %s\n", filepath.Base(path))
		fmt.Printf("Job: %s | Agent: %s | Model: %s\n", h.JobID, h.Agent, h.Model)
		fmt.Printf("Started: %s\n", h.StartedAt.Format(time.RFC3339))
		fmt.Println(strings.Repeat("-", 60))
	}

	// Print records
	for {
		rec, err := reader.Next()
		if err != nil {
			return fmt.Errorf("read record: %w", err)
		}
		if rec == nil {
			break
		}

		direction := "→"
		if rec.Direction == recorder.Inbound {
			direction = "←"
		}

		fmt.Printf("[%s] %s %s: ", rec.Timestamp.Format("15:04:05.000"), direction, rec.Type)

		// Pretty print the event
		var prettyEvent any
		if err := json.Unmarshal(rec.Event, &prettyEvent); err == nil {
			//nolint:errchkjson // Re-marshaling unmarshaled JSON won't fail
			eventJSON, _ := json.MarshalIndent(prettyEvent, "    ", "  ")
			fmt.Printf("\n    %s\n", eventJSON)
		} else {
			fmt.Printf("%s\n", rec.Event)
		}
	}

	return nil
}

func runRecordingsReplay(_ *cobra.Command, args []string) error {
	path := args[0]

	// If not absolute, assume it's in the recordings dir
	if !filepath.IsAbs(path) {
		path = filepath.Join(recordingsDir, path)
	}

	records, err := recorder.ReadAll(path)
	if err != nil {
		return fmt.Errorf("read recording: %w", err)
	}

	// Apply filter
	if recordingsTypeFilter != "" {
		types := strings.Split(recordingsTypeFilter, ",")
		for i := range types {
			types[i] = strings.TrimSpace(types[i])
		}
		records = recorder.FilterRecords(records, recorder.Filter{Types: types})
	}

	// Output filtered records
	for _, rec := range records {
		direction := "OUT"
		if rec.Direction == recorder.Inbound {
			direction = "IN "
		}

		fmt.Printf("%s [%s] %s: ", rec.Timestamp.Format("15:04:05.000"), direction, rec.Type)

		// Compact JSON output
		var compactEvent any
		if err := json.Unmarshal(rec.Event, &compactEvent); err == nil {
			//nolint:errchkjson // Re-marshaling unmarshaled JSON won't fail
			eventJSON, _ := json.Marshal(compactEvent)
			// Truncate long lines (rune-safe to preserve UTF-8)
			line := string(eventJSON)
			runes := []rune(line)
			if len(runes) > 120 {
				line = string(runes[:117]) + "..."
			}
			fmt.Println(line)
		} else {
			fmt.Printf("%s\n", rec.Event)
		}
	}

	fmt.Printf("\nTotal: %d record(s)\n", len(records))

	return nil
}

func runRecordingsClean(_ *cobra.Command, _ []string) error {
	duration, err := parseDuration(recordingsOlderThan)
	if err != nil {
		return fmt.Errorf("invalid --older-than duration: %w", err)
	}

	cutoff := time.Now().Add(-duration).Unix()
	removed, err := recorder.CleanOldRecordings(recordingsDir, cutoff)
	if err != nil {
		return fmt.Errorf("clean recordings: %w", err)
	}

	if removed == 0 {
		fmt.Println("No recordings to clean")
	} else {
		fmt.Printf("Removed %d recording(s)\n", removed)
	}

	return nil
}

// parseDuration parses duration strings like "24h", "7d", "30d".
// Returns an error if the duration is zero or negative.
func parseDuration(s string) (time.Duration, error) {
	var d time.Duration
	var err error

	// Handle day suffix
	if strings.HasSuffix(s, "d") {
		days, err := strconv.Atoi(strings.TrimSuffix(s, "d"))
		if err != nil {
			return 0, err
		}
		d = time.Duration(days) * 24 * time.Hour
	} else {
		// Standard duration
		d, err = time.ParseDuration(s)
		if err != nil {
			return 0, err
		}
	}

	if d <= 0 {
		return 0, fmt.Errorf("duration must be positive, got %v", d)
	}

	return d, nil
}
