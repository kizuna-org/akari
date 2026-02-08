package redis

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

// Client wraps the Redis client
type Client struct {
	client *redis.Client
}

// NewClient creates a new Redis client
func NewClient(host string, port int, password string, db int) (*Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", host, port),
		Password: password,
		DB:       db,
	})

	// Test connection
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &Client{
		client: client,
	}, nil
}

// Close closes the Redis client connection
func (c *Client) Close() error {
	if c.client != nil {
		return c.client.Close()
	}
	return nil
}

// GetClient returns the underlying Redis client
func (c *Client) GetClient() *redis.Client {
	return c.client
}
