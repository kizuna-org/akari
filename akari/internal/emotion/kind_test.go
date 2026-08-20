package emotion

import "testing"

func TestKindString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		kind Kind
		want string
	}{
		{name: "joy", kind: Joy, want: NameJoy},
		{name: "fun", kind: Fun, want: NameFun},
		{name: "relief", kind: Relief, want: NameRelief},
		{name: "anticipation", kind: Anticipation, want: NameAnticipation},
		{name: "anger", kind: Anger, want: NameAnger},
		{name: "sadness", kind: Sadness, want: NameSadness},
		{name: "anxiety", kind: Anxiety, want: NameAnxiety},
		{name: "boredom", kind: Boredom, want: NameBoredom},
		{name: "surprise", kind: Surprise, want: NameSurprise},
		{name: "out of range", kind: Kind(-1), want: NameUnknown},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := testCase.kind.String(); got != testCase.want {
				t.Fatalf("Kind.String() = %q, want %q", got, testCase.want)
			}
		})
	}
}

func TestTendencyString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		tendency Tendency
		want     string
	}{
		{name: "none", tendency: TendencyNone, want: NameNone},
		{name: "freeze", tendency: TendencyFreeze, want: NameFreeze},
		{name: "avoid", tendency: TendencyAvoid, want: NameAvoid},
		{name: "out of range", tendency: Tendency(-1), want: NameUnknown},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := testCase.tendency.String(); got != testCase.want {
				t.Fatalf("Tendency.String() = %q, want %q", got, testCase.want)
			}
		})
	}
}

func TestKindValence(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		kind Kind
		want float64
	}{
		{name: "joy is pleasant", kind: Joy, want: 1},
		{name: "fun is pleasant", kind: Fun, want: 1},
		{name: "relief is pleasant", kind: Relief, want: 1},
		{name: "anticipation is pleasant", kind: Anticipation, want: 1},
		{name: "anger is unpleasant", kind: Anger, want: -1},
		{name: "sadness is unpleasant", kind: Sadness, want: -1},
		{name: "anxiety is unpleasant", kind: Anxiety, want: -1},
		{name: "boredom is unpleasant", kind: Boredom, want: -1},
		{name: "surprise is neither", kind: Surprise, want: 0},
		{name: "out of range is neither", kind: Kind(-1), want: 0},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := testCase.kind.Valence(); got != testCase.want {
				t.Fatalf("Kind.Valence() = %v, want %v", got, testCase.want)
			}
		})
	}
}

func TestKindTendency(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		kind          Kind
		want          Tendency
		wantReflexive bool
	}{
		{name: "surprise freezes", kind: Surprise, want: TendencyFreeze, wantReflexive: true},
		{name: "anxiety avoids", kind: Anxiety, want: TendencyAvoid, wantReflexive: true},
		{name: "joy has no reflex", kind: Joy, want: TendencyNone, wantReflexive: false},
		{name: "fun has no reflex", kind: Fun, want: TendencyNone, wantReflexive: false},
		{name: "relief has no reflex", kind: Relief, want: TendencyNone, wantReflexive: false},
		{name: "anticipation has no reflex", kind: Anticipation, want: TendencyNone, wantReflexive: false},
		{name: "anger has no reflex", kind: Anger, want: TendencyNone, wantReflexive: false},
		{name: "sadness has no reflex", kind: Sadness, want: TendencyNone, wantReflexive: false},
		{name: "boredom has no reflex", kind: Boredom, want: TendencyNone, wantReflexive: false},
		{name: "out of range has no reflex", kind: Kind(-1), want: TendencyNone, wantReflexive: false},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := testCase.kind.Tendency(); got != testCase.want {
				t.Fatalf("Kind.Tendency() = %v, want %v", got, testCase.want)
			}

			if got := testCase.kind.IsReflexive(); got != testCase.wantReflexive {
				t.Fatalf("Kind.IsReflexive() = %v, want %v", got, testCase.wantReflexive)
			}
		})
	}
}

func TestKindValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		kind Kind
		want bool
	}{
		{name: "first kind", kind: Joy, want: true},
		{name: "last kind", kind: Surprise, want: true},
		{name: "below range", kind: Kind(-1), want: false},
		{name: "above range", kind: Kind(kindCount), want: false},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := testCase.kind.valid(); got != testCase.want {
				t.Fatalf("Kind.valid() = %v, want %v", got, testCase.want)
			}
		})
	}
}
