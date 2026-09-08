import type {
  BinaryOperationName,
  OperationArity,
  OperationName,
  UnaryOperationName,
} from "@/lib/api/types";

/**
 * Presentation metadata for one operation.
 *
 * The symbols mirror `domain.Registry()` on the Go side; `GET /api/v1/operations`
 * returns the same table at runtime.
 */
export interface OperationMeta {
  /** Glyph shown on the keypad and in history expressions. */
  symbol: string;
  /** Accessible name, e.g. for `aria-label`. */
  label: string;
  arity: OperationArity;
  kind: "binary" | "unary";
}

/** Single source of truth for keypad glyphs, labels and history formatting. */
export const OPERATIONS: Record<OperationName, OperationMeta> = {
  add: { symbol: "+", label: "add", arity: 2, kind: "binary" },
  subtract: { symbol: "−", label: "subtract", arity: 2, kind: "binary" },
  multiply: { symbol: "×", label: "multiply", arity: 2, kind: "binary" },
  divide: { symbol: "÷", label: "divide", arity: 2, kind: "binary" },
  power: { symbol: "xʸ", label: "power", arity: 2, kind: "binary" },
  modulo: { symbol: "mod", label: "modulo", arity: 2, kind: "binary" },
  negate: { symbol: "±", label: "negate", arity: 1, kind: "unary" },
  sqrt: { symbol: "√", label: "square root", arity: 1, kind: "unary" },
  square: { symbol: "x²", label: "square", arity: 1, kind: "unary" },
  percent: { symbol: "%", label: "percent", arity: 1, kind: "unary" },
};

export function isBinaryOperation(name: OperationName): name is BinaryOperationName {
  return OPERATIONS[name].kind === "binary";
}

export function isUnaryOperation(name: OperationName): name is UnaryOperationName {
  return OPERATIONS[name].kind === "unary";
}

/** Glyph for an operation, e.g. `÷`. */
export function operationSymbol(name: OperationName): string {
  return OPERATIONS[name].symbol;
}

/** Accessible name for an operation, e.g. `square root`. */
export function operationLabel(name: OperationName): string {
  return OPERATIONS[name].label;
}
