import { type RenderOptions, type RenderResult, render } from "@testing-library/react";
import userEvent, { type UserEvent } from "@testing-library/user-event";
import type { ReactElement, ReactNode } from "react";
import { type Mock, vi } from "vitest";
import { ApiClientProvider } from "@/features/calculator";
import type { ApiClient } from "@/lib/api/client";
import { OPERATION_NAMES } from "@/lib/api/types";
import { OPERATIONS } from "@/lib/operations";

/** An {@link ApiClient} whose every method is a Vitest mock. */
export type MockApiClient = { [K in keyof ApiClient]: Mock<ApiClient[K]> };

/**
 * A client that answers plausibly by default. Tests that care about results
 * override `calculate` — often with a deliberately wrong answer, to prove the
 * UI renders what the server said rather than computing anything.
 */
export function createMockApiClient(overrides: Partial<ApiClient> = {}): MockApiClient {
  const client: MockApiClient = {
    calculate: vi.fn<ApiClient["calculate"]>(async (request) => ({ ...request, result: 0 })),
    listOperations: vi.fn<ApiClient["listOperations"]>(async () => ({
      operations: OPERATION_NAMES.map((name) => ({
        name,
        symbol: OPERATIONS[name].symbol,
        arity: OPERATIONS[name].arity,
        description: OPERATIONS[name].label,
      })),
    })),
    health: vi.fn<ApiClient["health"]>(async () => ({ status: "ok", version: "test" })),
  };

  if (overrides.calculate !== undefined) {
    client.calculate.mockImplementation(overrides.calculate);
  }
  if (overrides.listOperations !== undefined) {
    client.listOperations.mockImplementation(overrides.listOperations);
  }
  if (overrides.health !== undefined) {
    client.health.mockImplementation(overrides.health);
  }

  return client;
}

export interface RenderWithProvidersOptions extends Omit<RenderOptions, "wrapper"> {
  client?: MockApiClient;
}

export interface RenderWithProvidersResult extends RenderResult {
  client: MockApiClient;
  user: UserEvent;
}

/** Renders `ui` inside `<ApiClientProvider>` with a mock client. */
export function renderWithProviders(
  ui: ReactElement,
  options: RenderWithProvidersOptions = {},
): RenderWithProvidersResult {
  const { client = createMockApiClient(), ...renderOptions } = options;
  const user = userEvent.setup();

  function Wrapper({ children }: { children: ReactNode }) {
    return <ApiClientProvider client={client}>{children}</ApiClientProvider>;
  }

  return { client, user, ...render(ui, { wrapper: Wrapper, ...renderOptions }) };
}

/** A promise plus the handles to settle it later — for testing pending states. */
export interface Deferred<T> {
  promise: Promise<T>;
  resolve: (value: T) => void;
  reject: (reason: unknown) => void;
}

export function createDeferred<T>(): Deferred<T> {
  let resolve!: (value: T) => void;
  let reject!: (reason: unknown) => void;
  const promise = new Promise<T>((resolvePromise, rejectPromise) => {
    resolve = resolvePromise;
    reject = rejectPromise;
  });
  return { promise, resolve, reject };
}
