package commands

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/valksor/go-mehrhof/internal/conductor"
	"github.com/valksor/go-mehrhof/internal/links"
	"github.com/valksor/go-mehrhof/internal/storage"
)

var (
	linksListFormat  string // Format for list output: table, JSON
	linksListVerbose bool   // Show additional context
)

var linksCmd = &cobra.Command{
	Use:   "links",
	Short: "Manage bidirectional links between specs, notes, and sessions",
	Long: `Manage Logseq-style bidirectional links between specifications, notes, sessions, and decisions.

The links system provides knowledge graph capabilities with:
- Bidirectional linking: see what references X and what X references
- Name-based references: use human-readable names like [[Authentication Spec]]
- Context tracking: see surrounding text for each link

SUBCOMMANDS:
  list        Show outgoing links from an entity
  backlinks   Show incoming links to an entity
  search      Find entities by name
  stats       Show link graph statistics
  rebuild     Rebuild index from workspace content

REFERENCE SYNTAX:
  [[spec:N]]                 Specification N (current task)
  [[spec:task-id:N]]          Specification N (specific task)
  [[session:timestamp]]       Session by timestamp
  [[decision:name]]           Named decision
  [[Entity Name]]             Human-readable name

WHEN TO USE:
  • Understanding relationships between specifications
  • Finding what depends on a specific decision
  • Discovering related work across tasks
  • Auditing knowledge graph structure

USE THIS COMMAND FOR:
  Exploring the knowledge graph of linked content

RELATED COMMANDS:
  note       - Add notes that can contain [[references]]
  optimize    - AI optimization sees your linked context
  memory      - Semantic search across all tasks

Examples:
  mehr links list                      # List all entities with links
  mehr links list spec:task-123:1     # Show outgoing links from spec
  mehr links backlinks spec:task-123:1 # Show what references this spec
  mehr links search "authentication"   # Find entities by name
  mehr links stats                     # Show graph statistics
  mehr links rebuild                   # Rebuild index from scratch`,
	RunE: runLinks,
}

func init() {
	rootCmd.AddCommand(linksCmd)

	// List subcommand
	listCmd := &cobra.Command{
		Use:   "list [entity]",
		Short: "Show outgoing links from an entity",
		Long:  `Show all outgoing links from a given entity (specs, notes, sessions).`,
		Args:  cobra.MaximumNArgs(1),
		RunE:  runLinksList,
	}
	listCmd.Flags().StringVar(&linksListFormat, "format", "table", "Output format: table, json")
	listCmd.Flags().BoolVar(&linksListVerbose, "verbose", false, "Show additional context")
	linksCmd.AddCommand(listCmd)

	// Backlinks subcommand
	backlinksCmd := &cobra.Command{
		Use:   "backlinks [entity]",
		Short: "Show incoming links to an entity",
		Long:  `Show all incoming links (backlinks) to a given entity. This shows what references this entity.`,
		Args:  cobra.MaximumNArgs(1),
		RunE:  runLinksBacklinks,
	}
	backlinksCmd.Flags().StringVar(&linksListFormat, "format", "table", "Output format: table, json")
	backlinksCmd.Flags().BoolVar(&linksListVerbose, "verbose", false, "Show additional context")
	linksCmd.AddCommand(backlinksCmd)

	// Search subcommand
	searchCmd := &cobra.Command{
		Use:   "search <name>",
		Short: "Find entities by human-readable name",
		Long:  `Search for entities by human-readable name. Supports partial matching and is case-insensitive by default.`,
		Args:  cobra.ExactArgs(1),
		RunE:  runLinksSearch,
	}
	linksCmd.AddCommand(searchCmd)

	// Stats subcommand
	statsCmd := &cobra.Command{
		Use:   "stats",
		Short: "Show link graph statistics",
		Long:  `Display statistics about the link graph including total links, entities, and most-linked items.`,
		Args:  cobra.NoArgs,
		RunE:  runLinksStats,
	}
	linksCmd.AddCommand(statsCmd)

	// Rebuild subcommand
	rebuildCmd := &cobra.Command{
		Use:   "rebuild",
		Short: "Rebuild link index from workspace content",
		Long:  `Rebuild the entire link index by scanning all workspace content (specs, notes, sessions). This is useful after manual edits or migration.`,
		Args:  cobra.NoArgs,
		RunE:  runLinksRebuild,
	}
	rebuildCmd.SilenceUsage = true
	linksCmd.AddCommand(rebuildCmd)
}

