package mind

import (
	"context"
	"maps"
	"math"
	"time"

	"github.com/kizuna-org/akari/internal/emotion"
	"github.com/kizuna-org/akari/internal/goal"
	"github.com/kizuna-org/akari/internal/interest"
	"github.com/kizuna-org/akari/internal/meaning"
	"github.com/kizuna-org/akari/internal/memory"
	"github.com/kizuna-org/akari/internal/persona"
)

const (
	ErrInner fault = "inner state requires a fixed persona, positive resource/time settings and a clock"
	ErrLoad  fault = "adopted conscious load must be finite and within awareness capacity; rest must be nonnegative"
)

type Part string

const (
	Emotion  Part = "emotion"
	Interest Part = "interest"
	Memory   Part = "memory"
	Goals    Part = "goals"
	Fatigue  Part = "fatigue"
)

// Experience is the subject's context: no configuration vector or full archival history.
type Experience struct {
	Emotion    emotion.State      `json:"emotion"`
	Interests  []interest.Focus   `json:"interests"`
	Goals      goal.State         `json:"goals"`
	Context    []memory.Fragment  `json:"context"`
	Working    []memory.Fragment  `json:"working"`
	Agreements []memory.Agreement `json:"agreements"`
	Fatigue    float64            `json:"fatigue"`
	Versions   map[Part]uint64    `json:"versions"`
}

// Update is an unadopted interpretation. Load/Rest are supplied by trusted runtime, not the model.
type Update struct {
	Expected  map[Part]uint64
	Appraisal *emotion.Appraisal
	Interests []interest.Focus
	Memory    *memory.Change
	Goals     *goal.Change
	Load      float64
	Rest      time.Duration
	Advance   bool
}

// Settings chooses scaffold time units and resource limits, not a new personality.
type Settings struct {
	Contents int
	Dynamics emotion.Dynamics
	RestUnit time.Duration
	Clock    func() time.Time
}

type innerState struct {
	profile   persona.Config
	settings  Settings
	emotion   emotion.State
	interests []interest.Focus
	goals     goal.State
	memory    memory.State
	fatigue   float64
	versions  map[Part]uint64
}

// Checkpoint is an administrative persistence envelope; never pass it to a Channel.
// This envelope covers inner state only; add an atomic action outbox before external updates.
type Checkpoint struct {
	Schema   int            `json:"schema"`
	Persona  persona.Config `json:"persona"`
	Snapshot Snapshot       `json:"snapshot"`
	Archive  memory.State   `json:"archive"`
}

func NewWithInner(capacity int, config persona.Config, settings Settings) (*Workspace, error) {
	err := config.Validate()
	if err != nil {
		return nil, err
	}

	if settings.Contents < 1 || settings.Clock == nil || settings.RestUnit <= 0 ||
		settings.Dynamics.EmotionHalfLife <= 0 || settings.Dynamics.MoodHalfLife <= 0 {
		return nil, ErrInner
	}

	instant := settings.Clock()
	if instant.IsZero() {
		return nil, ErrInner
	}

	workspace, err := New(capacity)
	if err != nil {
		return nil, err
	}

	workspace.inner = &innerState{
		profile: config.Clone(), settings: settings,
		emotion:   emotion.State{Feelings: nil, Readiness: nil, Mood: baseline(config), At: instant},
		interests: nil, goals: goal.State{Desires: nil, Intentions: nil},
		memory:  memory.State{Context: nil, Working: nil, Day: nil, Sleeping: nil, Agreements: nil},
		fatigue: 0, versions: map[Part]uint64{Emotion: 0, Interest: 0, Memory: 0, Goals: 0, Fatigue: 0},
	}

	return workspace, nil
}

func (experience *Experience) Clone() *Experience {
	if experience == nil {
		return nil
	}

	return &Experience{
		Emotion: experience.Emotion.Clone(), Interests: interest.Clone(experience.Interests),
		Goals: experience.Goals.Clone(), Context: memory.CloneFragments(experience.Context),
		Working: memory.CloneFragments(experience.Working), Agreements: memory.CloneAgreements(experience.Agreements),
		Fatigue: experience.Fatigue, Versions: maps.Clone(experience.Versions),
	}
}

func (update *Update) Clone() *Update {
	if update == nil {
		return nil
	}

	result := &Update{
		Expected: maps.Clone(update.Expected), Appraisal: nil, Interests: nil, Memory: nil, Goals: nil,
		Load: update.Load, Rest: update.Rest, Advance: update.Advance,
	}

	if update.Appraisal != nil {
		value := update.Appraisal.Clone()
		result.Appraisal = &value
	}

	if update.Interests != nil {
		result.Interests = interest.Clone(update.Interests)
	}

	if update.Memory != nil {
		value := update.Memory.Clone()
		result.Memory = &value
	}

	if update.Goals != nil {
		value := update.Goals.Clone()
		result.Goals = &value
	}

	return result
}

// Recall does not hold the commit lock while ranking a copied archive.
func (workspace *Workspace) Recall(ctx context.Context, query memory.Query) ([]memory.Fragment, error) {
	err := ctx.Err()
	if err != nil {
		return nil, err
	}

	workspace.mu.Lock()
	state := workspace.inner

	if state == nil {
		workspace.mu.Unlock()

		return nil, ErrInner
	}

	archive := state.memory.Clone()
	weights := [5]float64{}
	copy(weights[:], state.profile.Vector[persona.SemanticWeight:persona.WillWeight+1])
	workspace.mu.Unlock()

	return memory.Recall(archive, query, weights)
}

