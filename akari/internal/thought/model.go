package thought

import (
	"context"
	"maps"

	"github.com/kizuna-org/akari/internal/emotion"
	"github.com/kizuna-org/akari/internal/goal"
	"github.com/kizuna-org/akari/internal/interest"
	"github.com/kizuna-org/akari/internal/meaning"
	"github.com/kizuna-org/akari/internal/memory"
	"github.com/kizuna-org/akari/internal/mind"
)

const ErrModel fault = "model runner requires a provider, recall capability and an initialized inner context"

// Request contains felt experience and selected recollections, never persona configuration.
type Request struct {
	Experience *mind.Experience
	Recalled   []memory.Fragment
	Query      memory.Query
	Prediction bool
}

// Interpretation leaves appraisal/meaning to the subject; it cannot change persona or runtime load.
type Interpretation struct {
	Appraisal *emotion.Appraisal
	Interests []interest.Focus
	Memory    *memory.Change
	Goals     *goal.Change
}

type Model interface {
	Interpret(ctx context.Context, request Request) (Interpretation, error)
}

// Mode is issued by runtime. Subconscious work has zero load; the model cannot set this value.
type Mode struct {
	Prediction    bool
	ConsciousLoad float64
}

// FromModel prepares an unadopted proposal. Only subsequent Accept can affect the shared self.
// Every inner part is a dependency because this request reads the whole experience.
func FromModel(model Model, reader memory.Reader, query memory.Query, mode Mode) (Runner, error) {
	if model == nil || reader == nil || !meaning.Finite(mode.ConsciousLoad) || mode.ConsciousLoad < 0 {
		return nil, ErrModel
	}

	query.Cue = query.Cue.Clone()

	return func(ctx context.Context, snapshot mind.Snapshot) (mind.Proposal, error) {
		if snapshot.Experience == nil {
			return mind.Proposal{}, ErrModel
		}

		recallQuery := query
		recallQuery.Cue = query.Cue.Clone()

		recalled, err := reader.Recall(ctx, recallQuery)
		if err != nil {
			return mind.Proposal{}, err
		}

		// Give the adapter a separate copy; it cannot change recorded dependency versions.
		requestQuery := query
		requestQuery.Cue = query.Cue.Clone()

		interpretation, err := model.Interpret(ctx, Request{
			Experience: snapshot.Experience.Clone(), Recalled: memory.CloneFragments(recalled),
			Query: requestQuery, Prediction: mode.Prediction,
		})
		if err != nil {
			return mind.Proposal{}, err
		}

		return mind.Proposal{
			ID: "assigned by supervisor", Reads: nil, Writes: nil, Prediction: mode.Prediction,
			Inner: (&mind.Update{
				Expected: maps.Clone(snapshot.Experience.Versions), Appraisal: interpretation.Appraisal,
				Interests: interpretation.Interests, Memory: interpretation.Memory, Goals: interpretation.Goals,
				Load: mode.ConsciousLoad, Rest: 0, Advance: false,
			}).Clone(),
		}, nil
	}, nil
}
