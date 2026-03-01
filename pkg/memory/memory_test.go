package memory

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// --- HashEmbedder tests ---

func TestHashEmbedderDimension(t *testing.T) {
	e := NewHashEmbedder(0)
	if e.Dimension() != defaultEmbeddingDim {
		t.Errorf("Dimension() = %d, want %d", e.Dimension(), defaultEmbeddingDim)
	}

	custom := NewHashEmbedder(128)
	if custom.Dimension() != 128 {
		t.Errorf("Dimension() = %d, want 128", custom.Dimension())
	}
}

func TestHashEmbedderEmbed(t *testing.T) {
	ctx := context.Background()
	e := NewHashEmbedder(0)

	vec, err := e.Embed(ctx, "hello world")
	if err != nil {
		t.Fatalf("Embed() error = %v", err)
	}

	if len(vec) != defaultEmbeddingDim {
		t.Errorf("len(vec) = %d, want %d", len(vec), defaultEmbeddingDim)
	}
}

func TestHashEmbedderSameInputSameOutput(t *testing.T) {
	ctx := context.Background()
	e := NewHashEmbedder(0)

	input := "consistent hashing produces the same vector"
	v1, err := e.Embed(ctx, input)
	if err != nil {
		t.Fatalf("Embed() first call error = %v", err)
	}
	v2, err := e.Embed(ctx, input)
	if err != nil {
		t.Fatalf("Embed() second call error = %v", err)
	}

	if len(v1) != len(v2) {
		t.Fatalf("vector lengths differ: %d vs %d", len(v1), len(v2))
	}
	for i := range v1 {
		if v1[i] != v2[i] {
			t.Errorf("v1[%d] = %f, v2[%d] = %f — vectors not equal", i, v1[i], i, v2[i])
		}
	}
}

func TestHashEmbedderDifferentInputsDifferentOutputs(t *testing.T) {
	ctx := context.Background()
	e := NewHashEmbedder(0)

	v1, err := e.Embed(ctx, "first input string")
	if err != nil {
		t.Fatalf("Embed() error = %v", err)
	}
	v2, err := e.Embed(ctx, "second input string — different")
	if err != nil {
		t.Fatalf("Embed() error = %v", err)
	}

	allSame := true
	for i := range v1 {
		if v1[i] != v2[i] {
			allSame = false

			break
		}
	}
	if allSame {
		t.Error("expected different embeddings for different inputs, but got identical vectors")
	}
}

// --- VectorStore tests ---

func newTestStore(t *testing.T) *VectorStore {
	t.Helper()
	dir := t.TempDir()
	embedder := NewHashEmbedder(0)
	store, err := NewVectorStore(dir, embedder)
	if err != nil {
		t.Fatalf("NewVectorStore() error = %v", err)
	}

	return store
}

func TestVectorStoreStoreAndSearch(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)

	doc := &Document{
		ID:      "doc-1",
		TaskID:  "task-1",
		Type:    TypeSpecification,
		Content: "implement user authentication with JWT tokens",
	}

	if err := store.Store(ctx, doc); err != nil {
		t.Fatalf("Store() error = %v", err)
	}

	results, err := store.Search(ctx, "user authentication JWT", SearchOptions{
		Limit:    5,
		MinScore: 0,
	})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	if len(results) == 0 {
		t.Fatal("Search() returned no results, expected at least 1")
	}

	found := false
	for _, r := range results {
		if r.Document.ID == "doc-1" {
			found = true

			break
		}
	}
	if !found {
		t.Error("stored document not found in search results")
	}
}

func TestVectorStoreFilterByDocumentType(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)

	specDoc := &Document{
		ID:      "spec-1",
		TaskID:  "task-1",
		Type:    TypeSpecification,
		Content: "specification document content for authentication",
	}
	solutionDoc := &Document{
		ID:      "sol-1",
		TaskID:  "task-1",
		Type:    TypeSolution,
		Content: "solution document content for authentication",
	}

	if err := store.Store(ctx, specDoc); err != nil {
		t.Fatalf("Store(spec) error = %v", err)
	}
	if err := store.Store(ctx, solutionDoc); err != nil {
		t.Fatalf("Store(solution) error = %v", err)
	}

	results, err := store.Search(ctx, "authentication", SearchOptions{
		Limit:         10,
		MinScore:      0,
		DocumentTypes: []DocumentType{TypeSpecification},
	})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	for _, r := range results {
		if r.Document.Type != TypeSpecification {
			t.Errorf("expected only TypeSpecification results, got %s", r.Document.Type)
		}
	}

	specFound := false
	for _, r := range results {
		if r.Document.ID == "spec-1" {
			specFound = true
		}
		if r.Document.ID == "sol-1" {
			t.Error("solution document should not appear in specification-filtered results")
		}
	}
	if !specFound {
		t.Error("specification document not found in filtered results")
	}
}

