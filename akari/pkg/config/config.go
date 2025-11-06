//nolint:exhaustruct
package config

import (
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

	LLM     LLMConfig
	Log     LogConfig
	Discord DiscordConfig
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
	llmConfig := LLMConfig{}
	logConfig := LogConfig{}
	discordConfig := DiscordConfig{}

	err := envconfig.Process("akari", &c.config)
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

	c.config.LLM = llmConfig
	c.config.Log = logConfig
	c.config.Discord = discordConfig

	return nil
}