func runLinks(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	// Build conductor options
	opts := []conductor.Option{
		conductor.WithVerbose(verbose),
	}

	// Initialize conductor
	cond, err := initializeConductor(ctx, opts...)
	if err != nil {
		return err
	}

	// Check if links are enabled
	ws := cond.GetWorkspace()
	if ws == nil {
		return errors.New("workspace not initialized")
	}

	cfg, err := ws.LoadConfig()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	if cfg.Links == nil || !cfg.Links.Enabled {
		fmt.Println("Links are not enabled in this workspace.")
		fmt.Println("\nTo enable links, add to .mehrhof/config.yaml:")
		fmt.Println("  links:")
		fmt.Println("    enabled: true")

		return nil
	}

	// No entity specified, show overview
	if len(args) == 0 {
		return showLinksOverview(ctx, cond)
	}

	// Entity specified, show details
	entityID := args[0]

	return showLinksDetails(ctx, cond, entityID)
}

func showLinksOverview(ctx context.Context, cond *conductor.Conductor) error {
	ws := cond.GetWorkspace()
	linkMgr := storage.GetLinkManager(ctx, ws)

	stats := linkMgr.GetStats()
	if stats == nil {
		return errors.New("link index not available")
	}

	// Get all entities with links
	allSources := getAllEntitiesWithLinks(linkMgr)

	fmt.Printf("📊 Link Graph Overview\n\n")
	fmt.Printf("Total links: %d\n", stats.TotalLinks)
	fmt.Printf("Total entities: %d\n", stats.TotalSources)
	fmt.Printf("Orphan entities: %d\n\n", stats.OrphanEntities)

	if len(allSources) > 0 {
		fmt.Println("Entities with links:")
		for _, entity := range allSources {
			outgoing := len(linkMgr.GetOutgoing(entity.ID))
			incoming := len(linkMgr.GetIncoming(entity.ID))
			fmt.Printf("  • %s (%d outgoing, %d incoming)\n", entity.Title, outgoing, incoming)
		}
	}

	fmt.Println("\nUse a subcommand to explore:")
	fmt.Println("  mehr links list [entity]       - Show outgoing links")
	fmt.Println("  mehr links backlinks [entity]  - Show incoming links")
	fmt.Println("  mehr links search <name>       - Find by name")
	fmt.Println("  mehr links stats               - Detailed statistics")

	return nil
}

func showLinksDetails(ctx context.Context, cond *conductor.Conductor, entityID string) error {
	ws := cond.GetWorkspace()
	linkMgr := storage.GetLinkManager(ctx, ws)

	// Get outgoing and incoming links
	outgoing := linkMgr.GetOutgoing(entityID)
	incoming := linkMgr.GetIncoming(entityID)

	fmt.Printf("🔗 Links for %s\n\n", entityID)

	fmt.Printf("Outgoing (%d): references this entity\n", len(outgoing))
	if len(outgoing) > 0 {
		printLinks(outgoing, linksListVerbose)
	}

	fmt.Printf("\nIncoming (%d): referenced by\n", len(incoming))
	if len(incoming) > 0 {
		printLinks(incoming, linksListVerbose)
	}

	if len(outgoing) == 0 && len(incoming) == 0 {
		fmt.Println("  No links found.")
	}

	return nil
}

func runLinksList(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	// Initialize conductor
	cond, err := initializeConductor(ctx, conductor.WithVerbose(verbose))
	if err != nil {
		return err
	}

	ws := cond.GetWorkspace()
	if ws == nil {
		return errors.New("workspace not initialized")
	}

	linkMgr := storage.GetLinkManager(ctx, ws)

	// If no entity specified, show all entities with outgoing links
	if len(args) == 0 {
		return listAllEntities(linkMgr)
	}

	entityID := args[0]
	links := linkMgr.GetOutgoing(entityID)

	fmt.Printf("🔗 Outgoing links from %s\n\n", entityID)
	if len(links) == 0 {
		fmt.Println("  No outgoing links found.")

		return nil
	}

	printLinks(links, linksListVerbose)

	return nil
}

