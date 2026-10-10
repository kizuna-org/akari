package interest

import (
	"errors"
	"math"
	"testing"

	"github.com/kizuna-org/akari/internal/meaning"
)

func TestAffinityAndAvoidance(t *testing.T) {
	t.Parallel()

	for _, affinity := range []float64{-1, 0, 1} {
		t.Run(map[float64]string{-1: "avoid", 0: "indifferent", 1: "approach"}[affinity], func(t *testing.T) {
			t.Parallel()

			cue := meaning.Point{Text: "topic", Space: "test", Vector: []float64{1}}
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
