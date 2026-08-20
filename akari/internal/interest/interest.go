package interest

import (
	"math"
	"time"
)

// Tuning defaults for a persona whose curiosity behaves roughly like a person's.
// Every value is a per-persona knob (docs/01-vision.md principle 1).
const (
	// DefaultCuriosity means novelty pulls about as hard as learned liking.
	DefaultCuriosity = 1.0
	// DefaultHabituation means a third of the novelty wears off per engagement.
	DefaultHabituation = 0.34
	// DefaultAffinityLearning means one good experience moves liking a little.
	DefaultAffinityLearning = 0.3
)

const (
	// affinityHalfLife is how long learned liking takes to halve while a topic
	// goes untouched. Slow: what you care about outlasts the day.
	affinityHalfLife = 14 * 24 * time.Hour
	// noveltyRecoveryHalfLife is how long it takes for half the lost freshness
	// of a topic to come back. Absence makes things interesting again.
	noveltyRecoveryHalfLife = 30 * 24 * time.Hour
	// progressHalfLife is how long the sense of getting somewhere survives.
	// Fast: momentum is a feeling about right now.
	progressHalfLife = 2 * time.Hour

	// affinityWeight is how much learned liking counts toward interest.
	affinityWeight = 0.5
	// noveltyWeight is how much unfamiliarity counts toward interest.
	noveltyWeight = 0.3
	// progressWeight is how much a sense of getting somewhere counts.
	//
	// This is the guard against being captured by noise: novelty alone cannot
	// hold attention, because something unpredictable that never yields any
	// understanding scores only its novelty, and that novelty decays with every
	// fruitless engagement (docs/04-interest.md 4.3).
	progressWeight = 0.2

	// noveltyRecoveryShare is how much of the lost freshness can ever come
	// back. Below one, because something once met is never wholly new again.
	noveltyRecoveryShare = 0.5

	halving      = 0.5
	minScore     = 0.0
	maxScore     = 1.0
	minAffinity  = -1.0
	freshNovelty = 1.0
	neutralBias  = 1.0
)

// Tuning is the fixed shape of a persona's curiosity.
type Tuning struct {
	// Curiosity scales how hard unfamiliarity pulls. A high-curiosity persona
	// chases new things; a low one sticks to what it already likes.
	Curiosity float64
	// Habituation is the fraction of remaining novelty that wears off each time
	// a topic is engaged with. Higher means bored sooner.
	Habituation float64
	// AffinityLearning scales how far one experience moves learned liking.
	AffinityLearning float64
}

// DefaultTuning returns curiosity settings that behave roughly like a person's.
func DefaultTuning() Tuning {
	return Tuning{
		Curiosity:        DefaultCuriosity,
		Habituation:      DefaultHabituation,
		AffinityLearning: DefaultAffinityLearning,
	}
}

// Bias is how the current feeling tilts interest. It comes from the emotion
// package but is passed in as plain numbers, so interest never depends on how
// emotion is implemented.
//
// This is the indirect path of docs/02-emotion.md: emotion does not choose
// actions, it changes how much things seem to matter.
type Bias struct {
	// Overall scales every topic at once; above 1 for a good mood, below for low.
	Overall float64
	// Novelty scales the pull of unfamiliar things; boredom raises it.
	Novelty float64
}

// NeutralBias returns the bias of a persona feeling nothing in particular.
func NeutralBias() Bias {
	return Bias{Overall: neutralBias, Novelty: neutralBias}
}

// Result is what came of engaging with a topic.
type Result struct {
	// Enjoyment is how the experience felt, from -1 (unpleasant) to +1
	// (pleasant). It moves learned liking up or down.
	Enjoyment float64
	// Progress is how much the engagement actually got somewhere, in [0, 1].
	// Zero means nothing was gained, which is what lets a persona give up on
	// something it finds gripping but fruitless.
	Progress float64
}

// topic is what is known about one thing a persona might care about.
type topic struct {
	// affinity is learned liking, in [-1, 1]. It goes negative because a bad
	// experience does not merely fail to attract: it puts a persona off
	// (docs/04-interest.md 4.4).
	affinity    float64
	novelty     float64
	progress    float64
	lastTouched time.Time
}

