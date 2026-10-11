// Package memory defines Kiseki's owned fragments and cue-based recall contract.
package memory

import (
	"context"
	"math"
	"slices"
	"sort"
	"time"

	"github.com/kizuna-org/akari/internal/meaning"
)

type fault string

func (err fault) Error() string { return string(err) }

const ErrMemory fault = "invalid memory fragment, recall query, agreement or repeated experience"

type Fragment struct {
	ID         string        `json:"id"`
	Meaning    meaning.Point `json:"meaning"`
	OccurredAt time.Time     `json:"occurredAt"`
	Emotion    float64       `json:"emotion"`
	Will       float64       `json:"will"`
	Strength   float64       `json:"strength"`
	Accesses   uint64        `json:"accesses"`
	Sources    []string      `json:"sources"`
}

// Agreement attaches confidentiality to a conversation, not a keyword or feeling.
type Agreement struct {
	Conversation string   `json:"conversation"`
	Recipients   []string `json:"recipients"`
}

type State struct {
	Context    []Fragment  `json:"context"`
	Working    []Fragment  `json:"working"`
	Day        []Fragment  `json:"day"`
	Sleeping   []Fragment  `json:"sleeping"`
	Agreements []Agreement `json:"agreements"`
}

type Change struct {
	Context    []Fragment
	Working    []Fragment
	Remember   []Fragment
	Recalled   []string
	Agreements []Agreement
}

type Query struct {
	Cue       meaning.Point
	At        time.Time
	Freshness time.Duration
	Limit     int
}

// Reader is the capability supplied to a thinking Channel, never a full archive export.
type Reader interface {
	Recall(ctx context.Context, query Query) ([]Fragment, error)
}

// Consolidator supplies subject-side compact/oblivion/recollection; no fake forgetting is added.
type Consolidator interface {
	Consolidate(ctx context.Context, fragments []Fragment) ([]Fragment, error)
}

func CloneFragments(fragments []Fragment) []Fragment {
	result := make([]Fragment, len(fragments))

	for index, fragment := range fragments {
		fragment.Meaning = fragment.Meaning.Clone()
		fragment.Sources = slices.Clone(fragment.Sources)
		result[index] = fragment
	}

	return result
}

func CloneAgreements(agreements []Agreement) []Agreement {
	result := make([]Agreement, len(agreements))

	for index, agreement := range agreements {
		result[index] = Agreement{Conversation: agreement.Conversation, Recipients: slices.Clone(agreement.Recipients)}
	}

	return result
}

func (state State) Clone() State {
	return State{
		Context: CloneFragments(state.Context), Working: CloneFragments(state.Working),
		Day: CloneFragments(state.Day), Sleeping: CloneFragments(state.Sleeping),
		Agreements: CloneAgreements(state.Agreements),
	}
}

func (change Change) Clone() Change {
	cloned := Change{
		Context: nil, Working: nil, Remember: CloneFragments(change.Remember),
		Recalled: slices.Clone(change.Recalled), Agreements: CloneAgreements(change.Agreements),
	}

	if change.Context != nil {
		cloned.Context = CloneFragments(change.Context)
	}

	if change.Working != nil {
		cloned.Working = CloneFragments(change.Working)
	}

	return cloned
}

func Apply(state State, change Change) (State, error) {
	result := state.Clone()

	if change.Context != nil {
		result.Context = CloneFragments(change.Context)
	}

	if change.Working != nil {
		result.Working = CloneFragments(change.Working)
	}

	result.Day = append(result.Day, CloneFragments(change.Remember)...)
	result.Agreements = append(result.Agreements, CloneAgreements(change.Agreements)...)

	err := result.validate()
	if err != nil {
		return State{}, err
	}

	for _, identity := range change.Recalled {
		if !access(&result, identity) {
			return State{}, ErrMemory
		}
	}

	return result, nil
}

// Recall is read-only, including popularity; actual adoption records accesses separately.
func Recall(state State, query Query, weights [5]float64) ([]Fragment, error) {
	if query.Limit < 1 || query.Freshness <= 0 || !query.Cue.Valid() || query.At.IsZero() {
		return nil, ErrMemory
	}

	type candidate struct {
		fragment Fragment
		score    float64
	}

	candidates := make([]candidate, 0, len(state.Day)+len(state.Sleeping))

	for _, group := range [][]Fragment{state.Day, state.Sleeping} {
		for _, fragment := range group {
			score, err := relevance(fragment, query, weights)
			if err != nil {
				return nil, err
			}

			candidates = append(candidates, candidate{fragment: fragment, score: score})
		}
	}

	sort.Slice(candidates, func(left, right int) bool {
		if candidates[left].score == candidates[right].score {
			return candidates[left].fragment.ID < candidates[right].fragment.ID
		}

		return candidates[left].score > candidates[right].score
	})

	result := make([]Fragment, 0, min(query.Limit, len(candidates)))

	for _, candidate := range candidates[:min(query.Limit, len(candidates))] {
		result = append(result, candidate.fragment)
	}

	return CloneFragments(result), nil
}

// Allows checks known provenance; it does not claim to detect secret paraphrases.
func Allows(agreements []Agreement, sources []string, recipient string) bool {
	for _, agreement := range agreements {
		if slices.Contains(sources, agreement.Conversation) && !slices.Contains(agreement.Recipients, recipient) {
			return false
		}
	}

	return true
}

func (state State) validate() error {
	for _, group := range [][]Fragment{state.Context, state.Working, append(slices.Clone(state.Day), state.Sleeping...)} {
		err := validateFragments(group)
		if err != nil {
			return err
		}
	}

	for _, agreement := range state.Agreements {
		if agreement.Conversation == "" {
			return ErrMemory
		}
	}

	return nil
}

func validateFragments(fragments []Fragment) error {
	identities := make(map[string]bool, len(fragments))

	for _, fragment := range fragments {
		if fragment.ID == "" || identities[fragment.ID] || !fragment.Meaning.Valid() || fragment.OccurredAt.IsZero() ||
			!meaning.Unit(fragment.Emotion) || !meaning.Unit(fragment.Will) || !meaning.Unit(fragment.Strength) {
			return ErrMemory
		}

		identities[fragment.ID] = true
	}

	return nil
}

func access(state *State, identity string) bool {
	for _, group := range [][]Fragment{state.Day, state.Sleeping} {
		for index := range group {
			if group[index].ID == identity {
				if group[index].Accesses < math.MaxUint64 {
					group[index].Accesses++
				}

				return true
			}
		}
	}

	return false
}

func relevance(fragment Fragment, query Query, weights [5]float64) (float64, error) {
	similarity, err := meaning.Similarity(fragment.Meaning, query.Cue)
	if err != nil {
		return 0, err
	}

	age := math.Max(0, float64(query.At.Sub(fragment.OccurredAt)))
	popularity := float64(fragment.Accesses)
	scores := [5]float64{
		(similarity + 1) / 2, popularity / (1 + popularity),
		math.Exp(-age / float64(query.Freshness)), fragment.Emotion, fragment.Will,
	}
	total := 0.0

	for index, score := range scores {
		total += score * weights[index]
	}

	return total * fragment.Strength, nil
}
