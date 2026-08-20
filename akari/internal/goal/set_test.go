package goal

import (
	"errors"
	"math"
	"testing"
	"time"
)

const (
	tolerance = 1e-9
	restID    = "rest"
	finishID  = "finish"
)

// reference is a fixed instant so tests never depend on the wall clock.
func reference() time.Time {
	return time.Date(2026, time.August, 20, 9, 0, 0, 0, time.UTC)
}

func restDesire() Desire {
	return Desire{ID: restID, Want: "take a break", Pull: 0.4}
}

func finishDesire() Desire {
	return Desire{ID: finishID, Want: "finish the report", Pull: 0.8}
}

func TestDefaultTuning(t *testing.T) {
	t.Parallel()

	want := Tuning{
		Persistence:      DefaultPersistence,
		Capacity:         DefaultCapacity,
		SetbackTolerance: DefaultSetbackTolerance,
	}

	if got := DefaultTuning(); got != want {
		t.Fatalf("DefaultTuning() = %#v, want %#v", got, want)
	}
}

func TestStatusString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		status Status
		want   string
	}{
		{name: "active", status: StatusActive, want: NameActive},
		{name: "completed", status: StatusCompleted, want: NameCompleted},
		{name: "abandoned", status: StatusAbandoned, want: NameAbandoned},
		{name: "out of range", status: Status(-1), want: NameUnknown},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := testCase.status.String(); got != testCase.want {
				t.Fatalf("Status.String() = %q, want %q", got, testCase.want)
			}
		})
	}
}

func TestNewIsEmpty(t *testing.T) {
	t.Parallel()

	set := New(DefaultTuning())

	if got := set.Desires(); got != 0 {
		t.Fatalf("Desires() = %d, want 0", got)
	}

	if got := len(set.Active()); got != 0 {
		t.Fatalf("Active() = %d, want 0", got)
	}

	if got := len(set.Settled()); got != 0 {
		t.Fatalf("Settled() = %d, want 0", got)
	}
}

func TestWishAllowsContradictions(t *testing.T) {
	t.Parallel()

	set := New(DefaultTuning())
	set.Wish(restDesire())
	set.Wish(finishDesire())

	if got := set.Desires(); got != 2 {
		t.Fatalf("Desires() = %d, want both contradicting wishes to be held", got)
	}
}

func TestWishReplacesSameID(t *testing.T) {
	t.Parallel()

	set := New(DefaultTuning())
	set.Wish(restDesire())
	set.Wish(Desire{ID: restID, Want: "nap properly", Pull: 0.9})

	if got := set.Desires(); got != 1 {
		t.Fatalf("Desires() = %d, want 1", got)
	}

	strongest, found := set.Strongest()
	if !found {
		t.Fatal("Strongest() found = false, want true")
	}

	if strongest.Want != "nap properly" {
		t.Fatalf("Strongest() want = %q, want the wish to have been replaced", strongest.Want)
	}
}

func TestStrongest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		desires   []Desire
		wantFound bool
		wantID    string
	}{
		{name: "nothing wished for", desires: nil, wantFound: false, wantID: ""},
		{
			name:      "single wish",
			desires:   []Desire{restDesire()},
			wantFound: true,
			wantID:    restID,
		},
		{
			name:      "strongest pull wins",
			desires:   []Desire{restDesire(), finishDesire()},
			wantFound: true,
			wantID:    finishID,
		},
		{
			name:      "a wish with no pull still counts when it is all there is",
			desires:   []Desire{{ID: restID, Want: "idle", Pull: 0}},
			wantFound: true,
			wantID:    restID,
		},
		{
			name: "equal pulls resolve to the lowest identifier",
			desires: []Desire{
				{ID: "b", Want: "second", Pull: 0.5},
				{ID: "a", Want: "first", Pull: 0.5},
			},
			wantFound: true,
			wantID:    "a",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			set := New(DefaultTuning())
			for _, desire := range testCase.desires {
				set.Wish(desire)
			}

			got, found := set.Strongest()
			if found != testCase.wantFound {
				t.Fatalf("Strongest() found = %v, want %v", found, testCase.wantFound)
			}

			if got.ID != testCase.wantID {
				t.Fatalf("Strongest() id = %q, want %q", got.ID, testCase.wantID)
			}
		})
	}
}

