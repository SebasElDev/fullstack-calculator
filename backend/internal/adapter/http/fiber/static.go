package fiberadapter

import (
	"log/slog"
	"os"
	"path/filepath"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/static"
)

// indexFile is the SPA shell served for every client-side route.
const indexFile = "index.html"

// mountStatic serves the built single-page application from dir.
//
// Files are served as they are found; any other GET falls back to the SPA
// shell, so client-side routes survive a page reload. API routes never reach
// this handler: the /api catch-all is registered before it.
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

	app.Get("/*", static.New(dir, static.Config{
		IndexNames: []string{indexFile},
		NotFoundHandler: func(c fiber.Ctx) error {
			if !hasShell {
				return newNotFoundError(c.Method(), c.Path())
			}
			return c.Status(fiber.StatusOK).Type("html", "utf-8").Send(shell)
		},
	}))
}
