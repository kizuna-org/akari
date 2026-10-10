package action

import (
	"bytes"
	"slices"
)

// Command is an administrative frozen descriptor, never an executable closure or approval capability.
type Command struct {
	Tool          string   `json:"tool"`
	Arguments     []byte   `json:"arguments"`
	Impact        Impact   `json:"impact"`
	Destination   string   `json:"destination"`
	ConflictKey   string   `json:"conflictKey"`
	Sources       []string `json:"sources"`
	NeedsApproval bool     `json:"needsApproval"`
}

func (command Command) Clone() Command {
	command.Arguments = bytes.Clone(command.Arguments)
	command.Sources = slices.Clone(command.Sources)

	return command
}

func (pending *Pending) Command() Command { return pending.command.Clone() }
