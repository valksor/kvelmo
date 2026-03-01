package commands

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/valksor/kvelmo/pkg/meta"
	"github.com/valksor/kvelmo/pkg/socket"
)

var ConfigCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage " + meta.Name + " configuration",
	Long:  "View and modify " + meta.Name + " configuration settings.",
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current effective configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, ctx, cancel, err := globalSocketClient()
		if err != nil {
			return err
		}
		defer func() { _ = client.Close() }()
		defer cancel()

		resp, err := client.Call(ctx, "settings.get", nil)
		if err != nil {
			return fmt.Errorf("settings.get: %w", err)
		}

		var result struct {
			Effective json.RawMessage `json:"effective"`
		}
		if err := json.Unmarshal(resp.Result, &result); err != nil {
			return fmt.Errorf("parse response: %w", err)
		}

		pretty, err := json.MarshalIndent(result.Effective, "", "  ")
		if err != nil {
			return fmt.Errorf("format: %w", err)
		}
		fmt.Println(string(pretty))

		return nil
	},
}

var configPathCmd = &cobra.Command{
	Use:   "path",
	Short: "Show global configuration file path",
	Run: func(cmd *cobra.Command, args []string) {
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintln(os.Stderr, "cannot determine home directory:", err)

			return
		}
		fmt.Println(filepath.Join(home, meta.GlobalDir, meta.ConfigFile))
	},
}

var configGetCmd = &cobra.Command{
	Use:   "get <path>",
	Short: "Get a configuration value (e.g. workers.max, git.branch_pattern)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		path := args[0]

		client, ctx, cancel, err := globalSocketClient()
		if err != nil {
			return err
		}
		defer func() { _ = client.Close() }()
		defer cancel()

		resp, err := client.Call(ctx, "settings.get", nil)
		if err != nil {
			return fmt.Errorf("settings.get: %w", err)
		}

		var result struct {
			Effective map[string]any `json:"effective"`
		}
		if err := json.Unmarshal(resp.Result, &result); err != nil {
			return fmt.Errorf("parse response: %w", err)
		}

		value, err := nestedGet(result.Effective, path)
		if err != nil {
			return err
		}

		switch v := value.(type) {
		case string:
			fmt.Println(v)
		case bool:
			fmt.Println(v)
		default:
			data, err := json.Marshal(value)
			if err != nil {
				return fmt.Errorf("marshal value: %w", err)
			}
			fmt.Println(string(data))
		}

		return nil
	},
}

var configSetCmd = &cobra.Command{
	Use:   "set <path> <value>",
	Short: "Set a global configuration value (e.g. workers.max 5, git.auto_commit true)",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		path, raw := args[0], args[1]

		client, ctx, cancel, err := globalSocketClient()
		if err != nil {
			return err
		}
		defer func() { _ = client.Close() }()
		defer cancel()

		// Coerce the value: try JSON decode first (handles numbers and booleans),
		// then fall back to treating the raw string as a plain string value.
		var value any
		if err := json.Unmarshal([]byte(raw), &value); err != nil {
			value = raw
		}

		params := map[string]any{
			"scope":  "global",
			"values": map[string]any{path: value},
		}
		if _, err := client.Call(ctx, "settings.set", params); err != nil {
			return fmt.Errorf("settings.set: %w", err)
		}

		fmt.Printf("Set %s = %s\n", path, raw)

		return nil
	},
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize global configuration file",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, ctx, cancel, err := globalSocketClient()
		if err != nil {
			return err
		}
		defer func() { _ = client.Close() }()
		defer cancel()

		params := map[string]any{
			"scope":  "global",
			"values": map[string]any{},
		}
		if _, err := client.Call(ctx, "settings.set", params); err != nil {
			return fmt.Errorf("settings.set: %w", err)
		}

		home, _ := os.UserHomeDir()
		fmt.Printf("Configuration initialized at %s\n", filepath.Join(home, meta.GlobalDir, meta.ConfigFile))

		return nil
	},
}

// globalSocketClient connects to the global socket and returns a client, context, and cancel func.
func globalSocketClient() (*socket.Client, context.Context, context.CancelFunc, error) {
	gPath := socket.GlobalSocketPath()
	if !socket.SocketExists(gPath) {
		return nil, nil, nil, errors.New(meta.Name + " server not running\nRun '" + meta.Name + " serve' first")
	}
	client, err := socket.NewClient(gPath, socket.WithTimeout(5*time.Second))
	if err != nil {
		return nil, nil, nil, fmt.Errorf("connect to server: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

	return client, ctx, cancel, nil
}

// nestedGet extracts a value from a nested map using a dot-notation path.
func nestedGet(m map[string]any, path string) (any, error) {
	parts := strings.SplitN(path, ".", 2)
	val, ok := m[parts[0]]
	if !ok {
		return nil, fmt.Errorf("unknown configuration path: %s", path)
	}
	if len(parts) == 1 {
		return val, nil
	}
	nested, ok := val.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("unknown configuration path: %s", path)
	}

	return nestedGet(nested, parts[1])
}

func init() {
	ConfigCmd.AddCommand(configShowCmd)
	ConfigCmd.AddCommand(configPathCmd)
	ConfigCmd.AddCommand(configInitCmd)
	ConfigCmd.AddCommand(configSetCmd)
	ConfigCmd.AddCommand(configGetCmd)
}
