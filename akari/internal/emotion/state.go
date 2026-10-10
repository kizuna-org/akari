// Package emotion holds mixed feelings, action readiness and the three mood axes.
package emotion

import (
	"math"
	"time"

	"github.com/kizuna-org/akari/internal/meaning"
)

type fault string

func (err fault) Error() string { return string(err) }

const ErrAppraisal fault = "invalid emotional appraisal or time dynamics"

// Mood is pleasure, arousal and approach; fatigue is deliberately separate.
type Mood [3]float64

type Feeling struct {
	Meaning  meaning.Point
	Strength float64
}

// Appraisal includes readiness interpreted by the subject, not an unconditional external action.
type Appraisal struct {
	Feelings  []Feeling
	Others    []Feeling
	Readiness []Feeling
	MoodDelta Mood
}

type State struct {
	Feelings  []Feeling
	Readiness []Feeling
	Mood      Mood
	At        time.Time
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

	for _, group := range [][]Feeling{result.Feelings, result.Readiness} {
		for index := range group {
			group[index].Strength *= feelingScale
		}
	}

	for index := range result.Mood {
		result.Mood[index] = baseline[index] + (result.Mood[index]-baseline[index])*moodScale
	}

	result.At = instant

	return result, nil
}

func Respond(state State, appraisal Appraisal, reactivity, empathy float64) (State, error) {
	err := appraisal.Validate()
	if err != nil {
		return State{}, err
	}

	result := state.Clone()
	result.Feelings = scaled(appraisal.Feelings, reactivity)
	result.Feelings = append(result.Feelings, scaled(appraisal.Others, reactivity*empathy)...)
	result.Readiness = scaled(appraisal.Readiness, reactivity)

	for index, delta := range appraisal.MoodDelta {
		result.Mood[index] = math.Max(-1, math.Min(1, result.Mood[index]+delta*reactivity))
	}

	return result, nil
}

func scaled(feelings []Feeling, scale float64) []Feeling {
	result := CloneFeelings(feelings)

	for index := range result {
		result[index].Strength *= scale
	}

	return result
}
