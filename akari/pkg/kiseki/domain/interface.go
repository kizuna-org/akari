package domain

//go:generate go tool mockgen -package=mock -source=interface.go -destination=mock/interface.go

import (
	"context"

	"github.com/kizuna-org/akari/gen/kiseki"
)

type TaskHandler interface {
	Handle(ctx context.Context, task *PollingTask) (*PollingTaskResult, error)
}

type PollingService interface {
	Start(ctx context.Context) error
	Stop()
	IsRunning() bool
	RegisterHandler(taskType string, handler TaskHandler)
}

type PollingClient interface {
	PostMemoryPolling(
		ctx context.Context,
		characterID string,
		request *kiseki.MemoryPollingRequest,
	) (*kiseki.MemoryPollingResponse, error)
	HealthCheck(ctx context.Context) error
}