func TestCommit(t *testing.T) {
	t.Parallel()

	now := reference()
	set := New(DefaultTuning())
	set.Wish(finishDesire())

	intention, err := set.Commit(finishID, now)
	if err != nil {
		t.Fatalf("Commit() error = %v", err)
	}

	if intention.ID() != finishID {
		t.Fatalf("ID() = %q, want %q", intention.ID(), finishID)
	}

	if intention.Want() != finishDesire().Want {
		t.Fatalf("Want() = %q, want %q", intention.Want(), finishDesire().Want)
	}

	if intention.Status() != StatusActive {
		t.Fatalf("Status() = %v, want %v", intention.Status(), StatusActive)
	}

	if !intention.CommittedAt().Equal(now) {
		t.Fatalf("CommittedAt() = %v, want %v", intention.CommittedAt(), now)
	}

	if got := len(set.Active()); got != 1 {
		t.Fatalf("Active() = %d, want 1", got)
	}
}

func TestCommitErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		prepare func(set *Set, now time.Time)
		id      string
		wantErr error
	}{
		{
			name:    "unknown desire",
			prepare: func(_ *Set, _ time.Time) {},
			id:      "nothing",
			wantErr: ErrUnknownDesire,
		},
		{
			name: "already committed",
			prepare: func(set *Set, now time.Time) {
				set.Wish(finishDesire())
				mustCommitFinish(set, now)
			},
			id:      finishID,
			wantErr: ErrAlreadyCommitted,
		},
		{
			name: "no room for another intention",
			prepare: func(set *Set, now time.Time) {
				set.Wish(finishDesire())
				set.Wish(restDesire())
				mustCommitFinish(set, now)
			},
			id:      restID,
			wantErr: ErrNoCapacity,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			now := reference()
			set := New(DefaultTuning())
			testCase.prepare(set, now)

			_, err := set.Commit(testCase.id, now)
			if !errors.Is(err, testCase.wantErr) {
				t.Fatalf("Commit() error = %v, want %v", err, testCase.wantErr)
			}
		})
	}
}

func TestCommitHonoursCapacity(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		capacity int
		want     int
	}{
		{name: "one at a time is the human default", capacity: DefaultCapacity, want: 1},
		{name: "two can be carried when allowed", capacity: 2, want: 2},
		{name: "zero capacity still allows one", capacity: 0, want: 1},
		{name: "negative capacity still allows one", capacity: -5, want: 1},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			now := reference()
			tuning := DefaultTuning()
			tuning.Capacity = testCase.capacity
			set := New(tuning)
			set.Wish(finishDesire())
			set.Wish(restDesire())

			_, _ = set.Commit(finishID, now)
			_, _ = set.Commit(restID, now)

			if got := len(set.Active()); got != testCase.want {
				t.Fatalf("Active() = %d, want %d", got, testCase.want)
			}
		})
	}
}

func TestCompleteAndAbandon(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		settle     func(set *Set, now time.Time) error
		wantStatus Status
	}{
		{
			name:       "seeing it through",
			settle:     func(set *Set, now time.Time) error { return set.Complete(finishID, now) },
			wantStatus: StatusCompleted,
		},
		{
			name:       "letting it go",
			settle:     func(set *Set, now time.Time) error { return set.Abandon(finishID, now) },
			wantStatus: StatusAbandoned,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			now := reference()
			set := New(DefaultTuning())
			set.Wish(finishDesire())

			_, err := set.Commit(finishID, now)
			if err != nil {
				t.Fatalf("Commit() error = %v", err)
			}

			settledAt := now.Add(time.Hour)

			err = testCase.settle(set, settledAt)
			if err != nil {
				t.Fatalf("settle error = %v", err)
			}

			if got := len(set.Active()); got != 0 {
				t.Fatalf("Active() = %d, want 0", got)
			}

			settled := set.Settled()
			if len(settled) != 1 {
				t.Fatalf("Settled() = %d, want 1", len(settled))
			}

			if settled[0].Status() != testCase.wantStatus {
				t.Fatalf("Status() = %v, want %v", settled[0].Status(), testCase.wantStatus)
			}

			if !settled[0].SettledAt().Equal(settledAt) {
				t.Fatalf("SettledAt() = %v, want %v", settled[0].SettledAt(), settledAt)
			}
		})
	}
}

