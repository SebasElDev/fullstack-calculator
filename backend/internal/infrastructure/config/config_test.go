package config_test

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/SebasElDev/fullstack-calculator/backend/internal/infrastructure/config"
)

// setEnv applies an environment for one test. Unset variables are cleared, so a
// stray value in the developer's shell cannot influence the result.
func setEnv(t *testing.T, env map[string]string) {
	t.Helper()
	for _, name := range []string{config.EnvPort, config.EnvStaticDir, config.EnvCORSAllowedOrigins, config.EnvShutdownTimeout} {
		t.Setenv(name, env[name])
	}
}

func TestLoadDefaults(t *testing.T) {
	setEnv(t, nil)

	got, err := config.Load()
	if err != nil {
		t.Fatalf("Load() returned unexpected error: %v", err)
	}

	want := config.Config{Port: config.DefaultPort, ShutdownTimeout: config.DefaultShutdownTimeout}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Load() = %+v; want %+v", got, want)
	}
	if got.Address() != ":8080" {
		t.Errorf("Address() = %q; want %q", got.Address(), ":8080")
	}
}

func TestLoadValidEnvironments(t *testing.T) {
	staticDir := t.TempDir()

	tests := []struct {
		name string
		env  map[string]string
		want config.Config
	}{
		{
			name: "everything set",
			env: map[string]string{
				config.EnvPort:               "9090",
				config.EnvStaticDir:          staticDir,
				config.EnvCORSAllowedOrigins: "https://a.example,https://b.example",
				config.EnvShutdownTimeout:    "30s",
			},
			want: config.Config{
				Port:            9090,
				StaticDir:       staticDir,
				CORSOrigins:     []string{"https://a.example", "https://b.example"},
				ShutdownTimeout: 30 * time.Second,
			},
		},
		{
			name: "whitespace is trimmed everywhere",
			env: map[string]string{
				config.EnvPort:               " 3000 ",
				config.EnvCORSAllowedOrigins: " https://a.example , , https://b.example ",
				config.EnvShutdownTimeout:    " 1m ",
			},
			want: config.Config{
				Port:            3000,
				CORSOrigins:     []string{"https://a.example", "https://b.example"},
				ShutdownTimeout: time.Minute,
			},
		},
		{
			name: "a single origin",
			env:  map[string]string{config.EnvCORSAllowedOrigins: "http://localhost:5173"},
			want: config.Config{
				Port:            config.DefaultPort,
				CORSOrigins:     []string{"http://localhost:5173"},
				ShutdownTimeout: config.DefaultShutdownTimeout,
			},
		},
		{
			name: "only separators means no origins",
			env:  map[string]string{config.EnvCORSAllowedOrigins: " , , "},
			want: config.Config{Port: config.DefaultPort, ShutdownTimeout: config.DefaultShutdownTimeout},
		},
		{
			name: "lowest and highest port",
			env:  map[string]string{config.EnvPort: "65535"},
			want: config.Config{Port: 65535, ShutdownTimeout: config.DefaultShutdownTimeout},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			setEnv(t, tc.env)

			got, err := config.Load()
			if err != nil {
				t.Fatalf("Load() returned unexpected error: %v", err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("Load() = %+v; want %+v", got, tc.want)
			}
		})
	}
}

func TestLoadRejectsInvalidEnvironments(t *testing.T) {
	file := filepath.Join(t.TempDir(), "index.html")
	if err := os.WriteFile(file, []byte("<!doctype html>"), 0o600); err != nil {
		t.Fatalf("writing the fixture failed: %v", err)
	}

	tests := []struct {
		name string
		env  map[string]string
	}{
		{name: "port is not a number", env: map[string]string{config.EnvPort: "http"}},
		{name: "port is zero", env: map[string]string{config.EnvPort: "0"}},
		{name: "port is negative", env: map[string]string{config.EnvPort: "-1"}},
		{name: "port is above the range", env: map[string]string{config.EnvPort: "65536"}},
		{name: "static dir does not exist", env: map[string]string{config.EnvStaticDir: filepath.Join(t.TempDir(), "missing")}},
		{name: "static dir is a file", env: map[string]string{config.EnvStaticDir: file}},
		{name: "shutdown timeout is not a duration", env: map[string]string{config.EnvShutdownTimeout: "soon"}},
		{name: "shutdown timeout is zero", env: map[string]string{config.EnvShutdownTimeout: "0s"}},
		{name: "shutdown timeout is negative", env: map[string]string{config.EnvShutdownTimeout: "-5s"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			setEnv(t, tc.env)

			got, err := config.Load()
			if err == nil {
				t.Fatalf("Load() = %+v, nil; want an error", got)
			}
			if !reflect.DeepEqual(got, config.Config{}) {
				t.Errorf("Load() = %+v alongside an error; want the zero value", got)
			}
		})
	}
}

func TestAddress(t *testing.T) {
	t.Parallel()

	if got := (config.Config{Port: 18080}).Address(); got != ":18080" {
		t.Fatalf("Address() = %q; want %q", got, ":18080")
	}
}
