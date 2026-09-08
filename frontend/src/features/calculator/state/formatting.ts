import type { BinaryOperationName, UnaryOperationName } from "@/lib/api/types";
import { OPERATIONS } from "@/lib/operations";

/**
 * String work only. `String(value)` is *rendering*, not arithmetic: the digits
 * shown are exactly the ones the server sent (no rounding, no locale grouping,
 * no `toFixed`).
 */
export function formatOperand(value: number): string {
  return String(value);
}

/** `2 +` — a binary operation waiting for its right operand. */
export function pendingExpression(left: number, operation: BinaryOperationName): string {
  return `${formatOperand(left)} ${OPERATIONS[operation].symbol}`;
}

/** `2 + 3` — both operands of a binary operation. */
export function binaryExpression(
  left: number,
  operation: BinaryOperationName,
  right: number,
): string {
  return `${pendingExpression(left, operation)} ${formatOperand(right)}`;
}

/**
 * How each unary operation reads as an expression. `square` cannot reuse its
 * keypad glyph (`x²`) as a suffix, so it carries its own exponent character.
 */
const UNARY_EXPRESSION: Record<UnaryOperationName, (operand: string, symbol: string) => string> = {
  negate: (operand, symbol) => `${symbol}(${operand})`,
  sqrt: (operand, symbol) => `${symbol}(${operand})`,
  square: (operand) => `(${operand})²`,
  percent: (operand, symbol) => `(${operand})${symbol}`,
};

/** `√(9)` — a unary operation applied to its operand. */
export function unaryExpression(operation: UnaryOperationName, operand: number): string {
  return UNARY_EXPRESSION[operation](formatOperand(operand), OPERATIONS[operation].symbol);
}

/** `2 + 3 =` — an expression whose result has been requested. */
export function completedExpression(expression: string): string {
  return `${expression} =`;
}

/** `2 + 3 = 5` — a history line. */
export function historyLine(expression: string, result: number): string {
  return `${completedExpression(expression)} ${formatOperand(result)}`;
}
