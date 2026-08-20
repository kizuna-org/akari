package memory

import (
	"errors"
	"strings"
	"testing"
	"time"
)

const (
	catContent   = "talked with Aoi about cats"
	musicContent = "listened to a new album"
	catCue       = "cats"
	aoi          = "aoi"
	rin          = "rin"
	missingID    = "frag-999"
)

// reference is a fixed instant so tests never depend on the wall clock.
func reference() time.Time {
	return time.Date(2026, time.August, 20, 9, 0, 0, 0, time.UTC)
}

func vividExperience() Experience {
	return Experience{Content: catContent, Feeling: 0.9, Will: 0.5, Confidential: false}
}

func faintExperience() Experience {
	return Experience{Content: musicContent, Feeling: 0, Will: 0, Confidential: false}
}

func plainCue(text string) Cue {
	return Cue{Text: text, Audience: "", IncludeConfidential: false}
}

func TestDefaultTuning(t *testing.T) {
	t.Parallel()

	want := Tuning{
		Forgetfulness: DefaultForgetfulness,
		Compaction:    DefaultCompaction,
		RecallLimit:   DefaultRecallLimit,
	}

	if got := DefaultTuning(); got != want {
		t.Fatalf("DefaultTuning() = %#v, want %#v", got, want)
	}
}

func TestLayerString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		layer Layer
		want  string
	}{
		{name: "input", layer: LayerInput, want: NameInput},
		{name: "context", layer: LayerContext, want: NameContext},
		{name: "working", layer: LayerWorking, want: NameWorking},
		{name: "day", layer: LayerDay, want: NameDay},
		{name: "long term", layer: LayerLongTerm, want: NameLongTerm},
		{name: "out of range", layer: Layer(-1), want: NameUnknown},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := testCase.layer.String(); got != testCase.want {
				t.Fatalf("Layer.String() = %q, want %q", got, testCase.want)
			}
		})
	}
}

func TestLayerNext(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		layer Layer
		want  Layer
	}{
		{name: "input settles into context", layer: LayerInput, want: LayerContext},
		{name: "day settles into long term", layer: LayerDay, want: LayerLongTerm},
		{name: "long term is the end of the road", layer: LayerLongTerm, want: LayerLongTerm},
		{name: "beyond the range stays put", layer: Layer(99), want: LayerLongTerm},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := testCase.layer.next(); got != testCase.want {
				t.Fatalf("next() = %v, want %v", got, testCase.want)
			}
		})
	}
}

func TestNewFallsBackToKeywordOverlap(t *testing.T) {
	t.Parallel()

	store := New(DefaultTuning(), nil)
	store.Perceive(vividExperience(), reference())

	if got := store.Recall(plainCue(catCue), reference()); len(got) != 1 {
		t.Fatalf("Recall() returned %d memories, want 1", len(got))
	}
}

func TestPerceiveArrivesInInput(t *testing.T) {
	t.Parallel()

	now := reference()
	store := New(DefaultTuning(), TokenOverlap{})
	fragmentID := store.Perceive(vividExperience(), now)

	fragment, known := store.Get(fragmentID)
	if !known {
		t.Fatal("Get() known = false, want true")
	}

	if fragment.Layer() != LayerInput {
		t.Fatalf("Layer() = %v, want %v", fragment.Layer(), LayerInput)
	}

	if fragment.Strength() != fullStrength {
		t.Fatalf("Strength() = %v, want %v", fragment.Strength(), fullStrength)
	}

	if fragment.Recalls() != 0 {
		t.Fatalf("Recalls() = %d, want 0", fragment.Recalls())
	}

	if !fragment.CreatedAt().Equal(now) {
		t.Fatalf("CreatedAt() = %v, want %v", fragment.CreatedAt(), now)
	}

	if store.Len() != 1 {
		t.Fatalf("Len() = %d, want 1", store.Len())
	}
}

