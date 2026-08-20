package emotion

import (
	"math"
	"time"
)

// Tuning defaults for a persona that feels roughly like a person. Every one of
// these is a per-persona knob, not a fixed law: docs/01-vision.md principle 1
// keeps "how human" adjustable, so a flat persona and a volatile one are both
// valid configurations.
const (
	// DefaultVolatility means events land at face value.
	DefaultVolatility = 1.0
	// DefaultExpressivity means what is felt is what is shown.
	DefaultExpressivity = 1.0
	// DefaultMoodInertia means mood fades at the reference rate.
	DefaultMoodInertia = 1.0
	// DefaultEmpathy means another's feeling lands at half strength.
	DefaultEmpathy = 0.5
)

const (
	// emotionHalfLife is how long a fresh emotion takes to fade by half.
	// Emotions are the fast layer: seconds to minutes.
	emotionHalfLife = 3 * time.Minute
	// moodHalfLife is how long mood takes to drift halfway back to baseline.
	// Mood is the slow layer: hours to a day.
	moodHalfLife = 4 * time.Hour

	// reflexThreshold is how strong a reflexive emotion must be before it
	// produces a tendency on its own.
	reflexThreshold = 0.6

	// moodContribution is how much of a single emotion settles into mood.
	// Small, so mood is the residue of many moments rather than the last one.
	moodContribution = 0.2
	// moodInterestPull is how far mood can tilt interest either way.
	moodInterestPull = 0.4
	// boredomNoveltyPull is how much boredom sharpens the appetite for novelty.
	boredomNoveltyPull = 0.5
	// anxietyCautionPull is how much anxiety sharpens attention to risk.
	anxietyCautionPull = 0.5

	halving      = 0.5
	maxIntensity = 1.0
	minMood      = -1.0
	maxMood      = 1.0
	neutralBias  = 1.0
)

// Traits are the fixed personality of a persona: not what it feels now, but how
// readily it feels at all. Traits are set once and do not drift, because the
// design fixes personality and leaves only state in motion
// (docs/01-vision.md principle 7).
type Traits struct {
	// Volatility scales how far a single event moves an emotion.
	Volatility float64
	// Expressivity scales how much of a felt emotion reaches the outside. A
	// reserved persona feels the same as an open one and shows less.
	Expressivity float64
	// MoodInertia scales how long mood holds its charge; higher lingers longer.
	MoodInertia float64
	// Empathy scales how strongly someone else's feeling becomes this
	// persona's own.
	Empathy float64
}

// DefaultTraits returns traits for a persona that feels roughly like a person.
func DefaultTraits() Traits {
	return Traits{
		Volatility:   DefaultVolatility,
		Expressivity: DefaultExpressivity,
		MoodInertia:  DefaultMoodInertia,
		Empathy:      DefaultEmpathy,
	}
}

// State is what one persona feels at a point in time: the fast emotions and the
// slow mood underneath them.
//
// State is not safe for concurrent use. The design has many parallel channels
// reading one shared inner state, so ownership of a State belongs to whatever
// serialises access to it.
type State struct {
	traits      Traits
	intensities [kindCount]float64
	mood        float64
}

// New returns a calm state for a persona with the given traits.
func New(traits Traits) *State {
	return &State{
		traits:      traits,
		intensities: [kindCount]float64{},
		mood:        0,
	}
}

// Feel applies an event's emotional impact, scaled by the persona's volatility,
// so the same event moves an excitable persona further than a placid one.
// Intensity is expected in (0, 1]; anything at or below zero is ignored.
func (s *State) Feel(kind Kind, intensity float64) {
	if !kind.valid() || intensity <= 0 {
		return
	}

	scaled := intensity * s.traits.Volatility
	s.intensities[kind] = clampUnit(s.intensities[kind] + scaled)
	s.mood = clampSigned(s.mood + scaled*kind.Valence()*moodContribution)
}

// Empathize lets someone else's visible feeling become this persona's own,
// scaled by its empathy (docs/02-emotion.md 2.5).
func (s *State) Empathize(kind Kind, intensity float64) {
	s.Feel(kind, intensity*s.traits.Empathy)
}

