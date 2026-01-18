package orchestration

import (
	"context"
	"fmt"
	"math"
	"strings"

	"github.com/valksor/go-mehrhof/internal/memory"
	"github.com/valksor/go-mehrhof/internal/storage"
)

const (
	// minWordLength is the minimum length for a word to be considered in similarity calculations.
	minWordLength = 4
)

// EmbeddingModel is the interface for embedding models.
type EmbeddingModel interface {
	Embed(ctx context.Context, text string) ([]float32, error)
}

// cosineSimilarity calculates the cosine similarity between two vectors.
func cosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) {
		return 0
	}

	var dotProduct float32
	var normA float32
	var normB float32

	for i := range a {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (float32(math.Sqrt(float64(normA))) * float32(math.Sqrt(float64(normB))))
}

// calculateSemanticSimilarity calculates similarity between two strings using embeddings.
func calculateSemanticSimilarity(ctx context.Context, a, b string, embedder EmbeddingModel) float32 {
	if a == b {
		return 1.0
	}

	// Generate embeddings
	embeddingA, err := embedder.Embed(ctx, a)
	if err != nil {
		// Fall back to string similarity if embedding fails
		return calculateStringSimilarity(a, b)
	}

	embeddingB, err := embedder.Embed(ctx, b)
	if err != nil {
		return calculateStringSimilarity(a, b)
	}

	// Calculate cosine similarity
	return cosineSimilarity(embeddingA, embeddingB)
}

// buildConsensus builds consensus from multiple agent outputs.
func (o *Orchestrator) buildConsensus(ctx context.Context, result *PipelineResult, task *storage.TaskWork) (*ConsensusResult, error) {
	// Collect votes
	votes := make(map[string]string)
	for stepName, stepResult := range result.StepResults {
		votes[stepName] = stepResult.Output
	}

	// Calculate agreement based on mode
	agreement := o.calculateAgreement(ctx, votes)

	// Build synthesized result
	var synthesized string
	var reasoning string

	switch o.config.Consensus.Mode {
	case "majority":
		synthesized, reasoning = o.buildMajorityConsensus(votes, agreement)
	case "unanimous":
		synthesized, reasoning = o.buildUnanimousConsensus(votes, agreement)
	case "any":
		synthesized, reasoning = o.buildAnyConsensus(votes)
	default:
		return nil, fmt.Errorf("unsupported consensus mode: %s", o.config.Consensus.Mode)
	}

	// If synthesizer is configured, use it to refine the result
	if o.config.Consensus.Synthesizer != "" {
		refined, err := o.synthesizeResults(ctx, synthesized, votes, task)
		if err != nil {
			return nil, fmt.Errorf("synthesize results: %w", err)
		}
		synthesized = refined
	}

	return &ConsensusResult{
		Agreement:   agreement,
		Votes:       votes,
		Synthesized: synthesized,
		Reasoning:   reasoning,
	}, nil
}

// calculateAgreement calculates the agreement level between outputs.
func (o *Orchestrator) calculateAgreement(ctx context.Context, votes map[string]string) float32 {
	if len(votes) == 0 {
		return 0
	}

	// Determine similarity mode: "semantic" or "jaccard" (default)
	similarityMode := o.config.Consensus.Similarity
	if similarityMode == "" {
		similarityMode = "jaccard"
	}

	// Prepare embedder for semantic mode
	var embedder EmbeddingModel
	if similarityMode == "semantic" {
		var err error
		embedder, err = memory.NewLocalHashEmbedding("", o.config.Consensus.SemanticModel)
		// If embedder creation failed, fall back to jaccard
		if err != nil {
			similarityMode = "jaccard"
		}
	}

	totalSimilarity := float32(0)
	comparisons := 0

	voteList := make([]string, 0, len(votes))
	for _, vote := range votes {
		voteList = append(voteList, vote)
	}

	for i := range voteList {
		for j := i + 1; j < len(voteList); j++ {
			var similarity float32
			if similarityMode == "semantic" {
				similarity = calculateSemanticSimilarity(ctx, voteList[i], voteList[j], embedder)
			} else {
				similarity = calculateStringSimilarity(voteList[i], voteList[j])
			}
			totalSimilarity += similarity
			comparisons++
		}
	}

	if comparisons == 0 {
		return 1.0 // Only one vote, full agreement
	}

	return totalSimilarity / float32(comparisons)
}

