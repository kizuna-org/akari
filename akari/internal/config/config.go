package config

import (
	"fmt"
	"net"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

const (
	envTest       = "test"
	envProduction = "production"
)

type Config struct {
	Addr     string
	Database Database
}

type Database struct {
	Host     string
	Port     int
	User     string
	Password string
	Name     string
	SSLMode  string
}

func Load() (Config, error) {
	_ = godotenv.Load(envFile())

	port, err := strconv.Atoi(getenv("POSTGRES_PORT", "5432"))
	if err != nil {
		return Config{}, fmt.Errorf("parse POSTGRES_PORT: %w", err)
	}

	return Config{
		Addr: getenv("AKARI_ADDR", ":8080"),
		Database: Database{
			Host:     getenv("POSTGRES_HOST", "localhost"),
			Port:     port,
			User:     getenv("POSTGRES_USER", "postgres"),
			Password: getenv("POSTGRES_PASSWORD", "postgres"),
			Name:     getenv("POSTGRES_DB", "akari"),
			SSLMode:  getenv("POSTGRES_SSLMODE", "disable"),
		},
	}, nil
}

func (d Database) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		d.Host,
		d.Port,
		d.User,
		d.Password,
		d.Name,
		d.SSLMode,
	)
}

func (d Database) URL() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s/%s?sslmode=%s",
		d.User,
		d.Password,
		net.JoinHostPort(d.Host, strconv.Itoa(d.Port)),
		d.Name,
		d.SSLMode,
	)
}

func envFile() string {
	if os.Getenv("ENV") == envTest {
		return ".env.test"
	}

	if os.Getenv("ENV") == envProduction {
		return ""
	}

	return ".env"
}

func getenv(key string, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return fallback
}
