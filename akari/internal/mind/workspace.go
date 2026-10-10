// Package mind owns the ordered shared workspace for concurrent thoughts.
package mind

import (
	"maps"
	"sync"
)

type fault string

func (err fault) Error() string { return string(err) }

const (
	ErrConfig   fault = "positive workspace capacity is required"
	ErrCapacity fault = "workspace infrastructure capacity reached"
	ErrConflict fault = "proposal dependencies changed"
	ErrProposal fault = "proposal must identify itself and declare every write dependency"
)

// Item is shared context, not a long-term memory record or persona parameter.
type Item struct {
	Content string
	Version uint64
}

type Snapshot struct {
	Revision uint64
	Items    map[string]Item
}

type Proposal struct {
	ID     string
	Reads  map[string]uint64
	Writes map[string]string
}

// Clone transfers a proposal across ownership boundaries.
func (proposal Proposal) Clone() Proposal {
	return Proposal{ID: proposal.ID, Reads: maps.Clone(proposal.Reads), Writes: maps.Clone(proposal.Writes)}
}

// Workspace serializes short commits; it never calls a model or an external tool.
type Workspace struct {
	mu          sync.Mutex
	revision    uint64
	items       map[string]Item
	capacity    int
	watchers    map[uint64]chan struct{}
	nextWatcher uint64
}

func New(capacity int) (*Workspace, error) {
	if capacity < 1 {
		return nil, ErrConfig
	}

	return &Workspace{
		mu: sync.Mutex{}, revision: 0, items: make(map[string]Item),
		watchers: make(map[uint64]chan struct{}), nextWatcher: 0, capacity: capacity,
	}, nil
}

func (workspace *Workspace) Snapshot() Snapshot {
	workspace.mu.Lock()
	defer workspace.mu.Unlock()

	return Snapshot{Revision: workspace.revision, Items: maps.Clone(workspace.items)}
}

// Commit validates only declared dependencies, avoiding starvation from unrelated changes.
func (workspace *Workspace) Commit(proposal Proposal) (uint64, error) {
	workspace.mu.Lock()
	defer workspace.mu.Unlock()

	err := workspace.validate(proposal)
	if err != nil {
		return workspace.revision, err
	}

	workspace.revision++

	for key, content := range proposal.Writes {
		workspace.items[key] = Item{Content: content, Version: workspace.revision}
	}

	for _, watcher := range workspace.watchers {
		select {
		case watcher <- struct{}{}:
		default:
		}
	}

	return workspace.revision, nil
}

// Watch wakes a consumer to fetch the latest snapshot. A slow consumer cannot block commits.
// The caller must call stop when its Channel ends.
func (workspace *Workspace) Watch() (<-chan struct{}, func()) {
	workspace.mu.Lock()
	workspace.nextWatcher++
	identity := workspace.nextWatcher
	wake := make(chan struct{}, 1)
	workspace.watchers[identity] = wake
	workspace.mu.Unlock()

	once := new(sync.Once)
	stop := func() {
		once.Do(func() {
			workspace.mu.Lock()
			defer workspace.mu.Unlock()

			delete(workspace.watchers, identity)
			close(wake)
		})
	}

	return wake, stop
}

func (workspace *Workspace) validate(proposal Proposal) error {
	if proposal.ID == "" {
		return ErrProposal
	}

	for key := range proposal.Writes {
		if _, exists := proposal.Reads[key]; !exists {
			return ErrProposal
		}
	}

	for key, version := range proposal.Reads {
		if workspace.items[key].Version != version {
			return ErrConflict
		}
	}

	return workspace.checkCapacity(proposal)
}

func (workspace *Workspace) checkCapacity(proposal Proposal) error {
	count := len(workspace.items)

	for key := range proposal.Writes {
		if _, exists := workspace.items[key]; !exists {
			count++
		}
	}

	if count > workspace.capacity {
		return ErrCapacity
	}

	return nil
}
