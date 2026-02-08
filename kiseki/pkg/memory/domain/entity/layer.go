package entity

import (
	"time"

	"github.com/google/uuid"
)

// MemoryLayer represents the different layers of memory
type MemoryLayer string

const (
	LayerInputBuffer MemoryLayer = "input_buffer"   // Sensory memory (10 seconds)
	LayerContext     MemoryLayer = "context"        // Short-term memory (1 hour)
	LayerWorking     MemoryLayer = "working"        // Long-term memory (permanent in Qdrant)
	LayerDay         MemoryLayer = "day"            // Daily memory
	LayerSummary     MemoryLayer = "summary"        // Summary & sleeping memory
)

// MemoryFragment represents a piece of memory in any layer
type MemoryFragment struct {
	ID          uuid.UUID              `json:"id"`
	CharacterID uuid.UUID              `json:"characterId"`
	Layer       MemoryLayer            `json:"layer"`
	Content     string                 `json:"content"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	AccessCount int                    `json:"accessCount"`
	CreatedAt   time.Time              `json:"createdAt"`
	LastAccess  time.Time              `json:"lastAccess"`
	ExpiresAt   *time.Time             `json:"expiresAt,omitempty"` // For input buffer and context
}

// NewMemoryFragment creates a new memory fragment
func NewMemoryFragment(characterID uuid.UUID, layer MemoryLayer, content string, metadata map[string]interface{}) *MemoryFragment {
	now := time.Now()
	fragment := &MemoryFragment{
		ID:          uuid.New(),
		CharacterID: characterID,
		Layer:       layer,
		Content:     content,
		Metadata:    metadata,
		AccessCount: 0,
		CreatedAt:   now,
		LastAccess:  now,
	}

	// Set expiration based on layer
	switch layer {
	case LayerInputBuffer:
		expiresAt := now.Add(10 * time.Second)
		fragment.ExpiresAt = &expiresAt
	case LayerContext:
		expiresAt := now.Add(1 * time.Hour)
		fragment.ExpiresAt = &expiresAt
	}

	return fragment
}

// IncrementAccess increments the access count and updates last access time
func (f *MemoryFragment) IncrementAccess() {
	f.AccessCount++
	f.LastAccess = time.Now()
}

// IsExpired checks if the fragment has expired
func (f *MemoryFragment) IsExpired() bool {
	if f.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*f.ExpiresAt)
}

// ShouldPromote determines if the fragment should be promoted to the next layer
func (f *MemoryFragment) ShouldPromote(threshold int) bool {
	return f.AccessCount >= threshold
}

// ContextSnapshot represents the current context (short-term memory)
type ContextSnapshot struct {
	CharacterID uuid.UUID              `json:"characterId"`
	Fragments   []*MemoryFragment      `json:"fragments"`
	Summary     string                 `json:"summary,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	UpdatedAt   time.Time              `json:"updatedAt"`
}

// NewContextSnapshot creates a new context snapshot
func NewContextSnapshot(characterID uuid.UUID) *ContextSnapshot {
	return &ContextSnapshot{
		CharacterID: characterID,
		Fragments:   make([]*MemoryFragment, 0),
		Metadata:    make(map[string]interface{}),
		UpdatedAt:   time.Now(),
	}
}

// AddFragment adds a fragment to the context
func (c *ContextSnapshot) AddFragment(fragment *MemoryFragment) {
	c.Fragments = append(c.Fragments, fragment)
	c.UpdatedAt = time.Now()
}

// RemoveExpired removes expired fragments from the context
func (c *ContextSnapshot) RemoveExpired() {
	active := make([]*MemoryFragment, 0)
	for _, fragment := range c.Fragments {
		if !fragment.IsExpired() {
			active = append(active, fragment)
		}
	}
	c.Fragments = active
	c.UpdatedAt = time.Now()
}
