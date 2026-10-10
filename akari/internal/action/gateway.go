// Package action provides one execution boundary for every thinking Channel.
package action

import (
	"bytes"
	"context"
	"maps"
	"slices"
	"sync"
	"sync/atomic"
)

type fault string

func (err fault) Error() string { return string(err) }

const (
	ErrConfig     fault = "positive execution capacity and named tools are required"
	ErrUnknown    fault = "tool is not registered"
	ErrOperation  fault = "adapter returned an invalid operation"
	ErrScope      fault = "destination is outside the trusted session scope"
	ErrPrediction fault = "prediction cannot modify the world"
	ErrApproval   fault = "operation requires a matching external approval"
	ErrUsed       fault = "operation has already been consumed"
	ErrPanic      fault = "execution adapter panicked"
	ErrDisclosure fault = "known memory sources are not permitted for this destination"
)

type Impact string

const (
	Read         Impact = "read"
	Reversible   Impact = "reversible"
	Speech       Impact = "speech"
	Irreversible Impact = "irreversible"
)

// Operation is normalized by a trusted adapter, never by model output.
type Operation struct {
	Impact      Impact
	Destination string
	ConflictKey string
	Call        func(context.Context) ([]byte, error)
	CallWithKey func(context.Context, string) ([]byte, error) `exhaustruct:"optional"`
	Sources     []string                                      `exhaustruct:"optional"`
}

// Prepare must be pure: it normalizes arguments without performing external effects.
type Prepare func([]byte) (Operation, error)

// Scope comes from runtime provenance and the conversation's actual participants.
type Scope struct {
	AllowedDestinations []string
	ReadyRecipients     []string
	Disclosure          func(sources []string, destination string) bool `exhaustruct:"optional"`
}

type target struct {
	users int
	slot  chan struct{}
}

type Gateway struct {
	tools   map[string]Prepare
	slots   chan struct{}
	mu      sync.Mutex
	targets map[string]*target
}

// Stats contains host resource counts, not Akari's psychological state.
type Stats struct {
	Executing   int
	TargetUsers int
}

func New(tools map[string]Prepare, capacity int) (*Gateway, error) {
	if capacity < 1 {
		return nil, ErrConfig
	}

	for name, prepare := range tools {
		if name == "" || prepare == nil {
			return nil, ErrConfig
		}
	}

	return &Gateway{
		tools: maps.Clone(tools), slots: make(chan struct{}, capacity), mu: sync.Mutex{}, targets: make(map[string]*target),
	}, nil
}

// Session is the only capability given to a Channel; it cannot approve operations.
type Session struct {
	gateway    *Gateway
	prediction bool
	allowed    map[string]bool
	ready      map[string]bool
	disclosure func([]string, string) bool
}

func (gateway *Gateway) Open(prediction bool, scope Scope) *Session {
	allowed := make(map[string]bool, len(scope.AllowedDestinations))
	ready := make(map[string]bool, len(scope.ReadyRecipients))

	for _, destination := range scope.AllowedDestinations {
		allowed[destination] = true
	}

	for _, recipient := range scope.ReadyRecipients {
		ready[recipient] = true
	}

	return &Session{
		gateway: gateway, prediction: prediction, allowed: allowed, ready: ready, disclosure: scope.Disclosure,
	}
}

// Pending freezes one normalized operation. Approval is bound to this exact object.
type Pending struct {
	gateway       *Gateway
	operation     Operation
	needsApproval bool
	approved      atomic.Bool
	used          atomic.Bool
	disclosure    func([]string, string) bool
	command       Command
}

func (session *Session) Plan(ctx context.Context, name string, arguments []byte) (*Pending, error) {
	contextErr := ctx.Err()
	if contextErr != nil {
		return nil, contextErr
	}

	prepare, exists := session.gateway.tools[name]
	if !exists {
		return nil, ErrUnknown
	}

	operation, err := prepare(bytes.Clone(arguments))
	if err != nil {
		return nil, err
	}

	operation.Sources = slices.Clone(operation.Sources)

	validationErr := session.check(operation)
	if validationErr != nil {
		return nil, validationErr
	}

	needsApproval := operation.Impact == Irreversible ||
		(operation.Impact == Speech && !session.ready[operation.Destination])

	return &Pending{
		gateway: session.gateway, operation: operation,
		needsApproval: needsApproval,
		approved:      atomic.Bool{}, used: atomic.Bool{},
		disclosure: session.disclosure,
		command: Command{Tool: name, Arguments: bytes.Clone(arguments), Impact: operation.Impact,
			Destination: operation.Destination, ConflictKey: operation.ConflictKey,
			Sources: slices.Clone(operation.Sources), NeedsApproval: needsApproval},
	}, nil
}

func (session *Session) check(operation Operation) error {
	if !validOperation(operation) {
		return ErrOperation
	}

	if session.prediction && operation.Impact != Read {
		return ErrPrediction
	}

	if !session.allowed[operation.Destination] {
		return ErrScope
	}

	if !allows(session.disclosure, operation) {
		return ErrDisclosure
	}

	return nil
}

func (pending *Pending) NeedsApproval() bool { return pending.needsApproval }

