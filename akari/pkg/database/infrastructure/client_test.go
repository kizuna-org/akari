package infrastructure_test

import (
	"testing"
	"time"

	"github.com/kizuna-org/akari/pkg/config"
	"github.com/kizuna-org/akari/pkg/database/infrastructure"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	t.Parallel()

	cfg := config.DatabaseConfig{
		Host:               "localhost",
		Port:               5432,
		User:               "testuser",
		Password:           "testpass",
		Database:           "testdb",
		SSLMode:            "disable",
		MaxOpenConns:       10,
		MaxIdleConns:       5,
		ConnMaxLifetimeMin: 5,
		ConnMaxIdleTimeMin: 2,
		ConnMaxLifetime:    5 * time.Minute,
		ConnMaxIdleTime:    2 * time.Minute,
		Debug:              true,
	}

	client, err := infrastructure.NewClient(cfg)
	require.NoError(t, err)
	require.NotNil(t, client)

	defer func() {
		assert.NoError(t, client.Close())
	}()

	assert.NotNil(t, client.SystemPromptClient())
	assert.NotNil(t, client.CharacterClient())
	assert.NotNil(t, client.CharacterConfigClient())
}
