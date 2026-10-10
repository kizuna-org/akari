package mind

import (
	"maps"
	"reflect"

	"github.com/kizuna-org/akari/internal/emotion"
	"github.com/kizuna-org/akari/internal/goal"
	"github.com/kizuna-org/akari/internal/interest"
	"github.com/kizuna-org/akari/internal/meaning"
	"github.com/kizuna-org/akari/internal/memory"
	"github.com/kizuna-org/akari/internal/persona"
)

const ErrCheckpoint fault = "checkpoint schema, persona, versions or contents do not match"

// Restore creates a new owner and no surviving Channel generations or watchers.
func Restore(checkpoint Checkpoint, config persona.Config, capacity int, settings Settings) (*Workspace, error) {
	workspace, err := NewWithInner(capacity, config, settings)
	if err != nil {
		return nil, err
	}

	if checkpoint.Schema != 1 || !reflect.DeepEqual(checkpoint.Persona, config) ||
		checkpoint.Snapshot.Experience == nil || len(checkpoint.Snapshot.Items) > capacity {
		return nil, ErrCheckpoint
	}

	err = workspace.restore(checkpoint)
	if err != nil {
		return nil, err
	}

	return workspace, nil
}

func (workspace *Workspace) restore(checkpoint Checkpoint) error {
	experience := checkpoint.Snapshot.Experience.Clone()
	state := workspace.inner

	if !validVersions(checkpoint.Snapshot) || !validExperience(experience) {
		return ErrCheckpoint
	}

	archive, err := memory.Apply(checkpoint.Archive, memory.Change{
		Context: nil, Working: nil, Remember: nil, Recalled: nil, Agreements: nil,
	})
	if err != nil || !matchingMemory(experience, archive) {
		return ErrCheckpoint
	}

	goals, err := goal.Apply(experience.Goals, goal.Change{Desires: nil, Adopt: nil, End: nil},
		int(state.profile.Vector[persona.IntentionCount]))
	if err != nil {
		return ErrCheckpoint
	}

	state.emotion = experience.Emotion
	state.interests = experience.Interests
	state.goals = goals
	state.memory = archive
	state.fatigue = experience.Fatigue
	state.versions = experience.Versions

	if state.size() > state.settings.Contents {
		return ErrCapacity
	}

	workspace.revision = checkpoint.Snapshot.Revision
	workspace.items = maps.Clone(checkpoint.Snapshot.Items)

	return nil
}

func validVersions(snapshot Snapshot) bool {
	const partCount = 5

	if len(snapshot.Experience.Versions) != partCount {
		return false
	}

	for _, part := range []Part{Emotion, Interest, Memory, Goals, Fatigue} {
		version, exists := snapshot.Experience.Versions[part]
		if !exists || version > snapshot.Revision {
			return false
		}
	}

	for key, item := range snapshot.Items {
		if key == "" || item.Version == 0 || item.Version > snapshot.Revision {
			return false
		}
	}

	return true
}

func validExperience(experience *Experience) bool {
	if !meaning.Finite(experience.Fatigue) || experience.Fatigue < 0 || experience.Emotion.At.IsZero() {
		return false
	}

	appraisal := emotion.Appraisal{
		Feelings: experience.Emotion.Feelings, Others: nil,
		Readiness: experience.Emotion.Readiness, MoodDelta: experience.Emotion.Mood,
	}

	return appraisal.Validate() == nil && interest.Validate(experience.Interests) == nil
}

func matchingMemory(experience *Experience, archive memory.State) bool {
	return reflect.DeepEqual(experience.Context, archive.Context) &&
		reflect.DeepEqual(experience.Working, archive.Working) &&
		reflect.DeepEqual(experience.Agreements, archive.Agreements)
}
