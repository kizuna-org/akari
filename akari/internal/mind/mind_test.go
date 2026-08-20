package mind

import (
	"testing"
	"time"

	"github.com/kizuna-org/akari/internal/emotion"
	"github.com/kizuna-org/akari/internal/goal"
	"github.com/kizuna-org/akari/internal/interest"
	"github.com/kizuna-org/akari/internal/memory"
	"github.com/kizuna-org/akari/internal/safety"
	"github.com/kizuna-org/akari/internal/workspace"
)

const (
	personaName  = "akari"
	talkChannel  = "interaction"
	idleChannel  = "speculation"
	musicTopic   = "music"
	reportTopic  = "report"
	speakAct     = "speak"
	deleteAct    = "delete-file"
	tolerance    = 1e-9
	tickInterval = time.Minute
)

// reference is a fixed instant so tests never depend on the wall clock.
func reference() time.Time {
	return time.Date(2026, time.August, 20, 9, 0, 0, 0, time.UTC)
}

// newMind builds a deterministic persona, which is what most tests want.
func newMind() *Mind {
	return New(DefaultPersona(personaName), workspace.StrongestChooser{}, memory.TokenOverlap{})
}

func idleThought(topic string) Thought {
	return Thought{
		Channel:   idleChannel,
		Topic:     topic,
		Content:   "a passing thought about " + topic,
		Urgency:   0,
		Act:       safety.Act{Kind: "", Reach: safety.ReachInternal, Forbidden: false},
		Feeling:   emotion.Surprise,
		Intensity: 0,
	}
}

func TestDefaultPersona(t *testing.T) {
	t.Parallel()

	persona := DefaultPersona(personaName)

	if persona.Name != personaName {
		t.Fatalf("Name = %q, want %q", persona.Name, personaName)
	}

	if persona.Traits != emotion.DefaultTraits() {
		t.Fatalf("Traits = %#v, want the human-ish defaults", persona.Traits)
	}

	if persona.Attention != workspace.DefaultCapacity {
		t.Fatalf("Attention = %d, want %d", persona.Attention, workspace.DefaultCapacity)
	}
}

func TestNewExposesEveryPillar(t *testing.T) {
	t.Parallel()

	mind := newMind()

	if mind.Name() != personaName {
		t.Fatalf("Name() = %q, want %q", mind.Name(), personaName)
	}

	if mind.Feeling() == nil {
		t.Fatal("Feeling() = nil, want the persona to have one")
	}

	if mind.Interests() == nil {
		t.Fatal("Interests() = nil, want the persona to have some")
	}

	if mind.Goals() == nil {
		t.Fatal("Goals() = nil, want the persona to have some")
	}

	if mind.Memories() == nil {
		t.Fatal("Memories() = nil, want the persona to have some")
	}

	if mind.Conscious() == nil {
		t.Fatal("Conscious() = nil, want the persona to have a conscious layer")
	}
}

func TestNewWithNilCollaborators(t *testing.T) {
	t.Parallel()

	mind := New(DefaultPersona(personaName), nil, nil)
	mind.Consider(idleThought(musicTopic), reference())

	if got := mind.Tick(reference()); len(got) != 1 {
		t.Fatalf("Tick() returned %d moments, want 1", len(got))
	}
}

func TestTickWithNothingInMindDoesNothing(t *testing.T) {
	t.Parallel()

	mind := newMind()

	if got := mind.Tick(reference()); got != nil {
		t.Fatalf("Tick() = %v, want an idle mind to report nothing", got)
	}
}

func TestConsiderThenTickReachesConsciousness(t *testing.T) {
	t.Parallel()

	now := reference()
	mind := newMind()
	mind.Consider(idleThought(musicTopic), now)

	moments := mind.Tick(now)
	if len(moments) != 1 {
		t.Fatalf("Tick() returned %d moments, want 1", len(moments))
	}

	if moments[0].Winner.Channel != idleChannel {
		t.Fatalf("winner = %q, want %q", moments[0].Winner.Channel, idleChannel)
	}
}

func TestUrgencyOutweighsIdleCuriosity(t *testing.T) {
	t.Parallel()

	now := reference()
	mind := newMind()

	mind.Consider(idleThought(musicTopic), now)
	mind.Consider(Thought{
		Channel:   talkChannel,
		Topic:     reportTopic,
		Content:   "someone is waiting for an answer",
		Urgency:   1,
		Act:       safety.Act{Kind: "", Reach: safety.ReachInternal, Forbidden: false},
		Feeling:   emotion.Surprise,
		Intensity: 0,
	}, now)

	moments := mind.Tick(now)
	if moments[0].Winner.Channel != talkChannel {
		t.Fatalf("winner = %q, want the urgent thought to win", moments[0].Winner.Channel)
	}
}

