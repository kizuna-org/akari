package memory

import (
	"cmp"
	"errors"
	"fmt"
	"math"
	"slices"
	"strings"
	"time"
)

// Tuning defaults for a persona that remembers roughly like a person.
//
// Forgetfulness in particular is a knob, not a requirement: docs/01-vision.md
// principle 1 allows a persona that forgets almost nothing just as readily as a
// scatterbrained one. What is not allowed is a perfect searchable log that owes
// nothing to how the persona actually experienced things.
const (
	// DefaultForgetfulness means the faintest memories fall away at each sleep.
	DefaultForgetfulness = 1.0
	// DefaultCompaction means a memory keeps most of its vividness per sleep,
	// losing detail gradually rather than all at once.
	DefaultCompaction = 0.85
	// DefaultRecallLimit is how many memories come back at once. Recall is
	// narrow because attention is.
	DefaultRecallLimit = 5
)

const (
	// forgetThreshold is the retention score below which a memory is let go at
	// the reference forgetfulness.
	//
	// It sits just above what an unremarkable memory scores once its recency has
	// worn off, so today's dull moments survive tonight and fade a few nights
	// later. Nothing is forgotten the day it happens.
	forgetThreshold = 0.22

	// recencyHalfLife is how long a memory stays fresh in the recency term.
	recencyHalfLife = 48 * time.Hour
	// popularitySaturation is the number of recalls at which the popularity term
	// is effectively full: being recalled a few times matters, a hundred times
	// no more than that.
	popularitySaturation = 5.0

	// Weights of the terms that decide how well a memory holds on and how
	// readily it comes back (docs/03-memory.md 3.4).
	weightStrength   = 0.20
	weightEmotion    = 0.30
	weightWill       = 0.25
	weightPopularity = 0.10
	weightRecency    = 0.15
	weightSimilarity = 0.60

	halving      = 0.5
	minScore     = 0.0
	maxScore     = 1.0
	fullStrength = 1.0
)

// ErrUnknownFragment means no memory is held under that identifier.
var ErrUnknownFragment = errors.New("unknown fragment")

// Tuning is the fixed shape of a persona's memory.
type Tuning struct {
	// Forgetfulness scales how readily faint memories are let go at each sleep.
	// Zero keeps everything; higher lets go sooner.
	Forgetfulness float64
	// Compaction is the share of vividness a memory keeps per sleep. Lower
	// blurs detail faster, which is what makes recollections turn into gist.
	Compaction float64
	// RecallLimit is how many memories can come back at once.
	RecallLimit int
}

// DefaultTuning returns memory settings that behave roughly like a person's.
func DefaultTuning() Tuning {
	return Tuning{
		Forgetfulness: DefaultForgetfulness,
		Compaction:    DefaultCompaction,
		RecallLimit:   DefaultRecallLimit,
	}
}

// Experience is what is being committed to memory.
type Experience struct {
	// Content is what happened, in the persona's own words.
	Content string
	// Feeling is how strongly the moment was felt, in [0, 1]. Strong feeling
	// makes a memory vivid and easy to reach for (docs/03-memory.md 3.4).
	Feeling float64
	// Will is how deliberately the persona set out to remember this, in [0, 1].
	//
	// This is the honest way to keep hold of a name or a promise: not a special
	// rule that exempts it from forgetting, but having meant to remember it.
	Will float64
	// Confidential marks something agreed to be kept private, so it is never
	// offered up in recall for anyone else.
	Confidential bool
}

// Fragment is one remembered thing: the smallest unit memory deals in.
type Fragment struct {
	id           string
	content      string
	layer        Layer
	strength     float64
	feeling      float64
	will         float64
	recalls      int
	confidential bool
	pinned       bool
	createdAt    time.Time
	touchedAt    time.Time
}

// ID identifies the fragment.
func (f Fragment) ID() string { return f.id }

// Content is what is remembered, as far as it has survived compaction.
func (f Fragment) Content() string { return f.content }

// Layer reports how far the memory has settled.
func (f Fragment) Layer() Layer { return f.layer }

// Strength reports how vivid the memory still is, in [0, 1].
func (f Fragment) Strength() float64 { return f.strength }

// Recalls reports how many times the memory has been brought back.
func (f Fragment) Recalls() int { return f.recalls }

// Confidential reports whether the memory was agreed to be kept private.
func (f Fragment) Confidential() bool { return f.confidential }

// Pinned reports whether the memory is being held against forgetting.
func (f Fragment) Pinned() bool { return f.pinned }

// CreatedAt reports when the memory was formed.
func (f Fragment) CreatedAt() time.Time { return f.createdAt }