func runLinksBacklinks(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	// Initialize conductor
	cond, err := initializeConductor(ctx, conductor.WithVerbose(verbose))
	if err != nil {
		return err
	}

	ws := cond.GetWorkspace()
	if ws == nil {
		return errors.New("workspace not initialized")
	}

	linkMgr := storage.GetLinkManager(ctx, ws)

	if len(args) == 0 {
		return errors.New("entity ID required\n\nUsage: mehr links backlinks <entity>")
	}

	entityID := args[0]
	links := linkMgr.GetIncoming(entityID)

	fmt.Printf("🔗 Backlinks to %s\n\n", entityID)
	if len(links) == 0 {
		fmt.Println("  No backlinks found.")
		fmt.Println("\nThis entity is not referenced by any other content.")

		return nil
	}

	printLinks(links, linksListVerbose)

	return nil
}

func runLinksSearch(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	// Initialize conductor
	cond, err := initializeConductor(ctx, conductor.WithVerbose(verbose))
	if err != nil {
		return err
	}

	ws := cond.GetWorkspace()
	if ws == nil {
		return errors.New("workspace not initialized")
	}

	linkMgr := storage.GetLinkManager(ctx, ws)
	searchName := args[0]

	// Search for matching entities
	matches := searchEntitiesByName(linkMgr, searchName)

	fmt.Printf("🔍 Searching for %q\n\n", searchName)
	if len(matches) == 0 {
		fmt.Println("  No matches found.")
		fmt.Println("\nTips:")
		fmt.Println("  • Use partial names: mehr links search auth")
		fmt.Println("  • Search is case-insensitive")
		fmt.Println("  • Use list to see all entities: mehr links list")

		return nil
	}

	fmt.Printf("Found %d matching entities:\n\n", len(matches))
	for _, match := range matches {
		outgoing := len(linkMgr.GetOutgoing(match.ID))
		incoming := len(linkMgr.GetIncoming(match.ID))
		fmt.Printf("  • %s", color.CyanString(match.ID))
		if match.Title != match.ID {
			fmt.Printf(" (%s)", match.Title)
		}
		fmt.Printf(" [%d out, %d in]\n", outgoing, incoming)
	}

	return nil
}

func runLinksStats(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	// Initialize conductor
	cond, err := initializeConductor(ctx, conductor.WithVerbose(verbose))
	if err != nil {
		return err
	}

	ws := cond.GetWorkspace()
	if ws == nil {
		return errors.New("workspace not initialized")
	}

	linkMgr := storage.GetLinkManager(ctx, ws)
	stats := linkMgr.GetStats()

	if stats == nil {
		return errors.New("link index not available")
	}

	fmt.Printf("📊 Link Graph Statistics\n\n")
	fmt.Printf("Total links:     %s\n", color.GreenString(strconv.Itoa(stats.TotalLinks)))
	fmt.Printf("Total sources:   %s\n", color.CyanString(strconv.Itoa(stats.TotalSources)))
	fmt.Printf("Total targets:   %s\n", color.CyanString(strconv.Itoa(stats.TotalTargets)))
	fmt.Printf("Orphan entities: %s\n", formatOrphanCount(stats.OrphanEntities))

	// Show most linked entities (if available)
	allEntities := getAllEntitiesWithLinks(linkMgr)
	if len(allEntities) > 0 {
		// Sort by total links (incoming + outgoing)
		type entityStats struct {
			id    string
			title string
			total int
		}
		sorted := make([]entityStats, 0, len(allEntities))
		for _, e := range allEntities {
			outgoing := len(linkMgr.GetOutgoing(e.ID))
			incoming := len(linkMgr.GetIncoming(e.ID))
			sorted = append(sorted, entityStats{
				id:    e.ID,
				title: e.Title,
				total: outgoing + incoming,
			})
		}

		// Simple sort by total links
		for i := 0; i < len(sorted); i++ {
			for j := i + 1; j < len(sorted); j++ {
				if sorted[j].total > sorted[i].total {
					sorted[i], sorted[j] = sorted[j], sorted[i]
				}
			}
		}

		fmt.Println("\nMost linked entities:")
		count := 5
		if len(sorted) < count {
			count = len(sorted)
		}
		for i := range count {
			e := sorted[i]
			fmt.Printf("  %d. %s", i+1, color.CyanString(e.id))
			if e.title != e.id {
				fmt.Printf(" (%s)", e.title)
			}
			fmt.Printf(" [%d total links]\n", e.total)
		}
	}

	return nil
}

