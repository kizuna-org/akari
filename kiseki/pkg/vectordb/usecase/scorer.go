package usecase

import (
	"math"
	"time"

	"github.com/kizuna-org/akari/kiseki/pkg/config"
	"github.com/kizuna-org/akari/kiseki/pkg/vectordb/domain/entity"
)

// Scorer calculates final scores for search results
type Scorer struct {
	config config.ScoreConfig
}

// NewScorer creates a new scorer
func NewScorer(config config.ScoreConfig) *Scorer {
	return &Scorer{
		config: config,
	}
}

// CalculateTimeScore calculates the time-based score using the forgetting curve
// Formula: S_time = 1.84 / ((log t)^1.25 + 1.84)
// where t is the elapsed time in hours since last access
func (s *Scorer) CalculateTimeScore(lastAccessTime time.Time) float64 {
	if lastAccessTime.IsZero() {
		// If never accessed, treat as very old (low score)
		return 0.0
	}

	elapsed := time.Since(lastAccessTime)
	hours := elapsed.Hours()

	// Prevent log(0) and log of negative numbers
	if hours < 0.01 {
		hours = 0.01 // Minimum 0.01 hours (about 36 seconds)
	}

	logT := math.Log(hours)
	score := 1.84 / (math.Pow(logT, 1.25) + 1.84)

	// Ensure score is between 0 and 1
	if score < 0 {
		score = 0
	}
	if score > 1 {
		score = 1
	}

	return score
}

// CalculatePopularityScore calculates the popularity score based on access count
// Formula: S_pop = 1 - exp(-ε * Count)
func (s *Scorer) CalculatePopularityScore(accessCount int64) float64 {
	score := 1.0 - math.Exp(-s.config.Epsilon*float64(accessCount))

	// Ensure score is between 0 and 1
	if score < 0 {
		score = 0
	}
	if score > 1 {
		score = 1
	}

	return score
}

// CalculateFinalScore calculates the final combined score
// Formula: S = α * S_vec + β * S_pop + γ * S_time
func (s *Scorer) CalculateFinalScore(semanticScore, popularityScore, timeScore float64) float64 {
	score := s.config.Alpha*semanticScore +
		s.config.Beta*popularityScore +
		s.config.Gamma*timeScore

	// Ensure score is between 0 and 1
	if score < 0 {
		score = 0
	}
	if score > 1 {
		score = 1
	}

	return score
}

// RescoreResults rescores search results with popularity and time scores
func (s *Scorer) RescoreResults(results []entity.SearchResult, accessInfoMap map[string]*entity.AccessInfo) []entity.SearchResult {
	for i := range results {
		result := &results[i]
		fragmentIDStr := result.Fragment.ID.String()

		// Get access info
		accessInfo := accessInfoMap[fragmentIDStr]
		if accessInfo == nil {
			// No access info, use defaults
			result.PopularityScore = 0.0
			result.TimeScore = 0.0
		} else {
			result.PopularityScore = s.CalculatePopularityScore(accessInfo.AccessCount)
			result.TimeScore = s.CalculateTimeScore(accessInfo.LastAccessedAt)
		}

		// Calculate final score
		result.Score = s.CalculateFinalScore(
			result.SemanticScore,
			result.PopularityScore,
			result.TimeScore,
		)
	}

	return results
}
