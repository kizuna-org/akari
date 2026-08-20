package goal

import (
	"errors"
	"fmt"
	"math"
	"slices"
	"time"
)

// Tuning defaults for a persona that sees things through about as well as a
// person does. Each value is a per-persona knob (docs/01-vision.md principle 1):
// a flighty persona and a dogged one are both valid.
const (
	// DefaultPersistence means an intention resists middling distractions but
	// yields to strong ones.
	DefaultPersistence = 0.6
	// DefaultCapacity means one intention at a time, which is the human default.
	DefaultCapacity = 1
	// DefaultSetbackTolerance means a few failures exhaust the will to continue.
	DefaultSetbackTolerance = 3
)

const (
	minGrip = 0.0
	maxGrip = 1.0
)

// Errors returned when a caller asks for something the set cannot do.
var (
	// ErrUnknownDesire means no desire is held under that identifier.
	ErrUnknownDesire = errors.New("unknown desire")
	// ErrUnknownIntention means no intention is being carried under that
	// identifier.
	ErrUnknownIntention = errors.New("unknown intention")
	// ErrAlreadyCommitted means the desire has already become an intention.
	ErrAlreadyCommitted = errors.New("desire already committed")
	// ErrNoCapacity means the persona is already carrying as much as it can.
	ErrNoCapacity = errors.New("no room for another intention")
)

// Tuning is the fixed shape of a persona's follow-through.
type Tuning struct {
	// Persistence is how strongly a fresh intention resists being displaced,
	// in [0, 1]. Low is easily distracted; high finishes what it starts.
	Persistence float64
	// Capacity is how many intentions the persona can carry at once. One is the
	// human default; more than one is allowed but invites divided attention.
	Capacity int
	// SetbackTolerance is how many setbacks an intention survives before its
	// grip is spent and letting go becomes the natural move.
	SetbackTolerance int
}

// DefaultTuning returns follow-through settings that behave roughly like a
// person's.
func DefaultTuning() Tuning {
	return Tuning{
		Persistence:      DefaultPersistence,
		Capacity:         DefaultCapacity,
		SetbackTolerance: DefaultSetbackTolerance,
	}
}

// Set is what a persona wishes for and what it has actually taken on.
//
// Set is not safe for concurrent use; whatever serialises access to the
// persona's inner state owns it.
type Set struct {
	tuning  Tuning
	desires map[string]Desire
	active  map[string]Intention
	settled []Intention
}

// New returns an empty set for a persona with the given follow-through.
func New(tuning Tuning) *Set {
	return &Set{
		tuning:  tuning,
		desires: make(map[string]Desire),
		active:  make(map[string]Intention),
		settled: nil,
	}
}

// Wish records something the persona would like to be true, replacing any
// earlier wish with the same identifier. Wishes may contradict each other.
func (s *Set) Wish(desire Desire) {
	s.desires[desire.ID] = desire
}

// Desires reports how many wishes the persona is holding.
func (s *Set) Desires() int {
	return len(s.desires)
}

// Strongest reports the wish that pulls hardest and whether there was one at
// all. It is a suggestion about what to commit to next, not a decision.
//
// Equal pulls resolve to the lowest identifier, so the answer is stable: the
// same wishes always yield the same suggestion. A persona that changed its mind
// at random between two equally appealing options would not read as one that had
// preferences at all.
func (s *Set) Strongest() (Desire, bool) {
	strongest := Desire{ID: "", Want: "", Pull: 0}
	found := false

	for _, id := range s.sortedDesireIDs() {
		desire := s.desires[id]
		if found && desire.Pull <= strongest.Pull {
			continue
		}

		strongest = desire
		found = true
	}

	return strongest, found
}

// Commit turns a wish into an intention: the persona stops merely wanting the
// thing and starts carrying it.
func (s *Set) Commit(desireID string, now time.Time) (Intention, error) {
	desire, known := s.desires[desireID]
	if !known {
		return Intention{}, fmt.Errorf("commit %q: %w", desireID, ErrUnknownDesire)
	}

	if _, carrying := s.active[desireID]; carrying {
		return Intention{}, fmt.Errorf("commit %q: %w", desireID, ErrAlreadyCommitted)
	}

	if len(s.active) >= s.capacity() {
		return Intention{}, fmt.Errorf("commit %q: %w", desireID, ErrNoCapacity)
	}

	intention := Intention{
		id:          desire.ID,
		want:        desire.Want,
		status:      StatusActive,
		committedAt: now,
		settledAt:   time.Time{},
		setbacks:    0,
	}
	s.active[desireID] = intention

	return intention, nil
}

