package commands

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/valksor/go-mehrhof/internal/library"
	"github.com/valksor/go-toolkit/display"
)

var (
	// Pull flags.
	libraryPullName          string
	libraryPullMode          string
	libraryPullShared        bool
	libraryPullPaths         []string
	libraryPullTags          []string
	libraryPullMaxDepth      int
	libraryPullMaxPages      int
	libraryPullDryRun        bool
	libraryPullGitRef        string
	libraryPullGitPath       string
	libraryPullForce         bool
	libraryPullContinue      bool
	libraryPullRestart       bool
	libraryPullDomainScope   string
	libraryPullVersionFilter bool
	libraryPullVersion       string

	// Update flags.
	libraryUpdateFull bool

	// List flags.
	libraryListShared  bool
	libraryListProject bool
	libraryListVerbose bool
	libraryListTag     string
	libraryListMode    string

	// Remove flags.
	libraryRemoveForce bool
)

var libraryCmd = &cobra.Command{
	Use:   "library [list|show|search|pull|remove|update]",
	Short: "Manage documentation library collections",
	Long: `Manage documentation collections that can be automatically included in AI prompts.

The library system allows you to pull documentation from:
  - URLs (with optional crawling)
  - Local files and directories
  - Git repositories

Collections can be configured to auto-include when working on matching file paths,
or explicitly requested by name.

Examples:
  mehr library pull https://go.dev/doc/effective_go --name "Effective Go"
  mehr library pull ./docs --name "Project Docs" --paths "src/**"
  mehr library list
  mehr library show "Effective Go"
  mehr library remove "Effective Go"`,
}

var libraryPullCmd = &cobra.Command{
	Use:   "pull <source>",
	Short: "Pull documentation from a source",
	Long: `Pull documentation from a URL, local path, or git repository.

SOURCE TYPES:
  URL:   https://docs.example.com/guide
         Will crawl and convert HTML to markdown.

  File:  ./docs/guide.md or /path/to/docs/
         Will copy local markdown/text files.

  Git:   git@github.com:org/repo.git or https://github.com/org/repo
         Will clone and extract documentation.

INCLUDE MODES:
  auto      - Include when file paths match patterns (default)
  explicit  - Only include when explicitly requested by name
  always    - Always include in prompts

Examples:
  mehr library pull https://react.dev/reference --name "React API"
  mehr library pull ./vendor/docs --name "Vendor Docs" --mode explicit
  mehr library pull https://go.dev/doc --name "Go Docs" --max-pages 50
  mehr library pull git@github.com:org/docs.git --name "Internal Docs" --git-ref main --git-path docs/`,
	Args: cobra.ExactArgs(1),
	RunE: runLibraryPull,
}

var libraryListCmd = &cobra.Command{
	Use:   "list",
	Short: "List documentation collections",
	Long: `List all documentation collections in the library.

By default, shows both project-local and shared collections.
Use --project or --shared to filter.

Examples:
  mehr library list
  mehr library list --verbose
  mehr library list --shared
  mehr library list --tag api`,
	Args: cobra.NoArgs,
	RunE: runLibraryList,
}

var libraryShowCmd = &cobra.Command{
	Use:   "show <name> [page]",
	Short: "Show collection details or page content",
	Long: `Show details about a documentation collection or view a specific page.

Without a page argument, shows collection metadata.
With a page argument, shows the page content.

Examples:
  mehr library show "React API"
  mehr library show "React API" hooks/usestate.md`,
	Args: cobra.RangeArgs(1, 2),
	RunE: runLibraryShow,
}

var libraryRemoveCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove a documentation collection",
	Long: `Remove a documentation collection from the library.

This permanently deletes the collection and all its pages.

Examples:
  mehr library remove "Old Docs"
  mehr library remove "Old Docs" --force`,
	Args: cobra.ExactArgs(1),
	RunE: runLibraryRemove,
}

