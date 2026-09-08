// Package internal_test enforces the Clean Architecture dependency rule by
// inspecting the import graph of every package in the module with go/build.
// Source code dependencies must point inwards
// only: the domain knows nothing, the use case knows the domain, the adapters
// know the use case, and only the frameworks layer knows everything.
package internal_test

import (
	"go/build"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const modulePath = "github.com/SebasElDev/fullstack-calculator/backend"

// The inner packages, spelled once.
const (
	domainPkg  = modulePath + "/internal/domain"
	usecasePkg = modulePath + "/internal/usecase/calculator"
	dtoPkg     = modulePath + "/internal/adapter/http/dto"
	fiberPkg   = "github.com/gofiber/fiber/v3"
)

// rule lists, for one package, the non-standard-library imports it is allowed
// to have. A path matches an allowance when it equals it or is nested under it.
type rule struct {
	// dir is the package directory, relative to the module root.
	dir string
	// layer is used in failure messages.
	layer string
	// allowed are the import paths this package may depend on. Anything from
	// the standard library is always allowed.
	allowed []string
}

var rules = []rule{
	{
		dir:     "internal",
		layer:   "architecture test",
		allowed: nil,
	},
	{
		dir:     "internal/domain",
		layer:   "entities",
		allowed: nil,
	},
	{
		dir:     "internal/usecase/calculator",
		layer:   "use cases",
		allowed: []string{domainPkg},
	},
	{
		dir:     "internal/adapter/http/dto",
		layer:   "interface adapters",
		allowed: []string{domainPkg, usecasePkg},
	},
	{
		dir:     "internal/adapter/http/fiber",
		layer:   "interface adapters",
		allowed: []string{domainPkg, usecasePkg, dtoPkg, fiberPkg},
	},
	{
		dir:     "internal/infrastructure/config",
		layer:   "frameworks and drivers",
		allowed: nil,
	},
	{
		dir:   "cmd/api",
		layer: "frameworks and drivers",
		// The composition root may reach into every layer; it may not grow
		// dependencies on anything else.
		allowed: []string{modulePath + "/internal", fiberPkg},
	},
}

func TestDependencyRule(t *testing.T) {
	t.Parallel()

	root := moduleRoot(t)

	for _, r := range rules {
		t.Run(r.dir, func(t *testing.T) {
			t.Parallel()

			// An external test package (package x_test) imports the package it
			// exercises; that is not a layer violation.
			own := modulePath + "/" + r.dir

			for _, imported := range importsOf(t, filepath.Join(root, r.dir)) {
				if imported == own || isStandardLibrary(imported) || isAllowed(imported, r.allowed) {
					continue
				}
				t.Errorf("%s (%s layer) imports %q, which the dependency rule forbids; allowed: %v",
					r.dir, r.layer, imported, r.allowed)
			}
		})
	}
}

// TestEveryPackageHasARule makes the table above exhaustive: a new package
// cannot join the module without declaring the layer it belongs to.
func TestEveryPackageHasARule(t *testing.T) {
	t.Parallel()

	root := moduleRoot(t)
	covered := make(map[string]bool, len(rules))
	for _, r := range rules {
		covered[filepath.FromSlash(r.dir)] = true
	}

	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() {
			return nil
		}
		if name := entry.Name(); path != root && strings.HasPrefix(name, ".") {
			return filepath.SkipDir
		}
		if !containsGoFiles(t, path) {
			return nil
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if relative == "." {
			return nil
		}
		if !covered[relative] {
			t.Errorf("package %s has no entry in the dependency rule table", filepath.ToSlash(relative))
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking the module failed: %v", err)
	}
}

// TestTheFrameworkStaysInItsPackage is the headline guarantee: Fiber lives in
// the adapter and in the composition root, nowhere else.
func TestTheFrameworkStaysInItsPackage(t *testing.T) {
	t.Parallel()

	root := moduleRoot(t)
	allowed := map[string]bool{
		"internal/adapter/http/fiber": true,
		"cmd/api":                     true,
	}

	for _, r := range rules {
		if allowed[r.dir] {
			continue
		}
		for _, imported := range importsOf(t, filepath.Join(root, r.dir)) {
			if isAllowed(imported, []string{fiberPkg}) {
				t.Errorf("%s imports %q; the web framework must stay inside internal/adapter/http/fiber", r.dir, imported)
			}
		}
	}
}

// importsOf returns every import of a package, including the ones its tests
// use: a test file is as much a part of the layer as the code it exercises.
func importsOf(t *testing.T, dir string) []string {
	t.Helper()

	pkg, err := build.ImportDir(dir, build.ImportComment)
	if err != nil {
		t.Fatalf("inspecting %s failed: %v", dir, err)
	}

	imports := make([]string, 0, len(pkg.Imports)+len(pkg.TestImports)+len(pkg.XTestImports))
	imports = append(imports, pkg.Imports...)
	imports = append(imports, pkg.TestImports...)
	imports = append(imports, pkg.XTestImports...)
	return imports
}

// isStandardLibrary reports whether an import path belongs to the standard
// library. Only module paths contain a dot in their first segment.
func isStandardLibrary(importPath string) bool {
	first, _, _ := strings.Cut(importPath, "/")
	return !strings.Contains(first, ".")
}

func isAllowed(importPath string, allowed []string) bool {
	for _, prefix := range allowed {
		if importPath == prefix || strings.HasPrefix(importPath, prefix+"/") {
			return true
		}
	}
	return false
}

func containsGoFiles(t *testing.T, dir string) bool {
	t.Helper()

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("reading %s failed: %v", dir, err)
	}
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".go") {
			return true
		}
	}
	return false
}

// moduleRoot returns the backend module root: the tests run inside
// backend/internal.
func moduleRoot(t *testing.T) string {
	t.Helper()

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("resolving the working directory failed: %v", err)
	}
	root := filepath.Dir(wd)
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		t.Fatalf("%s does not look like the module root: %v", root, err)
	}
	return root
}