func runLinksRebuild(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	// Initialize conductor
	cond, err := initializeConductor(ctx, conductor.WithVerbose(verbose))
	if err != nil {
		return err
	}

	ws := cond.GetWorkspace()
	if ws == nil {
		return errors.New("workspace not initialized")
	}

	linkMgr := storage.GetLinkManager(ctx, ws)

	fmt.Println("🔄 Rebuilding link index from workspace content...")

	if err := linkMgr.Rebuild(); err != nil {
		return fmt.Errorf("rebuild index: %w", err)
	}

	stats := linkMgr.GetStats()
	fmt.Printf("\n✓ Index rebuilt successfully!\n")
	fmt.Printf("  Total links: %d\n", stats.TotalLinks)
	fmt.Printf("  Total entities: %d\n", stats.TotalSources)
	fmt.Printf("  Total targets: %d\n", stats.TotalTargets)

	return nil
}

// printLinks prints links in a formatted table.
func printLinks(linksSlice []links.Link, verbose bool) {
	if linksListFormat == "json" {
		printLinksJSON(linksSlice)

		return
	}

	// Table format
	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "TARGET\tCONTEXT") //nolint:errcheck // Writing to stdout

	for _, link := range linksSlice {
		context := link.Context
		if !verbose && len(context) > 60 {
			context = context[:57] + "..."
		}
		if context == "" {
			context = "-"
		}
		fmt.Fprintf(tw, "%s\t%s\n", color.GreenString(link.Target), context) //nolint:errcheck // Writing to stdout
	}

	tw.Flush() //nolint:errcheck // Writing to stdout
}

// printLinksJSON prints links as JSON.
func printLinksJSON(linksSlice []links.Link) {
	fmt.Println("[")
	for i, link := range linksSlice {
		fmt.Printf("  {\n")
		fmt.Printf("    \"source\": \"%s\",\n", link.Source)
		fmt.Printf("    \"target\": \"%s\",\n", link.Target)
		fmt.Printf("    \"context\": \"%s\",\n", link.Context)
		fmt.Printf("    \"created_at\": \"%s\"\n", link.CreatedAt.Format("2006-01-02T15:04:05Z"))
		if i < len(linksSlice)-1 {
			fmt.Println("  },")
		} else {
			fmt.Println("  }")
		}
	}
	fmt.Println("]")
}

// entityInfo represents an entity for display.
type entityInfo struct {
	ID    string
	Title string
}

// getAllEntitiesWithLinks returns all entities that have links.
func getAllEntitiesWithLinks(linkMgr *storage.LinkManager) []entityInfo {
	linkIndex := linkMgr.GetIndex()
	if linkIndex == nil {
		return []entityInfo{}
	}

	names := linkMgr.GetNames()
	result := make([]entityInfo, 0)

	// Collect all unique entities from both forward and backward indices
	seen := make(map[string]bool)

	// Add all sources (entities with outgoing links)
	for source := range linkIndex.Forward {
		if !seen[source] {
			seen[source] = true
			result = append(result, entityInfo{ID: source, Title: getTitleForEntity(names, source)})
		}
	}

	// Add all targets (entities with incoming links but possibly no outgoing)
	for target := range linkIndex.Backward {
		if !seen[target] {
			seen[target] = true
			result = append(result, entityInfo{ID: target, Title: getTitleForEntity(names, target)})
		}
	}

	return result
}