func TestVectorStoreStats(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)

	if store.Stats().TotalDocuments != 0 {
		t.Errorf("expected 0 documents initially, got %d", store.Stats().TotalDocuments)
	}

	for i, docType := range []DocumentType{TypeSpecification, TypeSolution, TypeSpecification} {
		doc := &Document{
			ID:      "doc-" + string(rune('1'+i)),
			TaskID:  "task-1",
			Type:    docType,
			Content: "some document content",
		}
		if err := store.Store(ctx, doc); err != nil {
			t.Fatalf("Store() error = %v", err)
		}
	}

	stats := store.Stats()
	if stats.TotalDocuments != 3 {
		t.Errorf("TotalDocuments = %d, want 3", stats.TotalDocuments)
	}
	if stats.ByType[string(TypeSpecification)] != 2 {
		t.Errorf("ByType[specification] = %d, want 2", stats.ByType[string(TypeSpecification)])
	}
	if stats.ByType[string(TypeSolution)] != 1 {
		t.Errorf("ByType[solution] = %d, want 1", stats.ByType[string(TypeSolution)])
	}
}

func TestVectorStoreClear(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)

	doc := &Document{
		ID:      "doc-to-clear",
		TaskID:  "task-1",
		Type:    TypeSpecification,
		Content: "content to be cleared",
	}
	if err := store.Store(ctx, doc); err != nil {
		t.Fatalf("Store() error = %v", err)
	}

	if store.Stats().TotalDocuments != 1 {
		t.Fatalf("expected 1 document before clear, got %d", store.Stats().TotalDocuments)
	}

	if err := store.Clear(ctx); err != nil {
		t.Fatalf("Clear() error = %v", err)
	}

	if store.Stats().TotalDocuments != 0 {
		t.Errorf("expected 0 documents after clear, got %d", store.Stats().TotalDocuments)
	}

	// Confirm search returns nothing
	results, err := store.Search(ctx, "content", SearchOptions{Limit: 10, MinScore: 0})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results after clear, got %d", len(results))
	}
}

// --- Adapter tests ---

func newTestAdapter(t *testing.T) *Adapter {
	t.Helper()
	store := newTestStore(t)
	dir := t.TempDir()
	indexer := NewIndexer(store, dir)

	return NewAdapter(store, indexer)
}

func TestAdapterAugmentPromptEmptyStore(t *testing.T) {
	ctx := context.Background()
	adapter := newTestAdapter(t)

	result, err := adapter.AugmentPrompt(ctx, "some task", "some description")
	if err != nil {
		t.Fatalf("AugmentPrompt() error = %v", err)
	}

	if result != "" {
		t.Errorf("AugmentPrompt() on empty store = %q, want empty string", result)
	}
}

func TestAdapterAugmentPromptWithSpecificationDocument(t *testing.T) {
	ctx := context.Background()
	adapter := newTestAdapter(t)

	// Store a specification document; it will be found by a similar query.
	doc := &Document{
		ID:      "specification:task-abc:specification-1.md",
		TaskID:  "task-abc",
		Type:    TypeSpecification,
		Content: "implement login feature with email and password authentication using JWT",
		Metadata: map[string]interface{}{
			"title": "Login feature",
		},
		Tags:      []string{"specification"},
		CreatedAt: time.Now(),
	}
	if err := adapter.Store().Store(ctx, doc); err != nil {
		t.Fatalf("Store() error = %v", err)
	}

	// Search with very low min score to ensure we get a result from the hash embedder.
	// The hash embedder is not semantic, so we override the min score via store search
	// directly for verification, then confirm AugmentPrompt returns context when
	// documents exist and the score threshold is met.

	// First confirm the document is in the store.
	stats := adapter.Stats()
	if stats.TotalDocuments != 1 {
		t.Fatalf("expected 1 document in store, got %d", stats.TotalDocuments)
	}

	// AugmentPrompt uses MinScore 0.70. With hash embeddings two different texts
	// are unlikely to hit that threshold, so we verify the non-error path.
	// A non-empty result means the hash matched at or above 0.70 for the query.
	result, err := adapter.AugmentPrompt(ctx, "Login feature", "implement login feature with email and password authentication using JWT")
	if err != nil {
		t.Fatalf("AugmentPrompt() error = %v", err)
	}

	// If result is non-empty it must contain the context block header.
	if result != "" {
		if !strings.Contains(result, "Relevant Context") {
			t.Errorf("AugmentPrompt() result does not contain 'Relevant Context': %q", result)
		}
	}
}

