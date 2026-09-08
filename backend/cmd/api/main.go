// Command api is the composition root of the calculator service: it reads the
// configuration, wires the use case into the HTTP adapter and runs the server
// until the platform asks it to stop.
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v3"

	fiberadapter "github.com/SebasElDev/fullstack-calculator/backend/internal/adapter/http/fiber"
	"github.com/SebasElDev/fullstack-calculator/backend/internal/infrastructure/config"
	"github.com/SebasElDev/fullstack-calculator/backend/internal/usecase/calculator"
)

// version is injected at build time with
// -ldflags "-X main.version=<git sha or semver>".
var version = "dev"

// healthcheckTimeout bounds the -healthcheck probe used by the container
// HEALTHCHECK, which has no client of its own.
const healthcheckTimeout = 3 * time.Second

func main() {
	healthcheck := flag.Bool("healthcheck", false, "probe the local health endpoint and exit 0 when it is healthy")
	flag.Parse()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		logger.Error("invalid configuration", slog.Any("error", err))
		os.Exit(1)
	}

	if *healthcheck {
		if err := probeHealth(cfg.Port); err != nil {
			logger.Error("health check failed", slog.Any("error", err))
			os.Exit(1)
		}
		return
	}

	if err := run(cfg, logger); err != nil {
		logger.Error("server stopped with an error", slog.Any("error", err))
		os.Exit(1)
	}
}

// run serves until a termination signal arrives, then drains in-flight requests.
func run(cfg config.Config, logger *slog.Logger) error {
	app := newApp(cfg, logger)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	serverErr := make(chan error, 1)
	go func() {
		serverErr <- app.Listen(cfg.Address(), fiber.ListenConfig{DisableStartupMessage: true})
	}()

	logger.Info("server started",
		slog.String("version", version),
		slog.String("address", cfg.Address()),
		slog.String("static_dir", cfg.StaticDir),
		slog.Any("cors_origins", cfg.CORSOrigins),
	)

	select {
	case err := <-serverErr:
		return fmt.Errorf("listen on %s: %w", cfg.Address(), err)
	case <-ctx.Done():
		stop() // a second signal now kills the process instead of waiting
		logger.Info("shutdown requested", slog.Duration("timeout", cfg.ShutdownTimeout))
		if err := app.ShutdownWithTimeout(cfg.ShutdownTimeout); err != nil {
			return fmt.Errorf("graceful shutdown: %w", err)
		}
		logger.Info("server stopped")
		return nil
	}
}

// newApp wires the layers together: the use case is the only implementation the
// HTTP adapter ever sees.
func newApp(cfg config.Config, logger *slog.Logger) *fiber.App {
	return fiberadapter.New(fiberadapter.Options{
		Calculator:  calculator.NewService(),
		Version:     version,
		CORSOrigins: cfg.CORSOrigins,
		StaticDir:   cfg.StaticDir,
		Logger:      logger,
	})
}

// probeHealth performs the container health check: a plain GET against the
// loopback interface, so the image needs no shell and no curl.
func probeHealth(port int) error {
	url := "http://" + net.JoinHostPort("127.0.0.1", strconv.Itoa(port)) + "/health"

	ctx, cancel := context.WithTimeout(context.Background(), healthcheckTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("build the probe request: %w", err)
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("probe %s: %w", url, err)
	}
	defer func() { _ = res.Body.Close() }()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("probe %s: status %d, want %d", url, res.StatusCode, http.StatusOK)
	}
	return nil
}
