package qdrant

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/kizuna-org/akari/kiseki/pkg/vectordb/domain"
	"github.com/kizuna-org/akari/kiseki/pkg/vectordb/domain/entity"
	qdrant "github.com/qdrant/go-client/qdrant"
)

// Repository implements the VectorDBRepository interface for Qdrant
type Repository struct {
	client     *Client
	vectorSize uint64
}

// NewRepository creates a new Qdrant repository
func NewRepository(client *Client, vectorSize uint64) *Repository {
	return &Repository{
		client:     client,
		vectorSize: vectorSize,
	}
}

var _ domain.VectorDBRepository = (*Repository)(nil)

// Upsert inserts or updates a fragment in the vector database
func (r *Repository) Upsert(ctx context.Context, fragment entity.Fragment, denseVector []float32, sparseVector map[uint32]float32) error {
	collectionName := CollectionName(fragment.CharacterID)

	// Ensure collection exists
	if err := r.client.EnsureCollection(ctx, fragment.CharacterID, r.vectorSize); err != nil {
		return fmt.Errorf("failed to ensure collection: %w", err)
	}

	// Convert sparse vector to Qdrant format
	var sparseIndices []uint32
	var sparseValues []float32
	for idx, val := range sparseVector {
		sparseIndices = append(sparseIndices, idx)
		sparseValues = append(sparseValues, val)
	}

	// Prepare payload
	payload := map[string]*qdrant.Value{
		"id":           qdrant.NewValueString(fragment.ID.String()),
		"character_id": qdrant.NewValueString(fragment.CharacterID.String()),
		"data":         qdrant.NewValueString(fragment.Data),
		"dtype":        qdrant.NewValueString(string(fragment.DType)),
		"created_at":   qdrant.NewValueString(fragment.CreatedAt.Format("2006-01-02T15:04:05Z07:00")),
		"updated_at":   qdrant.NewValueString(fragment.UpdatedAt.Format("2006-01-02T15:04:05Z07:00")),
	}

	// Add metadata if present
	if fragment.Metadata != nil {
		for key, val := range fragment.Metadata {
			payload[fmt.Sprintf("meta_%s", key)] = qdrant.NewValueString(fmt.Sprintf("%v", val))
		}
	}

	// Upsert point
	pointID := qdrant.NewIDUUID(fragment.ID.String())
	point := &qdrant.PointStruct{
		Id:      pointID,
		Payload: payload,
		Vectors: qdrant.NewVectorsMap(
			map[string]*qdrant.Vector{
				"": qdrant.NewVectorDense(denseVector),
				"text": qdrant.NewVectorSparse(
					sparseIndices,
					sparseValues,
				),
			},
		),
	}

	_, err := r.client.GetClient().Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: collectionName,
		Points:         []*qdrant.PointStruct{point},
	})
	if err != nil {
		return fmt.Errorf("failed to upsert point: %w", err)
	}

	return nil
}