var libraryUpdateCmd = &cobra.Command{
	Use:   "update [name]",
	Short: "Update documentation from source",
	Long: `Re-pull documentation from its original source.

Without a name, updates all collections that have a remote source.
With a name, updates only that collection.

Examples:
  mehr library update "React API"
  mehr library update`,
	Args: cobra.MaximumNArgs(1),
	RunE: runLibraryUpdate,
}

func init() {
	rootCmd.AddCommand(libraryCmd)

	// Add subcommands
	libraryCmd.AddCommand(libraryPullCmd)
	libraryCmd.AddCommand(libraryListCmd)
	libraryCmd.AddCommand(libraryShowCmd)
	libraryCmd.AddCommand(libraryRemoveCmd)
	libraryCmd.AddCommand(libraryUpdateCmd)

	// Pull flags
	libraryPullCmd.Flags().StringVarP(&libraryPullName, "name", "n", "", "Collection name (auto-generated if empty)")
	libraryPullCmd.Flags().StringVarP(&libraryPullMode, "mode", "m", "auto", "Include mode: auto, explicit, always")
	libraryPullCmd.Flags().BoolVar(&libraryPullShared, "shared", false, "Store in shared location (available to all projects)")
	libraryPullCmd.Flags().StringSliceVarP(&libraryPullPaths, "paths", "p", nil, "File path patterns for auto-include (glob syntax)")
	libraryPullCmd.Flags().StringSliceVarP(&libraryPullTags, "tag", "t", nil, "Tags for organization")
	libraryPullCmd.Flags().IntVar(&libraryPullMaxDepth, "max-depth", 0, "Maximum crawl depth for URLs (default: 3)")
	libraryPullCmd.Flags().IntVar(&libraryPullMaxPages, "max-pages", 0, "Maximum pages to crawl (default: 100)")
	libraryPullCmd.Flags().BoolVar(&libraryPullDryRun, "dry-run", false, "Show what would be pulled without saving")
	libraryPullCmd.Flags().StringVar(&libraryPullGitRef, "git-ref", "", "Git branch or tag for git sources")
	libraryPullCmd.Flags().StringVar(&libraryPullGitPath, "git-path", "", "Subdirectory path within git repo")
	libraryPullCmd.Flags().BoolVar(&libraryPullForce, "force", false, "Overwrite existing collection without confirmation")
	libraryPullCmd.Flags().BoolVar(&libraryPullContinue, "continue", false, "Resume an interrupted crawl")
	libraryPullCmd.Flags().BoolVar(&libraryPullRestart, "restart", false, "Ignore existing state and start fresh")

	// Crawl filtering flags
	libraryPullCmd.Flags().StringVar(&libraryPullDomainScope, "domain-scope", "",
		"Domain scope: 'same-host' (exact match, e.g., docs.example.com only) or 'same-domain' (all subdomains, e.g., *.example.com)")
	libraryPullCmd.Flags().BoolVar(&libraryPullVersionFilter, "version-filter", false,
		"Auto-detect version from URL path (e.g., /v24/, /v1.2.3/) and only crawl pages with that version")
	libraryPullCmd.Flags().StringVar(&libraryPullVersion, "version", "",
		"Only crawl pages containing this version in the path (e.g., 'v24', 'v1.2.3'). Takes precedence over --version-filter")

	// Update flags
	libraryUpdateCmd.Flags().BoolVar(&libraryUpdateFull, "full", false, "Re-fetch all pages (ignore incremental)")

	// List flags
	libraryListCmd.Flags().BoolVar(&libraryListShared, "shared", false, "Show only shared collections")
	libraryListCmd.Flags().BoolVar(&libraryListProject, "project", false, "Show only project collections")
	libraryListCmd.Flags().BoolVarP(&libraryListVerbose, "verbose", "v", false, "Show detailed information")
	libraryListCmd.Flags().StringVar(&libraryListTag, "tag", "", "Filter by tag")
	libraryListCmd.Flags().StringVar(&libraryListMode, "mode", "", "Filter by include mode")

	// Remove flags
	libraryRemoveCmd.Flags().BoolVarP(&libraryRemoveForce, "force", "f", false, "Skip confirmation prompt")
}

