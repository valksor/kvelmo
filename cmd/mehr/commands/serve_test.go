//go:build !testbinary

package commands

import (
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
