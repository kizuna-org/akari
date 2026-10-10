package continuity

import (
	"context"
	"crypto/rand"
	"errors"
	"reflect"
	"sync"

	"github.com/kizuna-org/akari/internal/action"
	"github.com/kizuna-org/akari/internal/memory"
	"github.com/kizuna-org/akari/internal/mind"
)

// Owner serializes durable writes without holding the snapshot lock during disk I/O.
// It exposes no mutable Workspace; every adoption takes the durable path.
type Owner struct {
	mu          sync.RWMutex
	gate        chan struct{}
	repository  Repository
	gateway     *action.Gateway
	config      Config
	workspace   *mind.Workspace
	frame       Frame
	unavailable bool
}

func Open(ctx context.Context, repository Repository, gateway *action.Gateway, config Config) (*Owner, error) {
	if repository == nil || gateway == nil || !validConfig(config) {
		return nil, ErrConfig
	}

	ctx, cancel := context.WithTimeout(ctx, config.SaveTimeout)
	defer cancel()

	config.Persona = config.Persona.Clone()

	frame, err := repository.Load(ctx)
	if errors.Is(err, ErrAbsent) {
		frame, err = initial(config)
		if err == nil {
			err = repository.Save(ctx, 0, frame.Clone())
		}
	}

	if err != nil {
		return nil, err
	}

	workspace, err := restore(frame, config)
	if err != nil {
		return nil, err
	}

	owner := &Owner{
		mu: sync.RWMutex{}, gate: make(chan struct{}, 1), repository: repository, gateway: gateway,
		config: config, workspace: workspace, frame: frame.Clone(), unavailable: false,
	}

	err = owner.recoverRunning(ctx)
	if err != nil {
		return nil, err
	}

	return owner, nil
}

func (owner *Owner) Snapshot() mind.Snapshot {
	owner.mu.RLock()
	defer owner.mu.RUnlock()

	return owner.workspace.Snapshot()
}

// CommitContext makes this durable owner usable by the thought supervisor.
// Runtime uses Adopt/AcceptWith when actions must accompany the same adoption.
func (owner *Owner) CommitContext(ctx context.Context, proposal mind.Proposal) (uint64, error) {
	return owner.Adopt(ctx, proposal, nil)
}

func (owner *Owner) Recall(ctx context.Context, query memory.Query) ([]memory.Fragment, error) {
	owner.mu.RLock()
	workspace := owner.workspace
	owner.mu.RUnlock()

	return workspace.Recall(ctx, query)
}

// Export is administrative; never provide the outbox or persona configuration to a model.
func (owner *Owner) Export() Frame {
	owner.mu.RLock()
	defer owner.mu.RUnlock()

	return owner.frame.Clone()
}

func (owner *Owner) AllowsDisclosure(sources []string, recipient string) bool {
	owner.mu.RLock()
	workspace := owner.workspace
	owner.mu.RUnlock()

	return workspace.AllowsDisclosure(sources, recipient)
}

// Adopt saves the candidate and all planned actions before exposing the new shared state.
func (owner *Owner) Adopt(ctx context.Context, proposal mind.Proposal, intents []Intent) (uint64, error) {
	err := owner.enter(ctx)
	if err != nil {
		return 0, err
	}

	defer owner.leave()

	next := owner.Export()

	workspace, err := mind.Restore(next.Checkpoint,
		owner.config.Persona, owner.config.WorkspaceCapacity, owner.config.Inner)
	if err != nil {
		return 0, err
	}

	revision, err := workspace.CommitContext(ctx, proposal.Clone())
	if err != nil {
		return 0, err
	}

	next.Checkpoint, err = workspace.Export()
	if err != nil {
		return 0, err
	}

	err = owner.plan(ctx, &next, workspace, intents)
	if err != nil {
		return 0, err
	}

	err = owner.save(ctx, next, workspace)
	if err != nil {
		return 0, err
	}

	return revision, nil
}

// Approve is a runtime/admin capability, never part of a Channel session.
func (owner *Owner) Approve(ctx context.Context, identity string) error {
	err := owner.enter(ctx)
	if err != nil {
		return err
	}

	defer owner.leave()

	next := owner.Export()

	record, exists := next.Outbox[identity]
	if !exists || record.Stage != Queued || !record.Command.NeedsApproval {
		return ErrAction
	}

	record.Approved = true
	next.Outbox[identity] = record

	return owner.save(ctx, next, nil)
}

