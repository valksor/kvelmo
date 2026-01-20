package commands

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"github.com/spf13/cobra"
)

var generateSecretCmd = &cobra.Command{
	Use:   "generate-secret",
	Short: "Generate a cryptographically secure secret for PR review state signing",
	Long: `Generate a random 32-byte secret for MEHRHOF_STATE_SECRET.

This secret is used to HMAC-sign review state embedded in PR comments,
preventing tampering with review history.

Example:
    mehr generate-secret
    export MEHRHOF_STATE_SECRET="$(mehr generate-secret)"`,
	RunE: runGenerateSecret,
}

func init() {
	rootCmd.AddCommand(generateSecretCmd)
}

func runGenerateSecret(cmd *cobra.Command, args []string) error {
	// Generate 32 random bytes (256 bits)
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return fmt.Errorf("generate random bytes: %w", err)
	}

	// Encode to base64 for easy use in environment variables
	secret := base64.StdEncoding.EncodeToString(b)

	// Output instructions
	fmt.Println("# Add this to your environment or CI/CD secrets:")
	fmt.Printf("export MEHRHOF_STATE_SECRET=\"%s\"\n", secret)
	fmt.Println("\n# Or in GitHub Actions / GitLab CI:")
	fmt.Printf("MEHRHOF_STATE_SECRET: %s\n", secret)

	return nil
}
