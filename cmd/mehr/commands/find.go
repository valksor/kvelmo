package commands

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/valksor/go-mehrhof/internal/conductor"
	"github.com/valksor/go-toolkit/display"
)

var (
	findPath    string
	findPattern string
	findFormat  string
	findStream  bool
	findAgent   string
	findContext int
)

var findCmd = &cobra.Command{
	Use:   "find <query>",
	Short: "AI-powered code search with focused results",
	Long: `Search codebase using AI with minimal fluff.

Unlike asking Claude directly, this command uses a specialized prompt that
instructs the agent to search efficiently without extra exploration or
explanations.

The agent uses Grep/Glob/Read tools to find code matching your query and
reports results in a structured format.

FEATURES:
  - Project-scoped: Works without an active task
  - Focused: Agent instructed to avoid exploratory behavior
  - Flexible: Multiple output formats for different use cases

OUTPUT FORMATS:
  concise    - File:line:snippet format (default)
  structured - Numbered list with context
  json       - Machine-readable JSON output

FLAGS:
  --path, -p      - Restrict search to directory
  --pattern       - Glob pattern for files to search
  --format        - Output format: concise|structured|json
  --stream        - Stream results as found (for large codebases)
  --agent         - Use specific agent
  --context, -C   - Lines of context to include (default: 3)

EXAMPLES:
  # Basic search
  mehr find "archive_blade database table"

  # Restrict to directory
  mehr find "authentication" --path ./internal/auth/

  # With file pattern
  mehr find "User struct" --pattern "**/*.go"

  # Structured output
  mehr find "API endpoints" --format structured

  # JSON for scripting
  mehr find "TODO comments" --format json | jq '.matches[]'

  # Stream results (for large codebases)
  mehr find "memory leak" --stream

  # Use specific agent
  mehr find "graphql" --agent opus`,
	RunE: runFind,
}

func init() {
	rootCmd.AddCommand(findCmd)

	findCmd.Flags().StringVarP(&findPath, "path", "p", "",
		"Restrict search to directory (relative to project root)")
	findCmd.Flags().StringVar(&findPattern, "pattern", "",
		"Glob pattern for files to search (e.g., **/*.go)")
	findCmd.Flags().StringVar(&findFormat, "format", "concise",
		"Output format: concise|structured|json")
	findCmd.Flags().BoolVar(&findStream, "stream", false,
		"Stream results as found (for large codebases)")
	findCmd.Flags().StringVar(&findAgent, "agent", "",
		"Agent to use for search")
	findCmd.Flags().IntVarP(&findContext, "context", "C", 3,
		"Lines of context to include in results")
}

func runFind(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	// Get query from args or flag
	var query string
	if len(args) > 0 {
		query = strings.Join(args, " ")
	} else {
		// Check if the query was piped in
		stat, _ := os.Stdin.Stat()
		if (stat.Mode() & os.ModeCharDevice) == 0 {
			// Input is being piped
			var sb strings.Builder
			scanner := newScanner(os.Stdin)
			for scanner.Scan() {
				sb.WriteString(scanner.Text())
			}
			query = sb.String()
		}
	}

	if query == "" {
		return errors.New("query is required")
	}

	// Validate format
	validFormats := map[string]bool{"concise": true, "structured": true, "json": true}
	if !validFormats[findFormat] {
		return fmt.Errorf("invalid format %q, must be one of: concise, structured, json", findFormat)
	}

	// Build conductor options
	opts := BuildConductorOptions(CommandOptions{
		Verbose: verbose,
	})

	if findAgent != "" {
		opts = append(opts, conductor.WithStepAgent("finding", findAgent))
	}

	// Initialize conductor (no active task required)
	cond, err := initializeConductor(ctx, opts...)
	if err != nil {
		return err
	}

	// Set up verbose event handlers if needed
	if verbose {
		SetupVerboseEventHandlers(cond)
	}

	// Build find options
	findOpts := conductor.FindOptions{
		Query:     query,
		Path:      findPath,
		Pattern:   findPattern,
		Context:   findContext,
		Workspace: cond.GetWorkspace(),
	}

	// Execute search
	if findStream || findFormat == "json" {
		// Stream mode - show spinner first
		if !verbose {
			spinner := display.NewSpinner("Searching...")
			spinner.Start()
			defer spinner.Stop()
		}

		return runFindStream(ctx, cond, findOpts, findFormat)
	}

	// Non-stream mode - collect all results first
	if verbose {
		fmt.Println(display.InfoMsg("Searching..."))
	} else {
		spinner := display.NewSpinner("Searching...")
		spinner.Start()
		defer spinner.Stop()
	}

	resultChan, err := cond.Find(ctx, findOpts)
	if err != nil {
		return fmt.Errorf("find: %w", err)
	}

	var results []conductor.FindResult
	for result := range resultChan {
		if result.File == "__error__" {
			return fmt.Errorf("search error: %s", result.Snippet)
		}
		results = append(results, result)
	}

	// Format and display results
	return formatFindResults(results, query, findFormat)
}

