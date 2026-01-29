package links

import (
	"slices"
	"time"
)

// QueryOption defines a query filter for link searches.
type QueryOption func(*Query)

// Query represents a link query with filters.
type Query struct {
	From           string       // Source entity ID
	To             string       // Target entity ID
	Types          []EntityType // Entity type filter
	MinDepth       int          // Minimum depth
	MaxDepth       int          // Maximum depth
	CreatedAfter   time.Time    // Filter by creation time
	IncludeOrphans bool         // Include entities with no links
}

// From filters links from a specific source entity.
func From(entityID string) QueryOption {
	return func(q *Query) {
		q.From = entityID
	}
}

// To filters links to a specific target entity.
func To(entityID string) QueryOption {
	return func(q *Query) {
		q.To = entityID
	}
}

// OfType filters links by entity type.
func OfType(entityType EntityType) QueryOption {
	return func(q *Query) {
		q.Types = append(q.Types, entityType)
	}
}

// WithMinDepth sets the minimum path depth.
func WithMinDepth(depth int) QueryOption {
	return func(q *Query) {
		q.MinDepth = depth
	}
}

// WithMaxDepth sets the maximum path depth.
func WithMaxDepth(depth int) QueryOption {
	return func(q *Query) {
		q.MaxDepth = depth
	}
}

// CreatedAfter filters links created after a specific time.
func CreatedAfter(t time.Time) QueryOption {
	return func(q *Query) {
		q.CreatedAfter = t
	}
}

// IncludeOrphans includes entities with no links in the results.
func IncludeOrphans() QueryOption {
	return func(q *Query) {
		q.IncludeOrphans = true
	}
}

// FindLinks searches for links matching the given criteria.
func (idx *LinkIndex) FindLinks(opts ...QueryOption) []Link {
	q := &Query{}
	for _, opt := range opts {
		opt(q)
	}

	var results []Link

	// If source specified, get outgoing links
	if q.From != "" {
		for _, link := range idx.Forward[q.From] {
			if q.matchFilters(link) {
				results = append(results, link)
			}
		}

		return results
	}

	// If target specified, get incoming links
	if q.To != "" {
		for _, link := range idx.Backward[q.To] {
			if q.matchFilters(link) {
				results = append(results, link)
			}
		}

		return results
	}

	// No specific source or target, search all links
	for _, links := range idx.Forward {
		for _, link := range links {
			if q.matchFilters(link) {
				results = append(results, link)
			}
		}
	}

	return results
}

// matchFilters checks if a link matches the query filters.
func (q *Query) matchFilters(link Link) bool {
	// Filter by type (extract from entity ID)
	if len(q.Types) > 0 {
		sourceType, _, _ := ParseEntityID(link.Source)
		if !slices.Contains(q.Types, sourceType) {
			return false
		}
	}

	// Filter by creation time
	if !q.CreatedAfter.IsZero() && link.CreatedAt.Before(q.CreatedAfter) {
		return false
	}

	return true
}

// FindBacklinks returns all links pointing to the given entity.
func (idx *LinkIndex) FindBacklinks(entityID string) []Link {
	return idx.Backward[entityID]
}

// FindOrphans returns all entity IDs that have no incoming or outgoing links.
// Orphans are entities that are referenced (have incoming links) but don't reference anything else (no outgoing links).
func (idx *LinkIndex) FindOrphans() []string {
	// Collect all entities that have outgoing links
	hasOutgoing := make(map[string]bool)
	for source := range idx.Forward {
		hasOutgoing[source] = true
	}

	// Find entities that appear as targets but not as sources
	// (entities that are referenced but don't reference anything else)
	var orphans []string
	for target := range idx.Backward {
		if !hasOutgoing[target] {
			orphans = append(orphans, target)
		}
	}

	return orphans
}

// FindPath finds the shortest path between two entities using BFS.
// Returns a slice of entity IDs representing the path, or nil if no path exists.
func (idx *LinkIndex) FindPath(from, to string) []string {
	if from == to {
		return []string{from}
	}

	// BFS to find shortest path
	queue := [][]string{{from}}
	visited := make(map[string]bool)
	visited[from] = true

	for len(queue) > 0 {
		path := queue[0]
		queue = queue[1:]

		current := path[len(path)-1]

		// Check if we've reached the target
		for _, link := range idx.Forward[current] {
			if link.Target == to {
				// Found path
				return append(path, to)
			}

			if !visited[link.Target] {
				visited[link.Target] = true
				newPath := make([]string, len(path)+1)
				copy(newPath, path)
				newPath[len(path)] = link.Target
				queue = append(queue, newPath)
			}
		}
	}

	// No path found
	return nil
}

