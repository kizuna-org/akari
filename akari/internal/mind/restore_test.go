package mind

import (
	"math"
	"reflect"
	"testing"
	"time"

	"github.com/kizuna-org/akari/internal/emotion"
	"github.com/kizuna-org/akari/internal/goal"
	"github.com/kizuna-org/akari/internal/memory"
	"github.com/kizuna-org/akari/internal/persona"
)

func TestRestoreCheckpoint(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name   string
		change func(*Checkpoint)
		want   error
	}{
		{name: "valid", change: func(*Checkpoint) {}, want: nil},
		{name: "schema", change: func(saved *Checkpoint) { saved.Schema++ }, want: ErrCheckpoint},
		{name: "persona identity", change: func(saved *Checkpoint) { saved.Persona.ID = "other" }, want: ErrCheckpoint},
		{name: "persona value", change: func(saved *Checkpoint) { saved.Persona.Vector[persona.Reactivity] = 0 },
			want: ErrCheckpoint},
		{name: "missing experience", change: func(saved *Checkpoint) { saved.Snapshot.Experience = nil },
			want: ErrCheckpoint},
		{name: "missing part", change: func(saved *Checkpoint) { delete(saved.Snapshot.Experience.Versions, Emotion) },
			want: ErrCheckpoint},
		{name: "unknown part", change: func(saved *Checkpoint) {
			delete(saved.Snapshot.Experience.Versions, Emotion)
			saved.Snapshot.Experience.Versions[Part("unknown")] = 0
		}, want: ErrCheckpoint},
		{name: "future part", change: func(saved *Checkpoint) { saved.Snapshot.Experience.Versions[Emotion] = 99 },
			want: ErrCheckpoint},
		{name: "invalid item", change: func(saved *Checkpoint) { saved.Snapshot.Items[""] = Item{Content: "x", Version: 0} },
			want: ErrCheckpoint},
		{name: "future item", change: func(saved *Checkpoint) {
			saved.Snapshot.Items["future"] = Item{Content: "x", Version: 99}
		}, want: ErrCheckpoint},
		{name: "fatigue", change: func(saved *Checkpoint) { saved.Snapshot.Experience.Fatigue = math.NaN() },
			want: ErrCheckpoint},
		{name: "timestamp", change: func(saved *Checkpoint) { saved.Snapshot.Experience.Emotion.At = time.Time{} },
			want: ErrCheckpoint},
		{name: "mood", change: func(saved *Checkpoint) { saved.Snapshot.Experience.Emotion.Mood[0] = 2 },
			want: ErrCheckpoint},
		{name: "memory mismatch", change: func(saved *Checkpoint) { saved.Archive.Context = []memory.Fragment{fragment()} },
			want: ErrCheckpoint},
		{name: "invalid memory", change: func(saved *Checkpoint) {
			saved.Archive.Day = []memory.Fragment{fragment(), fragment()}
		}, want: ErrCheckpoint},
		{name: "invalid intention", change: func(saved *Checkpoint) {
			saved.Snapshot.Experience.Goals.Intentions = map[string]goal.Intention{"invalid": {
				Meaning: semantic("invalid"), Claims: nil, Steps: nil, Next: -1, Paused: false,
			}}
		}, want: ErrCheckpoint},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			config := persona.Default()
			workspace := innerWorkspace(t, config, 100, func() time.Time { return time.Unix(100, 0) })
			commitInner(t, workspace, freshUpdate(workspace), nil)
			saved, err := workspace.Export()
			assertInnerError(t, err, nil)
			test.change(&saved)
			restored, err := Restore(saved, config, 100, Settings{Contents: 100, RestUnit: time.Hour,
				Dynamics: emotion.Dynamics{EmotionHalfLife: time.Hour, MoodHalfLife: time.Hour},
				Clock:    func() time.Time { return time.Unix(100, 0) }})
			assertInnerError(t, err, test.want)

			if err == nil && !reflect.DeepEqual(workspace.Snapshot(), restored.Snapshot()) {
				t.Fatal("restored snapshot changed")
			}
		})
	}
}