// Dispatch claims one persisted action before entering Gateway. No failed/unknown action is retried.
func (owner *Owner) Dispatch(ctx context.Context, identity string) (action.Observation, error) {
	record, pending, key, err := owner.claim(ctx, identity)
	if err != nil {
		return action.Observation{Status: action.NotExecuted, Data: nil}, err
	}

	observation, callErr := pending.ExecuteIdentified(ctx, key)

	// Persist cancellation outcomes using a bounded cleanup lifetime.
	cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), owner.config.SaveTimeout)
	defer cancel()

	err = owner.finish(cleanup, record, observation)
	if err != nil {
		owner.fail()

		return action.Observation{Status: action.Unknown, Data: nil}, errors.Join(callErr, err)
	}

	return observation, callErr
}

func (owner *Owner) Cancel(ctx context.Context, identity string) error {
	err := owner.enter(ctx)
	if err != nil {
		return err
	}

	defer owner.leave()

	next := owner.Export()

	record, exists := next.Outbox[identity]
	if !exists || record.Stage != Queued {
		return ErrAction
	}

	record.Stage = NotExecuted
	next.Outbox[identity] = record

	return owner.save(ctx, next, nil)
}

func (owner *Owner) Reconcile(ctx context.Context, identity string, resolver Resolver) error {
	err := owner.enter(ctx)
	if err != nil {
		return err
	}

	defer owner.leave()

	next := owner.Export()

	record, exists := next.Outbox[identity]
	if !exists || record.Stage != Unknown || resolver == nil {
		return ErrAction
	}

	observation, err := resolver.Lookup(ctx, next.Identity+"/"+identity, record.Command.Clone())
	if err != nil {
		return err
	}

	if observation.Status != action.Succeeded && observation.Status != action.NotExecuted {
		return ErrState
	}

	record.Stage = Stage(observation.Status)
	record.Data = observation.Data
	next.Outbox[identity] = record

	return owner.save(ctx, next, nil)
}

func (owner *Owner) claim(ctx context.Context, identity string) (Record, *action.Pending, string, error) {
	err := owner.enter(ctx)
	if err != nil {
		return Record{}, nil, "", err
	}
	defer owner.leave()

	next := owner.Export()

	record, exists := next.Outbox[identity]
	if !exists || record.Stage != Queued {
		return Record{}, nil, "", ErrAction
	}

	pending, err := owner.reprepare(ctx, record)
	if err != nil {
		return Record{}, nil, "", err
	}

	record.Stage = Running
	next.Outbox[identity] = record

	err = owner.save(ctx, next, nil)
	if err != nil {
		return Record{}, nil, "", err
	}

	return record, pending, next.Identity + "/" + identity, nil
}

func (owner *Owner) finish(ctx context.Context, record Record, observation action.Observation) error {
	err := owner.enterWait(ctx)
	if err != nil {
		return err
	}
	defer owner.leave()

	record.Stage = Stage(observation.Status)
	record.Data = observation.Data
	next := owner.Export()
	next.Outbox[record.ID] = record

	return owner.save(ctx, next, nil)
}

func (owner *Owner) enter(ctx context.Context) error {
	err := ctx.Err()
	if err != nil {
		return err
	}

	select {
	case owner.gate <- struct{}{}:
	default:
		return ErrBusy
	}

	owner.mu.RLock()
	unavailable := owner.unavailable
	owner.mu.RUnlock()

	if unavailable {
		owner.leave()

		return ErrUnavailable
	}

	return nil
}

func (owner *Owner) leave() { <-owner.gate }

func (owner *Owner) enterWait(ctx context.Context) error {
	select {
	case owner.gate <- struct{}{}:
	case <-ctx.Done():
		return ctx.Err()
	}

	owner.mu.RLock()
	unavailable := owner.unavailable
	owner.mu.RUnlock()

	if unavailable {
		owner.leave()

		return ErrUnavailable
	}

	return nil
}

func (owner *Owner) save(ctx context.Context, next Frame, workspace *mind.Workspace) error {
	ctx, cancel := context.WithTimeout(ctx, owner.config.SaveTimeout)
	defer cancel()

	err := ctx.Err()
	if err != nil {
		return err
	}

	expected := next.Sequence
	next.Sequence++

	err = owner.repository.Save(ctx, expected, next.Clone())
	if err != nil {
		owner.fail()

		return errors.Join(ErrUnavailable, err)
	}

	owner.mu.Lock()

	owner.frame = next.Clone()
	if workspace != nil {
		owner.workspace = workspace
	}
	owner.mu.Unlock()

	return nil
}

