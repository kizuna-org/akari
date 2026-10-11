package thought

import (
	"context"
	"io"
	"reflect"
	"testing"
	"time"

	"github.com/kizuna-org/akari/internal/emotion"
	"github.com/kizuna-org/akari/internal/goal"
	"github.com/kizuna-org/akari/internal/interest"
	"github.com/kizuna-org/akari/internal/meaning"
	"github.com/kizuna-org/akari/internal/memory"
	"github.com/kizuna-org/akari/internal/mind"
	"github.com/kizuna-org/akari/internal/persona"
)

type modelFunc func(context.Context, Request) (Interpretation, error)

func (model modelFunc) Interpret(ctx context.Context, request Request) (Interpretation, error) {
	return model(ctx, request)
}

type recallFunc func(context.Context, memory.Query) ([]memory.Fragment, error)

func (reader recallFunc) Recall(ctx context.Context, query memory.Query) ([]memory.Fragment, error) {
	return reader(ctx, query)
}

func TestAdoptedModelExperience(t *testing.T) {
	t.Parallel()

	for _, prediction := range []bool{false, true} {
		t.Run(map[bool]string{false: "actual", true: "prediction"}[prediction], func(t *testing.T) {
			t.Parallel()

			workspace := modelWorkspace(t)
			before := workspace.Snapshot()
			model := modelFunc(func(_ context.Context, request Request) (Interpretation, error) {
				if len(request.Recalled) != 0 || request.Query.Cue.Vector[0] != 1 || request.Prediction != prediction {
					return Interpretation{}, ErrModel
				}

				request.Experience.Versions[mind.Emotion] = 999
				request.Query.Cue.Vector[0] = 0

				return Interpretation{
					Appraisal: &emotion.Appraisal{Feelings: []emotion.Feeling{{Meaning: modelPoint(), Strength: 1}},
						Others: nil, Readiness: nil, MoodDelta: emotion.Mood{-1, 0, 0}},
					Interests: []interest.Focus{{ID: "avoid", Target: modelPoint(), Affinity: -1}},
					Memory: &memory.Change{Context: nil, Working: nil,
						Remember: []memory.Fragment{{ID: "experience", Meaning: modelPoint(), OccurredAt: time.Unix(100, 0),
							Emotion: 0, Will: 1, Strength: 1, Accesses: 0, Sources: []string{"conversation"}}},
						Recalled: nil, Agreements: nil},
					Goals: &goal.Change{Desires: map[string]meaning.Point{"self-generated": modelPoint()}, Adopt: nil, End: nil},
				}, nil
			})
			query := memory.Query{Cue: modelPoint(), At: time.Unix(100, 0), Freshness: time.Hour, Limit: 2}
			runner, err := FromModel(model, workspace, query, Mode{Prediction: prediction, ConsciousLoad: 1})
			assertError(t, err, nil)

			query.Cue.Vector[0] = 0
			supervisor, err := New(t.Context(), workspace, 2)
			assertError(t, err, nil)

			t.Cleanup(func() {
				ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
				defer cancel()

				assertError(t, supervisor.Close(ctx), nil)
			})

			_, err = supervisor.Start("interpreter", runner)
			assertError(t, err, nil)
			result := next(t, supervisor)

			if !reflect.DeepEqual(before, workspace.Snapshot()) {
				t.Fatal("an unadopted model call changed inner state or fatigue")
			}

			result.Proposal.Inner.Expected[mind.Emotion] = 999
			result.Proposal.Inner.Load = 999
			result.Proposal.Prediction = false
			_, err = supervisor.Accept(result.Token)
			want := map[bool]error{false: nil, true: mind.ErrPrediction}[prediction]
			assertError(t, err, want)
			checkModelAdoption(t, workspace, before, prediction)
		})
	}
}

func TestModelFailures(t *testing.T) {
	t.Parallel()

	for _, reason := range []string{"missing model", "missing reader", "invalid load", "no inner", "recall", "interpret"} {
		t.Run(reason, func(t *testing.T) {
			t.Parallel()

			workspace := modelWorkspace(t)

			var (
				model Model = modelFunc(func(context.Context, Request) (Interpretation, error) {
					return Interpretation{}, io.ErrUnexpectedEOF
				})
				reader memory.Reader = workspace
			)

			want := error(ErrModel)
			mode := Mode{Prediction: false, ConsciousLoad: 1}

			switch reason {
			case "missing model":
				model = nil
			case "missing reader":
				reader = nil
			case "invalid load":
				mode.ConsciousLoad = -1
			case "recall":
				reader = recallFunc(func(context.Context, memory.Query) ([]memory.Fragment, error) {
					return nil, io.ErrClosedPipe
				})
				want = io.ErrClosedPipe
			case "interpret":
				want = io.ErrUnexpectedEOF
			}

			runner, err := FromModel(model, reader,
				memory.Query{Cue: modelPoint(), At: time.Unix(100, 0), Freshness: time.Hour, Limit: 1},
				mode)
			if runner == nil {
				assertError(t, err, want)

				return
			}

			snapshot := workspace.Snapshot()

			if reason == "no inner" {
				snapshot.Experience = nil
			}

			_, err = runner(t.Context(), snapshot)
			assertError(t, err, want)
		})
	}
}

