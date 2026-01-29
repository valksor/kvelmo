package links

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFindLinks(t *testing.T) {
	idx := NewLinkIndex()

	// Add test links
	link1 := Link{
		Source:    "spec:task1:1",
		Target:    "spec:task1:2",
		Context:   "See API design",
		CreatedAt: time.Now(),
	}
	link2 := Link{
		Source:    "spec:task1:1",
		Target:    "decision:task1:cache",
		Context:   "Use cache strategy",
		CreatedAt: time.Now(),
	}
	link3 := Link{
		Source:    "note:task1:notes",
		Target:    "spec:task1:1",
		Context:   "Implemented auth",
		CreatedAt: time.Now(),
	}

	idx.AddLink(link1)
	idx.AddLink(link2)
	idx.AddLink(link3)

	t.Run("find from specific source", func(t *testing.T) {
		results := idx.FindLinks(From("spec:task1:1"))
		assert.Len(t, results, 2)
	})

	t.Run("find to specific target", func(t *testing.T) {
		results := idx.FindLinks(To("spec:task1:1"))
		assert.Len(t, results, 1)
		assert.Equal(t, "note:task1:notes", results[0].Source)
	})

	t.Run("find by type", func(t *testing.T) {
		results := idx.FindLinks(OfType(TypeSpec))
		assert.Len(t, results, 2)
	})

	t.Run("find all links", func(t *testing.T) {
		results := idx.FindLinks()
		assert.Len(t, results, 3)
	})
}

func TestFindBacklinks(t *testing.T) {
	idx := NewLinkIndex()

	link1 := Link{Source: "spec:task1:1", Target: "spec:task1:2", CreatedAt: time.Now()}
	link2 := Link{Source: "note:task1:notes", Target: "spec:task1:1", CreatedAt: time.Now()}

	idx.AddLink(link1)
	idx.AddLink(link2)

	t.Run("find backlinks to entity", func(t *testing.T) {
		backlinks := idx.FindBacklinks("spec:task1:1")
		assert.Len(t, backlinks, 1)
		assert.Equal(t, "note:task1:notes", backlinks[0].Source)
	})

	t.Run("no backlinks returns empty", func(t *testing.T) {
		// spec:task1:999 has no incoming links at all
		backlinks := idx.FindBacklinks("spec:task1:999")
		assert.Empty(t, backlinks)
	})
}

func TestFindPath(t *testing.T) {
	idx := NewLinkIndex()

	// Create a path: spec:1 → spec:2 → spec:3
	idx.AddLink(Link{Source: "spec:task1:1", Target: "spec:task1:2", CreatedAt: time.Now()})
	idx.AddLink(Link{Source: "spec:task1:2", Target: "spec:task1:3", CreatedAt: time.Now()})

	t.Run("find path between connected entities", func(t *testing.T) {
		path := idx.FindPath("spec:task1:1", "spec:task1:3")
		assert.NotNil(t, path)
		assert.Equal(t, []string{"spec:task1:1", "spec:task1:2", "spec:task1:3"}, path)
	})

	t.Run("same entity returns single node", func(t *testing.T) {
		path := idx.FindPath("spec:task1:1", "spec:task1:1")
		assert.Equal(t, []string{"spec:task1:1"}, path)
	})

	t.Run("no path returns nil", func(t *testing.T) {
		path := idx.FindPath("spec:task1:1", "spec:task1:999")
		assert.Nil(t, path)
	})
}

func TestFindOrphans(t *testing.T) {
	idx := NewLinkIndex()

	// Entity with only incoming links (orphan)
	link := Link{Source: "spec:task1:1", Target: "spec:task1:2", CreatedAt: time.Now()}
	idx.AddLink(link)

	// spec:task1:2 has incoming link but no outgoing links
	orphans := idx.FindOrphans()
	assert.NotEmpty(t, orphans)
	assert.Contains(t, orphans, "spec:task1:2")
}

