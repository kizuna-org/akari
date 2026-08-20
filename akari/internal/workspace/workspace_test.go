package workspace

import (
	"errors"
	"testing"
	"time"
)

const (
	talkChannel   = "interaction"
	idleChannel   = "speculation"
	actChannel    = "action"
	speakAct      = "speak"
	deleteAct     = "delete-file"
	testSeed      = 42
	sampleRuns    = 400
	minorityShare = 0.05
)

// reference is a fixed instant so tests never depend on the wall clock.
func reference() time.Time {
	return time.Date(2026, time.August, 20, 9, 0, 0, 0, time.UTC)
}

func bid(channel string, weight float64) Bid {
	return Bid{
		Channel:      channel,
		Content:      channel + " has something to say",
		Weight:       weight,
		Act:          "",
		Irreversible: false,
		Outward:      false,
	}
}

// recorder is a channel that remembers everything it was told.
type recorder struct {
	moments []Moment
}

func (r *recorder) Receive(moment Moment) {
	r.moments = append(r.moments, moment)
}

// allowingJudge permits everything, standing in for a permissive policy.
type allowingJudge struct{}

func (allowingJudge) Allows(string, bool, bool) bool { return true }

// refusingJudge holds everything back, standing in for the safety gate saying no.
type refusingJudge struct{}

func (refusingJudge) Allows(string, bool, bool) bool { return false }

// outwardJudge only holds back acts other people would see.
type outwardJudge struct{}

func (outwardJudge) Allows(_ string, outward, irreversible bool) bool {
	return !outward && !irreversible
}

func TestNewNormalisesArguments(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		capacity int
		want     int
	}{
		{name: "one is the default", capacity: DefaultCapacity, want: DefaultCapacity},
		{name: "a wider conscious layer is allowed", capacity: 3, want: 3},
		{name: "zero is raised to one", capacity: 0, want: DefaultCapacity},
		{name: "negative is raised to one", capacity: -5, want: DefaultCapacity},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			space := New(testCase.capacity, nil, nil)
			for index := range testCase.want + 2 {
				space.Bid(bid(talkChannel, float64(index+1)))
			}

			moments, err := space.Cycle(reference())
			if err != nil {
				t.Fatalf("Cycle() error = %v", err)
			}

			if len(moments) != testCase.want {
				t.Fatalf("Cycle() returned %d moments, want %d", len(moments), testCase.want)
			}
		})
	}
}

func TestCycleWithNothingBid(t *testing.T) {
	t.Parallel()

	space := New(DefaultCapacity, StrongestChooser{}, nil)

	_, err := space.Cycle(reference())
	if !errors.Is(err, ErrNothingBid) {
		t.Fatalf("Cycle() error = %v, want %v", err, ErrNothingBid)
	}
}

func TestPending(t *testing.T) {
	t.Parallel()

	space := New(DefaultCapacity, StrongestChooser{}, nil)

	if got := space.Pending(); got != 0 {
		t.Fatalf("Pending() = %d, want 0", got)
	}

	space.Bid(bid(talkChannel, 0.5))
	space.Bid(bid(idleChannel, 0.2))

	if got := space.Pending(); got != 2 {
		t.Fatalf("Pending() = %d, want 2", got)
	}

	_, err := space.Cycle(reference())
	if err != nil {
		t.Fatalf("Cycle() error = %v", err)
	}

	if got := space.Pending(); got != 0 {
		t.Fatalf("Pending() = %d, want a cycle to clear what was waiting", got)
	}
}

func TestCycleTheStrongestBidUsuallyWins(t *testing.T) {
	t.Parallel()

	space := New(DefaultCapacity, StrongestChooser{}, nil)
	space.Bid(bid(idleChannel, 0.1))
	space.Bid(bid(talkChannel, 0.9))

	moments, err := space.Cycle(reference())
	if err != nil {
		t.Fatalf("Cycle() error = %v", err)
	}

	if moments[0].Winner.Channel != talkChannel {
		t.Fatalf("winner = %q, want %q", moments[0].Winner.Channel, talkChannel)
	}

	if moments[0].Considered != 2 {
		t.Fatalf("Considered = %d, want 2", moments[0].Considered)
	}
}