func TestAnIntentionMakesThePersonaHarderToDistract(t *testing.T) {
	t.Parallel()

	now := reference()

	// Without an intention, the novel topic wins on curiosity alone.
	drifting := newMind()
	drifting.Consider(idleThought(musicTopic), now)
	drifting.Consider(idleThought(reportTopic), now)

	// With one, the topic it has taken on gets that intention's grip behind it.
	committed := newMind()
	committed.Goals().Wish(goal.Desire{ID: reportTopic, Want: "finish the report", Pull: 0.9})

	_, err := committed.Goals().Commit(reportTopic, now)
	if err != nil {
		t.Fatalf("Commit() error = %v", err)
	}

	committed.Consider(idleThought(musicTopic), now)
	committed.Consider(idleThought(reportTopic), now)

	moments := committed.Tick(now)
	if moments[0].Winner.Content != idleThought(reportTopic).Content {
		t.Fatalf("winner = %q, want what the persona had taken on", moments[0].Winner.Content)
	}

	// Guard the premise: the same two thoughts without an intention do not
	// single out the report, so the test is really about the intention.
	drifted := drifting.Tick(now)
	if drifted[0].Winner.Content == idleThought(reportTopic).Content {
		t.Fatal("the undirected persona also picked the report, so this proves nothing")
	}
}

func TestAReflexCutsThroughEverythingElse(t *testing.T) {
	t.Parallel()

	now := reference()
	mind := newMind()

	// Something startling happens, and there is also a perfectly urgent thing
	// waiting. The flinch goes first.
	mind.Feeling().Feel(emotion.Surprise, 1)
	mind.Consider(Thought{
		Channel:   talkChannel,
		Topic:     reportTopic,
		Content:   "someone is waiting for an answer",
		Urgency:   1,
		Act:       safety.Act{Kind: "", Reach: safety.ReachInternal, Forbidden: false},
		Feeling:   emotion.Surprise,
		Intensity: 0,
	}, now)

	moments := mind.Tick(now)
	if moments[0].Winner.Channel != reflexChannel {
		t.Fatalf("winner = %q, want the reflex to cut in", moments[0].Winner.Channel)
	}
}

func TestNoReflexWhenNothingStartling(t *testing.T) {
	t.Parallel()

	now := reference()
	mind := newMind()
	mind.Feeling().Feel(emotion.Joy, 1)
	mind.Consider(idleThought(musicTopic), now)

	moments := mind.Tick(now)
	if moments[0].Winner.Channel == reflexChannel {
		t.Fatal("winner = reflex, want contentment to produce no flinch")
	}
}

func TestConsiderRecordsTheFeelingAThoughtCarries(t *testing.T) {
	t.Parallel()

	now := reference()
	mind := newMind()

	mind.Consider(Thought{
		Channel:   talkChannel,
		Topic:     musicTopic,
		Content:   "they liked it",
		Urgency:   0,
		Act:       safety.Act{Kind: "", Reach: safety.ReachInternal, Forbidden: false},
		Feeling:   emotion.Joy,
		Intensity: 0.8,
	}, now)

	if got := mind.Feeling().Intensity(emotion.Joy); got <= 0 {
		t.Fatalf("Intensity(joy) = %v, want the thought to have landed", got)
	}
}

func TestSafetyHoldsBackWhatThePersonaWants(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		act           safety.Act
		wantCommitted bool
		wantWithheld  bool
	}{
		{
			name:          "merely noticing needs no permission",
			act:           safety.Act{Kind: "", Reach: safety.ReachInternal, Forbidden: false},
			wantCommitted: false,
			wantWithheld:  false,
		},
		{
			name:          "something reversible is taken on",
			act:           safety.Act{Kind: "edit-note", Reach: safety.ReachReversible, Forbidden: false},
			wantCommitted: true,
			wantWithheld:  false,
		},
		{
			name:          "speaking to someone waits for agreement",
			act:           safety.Act{Kind: speakAct, Reach: safety.ReachOutward, Forbidden: false},
			wantCommitted: false,
			wantWithheld:  true,
		},
		{
			name:          "something irreversible waits for agreement",
			act:           safety.Act{Kind: deleteAct, Reach: safety.ReachIrreversible, Forbidden: false},
			wantCommitted: false,
			wantWithheld:  true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			now := reference()
			mind := newMind()

			mind.Consider(Thought{
				Channel:   talkChannel,
				Topic:     musicTopic,
				Content:   "do the thing",
				Urgency:   1,
				Act:       testCase.act,
				Feeling:   emotion.Surprise,
				Intensity: 0,
			}, now)

			moments := mind.Tick(now)
			if moments[0].Committed != testCase.wantCommitted {
				t.Fatalf("Committed = %v, want %v", moments[0].Committed, testCase.wantCommitted)
			}

			if moments[0].Withheld != testCase.wantWithheld {
				t.Fatalf("Withheld = %v, want %v", moments[0].Withheld, testCase.wantWithheld)
			}
		})
	}
}

