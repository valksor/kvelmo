package commands

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/valksor/go-mehrhof/internal/memory"
	"github.com/valksor/go-mehrhof/internal/storage"
)

var (
	memoryQuery     string
	memoryLimit     int
	memoryTypes     []string
	memoryTaskID    string
	memoryClear     bool
	memoryStats     bool
	memoryIndexTask string
)

var memoryCmd = &cobra.Command{
	Use:   "memory",
	Short: "Manage semantic memory of past tasks",
	Long: `Search and manage the semantic memory system that stores embeddings
of code changes, specifications, and solutions from past tasks.

The memory system enables finding similar past tasks and auto-suggesting
solutions based on historical context.

Commands:
  mehr memory search <query>        - Search for similar past tasks
  mehr memory index                 - Index current/completed task
  mehr memory stats                 - Show memory statistics
  mehr memory clear                 - Clear all stored memory

Examples:
  mehr memory search "authentication"    - Find tasks related to authentication
  mehr memory search "api endpoint" --limit 10
  mehr memory index --task abc123        - Index specific task
  mehr memory stats                      - Show memory statistics
  mehr memory clear                      - Clear all memory`,
	RunE: runMemory,
}

func init() {
	rootCmd.AddCommand(memoryCmd)
	memoryCmd.GroupID = "utility" // Add to the utility group

	memoryCmd.Flags().StringVarP(&memoryQuery, "search", "s", "", "Search query for similar tasks")
	memoryCmd.Flags().IntVarP(&memoryLimit, "limit", "l", 5, "Maximum results to return")
	memoryCmd.Flags().StringSliceVarP(&memoryTypes, "type", "t", []string{}, "Filter by document type (code_change, specification, session, solution)")
	memoryCmd.Flags().StringVar(&memoryTaskID, "task", "", "Filter by task ID")
	memoryCmd.Flags().StringVar(&memoryIndexTask, "index", "", "Index a specific task")
	memoryCmd.Flags().BoolVar(&memoryStats, "stats", false, "Show memory statistics")
	memoryCmd.Flags().BoolVar(&memoryClear, "clear", false, "Clear all memory")
}

func runMemory(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	// Initialize conductor
	opts := BuildConductorOptions(CommandOptions{
		Verbose: verbose,
	})
	cond, err := initializeConductor(ctx, opts...)
	if err != nil {
		return err
	}

	// Get workspace config
	cfg, err := cond.GetWorkspace().LoadConfig()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// Check if memory is enabled
	if cfg.Memory == nil || !cfg.Memory.Enabled {
		return errors.New("memory system is not enabled. Enable it in .mehrhof/config.yaml with memory.enabled: true")
	}

	// Get memory system from a conductor
	mem := cond.GetMemory()
	if mem == nil {
		return errors.New("memory system is not available. Ensure memory is enabled in config")
	}

	ws := cond.GetWorkspace()

	// Handle different commands
	switch {
	case memoryClear:
		return clearMemory(ctx, mem)
	case memoryStats:
		return showMemoryStats(ctx, mem, ws)
	case memoryIndexTask != "":
		return indexTask(ctx, mem, ws, memoryIndexTask)
	case memoryQuery != "":
		return searchMemory(ctx, mem, memoryQuery)
	default:
		// Show stats by default
		return showMemoryStats(ctx, mem, ws)
	}
}

// clearMemory clears all stored memory.
func clearMemory(ctx context.Context, mem *memory.MemorySystem) error {
	fmt.Print("Are you sure you want to clear all memory? This cannot be undone. [y/N]: ")

	// Use bufio for robust input handling
	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("read input: %w", err)
	}

	// Accept y, yes, Y, YES
	response = strings.ToLower(strings.TrimSpace(response))
	if response != "y" && response != "yes" {
		fmt.Println("Cancelled.")

		return nil
	}

	if err := mem.Clear(ctx); err != nil {
		return fmt.Errorf("clear memory: %w", err)
	}

	fmt.Println("Memory cleared successfully.")

	return nil
}

