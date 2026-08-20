package emotion

import (
	"math"
	"testing"
	"time"
)

const tolerance = 1e-9

func TestDefaultTraits(t *testing.T) {
	t.Parallel()

	want := Traits{
		Volatility:   DefaultVolatility,
		Expressivity: DefaultExpressivity,
		MoodInertia:  DefaultMoodInertia,
		Empathy:      DefaultEmpathy,
	}

	if got := DefaultTraits(); got != want {
		t.Fatalf("DefaultTraits() = %#v, want %#v", got, want)
	}
}

func TestNewStartsCalm(t *testing.T) {
	t.Parallel()

	state := New(DefaultTraits())

	if got := state.Mood(); got != 0 {
		t.Fatalf("Mood() = %v, want 0", got)
	}

	for kind := range kindCount {
		if got := state.Intensity(Kind(kind)); got != 0 {
			t.Fatalf("Intensity(%v) = %v, want 0", Kind(kind), got)
		}
	}
}

func TestStateFeel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		volatility    float64
		kind          Kind
		intensity     float64
		wantIntensity float64
		wantMood      float64
	}{
		{
			name:          "pleasant emotion lifts mood",
			volatility:    DefaultVolatility,
			kind:          Joy,
			intensity:     0.5,
			wantIntensity: 0.5,
			wantMood:      0.5 * moodContribution,
		},
		{
			name:          "unpleasant emotion lowers mood",
			volatility:    DefaultVolatility,
			kind:          Sadness,
			intensity:     0.5,
			wantIntensity: 0.5,
			wantMood:      -0.5 * moodContribution,
		},
		{
			name:          "surprise leaves mood alone",
			volatility:    DefaultVolatility,
			kind:          Surprise,
			intensity:     0.5,
			wantIntensity: 0.5,
			wantMood:      0,
		},
		{
			name:          "volatile persona feels the same event harder",
			volatility:    2,
			kind:          Joy,
			intensity:     0.25,
			wantIntensity: 0.5,
			wantMood:      0.5 * moodContribution,
		},
		{
			name:          "flat persona feels nothing",
			volatility:    0,
			kind:          Joy,
			intensity:     0.5,
			wantIntensity: 0,
			wantMood:      0,
		},
		{
			name:          "intensity is capped at one",
			volatility:    DefaultVolatility,
			kind:          Joy,
			intensity:     5,
			wantIntensity: maxIntensity,
			wantMood:      maxMood,
		},
		{
			name:          "zero intensity is ignored",
			volatility:    DefaultVolatility,
			kind:          Joy,
			intensity:     0,
			wantIntensity: 0,
			wantMood:      0,
		},
		{
			name:          "negative intensity is ignored",
			volatility:    DefaultVolatility,
			kind:          Joy,
			intensity:     -1,
			wantIntensity: 0,
			wantMood:      0,
		},
		{
			name:          "unknown kind is ignored",
			volatility:    DefaultVolatility,
			kind:          Kind(-1),
			intensity:     0.5,
			wantIntensity: 0,
			wantMood:      0,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			traits := DefaultTraits()
			traits.Volatility = testCase.volatility
			state := New(traits)

			state.Feel(testCase.kind, testCase.intensity)

			if got := state.Intensity(testCase.kind); !closeTo(got, testCase.wantIntensity) {
				t.Fatalf("Intensity() = %v, want %v", got, testCase.wantIntensity)
			}

			if got := state.Mood(); !closeTo(got, testCase.wantMood) {
				t.Fatalf("Mood() = %v, want %v", got, testCase.wantMood)
			}
		})
	}
}

func TestStateFeelAccumulates(t *testing.T) {
	t.Parallel()

	state := New(DefaultTraits())
	state.Feel(Boredom, 0.3)
	state.Feel(Boredom, 0.3)

	if got := state.Intensity(Boredom); !closeTo(got, 0.6) {
		t.Fatalf("Intensity() = %v, want 0.6", got)
	}
}