func TestCycleBroadcastsToEveryChannel(t *testing.T) {
	t.Parallel()

	space := New(DefaultCapacity, StrongestChooser{}, nil)

	winner := &recorder{moments: nil}
	bystander := &recorder{moments: nil}

	space.Subscribe(talkChannel, winner)
	space.Subscribe(actChannel, bystander)

	space.Bid(bid(talkChannel, 0.9))

	_, err := space.Cycle(reference())
	if err != nil {
		t.Fatalf("Cycle() error = %v", err)
	}

	// The channel that said nothing is told too: that is what makes the
	// workspace global rather than just shared.
	if len(bystander.moments) != 1 {
		t.Fatalf("bystander heard %d moments, want 1", len(bystander.moments))
	}

	if len(winner.moments) != 1 {
		t.Fatalf("winner heard %d moments, want 1", len(winner.moments))
	}
}

func TestSubscribeReplacesTheSameName(t *testing.T) {
	t.Parallel()

	space := New(DefaultCapacity, StrongestChooser{}, nil)

	first := &recorder{moments: nil}
	second := &recorder{moments: nil}

	space.Subscribe(talkChannel, first)
	space.Subscribe(talkChannel, second)

	space.Bid(bid(talkChannel, 0.5))

	_, err := space.Cycle(reference())
	if err != nil {
		t.Fatalf("Cycle() error = %v", err)
	}

	if len(first.moments) != 0 {
		t.Fatalf("replaced subscriber heard %d moments, want 0", len(first.moments))
	}

	if len(second.moments) != 1 {
		t.Fatalf("current subscriber heard %d moments, want 1", len(second.moments))
	}
}

func TestCycleDropsWhatDidNotSurface(t *testing.T) {
	t.Parallel()

	space := New(DefaultCapacity, StrongestChooser{}, nil)
	space.Bid(bid(talkChannel, 0.9))
	space.Bid(bid(idleChannel, 0.1))

	_, err := space.Cycle(reference())
	if err != nil {
		t.Fatalf("Cycle() error = %v", err)
	}

	// The losing thought is not queued forever; whatever is still thinking it
	// must offer it again.
	if got := space.Pending(); got != 0 {
		t.Fatalf("Pending() = %d, want losing bids to be dropped", got)
	}

	_, err = space.Cycle(reference())
	if !errors.Is(err, ErrNothingBid) {
		t.Fatalf("Cycle() error = %v, want %v", err, ErrNothingBid)
	}
}

func TestCycleCommitsAndWithholds(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		judge         Judge
		bid           Bid
		wantCommitted bool
		wantWithheld  bool
	}{
		{
			name:  "a bare thought is only noticed",
			judge: allowingJudge{},
			bid: Bid{
				Channel: idleChannel, Content: "idle musing", Weight: 0.5,
				Act: "", Irreversible: false, Outward: false,
			},
			wantCommitted: false,
			wantWithheld:  false,
		},
		{
			name:  "an allowed act is taken on",
			judge: allowingJudge{},
			bid: Bid{
				Channel: talkChannel, Content: "say hello", Weight: 0.5,
				Act: speakAct, Irreversible: false, Outward: false,
			},
			wantCommitted: true,
			wantWithheld:  false,
		},
		{
			name:  "a refused act is held back",
			judge: refusingJudge{},
			bid: Bid{
				Channel: actChannel, Content: "delete it", Weight: 0.9,
				Act: deleteAct, Irreversible: true, Outward: false,
			},
			wantCommitted: false,
			wantWithheld:  true,
		},
		{
			name:  "an outward act is held back but an inward one is not",
			judge: outwardJudge{},
			bid: Bid{
				Channel: talkChannel, Content: "post it publicly", Weight: 0.9,
				Act: speakAct, Irreversible: false, Outward: true,
			},
			wantCommitted: false,
			wantWithheld:  true,
		},
		{
			name:  "with no judge at all everything is taken on",
			judge: nil,
			bid: Bid{
				Channel: actChannel, Content: "delete it", Weight: 0.9,
				Act: deleteAct, Irreversible: true, Outward: true,
			},
			wantCommitted: true,
			wantWithheld:  false,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			space := New(DefaultCapacity, StrongestChooser{}, testCase.judge)
			space.Bid(testCase.bid)

			moments, err := space.Cycle(reference())
			if err != nil {
				t.Fatalf("Cycle() error = %v", err)
			}

			if moments[0].Committed != testCase.wantCommitted {
				t.Fatalf("Committed = %v, want %v", moments[0].Committed, testCase.wantCommitted)
			}

			if moments[0].Withheld != testCase.wantWithheld {
				t.Fatalf("Withheld = %v, want %v", moments[0].Withheld, testCase.wantWithheld)
			}

			if moments[0].Note == "" {
				t.Fatal("Note is empty, want every moment to say what happened")
			}
		})
	}
}

