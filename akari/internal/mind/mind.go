// Package mind is where the pieces become one persona.
//
// Everything under it is deliberately ignorant of everything else: emotion does
// not know what interest is, interest does not know what a goal is, and the
// workspace does not know what any of them mean. This package is the only place
// that knows how they fit together, which is what keeps each of them simple
// enough to reason about on its own.
//
// The shape it assembles is the one in docs/06-architecture.md: parallel
// channels bidding for a single narrow conscious slot, with safety as a gate the
// result passes through rather than a weight in the sum.
package mind

import (
	"fmt"
	"time"

	"github.com/kizuna-org/akari/internal/emotion"
	"github.com/kizuna-org/akari/internal/goal"
	"github.com/kizuna-org/akari/internal/interest"
	"github.com/kizuna-org/akari/internal/memory"
	"github.com/kizuna-org/akari/internal/safety"
	"github.com/kizuna-org/akari/internal/workspace"
)

// Weights of what makes a thought worth attending to.
//
// Interest leads, but it does not rule: a persona driven by curiosity alone
// would never finish anything, and one deaf to urgency would be no use to
// anybody. Safety is not in this list on purpose; it is a gate, not a weight
// (docs/04-interest.md 4.5).
const (
	weightInterest = 0.40
	weightGoal     = 0.30
	weightUrgency  = 0.30

	// reflexWeight is what a reflex bids. It is above anything the ordinary
	// weights can reach, because flinching does not wait to be talked about
	// (docs/02-emotion.md 2.1).
	reflexWeight = 1.5

	// reflexChannel is the name a reflex bids under.
	reflexChannel = "reflex"
)

// Persona is the fixed shape of one Akari: how it feels, what draws it, how well
// it follows through, how it remembers, how wide its attention is, and what it
// is free to do.
//
// All of it is settable. A persona that forgets nothing, never tires of anything
// and holds several things in mind at once is as valid a configuration as a
// scatterbrained one; being unlike a person is a choice here, not a bug
// (docs/01-vision.md principle 1).
type Persona struct {
	// Name identifies the persona.
	Name string
	// Traits is how readily it feels.
	Traits emotion.Traits
	// Curiosity is how things draw it.
	Curiosity interest.Tuning
	// Resolve is how well it sees things through.
	Resolve goal.Tuning
	// Recollection is how it remembers and forgets.
	Recollection memory.Tuning
	// Attention is how many things it can be conscious of at once.
	Attention int
	// Policy is what it may do without asking.
	Policy safety.Policy
}

// DefaultPersona returns a persona that behaves roughly like a person.
func DefaultPersona(name string) Persona {
	return Persona{
		Name:         name,
		Traits:       emotion.DefaultTraits(),
		Curiosity:    interest.DefaultTuning(),
		Resolve:      goal.DefaultTuning(),
		Recollection: memory.DefaultTuning(),
		Attention:    workspace.DefaultCapacity,
		Policy:       safety.DefaultPolicy(),
	}
}

// Thought is what one channel offers up for attention.
type Thought struct {
	// Channel names the channel it came from.
	Channel string
	// Topic is what it is about, and the handle interest and memory use for it.
	Topic string
	// Content is the thought itself, in the persona's own words.
	Content string
	// Urgency is how much it will not wait, in [0, 1]. Deadlines and someone
	// waiting for an answer live here.
	Urgency float64
	// Act is what would follow from attending to it. Empty means merely noticing.
	Act safety.Act
	// Feeling is an emotion the thought carries, if any.
	Feeling emotion.Kind
	// Intensity is how strongly it carries it, in [0, 1].
	Intensity float64
}

// Mind is one persona: its inner state, and the conscious layer that turns many
// parallel thoughts into one thing it is doing.
//
// Mind is not safe for concurrent use. Channels may think in parallel, but they
// hand their thoughts in one at a time, because the conscious layer being serial
// is the whole point of it.
type Mind struct {
	persona   Persona
	feeling   *emotion.State
	interests *interest.Table
	goals     *goal.Set
	memories  *memory.Store
	gate      *safety.Gate
	conscious *workspace.Workspace
	lastTick  time.Time
	started   bool
}

// New assembles a persona. A nil chooser gives a deterministic mind, which is
// what tests and reproducible runs want; a weighted chooser gives one whose
// attention wanders a little, which is what a person is like.
func New(persona Persona, chooser workspace.Chooser, similarity memory.Similarity) *Mind {
	gate := safety.New(persona.Policy)

	mind := &Mind{
		persona:   persona,
		feeling:   emotion.New(persona.Traits),
		interests: interest.New(persona.Curiosity),
		goals:     goal.New(persona.Resolve),
		memories:  memory.New(persona.Recollection, similarity),
		gate:      gate,
		conscious: nil,
		lastTick:  time.Time{},
		started:   false,
	}

	mind.conscious = workspace.New(persona.Attention, chooser, gateAdapter{gate: gate})

	return mind
}

// Name reports which persona this is.
func (m *Mind) Name() string {
	return m.persona.Name
}

// Feeling gives access to what the persona feels, for channels that need to
// colour their own work by it.
func (m *Mind) Feeling() *emotion.State {
	return m.feeling
}

// Interests gives access to what draws the persona.
func (m *Mind) Interests() *interest.Table {
	return m.interests
}

// Goals gives access to what the persona wants and has taken on.
func (m *Mind) Goals() *goal.Set {
	return m.goals
}

// Memories gives access to what the persona remembers.
func (m *Mind) Memories() *memory.Store {
	return m.memories
}

// Conscious gives access to the conscious layer, for channels that want to
// listen in on what became of their bids.
func (m *Mind) Conscious() *workspace.Workspace {
	return m.conscious
}