func TestStateFeelMoodFloor(t *testing.T) {
	t.Parallel()

	state := New(DefaultTraits())
	for range 10 {
		state.Feel(Sadness, maxIntensity)
	}

	if got := state.Mood(); !closeTo(got, minMood) {
		t.Fatalf("Mood() = %v, want %v", got, minMood)
	}
}

func TestStateEmpathize(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		empathy float64
		want    float64
	}{
		{name: "default empathy halves the feeling", empathy: DefaultEmpathy, want: 0.4},
		{name: "high empathy takes it on fully", empathy: 1, want: 0.8},
		{name: "no empathy stays unmoved", empathy: 0, want: 0},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			traits := DefaultTraits()
			traits.Empathy = testCase.empathy
			state := New(traits)

			state.Empathize(Joy, 0.8)

			if got := state.Intensity(Joy); !closeTo(got, testCase.want) {
				t.Fatalf("Intensity() = %v, want %v", got, testCase.want)
			}
		})
	}
}

func TestStateDecay(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		elapsed time.Duration
		want    float64
	}{
		{name: "no time passing changes nothing", elapsed: 0, want: 0.8},
		{name: "negative elapsed changes nothing", elapsed: -time.Minute, want: 0.8},
		{name: "one half-life halves the feeling", elapsed: emotionHalfLife, want: 0.4},
		{name: "two half-lives quarter it", elapsed: 2 * emotionHalfLife, want: 0.2},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			state := New(DefaultTraits())
			state.Feel(Anger, 0.8)

			state.Decay(testCase.elapsed)

			if got := state.Intensity(Anger); !closeTo(got, testCase.want) {
				t.Fatalf("Intensity() = %v, want %v", got, testCase.want)
			}
		})
	}
}

func TestStateDecayMoodOutlastsEmotion(t *testing.T) {
	t.Parallel()

	state := New(DefaultTraits())
	state.Feel(Joy, maxIntensity)
	moodBefore := state.Mood()

	state.Decay(emotionHalfLife)

	if got := state.Intensity(Joy); got >= maxIntensity {
		t.Fatalf("Intensity() = %v, want it to have faded", got)
	}

	kept := state.Mood() / moodBefore
	if kept <= halving {
		t.Fatalf("mood kept %v of its charge, want more than an emotion's %v", kept, halving)
	}
}

func TestStateMoodInertia(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		inertia float64
	}{
		{name: "zero inertia falls back to the reference rate", inertia: 0},
		{name: "negative inertia falls back to the reference rate", inertia: -1},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			traits := DefaultTraits()
			traits.MoodInertia = testCase.inertia
			state := New(traits)

			if got := state.moodHalfLife(); got != moodHalfLife {
				t.Fatalf("moodHalfLife() = %v, want %v", got, moodHalfLife)
			}
		})
	}
}

func TestStateMoodInertiaLingers(t *testing.T) {
	t.Parallel()

	brooding := DefaultTraits()
	brooding.MoodInertia = 4
	brooder := New(brooding)
	brooder.Feel(Sadness, maxIntensity)

	resilient := New(DefaultTraits())
	resilient.Feel(Sadness, maxIntensity)

	brooder.Decay(moodHalfLife)
	resilient.Decay(moodHalfLife)

	if brooder.Mood() >= resilient.Mood() {
		t.Fatalf("brooding mood %v, want it lower than resilient %v", brooder.Mood(), resilient.Mood())
	}
}

func TestStateIntensityUnknownKind(t *testing.T) {
	t.Parallel()

	state := New(DefaultTraits())

	if got := state.Intensity(Kind(-1)); got != 0 {
		t.Fatalf("Intensity() = %v, want 0", got)
	}
}