func TestStreamIsTheOrderedRecord(t *testing.T) {
	t.Parallel()

	now := reference()
	space := New(DefaultCapacity, StrongestChooser{}, nil)

	for index := range 3 {
		space.Bid(bid(talkChannel, float64(index+1)))

		_, err := space.Cycle(now.Add(time.Duration(index) * time.Minute))
		if err != nil {
			t.Fatalf("Cycle() error = %v", err)
		}
	}

	stream := space.Stream()
	if len(stream) != 3 {
		t.Fatalf("Stream() = %d moments, want 3", len(stream))
	}

	for index := 1; index < len(stream); index++ {
		if !stream[index].At.After(stream[index-1].At) {
			t.Fatalf("stream is out of order at %d, want one line running forwards", index)
		}
	}
}

func TestStreamIsACopy(t *testing.T) {
	t.Parallel()

	space := New(DefaultCapacity, StrongestChooser{}, nil)
	space.Bid(bid(talkChannel, 0.5))

	_, err := space.Cycle(reference())
	if err != nil {
		t.Fatalf("Cycle() error = %v", err)
	}

	stream := space.Stream()

	var tampered Moment

	stream[0] = tampered

	if got := space.Stream()[0].Winner.Channel; got != talkChannel {
		t.Fatalf("Stream()[0] channel = %q, want the record to be unchanged", got)
	}
}

func TestLast(t *testing.T) {
	t.Parallel()

	space := New(DefaultCapacity, StrongestChooser{}, nil)

	if _, ok := space.Last(); ok {
		t.Fatal("Last() ok = true, want false before anything has been conscious")
	}

	space.Bid(bid(idleChannel, 0.2))

	_, err := space.Cycle(reference())
	if err != nil {
		t.Fatalf("Cycle() error = %v", err)
	}

	last, ok := space.Last()
	if !ok {
		t.Fatal("Last() ok = false, want true")
	}

	if last.Winner.Channel != idleChannel {
		t.Fatalf("Last() channel = %q, want %q", last.Winner.Channel, idleChannel)
	}
}

func TestStrongestChooser(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		bids []Bid
		want int
	}{
		{name: "nothing to pick", bids: nil, want: -1},
		{
			name: "the heaviest wins",
			bids: []Bid{bid(idleChannel, 0.1), bid(talkChannel, 0.9)},
			want: 1,
		},
		{
			name: "the first of equals wins, so the answer is stable",
			bids: []Bid{bid(idleChannel, 0.5), bid(talkChannel, 0.5)},
			want: 0,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := (StrongestChooser{}).Choose(testCase.bids); got != testCase.want {
				t.Fatalf("Choose() = %d, want %d", got, testCase.want)
			}
		})
	}
}

func TestWeightedChooserWithNothingToPick(t *testing.T) {
	t.Parallel()

	if got := NewWeightedChooser(testSeed).Choose(nil); got != -1 {
		t.Fatalf("Choose() = %d, want -1", got)
	}
}

func TestWeightedChooserWithNoWeightAnywhere(t *testing.T) {
	t.Parallel()

	chooser := NewWeightedChooser(testSeed)
	bids := []Bid{bid(idleChannel, 0), bid(talkChannel, -1)}

	if got := chooser.Choose(bids); got != 0 {
		t.Fatalf("Choose() = %d, want the stable first pick when nothing carries weight", got)
	}
}