func TestFindConnectedEntities(t *testing.T) {
	idx := NewLinkIndex()

	// Create a star pattern: spec:1 → spec:2, spec:1 → spec:3
	idx.AddLink(Link{Source: "spec:task1:1", Target: "spec:task1:2", CreatedAt: time.Now()})
	idx.AddLink(Link{Source: "spec:task1:1", Target: "spec:task1:3", CreatedAt: time.Now()})

	t.Run("find connected entities", func(t *testing.T) {
		connected := idx.FindConnectedEntities("spec:task1:1", 2)
		assert.Equal(t, 0, connected["spec:task1:1"]) // Distance 0
		assert.Equal(t, 1, connected["spec:task1:2"]) // Distance 1
		assert.Equal(t, 1, connected["spec:task1:3"]) // Distance 1
	})

	t.Run("max depth limits search", func(t *testing.T) {
		connected := idx.FindConnectedEntities("spec:task1:1", 0)
		assert.Len(t, connected, 1) // Only the source itself
		assert.Equal(t, 0, connected["spec:task1:1"])
	})
}

func TestFindMutualLinks(t *testing.T) {
	idx := NewLinkIndex()

	// Create mutual links: spec:1 ↔ spec:2
	idx.AddLink(Link{Source: "spec:task1:1", Target: "spec:task1:2", CreatedAt: time.Now()})
	idx.AddLink(Link{Source: "spec:task1:2", Target: "spec:task1:1", CreatedAt: time.Now()})

	t.Run("find mutual links", func(t *testing.T) {
		mutual := idx.FindMutualLinks("spec:task1:1")
		assert.Len(t, mutual, 1)
		assert.Contains(t, mutual, "spec:task1:2")
	})

	t.Run("one-way links returns empty", func(t *testing.T) {
		mutual := idx.FindMutualLinks("spec:task1:2")
		// Should still find the mutual link
		assert.Len(t, mutual, 1)
	})
}

func TestStatsWithDetails(t *testing.T) {
	idx := NewLinkIndex()

	// Add test data
	idx.AddLink(Link{Source: "spec:task1:1", Target: "spec:task1:2", CreatedAt: time.Now()})
	idx.AddLink(Link{Source: "spec:task1:1", Target: "spec:task1:3", CreatedAt: time.Now()})
	idx.AddLink(Link{Source: "spec:task1:2", Target: "spec:task1:3", CreatedAt: time.Now()})

	stats := idx.StatsWithDetails()

	// Link graph:
	// spec:task1:1 → spec:task1:2
	// spec:task1:1 → spec:task1:3
	// spec:task1:2 → spec:task1:3
	//
	// Forward: spec:task1:1 (2 out), spec:task1:2 (1 out)
	// Backward: spec:task1:2 (1 in), spec:task1:3 (2 in)
	//
	// Note: spec:task1:3 has no outgoing links, so it's not in MostConnected
	// (which only includes entities with outgoing links)

	assert.Equal(t, 3, stats.TotalLinks)
	assert.Equal(t, 2, stats.TotalSources) // spec:task1:1, spec:task1:2
	assert.Equal(t, 2, stats.TotalTargets) // spec:task1:2, spec:task1:3
	assert.Equal(t, 2, stats.MaxOutgoing)  // spec:task1:1 has 2 outgoing
	assert.Equal(t, 2, stats.MaxIncoming)  // spec:task1:3 has 2 incoming

	// Most connected only includes entities with outgoing links
	// spec:task1:1: 2 out + 0 in = 2 total
	// spec:task1:2: 1 out + 1 in = 2 total
	assert.Len(t, stats.MostConnected, 2)
	// First is spec:task1:1 because it has 2 outgoing (tiebreaker)
	assert.Equal(t, "spec:task1:1", stats.MostConnected[0])
	assert.Equal(t, "spec:task1:2", stats.MostConnected[1])
}

func TestQueryOptions(t *testing.T) {
	t.Run("from option sets source", func(t *testing.T) {
		q := &Query{}
		From("spec:task1:1")(q)
		assert.Equal(t, "spec:task1:1", q.From)
	})

	t.Run("to option sets target", func(t *testing.T) {
		q := &Query{}
		To("spec:task1:2")(q)
		assert.Equal(t, "spec:task1:2", q.To)
	})

	t.Run("ofType option adds type", func(t *testing.T) {
		q := &Query{}
		OfType(TypeSpec)(q)
		OfType(TypeDecision)(q)
		assert.Len(t, q.Types, 2)
		assert.Contains(t, q.Types, TypeSpec)
		assert.Contains(t, q.Types, TypeDecision)
	})

	t.Run("createdAfter option sets time", func(t *testing.T) {
		q := &Query{}
		testTime := time.Now().Add(-24 * time.Hour)
		CreatedAfter(testTime)(q)
		assert.False(t, q.CreatedAfter.IsZero())
	})
}
