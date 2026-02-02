package mcp

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCallToolTimeout(t *testing.T) {
	rootCmd := &cobra.Command{Use: "root"}
	slowCmd := &cobra.Command{
		Use:   "slow",
		Short: "A slow command",
		RunE: func(cmd *cobra.Command, args []string) error {
			select {
			case <-time.After(5 * time.Second):
				return nil
			case <-cmd.Context().Done():
				return cmd.Context().Err()
			}
		},
	}
	rootCmd.AddCommand(slowCmd)

	registry := NewToolRegistry(rootCmd)
	registry.RegisterCommand(slowCmd, DefaultArgMapper)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	result, err := registry.CallTool(ctx, "slow", map[string]interface{}{})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content[0].Text, "canceled")
}

func TestCallToolCobraError(t *testing.T) {
	tests := []struct {
		name        string
		cmd         *cobra.Command
		wantIsError bool
		wantText    string
	}{
		{
			name: "command returns error",
			cmd: &cobra.Command{
				Use:   "fail",
				Short: "A failing command",
				RunE: func(cmd *cobra.Command, args []string) error {
					return errors.New("command failed")
				},
			},
			wantIsError: true,
			wantText:    "command failed",
		},
		{
			name: "command panics",
			cmd: &cobra.Command{
				Use:   "panicker",
				Short: "A panicking command",
				Run: func(cmd *cobra.Command, args []string) {
					panic("test panic")
				},
			},
			wantIsError: true,
			wantText:    "panic",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rootCmd := &cobra.Command{Use: "root"}
			rootCmd.AddCommand(tt.cmd)

			registry := NewToolRegistry(rootCmd)
			registry.RegisterCommand(tt.cmd, DefaultArgMapper)

			result, err := registry.CallTool(context.Background(), tt.cmd.Use, map[string]interface{}{})
			require.NoError(t, err)
			require.NotNil(t, result)
			assert.True(t, result.IsError, "expected IsError to be true")
			assert.Contains(t, result.Content[0].Text, tt.wantText)
		})
	}
}

func TestFilterTools(t *testing.T) {
	tests := []struct {
		name      string
		tools     []string
		allowList []string
		wantCount int
	}{
		{
			name:      "filter to subset",
			tools:     []string{"tool_a", "tool_b", "tool_c"},
			allowList: []string{"tool_a", "tool_b"},
			wantCount: 2,
		},
		{
			name:      "empty allowlist keeps all",
			tools:     []string{"tool_a", "tool_b", "tool_c"},
			allowList: []string{},
			wantCount: 3,
		},
		{
			name:      "filter to empty",
			tools:     []string{"tool_a", "tool_b"},
			allowList: []string{"nonexistent"},
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rootCmd := &cobra.Command{Use: "root"}
			registry := NewToolRegistry(rootCmd)

			for _, name := range tt.tools {
				cmd := &cobra.Command{
					Use:   name,
					Short: "Test tool " + name,
					Run:   func(cmd *cobra.Command, args []string) {},
				}
				rootCmd.AddCommand(cmd)
				registry.RegisterCommand(cmd, DefaultArgMapper)
			}

			registry.FilterTools(tt.allowList)
			assert.Len(t, registry.ListTools(), tt.wantCount)
		})
	}
}

func TestCaptureStdout(t *testing.T) {
	tests := []struct {
		name string
		fn   func()
		want string
	}{
		{
			name: "captures fmt.Printf output",
			fn: func() {
				fmt.Printf("hello from printf\n")
			},
			want: "hello from printf\n",
		},
		{
			name: "returns empty when no output",
			fn:   func() {},
			want: "",
		},
		{
			name: "captures multiple writes",
			fn: func() {
				fmt.Print("one ")
				fmt.Print("two ")
				fmt.Println("three")
			},
			want: "one two three\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := captureStdout(context.Background(), tt.fn)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCaptureStdoutPanic(t *testing.T) {
	// Verify stdout is restored after panic inside captureStdout.
	// The panic should propagate (not be swallowed), and os.Stdout must
	// point back to the original fd, not the capture pipe.
	origStdout := os.Stdout

	assert.Panics(t, func() {
		_, _ = captureStdout(context.Background(), func() {
			fmt.Printf("before panic\n")
			panic("test panic in captureStdout")
		})
	})

	assert.Equal(t, origStdout, os.Stdout, "stdout must be restored after panic")
}

func TestCaptureStdoutWithCobra(t *testing.T) {
	// Verify that captureStdout works end-to-end with CallTool:
	// a command using fmt.Printf should have its output in the MCP result.
	rootCmd := &cobra.Command{Use: "root"}
	printCmd := &cobra.Command{
		Use:   "printer",
		Short: "Prints via fmt.Printf",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("captured output\n")
		},
	}
	rootCmd.AddCommand(printCmd)

	registry := NewToolRegistry(rootCmd)
	registry.RegisterCommand(printCmd, DefaultArgMapper)

	result, err := registry.CallTool(context.Background(), "printer", map[string]interface{}{})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.IsError)
	assert.Contains(t, result.Content[0].Text, "captured output")
}