func runLibraryPull(cmd *cobra.Command, args []string) error {
	source := args[0]
	ctx := cmd.Context()

	// Determine repo root (can be empty for shared-only)
	repoRoot := ""
	if !libraryPullShared {
		res, err := ResolveWorkspaceRoot(ctx)
		if err == nil {
			repoRoot = res.Root
		}
	}

	manager, err := library.NewManager(ctx, repoRoot)
	if err != nil {
		return fmt.Errorf("create library manager: %w", err)
	}

	// Parse include mode
	var includeMode library.IncludeMode
	switch strings.ToLower(libraryPullMode) {
	case "auto":
		includeMode = library.IncludeModeAuto
	case "explicit":
		includeMode = library.IncludeModeExplicit
	case "always":
		includeMode = library.IncludeModeAlways
	default:
		return fmt.Errorf("invalid mode %q: must be auto, explicit, or always", libraryPullMode)
	}

	// Validate domain scope
	if err := library.ValidateDomainScope(libraryPullDomainScope); err != nil {
		return err
	}

	opts := &library.PullOptions{
		Name:          libraryPullName,
		IncludeMode:   includeMode,
		Shared:        libraryPullShared,
		Paths:         libraryPullPaths,
		Tags:          libraryPullTags,
		MaxDepth:      libraryPullMaxDepth,
		MaxPages:      libraryPullMaxPages,
		DryRun:        libraryPullDryRun,
		GitRef:        libraryPullGitRef,
		GitPath:       libraryPullGitPath,
		Force:         libraryPullForce,
		Continue:      libraryPullContinue,
		ForceRestart:  libraryPullRestart,
		DomainScope:   libraryPullDomainScope,
		VersionFilter: libraryPullVersionFilter,
		VersionPath:   libraryPullVersion,
	}

	result, err := manager.Pull(ctx, source, opts)
	if err != nil {
		// Handle incomplete crawl error specially
		var incompleteErr *library.IncompleteCrawlError
		if errors.As(err, &incompleteErr) {
			fmt.Printf("%s Found incomplete crawl:\n", display.WarningMsg("!"))
			fmt.Printf("  Collection: %s\n", incompleteErr.CollectionID)
			fmt.Printf("  Started:    %s\n", incompleteErr.StartedAt.Format("2006-01-02 15:04:05"))
			fmt.Printf("  Progress:   %d/%d pages (%d failed, %d pending)\n",
				incompleteErr.Success, incompleteErr.Total,
				incompleteErr.Failed, incompleteErr.Pending)
			fmt.Println()
			fmt.Println("Options:")
			fmt.Printf("  mehr library pull %s --continue   # Resume from where it left off\n", source)
			fmt.Printf("  mehr library pull %s --restart    # Start fresh, discard progress\n", source)

			return nil
		}

		return fmt.Errorf("pull failed: %w", err)
	}

	if libraryPullDryRun {
		fmt.Println(display.SuccessMsg("Dry run completed"))
		fmt.Printf("Would pull %d URLs:\n", len(result.DryRunURLs))
		for i, u := range result.DryRunURLs {
			if i >= 10 {
				fmt.Printf("  ... and %d more\n", len(result.DryRunURLs)-10)

				break
			}
			fmt.Printf("  %s\n", u)
		}

		return nil
	}

	fmt.Printf("%s Pulled documentation: %s\n", display.SuccessMsg("✓"), result.Collection.Name)
	fmt.Printf("  ID:     %s\n", result.Collection.ID)
	fmt.Printf("  Pages:  %d\n", result.PagesWritten)
	fmt.Printf("  Source: %s\n", result.Collection.Source)
	if len(result.Collection.Paths) > 0 {
		fmt.Printf("  Paths:  %s\n", strings.Join(result.Collection.Paths, ", "))
	}

	return nil
}

