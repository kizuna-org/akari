package loop

import (
	"context"
	"testing"
	"time"

	"github.com/kizuna-org/akari/internal/emotion"
	"github.com/kizuna-org/akari/internal/memory"
	"github.com/kizuna-org/akari/internal/mind"
	"github.com/kizuna-org/akari/internal/safety"
	"github.com/kizuna-org/akari/internal/workspace"
)

const (
	personaName = "akari"
	idleChannel = "speculation"
	musicTopic  = "music"
)

// reference is a fixed instant so tests never depend on the wall clock.
func reference() time.Time {
	return time.Date(2026, time.August, 20, 9, 0, 0, 0, time.UTC)
}

// stepClock advances by a fixed amount each time it is read, so a loop can be
// walked through a simulated day without waiting for one.
type stepClock struct {
	at   time.Time
	step time.Duration
}

func (c *stepClock) Now() time.Time {
	now := c.at
	c.at = c.at.Add(c.step)

	return now
}

func newPersona() *mind.Mind {
	return mind.New(mind.DefaultPersona(personaName), workspace.StrongestChooser{}, memory.TokenOverlap{})
}

func idleThought() mind.Thought {
	return mind.Thought{
		Channel:   idleChannel,
		Topic:     musicTopic,
		Content:   "a passing thought about music",
		Urgency:   0,
		Act:       safety.Act{Kind: "", Reach: safety.ReachInternal, Forbidden: false},
		Feeling:   emotion.Surprise,
		Intensity: 0,
	}
}

func TestDefaultConfig(t *testing.T) {
	t.Parallel()

	want := Config{Interval: DefaultInterval, Nightly: DefaultNightly}

	if got := DefaultConfig(); got != want {
		t.Fatalf("DefaultConfig() = %#v, want %#v", got, want)
	}
}

func TestSystemClockAdvances(t *testing.T) {
	t.Parallel()

	if (SystemClock{}).Now().IsZero() {
		t.Fatal("Now() is the zero time, want a real reading")
	}
}

func TestNewNormalisesConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		config      Config
		wantTick    time.Duration
		wantNightly time.Duration
	}{
		{
			name:        "the reference pacing is kept",
			config:      DefaultConfig(),
			wantTick:    DefaultInterval,
			wantNightly: DefaultNightly,
		},
		{
			name:        "a chosen pacing is kept",
			config:      Config{Interval: time.Second, Nightly: time.Hour},
			wantTick:    time.Second,
			wantNightly: time.Hour,
		},
		{
			name:        "zero falls back to the reference",
			config:      Config{Interval: 0, Nightly: 0},
			wantTick:    DefaultInterval,
			wantNightly: DefaultNightly,
		},
		{
			name:        "negative falls back to the reference",
			config:      Config{Interval: -time.Second, Nightly: -time.Hour},
			wantTick:    DefaultInterval,
			wantNightly: DefaultNightly,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			runner := New(newPersona(), nil, testCase.config, nil)

			if runner.config.Interval != testCase.wantTick {
				t.Fatalf("Interval = %v, want %v", runner.config.Interval, testCase.wantTick)
			}

			if runner.config.Nightly != testCase.wantNightly {
				t.Fatalf("Nightly = %v, want %v", runner.config.Nightly, testCase.wantNightly)
			}
		})
	}
}

func TestNewFallsBackToTheRealClock(t *testing.T) {
	t.Parallel()

	runner := New(newPersona(), nil, DefaultConfig(), nil)

	if _, ok := runner.clock.(SystemClock); !ok {
		t.Fatalf("clock = %T, want the system clock", runner.clock)
	}
}

func TestStepOnAnIdleMind(t *testing.T) {
	t.Parallel()

	clock := &stepClock{at: reference(), step: time.Second}
	runner := New(newPersona(), clock, DefaultConfig(), nil)

	moments := runner.Step()
	if len(moments) != 0 {
		t.Fatalf("Step() returned %d moments, want an idle persona to do nothing", len(moments))
	}

	if runner.Ticks() != 1 {
		t.Fatalf("Ticks() = %d, want 1", runner.Ticks())
	}

	if runner.Moments() != 0 {
		t.Fatalf("Moments() = %d, want 0", runner.Moments())
	}
}

