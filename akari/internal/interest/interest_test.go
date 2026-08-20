package interest

import (
	"math"
	"testing"
	"time"
)

const (
	tolerance = 1e-9
	topicID   = "music"
	otherID   = "cooking"
)

// reference is a fixed instant so tests never depend on the wall clock.
func reference() time.Time {
	return time.Date(2026, time.August, 20, 9, 0, 0, 0, time.UTC)
}

func TestDefaultTuning(t *testing.T) {
	t.Parallel()

	want := Tuning{
		Curiosity:        DefaultCuriosity,
		Habituation:      DefaultHabituation,
		AffinityLearning: DefaultAffinityLearning,
	}

	if got := DefaultTuning(); got != want {
		t.Fatalf("DefaultTuning() = %#v, want %#v", got, want)
	}
}

func TestNeutralBias(t *testing.T) {
	t.Parallel()

	want := Bias{Overall: neutralBias, Novelty: neutralBias}

	if got := NeutralBias(); got != want {
		t.Fatalf("NeutralBias() = %#v, want %#v", got, want)
	}
}

func TestNewIsEmpty(t *testing.T) {
	t.Parallel()

	if got := New(DefaultTuning()).Len(); got != 0 {
		t.Fatalf("Len() = %d, want 0", got)
	}
}

func TestScoreUnseenTopicIsNovel(t *testing.T) {
	t.Parallel()

	table := New(DefaultTuning())

	got := table.Score(topicID, NeutralBias(), reference())
	if got <= minScore {
		t.Fatalf("Score() = %v, want an unseen topic to draw interest on novelty alone", got)
	}

	if table.Len() != 0 {
		t.Fatalf("Len() = %d, want reading a score not to record the topic", table.Len())
	}
}

func TestScoreRespondsToCuriosity(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		curiosity float64
	}{
		{name: "incurious persona", curiosity: 0},
		{name: "default persona", curiosity: DefaultCuriosity},
		{name: "very curious persona", curiosity: 2},
	}

	scores := make([]float64, 0, len(tests))

	for _, testCase := range tests {
		tuning := DefaultTuning()
		tuning.Curiosity = testCase.curiosity
		table := New(tuning)
		scores = append(scores, table.Score(topicID, NeutralBias(), reference()))
	}

	for i := 1; i < len(scores); i++ {
		if scores[i] <= scores[i-1] {
			t.Fatalf("scores = %v, want novelty to pull harder as curiosity rises", scores)
		}
	}
}

func TestEngageRecordsTopic(t *testing.T) {
	t.Parallel()

	table := New(DefaultTuning())
	table.Engage(topicID, Result{Enjoyment: 1, Progress: 1}, reference())

	if got := table.Len(); got != 1 {
		t.Fatalf("Len() = %d, want 1", got)
	}
}

func TestEngageEnjoymentMovesLiking(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		enjoyment float64
		compare   func(after, before float64) bool
		wantDesc  string
	}{
		{
			name:      "a good time raises interest",
			enjoyment: 1,
			compare:   func(after, before float64) bool { return after > before },
			wantDesc:  "higher than before",
		},
		{
			name:      "a bad time lowers interest",
			enjoyment: -1,
			compare:   func(after, before float64) bool { return after < before },
			wantDesc:  "lower than before",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			now := reference()
			table := New(DefaultTuning())

			// Settle at a middling liking so movement is visible in both directions.
			table.Engage(topicID, Result{Enjoyment: 1, Progress: 0}, now)
			before := table.Score(topicID, NeutralBias(), now)

			table.Engage(topicID, Result{Enjoyment: testCase.enjoyment, Progress: 0}, now)
			after := table.Score(topicID, NeutralBias(), now)

			if !testCase.compare(after, before) {
				t.Fatalf("score went %v -> %v, want %s", before, after, testCase.wantDesc)
			}
		})
	}
}

func TestEngageHabituatesNovelty(t *testing.T) {
	t.Parallel()

	now := reference()
	table := New(DefaultTuning())

	// Engage with no enjoyment and no progress, so only novelty can move.
	first := table.Score(topicID, NeutralBias(), now)
	table.Engage(topicID, Result{Enjoyment: 0, Progress: 0}, now)
	second := table.Score(topicID, NeutralBias(), now)
	table.Engage(topicID, Result{Enjoyment: 0, Progress: 0}, now)
	third := table.Score(topicID, NeutralBias(), now)

	if !(first > second && second > third) {
		t.Fatalf("scores = %v, %v, %v, want familiarity to wear novelty down", first, second, third)
	}
}