func runLibraryList(cmd *cobra.Command, _ []string) error {
	ctx := cmd.Context()

	// Get repo root if available
	repoRoot := ""
	res, err := ResolveWorkspaceRoot(ctx)
	if err == nil {
		repoRoot = res.Root
	}

	manager, err := library.NewManager(ctx, repoRoot)
	if err != nil {
		return fmt.Errorf("create library manager: %w", err)
	}

	opts := &library.ListOptions{
		SharedOnly:  libraryListShared,
		ProjectOnly: libraryListProject,
		Tag:         libraryListTag,
		IncludeMode: library.IncludeMode(libraryListMode),
	}

	collections, err := manager.List(ctx, opts)
	if err != nil {
		return fmt.Errorf("list failed: %w", err)
	}

	if len(collections) == 0 {
		fmt.Println(display.InfoMsg("No collections found"))
		fmt.Println("\nPull documentation with: mehr library pull <source>")

		return nil
	}

	fmt.Printf("Found %d collection(s):\n\n", len(collections))

	for _, coll := range collections {
		if libraryListVerbose {
			fmt.Printf("%-20s  %s\n", "Name:", coll.Name)
			fmt.Printf("%-20s  %s\n", "ID:", coll.ID)
			fmt.Printf("%-20s  %s\n", "Source:", coll.Source)
			fmt.Printf("%-20s  %s\n", "Mode:", coll.IncludeMode)
			fmt.Printf("%-20s  %d\n", "Pages:", coll.PageCount)
			fmt.Printf("%-20s  %s\n", "Location:", coll.Location)
			if len(coll.Paths) > 0 {
				fmt.Printf("%-20s  %s\n", "Paths:", strings.Join(coll.Paths, ", "))
			}
			if len(coll.Tags) > 0 {
				fmt.Printf("%-20s  %s\n", "Tags:", strings.Join(coll.Tags, ", "))
			}
			fmt.Printf("%-20s  %s\n", "Pulled:", coll.PulledAt.Format("2006-01-02 15:04"))
			fmt.Println()
		} else {
			location := "project"
			if coll.Location == "shared" {
				location = "shared"
			}
			fmt.Printf("  %-25s  %-10s  %3d pages  [%s]\n",
				truncateString(coll.Name, 25),
				coll.IncludeMode,
				coll.PageCount,
				location,
			)
		}
	}

	return nil
}

func runLibraryShow(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	nameOrID := args[0]

	// Get repo root if available
	repoRoot := ""
	res, err := ResolveWorkspaceRoot(ctx)
	if err == nil {
		repoRoot = res.Root
	}

	manager, err := library.NewManager(ctx, repoRoot)
	if err != nil {
		return fmt.Errorf("create library manager: %w", err)
	}

	// If page path provided, show page content
	if len(args) > 1 {
		pagePath := args[1]
		coll, err := manager.Show(ctx, nameOrID)
		if err != nil {
			return fmt.Errorf("collection not found: %w", err)
		}

		_, content, err := manager.ShowPage(ctx, coll.ID, pagePath)
		if err != nil {
			return fmt.Errorf("page not found: %w", err)
		}

		fmt.Println(content)

		return nil
	}

	// Show collection details
	coll, err := manager.Show(ctx, nameOrID)
	if err != nil {
		return fmt.Errorf("collection not found: %w", err)
	}

	fmt.Printf("Name:      %s\n", coll.Name)
	fmt.Printf("ID:        %s\n", coll.ID)
	fmt.Printf("Source:    %s\n", coll.Source)
	fmt.Printf("Type:      %s\n", coll.SourceType)
	fmt.Printf("Mode:      %s\n", coll.IncludeMode)
	fmt.Printf("Pages:     %d\n", coll.PageCount)
	fmt.Printf("Location:  %s\n", coll.Location)
	fmt.Printf("Pulled:    %s\n", coll.PulledAt.Format("2006-01-02 15:04:05"))

	if len(coll.Paths) > 0 {
		fmt.Printf("Paths:     %s\n", strings.Join(coll.Paths, ", "))
	}
	if len(coll.Tags) > 0 {
		fmt.Printf("Tags:      %s\n", strings.Join(coll.Tags, ", "))
	}

	// List pages
	pages, err := manager.ListPages(ctx, coll.ID)
	if err == nil && len(pages) > 0 {
		fmt.Printf("\nPages (%d):\n", len(pages))
		for i, p := range pages {
			if i >= 20 {
				fmt.Printf("  ... and %d more\n", len(pages)-20)

				break
			}
			fmt.Printf("  %s\n", p)
		}
	}

	return nil
}