func (owner *Owner) fail() {
	owner.mu.Lock()
	owner.unavailable = true
	owner.mu.Unlock()
}

func (owner *Owner) plan(ctx context.Context, next *Frame, workspace *mind.Workspace, intents []Intent) error {
	if len(next.Outbox)+len(intents) > owner.config.OutboxCapacity {
		return mind.ErrCapacity
	}

	scope := owner.config.Scope(workspace.Snapshot())
	scope.Disclosure = workspace.AllowsDisclosure
	session := owner.gateway.Open(false, scope)

	for _, intent := range intents {
		if _, exists := next.Outbox[intent.ID]; exists || intent.ID == "" {
			return ErrAction
		}

		pending, err := session.Plan(ctx, intent.Tool, intent.Arguments)
		if err != nil {
			return err
		}

		next.Outbox[intent.ID] = Record{
			ID: intent.ID, Revision: next.Checkpoint.Snapshot.Revision, Command: pending.Command(),
			Stage: Queued, Approved: false, Data: nil,
		}
	}

	return nil
}

func (owner *Owner) reprepare(ctx context.Context, record Record) (*action.Pending, error) {
	scope := owner.config.Scope(owner.Snapshot())
	scope.Disclosure = owner.AllowsDisclosure

	pending, err := owner.gateway.Open(false, scope).Plan(ctx, record.Command.Tool, record.Command.Arguments)
	if err != nil {
		return nil, err
	}

	if !reflect.DeepEqual(record.Command, pending.Command()) {
		return nil, ErrState
	}

	if record.Command.NeedsApproval {
		if !record.Approved {
			return nil, action.ErrApproval
		}

		err = owner.gateway.Approve(pending)
		if err != nil {
			return nil, err
		}
	}

	return pending, nil
}

func (owner *Owner) recoverRunning(ctx context.Context) error {
	next := owner.Export()
	changed := false

	for identity, record := range next.Outbox {
		if record.Stage == Running {
			record.Stage = Unknown
			next.Outbox[identity] = record
			changed = true
		}
	}

	if changed {
		return owner.save(ctx, next, nil)
	}

	return nil
}

func initial(config Config) (Frame, error) {
	workspace, err := mind.NewWithInner(config.WorkspaceCapacity, config.Persona, config.Inner)
	if err != nil {
		return Frame{}, err
	}

	checkpoint, err := workspace.Export()
	if err != nil {
		return Frame{}, err
	}

	return Frame{
		Schema: 1, Identity: rand.Text(), Sequence: 1,
		Checkpoint: checkpoint, Outbox: make(map[string]Record),
	}, nil
}

func restore(frame Frame, config Config) (*mind.Workspace, error) {
	if !validFrame(frame, config.OutboxCapacity) {
		return nil, ErrState
	}

	for identity, record := range frame.Outbox {
		if !validRecord(identity, record, frame.Checkpoint.Snapshot.Revision) {
			return nil, ErrState
		}
	}

	return mind.Restore(frame.Checkpoint, config.Persona, config.WorkspaceCapacity, config.Inner)
}

func validCommand(command action.Command) bool {
	switch command.Impact {
	case action.Read:
		return !command.NeedsApproval
	case action.Reversible, action.Speech:
		return command.ConflictKey != ""
	case action.Irreversible:
		return command.ConflictKey != "" && command.NeedsApproval
	default:
		return false
	}
}

func validConfig(config Config) bool {
	return config.Scope != nil && config.OutboxCapacity > 0 && config.SaveTimeout > 0
}

func validRecord(identity string, record Record, revision uint64) bool {
	if !validAddress(identity, record, revision) {
		return false
	}

	if !validCommand(record.Command) || (record.Approved && !record.Command.NeedsApproval) {
		return false
	}

	switch record.Stage {
	case Queued, Running, Succeeded, NotExecuted, Unknown:
		return true
	default:
		return false
	}
}

func validAddress(identity string, record Record, revision uint64) bool {
	return identity != "" && identity == record.ID && record.Revision > 0 && record.Revision <= revision &&
		record.Command.Tool != "" && record.Command.Destination != ""
}

func validFrame(frame Frame, capacity int) bool {
	return frame.Schema == 1 && frame.Identity != "" && frame.Sequence > frame.Checkpoint.Snapshot.Revision &&
		frame.Outbox != nil && len(frame.Outbox) <= capacity
}
