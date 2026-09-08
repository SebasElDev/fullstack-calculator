package fiberadapter_test

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SebasElDev/fullstack-calculator/backend/internal/adapter/http/dto"
	fiberadapter "github.com/SebasElDev/fullstack-calculator/backend/internal/adapter/http/fiber"
)

// newSPADir lays out a minimal Vite build: a shell, a hashed bundle and an
// unhashed public file.
func newSPADir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "index.html"), "<!doctype html><title>shell</title>")
	if err := os.MkdirAll(filepath.Join(dir, "assets"), 0o755); err != nil {
		t.Fatalf("creating assets dir: %v", err)
	}
	writeFile(t, filepath.Join(dir, "assets", "index-abc123.js"), "console.log('bundle')")
	writeFile(t, filepath.Join(dir, "robots.txt"), "User-agent: *\n")
	return dir
}

func TestSecurityHeadersOnEveryResponse(t *testing.T) {
	t.Parallel()

	app := newTestApp(t, fiberadapter.Options{Calculator: &fakeCalculator{}, StaticDir: newSPADir(t)})

	for _, target := range []string{"/", "/api/v1/health", "/api/v1/nope", "/some/client/route"} {
		t.Run(target, func(t *testing.T) {
			t.Parallel()

			res, _ := do(t, app, http.MethodGet, target, "")
			checks := map[string]string{
				"Content-Security-Policy": "default-src 'self'",
				"X-Content-Type-Options":  "nosniff",
				"X-Frame-Options":         "DENY",
				"Referrer-Policy":         "",
			}
			for header, want := range checks {
				got := res.Header.Get(header)
				if got == "" {
					t.Errorf("%s header is missing", header)
					continue
				}
				if !strings.Contains(got, want) {
					t.Errorf("%s = %q; want it to contain %q", header, got, want)
				}
			}
			if csp := res.Header.Get("Content-Security-Policy"); !strings.Contains(csp, "frame-ancestors 'none'") {
				t.Errorf("Content-Security-Policy = %q; want frame-ancestors 'none'", csp)
			}
			if got := res.Header.Get("Server"); got != "" {
				t.Errorf("Server header = %q; want none", got)
			}
		})
	}
}

func TestStaticCachingPolicy(t *testing.T) {
	t.Parallel()

	app := newTestApp(t, fiberadapter.Options{Calculator: &fakeCalculator{}, StaticDir: newSPADir(t)})

	tests := []struct {
		name         string
		target       string
		wantStatus   int
		wantCache    string
		wantBodyPart string
	}{
		{name: "hashed bundle is immutable", target: "/assets/index-abc123.js", wantStatus: http.StatusOK, wantCache: "public, max-age=31536000, immutable", wantBodyPart: "bundle"},
		{name: "shell at root revalidates", target: "/", wantStatus: http.StatusOK, wantCache: "no-cache", wantBodyPart: "shell"},
		{name: "client route gets the shell and revalidates", target: "/history/42", wantStatus: http.StatusOK, wantCache: "no-cache", wantBodyPart: "shell"},
		{name: "unhashed public file revalidates", target: "/robots.txt", wantStatus: http.StatusOK, wantCache: "no-cache", wantBodyPart: "User-agent"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			res, payload := do(t, app, http.MethodGet, tc.target, "")
			if res.StatusCode != tc.wantStatus {
				t.Fatalf("GET %s = %d; want %d. body: %s", tc.target, res.StatusCode, tc.wantStatus, payload)
			}
			if got := res.Header.Get("Cache-Control"); got != tc.wantCache {
				t.Errorf("GET %s Cache-Control = %q; want %q", tc.target, got, tc.wantCache)
			}
			if !strings.Contains(string(payload), tc.wantBodyPart) {
				t.Errorf("GET %s body = %q; want it to contain %q", tc.target, payload, tc.wantBodyPart)
			}
		})
	}

	t.Run("a missing bundle is a json 404, never the shell", func(t *testing.T) {
		t.Parallel()

		res, payload := do(t, app, http.MethodGet, "/assets/index-stale.js", "")
		if res.StatusCode != http.StatusNotFound {
			t.Fatalf("GET /assets/index-stale.js = %d; want 404. body: %s", res.StatusCode, payload)
		}
		if got := decode[dto.ErrorResponse](t, payload); got.Error.Code != dto.CodeNotFound {
			t.Errorf("code = %q; want %q", got.Error.Code, dto.CodeNotFound)
		}
	})
}
