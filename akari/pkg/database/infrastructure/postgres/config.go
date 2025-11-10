package postgres

import (
	"time"

	"github.com/kelseyhightower/envconfig"
	"github.com/kizuna-org/akari/pkg/config"
)

type Config struct {
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

func NewConfig(appConfig config.Config) (Config, error) {
	var cfg Config
	if err := envconfig.Process("postgres", &cfg); err != nil {
		return Config{}, err
	}

	cfg.ConnMaxLifetime = time.Duration(cfg.ConnMaxLifetimeMin) * time.Minute
	cfg.ConnMaxIdleTime = time.Duration(cfg.ConnMaxIdleTimeMin) * time.Minute

	cfg.Debug = appConfig.EnvMode == config.EnvModeDevelopment

	return cfg, nil
}