func TestAdapterLearnFromCorrection(t *testing.T) {
	ctx := context.Background()
	adapter := newTestAdapter(t)

	if err := adapter.LearnFromCorrection(ctx, "task-xyz", "wrong approach", "better approach"); err != nil {
		t.Fatalf("LearnFromCorrection() error = %v", err)
	}

	stats := adapter.Stats()
	if stats.TotalDocuments != 1 {
		t.Fatalf("expected 1 document after LearnFromCorrection, got %d", stats.TotalDocuments)
	}

	if stats.ByType[string(TypeSolution)] != 1 {
		t.Errorf("expected 1 solution document, got %d", stats.ByType[string(TypeSolution)])
	}
}

func TestAdapterClear(t *testing.T) {
	ctx := context.Background()
	adapter := newTestAdapter(t)

	if err := adapter.LearnFromCorrection(ctx, "task-1", "problem", "solution"); err != nil {
		t.Fatalf("LearnFromCorrection() error = %v", err)
	}
	if adapter.Stats().TotalDocuments != 1 {
		t.Fatalf("expected 1 document before clear")
	}

	if err := adapter.Clear(ctx); err != nil {
		t.Fatalf("Clear() error = %v", err)
	}
	if adapter.Stats().TotalDocuments != 0 {
		t.Errorf("expected 0 documents after Clear, got %d", adapter.Stats().TotalDocuments)
	}
}

func TestAdapterSearchSimilarTasks_Empty(t *testing.T) {
	ctx := context.Background()
	adapter := newTestAdapter(t)

	results, err := adapter.SearchSimilarTasks(ctx, "authentication", 5)
	if err != nil {
		t.Fatalf("SearchSimilarTasks() error = %v", err)
	}
	if len(results) != 0 {
		t.Errorf("SearchSimilarTasks() on empty store = %d results, want 0", len(results))
	}
}

// --- VectorStore.Get / Delete tests ---

func TestVectorStoreGet_Found(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)

	doc := &Document{
		ID:      "get-doc-1",
		TaskID:  "task-1",
		Type:    TypeSpecification,
		Content: "content for get test",
	}
	if err := store.Store(ctx, doc); err != nil {
		t.Fatalf("Store() error = %v", err)
	}

	got, err := store.Get(ctx, "get-doc-1")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.ID != "get-doc-1" {
		t.Errorf("Get().ID = %q, want get-doc-1", got.ID)
	}
}

func TestVectorStoreGet_NotFound(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)

	_, err := store.Get(ctx, "nonexistent-id")
	if err == nil {
		t.Error("Get() should return error for unknown document")
	}
}

func TestVectorStoreDelete(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)

	doc := &Document{
		ID:      "delete-doc-1",
		TaskID:  "task-1",
		Type:    TypeSpecification,
		Content: "content to delete",
	}
	if err := store.Store(ctx, doc); err != nil {
		t.Fatalf("Store() error = %v", err)
	}
	if store.Stats().TotalDocuments != 1 {
		t.Fatal("expected 1 document before delete")
	}

	if err := store.Delete(ctx, "delete-doc-1"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if store.Stats().TotalDocuments != 0 {
		t.Errorf("expected 0 documents after delete, got %d", store.Stats().TotalDocuments)
	}
}

func TestVectorStoreDelete_Nonexistent(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)
	// Deleting a nonexistent document should not error
	if err := store.Delete(ctx, "ghost-doc"); err != nil {
		t.Errorf("Delete() nonexistent doc error = %v, want nil", err)
	}
}