func (workspace *Workspace) Export() (Checkpoint, error) {
	workspace.mu.Lock()
	defer workspace.mu.Unlock()

	if workspace.inner == nil {
		return Checkpoint{}, ErrInner
	}

	return Checkpoint{
		Schema: 1, Persona: workspace.inner.profile.Clone(),
		Snapshot: Snapshot{
			Revision: workspace.revision, Items: maps.Clone(workspace.items), Experience: workspace.inner.experience(),
		},
		Archive: workspace.inner.memory.Clone(),
	}, nil
}

// AllowsDisclosure is the live provenance policy supplied to trusted external adapters.
// Recall remains permitted; only an outward action checks the sharing agreement.
func (workspace *Workspace) AllowsDisclosure(sources []string, recipient string) bool {
	workspace.mu.Lock()
	defer workspace.mu.Unlock()

	return workspace.inner != nil && memory.Allows(workspace.inner.memory.Agreements, sources, recipient)
}

func (update *Update) parts() []Part {
	const partCount = 5

	parts := make([]Part, 0, partCount)

	if update.Appraisal != nil || update.Advance {
		parts = append(parts, Emotion)
	}

	if update.Interests != nil {
		parts = append(parts, Interest)
	}

	if update.Memory != nil {
		parts = append(parts, Memory)
	}

	if update.Goals != nil {
		parts = append(parts, Goals)
	}

	if update.Rest > 0 {
		parts = append(parts, Fatigue)
	}

	return parts
}

func (state *innerState) experience() *Experience {
	if state == nil {
		return nil
	}

	return &Experience{
		Emotion: state.emotion.Clone(), Interests: interest.Clone(state.interests), Goals: state.goals.Clone(),
		Context: memory.CloneFragments(state.memory.Context), Working: memory.CloneFragments(state.memory.Working),
		Agreements: memory.CloneAgreements(state.memory.Agreements),
		Fatigue:    state.fatigue, Versions: maps.Clone(state.versions),
	}
}

func (workspace *Workspace) prepare(update *Update) (*innerState, error) {
	if update == nil {
		return workspace.inner, nil
	}

	err := workspace.checkUpdate(update)
	if err != nil {
		return nil, err
	}

	next := *workspace.inner
	next.versions = maps.Clone(next.versions)

	err = next.apply(update)
	if err != nil {
		return nil, err
	}

	if next.size() > next.settings.Contents {
		return nil, ErrCapacity
	}

	return &next, nil
}

func (workspace *Workspace) checkUpdate(update *Update) error {
	if workspace.inner == nil {
		return ErrInner
	}

	if !workspace.validLoad(update) {
		return ErrLoad
	}

	for _, part := range update.parts() {
		if _, exists := update.Expected[part]; !exists {
			return ErrProposal
		}
	}

	for part, version := range update.Expected {
		current, exists := workspace.inner.versions[part]
		if !exists || current != version {
			return ErrConflict
		}
	}

	return nil
}

func (workspace *Workspace) validLoad(update *Update) bool {
	return meaning.Finite(update.Load) && update.Load >= 0 &&
		update.Load <= workspace.inner.profile.Vector[persona.AwarenessCount] && update.Rest >= 0
}

func (state *innerState) apply(update *Update) error {
	err := state.applyEmotion(update)
	if err != nil {
		return err
	}

	if update.Interests != nil {
		err = interest.Validate(update.Interests)
		if err != nil {
			return err
		}

		state.interests = interest.Clone(update.Interests)
	}

	if update.Memory != nil {
		err = state.applyMemory(update)
		if err != nil {
			return err
		}
	}

	if update.Goals != nil {
		state.goals, err = goal.Apply(state.goals, *update.Goals, int(state.profile.Vector[persona.IntentionCount]))
		if err != nil {
			return err
		}
	}

	state.fatigue += update.Load * state.profile.Vector[persona.Fatigability]
	state.fatigue = math.Max(0, state.fatigue-float64(update.Rest)/float64(state.settings.RestUnit)*
		state.profile.Vector[persona.Recovery])

	return nil
}

func (state *innerState) applyMemory(update *Update) error {
	change := update.Memory.Clone()

	if update.Appraisal != nil {
		strength := 0.0

		for _, feeling := range state.emotion.Feelings {
			strength = math.Max(strength, feeling.Strength)
		}

		for index := range change.Remember {
			change.Remember[index].Emotion = math.Max(change.Remember[index].Emotion, strength)
		}
	}

	var err error

	state.memory, err = memory.Apply(state.memory, change)

	return err
}

func (state *innerState) applyEmotion(update *Update) error {
	if update.Appraisal == nil && !update.Advance {
		return nil
	}

	value, err := emotion.Advance(state.emotion, state.settings.Clock(), baseline(state.profile),
		state.profile.Vector[persona.Lingering], state.settings.Dynamics)
	if err != nil {
		return err
	}

	if update.Appraisal != nil {
		value, err = emotion.Respond(value, *update.Appraisal,
			state.profile.Vector[persona.Reactivity], state.profile.Vector[persona.Empathy])
		if err != nil {
			return err
		}
	}

	state.emotion = value

	return nil
}

func (state *innerState) size() int {
	count := len(state.emotion.Feelings) + len(state.emotion.Readiness) + len(state.interests) +
		len(state.goals.Desires) + len(state.goals.Intentions) + len(state.memory.Context) + len(state.memory.Working) +
		len(state.memory.Day) + len(state.memory.Sleeping) + len(state.memory.Agreements)

	for _, intention := range state.goals.Intentions {
		count += len(intention.Steps)
	}

	return count
}

func baseline(profile persona.Config) emotion.Mood {
	return emotion.Mood{2*profile.Vector[persona.Baseline] - 1, 0, 0}
}