func runLibraryRemove(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	nameOrID := args[0]

	// Get repo root if available
	repoRoot := ""
	res, err := ResolveWorkspaceRoot(ctx)
	if err == nil {
		repoRoot = res.Root
	}

	manager, err := library.NewManager(ctx, repoRoot)
	if err != nil {
		return fmt.Errorf("create library manager: %w", err)
	}

	// Verify collection exists
	coll, err := manager.Show(ctx, nameOrID)
	if err != nil {
		return fmt.Errorf("collection not found: %w", err)
	}

	// Confirm removal unless --force
	if !libraryRemoveForce {
		fmt.Printf("%s This will permanently delete collection %q (%d pages)\n",
			display.WarningMsg("▲"), coll.Name, coll.PageCount)
		fmt.Print("Continue? [y/N] ")
		var response string
		//nolint:nilerr // Scan error treated as "not confirmed"
		if _, err := fmt.Scanln(&response); err != nil || strings.ToLower(response) != "y" {
			fmt.Println(display.InfoMsg("Cancelled"))

			return nil
		}
	}

	if err := manager.Remove(ctx, coll.ID, libraryRemoveForce); err != nil {
		return fmt.Errorf("remove failed: %w", err)
	}

	fmt.Printf("%s Removed collection: %s\n", display.SuccessMsg("✓"), coll.Name)

	return nil
}

func runLibraryUpdate(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	// Get repo root if available
	repoRoot := ""
	res, err := ResolveWorkspaceRoot(ctx)
	if err == nil {
		repoRoot = res.Root
	}

	manager, err := library.NewManager(ctx, repoRoot)
	if err != nil {
		return fmt.Errorf("create library manager: %w", err)
	}

	if len(args) > 0 {
		// Update specific collection
		nameOrID := args[0]
		result, err := manager.UpdateWithOptions(ctx, nameOrID, true, libraryUpdateFull)
		if err != nil {
			return fmt.Errorf("update failed: %w", err)
		}

		fmt.Printf("%s Updated collection: %s\n", display.SuccessMsg("✓"), result.Collection.Name)
		fmt.Printf("  Pages: %d\n", result.PagesWritten)

		return nil
	}

	// Update all collections with remote sources
	collections, err := manager.List(ctx, nil)
	if err != nil {
		return fmt.Errorf("list failed: %w", err)
	}

	updated := 0
	for _, coll := range collections {
		// Skip file sources (they're local)
		if coll.SourceType == library.SourceFile {
			continue
		}

		fmt.Printf("%s Updating %s...\n", display.InfoMsg("→"), coll.Name)
		if _, err := manager.Update(ctx, coll.ID); err != nil {
			fmt.Printf("%s Failed to update %s: %v\n", display.ErrorMsg("✗"), coll.Name, err)

			continue
		}
		updated++
	}

	if updated == 0 {
		fmt.Println(display.InfoMsg("No remote collections to update"))
	} else {
		fmt.Printf("%s Updated %d collection(s)\n", display.SuccessMsg("✓"), updated)
	}

	return nil
}