// HybridSearch performs hybrid search combining dense and sparse vectors
func (r *Repository) HybridSearch(ctx context.Context, characterID uuid.UUID, denseVector []float32, sparseVector map[uint32]float32, limit int) ([]entity.SearchResult, error) {
	collectionName := CollectionName(characterID)

	// Check if collection exists
	exists, err := r.client.GetClient().CollectionExists(ctx, collectionName)
	if err != nil {
		return nil, fmt.Errorf("failed to check collection existence: %w", err)
	}
	if !exists {
		return []entity.SearchResult{}, nil
	}

	// Convert sparse vector to Qdrant format
	var sparseIndices []uint32
	var sparseValues []float32
	for idx, val := range sparseVector {
		sparseIndices = append(sparseIndices, idx)
		sparseValues = append(sparseValues, val)
	}

	// Perform query using RRF (Reciprocal Rank Fusion)
	// This combines dense and sparse vectors automatically
	searchResult, err := r.client.GetClient().Query(ctx, &qdrant.QueryPoints{
		CollectionName: collectionName,
		Prefetch: []*qdrant.PrefetchQuery{
			{
				Query: qdrant.NewQueryDense(denseVector),
				Using: qdrant.PtrOf(""),
				Limit: qdrant.PtrOf(uint64(limit * 2)), // Get more candidates for fusion
			},
			{
				Query: qdrant.NewQuerySparse(sparseIndices, sparseValues),
				Using: qdrant.PtrOf("text"),
				Limit: qdrant.PtrOf(uint64(limit * 2)),
			},
		},
		Query: qdrant.NewQueryFusion(qdrant.Fusion_RRF),
		Limit: qdrant.PtrOf(uint64(limit)),
		WithPayload: &qdrant.WithPayloadSelector{
			SelectorOptions: &qdrant.WithPayloadSelector_Enable{
				Enable: true,
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to search: %w", err)
	}

	// Convert results
	results := make([]entity.SearchResult, 0, len(searchResult))
	for _, point := range searchResult {
		fragment, err := pointToFragment(point)
		if err != nil {
			continue // Skip invalid points
		}

		results = append(results, entity.SearchResult{
			Fragment:      fragment,
			SemanticScore: float64(point.GetScore()),
			Score:         float64(point.GetScore()),
		})
	}

	return results, nil
}

// Delete removes a fragment from the vector database
func (r *Repository) Delete(ctx context.Context, fragmentID uuid.UUID) error {
	// Note: We don't know which collection the fragment belongs to without additional context
	// In practice, this should be called with the character ID as well
	// For now, return an error indicating this limitation
	return fmt.Errorf("delete requires character ID context")
}

// EnsureCollection ensures that the collection for a character exists
func (r *Repository) EnsureCollection(ctx context.Context, characterID uuid.UUID) error {
	return r.client.EnsureCollection(ctx, characterID, r.vectorSize)
}

// pointToFragment converts a Qdrant point to a Fragment entity
func pointToFragment(point *qdrant.ScoredPoint) (entity.Fragment, error) {
	payload := point.GetPayload()

	idStr, ok := payload["id"].GetKind().(*qdrant.Value_StringValue)
	if !ok {
		return entity.Fragment{}, fmt.Errorf("invalid id in payload")
	}
	id, err := uuid.Parse(idStr.StringValue)
	if err != nil {
		return entity.Fragment{}, fmt.Errorf("failed to parse id: %w", err)
	}

	characterIDStr, ok := payload["character_id"].GetKind().(*qdrant.Value_StringValue)
	if !ok {
		return entity.Fragment{}, fmt.Errorf("invalid character_id in payload")
	}
	characterID, err := uuid.Parse(characterIDStr.StringValue)
	if err != nil {
		return entity.Fragment{}, fmt.Errorf("failed to parse character_id: %w", err)
	}

	dataVal, ok := payload["data"].GetKind().(*qdrant.Value_StringValue)
	if !ok {
		return entity.Fragment{}, fmt.Errorf("invalid data in payload")
	}

	dtypeVal, ok := payload["dtype"].GetKind().(*qdrant.Value_StringValue)
	if !ok {
		return entity.Fragment{}, fmt.Errorf("invalid dtype in payload")
	}

	return entity.Fragment{
		ID:          id,
		CharacterID: characterID,
		Data:        dataVal.StringValue,
		DType:       entity.DType(dtypeVal.StringValue),
		Metadata:    extractMetadata(payload),
	}, nil
}

// extractMetadata extracts metadata fields from payload
func extractMetadata(payload map[string]*qdrant.Value) map[string]interface{} {
	metadata := make(map[string]interface{})
	for key, val := range payload {
		if len(key) > 5 && key[:5] == "meta_" {
			metaKey := key[5:]
			if strVal, ok := val.GetKind().(*qdrant.Value_StringValue); ok {
				metadata[metaKey] = strVal.StringValue
			}
		}
	}
	return metadata
}
