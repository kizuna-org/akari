package usecase

import (
	"math"
	"testing"
	"time"

	"github.com/kizuna-org/akari/kiseki/pkg/config"
)

func TestScorer_CalculateTimeScore(t *testing.T) {
	scorer := NewScorer(config.ScoreConfig{
		Alpha:   0.5,
		Beta:    0.3,
		Gamma:   0.2,
		Epsilon: 0.1,
	})

	tests := []struct {
		name             string
		lastAccessTime   time.Time
		wantScoreInRange bool
		minScore         float64
		maxScore         float64
	}{
		{
			name:             "very recent access (1 minute ago)",
			lastAccessTime:   time.Now().Add(-1 * time.Minute),
			wantScoreInRange: true,
			minScore:         0.8,
			maxScore:         1.0,
		},
		{
			name:             "recent access (1 hour ago)",
			lastAccessTime:   time.Now().Add(-1 * time.Hour),
			wantScoreInRange: true,
			minScore:         0.5,
			maxScore:         1.0,
		},
		{
			name:             "old access (24 hours ago)",
			lastAccessTime:   time.Now().Add(-24 * time.Hour),
			wantScoreInRange: true,
			minScore:         0.0,
			maxScore:         0.7,
		},
		{
			name:             "very old access (7 days ago)",
			lastAccessTime:   time.Now().Add(-7 * 24 * time.Hour),
			wantScoreInRange: true,
			minScore:         0.0,
			maxScore:         0.5,
		},
		{
			name:             "never accessed",
			lastAccessTime:   time.Time{},
			wantScoreInRange: true,
			minScore:         0.0,
			maxScore:         0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := scorer.CalculateTimeScore(tt.lastAccessTime)
			if tt.wantScoreInRange {
				if score < tt.minScore || score > tt.maxScore {
					t.Errorf("CalculateTimeScore() = %v, want in range [%v, %v]", score, tt.minScore, tt.maxScore)
				}
			}
			// Ensure score is in valid range [0, 1]
			if score < 0 || score > 1 {
				t.Errorf("CalculateTimeScore() = %v, want in range [0, 1]", score)
			}
		})
	}
}

func TestScorer_CalculatePopularityScore(t *testing.T) {
	scorer := NewScorer(config.ScoreConfig{
		Alpha:   0.5,
		Beta:    0.3,
		Gamma:   0.2,
		Epsilon: 0.1,
	})

	tests := []struct {
		name        string
		accessCount int64
		wantMin     float64
		wantMax     float64
	}{
		{
			name:        "no access",
			accessCount: 0,
			wantMin:     0.0,
			wantMax:     0.0,
		},
		{
			name:        "one access",
			accessCount: 1,
			wantMin:     0.09,
			wantMax:     0.11,
		},
		{
			name:        "ten accesses",
			accessCount: 10,
			wantMin:     0.6,
			wantMax:     0.7,
		},
		{
			name:        "many accesses",
			accessCount: 100,
			wantMin:     0.99,
			wantMax:     1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := scorer.CalculatePopularityScore(tt.accessCount)
			if score < tt.wantMin || score > tt.wantMax {
				t.Errorf("CalculatePopularityScore() = %v, want in range [%v, %v]", score, tt.wantMin, tt.wantMax)
			}
			// Ensure score is in valid range [0, 1]
			if score < 0 || score > 1 {
				t.Errorf("CalculatePopularityScore() = %v, want in range [0, 1]", score)
			}
		})
	}
}

func TestScorer_CalculateFinalScore(t *testing.T) {
	scorer := NewScorer(config.ScoreConfig{
		Alpha:   0.5,
		Beta:    0.3,
		Gamma:   0.2,
		Epsilon: 0.1,
	})

	tests := []struct {
		name            string
		semanticScore   float64
		popularityScore float64
		timeScore       float64
		want            float64
	}{
		{
			name:            "all scores are 1.0",
			semanticScore:   1.0,
			popularityScore: 1.0,
			timeScore:       1.0,
			want:            1.0,
		},
		{
			name:            "all scores are 0.0",
			semanticScore:   0.0,
			popularityScore: 0.0,
			timeScore:       0.0,
			want:            0.0,
		},
		{
			name:            "mixed scores",
			semanticScore:   0.8,
			popularityScore: 0.5,
			timeScore:       0.3,
			want:            0.5*0.8 + 0.3*0.5 + 0.2*0.3, // 0.61
		},
		{
			name:            "high semantic, low others",
			semanticScore:   0.9,
			popularityScore: 0.1,
			timeScore:       0.1,
			want:            0.5*0.9 + 0.3*0.1 + 0.2*0.1, // 0.5
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := scorer.CalculateFinalScore(tt.semanticScore, tt.popularityScore, tt.timeScore)
			if math.Abs(got-tt.want) > 0.001 {
				t.Errorf("CalculateFinalScore() = %v, want %v", got, tt.want)
			}
			// Ensure score is in valid range [0, 1]
			if got < 0 || got > 1 {
				t.Errorf("CalculateFinalScore() = %v, want in range [0, 1]", got)
			}
		})
	}
}