func TestSettleUnknownIntention(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		settle func(set *Set) error
	}{
		{name: "complete", settle: func(set *Set) error { return set.Complete(finishID, reference()) }},
		{name: "abandon", settle: func(set *Set) error { return set.Abandon(finishID, reference()) }},
		{name: "setback", settle: func(set *Set) error { return set.Setback(finishID) }},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			set := New(DefaultTuning())

			err := testCase.settle(set)
			if !errors.Is(err, ErrUnknownIntention) {
				t.Fatalf("error = %v, want %v", err, ErrUnknownIntention)
			}
		})
	}
}

func TestSettledIsACopy(t *testing.T) {
	t.Parallel()

	now := reference()
	set := New(DefaultTuning())
	set.Wish(finishDesire())

	_, err := set.Commit(finishID, now)
	if err != nil {
		t.Fatalf("Commit() error = %v", err)
	}

	err = set.Complete(finishID, now)
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}

	settled := set.Settled()
	settled[0] = Intention{
		id:          "tampered",
		want:        "",
		status:      StatusActive,
		committedAt: time.Time{},
		settledAt:   time.Time{},
		setbacks:    0,
	}

	if got := set.Settled()[0].ID(); got != finishID {
		t.Fatalf("Settled()[0].ID() = %q, want the record to be unchanged", got)
	}
}

func TestGrip(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		persistence float64
		setbacks    int
		want        float64
	}{
		{name: "fresh intention grips at full persistence", persistence: 0.6, setbacks: 0, want: 0.6},
		{name: "one setback loosens the grip", persistence: 0.6, setbacks: 1, want: 0.4},
		{name: "tolerance exhausted leaves no grip", persistence: 0.6, setbacks: 3, want: 0},
		{name: "beyond tolerance stays at zero", persistence: 0.6, setbacks: 9, want: 0},
		{name: "a flighty persona barely grips", persistence: 0, setbacks: 0, want: 0},
		{name: "persistence above one is capped", persistence: 5, setbacks: 0, want: maxGrip},
		{name: "persistence below zero is floored", persistence: -5, setbacks: 0, want: minGrip},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			now := reference()
			tuning := DefaultTuning()
			tuning.Persistence = testCase.persistence
			set := New(tuning)
			set.Wish(finishDesire())

			_, err := set.Commit(finishID, now)
			if err != nil {
				t.Fatalf("Commit() error = %v", err)
			}

			for range testCase.setbacks {
				err = set.Setback(finishID)
				if err != nil {
					t.Fatalf("Setback() error = %v", err)
				}
			}

			if got := set.Grip(finishID); !closeTo(got, testCase.want) {
				t.Fatalf("Grip() = %v, want %v", got, testCase.want)
			}
		})
	}
}

func TestGripOfSomethingNotCarried(t *testing.T) {
	t.Parallel()

	set := New(DefaultTuning())

	if got := set.Grip(finishID); got != minGrip {
		t.Fatalf("Grip() = %v, want %v", got, minGrip)
	}
}

func TestSetbackToleranceFloor(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		tolerance int
	}{
		{name: "zero tolerance behaves as one", tolerance: 0},
		{name: "negative tolerance behaves as one", tolerance: -3},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			now := reference()
			tuning := DefaultTuning()
			tuning.SetbackTolerance = testCase.tolerance
			set := New(tuning)
			set.Wish(finishDesire())

			_, err := set.Commit(finishID, now)
			if err != nil {
				t.Fatalf("Commit() error = %v", err)
			}

			err = set.Setback(finishID)
			if err != nil {
				t.Fatalf("Setback() error = %v", err)
			}

			if got := set.Grip(finishID); got != minGrip {
				t.Fatalf("Grip() = %v, want a single setback to exhaust the grip", got)
			}

			if !set.Spent(finishID) {
				t.Fatal("Spent() = false, want true")
			}
		})
	}
}