// Cue is what a persona is reaching for when it tries to remember.
type Cue struct {
	// Text is the handle: a topic, a name, a turn of phrase.
	Text string
	// Audience identifies who the recollection is for. Memories agreed to be
	// private are withheld unless the audience is the one they were shared with.
	Audience string
	// IncludeConfidential lets the persona reach its own private memories, for
	// thinking rather than telling.
	IncludeConfidential bool
}

// Store is everything a persona remembers.
//
// Store is not safe for concurrent use; whatever serialises access to the
// persona's inner state owns it.
type Store struct {
	tuning     Tuning
	similarity Similarity
	fragments  map[string]Fragment
	confidants map[string]string
	sequence   int
}

// New returns an empty store. Passing a nil Similarity falls back to keyword
// overlap, so a store is always usable.
func New(tuning Tuning, similarity Similarity) *Store {
	if similarity == nil {
		similarity = TokenOverlap{}
	}

	return &Store{
		tuning:     tuning,
		similarity: similarity,
		fragments:  make(map[string]Fragment),
		confidants: make(map[string]string),
		sequence:   0,
	}
}

// Len reports how many memories are held.
func (s *Store) Len() int {
	return len(s.fragments)
}

// Perceive takes something in. It arrives in the input layer, vivid but not yet
// settled anywhere.
func (s *Store) Perceive(experience Experience, now time.Time) string {
	s.sequence++
	fragmentID := fmt.Sprintf("frag-%d", s.sequence)

	s.fragments[fragmentID] = Fragment{
		id:           fragmentID,
		content:      experience.Content,
		layer:        LayerInput,
		strength:     fullStrength,
		feeling:      clampUnit(experience.Feeling),
		will:         clampUnit(experience.Will),
		recalls:      0,
		confidential: experience.Confidential,
		pinned:       false,
		createdAt:    now,
		touchedAt:    now,
	}

	return fragmentID
}

// Confide records who a private memory was shared with, so it can be spoken of
// with them and no one else.
func (s *Store) Confide(fragmentID, audience string) error {
	fragment, known := s.fragments[fragmentID]
	if !known {
		return fmt.Errorf("confide %q: %w", fragmentID, ErrUnknownFragment)
	}

	fragment.confidential = true
	s.fragments[fragmentID] = fragment
	s.confidants[fragmentID] = audience

	return nil
}

// Pin holds a memory against forgetting.
//
// This is a provisional scaffold, not the intended mechanism. The honest way for
// something to stick is for the persona to have meant to remember it, which is
// what Experience.Will expresses; pinning short-circuits that while the weights
// are still being tuned, and is meant to be removed (docs/03-memory.md 3.6).
func (s *Store) Pin(fragmentID string) error {
	fragment, known := s.fragments[fragmentID]
	if !known {
		return fmt.Errorf("pin %q: %w", fragmentID, ErrUnknownFragment)
	}

	fragment.pinned = true
	s.fragments[fragmentID] = fragment

	return nil
}

// Get returns one memory as it currently stands.
func (s *Store) Get(fragmentID string) (Fragment, bool) {
	fragment, known := s.fragments[fragmentID]

	return fragment, known
}

// Attend marks that a memory has been noticed, drawing it into the current
// thread of thought.
func (s *Store) Attend(fragmentID string, now time.Time) error {
	fragment, known := s.fragments[fragmentID]
	if !known {
		return fmt.Errorf("attend %q: %w", fragmentID, ErrUnknownFragment)
	}

	if fragment.layer < LayerContext {
		fragment.layer = LayerContext
	}

	fragment.touchedAt = now
	s.fragments[fragmentID] = fragment

	return nil
}

// Recall reaches for memories that match a cue, strongest first.
//
// Recalling is not searching: what comes back is shaped by how vivid the memory
// is, how it felt, whether it was meant to be kept, how often it has surfaced
// before and how long ago it was. A faint memory of something dull may simply
// not come, even though it is still there.
//
// What does come back lands in working memory, and comes back a little more
// readily next time for having been reached for.
func (s *Store) Recall(cue Cue, now time.Time) []Fragment {
	scored := s.matches(cue, now)

	// Strongest first, and ties broken by identifier so the same memories always
	// come back in the same order.
	slices.SortFunc(scored, func(left, right scoredFragment) int {
		if order := cmp.Compare(right.score, left.score); order != 0 {
			return order
		}

		return strings.Compare(left.fragment.id, right.fragment.id)
	})

	limit := s.recallLimit()
	if len(scored) > limit {
		scored = scored[:limit]
	}

	recalled := make([]Fragment, 0, len(scored))

	for _, candidate := range scored {
		fragment := s.fragments[candidate.fragment.id]
		fragment.recalls++
		fragment.touchedAt = now

		if fragment.layer > LayerWorking {
			fragment.layer = LayerWorking
		}

		s.fragments[fragment.id] = fragment
		recalled = append(recalled, fragment)
	}

	return recalled
}

