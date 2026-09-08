import type { CalculatorAction, CalculatorState } from "@/features/calculator/state/types";

/** How many results the session keeps. */
export const HISTORY_LIMIT = 20;

/** Guard against unbounded input strings (which would parse to `Infinity`). */
export const MAX_INPUT_LENGTH = 16;

/** The value shown before anything is typed, and after `AC`. */
export const INITIAL_INPUT = "0";

const DECIMAL_POINT = ".";

export function createInitialState(): CalculatorState {
  return {
    input: INITIAL_INPUT,
    accumulator: null,
    pendingOperation: null,
    overwrite: false,
    hasRightOperand: false,
    expression: null,
    status: "idle",
    error: null,
    history: [],
  };
}

/** Everything `AC` resets — history survives. */
function clearedState(state: CalculatorState): CalculatorState {
  return { ...createInitialState(), history: state.history };
}

/**
 * Pure transitions. String editing and assignment only: no `+`, `-`, `*`, `/`,
 * rounding or formatting of numbers happens here or anywhere else in the SPA.
 */
export function calculatorReducer(
  state: CalculatorState,
  action: CalculatorAction,
): CalculatorState {
  switch (action.type) {
    case "DIGIT": {
      if (state.status === "error") {
        return { ...clearedState(state), input: action.digit, hasRightOperand: true };
      }
      if (state.overwrite) {
        return { ...state, input: action.digit, overwrite: false, hasRightOperand: true };
      }
      if (state.input.length >= MAX_INPUT_LENGTH) {
        return state;
      }
      const input = state.input === INITIAL_INPUT ? action.digit : `${state.input}${action.digit}`;
      return { ...state, input, hasRightOperand: true };
    }

    case "DECIMAL": {
      if (state.status === "error") {
        return {
          ...clearedState(state),
          input: `${INITIAL_INPUT}${DECIMAL_POINT}`,
          hasRightOperand: true,
        };
      }
      if (state.overwrite) {
        return {
          ...state,
          input: `${INITIAL_INPUT}${DECIMAL_POINT}`,
          overwrite: false,
          hasRightOperand: true,
        };
      }
      if (state.input.includes(DECIMAL_POINT) || state.input.length >= MAX_INPUT_LENGTH) {
        return state;
      }
      return { ...state, input: `${state.input}${DECIMAL_POINT}`, hasRightOperand: true };
    }

    case "BACKSPACE": {
      if (state.status === "error") {
        return clearedState(state);
      }
      if (state.overwrite) {
        return { ...state, input: INITIAL_INPUT, overwrite: false, hasRightOperand: true };
      }
      const trimmed = state.input.slice(0, -1);
      const input = trimmed === "" || trimmed === "-" ? INITIAL_INPUT : trimmed;
      return { ...state, input, hasRightOperand: true };
    }

    case "CLEAR":
      return clearedState(state);

    case "OPERATOR_SELECTED":
      return {
        ...state,
        accumulator: action.accumulator,
        pendingOperation: action.operation,
        expression: action.expression,
        overwrite: true,
        hasRightOperand: false,
        status: "idle",
        error: null,
      };

    case "CALCULATION_STARTED":
      return { ...state, status: "calculating", error: null, expression: action.expression };

    case "CALCULATION_SUCCEEDED":
      return {
        ...state,
        input: String(action.result),
        accumulator: action.accumulator,
        pendingOperation: action.pendingOperation,
        expression: action.expression,
        overwrite: true,
        hasRightOperand: action.hasRightOperand,
        status: "idle",
        error: null,
        history: [action.entry, ...state.history].slice(0, HISTORY_LIMIT),
      };

    case "CALCULATION_FAILED":
      return {
        ...state,
        accumulator: null,
        pendingOperation: null,
        overwrite: true,
        hasRightOperand: false,
        status: "error",
        error: action.error,
      };
  }
}