// Approve is held by the external confirmation adapter, not a thinking Channel.
func (gateway *Gateway) Approve(pending *Pending) error {
	if pending == nil || pending.gateway != gateway || !pending.needsApproval || pending.used.Load() {
		return ErrApproval
	}

	pending.approved.Store(true)

	return nil
}

func (gateway *Gateway) Stats() Stats {
	gateway.mu.Lock()
	defer gateway.mu.Unlock()

	users := 0

	for _, resource := range gateway.targets {
		users += resource.users
	}

	return Stats{Executing: len(gateway.slots), TargetUsers: users}
}

type Status string

const (
	NotExecuted Status = "not_executed"
	Succeeded   Status = "succeeded"
	Unknown     Status = "unknown"
)

type Observation struct {
	Status Status
	Data   []byte
}

// Execute is one-shot. An error after dispatch is unknown, never an automatic retry.
func (pending *Pending) Execute(ctx context.Context) (Observation, error) {
	return pending.execute(ctx, "")
}

// ExecuteIdentified passes a durable idempotency key to adapters that support it.
func (pending *Pending) ExecuteIdentified(ctx context.Context, identity string) (Observation, error) {
	if identity == "" {
		return Observation{Status: NotExecuted, Data: nil}, ErrOperation
	}

	return pending.execute(ctx, identity)
}

func (pending *Pending) execute(ctx context.Context, identity string) (Observation, error) {
	if identity == "" && pending.operation.Call == nil {
		return Observation{Status: NotExecuted, Data: nil}, ErrOperation
	}

	authorizationErr := pending.authorize(ctx)
	if authorizationErr != nil {
		return Observation{Status: NotExecuted, Data: nil}, authorizationErr
	}

	unlock, err := pending.gateway.lockTarget(ctx, pending.operation.ConflictKey)
	if err != nil {
		return Observation{Status: NotExecuted, Data: nil}, err
	}

	defer unlock()

	select {
	case pending.gateway.slots <- struct{}{}:
		defer func() { <-pending.gateway.slots }()
	case <-ctx.Done():
		return Observation{Status: NotExecuted, Data: nil}, ctx.Err()
	}

	// Cancellation can race with acquiring a slot, so check again before dispatch.
	contextErr := ctx.Err()
	if contextErr != nil {
		return Observation{Status: NotExecuted, Data: nil}, contextErr
	}

	if !allows(pending.disclosure, pending.operation) {
		return Observation{Status: NotExecuted, Data: nil}, ErrDisclosure
	}

	data, err := invoke(ctx, pending.call(identity))
	if err != nil {
		return Observation{Status: Unknown, Data: bytes.Clone(data)}, err
	}

	return Observation{Status: Succeeded, Data: bytes.Clone(data)}, nil
}

func (pending *Pending) call(identity string) func(context.Context) ([]byte, error) {
	if identity != "" && pending.operation.CallWithKey != nil {
		return func(ctx context.Context) ([]byte, error) { return pending.operation.CallWithKey(ctx, identity) }
	}

	return pending.operation.Call
}

func (pending *Pending) authorize(ctx context.Context) error {
	contextErr := ctx.Err()
	if contextErr != nil {
		return contextErr
	}

	if pending.needsApproval && !pending.approved.Load() {
		return ErrApproval
	}

	if !pending.used.CompareAndSwap(false, true) {
		return ErrUsed
	}

	return nil
}

// Named results let recovery classify an adapter panic as an unknown execution outcome.
//
//nolint:nonamedreturns // Panic recovery writes the returned error.
func invoke(ctx context.Context, call func(context.Context) ([]byte, error)) (data []byte, err error) {
	defer func() {
		if recover() != nil {
			err = ErrPanic
		}
	}()

	return call(ctx)
}

func (gateway *Gateway) lockTarget(ctx context.Context, key string) (func(), error) {
	if key == "" {
		return func() {}, nil
	}

	gateway.mu.Lock()

	resource, exists := gateway.targets[key]
	if !exists {
		resource = &target{users: 0, slot: make(chan struct{}, 1)}
		gateway.targets[key] = resource
	}

	resource.users++
	gateway.mu.Unlock()

	select {
	case resource.slot <- struct{}{}:
		return func() {
			<-resource.slot
			gateway.dropTarget(key, resource)
		}, nil
	case <-ctx.Done():
		gateway.dropTarget(key, resource)

		return nil, ctx.Err()
	}
}

func (gateway *Gateway) dropTarget(key string, resource *target) {
	gateway.mu.Lock()
	defer gateway.mu.Unlock()

	resource.users--
	if resource.users == 0 {
		delete(gateway.targets, key)
	}
}

func allows(policy func([]string, string) bool, operation Operation) bool {
	if len(operation.Sources) == 0 {
		return true
	}

	return policy != nil && policy(slices.Clone(operation.Sources), operation.Destination)
}

func validOperation(operation Operation) bool {
	switch operation.Impact {
	case Read, Reversible, Speech, Irreversible:
		return (operation.Call != nil || operation.CallWithKey != nil) && operation.Destination != "" &&
			(operation.Impact == Read || operation.ConflictKey != "")
	default:
		return false
	}
}
