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

// setupTestServer creates a test server with real Redis connection
func setupTestServer(t *testing.T) (*echo.Echo, *redis.Client, func()) {
	// Connect to Redis (assumes Redis is running on localhost:6379)
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

	// Setup handlers
	characterRepo := characterRedis.NewRepository(redisClient)
	characterInteractor := characterUsecase.NewCharacterInteractor(characterRepo)
	characterHandler := characterAdapter.NewHandler(characterInteractor)

	// Setup memory handler (with mock repositories for now)
	cfg := config.Config{
		Score: config.ScoreConfig{
			Alpha:   0.5,
			Beta:    0.3,
			Gamma:   0.2,
			Epsilon: 0.1,
		},
		Qdrant: config.QdrantConfig{
			VectorSize: 768,
		},
	}

	// For character tests, we can use nil repositories for memory since we won't test memory endpoints
	// In a full integration test, we would set up real Qdrant and Redis connections
	qdrantClient, _ := qdrantInfra.NewClient("localhost", 6334, false)
	vectorDBRepo := qdrantInfra.NewRepository(qdrantClient, 768)
	redisClientWrapper, _ := redisInfra.NewClient("localhost", 6379, "", 0)
	kvsRepo := redisInfra.NewRepository(redisClientWrapper)
	memoryInteractor := vectordbUsecase.NewMemoryInteractor(vectorDBRepo, kvsRepo, cfg)
	
	// Setup task handler
	taskRepo := taskRedis.NewRepository(redisClient)
	taskInteractor := taskUsecase.NewTaskInteractor(taskRepo)
	taskHandler := taskAdapter.NewHandler(taskInteractor)
	
	// Setup memory handler (needs task interactor for async embedding)
	memoryHandler := vectordbAdapter.NewHandler(memoryInteractor, taskInteractor)
	
	server := adapter.NewServer(characterHandler, memoryHandler, taskHandler)

	// Setup Echo
	e := echo.New()
	gen.RegisterHandlers(e, server)

	cleanup := func() {
		// Clean up test data
		ctx := context.Background()
		redisClient.FlushDB(ctx)
		redisClient.Close()
	}

	return e, redisClient, cleanup
}

func TestIntegration_CharacterCRUD(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	e, _, cleanup := setupTestServer(t)
	defer cleanup()

	var characterID string

	t.Run("Create Character", func(t *testing.T) {
		reqBody := gen.CreateCharacterRequest{
			Name: "Test Character",
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/characters", bytes.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("Expected status 201, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		var char gen.Character
		if err := json.Unmarshal(rec.Body.Bytes(), &char); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if char.Name != "Test Character" {
			t.Errorf("Expected name 'Test Character', got '%s'", char.Name)
		}

		characterID = char.Id.String()
		t.Logf("Created character with ID: %s", characterID)
	})

	t.Run("Get Character", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/characters/%s", characterID), nil)
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("Expected status 200, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		var char gen.Character
		if err := json.Unmarshal(rec.Body.Bytes(), &char); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if char.Id.String() != characterID {
			t.Errorf("Expected ID '%s', got '%s'", characterID, char.Id.String())
		}
		if char.Name != "Test Character" {
			t.Errorf("Expected name 'Test Character', got '%s'", char.Name)
		}
	})

	t.Run("List Characters", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/characters", nil)
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("Expected status 200, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		var response gen.CharacterListResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if len(response.Items) == 0 {
			t.Error("Expected at least one character in list")
		}

		found := false
		for _, char := range response.Items {
			if char.Id.String() == characterID {
				found = true
				break
			}
		}
		if !found {
			t.Error("Created character not found in list")
		}
	})

	t.Run("Update Character", func(t *testing.T) {
		newName := "Updated Character"
		reqBody := gen.UpdateCharacterRequest{
			Name: &newName,
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/characters/%s", characterID), bytes.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("Expected status 200, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		var char gen.Character
		if err := json.Unmarshal(rec.Body.Bytes(), &char); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if char.Name != newName {
			t.Errorf("Expected name '%s', got '%s'", newName, char.Name)
		}
	})

	t.Run("Delete Character", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/characters/%s", characterID), nil)
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusNoContent {
			t.Fatalf("Expected status 204, got %d. Body: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("Get Deleted Character (Should 404)", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/characters/%s", characterID), nil)
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", rec.Code)
		}
	})
}

func TestIntegration_CharacterValidation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	e, _, cleanup := setupTestServer(t)
	defer cleanup()

	t.Run("Create Character with Empty Name", func(t *testing.T) {
		reqBody := gen.CreateCharacterRequest{
			Name: "",
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/characters", bytes.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", rec.Code)
		}
	})

	t.Run("Get Non-existent Character", func(t *testing.T) {
		nonExistentID := uuid.New().String()
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/characters/%s", nonExistentID), nil)
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", rec.Code)
		}
	})

	t.Run("Update with Invalid ID", func(t *testing.T) {
		newName := "Updated"
		reqBody := gen.UpdateCharacterRequest{
			Name: &newName,
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPut, "/characters/invalid-uuid", bytes.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", rec.Code)
		}
	})
}
