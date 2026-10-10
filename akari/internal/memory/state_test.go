package memory

import (
	"errors"
	"math"
	"testing"
	"time"

	"github.com/kizuna-org/akari/internal/meaning"
)

func TestRecallScoresAndOwnership(t *testing.T) {
	t.Parallel()

	for index, name := range []string{"meaning", "popularity", "recency", "emotion", "will"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			first := sample("a")
			second := sample("b")
			first.Emotion, first.Will, first.Accesses = 1, 1, 10
			second.Meaning.Vector = []float64{-1}
			second.OccurredAt = second.OccurredAt.Add(-time.Hour)
			state := State{Context: nil, Working: nil, Day: []Fragment{second}, Sleeping: []Fragment{first}, Agreements: nil}
			weights := [5]float64{}
			weights[index] = 1
			query := Query{Cue: first.Meaning.Clone(), At: first.OccurredAt, Freshness: time.Hour, Limit: 2}

			recalled, err := Recall(state, query, weights)
			if err != nil || recalled[0].ID != first.ID || len(recalled) != 2 {
				t.Fatalf("recall = %+v, %v", recalled, err)
			}

			recalled[0].Meaning.Vector[0] = 0
			recalled[0].Sources[0] = "changed"
			change := Change{Context: []Fragment{first}, Working: []Fragment{first}, Remember: nil,
				Recalled: []string{first.ID}, Agreements: []Agreement{{Conversation: "private", Recipients: []string{"alice"}}}}

			updated, err := Apply(state, change.Clone())
			if err != nil || updated.Sleeping[0].Accesses != 11 || state.Sleeping[0].Accesses != 10 ||
				state.Sleeping[0].Meaning.Vector[0] != 1 {
				t.Fatalf("recall mutated archive or adoption failed: %v", err)
			}

			checkSharing(t, updated, first)

			clearing := Change{Context: []Fragment{}, Working: []Fragment{}, Remember: nil, Recalled: nil, Agreements: nil}

			cleared, err := Apply(updated, clearing.Clone())
			checkCleared(t, cleared, err)
		})
	}
}

func TestInvalidMemoryChanges(t *testing.T) {
	t.Parallel()

	for _, reason := range []string{"duplicate", "invalid", "unknown access", "invalid agreement", "saturated access"} {
		t.Run(reason, func(t *testing.T) {
			t.Parallel()

			first := sample("a")
			state := State{Context: nil, Working: nil, Day: []Fragment{first}, Sleeping: nil, Agreements: nil}
			change := new(Change)
			want := error(ErrMemory)

			switch reason {
			case "duplicate":
				change.Remember = []Fragment{first}
			case "invalid":
				first.ID = ""
				change.Working = []Fragment{first}
			case "unknown access":
				change.Recalled = []string{"missing"}
			case "invalid agreement":
				change.Agreements = []Agreement{{Conversation: "", Recipients: nil}}
			case "saturated access":
				state.Day[0].Accesses = math.MaxUint64
				change.Recalled = []string{first.ID}
				want = nil
			}

			_, err := Apply(state, *change)
			if !errors.Is(err, want) {
				t.Fatal(err)
			}
		})
	}
}

func TestInvalidRecall(t *testing.T) {
	t.Parallel()

	for _, reason := range []string{"limit", "freshness", "cue", "time", "incompatible"} {
		t.Run(reason, func(t *testing.T) {
			t.Parallel()

			first := sample("a")
			query := Query{Cue: first.Meaning.Clone(), At: first.OccurredAt, Freshness: time.Hour, Limit: 1}
			want := error(ErrMemory)

			switch reason {
			case "limit":
				query.Limit = 0
			case "freshness":
				query.Freshness = 0
			case "cue":
				query.Cue.Vector = nil
			case "time":
				query.At = time.Time{}
			case "incompatible":
				query.Cue.Space = "different encoder"
				want = meaning.ErrPoint
			}

			_, err := Recall(State{Context: nil, Working: nil, Day: []Fragment{first}, Sleeping: nil, Agreements: nil},
				query, [5]float64{1, 0, 0, 0, 0})
			if !errors.Is(err, want) {
				t.Fatal(err)
			}
		})
	}
}

func sample(identity string) Fragment {
	return Fragment{ID: identity, Meaning: meaning.Point{Text: "recollection", Space: "test", Vector: []float64{1}},
		OccurredAt: time.Unix(100, 0), Emotion: 0, Will: 0, Strength: 1, Accesses: 0, Sources: []string{"private"}}
}

func checkSharing(t *testing.T, state State, fragment Fragment) {
	t.Helper()

	if !Allows(state.Agreements, fragment.Sources, "alice") || Allows(state.Agreements, fragment.Sources, "bob") ||
		!Allows(state.Agreements, nil, "bob") {
		t.Fatal("confidentiality did not apply to outward use only")
	}
}
func checkCleared(t *testing.T, state State, err error) {
	t.Helper()

	if err != nil || len(state.Context) != 0 || len(state.Working) != 0 || len(state.Sleeping) != 1 {
		t.Fatal("working memory was confused with the long-term archive")
	}
}