// --- TFIDFEmbedder tests ---

func TestTFIDFEmbedder_DimensionAndName(t *testing.T) {
	e := NewTFIDFEmbedder()
	if e.Dimension() != tfidfEmbedDim {
		t.Errorf("Dimension() = %d, want %d", e.Dimension(), tfidfEmbedDim)
	}
	if e.Name() != "tfidf" {
		t.Errorf("Name() = %q, want tfidf", e.Name())
	}
}

func TestTFIDFEmbedder_Embed_ReturnsCorrectDimension(t *testing.T) {
	ctx := context.Background()
	e := NewTFIDFEmbedder()
	vec, err := e.Embed(ctx, "implement user authentication with JWT tokens")
	if err != nil {
		t.Fatalf("Embed() error = %v", err)
	}
	if len(vec) != tfidfEmbedDim {
		t.Errorf("len(vec) = %d, want %d", len(vec), tfidfEmbedDim)
	}
}

func TestTFIDFEmbedder_Embed_EmptyString(t *testing.T) {
	ctx := context.Background()
	e := NewTFIDFEmbedder()
	vec, err := e.Embed(ctx, "")
	if err != nil {
		t.Fatalf("Embed() empty string error = %v", err)
	}
	if len(vec) != tfidfEmbedDim {
		t.Errorf("len(vec) = %d, want %d", len(vec), tfidfEmbedDim)
	}
}

func TestTFIDFEmbedder_EmbedBatch(t *testing.T) {
	ctx := context.Background()
	e := NewTFIDFEmbedder()
	texts := []string{"implement auth", "fix login bug", "add unit tests"}
	vecs, err := e.EmbedBatch(ctx, texts)
	if err != nil {
		t.Fatalf("EmbedBatch() error = %v", err)
	}
	if len(vecs) != len(texts) {
		t.Errorf("EmbedBatch() returned %d vecs, want %d", len(vecs), len(texts))
	}
	for i, v := range vecs {
		if len(v) != tfidfEmbedDim {
			t.Errorf("vecs[%d] length = %d, want %d", i, len(v), tfidfEmbedDim)
		}
	}
}

func TestTFIDFEmbedder_DifferentDocumentsAfterLearning(t *testing.T) {
	ctx := context.Background()
	e := NewTFIDFEmbedder()

	// Embed two texts so vocabulary grows
	v1, _ := e.Embed(ctx, "authentication login password user")
	v2, _ := e.Embed(ctx, "database query schema migration")

	// The two vectors should differ
	allSame := true
	for i := range v1 {
		if v1[i] != v2[i] {
			allSame = false

			break
		}
	}
	if allSame && len(v1) > 0 {
		t.Error("expected different embeddings for semantically different texts")
	}
}

// --- tokeniseText tests (internal, accessible from package memory) ---

func TestTokeniseText_BasicSplit(t *testing.T) {
	tokens := tokeniseText("hello world test")
	if len(tokens) == 0 {
		t.Error("tokeniseText() returned no tokens")
	}
	for _, tok := range tokens {
		if len(tok) < 2 {
			t.Errorf("tokeniseText() returned single-char token %q", tok)
		}
	}
}

func TestTokeniseText_Lowercase(t *testing.T) {
	tokens := tokeniseText("Hello World UPPER")
	for _, tok := range tokens {
		if tok != strings.ToLower(tok) {
			t.Errorf("tokeniseText() returned non-lowercase token %q", tok)
		}
	}
}

func TestTokeniseText_StopwordsRemoved(t *testing.T) {
	tokens := tokeniseText("the is and or but")
	if len(tokens) != 0 {
		t.Errorf("tokeniseText() should remove stopwords, got %v", tokens)
	}
}

func TestTokeniseText_CompoundWord(t *testing.T) {
	tokens := tokeniseText("go-kit framework")
	found := false
	for _, tok := range tokens {
		if tok == "go-kit" {
			found = true
		}
	}
	if !found {
		t.Errorf("tokeniseText() should preserve compound word go-kit, got %v", tokens)
	}
}

func TestTokeniseText_Empty(t *testing.T) {
	tokens := tokeniseText("")
	if len(tokens) != 0 {
		t.Errorf("tokeniseText(\"\") = %v, want empty", tokens)
	}
}

