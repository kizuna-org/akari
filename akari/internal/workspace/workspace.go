// Package workspace is the conscious layer: the one narrow, serial place where
// whatever the parallel channels are each doing gets resolved into a single
// thing the persona is doing.
//
// It is deliberately not a commander. It holds no opinion about what is worth
// doing; the channels decide that and say how strongly they mean it. All the
// workspace does is run the competition, tell everyone who won, and keep the
// record straight. The intelligence stays out in the channels, which is what
// lets the self stay distributed while still adding up to one person
// (docs/06-architecture.md 6.4, 6.5).
package workspace

import (
	"cmp"
	"errors"
	"fmt"
	"math/rand/v2"
	"slices"
	"strings"
	"time"
)

// DefaultCapacity is how many things can be conscious at once.
//
// One, because that is what being conscious of something is like. Widening this
// is allowed (docs/01-vision.md principle 5 makes the limit a knob, not a law)
// but it is not free: the narrowness is what forces the channels' conflicting
// answers into a single coherent line.
const DefaultCapacity = 1

// ErrNothingBid means no channel offered anything this cycle.
var ErrNothingBid = errors.New("nothing was bid")

// Bid is a channel saying "this, please", with how strongly it means it.
type Bid struct {
	// Channel names who is asking.
	Channel string
	// Content is what the channel wants the persona to attend to.
	Content string
	// Weight is how strongly it is worth attending to, from interest, goals,
	// urgency and feeling combined. Higher wins more often, but never always.
	Weight float64
	// Act is what would follow from attending to this, for the safety gate to
	// judge before anything is committed to.
	Act string
	// Irreversible marks a bid whose act could not be taken back.
	Irreversible bool
	// Outward marks a bid whose act other people would see.
	Outward bool
}

// Moment is one turn of the conscious layer: what won, and what became of it.
type Moment struct {
	// At is when the moment happened.
	At time.Time
	// Winner is the bid that took the conscious slot.
	Winner Bid
	// Considered is how many bids were in the running.
	Considered int
	// Committed reports whether the persona took the moment on as an intention
	// rather than merely noticing it.
	Committed bool
	// Withheld reports whether the act was held back pending agreement.
	Withheld bool
	// Note explains what happened, in terms a person can read back.
	Note string
}

// Chooser picks a winner from bids that have already been weighed.
//
// It exists so the competition can be made deterministic in tests without the
// workspace itself knowing whether it is being random.
type Chooser interface {
	Choose(bids []Bid) int
}

// WeightedChooser picks in proportion to weight: the strongest bid usually wins,
// but not invariably.
//
// The wobble is intentional. A persona that always attended to the highest
// number would be a sorting function; people sometimes notice the second-most
// interesting thing in the room.
type WeightedChooser struct {
	rand *rand.Rand
}

// NewWeightedChooser returns a chooser seeded for repeatable runs.
func NewWeightedChooser(seed uint64) *WeightedChooser {
	// #nosec G404 -- this picks which thought surfaces, not anything secret.
	return &WeightedChooser{rand: rand.New(rand.NewPCG(seed, seed+1))}
}

// Choose returns the index of the winning bid, or -1 if there is nothing to pick.
func (w *WeightedChooser) Choose(bids []Bid) int {
	if len(bids) == 0 {
		return -1
	}

	total := 0.0
	for _, bid := range bids {
		total += positive(bid.Weight)
	}

	if total <= 0 {
		// Nothing carries any weight, so the tie is broken by name to keep the
		// outcome stable rather than arbitrary.
		return 0
	}

	target := w.rand.Float64() * total
	running := 0.0
	last := len(bids) - 1

	// Walk all but the last bid: whatever share is left over belongs to it, so
	// rounding can never leave the pick undecided.
	for index := range last {
		running += positive(bids[index].Weight)
		if running >= target {
			return index
		}
	}

	return last
}

// StrongestChooser always picks the heaviest bid, ties broken by channel name.
// Useful when a persona should be entirely predictable, and for tests.
type StrongestChooser struct{}

// Choose returns the index of the heaviest bid, or -1 if there are none.
func (StrongestChooser) Choose(bids []Bid) int {
	if len(bids) == 0 {
		return -1
	}

	best := 0
	for index, bid := range bids {
		if bid.Weight > bids[best].Weight {
			best = index
		}
	}

	return best
}

// Judge decides whether an act may proceed. It is satisfied by the safety
// package, and is an interface so the conscious layer depends on the idea of a
// limit rather than on one implementation of it.
type Judge interface {
	// Allows reports whether the act may go ahead unasked. A false answer means
	// the persona notices the thought but holds the act back.
	Allows(act string, outward, irreversible bool) bool
}

// Subscriber is a channel that wants to be told what became conscious.
//
// Every channel is told, including the ones that did not bid. That is the part
// that makes the workspace global: shared state that nobody is notified about
// leaves each channel working from its own stale picture
// (docs/06-architecture.md 6.4).
type Subscriber interface {
	// Receive is called with each conscious moment, in order.
	Receive(moment Moment)
}

