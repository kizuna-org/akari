package memory

import "strings"

// Similarity reports how closely a remembered fragment matches what is being
// asked about, in [0, 1].
//
// It is an interface because judging meaning is a technology question, not a
// design one: an embedding model, a keyword index or something else can all
// answer it. The memory rules themselves must not care which.
type Similarity interface {
	Score(cue, content string) float64
}

// TokenOverlap scores by how many words two texts share. It is the plain
// keyword half of the hybrid search the design calls for, and it is enough to
// exercise and reason about the memory rules without pulling in a model.
type TokenOverlap struct{}

// Score reports the fraction of the cue's words that appear in the content.
func (TokenOverlap) Score(cue, content string) float64 {
	wanted := tokenize(cue)
	if len(wanted) == 0 {
		return 0
	}

	have := tokenize(content)
	matched := 0

	for token := range wanted {
		if _, ok := have[token]; ok {
			matched++
		}
	}

	return float64(matched) / float64(len(wanted))
}

// separators are the characters that end a word, in the scripts this stand-in
// needs to cope with.
const separators = " \t\n\r,.!?;:\u3001\u3002"

// tokenize splits text into a set of lowercase words.
func tokenize(text string) map[string]struct{} {
	fields := strings.FieldsFunc(strings.ToLower(text), func(r rune) bool {
		return strings.ContainsRune(separators, r)
	})

	tokens := make(map[string]struct{}, len(fields))
	for _, field := range fields {
		tokens[field] = struct{}{}
	}

	return tokens
}
