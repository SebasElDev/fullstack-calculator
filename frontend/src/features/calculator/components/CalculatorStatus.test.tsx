import { act, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { Calculator, HEALTH_POLL_INTERVAL_MS } from "@/features/calculator";
import type { CalculateResponse, HealthResponse } from "@/lib/api/types";
import {
  createDeferred,
  createMockApiClient,
  type MockApiClient,
  renderWithProviders,
} from "@/test/utils";

function renderStatus(client: MockApiClient) {
  return renderWithProviders(
    <Calculator>
      <Calculator.Status />
      <Calculator.Keypad />
    </Calculator>,
    { client },
  );
}

function status(): HTMLElement {
  return screen.getByTestId("status");
}

afterEach(() => {
  vi.useRealTimers();
});

describe("<Calculator.Status>", () => {
  it("reports the API as online once /health answers", async () => {
    const deferred = createDeferred<HealthResponse>();
    const client = createMockApiClient();
    client.health.mockReturnValueOnce(deferred.promise);
    renderStatus(client);

    expect(status()).toHaveAttribute("data-health", "checking");
    expect(status()).toHaveTextContent("Checking API");

    await act(async () => {
      deferred.resolve({ status: "ok", version: "1.2.3" });
    });

    expect(status()).toHaveAttribute("data-health", "online");
    expect(status()).toHaveTextContent("API online");
    expect(screen.getByTestId("status-version")).toHaveTextContent("1.2.3");
  });

  it("reports the API as offline when /health fails", async () => {
    const client = createMockApiClient();
    client.health.mockRejectedValue(new Error("connection refused"));
    renderStatus(client);

    await waitFor(() => {
      expect(status()).toHaveAttribute("data-health", "offline");
    });
    expect(status()).toHaveTextContent("API offline");
    expect(screen.queryByTestId("status-version")).not.toBeInTheDocument();
  });

  it("re-checks health on an interval", async () => {
    vi.useFakeTimers();
    const client = createMockApiClient();
    renderStatus(client);

    await act(async () => {
      await vi.advanceTimersByTimeAsync(0);
    });
    expect(client.health).toHaveBeenCalledTimes(1);

    await act(async () => {
      await vi.advanceTimersByTimeAsync(HEALTH_POLL_INTERVAL_MS);
    });
    expect(client.health).toHaveBeenCalledTimes(2);

    await act(async () => {
      await vi.advanceTimersByTimeAsync(HEALTH_POLL_INTERVAL_MS);
    });
    expect(client.health).toHaveBeenCalledTimes(3);
  });

  it("stops polling once unmounted", async () => {
    vi.useFakeTimers();
    const client = createMockApiClient();
    const { unmount } = renderStatus(client);

    await act(async () => {
      await vi.advanceTimersByTimeAsync(0);
    });
    unmount();

    await act(async () => {
      await vi.advanceTimersByTimeAsync(HEALTH_POLL_INTERVAL_MS);
    });

    expect(client.health).toHaveBeenCalledTimes(1);
  });

  it("ignores a health response that arrives after unmounting", async () => {
    const resolved = createDeferred<HealthResponse>();
    const rejected = createDeferred<HealthResponse>();
    const client = createMockApiClient();
    client.health.mockReturnValueOnce(resolved.promise);
    const first = renderStatus(client);
    first.unmount();

    client.health.mockReturnValueOnce(rejected.promise);
    const second = renderStatus(client);
    second.unmount();

    await act(async () => {
      resolved.resolve({ status: "ok", version: "1.0.0" });
      rejected.reject(new Error("too late"));
      await rejected.promise.catch(() => undefined);
    });

    expect(screen.queryByTestId("status")).not.toBeInTheDocument();
  });

  it("shows that a calculation is in flight", async () => {
    const deferred = createDeferred<CalculateResponse>();
    const client = createMockApiClient();
    client.calculate.mockReturnValueOnce(deferred.promise);
    const { user } = renderStatus(client);

    expect(screen.queryByTestId("status-calculating")).not.toBeInTheDocument();

    await user.click(screen.getByTestId("key-4"));
    await user.click(screen.getByTestId("key-square"));

    expect(screen.getByTestId("status-calculating")).toBeInTheDocument();

    await act(async () => {
      deferred.resolve({ operation: "square", operands: [4], result: 16 });
    });

    expect(screen.queryByTestId("status-calculating")).not.toBeInTheDocument();
  });
});
