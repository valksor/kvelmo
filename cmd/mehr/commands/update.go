package commands

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/valksor/go-mehrhof/internal/provider/github"
	"github.com/valksor/go-mehrhof/internal/update"
	"github.com/valksor/go-toolkit/display"
)

var (
	updateNightly   bool
	updateCheckOnly bool
	updateYes       bool
	updateVersion   string
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update Mehrhof to the latest version",
	Long: `Update Mehrhof to the latest version from GitHub releases.

By default, only stable releases are considered. Use --nightly to include
nightly/pre-release versions.

The update process:
1. Checks for the latest release
2. Downloads the checksums file and verifies its signature (if available)
3. Downloads the binary for your platform
4. Verifies SHA256 checksum
5. Replaces the current binary atomically

After a successful update, restart mehr to use the new version.`,
	RunE: runUpdate,
}

func init() {
	rootCmd.AddCommand(updateCmd)
	updateCmd.Flags().BoolVarP(&updateNightly, "nightly", "n", false,
		"Install latest available release including pre-releases")
	updateCmd.Flags().StringVarP(&updateVersion, "version", "v", "",
		"Install specific version tag (e.g. v1.2.3)")
	updateCmd.Flags().BoolVar(&updateCheckOnly, "check", false,
		"Check for updates without installing")
	updateCmd.Flags().BoolVarP(&updateYes, "yes", "y", false,
		"Skip confirmation prompt")
}

func runUpdate(cmd *cobra.Command, args []string) error {
	if updateNightly && updateVersion != "" {
		return errors.New("--nightly and --version are mutually exclusive")
	}

	// Resolve GitHub token (anonymous access works for public repos)
	token, err := github.ResolveToken("")
	if err != nil {
		// Continue without token - may hit rate limits but works for public repos
		fmt.Fprintf(os.Stderr, "%s Running without authentication (rate limits may apply)\n",
			display.Warning("→"))
	}

	targetTag := updateVersion

	opts := update.CheckOptions{
		CurrentVersion: Version,
		IncludeNightly: updateNightly,
		TargetTag:      targetTag,
	}

	// Show checking message
	fmt.Println(display.Info("→") + " Checking for updates...")

	// Check for updates
	ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
	defer cancel()

	// Create a checker
	checker := update.NewChecker(ctx, token, "valksor", "go-mehrhof")

	status, err := checker.Check(ctx, opts)
	if err != nil {
		if errors.Is(err, update.ErrNoUpdateAvailable) {
			display.Success("Already up to date")
			fmt.Printf("Current version: %s\n", Version)

			return nil
		}
		if errors.Is(err, update.ErrReleaseNotFound) {
			return fmt.Errorf("requested release not found: %w", err)
		}

		return fmt.Errorf("check for updates: %w", err)
	}

	// Display update available
	fmt.Printf("\n%s %s\n", display.Success("✓"), display.Bold("Update available"))
	fmt.Printf("  Current:   %s\n", display.Muted(status.CurrentVersion))
	fmt.Printf("  Latest:    %s\n", display.Success(status.LatestVersion))
	if status.ReleaseURL != "" {
		fmt.Printf("  Release:   %s\n", display.Muted(status.ReleaseURL))
	}
	if status.AssetSize > 0 {
		sizeMB := float64(status.AssetSize) / 1024 / 1024
		fmt.Printf("  Download:  %s (%.1f MB)\n", display.Muted(status.AssetName), sizeMB)
	}

	if updateCheckOnly {
		return nil
	}

	// Confirm before downloading
	if !updateYes {
		prompt := fmt.Sprintf("Download and install %s?", status.LatestVersion)
		confirmed, err := confirmAction(prompt, false)
		if err != nil {
			return err
		}
		if !confirmed {
			fmt.Println(display.Muted("Update cancelled"))

			return nil
		}
	}

	// Check if writable
	installer := update.NewInstaller()
	writable, _ := installer.IsWritable()
	if !writable {
		return fmt.Errorf("%s\n\nTry running with sudo: sudo mehr update", display.ErrorMsg(
			"Cannot write to binary directory"))
	}

	// Download the update with signature verification
	downloader := update.NewDownloader()
	spinner := display.NewSpinner("Downloading and verifying update")
	spinner.Start()

	// Construct URLs for checksums and signature
	checksumsURL, signatureURL := getReleaseURLs(status)

	downloadedPath, verifyResult, err := downloader.DownloadWithSignature(
		ctx,
		status.AssetURL,
		checksumsURL,
		signatureURL,
		status.AssetName,
		update.MinisignPublicKey,
	)
	if err != nil {
		// Check if it's a signature verification failure
		if errors.Is(err, update.ErrSignatureVerificationFailed) {
			spinner.StopWithError("Signature verification failed")
			fmt.Printf("\n%s %s\n", display.ErrorMsg("✗"),
				"The checksums file signature is invalid. This may indicate tampering.")
			fmt.Println("Update aborted for security. Please report this issue.")

			return err
		}
		spinner.StopWithError(fmt.Sprintf("Download failed: %v", err))

		return err
	}

	spinner.StopWithSuccess("Download complete")

	// Show verification status
	if verifyResult.SignatureVerified {
		fmt.Printf("%s Signature verified\n", display.Success("✓"))
	} else if verifyResult.SignatureSkipped {
		fmt.Printf("%s Signature verification skipped: %s\n",
			display.Warning("→"), verifyResult.SignatureError)
	}
	if verifyResult.ChecksumVerified {
		fmt.Printf("%s Checksum verified\n", display.Success("✓"))
	}

	// Install the update
	spinner = display.NewSpinner("Installing update")
	spinner.Start()

	if err := installer.Install(downloadedPath); err != nil {
		spinner.StopWithError(fmt.Sprintf("Installation failed: %v", err))

		return err
	}

	spinner.StopWithSuccess("Installation complete")

	fmt.Printf("\n%s Updated to %s\n", display.SuccessMsg(""), display.Bold(status.LatestVersion))
	fmt.Printf("%s Restart mehr to use the new version\n\n", display.Muted("→"))

	return nil
}

// getReleaseURLs returns the checksums and signature URLs for a release.
// GitHub releases follow: /owner/repo/releases/download/tag/asset.
func getReleaseURLs(status *update.UpdateStatus) (string, string) {
	tag := status.LatestVersion
	baseURL := "https://github.com/valksor/go-mehrhof/releases/download/" + tag

	return baseURL + "/checksums.txt", baseURL + "/checksums.txt.minisig"
}
