package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config represents the application configuration
type Config struct {
	Qdrant QdrantConfig
	Redis  RedisConfig
	Score  ScoreConfig
}

// QdrantConfig represents Qdrant-specific configuration
type QdrantConfig struct {
	Host       string
	Port       int
	UseTLS     bool
	VectorSize uint64
}

// RedisConfig represents Redis-specific configuration
type RedisConfig struct {
	Host     string
	Port     int
	Password string
	DB       int
}

// ScoreConfig represents scoring algorithm configuration
type ScoreConfig struct {
	Alpha   float64 // Weight for semantic score
	Beta    float64 // Weight for popularity score
	Gamma   float64 // Weight for time score
	Epsilon float64 // Parameter for popularity score calculation
}

// LoadFromEnv loads configuration from environment variables
func LoadFromEnv() (*Config, error) {
	config := &Config{
		Qdrant: QdrantConfig{
			Host:       getEnvOrDefault("QDRANT_HOST", "localhost"),
			Port:       getEnvAsIntOrDefault("QDRANT_PORT", 6334),
			UseTLS:     getEnvAsBoolOrDefault("QDRANT_USE_TLS", false),
			VectorSize: uint64(getEnvAsIntOrDefault("QDRANT_VECTOR_SIZE", 768)),
		},
		Redis: RedisConfig{
			Host:     getEnvOrDefault("REDIS_HOST", "localhost"),
			Port:     getEnvAsIntOrDefault("REDIS_PORT", 6379),
			Password: getEnvOrDefault("REDIS_PASSWORD", ""),
			DB:       getEnvAsIntOrDefault("REDIS_DB", 0),
		},
		Score: ScoreConfig{
			Alpha:   getEnvAsFloatOrDefault("SCORE_ALPHA", 0.5),
			Beta:    getEnvAsFloatOrDefault("SCORE_BETA", 0.3),
			Gamma:   getEnvAsFloatOrDefault("SCORE_GAMMA", 0.2),
			Epsilon: getEnvAsFloatOrDefault("SCORE_EPSILON", 0.1),
		},
	}

	// Validate score weights sum
	totalWeight := config.Score.Alpha + config.Score.Beta + config.Score.Gamma
	if totalWeight < 0.99 || totalWeight > 1.01 {
		return nil, fmt.Errorf("score weights (alpha + beta + gamma) must sum to 1.0, got: %f", totalWeight)
	}

	return config, nil
}

// getEnvOrDefault gets an environment variable or returns a default value
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvAsIntOrDefault gets an environment variable as an integer or returns a default value
func getEnvAsIntOrDefault(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// getEnvAsBoolOrDefault gets an environment variable as a boolean or returns a default value
func getEnvAsBoolOrDefault(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

// getEnvAsFloatOrDefault gets an environment variable as a float64 or returns a default value
func getEnvAsFloatOrDefault(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if floatValue, err := strconv.ParseFloat(value, 64); err == nil {
			return floatValue
		}
	}
	return defaultValue
}