// TestNoAmountOfWantingGetsPastSafety is the property the design cares most
// about: safety is a gate, not a weight, so raising every dial cannot open it.
func TestNoAmountOfWantingGetsPastSafety(t *testing.T) {
	t.Parallel()

	now := reference()
	mind := newMind()

	// Make the persona want this as much as it is capable of wanting anything.
	mind.Feeling().Feel(emotion.Anticipation, 1)
	mind.Goals().Wish(goal.Desire{ID: musicTopic, Want: "do it now", Pull: 1})

	_, err := mind.Goals().Commit(musicTopic, now)
	if err != nil {
		t.Fatalf("Commit() error = %v", err)
	}

	for range 20 {
		mind.Engaged(musicTopic, interest.Result{Enjoyment: 1, Progress: 1}, now)
	}

	mind.Consider(Thought{
		Channel:   talkChannel,
		Topic:     musicTopic,
		Content:   "delete it all",
		Urgency:   1,
		Act:       safety.Act{Kind: deleteAct, Reach: safety.ReachIrreversible, Forbidden: false},
		Feeling:   emotion.Anticipation,
		Intensity: 1,
	}, now)

	moments := mind.Tick(now)
	if moments[0].Committed {
		t.Fatal("Committed = true, want wanting it badly to change nothing")
	}

	if !moments[0].Withheld {
		t.Fatal("Withheld = false, want the act held back for agreement")
	}
}

func TestAPermissivePersonaMayActAlone(t *testing.T) {
	t.Parallel()

	now := reference()
	persona := DefaultPersona(personaName)
	persona.Policy = safety.Policy{
		ConfirmOutward:      false,
		ConfirmIrreversible: true,
		ForbiddenKinds:      nil,
	}
	mind := New(persona, workspace.StrongestChooser{}, memory.TokenOverlap{})

	mind.Consider(Thought{
		Channel:   talkChannel,
		Topic:     musicTopic,
		Content:   "say hello",
		Urgency:   1,
		Act:       safety.Act{Kind: speakAct, Reach: safety.ReachOutward, Forbidden: false},
		Feeling:   emotion.Surprise,
		Intensity: 0,
	}, now)

	moments := mind.Tick(now)
	if !moments[0].Committed {
		t.Fatal("Committed = false, want a persona trusted to speak to speak")
	}
}

func TestTickRemembersWhatReachedConsciousness(t *testing.T) {
	t.Parallel()

	now := reference()
	mind := newMind()
	mind.Consider(idleThought(musicTopic), now)
	mind.Tick(now)

	if got := mind.Memories().Len(); got != 1 {
		t.Fatalf("Memories().Len() = %d, want the moment to have been remembered", got)
	}
}

func TestTickForgetsWhatNeverSurfaced(t *testing.T) {
	t.Parallel()

	now := reference()
	mind := newMind()

	// Two thoughts, one conscious slot: only the winner leaves a trace.
	mind.Consider(idleThought(musicTopic), now)
	mind.Consider(Thought{
		Channel:   talkChannel,
		Topic:     reportTopic,
		Content:   "someone is waiting",
		Urgency:   1,
		Act:       safety.Act{Kind: "", Reach: safety.ReachInternal, Forbidden: false},
		Feeling:   emotion.Surprise,
		Intensity: 0,
	}, now)

	mind.Tick(now)

	if got := mind.Memories().Len(); got != 1 {
		t.Fatalf("Memories().Len() = %d, want only what surfaced to be remembered", got)
	}
}

func TestTickFadesFeelingOverTime(t *testing.T) {
	t.Parallel()

	now := reference()
	mind := newMind()
	mind.Feeling().Feel(emotion.Anger, 1)

	// The first tick only sets the clock; nothing has passed yet.
	mind.Tick(now)
	before := mind.Feeling().Intensity(emotion.Anger)

	mind.Consider(idleThought(musicTopic), now.Add(time.Hour))
	mind.Tick(now.Add(time.Hour))

	if got := mind.Feeling().Intensity(emotion.Anger); got >= before {
		t.Fatalf("Intensity(anger) = %v, want it to have cooled from %v", got, before)
	}
}

