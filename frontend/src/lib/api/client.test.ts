import { afterEach, describe, expect, it, vi } from "vitest";
import { ApiError, createApiClient, DEFAULT_API_BASE_URL, NETWORK_ERROR } from "@/lib/api/client";
import type { CalculateResponse, ErrorResponse } from "@/lib/api/types";

const BASE_URL = "/api/v1";

/** Minimal `Response` stand-in: the client only uses `ok`, `status` and `json()`. */
function stubResponse(
  body: unknown,
  { ok = true, status = 200 }: { ok?: boolean; status?: number } = {},
): Response {
  return { ok, status, json: async () => body } as unknown as Response;
}

function unparseableResponse(status: number): Response {
  return {
    ok: status < 400,
    status,
    json: async () => {
      throw new SyntaxError("Unexpected token < in JSON at position 0");
    },
  } as unknown as Response;
}

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("createApiClient", () => {
  it("exposes the same-origin base URL as the default", () => {
    expect(DEFAULT_API_BASE_URL).toBe("/api/v1");
  });

  it("POSTs the exact CalculateRequest payload to /calculate", async () => {
    const body: CalculateResponse = { operation: "add", operands: [2, 3], result: 5 };
    const fetchMock = vi.fn(async () => stubResponse(body));
    const client = createApiClient({ baseUrl: BASE_URL, fetch: fetchMock });

    const result = await client.calculate({ operation: "add", operands: [2, 3] });

    expect(result).toEqual(body);
    expect(fetchMock).toHaveBeenCalledTimes(1);
    const [url, init] = fetchMock.mock.calls[0] as [string, RequestInit];
    expect(url).toBe("/api/v1/calculate");
    expect(init.method).toBe("POST");
    expect(init.headers).toEqual({
      "Content-Type": "application/json",
      Accept: "application/json",
    });
    expect(init.body).toBe('{"operation":"add","operands":[2,3]}');
  });

  it("forwards an abort signal", async () => {
    const fetchMock = vi.fn(async () => stubResponse({ status: "ok", version: "1.0.0" }));
    const client = createApiClient({ baseUrl: BASE_URL, fetch: fetchMock });
    const controller = new AbortController();

    await client.health(controller.signal);

    const [, init] = fetchMock.mock.calls[0] as [string, RequestInit];
    expect(init.signal).toBe(controller.signal);
  });

  it("GETs /operations", async () => {
    const body = { operations: [{ name: "add", symbol: "+", arity: 2, description: "Sum" }] };
    const fetchMock = vi.fn(async () => stubResponse(body));
    const client = createApiClient({ baseUrl: BASE_URL, fetch: fetchMock });

    await expect(client.listOperations()).resolves.toEqual(body);
    const [url, init] = fetchMock.mock.calls[0] as [string, RequestInit];
    expect(url).toBe("/api/v1/operations");
    expect(init.method).toBe("GET");
    expect(init.headers).toEqual({ Accept: "application/json" });
  });

  it("GETs /health", async () => {
    const body = { status: "ok", version: "1.0.0" };
    const fetchMock = vi.fn(async () => stubResponse(body));
    const client = createApiClient({ baseUrl: "http://localhost:8080/api/v1", fetch: fetchMock });

    await expect(client.health()).resolves.toEqual(body);
    expect(fetchMock.mock.calls[0]?.[0]).toBe("http://localhost:8080/api/v1/health");
  });

  it("normalises trailing slashes in the base URL", async () => {
    const fetchMock = vi.fn(async () => stubResponse({ status: "ok", version: "dev" }));
    const client = createApiClient({ baseUrl: "/api/v1//", fetch: fetchMock });

    await client.health();

    expect(fetchMock.mock.calls[0]?.[0]).toBe("/api/v1/health");
  });

  it("falls back to the global fetch", async () => {
    const fetchMock = vi.fn(async () => stubResponse({ status: "ok", version: "dev" }));
    vi.stubGlobal("fetch", fetchMock);
    const client = createApiClient({ baseUrl: BASE_URL });

    await expect(client.health()).resolves.toEqual({ status: "ok", version: "dev" });
    expect(fetchMock).toHaveBeenCalledTimes(1);
  });

  it("turns an ErrorResponse envelope into an ApiError", async () => {
    const envelope: ErrorResponse = {
      error: { code: "DIVISION_BY_ZERO", message: "division by zero is undefined" },
    };
    const fetchMock = vi.fn(async () => stubResponse(envelope, { ok: false, status: 422 }));
    const client = createApiClient({ baseUrl: BASE_URL, fetch: fetchMock });

    const error = await client
      .calculate({ operation: "divide", operands: [1, 0] })
      .catch((cause: unknown) => cause);

    expect(error).toBeInstanceOf(ApiError);
    expect(error).toMatchObject({
      status: 422,
      code: "DIVISION_BY_ZERO",
      message: "division by zero is undefined",
      name: "ApiError",
    });
  });

  it("reports a failure body that does not match the contract as a network error", async () => {
    const fetchMock = vi.fn(async () =>
      stubResponse({ error: { code: "NOPE" } }, { ok: false, status: 502 }),
    );
    const client = createApiClient({ baseUrl: BASE_URL, fetch: fetchMock });

    const error = await client.health().catch((cause: unknown) => cause);

    expect(error).toBeInstanceOf(ApiError);
    expect(error).toMatchObject({ status: 502, code: NETWORK_ERROR });
    expect((error as ApiError).message).toContain("502");
  });

  it("reports a non-JSON response as a network error", async () => {
    const fetchMock = vi.fn(async () => unparseableResponse(200));
    const client = createApiClient({ baseUrl: BASE_URL, fetch: fetchMock });

    const error = await client.health().catch((cause: unknown) => cause);

    expect(error).toMatchObject({ status: 200, code: NETWORK_ERROR });
    expect((error as ApiError).message).toContain("/api/v1/health");
  });

  it("reports a failed fetch as a network error with no HTTP status", async () => {
    const fetchMock = vi.fn(async () => {
      throw new TypeError("Failed to fetch");
    });
    const client = createApiClient({ baseUrl: BASE_URL, fetch: fetchMock });

    const error = await client.health().catch((cause: unknown) => cause);

    expect(error).toMatchObject({ status: 0, code: NETWORK_ERROR, message: "Failed to fetch" });
  });

  it("describes the target when the rejection is not an Error", async () => {
    const fetchMock = vi.fn(async () => {
      throw "boom";
    });
    const client = createApiClient({ baseUrl: BASE_URL, fetch: fetchMock });

    const error = await client.health().catch((cause: unknown) => cause);

    expect((error as ApiError).message).toBe("Could not reach /api/v1/health");
  });
});
