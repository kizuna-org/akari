package usecase

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/kizuna-org/akari/kiseki/pkg/memory/domain"
	"github.com/kizuna-org/akari/kiseki/pkg/memory/domain/entity"
	vectordbUsecase "github.com/kizuna-org/akari/kiseki/pkg/vectordb/usecase"
)

const (
	// Promotion thresholds
	BufferToContextThreshold = 2  // Access count needed to promote from buffer to context
	ContextToWorkingThreshold = 5 // Access count needed to promote from context to working
)

// LayerInteractor handles memory layer operations and promotions
type LayerInteractor struct {
	inputBufferRepo  domain.InputBufferRepository
	contextRepo      domain.ContextRepository
	memoryInteractor *vectordbUsecase.MemoryInteractor
}

// NewLayerInteractor creates a new layer interactor
func NewLayerInteractor(
	inputBufferRepo domain.InputBufferRepository,
	contextRepo domain.ContextRepository,
	memoryInteractor *vectordbUsecase.MemoryInteractor,
) *LayerInteractor {
	return &LayerInteractor{
		inputBufferRepo:  inputBufferRepo,
		contextRepo:      contextRepo,
		memoryInteractor: memoryInteractor,
	}
}

// AddToInputBufferInput represents input for adding to input buffer
type AddToInputBufferInput struct {
	CharacterID uuid.UUID
	Content     string
	Metadata    map[string]interface{}
}

// AddToInputBufferOutput represents output from adding to input buffer
type AddToInputBufferOutput struct {
	Fragment *entity.MemoryFragment
}

// AddToInputBuffer adds a new item to the input buffer
func (i *LayerInteractor) AddToInputBuffer(ctx context.Context, input AddToInputBufferInput) (*AddToInputBufferOutput, error) {
	// Create fragment
	fragment := entity.NewMemoryFragment(
		input.CharacterID,
		entity.LayerInputBuffer,
		input.Content,
		input.Metadata,
	)

	// Save to input buffer
	if err := i.inputBufferRepo.Push(ctx, fragment); err != nil {
		return nil, fmt.Errorf("failed to add to input buffer: %w", err)
	}

	return &AddToInputBufferOutput{
		Fragment: fragment,
	}, nil
}

// PromoteFragmentInput represents input for promoting a fragment
type PromoteFragmentInput struct {
	Fragment *entity.MemoryFragment
}

// PromoteFragmentOutput represents output from promoting a fragment
type PromoteFragmentOutput struct {
	PromotedTo entity.MemoryLayer
	Fragment   *entity.MemoryFragment
}

// PromoteFragment promotes a fragment to the next layer if it meets the threshold
func (i *LayerInteractor) PromoteFragment(ctx context.Context, input PromoteFragmentInput) (*PromoteFragmentOutput, error) {
	fragment := input.Fragment

	switch fragment.Layer {
	case entity.LayerInputBuffer:
		// Promote to context if threshold met
		if fragment.ShouldPromote(BufferToContextThreshold) {
			return i.promoteToContext(ctx, fragment)
		}

	case entity.LayerContext:
		// Promote to working memory if threshold met
		if fragment.ShouldPromote(ContextToWorkingThreshold) {
			return i.promoteToWorking(ctx, fragment)
		}
	}

	// No promotion needed
	return &PromoteFragmentOutput{
		PromotedTo: fragment.Layer,
		Fragment:   fragment,
	}, nil
}

// promoteToContext promotes a fragment from input buffer to context
func (i *LayerInteractor) promoteToContext(ctx context.Context, fragment *entity.MemoryFragment) (*PromoteFragmentOutput, error) {
	// Update fragment layer
	fragment.Layer = entity.LayerContext

	// Add to context
	if err := i.contextRepo.AddFragment(ctx, fragment.CharacterID, fragment); err != nil {
		return nil, fmt.Errorf("failed to add to context: %w", err)
	}

	return &PromoteFragmentOutput{
		PromotedTo: entity.LayerContext,
		Fragment:   fragment,
	}, nil
}

// promoteToWorking promotes a fragment from context to working memory (VectorDB)
func (i *LayerInteractor) promoteToWorking(ctx context.Context, fragment *entity.MemoryFragment) (*PromoteFragmentOutput, error) {
	// Note: This requires embedding generation
	// For now, we'll mark it as needing async embedding generation
	// The actual promotion will happen when the embedding task completes

	fragment.Layer = entity.LayerWorking

	return &PromoteFragmentOutput{
		PromotedTo: entity.LayerWorking,
		Fragment:   fragment,
	}, nil
}

// GetContextInput represents input for getting context
type GetContextInput struct {
	CharacterID uuid.UUID
}

// GetContextOutput represents output from getting context
type GetContextOutput struct {
	Snapshot *entity.ContextSnapshot
}

// GetContext retrieves the current context for a character
func (i *LayerInteractor) GetContext(ctx context.Context, input GetContextInput) (*GetContextOutput, error) {
	snapshot, err := i.contextRepo.Get(ctx, input.CharacterID)
	if err != nil {
		return nil, fmt.Errorf("failed to get context: %w", err)
	}

	return &GetContextOutput{
		Snapshot: snapshot,
	}, nil
}

// UpdateContextInput represents input for updating context
type UpdateContextInput struct {
	CharacterID uuid.UUID
	Summary     string
	Metadata    map[string]interface{}
}

// UpdateContextOutput represents output from updating context
type UpdateContextOutput struct {
	Snapshot *entity.ContextSnapshot
}

// UpdateContext updates the context summary and metadata
func (i *LayerInteractor) UpdateContext(ctx context.Context, input UpdateContextInput) (*UpdateContextOutput, error) {
	updates := make(map[string]interface{})

	if input.Summary != "" {
		updates["summary"] = input.Summary
	}
	if input.Metadata != nil {
		updates["metadata"] = input.Metadata
	}

	if err := i.contextRepo.Update(ctx, input.CharacterID, updates); err != nil {
		return nil, fmt.Errorf("failed to update context: %w", err)
	}

	snapshot, err := i.contextRepo.Get(ctx, input.CharacterID)
	if err != nil {
		return nil, fmt.Errorf("failed to get updated context: %w", err)
	}

	return &UpdateContextOutput{
		Snapshot: snapshot,
	}, nil
}

// ProcessInputBufferInput represents input for processing input buffer
type ProcessInputBufferInput struct {
	CharacterID uuid.UUID
}

// ProcessInputBufferOutput represents output from processing input buffer
type ProcessInputBufferOutput struct {
	ProcessedCount int
	PromotedCount  int
}

// ProcessInputBuffer processes all items in the input buffer and promotes eligible ones
func (i *LayerInteractor) ProcessInputBuffer(ctx context.Context, input ProcessInputBufferInput) (*ProcessInputBufferOutput, error) {
	// Get all fragments from input buffer
	fragments, err := i.inputBufferRepo.GetAll(ctx, input.CharacterID)
	if err != nil {
		return nil, fmt.Errorf("failed to get input buffer: %w", err)
	}

	processedCount := 0
	promotedCount := 0

	for _, fragment := range fragments {
		// Increment access count (simulating access)
		fragment.IncrementAccess()

		// Try to promote
		output, err := i.PromoteFragment(ctx, PromoteFragmentInput{Fragment: fragment})
		if err != nil {
			continue // Skip errors
		}

		processedCount++

		if output.PromotedTo != entity.LayerInputBuffer {
			promotedCount++
		}
	}

	return &ProcessInputBufferOutput{
		ProcessedCount: processedCount,
		PromotedCount:  promotedCount,
	}, nil
}
