package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kizuna-org/akari/kiseki/gen"
	"github.com/kizuna-org/akari/kiseki/pkg/di"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func main() {
	// Setup logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	slog.Info("Starting Kiseki server...")

	// Initialize DI container
	container, err := di.NewContainer()
	if err != nil {
		slog.Error("Failed to initialize container", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := container.Close(); err != nil {
			slog.Error("Failed to close container", "error", err)
		}
	}()

	// Setup Echo
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	// Health check endpoint
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, gen.HealthResponse{
			Status:    gen.Healthy,
			Timestamp: timePtr(time.Now()),
			Version:   stringPtr("0.1.0"),
		})
	})

	// Register OpenAPI handlers
	gen.RegisterHandlers(e, container.Server)

	// Register custom task endpoints
	e.POST("/characters/:characterId/tasks", container.TaskHandler.CreateTask)
	e.GET("/tasks/:taskId", container.TaskHandler.GetTask)
	e.GET("/characters/:characterId/tasks", container.TaskHandler.ListTasks)

	// Start task worker
	workerCtx, workerCancel := context.WithCancel(context.Background())
	defer workerCancel()
	
	go func() {
		slog.Info("Task worker starting")
		container.TaskWorker.Start(workerCtx)
	}()

	// Start server
	port := getEnvOrDefault("PORT", "8080")
	addr := fmt.Sprintf(":%s", port)

	go func() {
		slog.Info("Server starting", "port", port)
		if err := e.Start(addr); err != nil && err != http.ErrServerClosed {
			slog.Error("Server failed to start", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("Shutting down server...")

	// Graceful shutdown
	slog.Info("Stopping task worker...")
	workerCancel()
	
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := e.Shutdown(ctx); err != nil {
		slog.Error("Server forced to shutdown", "error", err)
		os.Exit(1)
	}

	slog.Info("Server exited successfully")
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func timePtr(t time.Time) *time.Time {
	return &t
}

func stringPtr(s string) *string {
	return &s
}
