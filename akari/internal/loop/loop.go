// Package loop keeps a persona running.
//
// This is the always-on part of the design (docs/07-autonomy.md 7.1): the
// persona is not a request handler that exists only while being spoken to. It
// ticks along on its own, and being spoken to is one of the things that can
// happen to it rather than the reason it is running.
package loop

import (
	"context"
	"log/slog"
	"time"

	"github.com/kizuna-org/akari/internal/mind"
	"github.com/kizuna-org/akari/internal/workspace"
)

const (
	// DefaultInterval is how often the persona comes round to itself when
	// nothing is prodding it.
	DefaultInterval = 3 * time.Second
	// DefaultNightly is how often the day gets settled. Sleep is what turns a
	// day of moments into something compacted and partly forgotten.
	DefaultNightly = 24 * time.Hour
)

// Clock reports the time. It is an interface so a loop can be driven through a
// simulated day in a test without waiting for one.
type Clock interface {
	Now() time.Time
}

// SystemClock reads the real time.
type SystemClock struct{}

// Now returns the current time.
func (SystemClock) Now() time.Time { return time.Now() }

// Config is how often a persona stirs.
type Config struct {
	// Interval is the gap between ticks.
	Interval time.Duration
	// Nightly is the gap between sleeps.
	Nightly time.Duration
}

// DefaultConfig returns the reference pacing.
func DefaultConfig() Config {
	return Config{Interval: DefaultInterval, Nightly: DefaultNightly}
}

// Loop drives one persona.
type Loop struct {
	mind    *mind.Mind
	clock   Clock
	config  Config
	logger  *slog.Logger
	slept   time.Time
	ticks   int
	moments int
}

// New returns a loop for a persona. A nil clock reads the real time and a nil
// logger discards; both defaults keep construction simple at call sites.
func New(persona *mind.Mind, clock Clock, config Config, logger *slog.Logger) *Loop {
	if clock == nil {
		clock = SystemClock{}
	}

	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}

	if config.Interval <= 0 {
		config.Interval = DefaultInterval
	}

	if config.Nightly <= 0 {
		config.Nightly = DefaultNightly
	}

	return &Loop{
		mind:    persona,
		clock:   clock,
		config:  config,
		logger:  logger,
		slept:   time.Time{},
		ticks:   0,
		moments: 0,
	}
}

// Name reports which persona is running.
func (l *Loop) Name() string {
	return l.mind.Name()
}

// Ticks reports how many turns the loop has taken.
func (l *Loop) Ticks() int {
	return l.ticks
}

// Moments reports how many things the persona has been conscious of.
func (l *Loop) Moments() int {
	return l.moments
}

// Step takes one turn: the persona comes round, attends to whatever is most
// worth attending to, and settles the day if enough of one has passed.
//
// It returns whatever reached consciousness, which is empty most of the time.
// A persona with nothing on its mind does nothing, and that is not a problem to
// be fixed.
func (l *Loop) Step() []workspace.Moment {
	now := l.clock.Now()
	l.ticks++

	moments := l.mind.Tick(now)
	l.moments += len(moments)

	for _, moment := range moments {
		l.logger.Info("conscious",
			"persona", l.mind.Name(),
			"channel", moment.Winner.Channel,
			"content", moment.Winner.Content,
			"committed", moment.Committed,
			"withheld", moment.Withheld,
			"note", moment.Note,
		)
	}

	l.maybeSleep(now)

	return moments
}

// Run ticks until the context is done.
//
// Errors are not returned because there is nothing for a caller to do about
// them: a persona that stopped existing because one turn went badly would not be
// much of a persona. Anything worth knowing goes to the log and the loop carries
// on (docs/07-autonomy.md 7.1).
func (l *Loop) Run(ctx context.Context) {
	ticker := time.NewTicker(l.config.Interval)
	defer ticker.Stop()

	l.logger.Info("persona waking", "persona", l.mind.Name(), "interval", l.config.Interval)

	for {
		select {
		case <-ctx.Done():
			l.logger.Info("persona resting",
				"persona", l.mind.Name(),
				"ticks", l.ticks,
				"moments", l.moments,
			)

			return
		case <-ticker.C:
			l.Step()
		}
	}
}

// maybeSleep settles the day once enough of one has gone by.
func (l *Loop) maybeSleep(now time.Time) {
	if l.slept.IsZero() {
		l.slept = now

		return
	}

	if now.Sub(l.slept) < l.config.Nightly {
		return
	}

	forgotten, dropped := l.mind.Sleep(now)
	l.slept = now

	l.logger.Info("day settled",
		"persona", l.mind.Name(),
		"forgotten", forgotten,
		"dropped", dropped,
	)
}