// calculateStringSimilarity calculates similarity between two strings (0-1).
func calculateStringSimilarity(a, b string) float32 {
	if a == b {
		return 1.0
	}

	// Simple word overlap similarity
	wordsA := make(map[string]bool)
	for _, word := range strings.Fields(strings.ToLower(a)) {
		if len(word) >= minWordLength { // Skip short words
			wordsA[word] = true
		}
	}

	wordsB := make(map[string]bool)
	for _, word := range strings.Fields(strings.ToLower(b)) {
		if len(word) >= minWordLength {
			wordsB[word] = true
		}
	}

	// Calculate Jaccard similarity
	intersection := 0
	union := len(wordsA)

	for word := range wordsB {
		union++
		if wordsA[word] {
			intersection++
		}
	}

	if union == 0 {
		return 0
	}

	return float32(intersection) / float32(union)
}

// buildMajorityConsensus builds consensus from majority opinion.
func (o *Orchestrator) buildMajorityConsensus(votes map[string]string, agreement float32) (string, string) {
	// Find the most common output
	outputCount := make(map[string]int)
	for _, output := range votes {
		outputCount[output]++
	}

	var majorityOutput string
	maxCount := 0
	for output, count := range outputCount {
		if count > maxCount {
			maxCount = count
			majorityOutput = output
		}
	}

	// Check minimum votes
	minVotes := o.config.Consensus.MinVotes
	if minVotes == 0 {
		minVotes = (len(votes) / 2) + 1
	}

	if maxCount < minVotes {
		return majorityOutput, fmt.Sprintf("No clear majority (%d/%d votes). Using most common output.",
			maxCount, len(votes))
	}

	return majorityOutput, fmt.Sprintf("Majority consensus reached (%d/%d votes, %.0f%% agreement).",
		maxCount, len(votes), agreement*100)
}

// buildUnanimousConsensus checks if all agents agree.
func (o *Orchestrator) buildUnanimousConsensus(votes map[string]string, agreement float32) (string, string) {
	// Collect unique outputs
	outputs := make(map[string]bool)
	for _, output := range votes {
		outputs[output] = true
	}

	// Check if unanimous (only one unique output)
	if len(outputs) == 1 {
		// Get the single output from the map
		for output := range outputs {
			return output, fmt.Sprintf("Unanimous consensus reached (%.0f%% agreement).", agreement*100)
		}
	}

	// Not unanimous - get first output with warning
	for _, output := range votes {
		return output, fmt.Sprintf("WARNING: No unanimous consensus (%.0f%% agreement, %d different outputs). Using first output.",
			agreement*100, len(outputs))
	}

	// This should never be reached since votes is non-empty
	return "", "WARNING: No votes received"
}

// buildAnyConsensus uses any valid output (first successful one).
func (o *Orchestrator) buildAnyConsensus(votes map[string]string) (string, string) {
	// Get first output
	for _, output := range votes {
		return output, fmt.Sprintf("Using any available output (%d agent(s) executed).", len(votes))
	}

	// This should never be reached since votes is non-empty
	return "", "WARNING: No votes received"
}

// synthesizeResults uses a synthesizer agent to combine results.
func (o *Orchestrator) synthesizeResults(ctx context.Context, _ string, votes map[string]string, task *storage.TaskWork) (string, error) {
	// Get synthesizer agent
	agentInst, err := o.registry.Get(o.config.Consensus.Synthesizer)
	if err != nil {
		return "", fmt.Errorf("get synthesizer agent: %w", err)
	}

	// Build synthesis prompt
	var sb strings.Builder
	sb.WriteString("You are a synthesizer. Your task is to combine the following agent outputs into a single, coherent result.\n\n")
	sb.WriteString(fmt.Sprintf("Task: %s\n\n", task.Metadata.Title))
	sb.WriteString("Agent Outputs:\n\n")

	for stepName, output := range votes {
		sb.WriteString(fmt.Sprintf("### %s\n%s\n\n", stepName, output))
	}

	sb.WriteString("Please synthesize these outputs into a single, comprehensive result that incorporates the best aspects of each.")

	// Execute synthesizer
	taskCtx := TaskContext{
		TaskID:  task.Metadata.ID,
		Title:   task.Metadata.Title,
		State:   "synthesizing",
		WorkDir: o.storage.WorkPath(task.Metadata.ID),
	}

	synthesized, _, _, err := executeAgent(ctx, agentInst, &taskCtx, sb.String())
	if err != nil {
		return "", fmt.Errorf("execute synthesizer: %w", err)
	}

	return synthesized, nil
}
