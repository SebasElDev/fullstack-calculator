import { render, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

afterEach(() => {
  vi.unstubAllGlobals();
  vi.resetModules();
});

/** Imports App only after the global fetch is stubbed, since it builds its client at module load. */
async function renderApp(fetchMock: typeof globalThis.fetch) {
  vi.resetModules();
  vi.stubGlobal("fetch", fetchMock);
  const { default: App } = await import("@/App");
  return render(<App />);
}

describe("<App>", () => {
  it("renders the calculator wired to the real API client", async () => {
    const fetchMock = vi.fn(
      async () =>
        ({
          ok: true,
          status: 200,
          json: async () => ({ status: "ok", version: "1.0.0" }),
        }) as unknown as Response,
    );

    await renderApp(fetchMock as unknown as typeof globalThis.fetch);

    expect(screen.getByRole("heading", { level: 1, name: "Calculator" })).toBeInTheDocument();
    expect(screen.getByTestId("calculator")).toBeInTheDocument();
    expect(screen.getByTestId("display-value")).toHaveTextContent("0");
    expect(screen.getAllByRole("button").length).toBeGreaterThan(0);

    await waitFor(() => {
      expect(screen.getByTestId("status")).toHaveAttribute("data-health", "online");
    });
    expect(fetchMock).toHaveBeenCalledWith("/api/v1/health", expect.anything());
  });

  it("survives an unreachable API", async () => {
    const fetchMock = vi.fn(async () => {
      throw new TypeError("Failed to fetch");
    });

    await renderApp(fetchMock as unknown as typeof globalThis.fetch);

    await waitFor(() => {
      expect(screen.getByTestId("status")).toHaveAttribute("data-health", "offline");
    });
  });
});
