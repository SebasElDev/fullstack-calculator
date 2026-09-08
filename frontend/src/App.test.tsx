import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
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
  return { user: userEvent.setup(), ...render(<App />) };
}

/** Presses `2 + 3 =` on the rendered keypad. */
async function calculateTwoPlusThree(user: ReturnType<typeof userEvent.setup>) {
  await user.click(screen.getByTestId("key-2"));
  await user.click(screen.getByTestId("key-add"));
  await user.click(screen.getByTestId("key-3"));
  await user.click(screen.getByTestId("key-equals"));
}

describe("<App>", () => {
  it("renders the calculator wired to the real API client", async () => {
    const fetchMock = vi.fn(
      async () =>
        ({
          ok: true,
          status: 200,
          json: async () => ({ operation: "add", operands: [2, 3], result: 42 }),
        }) as unknown as Response,
    );

    const { user } = await renderApp(fetchMock as unknown as typeof globalThis.fetch);

    expect(screen.getByRole("heading", { level: 1, name: "Calculator" })).toBeInTheDocument();
    expect(screen.getByTestId("calculator")).toBeInTheDocument();
    expect(screen.getByTestId("display-value")).toHaveTextContent("0");
    expect(screen.getAllByRole("button").length).toBeGreaterThan(0);

    // The shell composes no status strip and makes no request until a key is pressed.
    expect(screen.queryByTestId("status")).not.toBeInTheDocument();
    expect(fetchMock).not.toHaveBeenCalled();

    await calculateTwoPlusThree(user);

    await waitFor(() => {
      expect(screen.getByTestId("display-value")).toHaveTextContent("42");
    });
    expect(fetchMock).toHaveBeenCalledWith("/api/v1/calculate", expect.anything());
  });

  it("surfaces an unreachable API in the display instead of crashing", async () => {
    const fetchMock = vi.fn(async () => {
      throw new TypeError("Failed to fetch");
    });

    const { user } = await renderApp(fetchMock as unknown as typeof globalThis.fetch);
    await calculateTwoPlusThree(user);

    await waitFor(() => {
      expect(screen.getByTestId("display")).toHaveAttribute("data-status", "error");
    });
    expect(screen.getByTestId("display-value")).toHaveTextContent("Failed to fetch");
    expect(screen.getByTestId("calculator")).toBeInTheDocument();
  });
});
