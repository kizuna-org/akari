package persona

import (
	"errors"
	"math"
	"testing"
)

const changedValue = "changed"

func TestFixedProfile(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name   string
		change func(*Config)
		want   error
	}{
		{name: "confirmed default", change: func(*Config) {}, want: nil},
		{name: "missing identity", change: func(config *Config) { config.ID = "" }, want: ErrConfig},
		{name: "unknown schema", change: func(config *Config) { config.Schema = 2 }, want: ErrConfig},
		{name: "NaN", change: func(config *Config) { config.Vector[Reactivity] = math.NaN() }, want: ErrConfig},
		{name: "negative", change: func(config *Config) { config.Vector[Reactivity] = -1 }, want: ErrConfig},
		{name: "zero count", change: func(config *Config) { config.Vector[IntentionCount] = 0 }, want: ErrConfig},
		{name: "fractional count", change: func(config *Config) { config.Vector[AwarenessCount] = 1.5 }, want: ErrConfig},
		{name: "weight budget", change: func(config *Config) { config.Vector[WillWeight] = 1 }, want: ErrConfig},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			config := Default()
			clone := config.Clone()
			clone.Likes[0] = changedValue
			clone.Dislikes[0] = changedValue
			clone.EmotionBiases[0] = changedValue
			test.change(&clone)

			if !errors.Is(clone.Validate(), test.want) || config.Likes[0] == changedValue || config.Dislikes[0] == changedValue {
				t.Fatal("profile is invalid or aliases startup settings")
			}

			if config.Vector[Persistence] != 0.6 || config.Vector[GivingUp] != 0.4 ||
				config.Vector[IntentionCount] != 2 || config.Vector[AwarenessCount] != 1 {
				t.Fatal("confirmed independent axes changed")
			}
		})
	}
}