// newTopic returns a topic never encountered before: wholly unfamiliar, with no
// liking or momentum yet.
func newTopic(now time.Time) topic {
	return topic{
		affinity:    0,
		novelty:     freshNovelty,
		progress:    0,
		lastTouched: now,
	}
}

// at returns the topic as it stands after time has passed, without recording
// anything. Fading is a pure function of elapsed time, so reading interest never
// changes it.
func (t topic) at(now time.Time) topic {
	elapsed := now.Sub(t.lastTouched)
	if elapsed <= 0 {
		return t
	}

	recovered := (freshNovelty - decayFactor(elapsed, noveltyRecoveryHalfLife)) * noveltyRecoveryShare

	return topic{
		affinity:    t.affinity * decayFactor(elapsed, affinityHalfLife),
		novelty:     clampUnit(t.novelty + (freshNovelty-t.novelty)*recovered),
		progress:    t.progress * decayFactor(elapsed, progressHalfLife),
		lastTouched: now,
	}
}

// Table holds what a persona cares about, keyed by an opaque topic identifier.
//
// Table is not safe for concurrent use; whatever serialises access to the
// persona's inner state owns it.
type Table struct {
	tuning Tuning
	topics map[string]topic
}

// New returns an empty table for a persona with the given curiosity.
func New(tuning Tuning) *Table {
	return &Table{
		tuning: tuning,
		topics: make(map[string]topic),
	}
}

// Len reports how many topics the persona currently holds any interest in.
func (t *Table) Len() int {
	return len(t.topics)
}

// Score reports how much the persona cares about a topic right now, in [0, 1].
//
// An unseen topic is not uninteresting: it scores on novelty alone, which is why
// a curious persona will look at something it has never met.
func (t *Table) Score(id string, bias Bias, now time.Time) float64 {
	current, seen := t.topics[id]
	if !seen {
		current = newTopic(now)
	}

	return t.score(current.at(now), bias)
}

// Engage records that the persona spent attention on a topic and what came of
// it. Liking moves toward the experience, familiarity grows, and the sense of
// momentum is replaced by what actually happened.
func (t *Table) Engage(topicID string, result Result, now time.Time) {
	current, seen := t.topics[topicID]
	if !seen {
		current = newTopic(now)
	}

	current = current.at(now)

	current.affinity = clampAffinity(current.affinity + result.Enjoyment*t.tuning.AffinityLearning)
	current.novelty = clampUnit(current.novelty * (freshNovelty - clampUnit(t.tuning.Habituation)))
	current.progress = clampUnit(result.Progress)
	current.lastTouched = now

	t.topics[topicID] = current
}

// Prune drops topics the persona has stopped caring about, and reports how many
// were dropped. Interest that has faded far enough simply stops being tracked,
// which keeps a long-lived persona from accumulating every passing whim.
func (t *Table) Prune(threshold float64, now time.Time) int {
	bias := NeutralBias()
	dropped := 0

	for id, current := range t.topics {
		if t.score(current.at(now), bias) >= threshold {
			continue
		}

		delete(t.topics, id)

		dropped++
	}

	return dropped
}

// score combines a faded topic with the persona's current feeling.
func (t *Table) score(current topic, bias Bias) float64 {
	raw := current.affinity*affinityWeight +
		current.novelty*noveltyWeight*t.tuning.Curiosity*bias.Novelty +
		current.progress*progressWeight

	return clampUnit(raw * bias.Overall)
}

// decayFactor returns the fraction of a value that survives elapsed time.
func decayFactor(elapsed, halfLife time.Duration) float64 {
	if halfLife <= 0 {
		return 0
	}

	return math.Pow(halving, float64(elapsed)/float64(halfLife))
}

// clampUnit confines value to [0, 1], the range novelty, progress and the final
// score live in.
func clampUnit(value float64) float64 {
	return math.Max(minScore, math.Min(maxScore, value))
}

// clampAffinity confines value to [-1, 1], the range learned liking lives in.
// Negative is aversion: a topic the persona would rather avoid.
func clampAffinity(value float64) float64 {
	return math.Max(minAffinity, math.Min(maxScore, value))
}