func TestParallelInnerAdoption(t *testing.T) {
	t.Parallel()

	for _, canceled := range []bool{false, true} {
		name := map[bool]string{false: "independent", true: "canceled"}[canceled]
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			workspace := modelWorkspace(t)
			supervisor, err := New(t.Context(), workspace, 2)
			assertError(t, err, nil)

			t.Cleanup(func() {
				ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
				defer cancel()

				assertError(t, supervisor.Close(ctx), nil)
			})

			started := make(chan struct{}, 1)
			release := make(chan struct{})
			_, err = supervisor.Start("recall", func(ctx context.Context, snapshot mind.Snapshot) (mind.Proposal, error) {
				started <- struct{}{}

				select {
				case <-release:
				case <-ctx.Done():
					return mind.Proposal{}, ctx.Err()
				}

				change := new(memory.Change)
				change.Remember = []memory.Fragment{{ID: "independent", Meaning: modelPoint(), OccurredAt: time.Unix(100, 0),
					Emotion: 0, Will: 1, Strength: 1, Accesses: 0, Sources: nil}}
				update := new(mind.Update)
				update.Expected = map[mind.Part]uint64{mind.Memory: snapshot.Experience.Versions[mind.Memory]}
				update.Memory = change

				return mind.Proposal{ID: fixtureProposalID, Reads: nil, Writes: nil, Inner: update, Prediction: false}, nil
			})
			assertError(t, err, nil)
			<-started

			_, err = supervisor.Start("interaction", func(_ context.Context, snapshot mind.Snapshot) (mind.Proposal, error) {
				update := new(mind.Update)
				update.Expected = map[mind.Part]uint64{mind.Emotion: snapshot.Experience.Versions[mind.Emotion]}
				update.Appraisal = &emotion.Appraisal{Feelings: nil, Others: nil, Readiness: nil,
					MoodDelta: emotion.Mood{-1, 0, 0}}
				update.Load = 1

				return mind.Proposal{ID: fixtureProposalID, Reads: nil, Writes: nil, Inner: update, Prediction: false}, nil
			})
			assertError(t, err, nil)
			fast := next(t, supervisor)
			_, err = supervisor.Accept(fast.Token)
			assertError(t, err, nil)

			if canceled {
				supervisor.Cancel("recall")
			}

			close(release)

			slow := next(t, supervisor)
			_, err = supervisor.Accept(slow.Token)
			want := map[bool]error{false: nil, true: context.Canceled}[canceled]
			assertError(t, err, want)
			recalled, err := workspace.Recall(t.Context(),
				memory.Query{Cue: modelPoint(), At: time.Unix(100, 0), Freshness: time.Hour, Limit: 2})
			assertError(t, err, nil)

			if len(recalled) != map[bool]int{false: 1, true: 0}[canceled] || workspace.Snapshot().Experience.Fatigue != 0.5 {
				t.Fatal("independent/canceled result changed the wrong shared inner state")
			}
		})
	}
}

func TestConsciousContentLoad(t *testing.T) {
	t.Parallel()

	for _, load := range []float64{0, 1} {
		t.Run(map[float64]string{0: "subconscious", 1: "conscious"}[load], func(t *testing.T) {
			t.Parallel()

			workspace := modelWorkspace(t)
			model := modelFunc(func(context.Context, Request) (Interpretation, error) {
				return Interpretation{Appraisal: &emotion.Appraisal{Feelings: nil, Others: nil, Readiness: nil,
					MoodDelta: emotion.Mood{-1, 0, 0}}, Interests: nil, Memory: nil, Goals: nil}, nil
			})
			runner, err := FromModel(model, workspace,
				memory.Query{Cue: modelPoint(), At: time.Unix(100, 0), Freshness: time.Hour, Limit: 1},
				Mode{Prediction: false, ConsciousLoad: load})
			assertError(t, err, nil)
			proposal, err := runner(t.Context(), workspace.Snapshot())
			assertError(t, err, nil)
			_, err = workspace.Commit(proposal)
			assertError(t, err, nil)

			if workspace.Snapshot().Experience.Fatigue != load/2 {
				t.Fatal("model call count was mistaken for adopted conscious content")
			}
		})
	}
}

func checkModelAdoption(t *testing.T, workspace *mind.Workspace, before mind.Snapshot, prediction bool) {
	t.Helper()

	if prediction {
		if !reflect.DeepEqual(before, workspace.Snapshot()) {
			t.Fatal("prediction became actual emotion, memory, interest, desire or fatigue")
		}

		return
	}

	experience := workspace.Snapshot().Experience
	if experience.Fatigue != 0.5 || experience.Interests[0].Affinity >= 0 ||
		len(experience.Goals.Desires) != 1 || experience.Emotion.Mood[0] >= before.Experience.Emotion.Mood[0] {
		t.Fatal("adopted experience did not influence the shared self")
	}

	recalled, err := workspace.Recall(t.Context(),
		memory.Query{Cue: modelPoint(), At: time.Unix(100, 0), Freshness: time.Hour, Limit: 1})
	assertError(t, err, nil)

	if len(recalled) != 1 || recalled[0].Emotion != 0.6 {
		t.Fatal("actual felt strength did not enter Kiseki")
	}
}

func modelWorkspace(t *testing.T) *mind.Workspace {
	t.Helper()

	workspace, err := mind.NewWithInner(2, persona.Default(), mind.Settings{
		Contents: 100, Dynamics: emotion.Dynamics{EmotionHalfLife: time.Hour, MoodHalfLife: time.Hour},
		RestUnit: time.Hour, Clock: func() time.Time { return time.Unix(100, 0) },
	})
	assertError(t, err, nil)

	return workspace
}

func modelPoint() meaning.Point {
	return meaning.Point{Text: "interpreted meaning", Space: "test", Vector: []float64{1}}
}
