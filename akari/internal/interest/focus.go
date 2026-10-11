// Package interest represents affinity and avoidance in semantic space.
package interest

import (
	"math"

	"github.com/kizuna-org/akari/internal/meaning"
)

type fault string

func (err fault) Error() string { return string(err) }

const ErrFocus fault = "focus must identify a valid semantic target with signed affinity"

type Focus struct {
	ID       string
	Target   meaning.Point
	Affinity float64
}

func Clone(foci []Focus) []Focus {
	result := make([]Focus, len(foci))

	for index, focus := range foci {
		result[index] = Focus{ID: focus.ID, Target: focus.Target.Clone(), Affinity: focus.Affinity}
	}

	return result
}

func Validate(foci []Focus) error {
	identities := make(map[string]bool, len(foci))

	for _, focus := range foci {
		if focus.ID == "" || identities[focus.ID] || !focus.Target.Valid() ||
			!meaning.Finite(focus.Affinity) || math.Abs(focus.Affinity) > 1 {
			return ErrFocus
		}

		identities[focus.ID] = true
	}

	return nil
}

// Salience is one input to selection, never a replacement for goals, duties or safety.
func Salience(foci []Focus, cue meaning.Point, pleasure, fatigue float64) (float64, error) {
	value := 0.0

	for _, focus := range foci {
		similarity, err := meaning.Similarity(focus.Target, cue)
		if err != nil {
			return 0, err
		}

		value += math.Max(0, similarity) * focus.Affinity
	}

	return value * math.Max(0, 1+pleasure) / (1 + fatigue), nil
}