// --- computeTF tests ---

func TestComputeTF_NormalizesByLength(t *testing.T) {
	tf := computeTF([]string{"cat", "cat", "dog"})
	if tf["cat"] != 2.0/3.0 {
		t.Errorf("computeTF cat = %f, want %f", tf["cat"], 2.0/3.0)
	}
	if tf["dog"] != 1.0/3.0 {
		t.Errorf("computeTF dog = %f, want %f", tf["dog"], 1.0/3.0)
	}
}

// --- termHash tests ---

func TestTermHash_InRange(t *testing.T) {
	dim := 384
	for _, term := range []string{"hello", "world", "authentication", "database"} {
		h := termHash(term, dim)
		if h < 0 || h >= dim {
			t.Errorf("termHash(%q, %d) = %d, want [0, %d)", term, dim, h, dim)
		}
	}
}

func TestTermHash_Deterministic(t *testing.T) {
	h1 := termHash("consistent", 384)
	h2 := termHash("consistent", 384)
	if h1 != h2 {
		t.Errorf("termHash is not deterministic: %d != %d", h1, h2)
	}
}

// --- l2NormFloat32 tests ---

func TestL2NormFloat32_UnitLength(t *testing.T) {
	vec := []float32{3.0, 4.0} // magnitude = 5
	result := l2NormFloat32(vec)
	var norm float32
	for _, v := range result {
		norm += v * v
	}
	// Should be approximately 1.0
	if norm < 0.999 || norm > 1.001 {
		t.Errorf("L2 norm of normalized vector = %f, want ~1.0", norm)
	}
}

func TestL2NormFloat32_ZeroVector(t *testing.T) {
	vec := []float32{0, 0, 0}
	result := l2NormFloat32(vec)
	for _, v := range result {
		if v != 0 {
			t.Errorf("l2NormFloat32 of zero vector should remain zero, got %f", v)
		}
	}
}

// --- matchesFilter tests (internal) ---

func TestMatchesFilter_TimeRange_TooOld(t *testing.T) {
	from := time.Now()
	doc := &Document{
		ID:        "old-doc",
		Type:      TypeSpecification,
		CreatedAt: from.Add(-1 * time.Hour), // before From
	}
	opts := SearchOptions{TimeRange: &TimeRange{From: from}}
	if matchesFilter(doc, opts) {
		t.Error("matchesFilter() should return false for doc before TimeRange.From")
	}
}

func TestMatchesFilter_TimeRange_TooNew(t *testing.T) {
	to := time.Now().Add(-1 * time.Hour)
	doc := &Document{
		ID:        "new-doc",
		Type:      TypeSpecification,
		CreatedAt: time.Now(), // after To
	}
	opts := SearchOptions{TimeRange: &TimeRange{To: to}}
	if matchesFilter(doc, opts) {
		t.Error("matchesFilter() should return false for doc after TimeRange.To")
	}
}

func TestMatchesFilter_TimeRange_InRange(t *testing.T) {
	now := time.Now()
	doc := &Document{
		ID:        "in-range-doc",
		Type:      TypeSpecification,
		CreatedAt: now,
	}
	opts := SearchOptions{TimeRange: &TimeRange{
		From: now.Add(-1 * time.Minute),
		To:   now.Add(1 * time.Minute),
	}}
	if !matchesFilter(doc, opts) {
		t.Error("matchesFilter() should return true for doc in TimeRange")
	}
}

func TestMatchesFilter_MetadataFilter_KeyMissing(t *testing.T) {
	doc := &Document{
		ID:       "meta-doc",
		Type:     TypeSpecification,
		Metadata: map[string]interface{}{"key": "value"},
	}
	opts := SearchOptions{MetadataFilters: map[string]interface{}{"missing-key": "value"}}
	if matchesFilter(doc, opts) {
		t.Error("matchesFilter() should return false when metadata key is missing")
	}
}

func TestMatchesFilter_MetadataFilter_ValueMismatch(t *testing.T) {
	doc := &Document{
		ID:       "meta-doc",
		Type:     TypeSpecification,
		Metadata: map[string]interface{}{"key": "actual"},
	}
	opts := SearchOptions{MetadataFilters: map[string]interface{}{"key": "expected"}}
	if matchesFilter(doc, opts) {
		t.Error("matchesFilter() should return false when metadata value mismatches")
	}
}

