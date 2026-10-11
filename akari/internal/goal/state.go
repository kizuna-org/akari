// Package goal distinguishes conflicting wishes from consistent adopted intentions.
package goal

import (
	"maps"
	"slices"

	"github.com/kizuna-org/akari/internal/meaning"
)

type fault string

func (err fault) Error() string { return string(err) }

const (
	ErrGoal     fault = "invalid goal, intention or resume position"
	ErrConflict fault = "adopted intentions make incompatible claims"
	ErrCapacity fault = "persona intention capacity reached"
)

// Intention claims normalized shared targets, not complete semantic contradiction detection.
type Intention struct {
	Meaning meaning.Point
	Claims  map[string]string
	Steps   []meaning.Point
	Next    int
	Paused  bool
}

type State struct {
	Desires    map[string]meaning.Point
	Intentions map[string]Intention
}

type Change struct {
	Desires map[string]meaning.Point
	Adopt   map[string]Intention
	End     []string
}

func ClonePoints(points map[string]meaning.Point) map[string]meaning.Point {
	result := make(map[string]meaning.Point, len(points))

	for identity, point := range points {
		result[identity] = point.Clone()
	}

	return result
}

func CloneIntentions(intentions map[string]Intention) map[string]Intention {
	result := make(map[string]Intention, len(intentions))

	for identity, intention := range intentions {
		steps := make([]meaning.Point, len(intention.Steps))

		for index, step := range intention.Steps {
			steps[index] = step.Clone()
		}

		result[identity] = Intention{
			Meaning: intention.Meaning.Clone(), Claims: maps.Clone(intention.Claims),
			Steps: steps, Next: intention.Next, Paused: intention.Paused,
		}
	}

	return result
}

func (state State) Clone() State {
	return State{Desires: ClonePoints(state.Desires), Intentions: CloneIntentions(state.Intentions)}
}

func (change Change) Clone() Change {
	return Change{
		Desires: ClonePoints(change.Desires), Adopt: CloneIntentions(change.Adopt), End: slices.Clone(change.End),
	}
}

func Apply(state State, change Change, capacity int) (State, error) {
	result := state.Clone()

	for _, identity := range change.End {
		delete(result.Intentions, identity)
	}

	for identity, desire := range change.Desires {
		result.Desires[identity] = desire.Clone()
	}

	for identity, intention := range change.Adopt {
		result.Intentions[identity] = CloneIntentions(map[string]Intention{identity: intention})[identity]
	}

	err := result.validate(capacity)
	if err != nil {
		return State{}, err
	}

	return result, nil
}

func (state State) validate(capacity int) error {
	if len(state.Intentions) > capacity {
		return ErrCapacity
	}

	for identity, desire := range state.Desires {
		if identity == "" || !desire.Valid() {
			return ErrGoal
		}
	}

	claims := make(map[string]string)

	for identity, intention := range state.Intentions {
		if identity == "" {
			return ErrGoal
		}

		err := validateIntention(intention)
		if err != nil {
			return err
		}

		for target, value := range intention.Claims {
			err = claim(claims, target, value)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func claim(claims map[string]string, target, value string) error {
	if target == "" || value == "" {
		return ErrGoal
	}

	if previous, exists := claims[target]; exists && previous != value {
		return ErrConflict
	}

	claims[target] = value

	return nil
}

func validateIntention(intention Intention) error {
	if !intention.Meaning.Valid() || intention.Next < 0 || intention.Next > len(intention.Steps) {
		return ErrGoal
	}

	for _, step := range intention.Steps {
		if !step.Valid() {
			return ErrGoal
		}
	}

	return nil
}