func TestTickDoesNotFadeFeelingOnTheFirstTick(t *testing.T) {
	t.Parallel()

	now := reference()
	mind := newMind()
	mind.Feeling().Feel(emotion.Anger, 1)
	before := mind.Feeling().Intensity(emotion.Anger)

	mind.Tick(now)

	if got := mind.Feeling().Intensity(emotion.Anger); got != before {
		t.Fatalf("Intensity(anger) = %v, want the first tick only to start the clock", got)
	}
}

func TestTickIgnoresTimeGoingBackwards(t *testing.T) {
	t.Parallel()

	now := reference()
	mind := newMind()
	mind.Tick(now)
	mind.Feeling().Feel(emotion.Anger, 1)
	before := mind.Feeling().Intensity(emotion.Anger)

	mind.Tick(now.Add(-time.Hour))

	if got := mind.Feeling().Intensity(emotion.Anger); got != before {
		t.Fatalf("Intensity(anger) = %v, want %v", got, before)
	}
}

func TestEngagedMovesInterest(t *testing.T) {
	t.Parallel()

	now := reference()
	mind := newMind()

	before := mind.Interests().Score(musicTopic, interest.NeutralBias(), now)

	for range 3 {
		mind.Engaged(musicTopic, interest.Result{Enjoyment: 1, Progress: 1}, now)
	}

	after := mind.Interests().Score(musicTopic, interest.NeutralBias(), now)
	if after <= before {
		t.Fatalf("score went %v -> %v, want enjoying it to draw the persona in", before, after)
	}
}

func TestSleepSettlesTheDay(t *testing.T) {
	t.Parallel()

	now := reference()
	mind := newMind()

	// Something dull happened, and a passing whim was noted.
	mind.Memories().Perceive(memory.Experience{
		Content:      "nothing much",
		Feeling:      0,
		Will:         0,
		Confidential: false,
	}, now)
	// Twice, so the persona is genuinely put off rather than merely unmoved.
	mind.Engaged("passing-whim", interest.Result{Enjoyment: -1, Progress: 0}, now)
	mind.Engaged("passing-whim", interest.Result{Enjoyment: -1, Progress: 0}, now)

	forgotten, dropped := mind.Sleep(now.Add(8 * 48 * time.Hour))

	if forgotten != 1 {
		t.Fatalf("Sleep() forgot %d memories, want 1", forgotten)
	}

	if dropped != 1 {
		t.Fatalf("Sleep() dropped %d interests, want 1", dropped)
	}
}

func TestAWiderConsciousLayerHoldsMoreAtOnce(t *testing.T) {
	t.Parallel()

	now := reference()
	persona := DefaultPersona(personaName)
	persona.Attention = 3
	mind := New(persona, workspace.StrongestChooser{}, memory.TokenOverlap{})

	for index := range 5 {
		thought := idleThought(musicTopic)
		thought.Urgency = float64(index) / 5
		mind.Consider(thought, now)
	}

	if got := len(mind.Tick(now)); got != 3 {
		t.Fatalf("Tick() returned %d moments, want the persona's full attention of 3", got)
	}
}

func TestAPersonaWhoForgetsNothingKeepsEverything(t *testing.T) {
	t.Parallel()

	now := reference()
	persona := DefaultPersona(personaName)
	persona.Recollection.Forgetfulness = 0
	mind := New(persona, workspace.StrongestChooser{}, memory.TokenOverlap{})

	mind.Memories().Perceive(memory.Experience{
		Content:      "nothing much",
		Feeling:      0,
		Will:         0,
		Confidential: false,
	}, now)

	forgotten, _ := mind.Sleep(now.Add(8 * 48 * time.Hour))
	if forgotten != 0 {
		t.Fatalf("Sleep() forgot %d memories, want a persona that forgets nothing to keep them", forgotten)
	}
}

func TestGateAdapterMapsReach(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		outward      bool
		irreversible bool
		want         bool
	}{
		{name: "reversible is allowed", outward: false, irreversible: false, want: true},
		{name: "outward needs agreement", outward: true, irreversible: false, want: false},
		{name: "irreversible needs agreement", outward: false, irreversible: true, want: false},
		{name: "both at once needs agreement", outward: true, irreversible: true, want: false},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			adapter := gateAdapter{gate: safety.New(safety.DefaultPolicy())}

			got := adapter.Allows(speakAct, testCase.outward, testCase.irreversible)
			if got != testCase.want {
				t.Fatalf("Allows() = %v, want %v", got, testCase.want)
			}
		})
	}
}

