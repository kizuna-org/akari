package console

import "github.com/kizuna-org/akari/internal/continuity"

type status struct {
	Mode             string                   `json:"mode"`
	Permission       string                   `json:"permission"`
	Revision         uint64                   `json:"revision"`
	Sequence         uint64                   `json:"sequence"`
	Schema           int                      `json:"schema"`
	Outbox           map[continuity.Stage]int `json:"outbox"`
	AwaitingApproval int                      `json:"awaitingApproval"`
	SharedItems      int                      `json:"sharedItems"`
	WorkingMemories  int                      `json:"workingMemories"`
	Interests        int                      `json:"interests"`
}

func project(frame continuity.Frame) status {
	stages := map[continuity.Stage]int{
		continuity.Queued: 0, continuity.Running: 0, continuity.Succeeded: 0,
		continuity.NotExecuted: 0, continuity.Unknown: 0,
	}

	approval := 0

	for _, record := range frame.Outbox {
		stages[record.Stage]++

		if record.Stage == continuity.Queued && record.Command.NeedsApproval && !record.Approved {
			approval++
		}
	}

	interests := 0
	if frame.Checkpoint.Snapshot.Experience != nil {
		interests = len(frame.Checkpoint.Snapshot.Experience.Interests)
	}

	return status{Mode: "foundation", Permission: "observer", Revision: frame.Checkpoint.Snapshot.Revision,
		Sequence: frame.Sequence, Schema: frame.Schema, Outbox: stages, AwaitingApproval: approval,
		SharedItems: len(frame.Checkpoint.Snapshot.Items), WorkingMemories: len(frame.Checkpoint.Archive.Working),
		Interests: interests}
}