func TestEngageFruitlessTopicLosesItsGrip(t *testing.T) {
	t.Parallel()

	now := reference()
	table := New(DefaultTuning())

	// A topic that keeps being unpredictable but never teaches anything: the
	// noisy-TV case the design guards against (docs/04-interest.md 4.3).
	scores := make([]float64, 0, 5)

	for range 5 {
		table.Engage(topicID, Result{Enjoyment: 0, Progress: 0}, now)
		scores = append(scores, table.Score(topicID, NeutralBias(), now))
	}

	if scores[len(scores)-1] >= scores[0] {
		t.Fatalf("scores = %v, want a fruitless topic to lose its grip", scores)
	}
}

func TestEngageProgressKeepsInterestAlive(t *testing.T) {
	t.Parallel()

	now := reference()
	fruitful := New(DefaultTuning())
	fruitless := New(DefaultTuning())

	for range 3 {
		fruitful.Engage(topicID, Result{Enjoyment: 0, Progress: 1}, now)
		fruitless.Engage(topicID, Result{Enjoyment: 0, Progress: 0}, now)
	}

	gotFruitful := fruitful.Score(topicID, NeutralBias(), now)
	gotFruitless := fruitless.Score(topicID, NeutralBias(), now)

	if gotFruitful <= gotFruitless {
		t.Fatalf("fruitful = %v, fruitless = %v, want getting somewhere to hold interest",
			gotFruitful, gotFruitless)
	}
}

func TestEngageClampsResult(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		result Result
	}{
		{name: "enjoyment far above range", result: Result{Enjoyment: 99, Progress: 0}},
		{name: "enjoyment far below range", result: Result{Enjoyment: -99, Progress: 0}},
		{name: "progress far above range", result: Result{Enjoyment: 0, Progress: 99}},
		{name: "progress far below range", result: Result{Enjoyment: 0, Progress: -99}},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			now := reference()
			table := New(DefaultTuning())
			table.Engage(topicID, testCase.result, now)

			got := table.Score(topicID, NeutralBias(), now)
			if got < minScore || got > maxScore {
				t.Fatalf("Score() = %v, want it within [%v, %v]", got, minScore, maxScore)
			}
		})
	}
}

func TestEngageHabituationIsClamped(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		habituation float64
	}{
		{name: "above one", habituation: 5},
		{name: "below zero", habituation: -5},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			now := reference()
			tuning := DefaultTuning()
			tuning.Habituation = testCase.habituation
			table := New(tuning)

			table.Engage(topicID, Result{Enjoyment: 0, Progress: 0}, now)

			got := table.Score(topicID, NeutralBias(), now)
			if got < minScore || got > maxScore {
				t.Fatalf("Score() = %v, want it within [%v, %v]", got, minScore, maxScore)
			}
		})
	}
}

func TestNeverHabituatingPersonaKeepsItsNovelty(t *testing.T) {
	t.Parallel()

	now := reference()
	tuning := DefaultTuning()
	tuning.Habituation = 0
	table := New(tuning)

	before := table.Score(topicID, NeutralBias(), now)
	table.Engage(topicID, Result{Enjoyment: 0, Progress: 0}, now)
	after := table.Score(topicID, NeutralBias(), now)

	if !closeTo(before, after) {
		t.Fatalf("score went %v -> %v, want a persona that never tires to hold steady", before, after)
	}
}

func TestScoreFadesWhileUntouched(t *testing.T) {
	t.Parallel()

	now := reference()
	table := New(DefaultTuning())
	table.Engage(topicID, Result{Enjoyment: 1, Progress: 1}, now)

	immediate := table.Score(topicID, NeutralBias(), now)
	later := table.Score(topicID, NeutralBias(), now.Add(progressHalfLife*4))

	if later >= immediate {
		t.Fatalf("score went %v -> %v, want momentum to fade once attention moves on", immediate, later)
	}
}

func TestScoreLikingOutlastsMomentum(t *testing.T) {
	t.Parallel()

	now := reference()
	table := New(DefaultTuning())
	table.Engage(topicID, Result{Enjoyment: 1, Progress: 1}, now)

	afterHours := table.Score(topicID, NeutralBias(), now.Add(progressHalfLife*4))
	afterMonths := table.Score(topicID, NeutralBias(), now.Add(affinityHalfLife*4))

	if afterMonths >= afterHours {
		t.Fatalf("score went %v -> %v, want liking to keep fading over months", afterHours, afterMonths)
	}
}

func TestScoreNoveltyReturnsWithAbsence(t *testing.T) {
	t.Parallel()

	now := reference()
	tuning := DefaultTuning()
	tuning.AffinityLearning = 0
	table := New(tuning)

	// Wear the novelty down, then stay away for a long time.
	for range 4 {
		table.Engage(topicID, Result{Enjoyment: 0, Progress: 0}, now)
	}

	worn := table.Score(topicID, NeutralBias(), now)
	rested := table.Score(topicID, NeutralBias(), now.Add(noveltyRecoveryHalfLife*4))

	if rested <= worn {
		t.Fatalf("score went %v -> %v, want a long absence to make it feel fresh again", worn, rested)
	}
}

