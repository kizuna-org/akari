package config

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

const (
	envTest       = "test"
	envProduction = "production"
)

// Defaults for the persona the program runs.
const (
	defaultPersonaName     = "akari"
	defaultPersonaSeed     = "1"
	defaultPersonaInterval = "3s"
	defaultPersonaNightly  = "24h"
)

type Config struct {
	Addr     string
	Database Database
	Persona  Persona
}

// Persona is how the running persona is set up.
//
// Only the pacing and the seed live here. What the persona is like — how it
// feels, what draws it, how well it follows through — belongs to the persona
// itself rather than to deployment configuration, so it is not settable by
// environment variable.
type Persona struct {
	// Name identifies the persona.
	Name string
	// Seed makes the persona's wandering attention reproducible.
	Seed uint64
	// Interval is how often it comes round to itself.
	Interval time.Duration
	// Nightly is how often it settles the day.
	Nightly time.Duration
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

	persona, err := loadPersona()
	if err != nil {
		return Config{}, err
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
		Persona: persona,
	}, nil
}

// loadPersona reads how the persona should be paced.
func loadPersona() (Persona, error) {
	seed, err := strconv.ParseUint(getenv("AKARI_PERSONA_SEED", defaultPersonaSeed), 10, 64)
	if err != nil {
		return Persona{}, fmt.Errorf("parse AKARI_PERSONA_SEED: %w", err)
	}

	interval, err := time.ParseDuration(getenv("AKARI_PERSONA_INTERVAL", defaultPersonaInterval))
	if err != nil {
		return Persona{}, fmt.Errorf("parse AKARI_PERSONA_INTERVAL: %w", err)
	}

	nightly, err := time.ParseDuration(getenv("AKARI_PERSONA_NIGHTLY", defaultPersonaNightly))
	if err != nil {
		return Persona{}, fmt.Errorf("parse AKARI_PERSONA_NIGHTLY: %w", err)
	}

	return Persona{
		Name:     getenv("AKARI_PERSONA_NAME", defaultPersonaName),
		Seed:     seed,
		Interval: interval,
		Nightly:  nightly,
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
