package fiberadapter

import (
	"log/slog"
	"os"
	"path/filepath"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/static"
)

const (
	// indexFile is the SPA shell served for every client-side route.
	indexFile = "index.html"
	// assetsPrefix is where Vite emits its content-hashed bundles.
	assetsPrefix = "/assets"
	// immutableCacheControl suits content-hashed files: a changed file gets a
	// new name, so the old one can be cached forever.
	immutableCacheControl = "public, max-age=31536000, immutable"
	// shellCacheControl makes browsers revalidate the shell (and any other
	// unhashed file) on every navigation, so a redeploy is picked up at once.
	shellCacheControl = "no-cache"
)

// mountStatic serves the built single-page application from dir.
//
// Hashed bundles under /assets are cached immutably and a missing one is an
// honest 404 — never the shell, which a module script would reject anyway.
// Every other file is served as found with revalidation, and any other GET
// falls back to the SPA shell so client-side routes survive a page reload. API
// routes never reach these handlers: the /api catch-all is registered first.
func mountStatic(app *fiber.App, dir string, logger *slog.Logger) {
	shell, err := os.ReadFile(filepath.Join(dir, indexFile))
	if err != nil {
		// Serving assets without a shell is still useful; the 404 below is
		// honest about the missing file.
		logger.Warn("spa shell is unavailable",
			slog.String("static_dir", dir),
			slog.Any("error", err),
		)
	}
	hasShell := err == nil

	app.Get(assetsPrefix+"/*", static.New(filepath.Join(dir, assetsPrefix), static.Config{
		ModifyResponse: setCacheControl(immutableCacheControl),
		NotFoundHandler: func(c fiber.Ctx) error {
			return newNotFoundError(c.Method(), c.Path())
		},
	}))

	app.Get("/*", static.New(dir, static.Config{
		IndexNames:     []string{indexFile},
		ModifyResponse: setCacheControl(shellCacheControl),
		NotFoundHandler: func(c fiber.Ctx) error {
			if !hasShell {
				return newNotFoundError(c.Method(), c.Path())
			}
			c.Set(fiber.HeaderCacheControl, shellCacheControl)
			return c.Status(fiber.StatusOK).Type("html", "utf-8").Send(shell)
		},
	}))
}

// setCacheControl is a static.Config.ModifyResponse hook that stamps the
// given Cache-Control value on every file the handler serves.
func setCacheControl(value string) fiber.Handler {
	return func(c fiber.Ctx) error {
		c.Set(fiber.HeaderCacheControl, value)
		return nil
	}
}
