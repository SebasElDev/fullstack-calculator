/**
 * Hand-written mirror of `api/openapi.yaml`.
 *
 * The contract is the source of truth: every type here corresponds to a schema
 * under `components.schemas`. Keep the names identical so the two can be diffed
 * by eye.
 */

/** `OperationName` members with arity 2. */
export const BINARY_OPERATION_NAMES = [
  "add",
  "subtract",
  "multiply",
  "divide",
  "power",
  "modulo",
] as const;

/** `OperationName` members with arity 1. */
export const UNARY_OPERATION_NAMES = ["negate", "sqrt", "square", "percent"] as const;

/** Every operation the API can perform, in registry order. */
export const OPERATION_NAMES = [...BINARY_OPERATION_NAMES, ...UNARY_OPERATION_NAMES] as const;

export type BinaryOperationName = (typeof BINARY_OPERATION_NAMES)[number];
export type UnaryOperationName = (typeof UNARY_OPERATION_NAMES)[number];

/** `components.schemas.OperationName` */
export type OperationName = BinaryOperationName | UnaryOperationName;

/** `components.schemas.OperationInfo.arity` */
export type OperationArity = 1 | 2;

/** `components.schemas.CalculateRequest` */
export interface CalculateRequest {
  operation: OperationName;
  /** Ordered operands; length equals the operation's arity. */
  operands: number[];
}

/** `components.schemas.CalculateResponse` */
export interface CalculateResponse {
  operation: OperationName;
  operands: number[];
  /** Finite result, normalised server-side to 15 significant digits. */
  result: number;
}

/** `components.schemas.OperationInfo` */
export interface OperationInfo {
  name: OperationName;
  symbol: string;
  arity: OperationArity;
  description: string;
}

/** `components.schemas.OperationsResponse` */
export interface OperationsResponse {
  operations: OperationInfo[];
}

/** `components.schemas.HealthResponse` */
export interface HealthResponse {
  status: "ok";
  version: string;
}

/** `components.schemas.ErrorCode` */
export const ERROR_CODES = [
  "INVALID_REQUEST",
  "UNSUPPORTED_OPERATION",
  "INVALID_OPERANDS",
  "DIVISION_BY_ZERO",
  "UNDEFINED_RESULT",
  "RESULT_OUT_OF_RANGE",
  "NOT_FOUND",
  "INTERNAL_ERROR",
] as const;

export type ErrorCode = (typeof ERROR_CODES)[number];

/** `components.schemas.ErrorResponse` */
export interface ErrorResponse {
  error: {
    code: ErrorCode;
    message: string;
  };
}
