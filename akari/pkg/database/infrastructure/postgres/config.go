package postgres

import (
	"os"
	"strconv"
	"time"

	"github.com/kizuna-org/akari/pkg/config"
)

const (
	defaultPort               = 5432
	defaultMaxOpenConns       = 25
	defaultMaxIdleConns       = 5
	defaultConnMaxLifetimeMin = 5
	defaultConnMaxIdleTimeMin = 2
)

type Config struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string
	SSLMode  string

	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration

	Debug bool
}

func NewConfig(appConfig config.Config) Config {
	return Config{
		Host:            getEnv("DB_HOST", "localhost"),
		Port:            getEnvAsInt("DB_PORT", defaultPort),
		User:            getEnv("DB_USER", "postgres"),
		Password:        getEnv("DB_PASSWORD", "postgres"),
		Database:        getEnv("DB_NAME", "akari"),
		SSLMode:         getEnv("DB_SSLMODE", "disable"),
		MaxOpenConns:    getEnvAsInt("DB_MAX_OPEN_CONNS", defaultMaxOpenConns),
		MaxIdleConns:    getEnvAsInt("DB_MAX_IDLE_CONNS", defaultMaxIdleConns),
		ConnMaxLifetime: time.Duration(getEnvAsInt("DB_CONN_MAX_LIFETIME_MINUTES", defaultConnMaxLifetimeMin)) * time.Minute,
		ConnMaxIdleTime: time.Duration(getEnvAsInt("DB_CONN_MAX_IDLE_TIME_MINUTES", defaultConnMaxIdleTimeMin)) * time.Minute,
		Debug:           appConfig.EnvMode == config.EnvModeDevelopment,
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}

	return defaultValue
}
