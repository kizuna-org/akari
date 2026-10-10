package persistence

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kizuna-org/akari/internal/continuity"
	"github.com/kizuna-org/akari/internal/memory"
	"github.com/kizuna-org/akari/internal/mind"
	"github.com/kizuna-org/akari/internal/persona"
)

func savedFrame(sequence uint64) continuity.Frame {
	return continuity.Frame{Schema: 1, Identity: "installation", Sequence: sequence,
		Checkpoint: mind.Checkpoint{Schema: 1, Persona: persona.Default(),
			Snapshot: mind.Snapshot{Revision: 0, Items: nil},
			Archive:  memory.State{Context: nil, Working: nil, Day: nil, Sleeping: nil, Agreements: nil}},
		Outbox: make(map[string]continuity.Record)}
}

func TestStoreRejectsOversizedOrUnencodableFrames(t *testing.T) {
	t.Parallel()

	for _, onDisk := range []bool{false, true} {
		t.Run(map[bool]string{false: "save limit", true: "load limit"}[onDisk], func(t *testing.T) {
			t.Parallel()

			file, err := New(t.TempDir())
			assertError(t, err, nil)

			if onDisk {
				data := "{}" + strings.Repeat(" ", maxBytes)
				assertError(t, os.WriteFile(filepath.Join(file.directory, "state.json"), []byte(data), privateFile), nil)
				_, err = file.Load(t.Context())
				assertError(t, err, continuity.ErrState)
			} else {
				frame := savedFrame(1)
				frame.Identity = strings.Repeat("x", maxBytes)
				assertError(t, file.Save(t.Context(), 0, frame), continuity.ErrState)
			}
		})
	}
}

func assertError(t *testing.T, actual, expected error) {
	t.Helper()

	if !errors.Is(actual, expected) {
		t.Fatalf("error = %v, want %v", actual, expected)
	}
}

func TestAtomicReplacementBoundaries(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name     string
		phase    string
		cancel   bool
		sequence uint64
		want     error
	}{
		{name: "normal", phase: "", cancel: false, sequence: 2, want: nil},
		{name: "crash before rename", phase: beforeRename, cancel: false, sequence: 1, want: continuity.ErrState},
		{name: "crash after rename", phase: "afterRename", cancel: false, sequence: 2, want: continuity.ErrState},
		{name: "cancel before rename", phase: beforeRename, cancel: true, sequence: 1, want: context.Canceled},
		{name: "cancel after rename", phase: "afterRename", cancel: true, sequence: 2, want: nil},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			file, err := New(t.TempDir())
			assertError(t, err, nil)
			assertError(t, file.Save(t.Context(), 0, savedFrame(1)), nil)

			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()

			file.hook = func(phase string) error {
				if phase != test.phase {
					return nil
				}

				if test.cancel {
					cancel()

					if phase == beforeRename {
						return ctx.Err()
					}

					return nil
				}

				return continuity.ErrState
			}
			assertError(t, file.Save(ctx, 1, savedFrame(2)), test.want)
			other, err := New(file.directory)
			assertError(t, err, nil)
			loaded, err := other.Load(t.Context())
			assertError(t, err, nil)

			if loaded.Sequence != test.sequence {
				t.Fatal("partial frame became visible")
			}

			info, err := os.Stat(filepath.Join(file.directory, "state.json"))
			assertError(t, err, nil)

			if info.Mode().Perm() != privateFile {
				t.Fatal("private state permissions changed")
			}

			leftovers, err := filepath.Glob(filepath.Join(file.directory, ".state-*"))
			assertError(t, err, nil)

			if len(leftovers) != 0 {
				t.Fatal("temporary files not cleaned up")
			}
		})
	}
}

func TestStoreCASAndCorruption(t *testing.T) {
	t.Parallel()

	for _, test := range []string{"missing", "stale", "sequence", "identity", "truncated", "unknown field", "trailing"} {
		t.Run(test, func(t *testing.T) {
			t.Parallel()

			file, err := New(t.TempDir())
			assertError(t, err, nil)

			if test == "missing" {
				_, err = file.Load(t.Context())
				assertError(t, err, continuity.ErrAbsent)

				return
			}

			assertError(t, file.Save(t.Context(), 0, savedFrame(1)), nil)

			next := savedFrame(2)

			switch test {
			case "stale":
				assertError(t, file.Save(t.Context(), 0, next), continuity.ErrConflict)
			case "sequence":
				next.Sequence++
				assertError(t, file.Save(t.Context(), 1, next), continuity.ErrConflict)
			case "identity":
				next.Identity = "other"
				assertError(t, file.Save(t.Context(), 1, next), continuity.ErrConflict)
			default:
				data := map[string]string{"truncated": "{", "unknown field": "{\"newField\":1}",
					"trailing": "{} {}"}[test]
				assertError(t, os.WriteFile(filepath.Join(file.directory, "state.json"), []byte(data), privateFile), nil)
				_, err = file.Load(t.Context())
				assertError(t, err, continuity.ErrState)
				assertError(t, file.Save(t.Context(), 1, next), continuity.ErrState)
			}
		})
	}
}

func TestStoreLockAndCancellation(t *testing.T) {
	t.Parallel()

	for _, canceled := range []bool{false, true} {
		t.Run(map[bool]string{false: "locked", true: "canceled"}[canceled], func(t *testing.T) {
			t.Parallel()

			file, err := New(t.TempDir())
			assertError(t, err, nil)
			unlock, err := file.lock(t.Context())
			assertError(t, err, nil)

			defer unlock()

			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()

			want := error(continuity.ErrBusy)

			if canceled {
				cancel()

				want = context.Canceled
			}

			_, err = file.Load(ctx)
			assertError(t, err, want)
			assertError(t, file.Save(ctx, 0, savedFrame(1)), want)
		})
	}
}
