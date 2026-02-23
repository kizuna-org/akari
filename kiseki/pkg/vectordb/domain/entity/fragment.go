package entity

import (
	"time"

	"github.com/google/uuid"
)

// DType represents the data type of a fragment
type DType string

const (
	DTypeText DType = "text"
)

// Fragment represents a memory fragment in the vector database
type Fragment struct {
	ID          uuid.UUID
	CharacterID uuid.UUID
	Data        string
	DType       DType
	Metadata    map[string]interface{}
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// AccessInfo represents access information for a fragment stored in KVS
type AccessInfo struct {
	FragmentID      uuid.UUID
	CharacterID     uuid.UUID
	AccessCount     int64
	LastAccessedAt  time.Time
	FirstAccessedAt time.Time
}

// SearchResult represents a search result with score
type SearchResult struct {
	Fragment        Fragment
	Score           float64
	SemanticScore   float64
	PopularityScore float64
	TimeScore       float64
}
