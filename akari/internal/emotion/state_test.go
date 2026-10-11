package emotion

import (
	"errors"
	"math"
	"testing"
	"time"

	"github.com/kizuna-org/akari/internal/meaning"
)

func TestMixedEmotionsAndContinuity(t *testing.T) {
	t.Parallel()

	for _, negative := range []bool{false, true} {
		t.Run(map[bool]string{false: "pleasant", true: "unpleasant"}[negative], func(t *testing.T) {
			t.Parallel()

			instant := time.Unix(100, 0)
			feeling := Feeling{Meaning: meaning.Point{Text: "unnamed feeling", Space: "test", Vector: []float64{1}}, Strength: 1}
			appraisal := Appraisal{Feelings: []Feeling{feeling}, Others: []Feeling{feeling},
				Readiness: []Feeling{feeling}, MoodDelta: Mood{1, 1, 1}}

			if negative {
				appraisal.MoodDelta = Mood{-1, -1, -1}
			}

			initial := State{Feelings: nil, Readiness: nil, Mood: Mood{}, At: instant}

			state, err := Respond(initial, appraisal, 1, 0.5)
			if err != nil || len(state.Feelings) != 2 || state.Feelings[1].Strength != 0.5 {
				t.Fatalf("response = %+v, %v", state, err)
			}

			_, err = Respond(state, appraisal, 1, 0.5)
			if err != nil {
				t.Fatal(err)
			}

			appraisal.Feelings[0].Meaning.Vector[0] = 0
			clone := appraisal.Clone()
			clone.MoodDelta[0] = 0

			dynamics := Dynamics{EmotionHalfLife: time.Hour, MoodHalfLife: time.Hour}

			later, err := Advance(state, instant.Add(time.Hour), Mood{}, 1, dynamics)
			checkDecay(t, state, later, err)

			_, err = Advance(state, instant.Add(-time.Hour), Mood{}, 1, dynamics)
			if !errors.Is(err, ErrAppraisal) {
				t.Fatal("clock reversed")
			}

			dynamics.MoodHalfLife = 0

			_, err = Advance(state, instant, Mood{}, 1, dynamics)
			if !errors.Is(err, ErrAppraisal) {
				t.Fatal("invalid dynamics")
			}
		})
	}
}

func TestInvalidAppraisals(t *testing.T) {
	t.Parallel()

	for _, appraisal := range []Appraisal{
		{Feelings: []Feeling{{Meaning: meaning.Point{Text: "", Space: "", Vector: nil}, Strength: 1}},
			Others: nil, Readiness: nil, MoodDelta: Mood{}},
		{Feelings: nil, Others: nil, Readiness: nil, MoodDelta: Mood{math.NaN(), 0, 0}},
	} {
		t.Run("invalid", func(t *testing.T) {
			t.Parallel()

			_, err := Respond(State{Feelings: nil, Readiness: nil, Mood: Mood{}, At: time.Time{}}, appraisal, 1, 1)
			if !errors.Is(err, ErrAppraisal) {
				t.Fatal(err)
			}
		})
	}
}

func checkDecay(t *testing.T, initial, later State, err error) {
	t.Helper()

	if err != nil || later.Feelings[0].Strength != 0.5 || math.Abs(later.Mood[0]) <= 0.5 ||
		initial.Feelings[0].Meaning.Vector[0] != 1 || later.Readiness[0].Strength != 0.5 {
		t.Fatalf("continuity = %+v, %v", later, err)
	}
}
