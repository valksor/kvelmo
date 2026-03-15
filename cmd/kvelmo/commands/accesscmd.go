package commands

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/valksor/kvelmo/pkg/access"
)

var (
	accessTokenRole  string
	accessTokenLabel string
)

var AccessCmd = &cobra.Command{
	Use:   "access",
	Short: "Socket access token management",
	Long:  "Create, revoke, and list access tokens for optional socket authentication.",
}

var accessTokenCmd = &cobra.Command{
	Use:   "token",
	Short: "Manage access tokens",
}

var accessTokenCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new access token",
	RunE:  runAccessTokenCreate,
}

var accessTokenRevokeCmd = &cobra.Command{
	Use:   "revoke <id>",
	Short: "Revoke an access token",
	Args:  cobra.ExactArgs(1),
	RunE:  runAccessTokenRevoke,
}

var accessTokenListCmd = &cobra.Command{
	Use:   "list",
	Short: "List access tokens",
	RunE:  runAccessTokenList,
}

func init() {
	AccessCmd.AddCommand(accessTokenCmd)
	accessTokenCmd.AddCommand(accessTokenCreateCmd)
	accessTokenCmd.AddCommand(accessTokenRevokeCmd)
	accessTokenCmd.AddCommand(accessTokenListCmd)

	accessTokenCreateCmd.Flags().StringVar(&accessTokenRole, "role", "operator", "Token role (operator or viewer)")
	accessTokenCreateCmd.Flags().StringVar(&accessTokenLabel, "label", "", "Human-readable label for the token")
}

func runAccessTokenCreate(_ *cobra.Command, _ []string) error {
	store := access.New("")
	plaintext, err := store.Create(access.Role(accessTokenRole), accessTokenLabel, nil)
	if err != nil {
		return fmt.Errorf("create token: %w", err)
	}

	fmt.Printf("Token created: %s\n", plaintext)
	fmt.Println("Store this token securely — it cannot be retrieved later.")

	return nil
}

func runAccessTokenRevoke(_ *cobra.Command, args []string) error {
	store := access.New("")
	if err := store.Revoke(args[0]); err != nil {
		return fmt.Errorf("revoke: %w", err)
	}

	fmt.Println("Token revoked.")

	return nil
}

func runAccessTokenList(_ *cobra.Command, _ []string) error {
	store := access.New("")
	tokens, err := store.List()
	if err != nil {
		return fmt.Errorf("list: %w", err)
	}

	if len(tokens) == 0 {
		fmt.Println("No access tokens configured.")

		return nil
	}

	data, err := json.MarshalIndent(tokens, "", "  ")
	if err != nil {
		return fmt.Errorf("format: %w", err)
	}

	fmt.Println(string(data))

	return nil
}
