// Package emotion holds mixed feelings, action readiness and the three mood axes.
package emotion

import (
	"math"
	"slices"
	"time"

	"github.com/kizuna-org/akari/internal/meaning"
)

type fault string

func (err fault) Error() string { return string(err) }

const ErrAppraisal fault = "invalid emotional appraisal or time dynamics"

const minimumRetainedStrength = 1e-6

// Mood is pleasure, arousal and approach; fatigue is deliberately separate.
type Mood [3]float64

type Feeling struct {
	Meaning  meaning.Point `json:"meaning"`
	Strength float64       `json:"strength"`
}

// Appraisal includes readiness interpreted by the subject, not an unconditional external action.
type Appraisal struct {
	Feelings  []Feeling
	Others    []Feeling
	Readiness []Feeling
	MoodDelta Mood
}

type State struct {
	Feelings  []Feeling `json:"feelings"`
	Readiness []Feeling `json:"readiness"`
	Mood      Mood      `json:"mood"`
	At        time.Time `json:"at"`
}

type Dynamics struct {
	EmotionHalfLife time.Duration
	MoodHalfLife    time.Duration
}

func CloneFeelings(feelings []Feeling) []Feeling {
	cloned := make([]Feeling, len(feelings))

	for index, feeling := range feelings {
		cloned[index] = Feeling{Meaning: feeling.Meaning.Clone(), Strength: feeling.Strength}
	}

	return cloned
}

func (appraisal Appraisal) Clone() Appraisal {
	return Appraisal{
		Feelings: CloneFeelings(appraisal.Feelings), Others: CloneFeelings(appraisal.Others),
		Readiness: CloneFeelings(appraisal.Readiness), MoodDelta: appraisal.MoodDelta,
	}
}

func (appraisal Appraisal) Validate() error {
	for _, group := range [][]Feeling{appraisal.Feelings, appraisal.Others, appraisal.Readiness} {
		for _, feeling := range group {
			if !feeling.Meaning.Valid() || !meaning.Unit(feeling.Strength) {
				return ErrAppraisal
			}
		}
	}

	for _, value := range appraisal.MoodDelta {
		if !meaning.Finite(value) || math.Abs(value) > 1 {
			return ErrAppraisal
		}
	}

	return nil
}

func (state State) Clone() State {
	return State{
		Feelings: CloneFeelings(state.Feelings), Readiness: CloneFeelings(state.Readiness), Mood: state.Mood, At: state.At,
	}
}

// Advance supplies a reversible scaffold; durations are injected, not new persona axes.
func Advance(state State, instant time.Time, baseline Mood, lingering float64, dynamics Dynamics) (State, error) {
	if instant.Before(state.At) || dynamics.EmotionHalfLife <= 0 || dynamics.MoodHalfLife <= 0 {
		return State{}, ErrAppraisal
	}

	result := state.Clone()
	elapsed := float64(instant.Sub(state.At))
	feelingScale := math.Exp2(-elapsed / float64(dynamics.EmotionHalfLife))
	moodScale := math.Exp2(-elapsed / (float64(dynamics.MoodHalfLife) * (1 + lingering)))

	result.Feelings = decay(result.Feelings, feelingScale)
	result.Readiness = decay(result.Readiness, feelingScale)

	for index := range result.Mood {
		result.Mood[index] = baseline[index] + (result.Mood[index]-baseline[index])*moodScale
	}

	result.At = instant

	return result, nil
}

func decay(feelings []Feeling, scale float64) []Feeling {
	retained := feelings[:0]

	for _, feeling := range feelings {
		feeling.Strength *= scale
		if feeling.Strength > minimumRetainedStrength {
			retained = append(retained, feeling)
		}
	}

	return retained
}

func Respond(state State, appraisal Appraisal, reactivity, empathy float64) (State, error) {
	err := appraisal.Validate()
	if err != nil {
		return State{}, err
	}

	result := state.Clone()
	result.Feelings = merge(nil, result.Feelings)
	result.Readiness = merge(nil, result.Readiness)
	result.Feelings = merge(result.Feelings, scaled(appraisal.Feelings, reactivity))
	result.Feelings = merge(result.Feelings, scaled(appraisal.Others, reactivity*empathy))
	result.Readiness = merge(result.Readiness, scaled(appraisal.Readiness, reactivity))

	for index, delta := range appraisal.MoodDelta {
		result.Mood[index] = math.Max(-1, math.Min(1, result.Mood[index]+delta*reactivity))
	}

	return result, nil
}

// Preparation measures how strongly current action readiness matches a possible
// action cue. It is a deliberation input and never grants Gateway authority.
func Preparation(readiness []Feeling, cue meaning.Point) (float64, error) {
	value := 0.0

	for _, feeling := range readiness {
		similarity, err := meaning.Similarity(feeling.Meaning, cue)
		if err != nil {
			return 0, err
		}

		value += similarity * feeling.Strength
	}

	return value, nil
}

// Express returns an outward-facing copy scaled by the fixed personality axis.
// The internally felt state is never reduced by a low expression setting.
func Express(feelings []Feeling, degree float64) []Feeling {
	return scaled(feelings, degree)
}

func scaled(feelings []Feeling, scale float64) []Feeling {
	result := CloneFeelings(feelings)

	for index := range result {
		result[index].Strength *= scale
	}

	return result
}

func merge(current, incoming []Feeling) []Feeling {
	result := CloneFeelings(current)

	for _, next := range incoming {
		merged := false

		for index := range result {
			if sameMeaning(result[index].Meaning, next.Meaning) {
				result[index].Strength = math.Min(1, result[index].Strength+next.Strength)
				merged = true

				break
			}
		}

		if !merged {
			result = append(result, next)
		}
	}

	return result
}

func sameMeaning(left, right meaning.Point) bool {
	return left.Text == right.Text && left.Space == right.Space && slices.Equal(left.Vector, right.Vector)
}
