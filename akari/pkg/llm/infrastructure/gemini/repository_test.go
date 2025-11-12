package gemini_test

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/kizuna-org/akari/pkg/config"
	"github.com/kizuna-org/akari/pkg/config/mock"
	"github.com/kizuna-org/akari/pkg/llm/infrastructure/gemini"
	"go.uber.org/mock/gomock"
)

func TestNewRepository(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		setupMock   func(*mock.MockConfigRepository)
		expectError bool
		skipReason  string
	}{
		{
			name: "successful repository creation",
			setupMock: func(mockCfg *mock.MockConfigRepository) {
				mockCfg.EXPECT().GetConfig().Return(config.Config{
					EnvMode: config.EnvModeDevelopment,
					LLM: config.LLMConfig{
						ProjectID: "test-project",
						Location:  "us-central1",
						ModelName: "gemini-1.5-flash",
					},
				})
			},
			expectError: false,
			skipReason:  "Requires valid GCP credentials",
		},
		{
			name: "with empty project ID",
			setupMock: func(mockCfg *mock.MockConfigRepository) {
				mockCfg.EXPECT().GetConfig().Return(config.Config{
					EnvMode: config.EnvModeDevelopment,
					LLM: config.LLMConfig{
						ProjectID: "",
						Location:  "us-central1",
						ModelName: "gemini-1.5-flash",
					},
				})
			},
			expectError: true,
			skipReason:  "Requires valid GCP credentials",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if testCase.skipReason != "" {
				t.Skip(testCase.skipReason)
			}

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockConfig := mock.NewMockConfigRepository(ctrl)
			testCase.setupMock(mockConfig)

			logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

			repo, err := gemini.NewRepository(mockConfig, logger)

			if testCase.expectError {
				if err == nil {
					t.Error("Expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}

				if repo == nil {
					t.Error("Expected non-nil repository")
				}
			}
		})
	}
}

func TestRepository_ContextHandling(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		setupCtx       func() context.Context
		expectCanceled bool
	}{
		{
			name:           "normal context",
			setupCtx:       t.Context,
			expectCanceled: false,
		},
		{
			name: "cancelled context",
			setupCtx: func() context.Context {
				ctx, cancel := context.WithCancel(t.Context())
				cancel()

				return ctx
			},
			expectCanceled: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctx := testCase.setupCtx()
			if ctx == nil {
				t.Fatal("Context should not be nil")
			}

			if testCase.expectCanceled {
				select {
				case <-ctx.Done():
					if ctx.Err() == nil {
						t.Error("Cancelled context should have an error")
					}
				default:
					t.Error("Context should be cancelled")
				}
			} else {
				select {
				case <-ctx.Done():
					t.Error("Context should not be cancelled")
				default:
				}
			}
		})
	}
}
