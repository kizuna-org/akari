package config

import (
	"os"
	"testing"
)

func TestLoadFromEnv(t *testing.T) {
	tests := []struct {
		name    string
		envVars map[string]string
		want    *Config
		wantErr bool
	}{
		{
			name:    "default values",
			envVars: map[string]string{},
			want: &Config{
				Qdrant: QdrantConfig{
					Host:       "localhost",
					Port:       6334,
					UseTLS:     false,
					VectorSize: 768,
				},
				Redis: RedisConfig{
					Host:     "localhost",
					Port:     6379,
					Password: "",
					DB:       0,
				},
				Score: ScoreConfig{
					Alpha:   0.5,
					Beta:    0.3,
					Gamma:   0.2,
					Epsilon: 0.1,
				},
			},
			wantErr: false,
		},
		{
			name: "custom values",
			envVars: map[string]string{
				"QDRANT_HOST":        "qdrant.example.com",
				"QDRANT_PORT":        "6333",
				"QDRANT_USE_TLS":     "true",
				"QDRANT_VECTOR_SIZE": "1024",
				"REDIS_HOST":         "redis.example.com",
				"REDIS_PORT":         "6380",
				"REDIS_PASSWORD":     "secret",
				"REDIS_DB":           "1",
				"SCORE_ALPHA":        "0.4",
				"SCORE_BETA":         "0.4",
				"SCORE_GAMMA":        "0.2",
				"SCORE_EPSILON":      "0.05",
			},
			want: &Config{
				Qdrant: QdrantConfig{
					Host:       "qdrant.example.com",
					Port:       6333,
					UseTLS:     true,
					VectorSize: 1024,
				},
				Redis: RedisConfig{
					Host:     "redis.example.com",
					Port:     6380,
					Password: "secret",
					DB:       1,
				},
				Score: ScoreConfig{
					Alpha:   0.4,
					Beta:    0.4,
					Gamma:   0.2,
					Epsilon: 0.05,
				},
			},
			wantErr: false,
		},
		{
			name: "invalid weights sum",
			envVars: map[string]string{
				"SCORE_ALPHA": "0.5",
				"SCORE_BETA":  "0.5",
				"SCORE_GAMMA": "0.5",
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			for key, value := range tt.envVars {
				os.Setenv(key, value)
			}
			defer func() {
				// Clean up environment variables
				for key := range tt.envVars {
					os.Unsetenv(key)
				}
			}()

			got, err := LoadFromEnv()
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadFromEnv() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}

			// Compare each field
			if got.Qdrant.Host != tt.want.Qdrant.Host {
				t.Errorf("Qdrant.Host = %v, want %v", got.Qdrant.Host, tt.want.Qdrant.Host)
			}
			if got.Qdrant.Port != tt.want.Qdrant.Port {
				t.Errorf("Qdrant.Port = %v, want %v", got.Qdrant.Port, tt.want.Qdrant.Port)
			}
			if got.Redis.Host != tt.want.Redis.Host {
				t.Errorf("Redis.Host = %v, want %v", got.Redis.Host, tt.want.Redis.Host)
			}
			if got.Score.Alpha != tt.want.Score.Alpha {
				t.Errorf("Score.Alpha = %v, want %v", got.Score.Alpha, tt.want.Score.Alpha)
			}
		})
	}
}
