package usecase

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/kizuna-org/akari/gen/kiseki"
	"github.com/kizuna-org/akari/pkg/kiseki/domain"
)

const defaultPollInterval = 5 * time.Second

type PollingInteractor struct {
	client    domain.PollingClient
	registry  *taskHandlerRegistry
	charID    string
	interval  time.Duration
	stopChan  chan struct{}
	mu        sync.Mutex
	isRunning bool
}

type taskHandlerRegistry struct {
	handlers map[string]domain.TaskHandler
	mu       sync.RWMutex
}

func newTaskHandlerRegistry() *taskHandlerRegistry {
	return &taskHandlerRegistry{
		handlers: make(map[string]domain.TaskHandler),
		mu:       sync.RWMutex{},
	}
}

func (r *taskHandlerRegistry) register(taskType string, handler domain.TaskHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.handlers[taskType] = handler
}

func (r *taskHandlerRegistry) get(taskType string) (domain.TaskHandler, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	handler, ok := r.handlers[taskType]

	return handler, ok
}

func NewPollingInteractor(client domain.PollingClient, characterID string) *PollingInteractor {
	return &PollingInteractor{
		client:    client,
		registry:  newTaskHandlerRegistry(),
		charID:    characterID,
		interval:  defaultPollInterval,
		stopChan:  make(chan struct{}),
		mu:        sync.Mutex{},
		isRunning: false,
	}
}

func (p *PollingInteractor) SetInterval(interval time.Duration) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.interval = interval
}

func (p *PollingInteractor) Start(ctx context.Context) error {
	p.mu.Lock()
	if p.isRunning {
		p.mu.Unlock()

		return errors.New("poller is already running")
	}

	p.isRunning = true
	p.stopChan = make(chan struct{})
	p.mu.Unlock()

	if err := p.client.HealthCheck(ctx); err != nil {
		p.mu.Lock()
		p.isRunning = false
		p.mu.Unlock()

		return fmt.Errorf("failed to verify Kiseki service: %w", err)
	}

	go p.pollLoop(ctx)

	return nil
}

func (p *PollingInteractor) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.isRunning {
		close(p.stopChan)
		p.isRunning = false
	}
}

func (p *PollingInteractor) IsRunning() bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	return p.isRunning
}

func (p *PollingInteractor) RegisterHandler(taskType string, handler domain.TaskHandler) {
	p.registry.register(taskType, handler)
}

func (p *PollingInteractor) pollLoop(ctx context.Context) {
	p.mu.Lock()
	interval := p.interval
	stopChan := p.stopChan
	p.mu.Unlock()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	emptyReq := &kiseki.MemoryPollingRequest{
		Items: []kiseki.PollingRequestItem{},
	}

	if err := p.processPoll(ctx, emptyReq); err != nil {
		log.Printf("error in initial poll: %v", err)
	}

	for {
		select {
		case <-ctx.Done():
			p.Stop()

			return
		case <-stopChan:
			return
		case <-ticker.C:
			if err := p.processPoll(ctx, emptyReq); err != nil {
				log.Printf("error in poll cycle: %v", err)
			}
		}
	}
}

func (p *PollingInteractor) processPoll(ctx context.Context, req *kiseki.MemoryPollingRequest) error {
	currentRequest := req

	for {
		resp, err := p.client.PostMemoryPolling(ctx, p.charID, currentRequest)
		if err != nil {
			return fmt.Errorf("polling request failed: %w", err)
		}

		if resp == nil {
			return errors.New("received empty response")
		}

		nextRequest := &kiseki.MemoryPollingRequest{
			Items: []kiseki.PollingRequestItem{},
		}

		for _, group := range resp.Items {
			p.handleGroup(ctx, &group, nextRequest)
		}

		if len(nextRequest.Items) == 0 {
			break
		}

		currentRequest = nextRequest
	}

	return nil
}

func (p *PollingInteractor) handleGroup(
	ctx context.Context,
	group *kiseki.PollingResponseGroup,
	nextRequest *kiseki.MemoryPollingRequest,
) {
	handler, ok := p.registry.get(group.TType)
	if !ok {
		log.Printf("no handler registered for task type: %s", group.TType)

		return
	}

	for _, task := range group.Items {
		domainTask := &domain.PollingTask{
			TaskID:   task.TaskId,
			TaskType: group.TType,
			Data:     &task.Data,
		}

		result, err := handler.Handle(ctx, domainTask)
		if err != nil {
			log.Printf("error handling task %s: %v", task.TaskId, err)

			continue
		}

		if result != nil {
			data := kiseki.PollingRequestItem_Data{}
			if result.Data != nil {
				data = *result.Data
			}

			item := kiseki.PollingRequestItem{
				TaskId: result.TaskID,
				DType:  "",
				Data:   data,
			}

			nextRequest.Items = append(nextRequest.Items, item)
		}
	}
}