func TestPerceiveClampsFeelingAndWill(t *testing.T) {
	t.Parallel()

	store := New(DefaultTuning(), TokenOverlap{})
	fragmentID := store.Perceive(Experience{
		Content:      catContent,
		Feeling:      99,
		Will:         -99,
		Confidential: false,
	}, reference())

	fragment, _ := store.Get(fragmentID)
	if got := store.retention(fragment, reference()); got < minScore || got > 2 {
		t.Fatalf("retention() = %v, want a sane figure from clamped inputs", got)
	}
}

func TestGetUnknown(t *testing.T) {
	t.Parallel()

	store := New(DefaultTuning(), TokenOverlap{})

	if _, known := store.Get(missingID); known {
		t.Fatal("Get() known = true, want false")
	}
}

func TestAttendDrawsIntoContext(t *testing.T) {
	t.Parallel()

	now := reference()
	store := New(DefaultTuning(), TokenOverlap{})
	fragmentID := store.Perceive(vividExperience(), now)

	err := store.Attend(fragmentID, now.Add(time.Minute))
	if err != nil {
		t.Fatalf("Attend() error = %v", err)
	}

	fragment, _ := store.Get(fragmentID)
	if fragment.Layer() != LayerContext {
		t.Fatalf("Layer() = %v, want %v", fragment.Layer(), LayerContext)
	}
}

func TestAttendDoesNotPullSettledMemoriesBack(t *testing.T) {
	t.Parallel()

	now := reference()
	store := New(DefaultTuning(), TokenOverlap{})
	fragmentID := store.Perceive(vividExperience(), now)

	// Sleep enough times to settle it into long-term memory.
	for range 5 {
		store.Sleep(now)
	}

	err := store.Attend(fragmentID, now)
	if err != nil {
		t.Fatalf("Attend() error = %v", err)
	}

	fragment, _ := store.Get(fragmentID)
	if fragment.Layer() != LayerLongTerm {
		t.Fatalf("Layer() = %v, want a settled memory to stay settled", fragment.Layer())
	}
}

func TestMutatorsRejectUnknownFragments(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		call func(store *Store) error
	}{
		{name: "attend", call: func(store *Store) error { return store.Attend(missingID, reference()) }},
		{name: "pin", call: func(store *Store) error { return store.Pin(missingID) }},
		{name: "confide", call: func(store *Store) error { return store.Confide(missingID, aoi) }},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			store := New(DefaultTuning(), TokenOverlap{})

			err := testCase.call(store)
			if !errors.Is(err, ErrUnknownFragment) {
				t.Fatalf("error = %v, want %v", err, ErrUnknownFragment)
			}
		})
	}
}

func TestRecallFindsWhatMatches(t *testing.T) {
	t.Parallel()

	now := reference()
	store := New(DefaultTuning(), TokenOverlap{})
	store.Perceive(vividExperience(), now)
	store.Perceive(faintExperience(), now)

	recalled := store.Recall(plainCue(catCue), now)
	if len(recalled) != 1 {
		t.Fatalf("Recall() returned %d memories, want only the matching one", len(recalled))
	}

	if recalled[0].Content() != catContent {
		t.Fatalf("Content() = %q, want %q", recalled[0].Content(), catContent)
	}
}

func TestRecallIgnoresWhatDoesNotMatch(t *testing.T) {
	t.Parallel()

	now := reference()
	store := New(DefaultTuning(), TokenOverlap{})
	store.Perceive(vividExperience(), now)

	if got := store.Recall(plainCue("quantum mechanics"), now); len(got) != 0 {
		t.Fatalf("Recall() returned %d memories, want none", len(got))
	}
}

func TestRecallEmptyCueFindsNothing(t *testing.T) {
	t.Parallel()

	now := reference()
	store := New(DefaultTuning(), TokenOverlap{})
	store.Perceive(vividExperience(), now)

	if got := store.Recall(plainCue(""), now); len(got) != 0 {
		t.Fatalf("Recall() returned %d memories, want none", len(got))
	}
}

