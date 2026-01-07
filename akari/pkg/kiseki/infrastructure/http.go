package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/kizuna-org/akari/gen/kiseki"
)

const defaultTimeout = 30 * time.Second

type HTTPClient struct {
	client *kiseki.Client
}

func NewHTTPClient(baseURL string) *HTTPClient {
	return &HTTPClient{
		client: &kiseki.Client{
			Server: baseURL,
			Client: &http.Client{
				Timeout:       defaultTimeout,
				Transport:     nil,
				CheckRedirect: nil,
				Jar:           nil,
			},
			RequestEditors: nil,
		},
	}
}

func (c *HTTPClient) PostMemoryPolling(
	ctx context.Context,
	characterID string,
	req *kiseki.MemoryPollingRequest,
) (*kiseki.MemoryPollingResponse, error) {
	parsedID, err := uuid.Parse(characterID)
	if err != nil {
		return nil, fmt.Errorf("infrastructure: invalid character ID: %w", err)
	}

	resp, err := c.client.PostMemoryPolling(ctx, parsedID, *req)
	if err != nil {
		return nil, fmt.Errorf("infrastructure: failed to execute polling request: %w", err)
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)

		return nil, fmt.Errorf("infrastructure: unexpected status code %d: %s", resp.StatusCode, string(body))
	}

	var result kiseki.MemoryPollingResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("infrastructure: failed to decode response: %w", err)
	}

	return &result, nil
}

func (c *HTTPClient) HealthCheck(ctx context.Context) error {
	resp, err := c.client.GetMemoryHealth(ctx)
	if err != nil {
		return fmt.Errorf("infrastructure: failed to check health: %w", err)
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("infrastructure: health check failed with status %d", resp.StatusCode)
	}

	return nil
}