func TestSpent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		carry    bool
		setbacks int
		want     bool
	}{
		{name: "not carried is not spent", carry: false, setbacks: 0, want: false},
		{name: "fresh is not spent", carry: true, setbacks: 0, want: false},
		{name: "part way is not spent", carry: true, setbacks: DefaultSetbackTolerance - 1, want: false},
		{name: "at tolerance is spent", carry: true, setbacks: DefaultSetbackTolerance, want: true},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			now := reference()
			set := New(DefaultTuning())

			if testCase.carry {
				set.Wish(finishDesire())
				mustCommitFinish(set, now)

				for range testCase.setbacks {
					err := set.Setback(finishID)
					if err != nil {
						t.Fatalf("Setback() error = %v", err)
					}
				}
			}

			if got := set.Spent(finishID); got != testCase.want {
				t.Fatalf("Spent() = %v, want %v", got, testCase.want)
			}
		})
	}
}

func TestHolds(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		carry      bool
		challenger float64
		want       bool
	}{
		{name: "nothing carried holds nothing", carry: false, challenger: 0, want: false},
		{name: "a mild distraction does not pull it away", carry: true, challenger: 0.2, want: true},
		{name: "a distraction equal to the grip is resisted", carry: true, challenger: DefaultPersistence, want: true},
		{name: "a strong distraction wins", carry: true, challenger: 0.9, want: false},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			now := reference()
			set := New(DefaultTuning())

			if testCase.carry {
				set.Wish(finishDesire())
				mustCommitFinish(set, now)
			}

			if got := set.Holds(finishID, testCase.challenger); got != testCase.want {
				t.Fatalf("Holds() = %v, want %v", got, testCase.want)
			}
		})
	}
}

func TestHoldsWeakensWithSetbacks(t *testing.T) {
	t.Parallel()

	now := reference()
	set := New(DefaultTuning())
	set.Wish(finishDesire())

	_, err := set.Commit(finishID, now)
	if err != nil {
		t.Fatalf("Commit() error = %v", err)
	}

	const distraction = 0.5

	if !set.Holds(finishID, distraction) {
		t.Fatal("Holds() = false, want a fresh intention to resist")
	}

	err = set.Setback(finishID)
	if err != nil {
		t.Fatalf("Setback() error = %v", err)
	}

	if set.Holds(finishID, distraction) {
		t.Fatal("Holds() = true, want trouble to have loosened the grip")
	}
}

func TestIntentionAge(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		settle bool
		at     time.Duration
		ask    time.Duration
		want   time.Duration
	}{
		{name: "active measures up to now", settle: false, at: 0, ask: time.Hour, want: time.Hour},
		{name: "settled stops at the moment it ended", settle: true, at: time.Hour, ask: 5 * time.Hour, want: time.Hour},
		{name: "asking about the past does not go negative", settle: false, at: 0, ask: -time.Hour, want: 0},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			now := reference()
			set := New(DefaultTuning())
			set.Wish(finishDesire())

			intention, err := set.Commit(finishID, now)
			if err != nil {
				t.Fatalf("Commit() error = %v", err)
			}

			if testCase.settle {
				err := set.Complete(finishID, now.Add(testCase.at))
				if err != nil {
					t.Fatalf("Complete() error = %v", err)
				}

				intention = set.Settled()[0]
			}

			if got := intention.Age(now.Add(testCase.ask)); got != testCase.want {
				t.Fatalf("Age() = %v, want %v", got, testCase.want)
			}
		})
	}
}

func TestIntentionSetbacks(t *testing.T) {
	t.Parallel()

	now := reference()
	set := New(DefaultTuning())
	set.Wish(finishDesire())

	_, err := set.Commit(finishID, now)
	if err != nil {
		t.Fatalf("Commit() error = %v", err)
	}

	err = set.Setback(finishID)
	if err != nil {
		t.Fatalf("Setback() error = %v", err)
	}

	if got := set.Active()[0].Setbacks(); got != 1 {
		t.Fatalf("Setbacks() = %d, want 1", got)
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
		{name: "below the range is raised", value: -2, want: minGrip},
		{name: "above the range is lowered", value: 2, want: maxGrip},
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

// mustCommitFinish commits the "finish" wish while setting up a case, failing
// loudly if the fixture itself is wrong.
func mustCommitFinish(set *Set, now time.Time) {
	_, err := set.Commit(finishID, now)
	if err != nil {
		panic(err)
	}
}