func TestRecallPrefersTheVividMemory(t *testing.T) {
	t.Parallel()

	now := reference()
	store := New(DefaultTuning(), TokenOverlap{})

	store.Perceive(Experience{
		Content:      "cats were fine",
		Feeling:      0,
		Will:         0,
		Confidential: false,
	}, now)
	store.Perceive(Experience{
		Content:      "cats were wonderful",
		Feeling:      1,
		Will:         0,
		Confidential: false,
	}, now)

	recalled := store.Recall(plainCue(catCue), now)
	if len(recalled) < 2 {
		t.Fatalf("Recall() returned %d memories, want both", len(recalled))
	}

	if recalled[0].Content() != "cats were wonderful" {
		t.Fatalf("first recalled = %q, want the strongly felt memory first", recalled[0].Content())
	}
}

func TestRecallLandsInWorkingMemory(t *testing.T) {
	t.Parallel()

	now := reference()
	store := New(DefaultTuning(), TokenOverlap{})
	fragmentID := store.Perceive(vividExperience(), now)

	// Settle it into long-term memory first.
	for range 5 {
		store.Sleep(now)
	}

	store.Recall(plainCue(catCue), now)

	fragment, _ := store.Get(fragmentID)
	if fragment.Layer() != LayerWorking {
		t.Fatalf("Layer() = %v, want a recalled memory on the working bench", fragment.Layer())
	}
}

func TestRecallStrengthensWhatComesBack(t *testing.T) {
	t.Parallel()

	now := reference()
	store := New(DefaultTuning(), TokenOverlap{})
	fragmentID := store.Perceive(vividExperience(), now)

	store.Recall(plainCue(catCue), now)
	store.Recall(plainCue(catCue), now)

	fragment, _ := store.Get(fragmentID)
	if fragment.Recalls() != 2 {
		t.Fatalf("Recalls() = %d, want 2", fragment.Recalls())
	}
}

func TestRecallHonoursTheLimit(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		limit int
		want  int
	}{
		{name: "narrow recall", limit: 2, want: 2},
		{name: "zero falls back to the default", limit: 0, want: DefaultRecallLimit},
		{name: "negative falls back to the default", limit: -3, want: DefaultRecallLimit},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			now := reference()
			tuning := DefaultTuning()
			tuning.RecallLimit = testCase.limit
			store := New(tuning, TokenOverlap{})

			for range 8 {
				store.Perceive(vividExperience(), now)
			}

			if got := len(store.Recall(plainCue(catCue), now)); got != testCase.want {
				t.Fatalf("Recall() returned %d memories, want %d", got, testCase.want)
			}
		})
	}
}

func TestRecallIsStableAcrossRuns(t *testing.T) {
	t.Parallel()

	now := reference()
	first := recallOrder(now)

	for range 5 {
		if got := recallOrder(now); got != first {
			t.Fatalf("recall order = %q, want the stable %q", got, first)
		}
	}
}

// recallOrder builds an identical store and reports the order recall came back
// in, so ties can be checked for stability.
func recallOrder(now time.Time) string {
	tuning := DefaultTuning()
	tuning.RecallLimit = 3
	store := New(tuning, TokenOverlap{})

	for range 5 {
		store.Perceive(vividExperience(), now)
	}

	order := ""

	var orderSb404 strings.Builder
	for _, fragment := range store.Recall(plainCue(catCue), now) {
		orderSb404.WriteString(fragment.ID() + " ")
	}

	order += orderSb404.String()

	return order
}

func TestConfidentialMemoryStaysPrivate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		cue  Cue
		want int
	}{
		{
			name: "withheld from someone else",
			cue:  Cue{Text: catCue, Audience: rin, IncludeConfidential: false},
			want: 0,
		},
		{
			name: "shared with the one who was told",
			cue:  Cue{Text: catCue, Audience: aoi, IncludeConfidential: false},
			want: 1,
		},
		{
			name: "reachable to the persona's own thinking",
			cue:  Cue{Text: catCue, Audience: rin, IncludeConfidential: true},
			want: 1,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			now := reference()
			store := New(DefaultTuning(), TokenOverlap{})
			fragmentID := store.Perceive(vividExperience(), now)

			err := store.Confide(fragmentID, aoi)
			if err != nil {
				t.Fatalf("Confide() error = %v", err)
			}

			if got := len(store.Recall(testCase.cue, now)); got != testCase.want {
				t.Fatalf("Recall() returned %d memories, want %d", got, testCase.want)
			}
		})
	}
}

