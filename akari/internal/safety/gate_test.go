package safety

import "testing"

const (
	kindSpeak  = "speak"
	kindDelete = "delete-file"
	kindThink  = "think"
)

func TestVerdictString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		verdict Verdict
		want    string
	}{
		{name: "allow", verdict: Allow, want: NameAllow},
		{name: "confirm", verdict: Confirm, want: NameConfirm},
		{name: "refuse", verdict: Refuse, want: NameRefuse},
		{name: "out of range", verdict: Verdict(-1), want: NameUnknown},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := testCase.verdict.String(); got != testCase.want {
				t.Fatalf("Verdict.String() = %q, want %q", got, testCase.want)
			}
		})
	}
}

func TestReachString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		reach Reach
		want  string
	}{
		{name: "internal", reach: ReachInternal, want: NameInternal},
		{name: "reversible", reach: ReachReversible, want: NameReversible},
		{name: "outward", reach: ReachOutward, want: NameOutward},
		{name: "irreversible", reach: ReachIrreversible, want: NameIrreversible},
		{name: "out of range", reach: Reach(-1), want: NameUnknown},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := testCase.reach.String(); got != testCase.want {
				t.Fatalf("Reach.String() = %q, want %q", got, testCase.want)
			}
		})
	}
}

func TestDefaultPolicy(t *testing.T) {
	t.Parallel()

	policy := DefaultPolicy()

	if !policy.ConfirmOutward {
		t.Fatal("ConfirmOutward = false, want the design's default of asking first")
	}

	if !policy.ConfirmIrreversible {
		t.Fatal("ConfirmIrreversible = false, want the design's default of asking first")
	}

	if policy.ForbiddenKinds != nil {
		t.Fatalf("ForbiddenKinds = %v, want nothing banned by default", policy.ForbiddenKinds)
	}
}

func TestJudge(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		policy Policy
		act    Act
		want   Verdict
	}{
		{
			name:   "thinking is the persona's own business",
			policy: DefaultPolicy(),
			act:    Act{Kind: kindThink, Reach: ReachInternal, Forbidden: false},
			want:   Allow,
		},
		{
			name:   "something that can be put back is allowed",
			policy: DefaultPolicy(),
			act:    Act{Kind: "edit-note", Reach: ReachReversible, Forbidden: false},
			want:   Allow,
		},
		{
			name:   "anything other people see is checked first",
			policy: DefaultPolicy(),
			act:    Act{Kind: kindSpeak, Reach: ReachOutward, Forbidden: false},
			want:   Confirm,
		},
		{
			name:   "anything that cannot be undone is checked first",
			policy: DefaultPolicy(),
			act:    Act{Kind: kindDelete, Reach: ReachIrreversible, Forbidden: false},
			want:   Confirm,
		},
		{
			name:   "an act marked never-to-perform is refused",
			policy: DefaultPolicy(),
			act:    Act{Kind: kindThink, Reach: ReachInternal, Forbidden: true},
			want:   Refuse,
		},
		{
			name: "a forbidden kind is refused",
			policy: Policy{
				ConfirmOutward:      true,
				ConfirmIrreversible: true,
				ForbiddenKinds:      []string{kindDelete},
			},
			act:  Act{Kind: kindDelete, Reach: ReachIrreversible, Forbidden: false},
			want: Refuse,
		},
		{
			name: "a permissive policy lets outward acts through",
			policy: Policy{
				ConfirmOutward:      false,
				ConfirmIrreversible: true,
				ForbiddenKinds:      nil,
			},
			act:  Act{Kind: kindSpeak, Reach: ReachOutward, Forbidden: false},
			want: Allow,
		},
		{
			name: "a permissive policy lets irreversible acts through",
			policy: Policy{
				ConfirmOutward:      true,
				ConfirmIrreversible: false,
				ForbiddenKinds:      nil,
			},
			act:  Act{Kind: kindDelete, Reach: ReachIrreversible, Forbidden: false},
			want: Allow,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			gate := New(testCase.policy)

			decision := gate.Judge(testCase.act)
			if decision.Verdict != testCase.want {
				t.Fatalf("Judge() verdict = %v, want %v", decision.Verdict, testCase.want)
			}

			if decision.Reason == "" {
				t.Fatal("Judge() reason is empty, want an explanation on every decision")
			}
		})
	}
}

func TestPermits(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		act  Act
		want bool
	}{
		{
			name: "internal acts need nobody's permission",
			act:  Act{Kind: kindThink, Reach: ReachInternal, Forbidden: false},
			want: true,
		},
		{
			name: "outward acts do",
			act:  Act{Kind: kindSpeak, Reach: ReachOutward, Forbidden: false},
			want: false,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			gate := New(DefaultPolicy())

			if got := gate.Permits(testCase.act); got != testCase.want {
				t.Fatalf("Permits() = %v, want %v", got, testCase.want)
			}
		})
	}
}

// TestJudgeIgnoresHowMuchThePersonaWants is the point of the whole package: the
// gate takes no account of the persona's state, so there is no amount of wanting
// that gets an act through. It is written as a test so that adding such a
// parameter later would have to break it deliberately.
func TestJudgeIgnoresHowMuchThePersonaWants(t *testing.T) {
	t.Parallel()

	gate := New(DefaultPolicy())
	act := Act{Kind: kindDelete, Reach: ReachIrreversible, Forbidden: false}

	first := gate.Judge(act)
	for range 100 {
		if got := gate.Judge(act); got != first {
			t.Fatalf("Judge() = %#v, want the same answer every time: %#v", got, first)
		}
	}

	if first.Verdict == Allow {
		t.Fatal("Judge() verdict = allow, want an irreversible act to need agreement")
	}
}