// Consider takes a thought from a channel, works out how much it is worth
// attending to, and offers it for the conscious slot.
//
// The weighing happens here rather than in the channel so that every channel is
// judged by the same measure. A channel decides what it is thinking; it does not
// get to decide how much that matters.
func (m *Mind) Consider(thought Thought, now time.Time) {
	if thought.Intensity > 0 {
		m.feeling.Feel(thought.Feeling, thought.Intensity)
	}

	m.conscious.Bid(workspace.Bid{
		Channel:      thought.Channel,
		Content:      thought.Content,
		Weight:       m.weigh(thought, now),
		Act:          thought.Act.Kind,
		Irreversible: thought.Act.Reach == safety.ReachIrreversible,
		Outward:      thought.Act.Reach == safety.ReachOutward,
	})
}

// Tick advances the persona to now and runs one turn of its conscious layer.
//
// Nothing to attend to is a perfectly good outcome: an idle mind reports no
// moments rather than inventing something to do. That is the design's "doing
// nothing is a real choice" (docs/07-autonomy.md 7.2).
func (m *Mind) Tick(now time.Time) []workspace.Moment {
	m.age(now)
	m.reflex()

	moments, err := m.conscious.Cycle(now)
	if err != nil {
		// Nothing was on the persona's mind this turn, so nothing became of it.
		// That is an ordinary outcome rather than a fault: doing nothing is a
		// real choice, and the loop carries on regardless of what one turn came
		// to (docs/07-autonomy.md 7.1, 7.2).
		return nil
	}

	for _, moment := range moments {
		m.remember(moment, now)
	}

	return moments
}

// Engaged records what came of actually spending time on a topic, so interest
// moves with experience rather than with what was merely intended.
func (m *Mind) Engaged(topic string, result interest.Result, now time.Time) {
	m.interests.Engage(topic, result, now)
}

// Sleep settles the day: memories are compacted, the faintest are let go, and
// interests nothing has come of are dropped. It reports what was forgotten and
// what was set aside.
func (m *Mind) Sleep(now time.Time) (int, int) {
	forgotten := m.memories.Sleep(now)
	dropped := m.interests.Prune(pruneThreshold, now)

	return forgotten, dropped
}

// pruneThreshold is how little interest a topic must hold before the persona
// stops keeping track of it at all.
const pruneThreshold = 0.05

// weigh works out how much a thought is worth attending to.
func (m *Mind) weigh(thought Thought, now time.Time) float64 {
	bias := interest.Bias{
		Overall: m.feeling.InterestBias(),
		Novelty: m.feeling.NoveltyBias(),
	}

	drawn := m.interests.Score(thought.Topic, bias, now)
	urgency := clampUnit(thought.Urgency) * m.feeling.CautionBias()

	return drawn*weightInterest +
		m.pull(thought)*weightGoal +
		clampUnit(urgency)*weightUrgency
}

// pull reports how much the persona's current intentions favour a thought.
//
// A thought that serves what the persona is already carrying gets that
// intention's grip behind it. A thought about anything else gets nothing, which
// is what makes a persona with something in hand harder to distract
// (docs/05-goal.md 5.5).
func (m *Mind) pull(thought Thought) float64 {
	return m.goals.Grip(thought.Topic)
}

// reflex offers a bid for whatever the persona's feeling is doing on its own.
//
// This is the fast path: it does not consult interest, and it bids above what
// the ordinary weighing can produce, so a jolt gets attention even mid-task.
func (m *Mind) reflex() {
	tendency, ok := m.feeling.Reflex()
	if !ok {
		return
	}

	m.conscious.Bid(workspace.Bid{
		Channel:      reflexChannel,
		Content:      fmt.Sprintf("something makes the persona want to %s", tendency),
		Weight:       reflexWeight,
		Act:          "",
		Irreversible: false,
		Outward:      false,
	})
}

// age lets feeling fade by however long has passed since the last tick.
func (m *Mind) age(now time.Time) {
	if !m.started {
		m.started = true
		m.lastTick = now

		return
	}

	elapsed := now.Sub(m.lastTick)
	if elapsed > 0 {
		m.feeling.Decay(elapsed)
		m.lastTick = now
	}
}

// remember commits a conscious moment to memory.
//
// Only what reached consciousness is remembered. The thoughts that lost their
// bid leave no trace, which is why a persona cannot account for everything that
// crossed its mind (docs/06-architecture.md 6.4).
func (m *Mind) remember(moment workspace.Moment, now time.Time) {
	_, intensity := m.feeling.Dominant()

	will := 0.0
	if moment.Committed {
		// Taking something on is itself an act of meaning to remember it.
		will = committedWill
	}

	m.memories.Perceive(memory.Experience{
		Content:      moment.Winner.Content,
		Feeling:      intensity,
		Will:         will,
		Confidential: false,
	}, now)
}

// committedWill is how deliberately a persona remembers something it took on.
const committedWill = 0.6

// clampUnit confines value to [0, 1].
func clampUnit(value float64) float64 {
	if value < 0 {
		return 0
	}

	if value > 1 {
		return 1
	}

	return value
}

// gateAdapter lets the safety gate satisfy the conscious layer's Judge without
// either package having to know about the other.
type gateAdapter struct {
	gate *safety.Gate
}

// Allows reports whether an act may go ahead unasked.
func (g gateAdapter) Allows(act string, outward, irreversible bool) bool {
	reach := safety.ReachReversible

	switch {
	case irreversible:
		reach = safety.ReachIrreversible
	case outward:
		reach = safety.ReachOutward
	}

	return g.gate.Permits(safety.Act{Kind: act, Reach: reach, Forbidden: false})
}
