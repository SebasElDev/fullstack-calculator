/// <reference types="vitest/config" />
import { fileURLToPath } from "node:url";
import tailwindcss from "@tailwindcss/vite";
import react from "@vitejs/plugin-react";
import { defineConfig } from "vite";

/** The Go backend the dev server proxies API traffic to. */
const BACKEND_ORIGIN = "http://localhost:8080";

/** Paths the Go binary owns; everything else is served by Vite in development. */
const PROXIED_PATHS = ["/api", "/health"];

export default defineConfig({
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: {
      "@": fileURLToPath(new URL("./src", import.meta.url)),
    },
  },
  server: {
    port: 5173,
    proxy: Object.fromEntries(
      PROXIED_PATHS.map((path) => [path, { target: BACKEND_ORIGIN, changeOrigin: true }]),
    ),
  },
  test: {
    globals: false,
    environment: "jsdom",
    setupFiles: ["src/test/setup.ts"],
    include: ["src/**/*.test.{ts,tsx}"],
    restoreMocks: true,
    coverage: {
      provider: "v8",
      reporter: ["text", "html"],
      include: ["src/features/**", "src/lib/**"],
      thresholds: {
        lines: 85,
        statements: 85,
        branches: 85,
        functions: 85,
      },
    },
  },
});