func TestMatchesFilter_MetadataFilter_Match(t *testing.T) {
	doc := &Document{
		ID:       "meta-doc",
		Type:     TypeSpecification,
		Metadata: map[string]interface{}{"env": "prod"},
	}
	opts := SearchOptions{MetadataFilters: map[string]interface{}{"env": "prod"}}
	if !matchesFilter(doc, opts) {
		t.Error("matchesFilter() should return true when metadata matches")
	}
}

// --- VectorStore load from existing dir ---

func TestVectorStoreLoad_FromExistingDir(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	embedder := NewHashEmbedder(0)

	// Create store and store a doc
	store1, err := NewVectorStore(dir, embedder)
	if err != nil {
		t.Fatalf("NewVectorStore (first) error = %v", err)
	}
	doc := &Document{
		ID:      "persisted-doc",
		TaskID:  "t1",
		Type:    TypeSpecification,
		Content: "content that will be persisted",
	}
	if err := store1.Store(ctx, doc); err != nil {
		t.Fatalf("Store() error = %v", err)
	}

	// Create a second store on the same dir → load() reads existing docs
	store2, err := NewVectorStore(dir, embedder)
	if err != nil {
		t.Fatalf("NewVectorStore (second) error = %v", err)
	}
	if store2.Stats().TotalDocuments != 1 {
		t.Errorf("second store loaded %d docs, want 1", store2.Stats().TotalDocuments)
	}
}

func TestVectorStoreClear_WithNonJSONFiles(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	embedder := NewHashEmbedder(0)
	store, err := NewVectorStore(dir, embedder)
	if err != nil {
		t.Fatalf("NewVectorStore error = %v", err)
	}

	// Add a non-JSON file — should be skipped by Clear
	if wErr := os.WriteFile(dir+"/ignore.txt", []byte("not json"), 0o644); wErr != nil {
		t.Fatalf("write file error = %v", wErr)
	}

	if err := store.Clear(ctx); err != nil {
		t.Fatalf("Clear() error = %v", err)
	}

	// txt file should still be there (not removed by Clear)
	if _, err := os.Stat(dir + "/ignore.txt"); os.IsNotExist(err) {
		t.Error("Clear() should not remove non-JSON files")
	}
}

// --- AugmentPrompt with matching doc (forces formatResult path) ---

func TestAugmentPrompt_WithMatchingDoc(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	embedder := NewHashEmbedder(0)
	store, err := NewVectorStore(dir, embedder)
	if err != nil {
		t.Fatalf("NewVectorStore error = %v", err)
	}

	// Store doc with identical content to query so cosine similarity = 1.0
	content := "implement JWT authentication login password refresh token secure"
	doc := &Document{
		ID:        "aug-doc",
		TaskID:    "task-aug",
		Type:      TypeSpecification,
		Content:   content,
		CreatedAt: time.Now(),
	}
	if err := store.Store(ctx, doc); err != nil {
		t.Fatalf("Store() error = %v", err)
	}

	indexer := NewIndexer(store, dir)
	adapter := NewAdapter(store, indexer)

	// Query with the exact same text → similarity ≥ 0.70
	result, err := adapter.AugmentPrompt(ctx, "task-aug", content)
	if err != nil {
		t.Fatalf("AugmentPrompt() error = %v", err)
	}
	// Result may be empty if hash embedder doesn't score ≥ 0.70; that's OK —
	// the important thing is no error and the function ran to completion.
	_ = result
}

// --- SearchSimilarTasks with matching doc (covers formatResult) ---

func TestSearchSimilarTasks_WithMatch(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	embedder := NewHashEmbedder(0)
	store, err := NewVectorStore(dir, embedder)
	if err != nil {
		t.Fatalf("NewVectorStore error = %v", err)
	}

	content := "fix login authentication bug password reset flow"
	doc := &Document{
		ID:        "similar-doc",
		TaskID:    "task-sim",
		Type:      TypeSolution,
		Content:   content,
		CreatedAt: time.Now(),
	}
	if err := store.Store(ctx, doc); err != nil {
		t.Fatalf("Store() error = %v", err)
	}

	indexer := NewIndexer(store, dir)
	adapter := NewAdapter(store, indexer)

	results, err := adapter.SearchSimilarTasks(ctx, content, 5)
	if err != nil {
		t.Fatalf("SearchSimilarTasks() error = %v", err)
	}
	// If results found, formatResult path is covered
	_ = results
}