func TestScoreIgnoresTimeGoingBackwards(t *testing.T) {
	t.Parallel()

	now := reference()
	table := New(DefaultTuning())
	table.Engage(topicID, Result{Enjoyment: 1, Progress: 1}, now)

	atNow := table.Score(topicID, NeutralBias(), now)
	inPast := table.Score(topicID, NeutralBias(), now.Add(-time.Hour))

	if !closeTo(atNow, inPast) {
		t.Fatalf("score at now = %v, in the past = %v, want them equal", atNow, inPast)
	}
}

func TestScoreAppliesBias(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		bias     Bias
		compare  func(biased, neutral float64) bool
		wantDesc string
	}{
		{
			name:     "a good mood lifts everything",
			bias:     Bias{Overall: 1.4, Novelty: neutralBias},
			compare:  func(biased, neutral float64) bool { return biased > neutral },
			wantDesc: "above the neutral score",
		},
		{
			name:     "a low mood dampens everything",
			bias:     Bias{Overall: 0.6, Novelty: neutralBias},
			compare:  func(biased, neutral float64) bool { return biased < neutral },
			wantDesc: "below the neutral score",
		},
		{
			name:     "boredom sharpens the pull of the unfamiliar",
			bias:     Bias{Overall: neutralBias, Novelty: 1.5},
			compare:  func(biased, neutral float64) bool { return biased > neutral },
			wantDesc: "above the neutral score",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			now := reference()
			table := New(DefaultTuning())
			table.Engage(topicID, Result{Enjoyment: 0.5, Progress: 0.5}, now)

			neutral := table.Score(topicID, NeutralBias(), now)
			biased := table.Score(topicID, testCase.bias, now)

			if !testCase.compare(biased, neutral) {
				t.Fatalf("biased = %v, neutral = %v, want %s", biased, neutral, testCase.wantDesc)
			}
		})
	}
}

func TestPrune(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		threshold   float64
		elapsed     time.Duration
		wantDropped int
		wantLen     int
	}{
		{
			name:        "keeps everything below a zero threshold",
			threshold:   0,
			elapsed:     0,
			wantDropped: 0,
			wantLen:     2,
		},
		{
			name:        "drops everything below an impossible threshold",
			threshold:   maxScore + 1,
			elapsed:     0,
			wantDropped: 2,
			wantLen:     0,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			now := reference()
			table := New(DefaultTuning())
			table.Engage(topicID, Result{Enjoyment: 1, Progress: 1}, now)
			table.Engage(otherID, Result{Enjoyment: 1, Progress: 1}, now)

			got := table.Prune(testCase.threshold, now.Add(testCase.elapsed))
			if got != testCase.wantDropped {
				t.Fatalf("Prune() = %d, want %d", got, testCase.wantDropped)
			}

			if table.Len() != testCase.wantLen {
				t.Fatalf("Len() = %d, want %d", table.Len(), testCase.wantLen)
			}
		})
	}
}

func TestPruneKeepsWhatIsStillCaredAbout(t *testing.T) {
	t.Parallel()

	now := reference()
	tuning := DefaultTuning()
	tuning.Curiosity = 0
	table := New(tuning)

	table.Engage(topicID, Result{Enjoyment: 1, Progress: 1}, now)
	table.Engage(otherID, Result{Enjoyment: -1, Progress: 0}, now)

	if got := table.Prune(0.1, now); got != 1 {
		t.Fatalf("Prune() = %d, want 1", got)
	}

	if got := table.Score(topicID, NeutralBias(), now); got <= 0 {
		t.Fatalf("Score(%q) = %v, want the liked topic to survive", topicID, got)
	}
}

func TestDecayFactor(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		elapsed  time.Duration
		halfLife time.Duration
		want     float64
	}{
		{name: "zero half-life leaves nothing", elapsed: time.Hour, halfLife: 0, want: 0},
		{name: "negative half-life leaves nothing", elapsed: time.Hour, halfLife: -time.Hour, want: 0},
		{name: "no time keeps everything", elapsed: 0, halfLife: time.Hour, want: 1},
		{name: "one half-life keeps half", elapsed: time.Hour, halfLife: time.Hour, want: halving},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := decayFactor(testCase.elapsed, testCase.halfLife); !closeTo(got, testCase.want) {
				t.Fatalf("decayFactor() = %v, want %v", got, testCase.want)
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

func closeTo(got, want float64) bool {
	return math.Abs(got-want) < tolerance
}
