package mind

import (
	"context"
	"errors"
	"maps"
	"math"
	"reflect"
	"testing"
	"time"

	"github.com/kizuna-org/akari/internal/emotion"
	"github.com/kizuna-org/akari/internal/goal"
	"github.com/kizuna-org/akari/internal/interest"
	"github.com/kizuna-org/akari/internal/meaning"
	"github.com/kizuna-org/akari/internal/memory"
	"github.com/kizuna-org/akari/internal/persona"
)

const capacityCase = "capacity"

func TestInnerContinuity(t *testing.T) {
	t.Parallel()

	for _, reactivity := range []float64{0, 1} {
		t.Run(map[float64]string{0: "quiet", 1: "reactive"}[reactivity], func(t *testing.T) {
			t.Parallel()

			config := persona.Default()
			config.Vector[persona.Reactivity] = reactivity
			instant := time.Unix(100, 0)
			workspace := innerWorkspace(t, config, 100, func() time.Time { return instant })
			config.Vector[persona.Reactivity] = 0.5
			config.Likes[0] = "changed after startup"
			update := freshUpdate(workspace)
			update.Appraisal = &emotion.Appraisal{
				Feelings: []emotion.Feeling{{Meaning: semantic("delight"), Strength: 1}}, Others: nil,
				Readiness: []emotion.Feeling{{Meaning: semantic("step back"), Strength: 1}}, MoodDelta: emotion.Mood{1, 0, 0},
			}
			update.Interests = []interest.Focus{{ID: "topic", Target: semantic("topic"), Affinity: 1}}
			update.Memory = &memory.Change{Context: nil, Working: nil, Remember: []memory.Fragment{fragment()},
				Recalled: nil, Agreements: []memory.Agreement{{Conversation: "private", Recipients: []string{"alice"}}}}
			update.Goals = &goal.Change{Desires: nil, Adopt: map[string]goal.Intention{"promise": intention()}, End: nil}
			update.Load = 1
			commitInner(t, workspace, update, nil)
			experience := workspace.Snapshot().Experience

			if experience.Emotion.Feelings[0].Strength != reactivity || experience.Fatigue != 0.5 ||
				len(experience.Goals.Intentions) != 1 || len(experience.Emotion.Readiness) != 1 {
				t.Fatal("fixed persona was replaced or adopted inner state was not retained")
			}

			recalled, err := workspace.Recall(t.Context(), query(instant))
			assertInnerError(t, err, nil)

			checkFeltMemory(t, workspace, recalled[0], reactivity)

			checkpoint, err := workspace.Export()
			assertInnerError(t, err, nil)

			checkpoint.Persona.Vector[persona.Reactivity] = 0.5
			checkpoint.Archive.Day[0].Meaning.Vector[0] = 0
			experience.Emotion.Feelings[0].Meaning.Vector[0] = 0
			experience.Goals.Intentions["promise"].Claims["work"] = "forged"
			experience.Interests[0].Target.Vector[0] = 0
			experience.Agreements[0].Recipients[0] = "bob"
			update.Appraisal.Feelings[0].Meaning.Vector[0] = 0
			update = freshUpdate(workspace)
			update.Memory = &memory.Change{Context: recalled, Working: recalled, Remember: nil,
				Recalled: []string{recalled[0].ID}, Agreements: nil}
			commitInner(t, workspace, update.Clone(), nil)

			latest := workspace.Snapshot().Experience
			checkNextConversation(t, latest)

			instant = instant.Add(time.Hour)
			update = freshUpdate(workspace)
			update.Advance = true
			update.Rest = time.Hour
			commitInner(t, workspace, update, nil)

			latest = workspace.Snapshot().Experience
			if latest.Fatigue != 0 || latest.Emotion.Feelings[0].Strength != reactivity/2 ||
				len(latest.Goals.Intentions) != 1 {
				t.Fatal("rest, decay or persistent intention failed")
			}
		})
	}
}