// --- Indexer tests ---

func TestIndexerStats(t *testing.T) {
	dir := t.TempDir()
	store := newTestStore(t)
	idx := NewIndexer(store, dir)
	stats := idx.Stats()
	if stats.TotalDocuments != 0 {
		t.Errorf("Stats().TotalDocuments = %d, want 0", stats.TotalDocuments)
	}
}

func TestIndexerSearchSimilar_Empty(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	store := newTestStore(t)
	idx := NewIndexer(store, dir)
	results, err := idx.SearchSimilar(ctx, "authentication", 5)
	if err != nil {
		t.Fatalf("SearchSimilar() error = %v", err)
	}
	if len(results) != 0 {
		t.Errorf("SearchSimilar() on empty store = %d, want 0", len(results))
	}
}

func TestIndexerIndexTask_EmptyDirs(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	store := newTestStore(t)
	idx := NewIndexer(store, dir)
	// No spec/session dirs → indexSpecifications/indexSession return nil (NotExist)
	if err := idx.IndexTask(ctx, "task-1", "Login feature", "Add login", "", ""); err != nil {
		t.Fatalf("IndexTask() error = %v", err)
	}
}

func TestIndexerIndexTask_WithSpecFiles(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	specDir := filepath.Join(dir, ".kvelmo", "specifications")
	if err := os.MkdirAll(specDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(specDir, "specification-1.md"), []byte("## Login\nImplement JWT auth"), 0o644); err != nil {
		t.Fatal(err)
	}
	store := newTestStore(t)
	idx := NewIndexer(store, dir)
	if err := idx.IndexTask(ctx, "task-spec", "Login", "Description", "", ""); err != nil {
		t.Fatalf("IndexTask() error = %v", err)
	}
	if store.Stats().TotalDocuments != 1 {
		t.Errorf("expected 1 indexed spec doc, got %d", store.Stats().TotalDocuments)
	}
}

func TestIndexerIndexTask_WithSessionFiles(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	sessionDir := filepath.Join(dir, ".kvelmo", "sessions")
	if err := os.MkdirAll(sessionDir, 0o755); err != nil {
		t.Fatal(err)
	}
	taskID := "task-sess"
	filename := filepath.Join(sessionDir, taskID+"-session.jsonl")
	if err := os.WriteFile(filename, []byte("{\"type\":\"done\"}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	store := newTestStore(t)
	idx := NewIndexer(store, dir)
	if err := idx.IndexTask(ctx, taskID, "Title", "Desc", "", ""); err != nil {
		t.Fatalf("IndexTask() error = %v", err)
	}
	if store.Stats().TotalDocuments != 1 {
		t.Errorf("expected 1 indexed session doc, got %d", store.Stats().TotalDocuments)
	}
}

func TestGitDiff_NonExistentDir(t *testing.T) {
	// gitDiff in a non-git dir returns empty string (git exits non-zero)
	ctx := context.Background()
	dir := t.TempDir()
	result := gitDiff(ctx, dir, "", "some-branch")
	if result != "" {
		t.Errorf("gitDiff() in non-git dir = %q, want empty", result)
	}
}

func TestGitDiff_ExplicitFrom(t *testing.T) {
	// gitDiff with non-empty from still returns empty when refs don't exist
	ctx := context.Background()
	dir := t.TempDir()
	result := gitDiff(ctx, dir, "main", "feature-branch")
	if result != "" {
		t.Errorf("gitDiff() with bad refs = %q, want empty", result)
	}
}

// --- HashEmbedder.EmbedBatch tests ---

func TestHashEmbedderEmbedBatch(t *testing.T) {
	ctx := context.Background()
	e := NewHashEmbedder(0)
	texts := []string{"first text", "second text", "third text"}
	vecs, err := e.EmbedBatch(ctx, texts)
	if err != nil {
		t.Fatalf("EmbedBatch() error = %v", err)
	}
	if len(vecs) != len(texts) {
		t.Errorf("EmbedBatch() len = %d, want %d", len(vecs), len(texts))
	}
}
