package persona

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/kizuna-org/akari/internal/config"
	"github.com/kizuna-org/akari/internal/emotion"
	"github.com/kizuna-org/akari/internal/mind"
	"github.com/kizuna-org/akari/internal/safety"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

const personaName = "akari"

func testConfig() config.Config {
	return config.Config{
		Addr:     ":8080",
		Database: config.Database{Host: "", Port: 0, User: "", Password: "", Name: "", SSLMode: ""},
		Persona: config.Persona{
			Name:     personaName,
			Seed:     1,
			Interval: time.Millisecond,
			Nightly:  24 * time.Hour,
		},
	}
}

func TestNewBuildsTheConfiguredPersona(t *testing.T) {
	t.Parallel()

	built := New(testConfig())

	if built.Name() != personaName {
		t.Fatalf("Name() = %q, want %q", built.Name(), personaName)
	}

	if built.Feeling() == nil {
		t.Fatal("Feeling() = nil, want the persona to have one")
	}
}

func TestNewIsReproducibleForASeed(t *testing.T) {
	t.Parallel()

	// The same seed must give the same run: a persona whose attention wanders
	// is still one whose day can be replayed.
	first := attentionOrder(t, 1)
	second := attentionOrder(t, 1)

	if first != second {
		t.Fatalf("runs diverged: %q then %q, want the same seed to replay", first, second)
	}
}

// attentionOrder runs a persona through a fixed set of thoughts and reports
// which channel won each turn.
func attentionOrder(t *testing.T, seed uint64) string {
	t.Helper()

	cfg := testConfig()
	cfg.Persona.Seed = seed
	built := New(cfg)

	now := time.Date(2026, time.August, 20, 9, 0, 0, 0, time.UTC)

	var order strings.Builder

	for index := range 12 {
		at := now.Add(time.Duration(index) * time.Second)

		built.Consider(thought("interaction", 0.5), at)
		built.Consider(thought("speculation", 0.4), at)

		for _, moment := range built.Tick(at) {
			order.WriteString(moment.Winner.Channel)
			order.WriteString(" ")
		}
	}

	return order.String()
}

func thought(channel string, urgency float64) mind.Thought {
	return mind.Thought{
		Channel:   channel,
		Topic:     channel,
		Content:   "something from " + channel,
		Urgency:   urgency,
		Act:       safety.Act{Kind: "", Reach: safety.ReachInternal, Forbidden: false},
		Feeling:   emotion.Surprise,
		Intensity: 0,
	}
}

func TestNewLoop(t *testing.T) {
	t.Parallel()

	runner := NewLoop(New(testConfig()), testConfig())

	if runner == nil {
		t.Fatal("NewLoop() = nil, want a loop")
	}

	if got := runner.Ticks(); got != 0 {
		t.Fatalf("Ticks() = %d, want a fresh loop to have taken none", got)
	}
}

func TestRegisterLifecycleRunsAndRests(t *testing.T) {
	t.Parallel()

	built := New(testConfig())
	runner := NewLoop(built, testConfig())
	lifecycle := fxtest.NewLifecycle(t)

	RegisterLifecycle(lifecycle, runner)
	lifecycle.RequireStart()

	// The loop is running on its own, without anything having asked it to.
	deadline := time.After(2 * time.Second)

	for runner.Ticks() == 0 {
		select {
		case <-deadline:
			lifecycle.RequireStop()
			t.Fatal("the persona never stirred, want it running on its own")
		default:
		}
	}

	lifecycle.RequireStop()

	// Stopping must actually stop it.
	settled := runner.Ticks()

	time.Sleep(20 * time.Millisecond)

	if runner.Ticks() > settled+1 {
		t.Fatalf("Ticks() went %d -> %d, want the persona to have rested", settled, runner.Ticks())
	}
}

func TestModuleWiresTogether(t *testing.T) {
	t.Parallel()

	// The persona and its loop must be constructible from configuration alone,
	// so the fx graph in internal/app can resolve them.
	app := fxtest.New(t,
		fx.Supply(testConfig()),
		fx.Provide(New, NewLoop),
		fx.Invoke(RegisterLifecycle),
	)

	app.RequireStart()
	app.RequireStop()
}

func TestRegisterLifecycleStopIsIdempotentWithContext(t *testing.T) {
	t.Parallel()

	runner := NewLoop(New(testConfig()), testConfig())
	lifecycle := fxtest.NewLifecycle(t)

	RegisterLifecycle(lifecycle, runner)

	err := lifecycle.Start(context.Background())
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	err = lifecycle.Stop(context.Background())
	if err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
}
