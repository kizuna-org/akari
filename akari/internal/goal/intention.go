package goal

import "time"

// Status is where an intention stands.
type Status int

const (
	// StatusActive means the persona is still carrying the intention.
	StatusActive Status = iota
	// StatusCompleted means the intention was seen through.
	StatusCompleted
	// StatusAbandoned means the persona let the intention go.
	StatusAbandoned
)

// Names used when reporting a status, kept as constants so the implementation
// and its tests cannot drift apart.
const (
	NameActive    = "active"
	NameCompleted = "completed"
	NameAbandoned = "abandoned"
	NameUnknown   = "unknown"
)

// String returns a stable lowercase name, for logs and debugging.
func (s Status) String() string {
	switch s {
	case StatusActive:
		return NameActive
	case StatusCompleted:
		return NameCompleted
	case StatusAbandoned:
		return NameAbandoned
	default:
		return NameUnknown
	}
}

// Desire is something a persona would like to be true. Desires are allowed to
// contradict each other: wanting to rest and wanting to finish can both be
// real at once (docs/05-goal.md 5.2).
type Desire struct {
	// ID identifies the desire.
	ID string
	// Want describes what is wished for, in the persona's own terms.
	Want string
	// Pull is how much the wish draws, in [0, 1].
	Pull float64
}

// Intention is a desire the persona has taken on: the one it is actually
// carrying. Unlike desires, intentions must not contradict each other, which is
// why committing to one is a deliberate act rather than a passing feeling.
//
// An intention is what makes a persona finish things. Interest wanders by
// nature; intention is what holds a course across that wandering
// (docs/05-goal.md 5.1).
type Intention struct {
	id          string
	want        string
	status      Status
	committedAt time.Time
	settledAt   time.Time
	setbacks    int
}

// ID returns the identifier of the desire this intention came from.
func (intention Intention) ID() string {
	return intention.id
}

// Want describes what the persona is trying to bring about.
func (intention Intention) Want() string {
	return intention.want
}

// Status reports whether the intention is still being carried, and if not, how
// it ended.
func (intention Intention) Status() Status {
	return intention.status
}

// Setbacks reports how many times the persona has run into trouble pursuing
// this intention. Enough of them and its grip is spent.
func (intention Intention) Setbacks() int {
	return intention.setbacks
}

// CommittedAt reports when the persona took the intention on.
func (intention Intention) CommittedAt() time.Time {
	return intention.committedAt
}

// SettledAt reports when the intention was completed or abandoned. It is the
// zero time while the intention is still active.
func (intention Intention) SettledAt() time.Time {
	return intention.settledAt
}

// Age reports how long the persona has been carrying the intention. For an
// intention that has already ended, it measures up to the moment it settled.
func (intention Intention) Age(now time.Time) time.Duration {
	end := now
	if intention.status != StatusActive {
		end = intention.settledAt
	}

	elapsed := end.Sub(intention.committedAt)
	if elapsed < 0 {
		return 0
	}

	return elapsed
}