// runFindStream handles streaming output mode.
func runFindStream(ctx context.Context, cond *conductor.Conductor, opts conductor.FindOptions, format string) error {
	resultChan, err := cond.Find(ctx, opts)
	if err != nil {
		return fmt.Errorf("find: %w", err)
	}

	if format == "json" {
		return streamFindResultsJSON(resultChan)
	}

	return streamFindResultsText(resultChan)
}

// streamFindResultsJSON streams results in JSON format.
func streamFindResultsJSON(resultChan <-chan conductor.FindResult) error {
	fmt.Println(`{"matches": [`)

	first := true
	for result := range resultChan {
		if result.File == "__error__" {
			fmt.Fprintf(os.Stderr, "Error: %s\n", result.Snippet)

			continue
		}

		if !first {
			fmt.Println(",")
		}
		first = false

		data, err := json.Marshal(result)
		if err != nil {
			data = []byte(`{"file": "__error__", "snippet": "failed to marshal result"}`)
		}
		fmt.Printf("  %s", string(data))
	}

	fmt.Println("\n]}")

	return nil
}

// streamFindResultsText streams results in text format.
func streamFindResultsText(resultChan <-chan conductor.FindResult) error {
	count := 0
	for result := range resultChan {
		if result.File == "__error__" {
			fmt.Fprintf(os.Stderr, "Error: %s\n", result.Snippet)

			continue
		}

		count++
		fmt.Printf("%s:%d: %s\n", result.File, result.Line, result.Snippet)
	}

	if count == 0 {
		fmt.Println("No matches found.")
	}

	return nil
}

// formatFindResults formats and displays results.
func formatFindResults(results []conductor.FindResult, query, format string) error {
	if len(results) == 0 {
		fmt.Println("No matches found.")

		return nil
	}

	switch format {
	case "concise":
		return formatFindConcise(results)
	case "structured":
		return formatFindStructured(results, query)
	case "json":
		return formatFindJSON(results)
	}

	return nil
}

// formatFindConcise outputs results in "file:line: snippet" format.
func formatFindConcise(results []conductor.FindResult) error {
	for _, r := range results {
		fmt.Printf("%s:%d: %s\n", r.File, r.Line, r.Snippet)
	}

	return nil
}

// formatFindStructured outputs results in a numbered, formatted list.
func formatFindStructured(results []conductor.FindResult, query string) error {
	fmt.Printf("Found %d match(es) for %q\n\n", len(results), query)

	for i, r := range results {
		fmt.Printf("%d. %s:%d\n", i+1, r.File, r.Line)

		// Show snippet
		if r.Snippet != "" {
			fmt.Printf("   %s\n", display.Muted(r.Snippet))
		}

		// Show context if available
		if len(r.Context) > 0 {
			fmt.Printf("   %s\n", display.Muted("---"))
			for _, ctx := range r.Context {
				fmt.Printf("   %s\n", display.Muted(ctx))
			}
		}

		// Show reason if available
		if r.Reason != "" {
			fmt.Printf("   → %s\n", r.Reason)
		}

		fmt.Println()
	}

	return nil
}

// formatFindJSON outputs results as JSON.
func formatFindJSON(results []conductor.FindResult) error {
	output := struct {
		Query   string                 `json:"query"`
		Count   int                    `json:"count"`
		Matches []conductor.FindResult `json:"matches"`
	}{
		Query:   "",
		Count:   len(results),
		Matches: results,
	}

	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return err
	}

	fmt.Println(string(data))

	return nil
}

// newScanner creates a buffered scanner.
func newScanner(r *os.File) *bufio.Scanner {
	return bufio.NewScanner(r)
}