func TestStateExpressed(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		expressivity float64
		kind         Kind
		want         float64
	}{
		{name: "open persona shows it all", expressivity: 1, kind: Anger, want: 0.8},
		{name: "reserved persona shows less", expressivity: 0.25, kind: Anger, want: 0.2},
		{name: "guarded persona shows nothing", expressivity: 0, kind: Anger, want: 0},
		{name: "expression is capped at one", expressivity: 4, kind: Anger, want: maxIntensity},
		{name: "unknown kind shows nothing", expressivity: 1, kind: Kind(-1), want: 0},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			traits := DefaultTraits()
			traits.Expressivity = testCase.expressivity
			state := New(traits)
			state.Feel(Anger, 0.8)

			if got := state.Expressed(testCase.kind); !closeTo(got, testCase.want) {
				t.Fatalf("Expressed() = %v, want %v", got, testCase.want)
			}
		})
	}
}

func TestStateExpressedKeepsFeelingIntact(t *testing.T) {
	t.Parallel()

	traits := DefaultTraits()
	traits.Expressivity = 0
	state := New(traits)
	state.Feel(Anger, 0.8)

	if got := state.Expressed(Anger); got != 0 {
		t.Fatalf("Expressed() = %v, want 0", got)
	}

	if got := state.Intensity(Anger); !closeTo(got, 0.8) {
		t.Fatalf("Intensity() = %v, want the feeling to remain at 0.8", got)
	}
}

func TestStateDominant(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		feelings      map[Kind]float64
		wantKind      Kind
		wantIntensity float64
	}{
		{
			name:          "calm reports nothing felt",
			feelings:      nil,
			wantKind:      Joy,
			wantIntensity: 0,
		},
		{
			name:          "single feeling dominates",
			feelings:      map[Kind]float64{Sadness: 0.4},
			wantKind:      Sadness,
			wantIntensity: 0.4,
		},
		{
			name:          "strongest of several wins",
			feelings:      map[Kind]float64{Joy: 0.3, Anxiety: 0.7, Boredom: 0.5},
			wantKind:      Anxiety,
			wantIntensity: 0.7,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			state := New(DefaultTraits())
			for kind, intensity := range testCase.feelings {
				state.Feel(kind, intensity)
			}

			gotKind, gotIntensity := state.Dominant()
			if gotKind != testCase.wantKind {
				t.Fatalf("Dominant() kind = %v, want %v", gotKind, testCase.wantKind)
			}

			if !closeTo(gotIntensity, testCase.wantIntensity) {
				t.Fatalf("Dominant() intensity = %v, want %v", gotIntensity, testCase.wantIntensity)
			}
		})
	}
}

func TestStateReflex(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		feelings map[Kind]float64
		want     Tendency
		wantOK   bool
	}{
		{
			name:     "calm produces no reflex",
			feelings: nil,
			want:     TendencyNone,
			wantOK:   false,
		},
		{
			name:     "strong surprise freezes",
			feelings: map[Kind]float64{Surprise: 0.9},
			want:     TendencyFreeze,
			wantOK:   true,
		},
		{
			name:     "strong anxiety avoids",
			feelings: map[Kind]float64{Anxiety: 0.9},
			want:     TendencyAvoid,
			wantOK:   true,
		},
		{
			name:     "mild surprise stays below the threshold",
			feelings: map[Kind]float64{Surprise: reflexThreshold},
			want:     TendencyNone,
			wantOK:   false,
		},
		{
			name:     "strong joy has no reflex to give",
			feelings: map[Kind]float64{Joy: maxIntensity},
			want:     TendencyNone,
			wantOK:   false,
		},
		{
			name:     "strong anger has no reflex to give",
			feelings: map[Kind]float64{Anger: maxIntensity},
			want:     TendencyNone,
			wantOK:   false,
		},
		{
			name:     "the stronger reflex wins",
			feelings: map[Kind]float64{Surprise: 0.7, Anxiety: 0.95},
			want:     TendencyAvoid,
			wantOK:   true,
		},
		{
			name:     "reflex outranks a louder non-reflexive feeling",
			feelings: map[Kind]float64{Joy: maxIntensity, Surprise: 0.7},
			want:     TendencyFreeze,
			wantOK:   true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			state := New(DefaultTraits())
			for kind, intensity := range testCase.feelings {
				state.Feel(kind, intensity)
			}

			got, gotOK := state.Reflex()
			if got != testCase.want {
				t.Fatalf("Reflex() = %v, want %v", got, testCase.want)
			}

			if gotOK != testCase.wantOK {
				t.Fatalf("Reflex() ok = %v, want %v", gotOK, testCase.wantOK)
			}
		})
	}
}

