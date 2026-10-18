package emotion

import (
	"errors"
	"math"
	"testing"
	"time"

	"github.com/kizuna-org/akari/internal/meaning"
)

const testSpace = "test"

func TestRespondNormalizesRestoredDuplicates(t *testing.T) {
	t.Parallel()

	point := meaning.Point{Text: "recover", Space: testSpace, Vector: []float64{1}}
	state := State{
		Feelings:  []Feeling{{Meaning: point, Strength: 0.8}, {Meaning: point, Strength: 0.4}},
		Readiness: []Feeling{{Meaning: point, Strength: 0.8}, {Meaning: point, Strength: 0.4}},
		Mood:      Mood{}, At: time.Unix(100, 0),
	}
	appraisal := Appraisal{
		Feelings: []Feeling{{Meaning: point, Strength: 0.2}},
		Others:   nil, Readiness: []Feeling{{Meaning: point, Strength: 0.2}}, MoodDelta: Mood{},
	}

	result, err := Respond(state, appraisal, 1, 0)
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Feelings) != 1 || result.Feelings[0].Strength != 1 ||
		len(result.Readiness) != 1 || result.Readiness[0].Strength != 1 {
		t.Fatalf("normalized response = %+v", result)
	}
}

func TestMixedEmotionsAndContinuity(t *testing.T) {
	t.Parallel()

	for _, negative := range []bool{false, true} {
		t.Run(map[bool]string{false: "pleasant", true: "unpleasant"}[negative], func(t *testing.T) {
			t.Parallel()

			instant := time.Unix(100, 0)
			feeling := Feeling{Meaning: meaning.Point{
				Text: "unnamed feeling", Space: testSpace, Vector: []float64{1},
			}, Strength: 1}
			appraisal := Appraisal{Feelings: []Feeling{feeling}, Others: []Feeling{feeling},
				Readiness: []Feeling{feeling}, MoodDelta: Mood{1, 1, 1}}

			if negative {
				appraisal.MoodDelta = Mood{-1, -1, -1}
			}

			initial := State{Feelings: nil, Readiness: nil, Mood: Mood{}, At: instant}

			state, err := Respond(initial, appraisal, 1, 0.5)
			if err != nil || len(state.Feelings) != 1 || state.Feelings[0].Strength != 1 {
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

func TestSustainedEmotionPreparationAndExpression(t *testing.T) {
	t.Parallel()

	point := meaning.Point{Text: "seek conversation", Space: testSpace, Vector: []float64{1}}
	secondPoint := meaning.Point{Text: "seek conversation again", Space: testSpace, Vector: []float64{1}}
	other := meaning.Point{Text: "withdraw", Space: testSpace, Vector: []float64{-1}}

	tests := []struct {
		name           string
		firstStrength  float64
		secondStrength float64
		expression     float64
		cue            meaning.Point
		wantPrep       float64
		wantExpressed  float64
	}{
		{name: "aligned direct preparation", firstStrength: 0.8, secondStrength: 0.4,
			expression: 0.25, cue: point, wantPrep: 1.2, wantExpressed: 0.3},
		{name: "opposed preparation", firstStrength: 0.8, secondStrength: 0.4,
			expression: 1, cue: other, wantPrep: -1.2, wantExpressed: 1.2},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			state := State{Feelings: []Feeling{{Meaning: point, Strength: test.firstStrength}},
				Readiness: []Feeling{{Meaning: point, Strength: test.firstStrength}}, Mood: Mood{0.2, 0, 0}, At: time.Unix(100, 0)}
			appraisal := Appraisal{Feelings: []Feeling{{Meaning: secondPoint, Strength: test.secondStrength}},
				Others: nil, Readiness: []Feeling{{Meaning: secondPoint, Strength: test.secondStrength}}, MoodDelta: Mood{}}

			next, err := Respond(state, appraisal, 1, 0)
			if err != nil {
				t.Fatalf("response error = %v", err)
			}

			if len(next.Feelings) != 2 || next.Feelings[0].Strength != test.firstStrength {
				t.Fatalf("sustained response = %+v, %v", next, err)
			}

			preparation, err := Preparation(next.Readiness, test.cue)
			if err != nil || math.Abs(preparation-test.wantPrep) > 1e-9 {
				t.Fatalf("preparation = %v, %v; want %v", preparation, err, test.wantPrep)
			}

			visible := Express(next.Feelings, test.expression)
			if math.Abs(visible[0].Strength+visible[1].Strength-test.wantExpressed) > 1e-9 ||
				next.Feelings[0].Strength != test.firstStrength {
				t.Fatalf("expression changed feeling or strength = %+v", visible)
			}
		})
	}
}

func TestEmpathyAndMoodAccumulation(t *testing.T) {
	t.Parallel()

	point := meaning.Point{Text: "another person", Space: testSpace, Vector: []float64{1}}

	for _, test := range []struct {
		name       string
		reactivity float64
		empathy    float64
		want       float64
	}{
		{name: "not affected", reactivity: 1, empathy: 0, want: 0},
		{name: "partial empathy", reactivity: 1, empathy: 0.4, want: 0.4},
		{name: "reactivity also scales empathy", reactivity: 0.5, empathy: 0.4, want: 0.2},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			initial := State{Feelings: nil, Readiness: nil, Mood: Mood{0.1, 0, 0}, At: time.Unix(100, 0)}
			appraisal := Appraisal{Feelings: nil, Others: []Feeling{{Meaning: point, Strength: 1}},
				Readiness: nil, MoodDelta: Mood{0.4, 0.2, -0.2}}

			next, err := Respond(initial, appraisal, test.reactivity, test.empathy)
			if err != nil {
				t.Fatalf("response error = %v", err)
			}

			if len(next.Feelings) != 1 || math.Abs(next.Feelings[0].Strength-test.want) > 1e-9 ||
				math.Abs(next.Mood[0]-(0.1+0.4*test.reactivity)) > 1e-9 ||
				math.Abs(next.Mood[1]-0.2*test.reactivity) > 1e-9 ||
				math.Abs(next.Mood[2]+0.2*test.reactivity) > 1e-9 {
				t.Fatalf("empathy or mood response = %+v, %v", next, err)
			}
		})
	}
}

func TestPreparationRejectsIncompatibleMeaningSpaces(t *testing.T) {
	t.Parallel()

	readiness := []Feeling{{Meaning: meaning.Point{
		Text: "prepare", Space: testSpace, Vector: []float64{1},
	}, Strength: 0.5}}
	cue := meaning.Point{Text: "other space", Space: "other", Vector: []float64{1}}

	_, err := Preparation(readiness, cue)

	if !errors.Is(err, meaning.ErrPoint) {
		t.Fatalf("incompatible preparation cue error = %v", err)
	}
}

func checkDecay(t *testing.T, initial, later State, err error) {
	t.Helper()

	if err != nil || later.Feelings[0].Strength != 0.5 || math.Abs(later.Mood[0]) <= 0.5 ||
		initial.Feelings[0].Meaning.Vector[0] != 1 || later.Readiness[0].Strength != 0.5 {
		t.Fatalf("continuity = %+v, %v", later, err)
	}
}
