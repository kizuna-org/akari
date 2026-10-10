package lifecycle

import (
	"time"

	"github.com/kizuna-org/akari/internal/action"
	"github.com/kizuna-org/akari/internal/continuity"
	"github.com/kizuna-org/akari/internal/emotion"
	"github.com/kizuna-org/akari/internal/mind"
	"github.com/kizuna-org/akari/internal/persona"
)

// FoundationConfig starts only state ownership. No Channel, emotion tick or external tool is started.
// Scaffold units remain inactive placeholders, not production psychological calibration.
func FoundationConfig() Config {
	const (
		contents          = 4096
		workspaceCapacity = 1024
		outboxCapacity    = 1024
		thoughtCapacity   = 16
		drainTimeout      = 2 * time.Second
	)

	return Config{
		State: continuity.Config{
			Persona: persona.Default(), WorkspaceCapacity: workspaceCapacity, OutboxCapacity: outboxCapacity,
			SaveTimeout: time.Second,
			Inner: mind.Settings{Contents: contents, Dynamics: emotion.Dynamics{
				EmotionHalfLife: time.Hour, MoodHalfLife: time.Hour,
			}, RestUnit: time.Hour, Clock: time.Now},
			Scope: func(mind.Snapshot) action.Scope {
				return action.Scope{AllowedDestinations: nil, ReadyRecipients: nil}
			},
		},
		ThoughtCapacity: thoughtCapacity, DrainTimeout: drainTimeout, FinalizeTimeout: time.Second,
	}
}
