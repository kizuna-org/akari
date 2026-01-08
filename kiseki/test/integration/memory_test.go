package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/kizuna-org/akari/kiseki/gen"
	"github.com/kizuna-org/akari/kiseki/pkg/adapter"
	characterAdapter "github.com/kizuna-org/akari/kiseki/pkg/character/adapter"
	characterRedis "github.com/kizuna-org/akari/kiseki/pkg/character/infrastructure/redis"
	characterUsecase "github.com/kizuna-org/akari/kiseki/pkg/character/usecase"
	"github.com/kizuna-org/akari/kiseki/pkg/config"
	taskAdapter "github.com/kizuna-org/akari/kiseki/pkg/task/adapter"
	taskRedis "github.com/kizuna-org/akari/kiseki/pkg/task/infrastructure/redis"
	taskUsecase "github.com/kizuna-org/akari/kiseki/pkg/task/usecase"
	vectordbAdapter "github.com/kizuna-org/akari/kiseki/pkg/vectordb/adapter"
	qdrantInfra "github.com/kizuna-org/akari/kiseki/pkg/vectordb/infrastructure/qdrant"
	redisInfra "github.com/kizuna-org/akari/kiseki/pkg/vectordb/infrastructure/redis"
	vectordbUsecase "github.com/kizuna-org/akari/kiseki/pkg/vectordb/usecase"
	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"
)

// setupMemoryTestServer creates a test server with real Qdrant and Redis connections
func setupMemoryTestServer(t *testing.T) (*echo.Echo, func()) {
	// Connect to Redis
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   15, // Use separate DB for testing
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		t.Skipf("Redis not available: %v", err)
	}

	// Connect to Qdrant
	qdrantClient, err := qdrantInfra.NewClient("localhost", 6334, false)
	if err != nil {
		t.Skipf("Qdrant not available: %v", err)
	}

	// Setup configuration
	cfg := config.Config{
		Score: config.ScoreConfig{
			Alpha:   0.5,
			Beta:    0.3,
			Gamma:   0.2,
			Epsilon: 0.1,
		},
		Qdrant: config.QdrantConfig{
			Host:       "localhost",
			Port:       6334,
			UseTLS:     false,
			VectorSize: 768,
		},
		Redis: config.RedisConfig{
			Host:     "localhost",
			Port:     6379,
			Password: "",
			DB:       15,
		},
	}

	// Setup repositories
	characterRepo := characterRedis.NewRepository(redisClient)
	vectorDBRepo := qdrantInfra.NewRepository(qdrantClient, cfg.Qdrant.VectorSize)
	redisClientWrapper, _ := redisInfra.NewClient(cfg.Redis.Host, cfg.Redis.Port, cfg.Redis.Password, cfg.Redis.DB)
	kvsRepo := redisInfra.NewRepository(redisClientWrapper)

	// Setup usecases
	characterInteractor := characterUsecase.NewCharacterInteractor(characterRepo)
	memoryInteractor := vectordbUsecase.NewMemoryInteractor(vectorDBRepo, kvsRepo, cfg)

	// Setup handlers
	characterHandler := characterAdapter.NewHandler(characterInteractor)
	taskRepo := taskRedis.NewRepository(redisClient)
	taskInteractor := taskUsecase.NewTaskInteractor(taskRepo)
	memoryHandler := vectordbAdapter.NewHandler(memoryInteractor, taskInteractor)
	taskHandler := taskAdapter.NewHandler(taskInteractor)
	server := adapter.NewServer(characterHandler, memoryHandler, taskHandler)

	// Setup Echo
	e := echo.New()
	gen.RegisterHandlers(e, server)

	cleanup := func() {
		// Clean up test data
		ctx := context.Background()
		redisClient.FlushDB(ctx)
		redisClient.Close()
		qdrantClient.Close()
	}

	return e, cleanup
}