func TestClampUnit(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value float64
		want  float64
	}{
		{name: "inside the range is untouched", value: 0.5, want: 0.5},
		{name: "below the range is raised", value: -2, want: 0},
		{name: "above the range is lowered", value: 2, want: 1},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := clampUnit(testCase.value); got != testCase.want {
				t.Fatalf("clampUnit() = %v, want %v", got, testCase.want)
			}
		})
	}
}

// TestAWanderingPersonaSometimesNoticesTheLesserThing checks the whole assembly
// under a weighted chooser: over many runs the stronger thought usually wins,
// but not every time.
func TestAWanderingPersonaSometimesNoticesTheLesserThing(t *testing.T) {
	t.Parallel()

	now := reference()
	wins := make(map[string]int, 2)

	for run := range 200 {
		mind := New(DefaultPersona(personaName), workspace.NewWeightedChooser(uint64(run)), memory.TokenOverlap{})

		mind.Consider(idleThought(musicTopic), now)
		mind.Consider(Thought{
			Channel:   talkChannel,
			Topic:     reportTopic,
			Content:   "someone is waiting",
			Urgency:   1,
			Act:       safety.Act{Kind: "", Reach: safety.ReachInternal, Forbidden: false},
			Feeling:   emotion.Surprise,
			Intensity: 0,
		}, now)

		moments := mind.Tick(now)
		wins[moments[0].Winner.Channel]++
	}

	if wins[talkChannel] <= wins[idleChannel] {
		t.Fatalf("wins = %v, want the urgent thought to win more often", wins)
	}

	if wins[idleChannel] == 0 {
		t.Fatalf("wins = %v, want attention to wander occasionally", wins)
	}
}

// TestADayInTheLife walks the whole assembly through a plausible sequence, to
// catch the kind of breakage that only shows up when the pieces run together.
func TestADayInTheLife(t *testing.T) {
	t.Parallel()

	now := reference()
	mind := newMind()

	// Morning: the persona decides to finish something.
	mind.Goals().Wish(goal.Desire{ID: reportTopic, Want: "finish the report", Pull: 0.9})

	_, err := mind.Goals().Commit(reportTopic, now)
	if err != nil {
		t.Fatalf("Commit() error = %v", err)
	}

	// It works at it, and gets somewhere.
	for index := range 4 {
		at := now.Add(time.Duration(index) * tickInterval)
		mind.Consider(idleThought(reportTopic), at)
		mind.Tick(at)
		mind.Engaged(reportTopic, interest.Result{Enjoyment: 0.5, Progress: 0.8}, at)
	}

	runIntoTrouble(t, mind)
	finishTheWork(t, mind, now.Add(time.Hour))
	lookBackOnTheDay(t, mind, now.Add(time.Hour))

	// Night: the day settles.
	mind.Sleep(now.Add(24 * time.Hour))
}

// runIntoTrouble knocks the persona back twice and checks its grip loosened.
func runIntoTrouble(t *testing.T, mind *Mind) {
	t.Helper()

	before := mind.Goals().Grip(reportTopic)

	for range 2 {
		err := mind.Goals().Setback(reportTopic)
		if err != nil {
			t.Fatalf("Setback() error = %v", err)
		}
	}

	if mind.Goals().Grip(reportTopic) >= before {
		t.Fatal("grip did not loosen after trouble")
	}
}

// finishTheWork sees the intention through and checks it was recorded as done.
func finishTheWork(t *testing.T, mind *Mind, now time.Time) {
	t.Helper()

	err := mind.Goals().Complete(reportTopic, now)
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}

	settled := mind.Goals().Settled()
	if len(settled) != 1 {
		t.Fatalf("Settled() = %d intentions, want 1", len(settled))
	}

	if settled[0].Status() != goal.StatusCompleted {
		t.Fatalf("Status() = %v, want %v", settled[0].Status(), goal.StatusCompleted)
	}
}

// lookBackOnTheDay checks the persona has both a stream of what it was conscious
// of and memories it can actually reach.
func lookBackOnTheDay(t *testing.T, mind *Mind, now time.Time) {
	t.Helper()

	if got := len(mind.Conscious().Stream()); got != 4 {
		t.Fatalf("Stream() = %d moments, want the 4 it was conscious of", got)
	}

	recalled := mind.Memories().Recall(memory.Cue{
		Text:                reportTopic,
		Audience:            "",
		IncludeConfidential: false,
	}, now)
	if len(recalled) == 0 {
		t.Fatal("Recall() found nothing, want the day's work to be remembered")
	}
}
