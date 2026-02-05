package memory

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

// Tokenizer handles WordPiece tokenization for BERT-family models.
type Tokenizer struct {
	vocab       map[string]int // Token to ID mapping
	idToToken   map[int]string // ID to token mapping
	maxLength   int            // Maximum sequence length
	clsTokenID  int            // [CLS] token ID
	sepTokenID  int            // [SEP] token ID
	padTokenID  int            // [PAD] token ID
	unkTokenID  int            // [UNK] token ID
	vocabSize   int
	doLowerCase bool
}

// TokenizerOutput contains the tokenized output.
type TokenizerOutput struct {
	InputIDs      []int64 // Token IDs
	AttentionMask []int64 // 1 for real tokens, 0 for padding
	TokenTypeIDs  []int64 // Segment IDs (all 0 for single sequence)
}

// tokenizerJSON represents the HuggingFace tokenizer.json structure.
type tokenizerJSON struct {
	Model struct {
		Type  string         `json:"type"`
		Vocab map[string]int `json:"vocab"`
	} `json:"model"`
	AddedTokens []struct {
		Content string `json:"content"`
		ID      int    `json:"id"`
	} `json:"added_tokens"`
	Normalizer struct {
		Type      string `json:"type"`
		Lowercase bool   `json:"lowercase"`
	} `json:"normalizer"`
}

// NewTokenizer loads a tokenizer from a model directory.
func NewTokenizer(modelPath string, maxLength int) (*Tokenizer, error) {
	if maxLength <= 0 {
		maxLength = 256 // Default max length
	}

	tokenizerPath := filepath.Join(modelPath, "tokenizer.json")

	data, err := os.ReadFile(tokenizerPath)
	if err != nil {
		return nil, fmt.Errorf("read tokenizer.json: %w", err)
	}

	var tj tokenizerJSON
	if err := json.Unmarshal(data, &tj); err != nil {
		return nil, fmt.Errorf("parse tokenizer.json: %w", err)
	}

	// Build vocab maps
	vocab := tj.Model.Vocab
	idToToken := make(map[int]string, len(vocab))

	for token, id := range vocab {
		idToToken[id] = token
	}

	// Add any added tokens to vocab
	for _, added := range tj.AddedTokens {
		vocab[added.Content] = added.ID
		idToToken[added.ID] = added.Content
	}

	// Find special token IDs
	clsID, ok := vocab["[CLS]"]
	if !ok {
		return nil, errors.New("missing [CLS] token in vocab")
	}

	sepID, ok := vocab["[SEP]"]
	if !ok {
		return nil, errors.New("missing [SEP] token in vocab")
	}

	padID, ok := vocab["[PAD]"]
	if !ok {
		return nil, errors.New("missing [PAD] token in vocab")
	}

	unkID, ok := vocab["[UNK]"]
	if !ok {
		return nil, errors.New("missing [UNK] token in vocab")
	}

	return &Tokenizer{
		vocab:       vocab,
		idToToken:   idToToken,
		maxLength:   maxLength,
		clsTokenID:  clsID,
		sepTokenID:  sepID,
		padTokenID:  padID,
		unkTokenID:  unkID,
		vocabSize:   len(vocab),
		doLowerCase: tj.Normalizer.Lowercase,
	}, nil
}

// Encode tokenizes a single text and returns padded token IDs.
func (t *Tokenizer) Encode(text string) *TokenizerOutput {
	// Normalize text
	if t.doLowerCase {
		text = strings.ToLower(text)
	}

	// Tokenize
	tokens := t.tokenize(text)

	// Truncate if necessary (leave room for [CLS] and [SEP])
	maxTokens := t.maxLength - 2
	if len(tokens) > maxTokens {
		tokens = tokens[:maxTokens]
	}

	// Build sequence: [CLS] + tokens + [SEP]
	inputIDs := make([]int64, t.maxLength)
	attentionMask := make([]int64, t.maxLength)
	tokenTypeIDs := make([]int64, t.maxLength)

	inputIDs[0] = int64(t.clsTokenID)
	attentionMask[0] = 1

	for i, token := range tokens {
		id, ok := t.vocab[token]
		if !ok {
			id = t.unkTokenID
		}

		inputIDs[i+1] = int64(id)
		attentionMask[i+1] = 1
	}

	// Add [SEP]
	sepPos := len(tokens) + 1
	inputIDs[sepPos] = int64(t.sepTokenID)
	attentionMask[sepPos] = 1

	// Remaining positions are [PAD] (already 0 in attentionMask)
	for i := sepPos + 1; i < t.maxLength; i++ {
		inputIDs[i] = int64(t.padTokenID)
	}

	return &TokenizerOutput{
		InputIDs:      inputIDs,
		AttentionMask: attentionMask,
		TokenTypeIDs:  tokenTypeIDs,
	}
}