func TestIntegration_MemoryIO(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	e, cleanup := setupMemoryTestServer(t)
	defer cleanup()

	// Create a test character first
	characterID := uuid.New()

	// Create test vectors
	denseVector := make([]float32, 768)
	for i := range denseVector {
		denseVector[i] = float32(i) / 768.0
	}

	sparseVector := map[uint32]float32{
		0:   0.5,
		10:  0.3,
		100: 0.2,
	}

	t.Run("Store Memory Fragment", func(t *testing.T) {
		storeData := map[string]interface{}{
			"content":      "This is a test memory fragment",
			"denseVector":  denseVector,
			"sparseVector": sparseVector,
			"metadata": map[string]interface{}{
				"source": "test",
				"type":   "conversation",
			},
		}

		// Convert to proper request format
		dataBytes, _ := json.Marshal(storeData)
		var dataUnion gen.MemoryIORequest_Data
		_ = dataUnion.UnmarshalJSON(dataBytes)

		reqBody := gen.MemoryIORequest{
			DType: gen.Text,
			Data:  dataUnion,
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/characters/%s/memory", characterID.String()), bytes.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusNoContent {
			t.Fatalf("Expected status 204, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		t.Log("Memory fragment stored successfully")
	})

	// Wait a bit for the data to be indexed
	time.Sleep(1 * time.Second)

	t.Run("Search Memory Fragment", func(t *testing.T) {
		searchData := map[string]interface{}{
			"query":        "test memory",
			"denseVector":  denseVector,
			"sparseVector": sparseVector,
			"limit":        5,
		}
		searchDataJSON, _ := json.Marshal(searchData)

		req := httptest.NewRequest(
			http.MethodGet,
			fmt.Sprintf("/characters/%s/memory?dType=text&data=%s", characterID.String(), string(searchDataJSON)),
			nil,
		)
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("Expected status 200, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		var response gen.MemoryIOResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if len(response.Items) == 0 {
			t.Error("Expected at least one memory fragment in search results")
		}

		t.Logf("Found %d memory fragments", len(response.Items))
	})

	t.Run("Store Multiple Fragments", func(t *testing.T) {
		fragments := []string{
			"First test fragment",
			"Second test fragment",
			"Third test fragment",
		}

		for i, content := range fragments {
			// Slightly modify vectors for each fragment
			modifiedDenseVector := make([]float32, 768)
			copy(modifiedDenseVector, denseVector)
			modifiedDenseVector[0] = float32(i) * 0.1

			storeData := map[string]interface{}{
				"content":      content,
				"denseVector":  modifiedDenseVector,
				"sparseVector": sparseVector,
			}

			dataBytes, _ := json.Marshal(storeData)
			var dataUnion gen.MemoryIORequest_Data
			_ = dataUnion.UnmarshalJSON(dataBytes)

			reqBody := gen.MemoryIORequest{
				DType: gen.Text,
				Data:  dataUnion,
			}
			body, _ := json.Marshal(reqBody)

			req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/characters/%s/memory", characterID.String()), bytes.NewReader(body))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()

			e.ServeHTTP(rec, req)

			if rec.Code != http.StatusNoContent {
				t.Errorf("Fragment %d: Expected status 204, got %d", i, rec.Code)
			}
		}

		t.Log("Multiple fragments stored successfully")
	})

	// Wait for indexing
	time.Sleep(1 * time.Second)

	t.Run("Search Returns Multiple Results", func(t *testing.T) {
		searchData := map[string]interface{}{
			"query":        "test fragment",
			"denseVector":  denseVector,
			"sparseVector": sparseVector,
			"limit":        10,
		}
		searchDataJSON, _ := json.Marshal(searchData)

		req := httptest.NewRequest(
			http.MethodGet,
			fmt.Sprintf("/characters/%s/memory?dType=text&data=%s", characterID.String(), string(searchDataJSON)),
			nil,
		)
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("Expected status 200, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		var response gen.MemoryIOResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if len(response.Items) < 3 {
			t.Errorf("Expected at least 3 fragments, got %d", len(response.Items))
		}

		t.Logf("Search returned %d fragments", len(response.Items))
	})
}

func TestIntegration_MemoryValidation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	e, cleanup := setupMemoryTestServer(t)
	defer cleanup()

	characterID := uuid.New()

	t.Run("Store with Missing Dense Vector", func(t *testing.T) {
		storeData := map[string]interface{}{
			"content": "Test without vector",
		}

		dataBytes, _ := json.Marshal(storeData)
		var dataUnion gen.MemoryIORequest_Data
		_ = dataUnion.UnmarshalJSON(dataBytes)

		reqBody := gen.MemoryIORequest{
			DType: gen.Text,
			Data:  dataUnion,
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/characters/%s/memory", characterID.String()), bytes.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", rec.Code)
		}
	})

	t.Run("Search with Invalid Character ID", func(t *testing.T) {
		searchData := map[string]interface{}{
			"query":       "test",
			"denseVector": make([]float32, 768),
		}
		searchDataJSON, _ := json.Marshal(searchData)

		req := httptest.NewRequest(
			http.MethodGet,
			fmt.Sprintf("/characters/invalid-uuid/memory?dType=text&data=%s", string(searchDataJSON)),
			nil,
		)
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", rec.Code)
		}
	})
}
