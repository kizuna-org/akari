package config_test

import (
	"testing"
	"time"

	"github.com/kizuna-org/akari/pkg/config"
	"github.com/stretchr/testify/assert"
)

func TestDatabaseConfig_BuildDSN(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		dbConfig config.DatabaseConfig
		want     string
	}{
		{
			name: "default configuration",
			dbConfig: config.DatabaseConfig{
				Host:               "localhost",
				Port:               5432,
				User:               "postgres",
				Password:           "postgres",
				Database:           "akari",
				SSLMode:            "disable",
				MaxOpenConns:       0,
				MaxIdleConns:       0,
				ConnMaxLifetimeMin: 0,
				ConnMaxIdleTimeMin: 0,
				ConnMaxLifetime:    0,
				ConnMaxIdleTime:    0,
				Debug:              false,
			},
			want: "host=localhost port=5432 user=postgres password=postgres dbname=akari sslmode=disable",
		},
		{
			name: "custom configuration with special characters",
			dbConfig: config.DatabaseConfig{
				Host:               "db.example.com",
				Port:               5433,
				User:               "user",
				Password:           "p@ss!word#123",
				Database:           "testdb",
				SSLMode:            "require",
				MaxOpenConns:       0,
				MaxIdleConns:       0,
				ConnMaxLifetimeMin: 0,
				ConnMaxIdleTimeMin: 0,
				ConnMaxLifetime:    0,
				ConnMaxIdleTime:    0,
				Debug:              false,
			},
			want: "host=db.example.com port=5433 user=user password=p@ss!word#123 dbname=testdb sslmode=require",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, testCase.want, testCase.dbConfig.BuildDSN())
		})
	}
}

func TestDatabaseConfig_BuildURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		dbConfig config.DatabaseConfig
		want     string
	}{
		{
			name: "default configuration",
			dbConfig: config.DatabaseConfig{
				Host:               "localhost",
				Port:               5432,
				User:               "postgres",
				Password:           "postgres",
				Database:           "akari",
				SSLMode:            "disable",
				MaxOpenConns:       0,
				MaxIdleConns:       0,
				ConnMaxLifetimeMin: 0,
				ConnMaxIdleTimeMin: 0,
				ConnMaxLifetime:    0,
				ConnMaxIdleTime:    0,
				Debug:              false,
			},
			want: "postgres://postgres:postgres@localhost:5432/akari?sslmode=disable",
		},
		{
			name: "custom configuration",
			dbConfig: config.DatabaseConfig{
				Host:               "db.example.com",
				Port:               5433,
				User:               "dbuser",
				Password:           "secret123",
				Database:           "testdb",
				SSLMode:            "require",
				MaxOpenConns:       0,
				MaxIdleConns:       0,
				ConnMaxLifetimeMin: 0,
				ConnMaxIdleTimeMin: 0,
				ConnMaxLifetime:    0,
				ConnMaxIdleTime:    0,
				Debug:              false,
			},
			want: "postgres://dbuser:secret123@db.example.com:5433/testdb?sslmode=require",
		},
		{
			name: "IPv6 host",
			dbConfig: config.DatabaseConfig{
				Host:               "::1",
				Port:               5432,
				User:               "postgres",
				Password:           "postgres",
				Database:           "akari",
				SSLMode:            "disable",
				MaxOpenConns:       0,
				MaxIdleConns:       0,
				ConnMaxLifetimeMin: 0,
				ConnMaxIdleTimeMin: 0,
				ConnMaxLifetime:    0,
				ConnMaxIdleTime:    0,
				Debug:              false,
			},
			want: "postgres://postgres:postgres@[::1]:5432/akari?sslmode=disable",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, testCase.want, testCase.dbConfig.BuildURL())
		})
	}
}

func TestConfigRepository_LoadConfig(t *testing.T) {
	tests := []struct {
		envMode  string
		expected config.EnvMode
	}{
		{"production", config.EnvModeProduction},
		{"test", config.EnvModeTest},
	}

	for _, testCase := range tests {
		t.Run(testCase.envMode, func(t *testing.T) {
			t.Setenv("ENV", testCase.envMode)

			cfg := config.NewConfigRepository().GetConfig()
			assert.Equal(t, testCase.expected, cfg.EnvMode)
			assert.False(t, cfg.Database.Debug)
		})
	}
}

func TestConfigRepository_DatabaseConfig(t *testing.T) {
	t.Setenv("ENV", "test")
	t.Setenv("POSTGRES_CONN_MAX_LIFETIME_MINUTES", "10")
	t.Setenv("POSTGRES_CONN_MAX_IDLE_TIME_MINUTES", "3")

	cfg := config.NewConfigRepository().GetConfig()

	assert.Equal(t, 10, cfg.Database.ConnMaxLifetimeMin)
	assert.Equal(t, 10*time.Minute, cfg.Database.ConnMaxLifetime)
	assert.Equal(t, 3, cfg.Database.ConnMaxIdleTimeMin)
	assert.Equal(t, 3*time.Minute, cfg.Database.ConnMaxIdleTime)
}