// Decay fades emotions toward calm and lets mood drift back to baseline. Left
// alone, a persona returns to itself.
func (s *State) Decay(elapsed time.Duration) {
	if elapsed <= 0 {
		return
	}

	kept := decayFactor(elapsed, emotionHalfLife)
	for kind := range s.intensities {
		s.intensities[kind] *= kept
	}

	s.mood *= decayFactor(elapsed, s.moodHalfLife())
}

// Intensity reports how strongly one emotion is felt right now, in [0, 1].
func (s *State) Intensity(kind Kind) float64 {
	if !kind.valid() {
		return 0
	}

	return s.intensities[kind]
}

// Expressed reports how much of a felt emotion actually shows outward.
func (s *State) Expressed(kind Kind) float64 {
	if !kind.valid() {
		return 0
	}

	return clampUnit(s.intensities[kind] * s.traits.Expressivity)
}

// Mood reports the slow background feeling, from -1 (low) to +1 (good).
func (s *State) Mood() float64 {
	return s.mood
}

// Dominant reports the strongest emotion felt right now and its intensity. An
// intensity of zero means the persona is calm and the kind is not meaningful.
func (s *State) Dominant() (Kind, float64) {
	strongest := Joy
	peak := 0.0

	for kind, intensity := range s.intensities {
		if intensity > peak {
			strongest = Kind(kind)
			peak = intensity
		}
	}

	return strongest, peak
}

// Reflex reports the readiness to act that the current feeling produces on its
// own, bypassing interest entirely.
//
// This is the fast direct path (docs/02-emotion.md 2.1): a jolt stops you before
// you have finished working out whether the thing was interesting. It reports
// false when nothing is strong enough to act by itself, which is most of the
// time; everyday behaviour goes the slow way, through interest.
func (s *State) Reflex() (Tendency, bool) {
	tendency := TendencyNone
	peak := reflexThreshold

	for kind, intensity := range s.intensities {
		if intensity <= peak || !Kind(kind).IsReflexive() {
			continue
		}

		tendency = Kind(kind).Tendency()
		peak = intensity
	}

	return tendency, tendency != TendencyNone
}

// InterestBias reports how the current mood tilts interest overall: above 1
// means things feel more worth caring about than usual, below 1 less.
//
// This is the slow indirect path, and it is where most of emotion's influence on
// behaviour lives. Emotion does not pick actions here; it changes how much
// things matter, and interest does the picking (docs/04-interest.md 4.4).
func (s *State) InterestBias() float64 {
	return neutralBias + s.mood*moodInterestPull
}

// NoveltyBias reports the extra pull unfamiliar things have right now. Boredom
// sharpens the appetite for anything new.
func (s *State) NoveltyBias() float64 {
	return neutralBias + s.intensities[Boredom]*boredomNoveltyPull
}

// CautionBias reports the extra pull that risks and things worth checking have
// right now. Anxiety makes a persona look harder before it moves.
func (s *State) CautionBias() float64 {
	return neutralBias + s.intensities[Anxiety]*anxietyCautionPull
}

// moodHalfLife returns the persona's own mood half-life, stretched by inertia.
func (s *State) moodHalfLife() time.Duration {
	if s.traits.MoodInertia <= 0 {
		return moodHalfLife
	}

	return time.Duration(float64(moodHalfLife) * s.traits.MoodInertia)
}

// decayFactor returns the fraction of a value that survives elapsed time, given
// a half-life.
func decayFactor(elapsed, halfLife time.Duration) float64 {
	if halfLife <= 0 {
		return 0
	}

	return math.Pow(halving, float64(elapsed)/float64(halfLife))
}

// clampUnit confines value to [0, 1], the range intensities live in.
func clampUnit(value float64) float64 {
	return math.Max(0, math.Min(maxIntensity, value))
}

// clampSigned confines value to [-1, 1], the range mood lives in.
func clampSigned(value float64) float64 {
	return math.Max(minMood, math.Min(maxMood, value))
}
