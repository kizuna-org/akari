package mind

import (
	"errors"
	"strconv"
	"testing"
)

func TestNew(t *testing.T) {
	t.Parallel()

	for _, capacity := range []int{-1, 0, 1} {
		t.Run(strconv.Itoa(capacity), func(t *testing.T) {
			t.Parallel()

			workspace, err := New(capacity)
			if (err != nil) != (capacity < 1) || (workspace == nil) != (capacity < 1) {
				t.Fatalf("New(%d) = %v, %v", capacity, workspace, err)
			}
		})
	}
}

func TestCommit(t *testing.T) {
	const updated = "new"

	t.Parallel()

	tests := []struct {
		name     string
		proposal Proposal
		want     error
	}{
		{name: "missing identity", proposal: Proposal{ID: "", Reads: nil, Writes: nil}, want: ErrProposal},
		{name: "blind write",
			proposal: Proposal{ID: "p", Reads: nil, Writes: map[string]string{"a": "bad"}},
			want:     ErrProposal},
		{name: "stale read", proposal: Proposal{ID: "p", Reads: map[string]uint64{"a": 0}, Writes: nil}, want: ErrConflict},
		{name: "unrelated change is valid",
			proposal: Proposal{ID: "p", Reads: map[string]uint64{"b": 0}, Writes: map[string]string{"b": updated}},
			want:     nil},
		{name: "update existing context",
			proposal: Proposal{ID: "p", Reads: map[string]uint64{"a": 1}, Writes: map[string]string{"a": updated}},
			want:     nil},
		{name: "resource bound",
			proposal: Proposal{ID: "p", Reads: map[string]uint64{"b": 0, "c": 0},
				Writes: map[string]string{"b": updated, "c": updated}},
			want: ErrCapacity},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			workspace := newWorkspace(t)
			initial := Proposal{ID: "initial", Reads: map[string]uint64{"a": 0}, Writes: map[string]string{"a": "old"}}

			_, err := workspace.Commit(initial)
			if err != nil {
				t.Fatal(err)
			}

			revision, err := workspace.Commit(test.proposal)
			if !errors.Is(err, test.want) {
				t.Fatalf("Commit error = %v; want %v", err, test.want)
			}

			if test.want != nil && (revision != 1 || workspace.Snapshot().Items["a"].Content != "old") {
				t.Fatal("rejected proposal changed shared state")
			}

			if test.want == nil && revision != 2 {
				t.Fatalf("accepted revision = %d", revision)
			}
		})
	}
}

func TestOwnershipAndBroadcast(t *testing.T) {
	t.Parallel()

	for _, watchers := range []int{0, 1, 2} {
		t.Run(strconv.Itoa(watchers), func(t *testing.T) {
			t.Parallel()

			workspace := newWorkspace(t)
			wakes := make([]<-chan struct{}, 0, watchers)
			stops := make([]func(), 0, watchers)

			for range watchers {
				wake, stop := workspace.Watch()
				wakes = append(wakes, wake)
				stops = append(stops, stop)
			}

			proposal := Proposal{ID: "first", Reads: map[string]uint64{"a": 0}, Writes: map[string]string{"a": "original"}}
			cloned := proposal.Clone()
			proposal.Reads["a"] = 999
			proposal.Writes["a"] = "mutated"

			_, err := workspace.Commit(cloned)
			if err != nil {
				t.Fatal(err)
			}

			snapshot := workspace.Snapshot()
			snapshot.Items["a"] = Item{Content: "mutated", Version: 999}
			cloned.ID = "second"
			cloned.Reads["a"] = 1
			cloned.Writes["a"] = "latest"

			_, err = workspace.Commit(cloned)
			if err != nil || workspace.Snapshot().Items["a"].Content != "latest" {
				t.Fatalf("ownership or commit failed: %v", err)
			}

			for index, wake := range wakes {
				select {
				case <-wake:
				default:
					t.Fatal("subscriber was not notified")
				}

				stops[index]()
				stops[index]()

				if _, open := <-wake; open {
					t.Fatal("subscription was not closed")
				}
			}
		})
	}
}

func newWorkspace(t *testing.T) *Workspace {
	t.Helper()

	workspace, err := New(2)
	if err != nil {
		t.Fatal(err)
	}

	return workspace
}
