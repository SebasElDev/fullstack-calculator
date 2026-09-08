// Package fiberadapter is the delivery mechanism of the service: it maps HTTP
// requests onto the Calculator input port. It is the only package in the
// module that imports the Fiber framework, so replacing Fiber with another
// router touches this package and nothing else.
package fiberadapter

import (
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/helmet"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/gofiber/fiber/v3/middleware/requestid"

	"github.com/SebasElDev/fullstack-calculator/backend/internal/usecase/calculator"
)

const (
	// maxRequestBodyBytes is the application-level cap on request bodies: the
	// largest legal request is an operation name plus two numbers, so a
	// kibibyte is generous. Enforced by the handler so the rejection carries the
	// JSON envelope and is logged like any other 4xx.
	maxRequestBodyBytes = 1024
	// transportBodyLimit is the hard ceiling enforced by the HTTP server while
	// it is still reading the request, before any handler runs. It protects the
	// process from abusive payloads; ordinary oversized bodies never reach it.
	transportBodyLimit = 64 * 1024
	// appName identifies the service in Fiber's config and startup output. No
	// Server header is emitted, which is deliberate.
	appName = "fullstack-calculator"

	// contentSecurityPolicy fits the bundled SPA exactly: same-origin scripts,
	// styles and API calls, inline style attributes (Tailwind/React), data URIs
	// for images, and no framing or plugins whatsoever.
	contentSecurityPolicy = "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; " +
		"img-src 'self' data:; connect-src 'self'; object-src 'none'; frame-ancestors 'none'; " +
		"base-uri 'none'; form-action 'none'"

	apiPrefix   = "/api"
	apiV1Prefix = "/api/v1"

	healthPath     = "/health"
	operationsPath = "/operations"
	calculatePath  = "/calculate"

	// Timeouts for a service whose handlers are pure arithmetic: anything
	// slower than this is a stuck client, not a slow computation.
	readTimeout  = 15 * time.Second
	writeTimeout = 15 * time.Second
	idleTimeout  = 60 * time.Second
)

// Options configures the HTTP application. Only Calculator is mandatory.
type Options struct {
	// Calculator is the input port the handlers delegate to.
	Calculator calculator.Calculator
	// Version is reported by the health endpoints.
	Version string
	// CORSOrigins enables the cors middleware when non-empty. Same-origin
	// deployments (the binary serves the SPA itself) need none.
	CORSOrigins []string
	// StaticDir enables SPA hosting when non-empty.
	StaticDir string
	// Logger receives the structured request log. Defaults to slog.Default().
	Logger *slog.Logger
}

// New builds the HTTP application described in docs/ARCHITECTURE.md §2.5.
func New(opts Options) *fiber.App {
	logger := opts.Logger
	if logger == nil {
		logger = slog.Default()
	}

	h := handlers{calculator: opts.Calculator, version: opts.Version}

	app := fiber.New(fiber.Config{
		AppName:      appName,
		BodyLimit:    transportBodyLimit,
		ErrorHandler: errorHandler(logger),
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
		IdleTimeout:  idleTimeout,
	})

	// The request logger deliberately wraps recover: panics are turned into
	// errors underneath it, so every request — successful, failed or panicking —
	// is logged exactly once with its final status.
	app.Use(requestid.New())
	app.Use(requestLogger(logger))
	app.Use(recover.New())
	app.Use(helmet.New(helmet.Config{
		ContentSecurityPolicy: contentSecurityPolicy,
		XFrameOptions:         "DENY",
	}))

	if len(opts.CORSOrigins) > 0 {
		app.Use(cors.New(cors.Config{
			AllowOrigins: opts.CORSOrigins,
			AllowMethods: []string{fiber.MethodGet, fiber.MethodPost, fiber.MethodHead, fiber.MethodOptions},
			AllowHeaders: []string{fiber.HeaderContentType},
		}))
	}

	// Container platforms probe the prefix-less route.
	app.Get(healthPath, h.health)

	v1 := app.Group(apiV1Prefix)
	v1.Get(healthPath, h.health)
	v1.Get(operationsPath, h.operations)
	v1.Post(calculatePath, h.calculate)

	// Registered after the API routes, so it only sees requests none of them
	// matched: unknown API paths get a JSON 404, never the SPA shell.
	app.Use(apiPrefix, h.notFound)

	if opts.StaticDir != "" {
		mountStatic(app, opts.StaticDir, logger)
	}

	return app
}

// localsKey is the private key type for values stored on the request context,
// so no other package can collide with them.
type localsKey int

// requestLoggedKey holds the status the request logger reported for this
// request. The error boundary compares it with the status it is about to send
// and logs the rejection itself when the two differ.
const requestLoggedKey localsKey = iota

// requestLogger writes one structured line per request.
func requestLogger(logger *slog.Logger) fiber.Handler {
	return func(c fiber.Ctx) error {
		start := time.Now()
		err := c.Next()

		// On the error path the boundary has not written the response yet, so
		// the status is resolved from the error with the very same rules.
		status := c.Response().StatusCode()
		if err != nil {
			status, _ = resolveError(err)
		}

		logger.Info("http request",
			slog.String("method", c.Method()),
			slog.String("path", c.Path()),
			slog.Int("status", status),
			slog.Duration("duration", time.Since(start)),
			slog.String("request_id", c.RequestID()),
		)
		c.Locals(requestLoggedKey, status)
		return err
	}
}
