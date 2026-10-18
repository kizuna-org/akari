package interest

import (
	"errors"
	"math"
	"testing"

	"github.com/kizuna-org/akari/internal/emotion"
	"github.com/kizuna-org/akari/internal/meaning"
)

const testSpace = "test"
const testTopic = "topic"

func TestAffinityAndAvoidance(t *testing.T) {
	t.Parallel()

	for _, affinity := range []float64{-1, 0, 1} {
		t.Run(map[float64]string{-1: "avoid", 0: "indifferent", 1: "approach"}[affinity], func(t *testing.T) {
			t.Parallel()

			cue := meaning.Point{Text: testTopic, Space: testSpace, Vector: []float64{1}}
			foci := []Focus{{ID: "topic", Target: cue, Affinity: affinity}}
			clone := Clone(foci)
			clone[0].Target.Vector[0] = 0

			full, err := Salience(foci, cue, 0, 0)
			if err != nil || full != affinity || Validate(foci) != nil {
				t.Fatalf("affinity = %v, %v", full, err)
			}

			tired, err := Salience(foci, cue, 0, 1)
			if err != nil || math.Abs(tired) > math.Abs(full) {
				t.Fatal("fatigue did not reduce general engagement")
			}

			if !errors.Is(Validate(clone), ErrFocus) || !errors.Is(Validate(append(foci, foci...)), ErrFocus) {
				t.Fatal("invalid focus accepted")
			}

			_, err = Salience(clone, cue, 0, 0)
			if !errors.Is(err, meaning.ErrPoint) {
				t.Fatal(err)
			}
		})
	}
}

func TestEmotionChangesRelatedInterestSalience(t *testing.T) {
	t.Parallel()

	near := meaning.Point{Text: "related topic", Space: testSpace, Vector: []float64{1, 0.1}}
	far := meaning.Point{Text: "unrelated topic", Space: testSpace, Vector: []float64{-1, 0}}
	topic := near.Clone()

	for _, test := range []struct {
		name     string
		feeling  emotion.Feeling
		mood     emotion.Mood
		wantNear float64
	}{
		{name: "related feeling and positive mood",
			feeling: emotion.Feeling{Meaning: topic, Strength: 0.5}, mood: emotion.Mood{0.5}, wantNear: 2.25},
		{name: "low mood dampens interest",
			feeling: emotion.Feeling{Meaning: topic, Strength: 0.5}, mood: emotion.Mood{-0.5}, wantNear: 0.75},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			foci := []Focus{{ID: "near", Target: near, Affinity: 1}, {ID: "far", Target: far, Affinity: 1}}
			got, err := SalienceWithEmotion(foci, near, test.mood, []emotion.Feeling{test.feeling}, 0)
			assertSalience(t, got, err, test.wantNear)
		})
	}
}

func assertSalience(t *testing.T, got float64, err error, want float64) {
	t.Helper()

	if err != nil {
		t.Fatalf("salience error = %v", err)
	}

	if math.Abs(got-want) > 1e-9 {
		t.Fatalf("salience = %v; want %v", got, want)
	}
}

func TestSalienceRejectsIncompatibleEmotionSpace(t *testing.T) {
	t.Parallel()

	topic := meaning.Point{Text: testTopic, Space: testSpace, Vector: []float64{1}}
	feeling := emotion.Feeling{Meaning: meaning.Point{
		Text: "feeling", Space: "other", Vector: []float64{1},
	}, Strength: 0.5}
	_, err := SalienceWithEmotion([]Focus{{ID: "topic", Target: topic, Affinity: 1}}, topic,
		emotion.Mood{}, []emotion.Feeling{feeling}, 0)

	if !errors.Is(err, meaning.ErrPoint) {
		t.Fatalf("incompatible emotional salience error = %v", err)
	}
}

func TestEmotionDoesNotBoostOrthogonalCue(t *testing.T) {
	t.Parallel()

	focus := meaning.Point{Text: "focus", Space: testSpace, Vector: []float64{1, 0}}
	cue := meaning.Point{Text: "orthogonal cue", Space: testSpace, Vector: []float64{0, 1}}
	feeling := emotion.Feeling{Meaning: focus, Strength: 0.5}

	got, err := SalienceWithEmotion([]Focus{{ID: "focus", Target: focus, Affinity: 1}}, cue,
		emotion.Mood{}, []emotion.Feeling{feeling}, 0)
	assertSalience(t, got, err, 0)
}
