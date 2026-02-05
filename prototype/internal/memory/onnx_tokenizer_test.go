package memory

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// createTestTokenizer creates a minimal tokenizer.json for testing.
func createTestTokenizer(t *testing.T, dir string) {
	t.Helper()

	vocab := map[string]int{
		"[PAD]":    0,
		"[UNK]":    1,
		"[CLS]":    2,
		"[SEP]":    3,
		"[MASK]":   4,
		"hello":    5,
		"world":    6,
		"test":     7,
		"the":      8,
		"a":        9,
		"##ing":    10,
		"##ed":     11,
		"##s":      12,
		"##ly":     13,
		"quick":    14,
		"brown":    15,
		"fox":      16,
		"jump":     17,
		".":        18,
		",":        19,
		"!":        20,
		"?":        21,
		"embed":    22,
		"##ding":   23,
		"semantic": 24,
		"search":   25,
	}

	tj := struct {
		Model struct {
			Type  string         `json:"type"`
			Vocab map[string]int `json:"vocab"`
		} `json:"model"`
		Normalizer struct {
			Type      string `json:"type"`
			Lowercase bool   `json:"lowercase"`
		} `json:"normalizer"`
	}{
		Model: struct {
			Type  string         `json:"type"`
			Vocab map[string]int `json:"vocab"`
		}{
			Type:  "WordPiece",
			Vocab: vocab,
		},
		Normalizer: struct {
			Type      string `json:"type"`
			Lowercase bool   `json:"lowercase"`
		}{
			Type:      "BertNormalizer",
			Lowercase: true,
		},
	}

	data, err := json.Marshal(tj)
	if err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(dir, "tokenizer.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestNewTokenizer(t *testing.T) {
	dir := t.TempDir()
	createTestTokenizer(t, dir)

	tok, err := NewTokenizer(dir, 32)
	if err != nil {
		t.Fatalf("NewTokenizer: %v", err)
	}

	if tok.maxLength != 32 {
		t.Errorf("maxLength: got %d, want 32", tok.maxLength)
	}

	if tok.vocabSize < 20 {
		t.Errorf("vocabSize too small: %d", tok.vocabSize)
	}

	if !tok.doLowerCase {
		t.Error("expected doLowerCase to be true")
	}
}

func TestNewTokenizer_DefaultMaxLength(t *testing.T) {
	dir := t.TempDir()
	createTestTokenizer(t, dir)

	tok, err := NewTokenizer(dir, 0)
	if err != nil {
		t.Fatalf("NewTokenizer: %v", err)
	}

	if tok.maxLength != 256 {
		t.Errorf("default maxLength: got %d, want 256", tok.maxLength)
	}
}

func TestTokenizer_Encode(t *testing.T) {
	dir := t.TempDir()
	createTestTokenizer(t, dir)

	tok, _ := NewTokenizer(dir, 16)

	output := tok.Encode("hello world")

	// Should start with [CLS] and have [SEP] after tokens
	if output.InputIDs[0] != int64(tok.clsTokenID) {
		t.Errorf("first token should be [CLS], got %d", output.InputIDs[0])
	}

	// Check attention mask - should be 1 for [CLS], hello, world, [SEP]
	// At positions 0, 1, 2, 3
	for i := range 4 {
		if output.AttentionMask[i] != 1 {
			t.Errorf("attention mask[%d] should be 1, got %d", i, output.AttentionMask[i])
		}
	}

	// Padding positions should be 0
	for i := 4; i < len(output.AttentionMask); i++ {
		if output.AttentionMask[i] != 0 {
			t.Errorf("attention mask[%d] should be 0 (padding), got %d", i, output.AttentionMask[i])
		}
	}

	// Token type IDs should all be 0 for single sequence
	for i, id := range output.TokenTypeIDs {
		if id != 0 {
			t.Errorf("token type[%d] should be 0, got %d", i, id)
		}
	}
}

func TestTokenizer_EncodeWithWordPiece(t *testing.T) {
	dir := t.TempDir()
	createTestTokenizer(t, dir)

	tok, _ := NewTokenizer(dir, 16)

	// "jumping" should tokenize to "jump" + "##ing"
	output := tok.Encode("jumping")

	// Find the token IDs
	jumpID := int64(tok.vocab["jump"])
	ingID := int64(tok.vocab["##ing"])

	// Should have [CLS], jump, ##ing, [SEP], then padding
	if output.InputIDs[1] != jumpID {
		t.Errorf("expected 'jump' at position 1, got %d", output.InputIDs[1])
	}

	if output.InputIDs[2] != ingID {
		t.Errorf("expected '##ing' at position 2, got %d", output.InputIDs[2])
	}

	if output.InputIDs[3] != int64(tok.sepTokenID) {
		t.Errorf("expected [SEP] at position 3, got %d", output.InputIDs[3])
	}
}

func TestTokenizer_EncodeTruncation(t *testing.T) {
	dir := t.TempDir()
	createTestTokenizer(t, dir)

	tok, _ := NewTokenizer(dir, 8) // Very short max length

	// Long text that exceeds max length
	output := tok.Encode("hello world test the quick brown fox")

	// Should be exactly maxLength
	if len(output.InputIDs) != 8 {
		t.Errorf("InputIDs length: got %d, want 8", len(output.InputIDs))
	}

	// Should still have [CLS] at start and [SEP] somewhere
	if output.InputIDs[0] != int64(tok.clsTokenID) {
		t.Error("truncated output should still start with [CLS]")
	}

	// Last non-padding token should be [SEP]
	lastReal := 0
	for i, mask := range output.AttentionMask {
		if mask == 1 {
			lastReal = i
		}
	}

	if output.InputIDs[lastReal] != int64(tok.sepTokenID) {
		t.Error("truncated output should end with [SEP]")
	}
}

func TestTokenizer_EncodeUnknownToken(t *testing.T) {
	dir := t.TempDir()
	createTestTokenizer(t, dir)

	tok, _ := NewTokenizer(dir, 16)

	// "xyz" is not in vocab
	output := tok.Encode("xyz")

	// Should have [CLS], [UNK] (x3), [SEP]
	// Actually depends on WordPiece behavior - each char becomes [UNK]
	foundUNK := false

	for i := 1; i < len(output.InputIDs)-1; i++ {
		if output.AttentionMask[i] == 0 {
			break
		}

		if output.InputIDs[i] == int64(tok.unkTokenID) {
			foundUNK = true

			break
		}
	}

	if !foundUNK {
		t.Error("unknown word should produce [UNK] token")
	}
}

func TestTokenizer_EncodeBatch(t *testing.T) {
	dir := t.TempDir()
	createTestTokenizer(t, dir)

	tok, _ := NewTokenizer(dir, 16)

	texts := []string{"hello", "world", "test"}
	outputs := tok.EncodeBatch(texts)

	if len(outputs) != 3 {
		t.Errorf("batch output count: got %d, want 3", len(outputs))
	}

	// Each output should have correct length
	for i, out := range outputs {
		if len(out.InputIDs) != 16 {
			t.Errorf("output[%d] InputIDs length: got %d, want 16", i, len(out.InputIDs))
		}
	}
}

func TestTokenizer_Decode(t *testing.T) {
	dir := t.TempDir()
	createTestTokenizer(t, dir)

	tok, _ := NewTokenizer(dir, 16)

	// Encode and decode
	output := tok.Encode("hello world")
	decoded := tok.Decode(output.InputIDs)

	if decoded != "hello world" {
		t.Errorf("decoded: got %q, want %q", decoded, "hello world")
	}
}

func TestTokenizer_DecodePunctuation(t *testing.T) {
	dir := t.TempDir()
	createTestTokenizer(t, dir)

	tok, _ := NewTokenizer(dir, 16)

	output := tok.Encode("hello!")
	decoded := tok.Decode(output.InputIDs)

	// Punctuation is split, so output will have space before !
	if decoded != "hello !" {
		t.Errorf("decoded with punct: got %q", decoded)
	}
}

func TestTokenizer_LowerCase(t *testing.T) {
	dir := t.TempDir()
	createTestTokenizer(t, dir)

	tok, _ := NewTokenizer(dir, 16)

	// HELLO should be lowercased to hello
	output := tok.Encode("HELLO")

	helloID := int64(tok.vocab["hello"])
	if output.InputIDs[1] != helloID {
		t.Errorf("uppercase HELLO should tokenize as hello, got ID %d", output.InputIDs[1])
	}
}

func TestTokenizer_VocabSize(t *testing.T) {
	dir := t.TempDir()
	createTestTokenizer(t, dir)

	tok, _ := NewTokenizer(dir, 16)

	if tok.VocabSize() < 20 {
		t.Errorf("VocabSize: got %d, expected at least 20", tok.VocabSize())
	}
}

func TestTokenizer_MaxLength(t *testing.T) {
	dir := t.TempDir()
	createTestTokenizer(t, dir)

	tok, _ := NewTokenizer(dir, 64)

	if tok.MaxLength() != 64 {
		t.Errorf("MaxLength: got %d, want 64", tok.MaxLength())
	}
}

func TestBasicTokenize(t *testing.T) {
	dir := t.TempDir()
	createTestTokenizer(t, dir)

	tok, _ := NewTokenizer(dir, 16)

	tests := []struct {
		input    string
		expected []string
	}{
		{"hello world", []string{"hello", "world"}},
		{"hello  world", []string{"hello", "world"}}, // Multiple spaces
		{"hello, world!", []string{"hello", ",", "world", "!"}},
		{"  hello  ", []string{"hello"}}, // Leading/trailing spaces
		{"", nil},                        // Empty string
	}

	for _, tt := range tests {
		result := tok.basicTokenize(tt.input)
		if len(result) != len(tt.expected) {
			t.Errorf("basicTokenize(%q): got %v, want %v", tt.input, result, tt.expected)

			continue
		}

		for i := range result {
			if result[i] != tt.expected[i] {
				t.Errorf("basicTokenize(%q)[%d]: got %q, want %q", tt.input, i, result[i], tt.expected[i])
			}
		}
	}
}