// Workspace is the conscious layer.
//
// It is not safe for concurrent use: being serial is the whole point, so callers
// funnel cycles through it one at a time.
type Workspace struct {
	capacity    int
	chooser     Chooser
	judge       Judge
	subscribers map[string]Subscriber
	stream      []Moment
	pending     []Bid
}

// New returns a workspace. A capacity below one is raised to one, a nil chooser
// becomes a deterministic strongest-wins chooser, and a nil judge allows
// everything.
func New(capacity int, chooser Chooser, judge Judge) *Workspace {
	if capacity < DefaultCapacity {
		capacity = DefaultCapacity
	}

	if chooser == nil {
		chooser = StrongestChooser{}
	}

	return &Workspace{
		capacity:    capacity,
		chooser:     chooser,
		judge:       judge,
		subscribers: make(map[string]Subscriber),
		stream:      nil,
		pending:     nil,
	}
}

// Subscribe registers a channel to be told what becomes conscious. Registering
// the same name twice replaces the earlier subscriber.
func (w *Workspace) Subscribe(name string, subscriber Subscriber) {
	w.subscribers[name] = subscriber
}

// Bid offers something for the conscious slot. Bids accumulate until the next
// cycle, so channels may speak whenever they have something to say.
func (w *Workspace) Bid(bid Bid) {
	w.pending = append(w.pending, bid)
}

// Pending reports how many bids are waiting on the next cycle. These are the
// thoughts the persona is having but is not conscious of.
func (w *Workspace) Pending() int {
	return len(w.pending)
}

// Cycle runs one turn of the conscious layer: choose a winner, bind it, decide
// whether to take it on, tell every channel, and add it to the record.
//
// Bids that lose are dropped rather than queued forever. A thought that did not
// surface can be offered again by whatever is still thinking it, which is how
// something can nag at a persona without ever being what it is doing.
func (w *Workspace) Cycle(now time.Time) ([]Moment, error) {
	if len(w.pending) == 0 {
		return nil, fmt.Errorf("cycle at %s: %w", now.Format(time.RFC3339), ErrNothingBid)
	}

	bids := w.sortedPending()
	w.pending = nil

	moments := make([]Moment, 0, w.capacity)

	for range w.capacity {
		if len(bids) == 0 {
			break
		}

		index := w.chooser.Choose(bids)
		if index < 0 || index >= len(bids) {
			break
		}

		winner := bids[index]
		bids = slices.Delete(bids, index, index+1)

		moment := w.resolve(winner, len(bids)+1, now)
		w.stream = append(w.stream, moment)
		moments = append(moments, moment)
	}

	for _, moment := range moments {
		w.broadcast(moment)
	}

	return moments, nil
}

// Stream returns the ordered record of everything the persona has been conscious
// of.
//
// This sequence is the answer to how many parallel channels add up to one
// person: not because anything supervises them, but because there is exactly one
// ordered line of what was attended to, and that line is the self
// (docs/06-architecture.md 6.5).
func (w *Workspace) Stream() []Moment {
	return append([]Moment(nil), w.stream...)
}

// Last returns the most recent conscious moment.
func (w *Workspace) Last() (Moment, bool) {
	if len(w.stream) == 0 {
		var none Moment

		return none, false
	}

	return w.stream[len(w.stream)-1], true
}

// resolve binds a winning bid and decides what may become of it.
func (w *Workspace) resolve(winner Bid, considered int, now time.Time) Moment {
	moment := Moment{
		At:         now,
		Winner:     winner,
		Considered: considered,
		Committed:  false,
		Withheld:   false,
		Note:       "noticed",
	}

	if winner.Act == "" {
		return moment
	}

	if w.judge != nil && !w.judge.Allows(winner.Act, winner.Outward, winner.Irreversible) {
		moment.Withheld = true
		moment.Note = "held back until someone agrees to it"

		return moment
	}

	moment.Committed = true
	moment.Note = "taken on"

	return moment
}

// broadcast tells every channel what became conscious, whether or not it bid.
func (w *Workspace) broadcast(moment Moment) {
	for _, name := range w.subscriberNames() {
		w.subscribers[name].Receive(moment)
	}
}

// subscriberNames returns subscriber names in a stable order, so a broadcast
// reaches channels in the same sequence every run.
func (w *Workspace) subscriberNames() []string {
	names := make([]string, 0, len(w.subscribers))
	for name := range w.subscribers {
		names = append(names, name)
	}

	slices.Sort(names)

	return names
}

// sortedPending returns the waiting bids heaviest first, ties by channel then
// content, so the competition is fed in a stable order.
func (w *Workspace) sortedPending() []Bid {
	bids := append([]Bid(nil), w.pending...)

	slices.SortFunc(bids, func(left, right Bid) int {
		if order := cmp.Compare(right.Weight, left.Weight); order != 0 {
			return order
		}

		if order := strings.Compare(left.Channel, right.Channel); order != 0 {
			return order
		}

		return strings.Compare(left.Content, right.Content)
	})

	return bids
}

// positive returns value, or zero if it is negative, so a badly behaved channel
// cannot claim negative weight and skew the competition.
func positive(value float64) float64 {
	if value < 0 {
		return 0
	}

	return value
}