func TestConfidentialFromTheStart(t *testing.T) {
	t.Parallel()

	now := reference()
	store := New(DefaultTuning(), TokenOverlap{})
	fragmentID := store.Perceive(Experience{
		Content:      catContent,
		Feeling:      0.9,
		Will:         0.5,
		Confidential: true,
	}, now)

	fragment, _ := store.Get(fragmentID)
	if !fragment.Confidential() {
		t.Fatal("Confidential() = false, want true")
	}

	// No confidant was recorded, so it is the persona's own to think about and
	// nobody else's to hear.
	if got := len(store.Recall(Cue{Text: catCue, Audience: aoi, IncludeConfidential: false}, now)); got != 0 {
		t.Fatalf("Recall() returned %d memories, want none", got)
	}
}

func TestSleepForgetsTheFaintest(t *testing.T) {
	t.Parallel()

	now := reference()
	store := New(DefaultTuning(), TokenOverlap{})
	store.Perceive(vividExperience(), now)
	store.Perceive(faintExperience(), now)

	// A day later the faint memory has nothing holding it up.
	forgotten := store.Sleep(now.Add(4 * recencyHalfLife))
	if forgotten != 1 {
		t.Fatalf("Sleep() forgot %d memories, want 1", forgotten)
	}

	if store.Len() != 1 {
		t.Fatalf("Len() = %d, want the vivid memory to survive", store.Len())
	}
}

func TestSleepKeepsWhatWasMeantToBeKept(t *testing.T) {
	t.Parallel()

	now := reference()
	store := New(DefaultTuning(), TokenOverlap{})

	// Nothing remarkable happened, but the persona meant to remember it. This is
	// how a name or a promise sticks, without a special rule for it.
	store.Perceive(Experience{
		Content:      "her name is Aoi",
		Feeling:      0,
		Will:         1,
		Confidential: false,
	}, now)

	if got := store.Sleep(now.Add(4 * recencyHalfLife)); got != 0 {
		t.Fatalf("Sleep() forgot %d memories, want it to keep what was meant to stick", got)
	}
}

func TestSleepCompactsSurvivors(t *testing.T) {
	t.Parallel()

	now := reference()
	store := New(DefaultTuning(), TokenOverlap{})
	fragmentID := store.Perceive(vividExperience(), now)

	store.Sleep(now)

	fragment, _ := store.Get(fragmentID)
	if fragment.Strength() >= fullStrength {
		t.Fatalf("Strength() = %v, want detail to have been given up", fragment.Strength())
	}
}

func TestSleepAdvancesSurvivorsOneLayer(t *testing.T) {
	t.Parallel()

	now := reference()
	store := New(DefaultTuning(), TokenOverlap{})
	fragmentID := store.Perceive(vividExperience(), now)

	wantLayers := []Layer{LayerContext, LayerWorking, LayerDay, LayerLongTerm, LayerLongTerm}
	for _, want := range wantLayers {
		store.Sleep(now)

		fragment, known := store.Get(fragmentID)
		if !known {
			t.Fatal("Get() known = false, want the memory to survive")
		}

		if fragment.Layer() != want {
			t.Fatalf("Layer() = %v, want %v", fragment.Layer(), want)
		}
	}
}

