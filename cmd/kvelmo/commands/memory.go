package commands

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/valksor/kvelmo/pkg/meta"
	"github.com/valksor/kvelmo/pkg/socket"
)

// MemoryCmd is the root command for memory subcommands.
var MemoryCmd = &cobra.Command{
	Use:   "memory",
	Short: "Manage semantic memory",
	Long:  "Search, inspect, and clear the semantic memory store used to augment agent prompts.",
}

var memorySearchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search semantic memory",
	Long:  "Search stored task context for documents similar to the given query.",
	Args:  cobra.ExactArgs(1),
	RunE:  runMemorySearch,
}

var memoryStatsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show memory store statistics",
	Long:  "Display the number of documents stored in the semantic memory store.",
	RunE:  runMemoryStats,
}

var memoryClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Clear all memory",
	Long:  "Remove all documents from the semantic memory store.",
	RunE:  runMemoryClear,
}

var (
	memorySearchJSON bool
	memoryStatsJSON  bool
)

func init() {
	MemoryCmd.AddCommand(memorySearchCmd)
	MemoryCmd.AddCommand(memoryStatsCmd)
	MemoryCmd.AddCommand(memoryClearCmd)

	memorySearchCmd.Flags().IntP("limit", "n", 10, "Maximum number of results")
	memorySearchCmd.Flags().Float32P("min-score", "s", 0.0, "Minimum similarity score (0-1)")
	memorySearchCmd.Flags().StringSliceP("types", "t", nil, "Filter by document type (specification,code_change,session,decision,solution)")
	memorySearchCmd.Flags().BoolVar(&memorySearchJSON, "json", false, "Output raw JSON response")
	memoryStatsCmd.Flags().BoolVar(&memoryStatsJSON, "json", false, "Output raw JSON response")
}

func runMemorySearch(cmd *cobra.Command, args []string) error {
	query := args[0]

	globalPath := socket.GlobalSocketPath()
	if !socket.SocketExists(globalPath) {
		return errors.New("global socket not running\nRun '" + meta.Name + " serve' first")
	}

	client, err := socket.NewClient(globalPath, socket.WithTimeout(10*time.Second))
	if err != nil {
		return fmt.Errorf("connect to global socket: %w", err)
	}
	defer func() { _ = client.Close() }()

	limit, _ := cmd.Flags().GetInt("limit")
	minScore, _ := cmd.Flags().GetFloat32("min-score")
	types, _ := cmd.Flags().GetStringSlice("types")

	params := map[string]any{
		"query":     query,
		"limit":     limit,
		"min_score": minScore,
	}
	if len(types) > 0 {
		params["document_types"] = types
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := client.Call(ctx, "memory.search", params)
	if err != nil {
		return fmt.Errorf("memory.search: %w", err)
	}

	if memorySearchJSON {
		var pretty any
		if jsonErr := json.Unmarshal(resp.Result, &pretty); jsonErr != nil {
			fmt.Println(string(resp.Result))

			return nil
		}
		out, jsonErr := json.MarshalIndent(pretty, "", "  ")
		if jsonErr != nil {
			fmt.Println(string(resp.Result))

			return nil
		}
		fmt.Println(string(out))

		return nil
	}

	var result struct {
		Results []struct {
			ID      string  `json:"id"`
			TaskID  string  `json:"task_id"`
			Type    string  `json:"type"`
			Content string  `json:"content"`
			Score   float32 `json:"score"`
		} `json:"results"`
		Total int `json:"total"`
	}
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return fmt.Errorf("parse result: %w", err)
	}

	if result.Total == 0 {
		fmt.Println("No results found.")

		return nil
	}

	fmt.Printf("Found %d result(s) for %q:\n\n", result.Total, query)
	for i, hit := range result.Results {
		fmt.Printf("%d. [%.2f] %s (%s)\n", i+1, hit.Score, hit.TaskID, hit.Type)
		fmt.Printf("   ID: %s\n", hit.ID)
		if hit.Content != "" {
			preview := hit.Content
			if len(preview) > 200 {
				preview = preview[:200] + "..."
			}
			fmt.Printf("   %s\n", preview)
		}
		fmt.Println()
	}

	return nil
}

func runMemoryStats(cmd *cobra.Command, args []string) error {
	globalPath := socket.GlobalSocketPath()
	if !socket.SocketExists(globalPath) {
		return errors.New("global socket not running\nRun '" + meta.Name + " serve' first")
	}

	client, err := socket.NewClient(globalPath, socket.WithTimeout(5*time.Second))
	if err != nil {
		return fmt.Errorf("connect to global socket: %w", err)
	}
	defer func() { _ = client.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := client.Call(ctx, "memory.stats", nil)
	if err != nil {
		return fmt.Errorf("memory.stats: %w", err)
	}

	if memoryStatsJSON {
		var pretty any
		if jsonErr := json.Unmarshal(resp.Result, &pretty); jsonErr != nil {
			fmt.Println(string(resp.Result))

			return nil
		}
		out, jsonErr := json.MarshalIndent(pretty, "", "  ")
		if jsonErr != nil {
			fmt.Println(string(resp.Result))

			return nil
		}
		fmt.Println(string(out))

		return nil
	}

	var result struct {
		TotalDocuments int            `json:"total_documents"`
		ByType         map[string]int `json:"by_type"`
		Embedder       string         `json:"embedder"`
	}
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return fmt.Errorf("parse result: %w", err)
	}

	fmt.Printf("Memory Store Statistics\n")
	fmt.Printf("  Total documents: %d\n", result.TotalDocuments)
	if result.Embedder != "" {
		fmt.Printf("  Embedder:        %s\n", result.Embedder)
	}
	if len(result.ByType) > 0 {
		fmt.Println("  By type:")
		for t, count := range result.ByType {
			fmt.Printf("    %-20s %d\n", t, count)
		}
	}

	return nil
}

func runMemoryClear(cmd *cobra.Command, args []string) error {
	globalPath := socket.GlobalSocketPath()
	if !socket.SocketExists(globalPath) {
		return errors.New("global socket not running\nRun '" + meta.Name + " serve' first")
	}

	client, err := socket.NewClient(globalPath, socket.WithTimeout(5*time.Second))
	if err != nil {
		return fmt.Errorf("connect to global socket: %w", err)
	}
	defer func() { _ = client.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := client.Call(ctx, "memory.clear", nil)
	if err != nil {
		return fmt.Errorf("memory.clear: %w", err)
	}

	var result struct {
		OK bool `json:"ok"`
	}
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return fmt.Errorf("parse result: %w", err)
	}

	if result.OK {
		fmt.Println("Memory store cleared.")
	} else {
		fmt.Println("Clear operation returned false.")
	}

	return nil
}
