package infrastructure_test

import (
	"testing"

	"github.com/kizuna-org/akari/pkg/discord/infrastructure"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDiscordClient(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		token   string
		wantErr bool
	}{
		{
			name:    "success",
			token:   "test-token",
			wantErr: false,
		},
		{
			name:    "empty token",
			token:   "",
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			client, err := infrastructure.NewDiscordClient(testCase.token)

			if testCase.wantErr {
				require.Error(t, err)
				assert.Nil(t, client)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, client)
				assert.NotNil(t, client.Session)
				assert.NotNil(t, client.Session.Identify.Intents)
			}
		})
	}
}