func TestSleepHonoursForgetfulness(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		forgetfulness float64
		wantForgotten int
	}{
		{name: "a persona that forgets nothing", forgetfulness: 0, wantForgotten: 0},
		{name: "a persona that forgets like a person", forgetfulness: DefaultForgetfulness, wantForgotten: 1},
		{name: "a scatterbrained persona", forgetfulness: 4, wantForgotten: 1},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			now := reference()
			tuning := DefaultTuning()
			tuning.Forgetfulness = testCase.forgetfulness
			store := New(tuning, TokenOverlap{})
			store.Perceive(faintExperience(), now)

			got := store.Sleep(now.Add(4 * recencyHalfLife))
			if got != testCase.wantForgotten {
				t.Fatalf("Sleep() forgot %d memories, want %d", got, testCase.wantForgotten)
			}
		})
	}
}

func TestSleepHonoursCompactionBounds(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		compaction float64
	}{
		{name: "zero falls back to the default", compaction: 0},
		{name: "negative falls back to the default", compaction: -1},
		{name: "above one falls back to the default", compaction: 2},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			now := reference()
			tuning := DefaultTuning()
			tuning.Compaction = testCase.compaction
			store := New(tuning, TokenOverlap{})
			fragmentID := store.Perceive(vividExperience(), now)

			store.Sleep(now)

			fragment, _ := store.Get(fragmentID)
			if got := fragment.Strength(); got != DefaultCompaction {
				t.Fatalf("Strength() = %v, want %v", got, DefaultCompaction)
			}
		})
	}
}

func TestPinHoldsAgainstForgetting(t *testing.T) {
	t.Parallel()

	now := reference()
	store := New(DefaultTuning(), TokenOverlap{})
	fragmentID := store.Perceive(faintExperience(), now)

	err := store.Pin(fragmentID)
	if err != nil {
		t.Fatalf("Pin() error = %v", err)
	}

	if got := store.Sleep(now.Add(8 * recencyHalfLife)); got != 0 {
		t.Fatalf("Sleep() forgot %d memories, want the pinned one to survive", got)
	}

	fragment, _ := store.Get(fragmentID)
	if !fragment.Pinned() {
		t.Fatal("Pinned() = false, want true")
	}
}

func TestSleepDropsTheConfidantRecordToo(t *testing.T) {
	t.Parallel()

	now := reference()
	store := New(DefaultTuning(), TokenOverlap{})
	fragmentID := store.Perceive(faintExperience(), now)

	err := store.Confide(fragmentID, aoi)
	if err != nil {
		t.Fatalf("Confide() error = %v", err)
	}

	store.Sleep(now.Add(8 * recencyHalfLife))

	if len(store.confidants) != 0 {
		t.Fatalf("confidants = %d, want a forgotten memory to leave nothing behind", len(store.confidants))
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
		{name: "no time keeps everything", elapsed: 0, halfLife: time.Hour, want: maxScore},
		{name: "time going backwards keeps everything", elapsed: -time.Hour, halfLife: time.Hour, want: maxScore},
		{name: "zero half-life leaves nothing", elapsed: time.Hour, halfLife: 0, want: minScore},
		{name: "one half-life keeps half", elapsed: time.Hour, halfLife: time.Hour, want: halving},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := decayFactor(testCase.elapsed, testCase.halfLife); got != testCase.want {
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
		{name: "below the range is raised", value: -2, want: minScore},
		{name: "above the range is lowered", value: 2, want: maxScore},
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

func TestTokenOverlapScore(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		cue     string
		content string
		want    float64
	}{
		{name: "no cue matches nothing", cue: "", content: catContent, want: 0},
		{name: "every word present", cue: "cats about", content: catContent, want: 1},
		{name: "half the words present", cue: "cats dogs", content: catContent, want: halving},
		{name: "nothing in common", cue: "dogs", content: catContent, want: 0},
		{name: "case is ignored", cue: "CATS", content: catContent, want: 1},
		{name: "punctuation is ignored", cue: "cats!", content: catContent, want: 1},
		{name: "ideographic punctuation is ignored", cue: "\u732b", content: "\u732b\u3001\u53ef\u611b\u3044", want: 1},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := (TokenOverlap{}).Score(testCase.cue, testCase.content); got != testCase.want {
				t.Fatalf("Score() = %v, want %v", got, testCase.want)
			}
		})
	}
}
