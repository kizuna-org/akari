package config

import (
	"os"
	"testing"
)

const (
	testAddr        = ":8080"
	testDatabase    = "akari"
	testFallback    = "fallback"
	testHost        = "localhost"
	testPassword    = "secret"
	testPort        = 5432
	testPortEnv     = "POSTGRES_PORT"
	testProduction  = "production"
	testSSLMode     = "disable"
	testUser        = "postgres"
	testValue       = "value"
	testEnvironment = "test"
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
				Host:     testHost,
				Port:     testPort,
				User:     testUser,
				Password: testPassword, // #nosec G101 -- test fixture only.
				Name:     testDatabase,
				SSLMode:  testSSLMode,
			},
			want: "postgres://postgres:secret@localhost:5432/akari?sslmode=disable",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := testCase.db.URL(); got != testCase.want {
				t.Fatalf("URL() = %q, want %q", got, testCase.want)
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
				Host:     testHost,
				Port:     testPort,
				User:     testUser,
				Password: testPassword,
				Name:     testDatabase,
				SSLMode:  testSSLMode,
			},
			want: "host=localhost port=5432 user=postgres password=secret dbname=akari sslmode=disable",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := testCase.db.DSN(); got != testCase.want {
				t.Fatalf("DSN() = %q, want %q", got, testCase.want)
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
				Addr: testAddr,
				Database: Database{
					Host:     testHost,
					Port:     testPort,
					User:     testUser,
					Password: testUser,
					Name:     testDatabase,
					SSLMode:  testSSLMode,
				},
			},
		},
		{
			name: "loads environment overrides",
			env: map[string]string{
				"AKARI_ADDR":        ":9090",
				"POSTGRES_HOST":     "db",
				testPortEnv:         "15432",
				"POSTGRES_USER":     testDatabase,
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
				testPortEnv: "invalid",
			},
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			clearConfigEnv(t)

			for key, value := range testCase.env {
				t.Setenv(key, value)
			}

			got, err := Load()
			if testCase.wantErr {
				if err == nil {
					t.Fatal("Load() error = nil, want error")
				}

				return
			}

			if err != nil {
				t.Fatalf("Load() error = %v", err)
			}

			if got != testCase.want {
				t.Fatalf("Load() = %#v, want %#v", got, testCase.want)
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
		{name: testEnvironment, env: testEnvironment, want: ".env.test"},
		{name: testProduction, env: testProduction, want: ""},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Setenv("ENV", testCase.env)

			if got := envFile(); got != testCase.want {
				t.Fatalf("envFile() = %q, want %q", got, testCase.want)
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
		{name: "uses fallback", fallback: testFallback, want: testFallback},
		{name: "uses env value", value: testValue, fallback: testFallback, want: testValue},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			if testCase.value != "" {
				t.Setenv("AKARI_TEST_VALUE", testCase.value)
			}

			if got := getenv("AKARI_TEST_VALUE", testCase.fallback); got != testCase.want {
				t.Fatalf("getenv() = %q, want %q", got, testCase.want)
			}
		})
	}
}

func clearConfigEnv(t *testing.T) {
	t.Helper()

	keys := []string{
		"AKARI_ADDR",
		"POSTGRES_HOST",
		testPortEnv,
		"POSTGRES_USER",
		"POSTGRES_PASSWORD",
		"POSTGRES_DB",
		"POSTGRES_SSLMODE",
	}
	for _, key := range keys {
		t.Setenv(key, "")
	}

	err := os.Unsetenv("ENV")
	if err != nil {
		t.Fatalf("Unsetenv() error = %v", err)
	}
}