func TestStepAttendsToWhatIsOnThePersonasMind(t *testing.T) {
	t.Parallel()

	clock := &stepClock{at: reference(), step: time.Second}
	persona := newPersona()
	runner := New(persona, clock, DefaultConfig(), nil)

	persona.Consider(idleThought(), reference())

	moments := runner.Step()
	if len(moments) != 1 {
		t.Fatalf("Step() returned %d moments, want 1", len(moments))
	}

	if runner.Moments() != 1 {
		t.Fatalf("Moments() = %d, want 1", runner.Moments())
	}
}

func TestStepLogsWhatBecameConscious(t *testing.T) {
	t.Parallel()

	clock := &stepClock{at: reference(), step: time.Second}
	persona := newPersona()
	logger, records := recordingLogger()
	runner := New(persona, clock, DefaultConfig(), logger)

	persona.Consider(idleThought(), reference())
	runner.Step()

	if len(*records) == 0 {
		t.Fatal("nothing was logged, want the conscious moment on the record")
	}
}

func TestStepSettlesTheDayOnceEnoughHasPassed(t *testing.T) {
	t.Parallel()

	// A clock that jumps a day per read, so the second step is a night later.
	clock := &stepClock{at: reference(), step: DefaultNightly}
	persona := newPersona()
	runner := New(persona, clock, DefaultConfig(), nil)

	persona.Memories().Perceive(memory.Experience{
		Content:      "nothing much",
		Feeling:      0,
		Will:         0,
		Confidential: false,
	}, reference())

	// The first step only marks when the day began.
	runner.Step()

	if got := persona.Memories().Len(); got != 1 {
		t.Fatalf("Memories().Len() = %d, want nothing forgotten on the first step", got)
	}

	// By the next step a night has gone by, so the day gets settled. The dull
	// memory survives its first night and fades over later ones.
	runner.Step()

	fragment, known := persona.Memories().Get("frag-1")
	if !known {
		t.Fatal("the memory is gone, want a dull memory to survive its first night")
	}

	if fragment.Strength() >= 1 {
		t.Fatalf("Strength() = %v, want the night to have cost it some detail", fragment.Strength())
	}
}

func TestStepDoesNotSettleTheDayTooSoon(t *testing.T) {
	t.Parallel()

	clock := &stepClock{at: reference(), step: time.Second}
	persona := newPersona()
	runner := New(persona, clock, DefaultConfig(), nil)

	persona.Memories().Perceive(memory.Experience{
		Content:      "nothing much",
		Feeling:      0,
		Will:         0,
		Confidential: false,
	}, reference())

	for range 5 {
		runner.Step()
	}

	fragment, known := persona.Memories().Get("frag-1")
	if !known {
		t.Fatal("the memory is gone, want no settling within a single day")
	}

	if fragment.Strength() != 1 {
		t.Fatalf("Strength() = %v, want it untouched within the day", fragment.Strength())
	}
}

func TestRunStopsWhenTheContextIsDone(t *testing.T) {
	t.Parallel()

	clock := &stepClock{at: reference(), step: time.Second}
	persona := newPersona()
	runner := New(persona, clock, Config{Interval: time.Millisecond, Nightly: DefaultNightly}, nil)

	ctx, cancel := context.WithCancel(context.Background())
	stopped := make(chan struct{})

	go func() {
		runner.Run(ctx)
		close(stopped)
	}()

	// Let it turn over a few times, then ask it to rest.
	deadline := time.After(time.Second)

	for runner.Ticks() < 3 {
		select {
		case <-deadline:
			cancel()
			t.Fatal("the loop did not tick, want it running on its own")
		default:
		}
	}

	cancel()

	select {
	case <-stopped:
	case <-time.After(time.Second):
		t.Fatal("the loop did not stop, want it to rest when asked")
	}
}

func TestRunStopsImmediatelyOnACancelledContext(t *testing.T) {
	t.Parallel()

	clock := &stepClock{at: reference(), step: time.Second}
	runner := New(newPersona(), clock, Config{Interval: time.Hour, Nightly: DefaultNightly}, nil)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	stopped := make(chan struct{})

	go func() {
		runner.Run(ctx)
		close(stopped)
	}()

	select {
	case <-stopped:
	case <-time.After(time.Second):
		t.Fatal("the loop kept going, want it to rest at once")
	}
}
