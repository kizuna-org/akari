//nolint:exhaustruct
package config

//go:generate go tool mockgen -package=mock -source=config.go -destination=mock/config.go

import (
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"time"

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
	Token         string
	BotNameRegExp string
	ReadyTimeout  int `default:"10"`
}

type DatabaseConfig struct {
	Host     string `default:"localhost" envconfig:"HOST"`
	Port     int    `default:"5432"      envconfig:"PORT"`
	User     string `default:"postgres"  envconfig:"USER"`
	Password string `default:"postgres"  envconfig:"PASSWORD"`
	Database string `default:"akari"     envconfig:"NAME"`
	SSLMode  string `default:"disable"   envconfig:"SSLMODE"`

	MaxOpenConns       int `default:"25" envconfig:"MAX_OPEN_CONNS"`
	MaxIdleConns       int `default:"5"  envconfig:"MAX_IDLE_CONNS"`
	ConnMaxLifetimeMin int `default:"5"  envconfig:"CONN_MAX_LIFETIME_MINUTES"`
	ConnMaxIdleTimeMin int `default:"2"  envconfig:"CONN_MAX_IDLE_TIME_MINUTES"`

	ConnMaxLifetime time.Duration `ignored:"true"`
	ConnMaxIdleTime time.Duration `ignored:"true"`

	Debug bool `ignored:"true"`
}

// BuildDSN builds a PostgreSQL data source name from the database configuration.
func (d *DatabaseConfig) BuildDSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		d.Host,
		d.Port,
		d.User,
		d.Password,
		d.Database,
		d.SSLMode,
	)
}

// BuildURL builds a PostgreSQL connection URL from the database configuration.
func (d *DatabaseConfig) BuildURL() string {
	hostPort := net.JoinHostPort(d.Host, strconv.Itoa(d.Port))

	return fmt.Sprintf(
		"postgres://%s:%s@%s/%s?sslmode=%s",
		d.User,
		d.Password,
		hostPort,
		d.Database,
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
		loadPath := envFile

		if !filepath.IsAbs(envFile) {
			wd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("config: failed to get working directory: %w", err)
			}

			projectRoot := c.findProjectRoot(wd)
			loadPath = filepath.Join(projectRoot, envFile)
		}

		err := godotenv.Load(loadPath)
		if err != nil {
			return err
		}
	}

	c.config.EnvMode = envMode

	err := c.loadAllConfigs()
	if err != nil {
		return err
	}

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
		if envFile == "" {
			envFile = ".env.test"
		}

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

func (c *configRepositoryImpl) findProjectRoot(startDir string) string {
	dir := startDir

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached root directory
			break
		}

		dir = parent
	}

	return startDir
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

	databaseConfig.ConnMaxLifetime = time.Duration(databaseConfig.ConnMaxLifetimeMin) * time.Minute
	databaseConfig.ConnMaxIdleTime = time.Duration(databaseConfig.ConnMaxIdleTimeMin) * time.Minute
	databaseConfig.Debug = c.config.EnvMode == EnvModeDevelopment

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