// Active returns the intentions the persona is currently carrying.
func (s *Set) Active() []Intention {
	carried := make([]Intention, 0, len(s.active))
	for _, intention := range s.active {
		carried = append(carried, intention)
	}

	return carried
}

// Settled returns the intentions that have already been completed or abandoned,
// oldest first. This is the record of what the persona actually followed
// through on.
func (s *Set) Settled() []Intention {
	return append([]Intention(nil), s.settled...)
}

// Complete records that an intention was seen through.
func (s *Set) Complete(id string, now time.Time) error {
	return s.settle(id, StatusCompleted, now)
}

// Abandon records that the persona let an intention go. Giving up is a normal
// move, not a failure of the design: holding on forever is not human either
// (docs/05-goal.md 5.3).
func (s *Set) Abandon(id string, now time.Time) error {
	return s.settle(id, StatusAbandoned, now)
}

// Setback records that pursuing an intention ran into trouble, weakening the
// persona's grip on it. Enough setbacks and the grip is spent, at which point
// the intention no longer resists anything.
func (s *Set) Setback(intentionID string) error {
	intention, carrying := s.active[intentionID]
	if !carrying {
		return fmt.Errorf("setback %q: %w", intentionID, ErrUnknownIntention)
	}

	intention.setbacks++
	s.active[intentionID] = intention

	return nil
}

// Grip reports how strongly the persona still holds an intention, in [0, 1]. It
// starts at the persona's persistence and wears down with every setback. An
// intention not being carried has no grip at all.
func (s *Set) Grip(id string) float64 {
	intention, carrying := s.active[id]
	if !carrying {
		return minGrip
	}

	tolerance := s.setbackTolerance()
	spent := float64(intention.setbacks) / float64(tolerance)

	return clampUnit(s.persistence() * (maxGrip - spent))
}

// Spent reports whether an intention has taken all the setbacks it can bear, so
// letting go is now the natural move.
func (s *Set) Spent(id string) bool {
	intention, carrying := s.active[id]
	if !carrying {
		return false
	}

	return intention.setbacks >= s.setbackTolerance()
}

// Holds reports whether the persona stays its course against a distraction of
// the given weight.
//
// This is what keeps a persona from chasing every passing interest: carrying on
// is the default, and a challenger has to be worth more than the current grip
// to pull it away (docs/07-autonomy.md 7.2).
func (s *Set) Holds(id string, challenger float64) bool {
	if _, carrying := s.active[id]; !carrying {
		return false
	}

	return s.Grip(id) >= challenger
}

// sortedDesireIDs returns every held wish's identifier in a stable order.
func (s *Set) sortedDesireIDs() []string {
	ids := make([]string, 0, len(s.desires))
	for id := range s.desires {
		ids = append(ids, id)
	}

	slices.Sort(ids)

	return ids
}

// settle moves an intention out of the carried set with a final status.
func (s *Set) settle(intentionID string, status Status, now time.Time) error {
	intention, carrying := s.active[intentionID]
	if !carrying {
		return fmt.Errorf("settle %q: %w", intentionID, ErrUnknownIntention)
	}

	intention.status = status
	intention.settledAt = now

	delete(s.active, intentionID)

	s.settled = append(s.settled, intention)

	return nil
}

// capacity returns how many intentions may be carried, never less than one.
func (s *Set) capacity() int {
	if s.tuning.Capacity < DefaultCapacity {
		return DefaultCapacity
	}

	return s.tuning.Capacity
}

// setbackTolerance returns how many setbacks an intention bears, never less
// than one, so grip always has somewhere to fall to.
func (s *Set) setbackTolerance() int {
	if s.tuning.SetbackTolerance < 1 {
		return 1
	}

	return s.tuning.SetbackTolerance
}

// persistence returns the persona's follow-through, confined to [0, 1].
func (s *Set) persistence() float64 {
	return clampUnit(s.tuning.Persistence)
}

// clampUnit confines value to [0, 1], the range grip lives in.
func clampUnit(value float64) float64 {
	return math.Max(minGrip, math.Min(maxGrip, value))
}
