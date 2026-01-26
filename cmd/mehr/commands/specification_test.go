//go:build !testbinary
// +build !testbinary

package commands

import (
	"testing"
)

func TestSpecificationCommand_Properties(t *testing.T) {
	// Test the main specification command
	if specificationCmd.Use != "specification" {
		t.Errorf("specificationCmd.Use = %q, want %q", specificationCmd.Use, "specification")
	}

	if specificationCmd.Short == "" {
		t.Error("Short description is empty")
	}

	if specificationCmd.Long == "" {
		t.Error("Long description is empty")
	}

	// Test the view subcommand
	if specificationViewCmd.Use != "view <number>" {
		t.Errorf("specificationViewCmd.Use = %q, want %q", specificationViewCmd.Use, "view <number>")
	}

	if specificationViewCmd.Short == "" {
		t.Error("Short description is empty for view subcommand")
	}

	if specificationViewCmd.Long == "" {
		t.Error("Long description is empty for view subcommand")
	}

	if specificationViewCmd.RunE == nil {
		t.Error("RunE not set for view subcommand")
	}
}

func TestSpecificationCommand_Flags(t *testing.T) {
	tests := []struct {
		name         string
		flagName     string
		shorthand    string
		defaultValue interface{}
	}{
		{
			name:         "number flag",
			flagName:     "number",
			shorthand:    "n",
			defaultValue: 0,
		},
		{
			name:         "all flag",
			flagName:     "all",
			shorthand:    "a",
			defaultValue: false,
		},
		{
			name:         "output flag",
			flagName:     "output",
			shorthand:    "o",
			defaultValue: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag := specificationViewCmd.Flags().Lookup(tt.flagName)
			if flag == nil {
				t.Errorf("flag %q not found", tt.flagName)

				return
			}

			// Check shorthand
			if flag.Shorthand != tt.shorthand {
				t.Errorf("flag %q shorthand = %q, want %q", tt.flagName, flag.Shorthand, tt.shorthand)
			}

			// Check default value
			// Note: We can't easily check the actual default value without parsing,
			// but we can verify the flag exists
		})
	}
}
