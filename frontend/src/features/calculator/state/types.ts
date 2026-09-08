import type { ApiErrorCode } from "@/lib/api/client";
import type { BinaryOperationName, UnaryOperationName } from "@/lib/api/types";

/** The ten characters that may be typed into {@link CalculatorState.input}. */
export type Digit = "0" | "1" | "2" | "3" | "4" | "5" | "6" | "7" | "8" | "9";

export type CalculatorStatus = "idle" | "calculating" | "error";

export interface CalculatorError {
  code: ApiErrorCode;
  /** Server-provided, safe to display. */
  message: string;
}

export interface HistoryEntry {
  id: string;
  /** Left-hand side only, e.g. `2 + 3` or `√(9)`. */
  expression: string;
  /** Exactly the number the server returned. */
  result: number;
}

export interface CalculatorState {
  /** What the user is typing; always a valid numeric string, starts at `"0"`. */
  input: string;
  /** Left operand. Only ever a server result or `Number(input)` — never computed. */
  accumulator: number | null;
  pendingOperation: BinaryOperationName | null;
  /** When true the next digit replaces `input` instead of appending to it. */
  overwrite: boolean;
  /**
   * True once `input` holds an operand the user supplied for the pending
   * operation — a typed number, or the result of a unary key applied to one.
   * It is what tells `=` and the next operator that the chain can be settled;
   * `overwrite` cannot stand in for it, because a result sets `overwrite` too.
   */
  hasRightOperand: boolean;
  /** Secondary display line, e.g. `2 +` or `2 + 3 =`. */
  expression: string | null;
  status: CalculatorStatus;
  error: CalculatorError | null;
  /** Newest first, capped at `HISTORY_LIMIT`. */
  history: HistoryEntry[];
}

/**
 * Everything a user can press, whether by clicking a key or typing.
 *
 * `<Calculator.Key action={...} />` takes exactly this shape, which is why the
 * union is part of the feature's public surface.
 */
export type KeyAction =
  | { type: "digit"; digit: Digit }
  | { type: "decimal" }
  | { type: "backspace" }
  | { type: "clear" }
  | { type: "operator"; operation: BinaryOperationName }
  | { type: "unary"; operation: UnaryOperationName }
  | { type: "equals" };

/**
 * Reducer actions. They carry every value they need: the reducer only edits
 * strings and assigns numbers that were produced elsewhere (a server result or
 * `Number(input)`), so no transition can ever perform arithmetic.
 */
export type CalculatorAction =
  | { type: "DIGIT"; digit: Digit }
  | { type: "DECIMAL" }
  | { type: "BACKSPACE" }
  | { type: "CLEAR" }
  | {
      type: "OPERATOR_SELECTED";
      operation: BinaryOperationName;
      /** `Number(input)` at the time the operator was pressed. */
      accumulator: number;
      expression: string;
    }
  | { type: "CALCULATION_STARTED"; expression: string }
  | {
      type: "CALCULATION_SUCCEEDED";
      /** Verbatim `result` field of the server response. */
      result: number;
      entry: HistoryEntry;
      /** State of the pending chain after this result. */
      accumulator: number | null;
      pendingOperation: BinaryOperationName | null;
      expression: string | null;
      /**
       * Whether the result itself is the right operand of `pendingOperation`
       * (true after a unary key pressed mid-chain, false after `=` or after an
       * operator settled the chain and started a new one).
       */
      hasRightOperand: boolean;
    }
  | { type: "CALCULATION_FAILED"; error: CalculatorError };