// getTitleForEntity tries to find a human-readable title for an entity ID.
// It searches the name registry and falls back to the entity ID itself.
func getTitleForEntity(names *links.NameRegistry, entityID string) string {
	if names == nil {
		return entityID
	}

	// Parse the entity ID to get type and ID
	entityType, _, id := links.ParseEntityID(entityID)

	// Try to find a name in the appropriate registry
	switch entityType {
	case links.TypeSpec:
		for name, eid := range names.Specs {
			if eid == entityID {
				return name
			}
		}
	case links.TypeSession:
		for name, eid := range names.Sessions {
			if eid == entityID {
				return name
			}
		}
	case links.TypeDecision:
		for name, eid := range names.Decisions {
			if eid == entityID {
				return name
			}
		}
	case links.TypeTask:
		for name, eid := range names.Tasks {
			if eid == entityID {
				return name
			}
		}
	case links.TypeNote, links.TypeSolution, links.TypeError:
		for name, eid := range names.Notes {
			if eid == entityID {
				return name
			}
		}
	}

	// If the entity ID has a meaningful number/ID portion, use that
	if id != "" && id != entityID {
		return id
	}

	return entityID
}

// listAllEntities lists all entities with outgoing links.
func listAllEntities(linkMgr *storage.LinkManager) error {
	entities := getAllEntitiesWithLinks(linkMgr)

	if linksListFormat == "json" {
		return listAllEntitiesJSON(entities, linkMgr)
	}

	// Table format
	if len(entities) == 0 {
		fmt.Println("No entities with links found.")
		fmt.Println("\nLinks are created when content contains [[references]].")

		return nil
	}

	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "ENTITY\tTITLE\tOUTGOING\tINCOMING") //nolint:errcheck // Writing to stdout

	for _, e := range entities {
		outgoing := len(linkMgr.GetOutgoing(e.ID))
		incoming := len(linkMgr.GetIncoming(e.ID))
		fmt.Fprintf(tw, "%s\t%s\t%d\t%d\n", e.ID, e.Title, outgoing, incoming) //nolint:errcheck // Writing to stdout
	}

	tw.Flush() //nolint:errcheck // Writing to stdout

	return nil
}

// listAllEntitiesJSON prints all entities in JSON format.
func listAllEntitiesJSON(entities []entityInfo, linkMgr *storage.LinkManager) error {
	fmt.Println("[")
	for i, e := range entities {
		outgoing := len(linkMgr.GetOutgoing(e.ID))
		incoming := len(linkMgr.GetIncoming(e.ID))
		fmt.Printf("  {\n")
		fmt.Printf("    \"id\": \"%s\",\n", e.ID)
		fmt.Printf("    \"title\": \"%s\",\n", e.Title)
		fmt.Printf("    \"outgoing\": %d,\n", outgoing)
		fmt.Printf("    \"incoming\": %d\n", incoming)
		if i < len(entities)-1 {
			fmt.Println("  },")
		} else {
			fmt.Println("  }")
		}
	}
	fmt.Println("]")

	return nil
}

// searchEntitiesByName searches for entities matching the given name.
func searchEntitiesByName(linkMgr *storage.LinkManager, name string) []entityInfo {
	names := linkMgr.GetNames()
	if names == nil {
		return []entityInfo{}
	}

	result := make([]entityInfo, 0)
	seen := make(map[string]bool)

	searchLower := strings.ToLower(name)

	// Search in all registries
	searchRegistry := func(registry map[string]string) {
		for regName, entityID := range registry {
			if !seen[entityID] && strings.Contains(strings.ToLower(regName), searchLower) {
				seen[entityID] = true
				result = append(result, entityInfo{ID: entityID, Title: regName})
			}
		}
	}

	searchRegistry(names.Specs)
	searchRegistry(names.Sessions)
	searchRegistry(names.Decisions)
	searchRegistry(names.Tasks)
	searchRegistry(names.Notes)

	return result
}

// formatOrphanCount formats orphan count with color.
func formatOrphanCount(count int) string {
	if count == 0 {
		return color.GreenString(strconv.Itoa(count))
	}
	if count > 10 {
		return color.RedString(strconv.Itoa(count))
	}

	return color.YellowString(strconv.Itoa(count))
}
