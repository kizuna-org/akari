// Package continuity owns durable adoption and the administrative action outbox.
package continuity

import (
	"bytes"
	"context"
	"maps"
	"time"

	"github.com/kizuna-org/akari/internal/action"
	"github.com/kizuna-org/akari/internal/mind"
	"github.com/kizuna-org/akari/internal/persona"
)

type fault string

func (err fault) Error() string { return string(err) }

const (
	ErrAbsent      fault = "no saved state"
	ErrConflict    fault = "saved sequence changed"
	ErrState       fault = "invalid durable state or action descriptor"
	ErrConfig      fault = "durable owner requires repository, gateway and positive limits"
	ErrBusy        fault = "durable mutation or dispatch already in progress"
	ErrUnavailable fault = "save outcome is uncertain; reopen before further mutations"
	ErrAction      fault = "action identity is missing, reused or not in the required state"
)

type Stage string

const (
	Queued      Stage = "queued"
	Running     Stage = "running"
	Succeeded   Stage = "succeeded"
	NotExecuted Stage = "not_executed"
	Unknown     Stage = "unknown"
)

type Intent struct {
	ID        string
	Tool      string
	Arguments []byte
}

type Record struct {
	ID       string         `json:"id"`
	Revision uint64         `json:"revision"`
	Command  action.Command `json:"command"`
	Stage    Stage          `json:"stage"`
	Approved bool           `json:"approved"`
	Data     []byte         `json:"data"`
}

// Frame atomically couples adopted state and actions. Sequence also counts administrative transitions.
type Frame struct {
	Schema     int               `json:"schema"`
	Identity   string            `json:"identity"`
	Sequence   uint64            `json:"sequence"`
	Checkpoint mind.Checkpoint   `json:"checkpoint"`
	Outbox     map[string]Record `json:"outbox"`
}

func (frame Frame) Clone() Frame {
	frame.Checkpoint.Persona = frame.Checkpoint.Persona.Clone()
	frame.Checkpoint.Snapshot.Items = maps.Clone(frame.Checkpoint.Snapshot.Items)
	frame.Checkpoint.Snapshot.Experience = frame.Checkpoint.Snapshot.Experience.Clone()
	frame.Checkpoint.Archive = frame.Checkpoint.Archive.Clone()
	frame.Outbox = maps.Clone(frame.Outbox)

	for identity, record := range frame.Outbox {
		record.Command = record.Command.Clone()
		record.Data = bytes.Clone(record.Data)
		frame.Outbox[identity] = record
	}

	return frame
}

// Repository.Save must be atomic compare-and-swap and durable before returning nil.
// An error may be ambiguous; callers stop and reload rather than retrying effects.
type Repository interface {
	Load(ctx context.Context) (Frame, error)
	Save(ctx context.Context, expected uint64, next Frame) error
}

type Config struct {
	Persona           persona.Config
	Inner             mind.Settings
	WorkspaceCapacity int
	OutboxCapacity    int
	SaveTimeout       time.Duration
	Scope             func(mind.Snapshot) action.Scope
}

// Resolver must prove an outcome by the same identity; it must never dispatch the operation.
type Resolver interface {
	Lookup(ctx context.Context, identity string, command action.Command) (action.Observation, error)
}