func TestWeightedChooserFavoursWeightWithoutGuaranteeingIt(t *testing.T) {
	t.Parallel()

	chooser := NewWeightedChooser(testSeed)
	heavy := bid(talkChannel, 0.9)
	light := bid(idleChannel, 0.1)

	wins := make(map[int]int, 2)
	for range sampleRuns {
		wins[chooser.Choose([]Bid{heavy, light})]++
	}

	if wins[0] <= wins[1] {
		t.Fatalf("wins = %v, want the heavier bid to win more often", wins)
	}

	// The lighter bid must still get through sometimes: a persona that always
	// attended to the largest number would just be a sort.
	if wins[1] == 0 {
		t.Fatalf("wins = %v, want the lighter bid to surface occasionally", wins)
	}

	if float64(wins[1])/float64(sampleRuns) < minorityShare {
		t.Fatalf("lighter bid won %d of %d, want a noticeable minority", wins[1], sampleRuns)
	}
}

func TestWeightedChooserIsRepeatable(t *testing.T) {
	t.Parallel()

	bids := []Bid{bid(talkChannel, 0.6), bid(idleChannel, 0.4), bid(actChannel, 0.2)}

	first := make([]int, 0, sampleRuns)
	chooser := NewWeightedChooser(testSeed)

	for range sampleRuns {
		first = append(first, chooser.Choose(bids))
	}

	replay := NewWeightedChooser(testSeed)
	for index := range first {
		if got := replay.Choose(bids); got != first[index] {
			t.Fatalf("replay at %d chose %d, want %d", index, got, first[index])
		}
	}
}

func TestCycleToleratesAChooserThatMisbehaves(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		index int
	}{
		{name: "index below the range", index: -1},
		{name: "index above the range", index: 99},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			space := New(DefaultCapacity, fixedChooser{index: testCase.index}, nil)
			space.Bid(bid(talkChannel, 0.5))

			moments, err := space.Cycle(reference())
			if err != nil {
				t.Fatalf("Cycle() error = %v", err)
			}

			if len(moments) != 0 {
				t.Fatalf("Cycle() returned %d moments, want none", len(moments))
			}
		})
	}
}

// fixedChooser always returns the same index, valid or not.
type fixedChooser struct {
	index int
}

func (f fixedChooser) Choose([]Bid) int { return f.index }

func TestCycleStopsWhenBidsRunOutBeforeCapacity(t *testing.T) {
	t.Parallel()

	space := New(4, StrongestChooser{}, nil)
	space.Bid(bid(talkChannel, 0.5))
	space.Bid(bid(idleChannel, 0.4))

	moments, err := space.Cycle(reference())
	if err != nil {
		t.Fatalf("Cycle() error = %v", err)
	}

	if len(moments) != 2 {
		t.Fatalf("Cycle() returned %d moments, want the 2 that were bid", len(moments))
	}
}

func TestSortedPendingIsStable(t *testing.T) {
	t.Parallel()

	space := New(DefaultCapacity, StrongestChooser{}, nil)
	space.Bid(Bid{
		Channel: talkChannel, Content: "b", Weight: 0.5,
		Act: "", Irreversible: false, Outward: false,
	})
	space.Bid(Bid{
		Channel: talkChannel, Content: "a", Weight: 0.5,
		Act: "", Irreversible: false, Outward: false,
	})
	space.Bid(Bid{
		Channel: actChannel, Content: "c", Weight: 0.5,
		Act: "", Irreversible: false, Outward: false,
	})

	sorted := space.sortedPending()

	wantChannels := []string{actChannel, talkChannel, talkChannel}
	wantContents := []string{"c", "a", "b"}

	for index := range sorted {
		if sorted[index].Channel != wantChannels[index] {
			t.Fatalf("bid %d channel = %q, want %q", index, sorted[index].Channel, wantChannels[index])
		}

		if sorted[index].Content != wantContents[index] {
			t.Fatalf("bid %d content = %q, want %q", index, sorted[index].Content, wantContents[index])
		}
	}
}

func TestPositive(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value float64
		want  float64
	}{
		{name: "positive is kept", value: 0.5, want: 0.5},
		{name: "zero is kept", value: 0, want: 0},
		{name: "negative is floored", value: -3, want: 0},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := positive(testCase.value); got != testCase.want {
				t.Fatalf("positive() = %v, want %v", got, testCase.want)
			}
		})
	}
}