func TestStateInterestBias(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		kind      Kind
		intensity float64
		compare   func(bias float64) bool
		wantDesc  string
	}{
		{
			name:      "calm is neutral",
			kind:      Surprise,
			intensity: 0,
			compare:   func(bias float64) bool { return closeTo(bias, neutralBias) },
			wantDesc:  "neutral",
		},
		{
			name:      "good mood lifts interest",
			kind:      Joy,
			intensity: maxIntensity,
			compare:   func(bias float64) bool { return bias > neutralBias },
			wantDesc:  "above neutral",
		},
		{
			name:      "low mood dampens interest",
			kind:      Sadness,
			intensity: maxIntensity,
			compare:   func(bias float64) bool { return bias < neutralBias },
			wantDesc:  "below neutral",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			state := New(DefaultTraits())
			state.Feel(testCase.kind, testCase.intensity)

			if got := state.InterestBias(); !testCase.compare(got) {
				t.Fatalf("InterestBias() = %v, want %s", got, testCase.wantDesc)
			}
		})
	}
}

func TestStateNoveltyBias(t *testing.T) {
	t.Parallel()

	calm := New(DefaultTraits())
	if got := calm.NoveltyBias(); !closeTo(got, neutralBias) {
		t.Fatalf("NoveltyBias() = %v, want %v", got, neutralBias)
	}

	bored := New(DefaultTraits())
	bored.Feel(Boredom, maxIntensity)

	if got := bored.NoveltyBias(); got <= neutralBias {
		t.Fatalf("NoveltyBias() = %v, want boredom to sharpen the appetite for novelty", got)
	}
}

func TestStateCautionBias(t *testing.T) {
	t.Parallel()

	calm := New(DefaultTraits())
	if got := calm.CautionBias(); !closeTo(got, neutralBias) {
		t.Fatalf("CautionBias() = %v, want %v", got, neutralBias)
	}

	anxious := New(DefaultTraits())
	anxious.Feel(Anxiety, maxIntensity)

	if got := anxious.CautionBias(); got <= neutralBias {
		t.Fatalf("CautionBias() = %v, want anxiety to sharpen caution", got)
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
		{name: "zero half-life leaves nothing", elapsed: time.Minute, halfLife: 0, want: 0},
		{name: "negative half-life leaves nothing", elapsed: time.Minute, halfLife: -time.Minute, want: 0},
		{name: "no time keeps everything", elapsed: 0, halfLife: time.Minute, want: 1},
		{name: "one half-life keeps half", elapsed: time.Minute, halfLife: time.Minute, want: halving},
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
		{name: "above the range is lowered", value: 2, want: maxIntensity},
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

func TestClampSigned(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value float64
		want  float64
	}{
		{name: "inside the range is untouched", value: 0.5, want: 0.5},
		{name: "below the range is raised", value: -2, want: minMood},
		{name: "above the range is lowered", value: 2, want: maxMood},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := clampSigned(testCase.value); got != testCase.want {
				t.Fatalf("clampSigned() = %v, want %v", got, testCase.want)
			}
		})
	}
}

func closeTo(got, want float64) bool {
	return math.Abs(got-want) < tolerance
}
