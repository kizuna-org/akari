package config

import (
	"os"
	"testing"
)

func TestDatabaseURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		db   Database
		want string
	}{
		{
			name: "builds postgres URL",
			db: Database{
				Host:     "localhost",
				Port:     5432,
				User:     "postgres",
				Password: "secret",
				Name:     "akari",
				SSLMode:  "disable",
			},
			want: "postgres://postgres:secret@localhost:5432/akari?sslmode=disable",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tt.db.URL(); got != tt.want {
				t.Fatalf("URL() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDatabaseDSN(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		db   Database
		want string
	}{
		{
			name: "builds postgres DSN",
			db: Database{
				Host:     "localhost",
				Port:     5432,
				User:     "postgres",
				Password: "secret",
				Name:     "akari",
				SSLMode:  "disable",
			},
			want: "host=localhost port=5432 user=postgres password=secret dbname=akari sslmode=disable",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tt.db.DSN(); got != tt.want {
				t.Fatalf("DSN() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestLoad(t *testing.T) {
	tests := []struct {
		name    string
		env     map[string]string
		want    Config
		wantErr bool
	}{
		{
			name: "loads defaults",
			want: Config{
				Addr: ":8080",
				Database: Database{
					Host:     "localhost",
					Port:     5432,
					User:     "postgres",
					Password: "postgres",
					Name:     "akari",
					SSLMode:  "disable",
				},
			},
		},
		{
			name: "loads environment overrides",
			env: map[string]string{
				"AKARI_ADDR":        ":9090",
				"POSTGRES_HOST":     "db",
				"POSTGRES_PORT":     "15432",
				"POSTGRES_USER":     "akari",
				"POSTGRES_PASSWORD": "password",
				"POSTGRES_DB":       "akari_dev",
				"POSTGRES_SSLMODE":  "require",
			},
			want: Config{
				Addr: ":9090",
				Database: Database{
					Host:     "db",
					Port:     15432,
					User:     "akari",
					Password: "password",
					Name:     "akari_dev",
					SSLMode:  "require",
				},
			},
		},
		{
			name: "rejects invalid database port",
			env: map[string]string{
				"POSTGRES_PORT": "invalid",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			clearConfigEnv(t)
			for key, value := range tt.env {
				t.Setenv(key, value)
			}

			got, err := Load()
			if tt.wantErr {
				if err == nil {
					t.Fatal("Load() error = nil, want error")
				}

				return
			}

			if err != nil {
				t.Fatalf("Load() error = %v", err)
			}

			if got != tt.want {
				t.Fatalf("Load() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestEnvFile(t *testing.T) {
	tests := []struct {
		name string
		env  string
		want string
	}{
		{name: "development default", want: ".env"},
		{name: "test", env: "test", want: ".env.test"},
		{name: "production", env: "production", want: ""},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("ENV", tt.env)

			if got := envFile(); got != tt.want {
				t.Fatalf("envFile() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGetenv(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		fallback string
		want     string
	}{
		{name: "uses fallback", fallback: "fallback", want: "fallback"},
		{name: "uses env value", value: "value", fallback: "fallback", want: "value"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			if tt.value != "" {
				t.Setenv("AKARI_TEST_VALUE", tt.value)
			}

			if got := getenv("AKARI_TEST_VALUE", tt.fallback); got != tt.want {
				t.Fatalf("getenv() = %q, want %q", got, tt.want)
			}
		})
	}
}

func clearConfigEnv(t *testing.T) {
	t.Helper()

	keys := []string{
		"AKARI_ADDR",
		"POSTGRES_HOST",
		"POSTGRES_PORT",
		"POSTGRES_USER",
		"POSTGRES_PASSWORD",
		"POSTGRES_DB",
		"POSTGRES_SSLMODE",
	}
	for _, key := range keys {
		t.Setenv(key, "")
	}

	if err := os.Unsetenv("ENV"); err != nil {
		t.Fatalf("Unsetenv() error = %v", err)
	}
}