// FindConnectedEntities finds all entities reachable from the given entity.
// Returns a map of entity ID to distance (number of hops).
// maxDepth of 0 means only include the source entity itself.
func (idx *LinkIndex) FindConnectedEntities(from string, maxDepth int) map[string]int {
	visited := make(map[string]int)
	visited[from] = 0

	// If maxDepth is 0, only return the source itself
	if maxDepth == 0 {
		return visited
	}

	queue := []string{from}
	depth := 0

	for len(queue) > 0 && depth < maxDepth {
		levelSize := len(queue)
		depth++

		for range levelSize {
			current := queue[0]
			queue = queue[1:]

			for _, link := range idx.Forward[current] {
				if _, exists := visited[link.Target]; !exists {
					visited[link.Target] = depth
					queue = append(queue, link.Target)
				}
			}
		}
	}

	return visited
}

// FindMutualLinks finds entities that are linked in both directions.
// Returns a map of entity ID to the pair of links (forward, backward).
func (idx *LinkIndex) FindMutualLinks(entityID string) map[string][]Link {
	mutual := make(map[string][]Link)

	// Get outgoing links
	outgoing := idx.Forward[entityID]
	for _, link := range outgoing {
		// Check if target links back to us
		// We need to check if any backlink has entityID as its source
		incoming := idx.Backward[link.Target]
		for _, backlink := range incoming {
			// backlink.Target is link.Target (same entity we're looking up in Backward)
			// We want to check if backlink.Source is our original entityID
			if backlink.Source == entityID {
				mutual[link.Target] = []Link{link, backlink}

				break
			}
		}
	}

	return mutual
}

// FindCircularPaths finds all circular paths (cycles) starting and ending at the given entity.
// Uses DFS to find cycles up to a maximum depth.
func (idx *LinkIndex) FindCircularPaths(from string, maxDepth int) [][]string {
	var paths [][]string
	visited := make(map[string]bool)

	var dfs func(current string, path []string, depth int)
	dfs = func(current string, path []string, depth int) {
		if depth > maxDepth {
			return
		}

		// Check if we've completed a cycle
		if len(path) > 1 && current == from {
			// Found a cycle (don't include the duplicate from at the end)
			cycle := make([]string, len(path))
			copy(cycle, path)
			paths = append(paths, cycle)

			return
		}

		// Avoid revisiting nodes in the current path
		if visited[current] {
			return
		}
		visited[current] = true

		// Explore outgoing links
		for _, link := range idx.Forward[current] {
			newPath := append(path, link.Target)
			dfs(link.Target, newPath, depth+1)
		}

		visited[current] = false
	}

	dfs(from, []string{from}, 0)

	return paths
}

// StatsWithDetails returns detailed statistics about the link graph.
func (idx *LinkIndex) StatsWithDetails() LinkStatsDetails {
	totalLinks := 0
	maxOutgoing := 0
	maxIncoming := 0

	// Calculate statistics
	for _, links := range idx.Forward {
		totalLinks += len(links)
		if len(links) > maxOutgoing {
			maxOutgoing = len(links)
		}
	}

	for _, links := range idx.Backward {
		if len(links) > maxIncoming {
			maxIncoming = len(links)
		}
	}

	// Find most connected entities
	type entityStats struct {
		id    string
		total int
		out   int
		in    int
	}

	var entities []entityStats
	for source, forwardLinks := range idx.Forward {
		out := len(forwardLinks)
		in := len(idx.Backward[source])
		entities = append(entities, entityStats{
			id:    source,
			total: out + in,
			out:   out,
			in:    in,
		})
	}

	slices.SortFunc(entities, func(a, b entityStats) int {
		if b.total != a.total {
			return b.total - a.total // Descending by total
		}
		if b.out != a.out {
			return b.out - a.out // Then by outgoing
		}

		return b.in - a.in // Then by incoming
	})

	// Get top 10
	var mostConnected []string
	for i, e := range entities {
		if i >= 10 {
			break
		}
		mostConnected = append(mostConnected, e.id)
	}

	return LinkStatsDetails{
		TotalLinks:      totalLinks,
		TotalSources:    len(idx.Forward),
		TotalTargets:    len(idx.Backward),
		MaxOutgoing:     maxOutgoing,
		MaxIncoming:     maxIncoming,
		MostConnected:   mostConnected,
		AverageOutgoing: float64(totalLinks) / float64(len(idx.Forward)),
		AverageIncoming: float64(totalLinks) / float64(len(idx.Backward)),
	}
}

// LinkStatsDetails represents detailed link statistics.
type LinkStatsDetails struct {
	TotalLinks      int      // Total number of links
	TotalSources    int      // Number of entities with outgoing links
	TotalTargets    int      // Number of entities with incoming links
	MaxOutgoing     int      // Maximum number of outgoing links from a single entity
	MaxIncoming     int      // Maximum number of incoming links to a single entity
	MostConnected   []string // Top 10 most connected entities
	AverageOutgoing float64  // Average number of outgoing links per source
	AverageIncoming float64  // Average number of incoming links per target
}
