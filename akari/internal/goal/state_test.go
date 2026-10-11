package goal

import (
	"errors"
	"testing"

	"github.com/kizuna-org/akari/internal/meaning"
)

const workID = "work"

func TestWishesAndIntentions(t *testing.T) {
	t.Parallel()

	for _, reason := range []string{
		"consistent", "conflicting", "capacity", "identity", "meaning", "step", "position", "claim", "desire",
	} {
		t.Run(reason, func(t *testing.T) {
			t.Parallel()

			first := intended()
			change := Change{Desires: map[string]meaning.Point{"rest": point("rest"), workID: point(workID)},
				Adopt: map[string]Intention{workID: first}, End: nil}
			capacity := 2

			want := configureCase(reason, &change, &capacity)

			state, err := Apply(State{Desires: nil, Intentions: nil}, change.Clone(), capacity)
			if !errors.Is(err, want) {
				t.Fatalf("adoption = %v; want %v", err, want)
			}

			if want != nil {
				return
			}

			first.Claims["shared target"] = "changed"

			state, err = Apply(state, Change{Desires: nil, Adopt: nil, End: nil}, capacity)
			checkPersistent(t, state, err)

			state, err = Apply(state, Change{Desires: nil, Adopt: nil, End: []string{workID}}, capacity)
			if err != nil || len(state.Intentions) != 0 || len(state.Desires) != 2 {
				t.Fatal("explicit completion did not release intention")
			}
		})
	}
}

func intended() Intention {
	return Intention{Meaning: point("finish work"), Claims: map[string]string{"shared target": "finish"},
		Steps: []meaning.Point{point("first"), point("next")}, Next: 1, Paused: true}
}

func point(text string) meaning.Point {
	return meaning.Point{Text: text, Space: "test", Vector: []float64{1}}
}

func configureCase(reason string, change *Change, capacity *int) error {
	first := change.Adopt[workID]

	var want error

	switch reason {
	case "conflicting":
		other := intended()
		other.Claims["shared target"] = "rest"
		change.Adopt["rest"] = other
		want = ErrConflict
	case "capacity":
		*capacity = 0
		want = ErrCapacity
	case "identity":
		change.Adopt[""] = first
		want = ErrGoal
	case "meaning":
		first.Meaning.Vector = nil
		change.Adopt[workID] = first
		want = ErrGoal
	case "step":
		first.Steps[0].Vector = nil
		want = ErrGoal
	case "position":
		first.Next = -1
		change.Adopt[workID] = first
		want = ErrGoal
	case "claim":
		first.Claims[""] = "invalid"
		want = ErrGoal
	case "desire":
		change.Desires[""] = point("invalid")
		want = ErrGoal
	}

	return want
}

func checkPersistent(t *testing.T, state State, err error) {
	t.Helper()

	if err != nil || !state.Intentions[workID].Paused || state.Intentions[workID].Next != 1 ||
		state.Intentions[workID].Claims["shared target"] != "finish" || len(state.Desires) != 2 {
		t.Fatal("attention change reset adopted intention or contradictory wishes")
	}
}
