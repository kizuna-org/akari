package entity

import (
	"time"

	"github.com/google/uuid"
)

// Character represents a character entity
type Character struct {
	ID        uuid.UUID
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// NewCharacter creates a new character with the given name
func NewCharacter(name string) *Character {
	now := time.Now()
	return &Character{
		ID:        uuid.New(),
		Name:      name,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// Update updates the character's name
func (c *Character) Update(name string) {
	c.Name = name
	c.UpdatedAt = time.Now()
}
