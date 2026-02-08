package di

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/kizuna-org/akari/kiseki/pkg/adapter"
	characterAdapter "github.com/kizuna-org/akari/kiseki/pkg/character/adapter"
	characterRedis "github.com/kizuna-org/akari/kiseki/pkg/character/infrastructure/redis"
	characterUsecase "github.com/kizuna-org/akari/kiseki/pkg/character/usecase"
	"github.com/kizuna-org/akari/kiseki/pkg/config"
	taskAdapter "github.com/kizuna-org/akari/kiseki/pkg/task/adapter"
	"github.com/kizuna-org/akari/kiseki/pkg/task/infrastructure/embedding"
	taskRedis "github.com/kizuna-org/akari/kiseki/pkg/task/infrastructure/redis"
	taskUsecase "github.com/kizuna-org/akari/kiseki/pkg/task/usecase"
	vectordbAdapter "github.com/kizuna-org/akari/kiseki/pkg/vectordb/adapter"
	qdrantInfra "github.com/kizuna-org/akari/kiseki/pkg/vectordb/infrastructure/qdrant"
	redisInfra "github.com/kizuna-org/akari/kiseki/pkg/vectordb/infrastructure/redis"
	vectordbUsecase "github.com/kizuna-org/akari/kiseki/pkg/vectordb/usecase"
	"github.com/redis/go-redis/v9"
)

// Container holds all dependencies
type Container struct {
	Config *config.Config

	// Clients
	RedisClient  *redis.Client
	QdrantClient *qdrantInfra.Client

	// Repositories
	CharacterRepo *characterRedis.Repository
	VectorDBRepo  *qdrantInfra.Repository
	KVSRepo       *redisInfra.Repository
	TaskRepo      *taskRedis.Repository

	// Services
	EmbeddingService *embedding.MockService

	// Usecases
	CharacterInteractor *characterUsecase.CharacterInteractor
	MemoryInteractor    *vectordbUsecase.MemoryInteractor
	TaskInteractor      *taskUsecase.TaskInteractor
	TaskWorker          *taskUsecase.Worker

	// Handlers
	CharacterHandler *characterAdapter.Handler
	MemoryHandler    *vectordbAdapter.Handler
	TaskHandler      *taskAdapter.Handler
	Server           *adapter.Server
}

// NewContainer creates a new DI container with all dependencies
func NewContainer() (*Container, error) {
	// Load configuration
	cfg, err := config.LoadFromEnv()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	slog.Info("Configuration loaded",
		"qdrant_host", cfg.Qdrant.Host,
		"redis_host", cfg.Redis.Host,
		"score_alpha", cfg.Score.Alpha,
	)

	// Initialize Redis client
	redisClient := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	// Test Redis connection
	ctx := context.Background()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}
	slog.Info("Redis connection established")

	// Initialize Qdrant client
	qdrantClient, err := qdrantInfra.NewClient(
		cfg.Qdrant.Host,
		cfg.Qdrant.Port,
		cfg.Qdrant.UseTLS,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create Qdrant client: %w", err)
	}
	slog.Info("Qdrant client created")

	// Initialize repositories
	characterRepo := characterRedis.NewRepository(redisClient)
	vectorDBRepo := qdrantInfra.NewRepository(qdrantClient, cfg.Qdrant.VectorSize)
	taskRepo := taskRedis.NewRepository(redisClient)

	// Create KVS repository with Redis client wrapper
	redisClientWrapper, err := redisInfra.NewClient(cfg.Redis.Host, cfg.Redis.Port, cfg.Redis.Password, cfg.Redis.DB)
	if err != nil {
		return nil, fmt.Errorf("failed to create Redis client wrapper: %w", err)
	}
	kvsRepo := redisInfra.NewRepository(redisClientWrapper)

	// Initialize services
	embeddingService := embedding.NewMockService(int(cfg.Qdrant.VectorSize))
	slog.Info("Using mock embedding service for development")

	// Initialize use cases
	characterInteractor := characterUsecase.NewCharacterInteractor(characterRepo)
	memoryInteractor := vectordbUsecase.NewMemoryInteractor(vectorDBRepo, kvsRepo, *cfg)
	taskInteractor := taskUsecase.NewTaskInteractor(taskRepo)
	pollingInteractor := taskUsecase.NewPollingInteractor(taskRepo, memoryInteractor)
	taskWorker := taskUsecase.NewWorker(taskRepo, embeddingService, memoryInteractor, 5*time.Second)

	// Initialize handlers
	characterHandler := characterAdapter.NewHandler(characterInteractor)
	memoryHandler := vectordbAdapter.NewHandler(memoryInteractor, taskInteractor)
	taskHandler := taskAdapter.NewHandler(taskInteractor, pollingInteractor)
	server := adapter.NewServer(characterHandler, memoryHandler, taskHandler)

	slog.Info("All dependencies initialized successfully")

	return &Container{
		Config:              cfg,
		RedisClient:         redisClient,
		QdrantClient:        qdrantClient,
		CharacterRepo:       characterRepo,
		VectorDBRepo:        vectorDBRepo,
		KVSRepo:             kvsRepo,
		TaskRepo:            taskRepo,
		EmbeddingService:    embeddingService,
		CharacterInteractor: characterInteractor,
		MemoryInteractor:    memoryInteractor,
		TaskInteractor:      taskInteractor,
		TaskWorker:          taskWorker,
		CharacterHandler:    characterHandler,
		MemoryHandler:       memoryHandler,
		TaskHandler:         taskHandler,
		Server:              server,
	}, nil
}

// Close closes all connections
func (c *Container) Close() error {
	var errs []error

	if c.RedisClient != nil {
		if err := c.RedisClient.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close Redis: %w", err))
		}
	}

	if c.QdrantClient != nil {
		if err := c.QdrantClient.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close Qdrant: %w", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors during cleanup: %v", errs)
	}

	slog.Info("All connections closed")
	return nil
}
