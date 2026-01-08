package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/kizuna-org/akari/gen/kiseki"
	"github.com/kizuna-org/akari/pkg/kiseki/domain"
	"github.com/kizuna-org/akari/pkg/kiseki/domain/mock"
	"github.com/kizuna-org/akari/pkg/kiseki/usecase"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func emptyResponse() *kiseki.MemoryPollingResponse {
	return &kiseki.MemoryPollingResponse{Items: []kiseki.PollingResponseGroup{}}
}

func setupHealthCheckOnly(ctrl *gomock.Controller, shouldFail bool) domain.PollingClient {
	mockClient := mock.NewMockPollingClient(ctrl)

	var err error
	if shouldFail {
		err = errors.New("health check failed")
	}

	mockClient.EXPECT().HealthCheck(gomock.Any()).Return(err)
	mockClient.EXPECT().PostMemoryPolling(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(emptyResponse(), nil).AnyTimes()

	return mockClient
}

func setupErrorHandling(ctrl *gomock.Controller) (domain.PollingClient, domain.TaskHandler) {
	mockClient := mock.NewMockPollingClient(ctrl)
	mockHandler := mock.NewMockTaskHandler(ctrl)
	gomock.InOrder(
		mockClient.EXPECT().HealthCheck(gomock.Any()).Return(nil),
		mockClient.EXPECT().PostMemoryPolling(gomock.Any(), "char-001", gomock.Any()).
			Return(&kiseki.MemoryPollingResponse{Items: []kiseki.PollingResponseGroup{
				{TType: "test-task", Items: []kiseki.PollingResponseItem{
					{TaskId: "task-001", Data: kiseki.PollingResponseItem_Data{}},
				}},
				{TType: "unknown-task", Items: []kiseki.PollingResponseItem{
					{TaskId: "task-002", Data: kiseki.PollingResponseItem_Data{}},
				}},
			}}, nil),
		mockHandler.EXPECT().Handle(gomock.Any(), gomock.Any()).Return(nil, errors.New("handler error")),
		mockClient.EXPECT().PostMemoryPolling(gomock.Any(), "char-001", gomock.Any()).
			Return(nil, errors.New("client error")),
		mockClient.EXPECT().PostMemoryPolling(gomock.Any(), "char-001", gomock.Any()).
			Return(nil, nil),
		mockClient.EXPECT().PostMemoryPolling(gomock.Any(), "char-001", gomock.Any()).
			Return(emptyResponse(), nil).AnyTimes(),
	)

	return mockClient, mockHandler
}

func setupNilResults(ctrl *gomock.Controller) (domain.PollingClient, domain.TaskHandler) {
	mockClient := mock.NewMockPollingClient(ctrl)
	mockHandler := mock.NewMockTaskHandler(ctrl)
	gomock.InOrder(
		mockClient.EXPECT().HealthCheck(gomock.Any()).Return(nil),
		mockClient.EXPECT().PostMemoryPolling(gomock.Any(), "char-001", gomock.Any()).
			Return(&kiseki.MemoryPollingResponse{Items: []kiseki.PollingResponseGroup{
				{TType: "test-task", Items: []kiseki.PollingResponseItem{
					{TaskId: "task-001", Data: kiseki.PollingResponseItem_Data{}},
					{TaskId: "task-002", Data: kiseki.PollingResponseItem_Data{}},
				}},
			}}, nil),
		mockHandler.EXPECT().Handle(gomock.Any(), gomock.Any()).Return(nil, nil),
		mockHandler.EXPECT().Handle(gomock.Any(), gomock.Any()).Return(&domain.PollingTaskResult{TaskID: "task-002"}, nil),
		mockClient.EXPECT().PostMemoryPolling(gomock.Any(), "char-001", gomock.Any()).
			Return(nil, errors.New("submission error")),
		mockClient.EXPECT().PostMemoryPolling(gomock.Any(), "char-001", gomock.Any()).
			Return(emptyResponse(), nil).AnyTimes(),
	)

	return mockClient, mockHandler
}

func setupHandlerErrorPath(ctrl *gomock.Controller) (domain.PollingClient, domain.TaskHandler) {
	mockClient := mock.NewMockPollingClient(ctrl)
	mockHandler := mock.NewMockTaskHandler(ctrl)
	gomock.InOrder(
		mockClient.EXPECT().HealthCheck(gomock.Any()).Return(nil),
		mockClient.EXPECT().PostMemoryPolling(gomock.Any(), "char-001", gomock.Any()).
			Return(&kiseki.MemoryPollingResponse{Items: []kiseki.PollingResponseGroup{
				{TType: "test-task", Items: []kiseki.PollingResponseItem{
					{TaskId: "task-001", Data: kiseki.PollingResponseItem_Data{}},
				}},
			}}, nil),
		mockHandler.EXPECT().Handle(gomock.Any(), gomock.Any()).Return(nil, errors.New("handler error")),
		mockClient.EXPECT().PostMemoryPolling(gomock.Any(), "char-001", gomock.Any()).
			Return(emptyResponse(), nil).AnyTimes(),
	)

	return mockClient, mockHandler
}

func setupHandlerWithResultData(ctrl *gomock.Controller) (domain.PollingClient, domain.TaskHandler) {
	mockClient := mock.NewMockPollingClient(ctrl)
	mockHandler := mock.NewMockTaskHandler(ctrl)
	gomock.InOrder(
		mockClient.EXPECT().HealthCheck(gomock.Any()).Return(nil),
		mockClient.EXPECT().PostMemoryPolling(gomock.Any(), "char-001", gomock.Any()).
			Return(&kiseki.MemoryPollingResponse{Items: []kiseki.PollingResponseGroup{
				{TType: "test-task", Items: []kiseki.PollingResponseItem{
					{TaskId: "task-001", Data: kiseki.PollingResponseItem_Data{}},
				}},
			}}, nil),
		mockHandler.EXPECT().Handle(gomock.Any(), gomock.Any()).Return(&domain.PollingTaskResult{
			TaskID: "result-001",
			Data:   &kiseki.PollingRequestItem_Data{},
		}, nil),
		mockClient.EXPECT().PostMemoryPolling(gomock.Any(), "char-001", gomock.Any()).
			Return(emptyResponse(), nil).AnyTimes(),
	)

	return mockClient, mockHandler
}

func TestPollingInteractor(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name     string
		setup    func(*gomock.Controller) (domain.PollingClient, map[string]domain.TaskHandler)
		testFn   func(*testing.T, *usecase.PollingInteractor)
		interval time.Duration
	}

	tests := []testCase{
		{
			name: "start, run, and stop successfully",
			setup: func(ctrl *gomock.Controller) (domain.PollingClient, map[string]domain.TaskHandler) {
				return setupHealthCheckOnly(ctrl, false), nil
			},
			testFn: func(t *testing.T, interactor *usecase.PollingInteractor) {
				t.Helper()
				ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
				defer cancel()
				require.NoError(t, interactor.Start(ctx))
				assert.True(t, interactor.IsRunning())
				interactor.Stop()
				time.Sleep(100 * time.Millisecond)
				assert.False(t, interactor.IsRunning())
			},
			interval: 1 * time.Hour,
		},
		{
			name: "health check failure",
			setup: func(ctrl *gomock.Controller) (domain.PollingClient, map[string]domain.TaskHandler) {
				return setupHealthCheckOnly(ctrl, true), nil
			},
			testFn: func(t *testing.T, interactor *usecase.PollingInteractor) {
				t.Helper()
				ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
				defer cancel()
				require.Error(t, interactor.Start(ctx))
				assert.False(t, interactor.IsRunning())
			},
			interval: 1 * time.Hour,
		},
		{
			name: "already running error and duplicate start",
			setup: func(ctrl *gomock.Controller) (domain.PollingClient, map[string]domain.TaskHandler) {
				return setupHealthCheckOnly(ctrl, false), nil
			},
			testFn: func(t *testing.T, interactor *usecase.PollingInteractor) {
				t.Helper()
				ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
				defer cancel()
				require.NoError(t, interactor.Start(ctx))
				defer interactor.Stop()
				err := interactor.Start(ctx)
				require.Error(t, err)
				assert.Equal(t, "poller is already running", err.Error())
			},
			interval: 1 * time.Hour,
		},
		{
			name: "handle errors gracefully: handler error, unknown task type, client error, invalid response",
			setup: func(ctrl *gomock.Controller) (domain.PollingClient, map[string]domain.TaskHandler) {
				client, handler := setupErrorHandling(ctrl)

				return client, map[string]domain.TaskHandler{"test-task": handler}
			},
			testFn: func(t *testing.T, interactor *usecase.PollingInteractor) {
				t.Helper()
				ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
				defer cancel()
				require.NoError(t, interactor.Start(ctx))
				defer interactor.Stop()
				time.Sleep(500 * time.Millisecond)
			},
			interval: 100 * time.Millisecond,
		},
		{
			name: "handle nil results and submission error",
			setup: func(ctrl *gomock.Controller) (domain.PollingClient, map[string]domain.TaskHandler) {
				client, handler := setupNilResults(ctrl)

				return client, map[string]domain.TaskHandler{"test-task": handler}
			},
			testFn: func(t *testing.T, interactor *usecase.PollingInteractor) {
				t.Helper()
				ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
				defer cancel()
				require.NoError(t, interactor.Start(ctx))
				defer interactor.Stop()
				time.Sleep(500 * time.Millisecond)
			},
			interval: 100 * time.Millisecond,
		},
		{
			name: "handle task handler error and continue processing",
			setup: func(ctrl *gomock.Controller) (domain.PollingClient, map[string]domain.TaskHandler) {
				client, handler := setupHandlerErrorPath(ctrl)

				return client, map[string]domain.TaskHandler{"test-task": handler}
			},
			testFn: func(t *testing.T, interactor *usecase.PollingInteractor) {
				t.Helper()
				ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
				defer cancel()
				require.NoError(t, interactor.Start(ctx))
				defer interactor.Stop()
				time.Sleep(500 * time.Millisecond)
			},
			interval: 100 * time.Millisecond,
		},
		{
			name: "handle result with data",
			setup: func(ctrl *gomock.Controller) (domain.PollingClient, map[string]domain.TaskHandler) {
				client, handler := setupHandlerWithResultData(ctrl)

				return client, map[string]domain.TaskHandler{"test-task": handler}
			},
			testFn: func(t *testing.T, interactor *usecase.PollingInteractor) {
				t.Helper()
				ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
				defer cancel()
				require.NoError(t, interactor.Start(ctx))
				defer interactor.Stop()
				time.Sleep(500 * time.Millisecond)
			},
			interval: 100 * time.Millisecond,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			client, handlers := testCase.setup(ctrl)
			interactor := usecase.NewPollingInteractor(client, "char-001")
			interactor.SetInterval(testCase.interval)

			for taskType, handler := range handlers {
				interactor.RegisterHandler(taskType, handler)
			}

			testCase.testFn(t, interactor)
		})
	}
}