func TestConfigRepository_AllEnvironmentVariables(t *testing.T) {
	t.Setenv("ENV", "test")
	t.Setenv("POSTGRES_HOST", "testhost")
	t.Setenv("POSTGRES_PORT", "5433")
	t.Setenv("POSTGRES_USER", "testuser")
	t.Setenv("POSTGRES_PASSWORD", "testpass")
	t.Setenv("POSTGRES_NAME", "testdb")
	t.Setenv("POSTGRES_SSLMODE", "require")
	t.Setenv("POSTGRES_MAX_OPEN_CONNS", "50")
	t.Setenv("POSTGRES_MAX_IDLE_CONNS", "10")
	t.Setenv("POSTGRES_CONN_MAX_LIFETIME_MINUTES", "15")
	t.Setenv("POSTGRES_CONN_MAX_IDLE_TIME_MINUTES", "5")
	t.Setenv("LLM_PROJECT_ID", "test-project")
	t.Setenv("LLM_LOCATION", "us-central1")
	t.Setenv("LLM_MODEL_NAME", "gemini-pro")
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("LOG_FORMAT", "json")
	t.Setenv("DISCORD_TOKEN", "test-token-123")
	t.Setenv("DISCORD_BOTNAMEREGEXP", "^bot-.*")

	cfg := config.NewConfigRepository().GetConfig()

	// Database config
	assert.Equal(t, "testhost", cfg.Database.Host)
	assert.Equal(t, 5433, cfg.Database.Port)
	assert.Equal(t, "testuser", cfg.Database.User)
	assert.Equal(t, "testpass", cfg.Database.Password)
	assert.Equal(t, "testdb", cfg.Database.Database)
	assert.Equal(t, "require", cfg.Database.SSLMode)
	assert.Equal(t, 50, cfg.Database.MaxOpenConns)
	assert.Equal(t, 10, cfg.Database.MaxIdleConns)
	assert.Equal(t, 15, cfg.Database.ConnMaxLifetimeMin)
	assert.Equal(t, 15*time.Minute, cfg.Database.ConnMaxLifetime)
	assert.Equal(t, 5, cfg.Database.ConnMaxIdleTimeMin)
	assert.Equal(t, 5*time.Minute, cfg.Database.ConnMaxIdleTime)

	// LLM config
	assert.Equal(t, "test-project", cfg.LLM.ProjectID)
	assert.Equal(t, "us-central1", cfg.LLM.Location)
	assert.Equal(t, "gemini-pro", cfg.LLM.ModelName)

	// Log config
	assert.Equal(t, "debug", cfg.Log.Level)
	assert.Equal(t, "json", cfg.Log.Format)

	// Discord config
	assert.Equal(t, "test-token-123", cfg.Discord.Token)
}

func TestConfigRepository_EdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		envVars map[string]string
	}{
		{
			name: "invalid environment values",
			envVars: map[string]string{
				"ENV":                     "test",
				"POSTGRES_PORT":           "invalid",
				"POSTGRES_MAX_OPEN_CONNS": "not_a_number",
			},
		},
		{
			name: "test env file variable",
			envVars: map[string]string{
				"ENV":      "test",
				"TEST_ENV": ".env.nonexistent",
			},
		},
		{
			name: "unknown mode defaults to development",
			envVars: map[string]string{
				"ENV": "unknown_mode",
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			for key, value := range testCase.envVars {
				t.Setenv(key, value)
			}

			cfg := config.NewConfigRepository().GetConfig()
			assert.NotNil(t, cfg)
		})
	}
}

func TestConfigRepository_IndividualConfigs(t *testing.T) {
	t.Run("LLM configuration", func(t *testing.T) {
		t.Setenv("ENV", "test")
		t.Setenv("LLM_PROJECT_ID", "test-proj")
		t.Setenv("LLM_LOCATION", "asia-northeast1")
		t.Setenv("LLM_MODEL_NAME", "gemini-1.5-pro")

		cfg := config.NewConfigRepository().GetConfig()
		assert.Equal(t, "test-proj", cfg.LLM.ProjectID)
		assert.Equal(t, "asia-northeast1", cfg.LLM.Location)
		assert.Equal(t, "gemini-1.5-pro", cfg.LLM.ModelName)
	})

	t.Run("Log configuration", func(t *testing.T) {
		t.Setenv("ENV", "test")
		t.Setenv("LOG_LEVEL", "error")
		t.Setenv("LOG_FORMAT", "text")

		cfg := config.NewConfigRepository().GetConfig()
		assert.Equal(t, "error", cfg.Log.Level)
		assert.Equal(t, "text", cfg.Log.Format)
	})

	t.Run("Discord configuration", func(t *testing.T) {
		t.Setenv("ENV", "test")
		t.Setenv("DISCORD_TOKEN", "test-123")
		t.Setenv("DISCORD_BOTNAMEREGEXP", "^test")

		cfg := config.NewConfigRepository().GetConfig()
		assert.Equal(t, "test-123", cfg.Discord.Token)
	})
}

func TestDatabaseConfig_SSLModes(t *testing.T) {
	t.Parallel()

	sslModes := []string{"disable", "require", "verify-ca", "verify-full"}

	for _, mode := range sslModes {
		t.Run("sslmode_"+mode, func(t *testing.T) {
			t.Parallel()

			dbConfig := config.DatabaseConfig{
				Host:               "localhost",
				Port:               5432,
				User:               "user",
				Password:           "pass",
				Database:           "db",
				SSLMode:            mode,
				MaxOpenConns:       0,
				MaxIdleConns:       0,
				ConnMaxLifetimeMin: 0,
				ConnMaxIdleTimeMin: 0,
				ConnMaxLifetime:    0,
				ConnMaxIdleTime:    0,
				Debug:              false,
			}

			dsn := dbConfig.BuildDSN()
			assert.Contains(t, dsn, "sslmode="+mode)

			url := dbConfig.BuildURL()
			assert.Contains(t, url, "sslmode="+mode)
		})
	}
}
