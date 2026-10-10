// Package persona holds startup configuration outside Akari's subjective context.
package persona

import (
	"math"
	"slices"

	"github.com/kizuna-org/akari/internal/meaning"
)

type fault string

func (err fault) Error() string { return string(err) }

const ErrConfig fault = "persona must use schema 1 and valid continuous values or positive integral counts"

// Axis order is schema-versioned; these are the existing axes in docs/12-persona.md.
type Axis int

const (
	Reactivity Axis = iota
	Expression
	Lingering
	Baseline
	Empathy
	NegativeExpression
	Forgetfulness
	RecallVariability
	Ambiguity
	SemanticWeight
	PopularityWeight
	RecencyWeight
	EmotionWeight
	WillWeight
	Curiosity
	Fickleness
	CoreFixity
	Persistence
	GivingUp
	GoalBalance
	IntentionCount
	Switching
	Habituation
	Speculation
	PoolPersistence
	AwarenessCount
	Initiative
	Fatigability
	Recovery
	Reserve
	Verbosity
	Interruption
	Anticipation
	SocialVariation
	AxisCount
)

// Config is for configuration/administrative persistence, never a model request.
// Qualitative seeds await a chosen encoder; no synthetic semantic coordinates are assigned.
type Config struct {
	ID            string             `json:"id"`
	Schema        int                `json:"schema"`
	Vector        [AxisCount]float64 `json:"vector"`
	Likes         []string           `json:"likes"`
	Dislikes      []string           `json:"dislikes"`
	EmotionBiases []string           `json:"emotionBiases"`
}

//nolint:gosmopolitan // Preserve the confirmed Japanese seed descriptions verbatim.
func Default() Config {
	return Config{
		ID: "akari", Schema: 1,
		Vector: [AxisCount]float64{
			0.6, 0.7, 0.4, 0.6, 0.6, 0.4,
			0.5, 0.5, 0.5, 0.30, 0.15, 0.20, 0.25, 0.10,
			0.7, 0.5, 0.7, 0.6, 0.4, 0.55, 2,
			0.5, 0.5, 0.6, 0.5, 1, 0.5, 0.5, 0.6, 0.6, 0.6, 0.3, 0.5, 0.5,
		},
		Likes:         []string{"ことばと言い回し", "ものの仕組み", "生き物", "人の身の上話"},
		Dislikes:      []string{"急かされること", "曖昧なまま進めること"},
		EmotionBiases: []string{"退屈・驚きが出やすい", "怒りは出にくい"},
	}
}

func (config Config) Clone() Config {
	config.Likes = slices.Clone(config.Likes)
	config.Dislikes = slices.Clone(config.Dislikes)
	config.EmotionBiases = slices.Clone(config.EmotionBiases)

	return config
}

func (config Config) Validate() error {
	if config.ID == "" || config.Schema != 1 {
		return ErrConfig
	}

	for axis, value := range config.Vector {
		if !validCoordinate(Axis(axis), value) {
			return ErrConfig
		}
	}

	sum := 0.0

	for axis := SemanticWeight; axis <= WillWeight; axis++ {
		sum += config.Vector[axis]
	}

	const weightTolerance = 1e-9

	if math.Abs(sum-1) > weightTolerance {
		return ErrConfig
	}

	return nil
}

func validCoordinate(axis Axis, value float64) bool {
	if axis == IntentionCount || axis == AwarenessCount {
		return meaning.Finite(value) && value >= 1 && value <= math.MaxInt32 && math.Trunc(value) == value
	}

	return meaning.Unit(value)
}