// EncodeBatch tokenizes multiple texts.
func (t *Tokenizer) EncodeBatch(texts []string) []*TokenizerOutput {
	outputs := make([]*TokenizerOutput, len(texts))
	for i, text := range texts {
		outputs[i] = t.Encode(text)
	}

	return outputs
}

// tokenize performs WordPiece tokenization on preprocessed text.
func (t *Tokenizer) tokenize(text string) []string {
	// Split into words (basic whitespace + punctuation tokenization)
	words := t.basicTokenize(text)

	// Apply WordPiece to each word
	var tokens []string

	for _, word := range words {
		wordTokens := t.wordPieceTokenize(word)
		tokens = append(tokens, wordTokens...)
	}

	return tokens
}

// basicTokenize splits text into words and punctuation.
func (t *Tokenizer) basicTokenize(text string) []string {
	// Normalize whitespace
	text = strings.TrimSpace(text)

	var tokens []string
	var current strings.Builder

	for _, r := range text {
		if unicode.IsSpace(r) {
			if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
		} else if unicode.IsPunct(r) || isChinesePunct(r) {
			// Punctuation becomes its own token
			if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}

			tokens = append(tokens, string(r))
		} else {
			current.WriteRune(r)
		}
	}

	if current.Len() > 0 {
		tokens = append(tokens, current.String())
	}

	return tokens
}

// wordPieceTokenize applies WordPiece algorithm to a single word.
func (t *Tokenizer) wordPieceTokenize(word string) []string {
	if len(word) == 0 {
		return nil
	}

	// Check if whole word is in vocab
	if _, ok := t.vocab[word]; ok {
		return []string{word}
	}

	var tokens []string
	start := 0
	wordRunes := []rune(word)

	for start < len(wordRunes) {
		end := len(wordRunes)
		found := false

		for end > start {
			substr := string(wordRunes[start:end])
			if start > 0 {
				substr = "##" + substr
			}

			if _, ok := t.vocab[substr]; ok {
				tokens = append(tokens, substr)
				found = true

				break
			}

			end--
		}

		if !found {
			// Character not in vocab, use [UNK]
			tokens = append(tokens, "[UNK]")
			start++
		} else {
			start = end
		}
	}

	return tokens
}

// isChinesePunct checks for Chinese punctuation characters.
func isChinesePunct(r rune) bool {
	// Chinese punctuation range
	return (r >= 0x3000 && r <= 0x303F) ||
		(r >= 0xFF00 && r <= 0xFFEF)
}

// Decode converts token IDs back to text.
func (t *Tokenizer) Decode(ids []int64) string {
	var tokens []string

	for _, id := range ids {
		if int(id) == t.clsTokenID || int(id) == t.sepTokenID || int(id) == t.padTokenID {
			continue
		}

		if token, ok := t.idToToken[int(id)]; ok {
			tokens = append(tokens, token)
		}
	}

	// Join tokens, removing ## prefixes
	var result strings.Builder

	for i, token := range tokens {
		if strings.HasPrefix(token, "##") {
			result.WriteString(token[2:])
		} else {
			if i > 0 {
				result.WriteString(" ")
			}

			result.WriteString(token)
		}
	}

	return result.String()
}

// VocabSize returns the vocabulary size.
func (t *Tokenizer) VocabSize() int {
	return t.vocabSize
}

// MaxLength returns the maximum sequence length.
func (t *Tokenizer) MaxLength() int {
	return t.maxLength
}