// Sleep settles the day's memories, and reports how many were let go.
//
// Three things happen at once, as docs/03-memory.md 3.3 describes: what was too
// faint to keep is forgotten, what remains loses some of its detail, and what
// survives moves one step further toward being simply known.
func (s *Store) Sleep(now time.Time) int {
	forgotten := 0
	threshold := s.forgetThreshold()

	for _, fragmentID := range s.sortedIDs() {
		fragment := s.fragments[fragmentID]

		if !fragment.pinned && s.retention(fragment, now) < threshold {
			delete(s.fragments, fragmentID)
			delete(s.confidants, fragmentID)

			forgotten++

			continue
		}

		fragment.strength = clampUnit(fragment.strength * s.compaction())
		fragment.layer = fragment.layer.next()
		s.fragments[fragmentID] = fragment
	}

	return forgotten
}

// scoredFragment pairs a memory with how well it answers a cue.
type scoredFragment struct {
	fragment Fragment
	score    float64
}

// matches gathers every memory the cue is allowed to reach, with its score.
func (s *Store) matches(cue Cue, now time.Time) []scoredFragment {
	scored := make([]scoredFragment, 0, len(s.fragments))

	for _, fragmentID := range s.sortedIDs() {
		fragment := s.fragments[fragmentID]
		if !s.mayRecall(fragment, cue) {
			continue
		}

		similarity := clampUnit(s.similarity.Score(cue.Text, fragment.content))
		if similarity <= minScore {
			continue
		}

		scored = append(scored, scoredFragment{
			fragment: fragment,
			score:    s.retention(fragment, now) + similarity*weightSimilarity,
		})
	}

	return scored
}

// mayRecall reports whether a memory can be surfaced for this cue. Something
// agreed to be private stays private, whatever the persona happens to feel.
func (s *Store) mayRecall(fragment Fragment, cue Cue) bool {
	if !fragment.confidential {
		return true
	}

	if cue.IncludeConfidential {
		return true
	}

	confidant, agreed := s.confidants[fragment.id]

	return agreed && confidant == cue.Audience
}

// retention reports how firmly a memory holds on, independent of any cue.
func (s *Store) retention(fragment Fragment, now time.Time) float64 {
	elapsed := now.Sub(fragment.touchedAt)
	recency := decayFactor(elapsed, recencyHalfLife)
	popularity := math.Min(float64(fragment.recalls)/popularitySaturation, maxScore)

	return fragment.strength*weightStrength +
		fragment.feeling*weightEmotion +
		fragment.will*weightWill +
		popularity*weightPopularity +
		recency*weightRecency
}

// sortedIDs returns every held memory's identifier in a stable order, so
// forgetting and recall never depend on map iteration order.
func (s *Store) sortedIDs() []string {
	ids := make([]string, 0, len(s.fragments))
	for fragmentID := range s.fragments {
		ids = append(ids, fragmentID)
	}

	slices.Sort(ids)

	return ids
}

// forgetThreshold returns the retention below which a memory is let go. A
// persona with no forgetfulness keeps everything.
func (s *Store) forgetThreshold() float64 {
	if s.tuning.Forgetfulness <= 0 {
		return 0
	}

	return forgetThreshold * s.tuning.Forgetfulness
}

// compaction returns the share of vividness kept per sleep.
func (s *Store) compaction() float64 {
	if s.tuning.Compaction <= 0 || s.tuning.Compaction > maxScore {
		return DefaultCompaction
	}

	return s.tuning.Compaction
}

// recallLimit returns how many memories may come back at once.
func (s *Store) recallLimit() int {
	if s.tuning.RecallLimit < 1 {
		return DefaultRecallLimit
	}

	return s.tuning.RecallLimit
}

// decayFactor returns the fraction of a value that survives elapsed time.
func decayFactor(elapsed, halfLife time.Duration) float64 {
	if elapsed <= 0 {
		return maxScore
	}

	if halfLife <= 0 {
		return minScore
	}

	return math.Pow(halving, float64(elapsed)/float64(halfLife))
}

// clampUnit confines value to [0, 1], the range every memory figure lives in.
func clampUnit(value float64) float64 {
	return math.Max(minScore, math.Min(maxScore, value))
}