func TestAtomicInnerRejection(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name   string
		change func(*Update)
		want   error
	}{
		{name: "missing dependency", change: func(update *Update) { delete(update.Expected, Emotion) }, want: ErrProposal},
		{name: "stale dependency", change: func(update *Update) { update.Expected[Emotion] = 99 }, want: ErrConflict},
		{name: "unknown dependency", change: func(update *Update) { update.Expected["invented"] = 0 }, want: ErrConflict},
		{name: "invalid feeling", change: func(update *Update) { update.Appraisal.MoodDelta[0] = math.NaN() },
			want: emotion.ErrAppraisal},
		{name: "invalid interest", change: func(update *Update) {
			update.Interests = []interest.Focus{{ID: "", Target: semantic("x"), Affinity: 1}}
		}, want: interest.ErrFocus},
		{name: "invalid memory", change: func(update *Update) {
			update.Memory = new(memory.Change)
			update.Memory.Recalled = []string{"missing"}
		}, want: memory.ErrMemory},
		{name: "invalid intention", change: func(update *Update) {
			update.Goals = new(goal.Change)
			update.Goals.Adopt = map[string]goal.Intention{"bad": {
				Meaning: semantic("x"), Claims: nil, Steps: nil, Next: -1, Paused: false}}
		}, want: goal.ErrGoal},
		{name: capacityCase, change: func(update *Update) {
			update.Interests = make([]interest.Focus, 2)
			update.Interests[0] = interest.Focus{ID: "a", Target: semantic("x"), Affinity: 1}
			update.Interests[1] = interest.Focus{ID: "b", Target: semantic("x"), Affinity: 1}
		}, want: ErrCapacity},
		{name: "excess conscious load", change: func(update *Update) { update.Load = 2 }, want: ErrLoad},
		{name: "negative rest", change: func(update *Update) { update.Rest = -1 }, want: ErrLoad},
		{name: "reversed clock", change: func(*Update) {}, want: emotion.ErrAppraisal},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			instant := time.Unix(100, 0)
			workspace := innerWorkspace(t, persona.Default(), 1, func() time.Time { return instant })
			before := workspace.Snapshot()

			if test.name == "reversed clock" {
				instant = instant.Add(-time.Hour)
			}

			update := freshUpdate(workspace)
			update.Appraisal = &emotion.Appraisal{Feelings: nil, Others: nil, Readiness: nil, MoodDelta: emotion.Mood{1, 0, 0}}
			test.change(update)
			commitInner(t, workspace, update, test.want)

			if !reflect.DeepEqual(before, workspace.Snapshot()) {
				t.Fatal("rejected interpretation partially changed mood, memory, fatigue or shared revision")
			}
		})
	}
}

func TestIndependentInnerVersions(t *testing.T) {
	t.Parallel()

	for _, samePart := range []bool{false, true} {
		t.Run(map[bool]string{false: "independent", true: "changed"}[samePart], func(t *testing.T) {
			t.Parallel()

			workspace := innerWorkspace(t, persona.Default(), 10, func() time.Time { return time.Unix(100, 0) })
			old := freshUpdate(workspace)
			old.Memory = new(memory.Change)
			old.Expected = map[Part]uint64{Memory: 0}
			current := freshUpdate(workspace)

			if samePart {
				current.Memory = new(memory.Change)
			} else {
				current.Advance = true
			}

			commitInner(t, workspace, current, nil)

			want := map[bool]error{false: nil, true: ErrConflict}[samePart]
			commitInner(t, workspace, old, want)
		})
	}
}

