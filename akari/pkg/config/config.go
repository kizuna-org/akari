//nolint:exhaustruct
package config

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
)

type ConfigRepository interface {
	LoadConfig() error
	GetConfig() Config
}

type configRepositoryImpl struct {
	config Config
}

type EnvMode string

const (
	EnvModeTest        EnvMode = "test"
	EnvModeProduction  EnvMode = "production"
	EnvModeDevelopment EnvMode = "development"
)

type Config struct {
	EnvMode EnvMode

	Database DatabaseConfig
	LLM      LLMConfig
	Log      LogConfig
	Discord  DiscordConfig
}

type LLMConfig struct {
	ProjectID string `split_words:"true"`
	Location  string
	ModelName string `split_words:"true"`
}

type LogConfig struct {
	Level  string
	Format string
}

type DiscordConfig struct {
	Token string
}

type DatabaseConfig struct {
	Host     string `split_words:"true"`
	Port     string `split_words:"true"`
	User     string `split_words:"true"`
	Password string `split_words:"true"`
	DB       string `split_words:"true"`
	SSLMode  string `split_words:"true" default:"disable"`
}

// BuildDSN builds a PostgreSQL data source name from the database configuration.
func (d *DatabaseConfig) BuildDSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		d.Host,
		d.Port,
		d.User,
		d.Password,
		d.DB,
		d.SSLMode,
	)
}

// BuildURL builds a PostgreSQL connection URL from the database configuration.
func (d *DatabaseConfig) BuildURL() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		d.User,
		d.Password,
		d.Host,
		d.Port,
		d.DB,
		d.SSLMode,
	)
}

func NewConfigRepository() ConfigRepository {
	configRepo := &configRepositoryImpl{}

	err := configRepo.LoadConfig()
	if err != nil {
		log.Printf("failed to load config: %v", err)
	}

	return configRepo
}

func (c *configRepositoryImpl) LoadConfig() error {
	envMode, envFile := c.determineEnvMode()

	if envFile != "" {
		err := godotenv.Load(envFile)
		if err != nil {
			return err
		}
	}

	err := c.loadAllConfigs()
	if err != nil {
		return err
	}

	c.config.EnvMode = envMode

	return nil
}

func (c *configRepositoryImpl) GetConfig() Config {
	return c.config
}

func (c *configRepositoryImpl) determineEnvMode() (EnvMode, string) {
	env := os.Getenv("ENV")

	var envFile string

	var envMode EnvMode

	switch env {
	case "test":
		envFile = os.Getenv("TEST_ENV")
		// if envFile == "" {
		//	envFile = "../../.env.test"
		// }

		envMode = EnvModeTest
	case "production":
		envFile = ""
		envMode = EnvModeProduction
	default:
		envFile = ".env"
		envMode = EnvModeDevelopment
	}

	return envMode, envFile
}

func (c *configRepositoryImpl) loadAllConfigs() error {
	databaseConfig := DatabaseConfig{}
	llmConfig := LLMConfig{}
	logConfig := LogConfig{}
	discordConfig := DiscordConfig{}

	err := envconfig.Process("akari", &c.config)
	if err != nil {
		return err
	}

	err = envconfig.Process("postgres", &databaseConfig)
	if err != nil {
		return err
	}

	err = envconfig.Process("llm", &llmConfig)
	if err != nil {
		return err
	}

	err = envconfig.Process("log", &logConfig)
	if err != nil {
		return err
	}

	err = envconfig.Process("discord", &discordConfig)
	if err != nil {
		return err
	}

	c.config.Database = databaseConfig
	c.config.LLM = llmConfig
	c.config.Log = logConfig
	c.config.Discord = discordConfig

	return nil
}
