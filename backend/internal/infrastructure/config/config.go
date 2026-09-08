// Package config reads the process environment into a validated Config. It is
// the outermost layer: nothing inside the application knows that environment
// variables exist.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Environment variables read by Load.
const (
	EnvPort               = "PORT"
	EnvStaticDir          = "STATIC_DIR"
	EnvCORSAllowedOrigins = "CORS_ALLOWED_ORIGINS"
	EnvShutdownTimeout    = "SHUTDOWN_TIMEOUT"
)

// Defaults applied when a variable is unset or empty.
const (
	DefaultPort            = 8080
	DefaultShutdownTimeout = 10 * time.Second
)

const (
	minPort = 1
	maxPort = 65535
	// originSeparator splits CORS_ALLOWED_ORIGINS.
	originSeparator = ","
)

// Config is the runtime configuration of the API server.
type Config struct {
	// Port is the TCP port the HTTP server listens on.
	Port int
	// StaticDir is the directory holding the built SPA. Empty disables static hosting.
	StaticDir string
	// CORSOrigins lists the allowed browser origins. Empty disables CORS.
	CORSOrigins []string
	// ShutdownTimeout bounds the graceful shutdown of in-flight requests.
	ShutdownTimeout time.Duration
}

// Address returns the listen address for the configured port.
func (c Config) Address() string { return ":" + strconv.Itoa(c.Port) }

// Load reads the environment and validates it. Every variable is optional; an
// invalid value is an error rather than a silent fallback.
func Load() (Config, error) {
	port, err := loadPort()
	if err != nil {
		return Config{}, err
	}

	staticDir, err := loadStaticDir()
	if err != nil {
		return Config{}, err
	}

	shutdownTimeout, err := loadShutdownTimeout()
	if err != nil {
		return Config{}, err
	}

	return Config{
		Port:            port,
		StaticDir:       staticDir,
		CORSOrigins:     loadCORSOrigins(),
		ShutdownTimeout: shutdownTimeout,
	}, nil
}

func loadPort() (int, error) {
	raw := lookup(EnvPort)
	if raw == "" {
		return DefaultPort, nil
	}
	port, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("%s: %q is not a number", EnvPort, raw)
	}
	if port < minPort || port > maxPort {
		return 0, fmt.Errorf("%s: %d is outside the valid range %d-%d", EnvPort, port, minPort, maxPort)
	}
	return port, nil
}

func loadStaticDir() (string, error) {
	raw := lookup(EnvStaticDir)
	if raw == "" {
		return "", nil
	}
	info, err := os.Stat(raw)
	if err != nil {
		return "", fmt.Errorf("%s: %w", EnvStaticDir, err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("%s: %q is not a directory", EnvStaticDir, raw)
	}
	return raw, nil
}

func loadShutdownTimeout() (time.Duration, error) {
	raw := lookup(EnvShutdownTimeout)
	if raw == "" {
		return DefaultShutdownTimeout, nil
	}
	timeout, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("%s: %q is not a duration such as 10s", EnvShutdownTimeout, raw)
	}
	if timeout <= 0 {
		return 0, fmt.Errorf("%s: %s must be positive", EnvShutdownTimeout, timeout)
	}
	return timeout, nil
}

// loadCORSOrigins splits a comma-separated list, dropping blanks. An empty
// result disables the cors middleware, which is what same-origin deployments
// want.
func loadCORSOrigins() []string {
	raw := lookup(EnvCORSAllowedOrigins)
	if raw == "" {
		return nil
	}
	var origins []string
	for _, origin := range strings.Split(raw, originSeparator) {
		if trimmed := strings.TrimSpace(origin); trimmed != "" {
			origins = append(origins, trimmed)
		}
	}
	return origins
}

// lookup reads a variable and trims the surrounding whitespace shells tend to
// leave behind.
func lookup(name string) string {
	return strings.TrimSpace(os.Getenv(name))
}
