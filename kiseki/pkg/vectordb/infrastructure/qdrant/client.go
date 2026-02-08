package qdrant

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	qdrant "github.com/qdrant/go-client/qdrant"
)

// Client wraps the Qdrant gRPC client
type Client struct {
	client *qdrant.Client
}

// NewClient creates a new Qdrant client
func NewClient(host string, port int, useTLS bool) (*Client, error) {
	client, err := qdrant.NewClient(&qdrant.Config{
		Host:   host,
		Port:   port,
		UseTLS: useTLS,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Qdrant client: %w", err)
	}

	return &Client{
		client: client,
	}, nil
}

// Close closes the Qdrant client connection
func (c *Client) Close() error {
	if c.client != nil {
		return c.client.Close()
	}
	return nil
}

// GetClient returns the underlying Qdrant client
func (c *Client) GetClient() *qdrant.Client {
	return c.client
}

// CollectionName returns the collection name for a character
func CollectionName(characterID uuid.UUID) string {
	return fmt.Sprintf("character_%s", characterID.String())
}

// EnsureCollection ensures that a collection exists for the character
func (c *Client) EnsureCollection(ctx context.Context, characterID uuid.UUID, vectorSize uint64) error {
	collectionName := CollectionName(characterID)

	// Check if collection exists
	exists, err := c.client.CollectionExists(ctx, collectionName)
	if err != nil {
		return fmt.Errorf("failed to check collection existence: %w", err)
	}

	if exists {
		return nil
	}

	// Create collection with hybrid search support (dense + sparse vectors)
	err = c.client.CreateCollection(ctx, &qdrant.CreateCollection{
		CollectionName: collectionName,
		VectorsConfig: qdrant.NewVectorsConfig(&qdrant.VectorParams{
			Size:     vectorSize,
			Distance: qdrant.Distance_Cosine,
		}),
		SparseVectorsConfig: &qdrant.SparseVectorConfig{
			Map: map[string]*qdrant.SparseVectorParams{
				"text": {},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create collection: %w", err)
	}

	return nil
}