func TestInnerConfiguration(t *testing.T) {
	t.Parallel()

	for _, reason := range []string{"profile", "settings", "clock", capacityCase, "absent"} {
		t.Run(reason, func(t *testing.T) {
			t.Parallel()

			config := persona.Default()
			settings := innerSettings(func() time.Time { return time.Unix(100, 0) }, 10)
			capacity := 2
			want := error(ErrInner)

			switch reason {
			case "profile":
				config.ID = ""
				want = persona.ErrConfig
			case "settings":
				settings.Clock = nil
			case "clock":
				settings.Clock = func() time.Time { return time.Time{} }
			case capacityCase:
				capacity = 0
				want = ErrConfig
			case "absent":
				workspace := newWorkspace(t)
				_, err := workspace.Export()
				assertInnerError(t, err, ErrInner)
				_, err = workspace.Recall(t.Context(), query(time.Unix(100, 0)))
				assertInnerError(t, err, ErrInner)
				ctx, cancel := context.WithCancel(t.Context())
				cancel()

				_, err = workspace.Recall(ctx, query(time.Unix(100, 0)))
				assertInnerError(t, err, context.Canceled)
				commitInner(t, workspace, new(Update), ErrInner)

				if workspace.AllowsDisclosure(nil, "alice") {
					t.Fatal("uninitialized policy allowed an attributed disclosure")
				}

				return
			}

			_, err := NewWithInner(capacity, config, settings)
			assertInnerError(t, err, want)
		})
	}
}

func innerWorkspace(t *testing.T, config persona.Config, contents int, clock func() time.Time) *Workspace {
	t.Helper()

	workspace, err := NewWithInner(2, config, innerSettings(clock, contents))
	assertInnerError(t, err, nil)

	return workspace
}

func innerSettings(clock func() time.Time, contents int) Settings {
	return Settings{Contents: contents, Clock: clock, RestUnit: time.Hour,
		Dynamics: emotion.Dynamics{EmotionHalfLife: time.Hour, MoodHalfLife: time.Hour}}
}

func freshUpdate(workspace *Workspace) *Update {
	update := new(Update)
	update.Expected = maps.Clone(workspace.Snapshot().Experience.Versions)

	return update
}

func commitInner(t *testing.T, workspace *Workspace, update *Update, want error) {
	t.Helper()

	_, err := workspace.Commit(Proposal{ID: "test", Reads: nil, Writes: nil, Inner: update, Prediction: false})
	assertInnerError(t, err, want)
}

func semantic(text string) meaning.Point {
	return meaning.Point{Text: text, Space: "test", Vector: []float64{1}}
}

func fragment() memory.Fragment {
	return memory.Fragment{ID: "memory", Meaning: semantic("experience"), OccurredAt: time.Unix(100, 0),
		Emotion: 0, Will: 1, Strength: 1, Accesses: 0, Sources: []string{"private"}}
}

func intention() goal.Intention {
	return goal.Intention{Meaning: semantic("keep promise"), Claims: map[string]string{"work": "finish"},
		Steps: []meaning.Point{semantic("continue")}, Next: 0, Paused: true}
}

func query(instant time.Time) memory.Query {
	return memory.Query{Cue: semantic("cue"), At: instant, Freshness: time.Hour, Limit: 1}
}

func assertInnerError(t *testing.T, got, want error) {
	t.Helper()

	if !errors.Is(got, want) {
		t.Fatalf("error = %v; want %v", got, want)
	}
}

func checkNextConversation(t *testing.T, latest *Experience) {
	t.Helper()

	if latest.Emotion.Feelings[0].Meaning.Vector[0] != 1 ||
		latest.Goals.Intentions["promise"].Claims["work"] != "finish" || len(latest.Working) != 1 || latest.Fatigue != 0.5 {
		t.Fatal("next conversation reset or aliased the shared self")
	}
}
func checkFeltMemory(t *testing.T, workspace *Workspace, fragment memory.Fragment, strength float64) {
	t.Helper()

	if fragment.Emotion != strength || workspace.AllowsDisclosure(fragment.Sources, "bob") ||
		!workspace.AllowsDisclosure(fragment.Sources, "alice") {
		t.Fatal("felt intensity or conversation confidentiality was lost")
	}
}
