import {
  type CalculateRequest,
  type CalculateResponse,
  ERROR_CODES,
  type ErrorCode,
  type ErrorResponse,
  type HealthResponse,
  OPERATION_NAMES,
  type OperationInfo,
  type OperationName,
  type OperationsResponse,
} from "@/lib/api/types";

/**
 * Pseudo error code used when the transport itself failed: the request never
 * produced a parseable `ErrorResponse` (offline, DNS failure, CORS, HTML error
 * page from a proxy, ...). It is deliberately NOT part of {@link ErrorCode},
 * which mirrors the OpenAPI enum exactly.
 */
export const NETWORK_ERROR = "NETWORK_ERROR";

/** Every code the UI may have to render. */
export type ApiErrorCode = ErrorCode | typeof NETWORK_ERROR;

/** Base URL used when `VITE_API_BASE_URL` is not set: same-origin API. */
export const DEFAULT_API_BASE_URL = "/api/v1";

/** HTTP status reported for failures that never reached the server. */
const NO_HTTP_STATUS = 0;

/** Paths, relative to the base URL, as defined by `api/openapi.yaml`. */
const ENDPOINTS = {
  health: "/health",
  operations: "/operations",
  calculate: "/calculate",
} as const;

const JSON_MEDIA_TYPE = "application/json";

/** Any failed API call, whether the server answered or not. */
export class ApiError extends Error {
  readonly status: number;
  readonly code: ApiErrorCode;

  constructor(status: number, code: ApiErrorCode, message: string) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.code = code;
  }
}

export interface ApiClient {
  /** `POST /calculate` — the only place arithmetic ever happens. */
  calculate(request: CalculateRequest, signal?: AbortSignal): Promise<CalculateResponse>;
  /** `GET /operations` — canonical registry, for discovery. */
  listOperations(signal?: AbortSignal): Promise<OperationsResponse>;
  /** `GET /health` — liveness probe. */
  health(signal?: AbortSignal): Promise<HealthResponse>;
}

export interface ApiClientOptions {
  /** Where the API lives, e.g. `/api/v1` or `http://localhost:8080/api/v1`. */
  baseUrl: string;
  /** Injection seam for tests; defaults to the global `fetch`. */
  fetch?: typeof globalThis.fetch;
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null;
}

function isErrorCode(value: unknown): value is ErrorCode {
  return typeof value === "string" && (ERROR_CODES as readonly string[]).includes(value);
}

/** Narrows a parsed body to the contract's `ErrorResponse` shape. */
function isErrorResponse(value: unknown): value is ErrorResponse {
  if (!isRecord(value) || !isRecord(value.error)) {
    return false;
  }
  return isErrorCode(value.error.code) && typeof value.error.message === "string";
}

/**
 * Narrows a successful body to the schema the endpoint promises.
 *
 * A 200 is not a guarantee: a proxy, a stale deployment or a partial write can
 * all produce a well-formed JSON body that does not match the contract. Letting
 * one through would put `undefined` on the display and then send it back as an
 * operand, so every success is narrowed exactly like a failure envelope is.
 */
type ResponseGuard<T> = (value: unknown) => value is T;

function isOperationName(value: unknown): value is OperationName {
  return typeof value === "string" && (OPERATION_NAMES as readonly string[]).includes(value);
}

/** `components.schemas.Operand`: a finite IEEE-754 double. */
function isOperand(value: unknown): value is number {
  return typeof value === "number" && Number.isFinite(value);
}

function isOperandList(value: unknown): value is number[] {
  return Array.isArray(value) && value.every(isOperand);
}

function isCalculateResponse(value: unknown): value is CalculateResponse {
  return (
    isRecord(value) &&
    isOperationName(value.operation) &&
    isOperandList(value.operands) &&
    isOperand(value.result)
  );
}

function isOperationInfo(value: unknown): value is OperationInfo {
  return (
    isRecord(value) &&
    isOperationName(value.name) &&
    typeof value.symbol === "string" &&
    (value.arity === 1 || value.arity === 2) &&
    typeof value.description === "string"
  );
}

function isOperationsResponse(value: unknown): value is OperationsResponse {
  return (
    isRecord(value) && Array.isArray(value.operations) && value.operations.every(isOperationInfo)
  );
}

function isHealthResponse(value: unknown): value is HealthResponse {
  return isRecord(value) && value.status === "ok" && typeof value.version === "string";
}

function joinUrl(baseUrl: string, path: string): string {
  return `${baseUrl.replace(/\/+$/, "")}${path}`;
}

export function createApiClient({ baseUrl, fetch }: ApiClientOptions): ApiClient {
  const doFetch: typeof globalThis.fetch = fetch ?? globalThis.fetch.bind(globalThis);

  async function request<T>(
    path: string,
    init: RequestInit,
    isExpected: ResponseGuard<T>,
  ): Promise<T> {
    const url = joinUrl(baseUrl, path);

    let response: Response;
    try {
      response = await doFetch(url, init);
    } catch (cause) {
      throw new ApiError(
        NO_HTTP_STATUS,
        NETWORK_ERROR,
        cause instanceof Error ? cause.message : `Could not reach ${url}`,
      );
    }

    let body: unknown;
    try {
      body = await response.json();
    } catch {
      throw new ApiError(
        response.status,
        NETWORK_ERROR,
        `Expected JSON from ${url} but the response could not be parsed`,
      );
    }

    if (!response.ok) {
      if (isErrorResponse(body)) {
        throw new ApiError(response.status, body.error.code, body.error.message);
      }
      throw new ApiError(
        response.status,
        NETWORK_ERROR,
        `Request to ${url} failed with status ${response.status}`,
      );
    }

    if (!isExpected(body)) {
      throw new ApiError(response.status, NETWORK_ERROR, `Malformed response from ${url}`);
    }

    return body;
  }

  return {
    calculate(payload, signal) {
      return request(
        ENDPOINTS.calculate,
        {
          method: "POST",
          headers: {
            "Content-Type": JSON_MEDIA_TYPE,
            Accept: JSON_MEDIA_TYPE,
          },
          body: JSON.stringify(payload),
          signal,
        },
        isCalculateResponse,
      );
    },
    listOperations(signal) {
      return request(
        ENDPOINTS.operations,
        {
          method: "GET",
          headers: { Accept: JSON_MEDIA_TYPE },
          signal,
        },
        isOperationsResponse,
      );
    },
    health(signal) {
      return request(
        ENDPOINTS.health,
        {
          method: "GET",
          headers: { Accept: JSON_MEDIA_TYPE },
          signal,
        },
        isHealthResponse,
      );
    },
  };
}