// showMemoryStats displays memory statistics.
func showMemoryStats(ctx context.Context, mem *memory.MemorySystem, ws *storage.Workspace) error {
	// Create an indexer to get stats
	indexer := memory.NewIndexer(mem, ws, nil)
	stats, err := indexer.GetStats(ctx)
	if err != nil {
		return fmt.Errorf("get stats: %w", err)
	}

	fmt.Println("\n=== Memory Statistics ===")
	fmt.Printf("Total Documents: %d\n", stats.TotalDocuments)

	if len(stats.ByType) > 0 {
		fmt.Println("\nDocuments by Type:")
		// Sort by type for consistent output
		types := make([]string, 0, len(stats.ByType))
		for t := range stats.ByType {
			types = append(types, t)
		}
		sort.Strings(types)

		for _, t := range types {
			fmt.Printf("  %s: %d\n", t, stats.ByType[t])
		}
	}

	fmt.Println("\n=== Configuration ===")
	fmt.Println("Vector Store: ChromaDB (persistent disk storage)")

	return nil
}

// indexTask indexes a specific task.
func indexTask(ctx context.Context, mem *memory.MemorySystem, ws *storage.Workspace, taskID string) error {
	// Verify task exists
	work, err := ws.LoadWork(taskID)
	if err != nil {
		return fmt.Errorf("load task: %w", err)
	}

	fmt.Printf("Indexing task %s: %s\n\n", taskID, work.Metadata.Title)

	// Create indexer
	indexer := memory.NewIndexer(mem, ws, nil)

	// Index the task
	if err := indexer.IndexTask(ctx, taskID); err != nil {
		return fmt.Errorf("index task: %w", err)
	}

	fmt.Println("Task indexed successfully.")
	fmt.Printf("Indexed specifications, code changes, and session logs.\n")

	return nil
}

// searchMemory searches for similar tasks.
func searchMemory(ctx context.Context, mem *memory.MemorySystem, query string) error {
	fmt.Printf("Searching for: %s\n\n", query)

	// Parse document types
	var docTypes []memory.DocumentType
	for _, t := range memoryTypes {
		switch strings.ToLower(t) {
		case "code_change", "code":
			docTypes = append(docTypes, memory.TypeCodeChange)
		case "specification", "spec":
			docTypes = append(docTypes, memory.TypeSpecification)
		case "session":
			docTypes = append(docTypes, memory.TypeSession)
		case "solution":
			docTypes = append(docTypes, memory.TypeSolution)
		case "decision":
			docTypes = append(docTypes, memory.TypeDecision)
		case "error":
			docTypes = append(docTypes, memory.TypeError)
		}
	}

	// Search memory
	results, err := mem.Search(ctx, query, memory.SearchOptions{
		Limit:         memoryLimit,
		MinScore:      0.65,
		DocumentTypes: docTypes,
	})
	if err != nil {
		return fmt.Errorf("search memory: %w", err)
	}

	if len(results) == 0 {
		fmt.Println("No similar tasks found.")

		return nil
	}

	fmt.Printf("Found %d similar task(s):\n\n", len(results))

	for i, result := range results {
		doc := result.Document
		fmt.Printf("%d. Task %s (Similarity: %.0f%%)\n", i+1, doc.TaskID, result.Score*100)
		fmt.Printf("   Type: %s\n", doc.Type)

		// Show metadata
		if len(doc.Metadata) > 0 {
			fmt.Print("   Metadata: ")
			var meta []string
			for k, v := range doc.Metadata {
				meta = append(meta, fmt.Sprintf("%s=%v", k, v))
			}
			fmt.Printf("%s\n", strings.Join(meta, ", "))
		}

		// Show content preview
		preview := doc.Content
		maxLen := 300
		if len(preview) > maxLen {
			preview = preview[:maxLen] + "..."
		}
		fmt.Printf("\n   Content:\n   %s\n\n", indentText(preview, "   "))
	}

	return nil
}

// indentText adds indentation to each line of text.
func indentText(text, indent string) string {
	lines := strings.Split(text, "\n")
	for i := range lines {
		lines[i] = indent + lines[i]
	}

	return strings.Join(lines, "\n")
}
