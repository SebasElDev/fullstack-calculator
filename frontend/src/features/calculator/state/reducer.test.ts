import { describe, expect, it } from "vitest";
import {
  calculatorReducer,
  createInitialState,
  HISTORY_LIMIT,
  INITIAL_INPUT,
  MAX_INPUT_LENGTH,
} from "@/features/calculator/state/reducer";
import type {
  CalculatorState,
  Digit,
  HistoryEntry,
} from "@/features/calculator/state/types";

function stateWith(partial: Partial<CalculatorState> = {}): CalculatorState {
  return { ...createInitialState(), ...partial };
}

function entry(id: string, expression: string, result: number): HistoryEntry {
  return { id, expression, result };
}

function typeDigits(state: CalculatorState, digits: string): CalculatorState {
  return [...digits].reduce(
    (current, digit) => calculatorReducer(current, { type: "DIGIT", digit: digit as Digit }),
    state,
  );
}

describe("calculatorReducer", () => {
  it("starts at zero with nothing pending", () => {
    expect(createInitialState()).toEqual({
      input: "0",
      accumulator: null,
      pendingOperation: null,
      overwrite: false,
      expression: null,
      status: "idle",
      error: null,
      history: [],
    });
  });

  describe("DIGIT", () => {
    it("replaces the leading zero", () => {
      expect(typeDigits(stateWith(), "7").input).toBe("7");
    });

    it("appends to what is already typed", () => {
      expect(typeDigits(stateWith(), "120").input).toBe("120");
    });

    it("replaces the value in overwrite mode and leaves overwrite", () => {
      const next = calculatorReducer(stateWith({ input: "42", overwrite: true }), {
        type: "DIGIT",
        digit: "5",
      });
      expect(next).toMatchObject({ input: "5", overwrite: false });
    });

    it("stops at the maximum input length", () => {
      const long = "1".repeat(MAX_INPUT_LENGTH);
      const next = calculatorReducer(stateWith({ input: long }), { type: "DIGIT", digit: "9" });
      expect(next.input).toBe(long);
    });

    it("clears an error and starts a fresh number, keeping history", () => {
      const history = [entry("a", "1 ÷ 0", 0)];
      const next = calculatorReducer(
        stateWith({
          input: "0",
          status: "error",
          error: { code: "DIVISION_BY_ZERO", message: "division by zero is undefined" },
          accumulator: 1,
          pendingOperation: "divide",
          expression: "1 ÷ 0 =",
          history,
        }),
        { type: "DIGIT", digit: "8" },
      );
      expect(next).toEqual(stateWith({ input: "8", history }));
    });
  });

  describe("DECIMAL", () => {
    it("appends a decimal point", () => {
      expect(calculatorReducer(stateWith({ input: "12" }), { type: "DECIMAL" }).input).toBe("12.");
    });

    it("allows at most one decimal point", () => {
      const state = stateWith({ input: "1.5" });
      expect(calculatorReducer(state, { type: "DECIMAL" })).toBe(state);
    });

    it("starts a new decimal number in overwrite mode", () => {
      const next = calculatorReducer(stateWith({ input: "42", overwrite: true }), {
        type: "DECIMAL",
      });
      expect(next).toMatchObject({ input: "0.", overwrite: false });
    });

    it("respects the maximum input length", () => {
      const long = "1".repeat(MAX_INPUT_LENGTH);
      expect(calculatorReducer(stateWith({ input: long }), { type: "DECIMAL" }).input).toBe(long);
    });

    it("clears an error", () => {
      const next = calculatorReducer(
        stateWith({ status: "error", error: { code: "INTERNAL_ERROR", message: "boom" } }),
        { type: "DECIMAL" },
      );
      expect(next).toMatchObject({ input: "0.", status: "idle", error: null });
    });
  });

  describe("BACKSPACE", () => {
    it("removes the last character", () => {
      expect(calculatorReducer(stateWith({ input: "123" }), { type: "BACKSPACE" }).input).toBe("12");
    });

    it("falls back to zero on the last character", () => {
      expect(calculatorReducer(stateWith({ input: "7" }), { type: "BACKSPACE" }).input).toBe(
        INITIAL_INPUT,
      );
    });

    it("falls back to zero rather than leaving a lone minus sign", () => {
      expect(calculatorReducer(stateWith({ input: "-4" }), { type: "BACKSPACE" }).input).toBe(
        INITIAL_INPUT,
      );
    });

    it("discards a result instead of editing it", () => {
      const next = calculatorReducer(stateWith({ input: "42", overwrite: true }), {
        type: "BACKSPACE",
      });
      expect(next).toMatchObject({ input: INITIAL_INPUT, overwrite: false });
    });

    it("clears an error", () => {
      const history = [entry("a", "√(-1)", 0)];
      const next = calculatorReducer(
        stateWith({
          input: "-1",
          status: "error",
          error: { code: "UNDEFINED_RESULT", message: "result is undefined" },
          history,
        }),
        { type: "BACKSPACE" },
      );
      expect(next).toEqual(stateWith({ history }));
    });
  });

  it("CLEAR resets everything except history", () => {
    const history = [entry("a", "2 + 3", 5)];
    const next = calculatorReducer(
      stateWith({
        input: "99",
        accumulator: 2,
        pendingOperation: "add",
        overwrite: true,
        expression: "2 +",
        status: "error",
        error: { code: "INVALID_OPERANDS", message: "nope" },
        history,
      }),
      { type: "CLEAR" },
    );
    expect(next).toEqual(stateWith({ history }));
  });

  it("OPERATOR_SELECTED stores the operand and waits for the next one", () => {
    const next = calculatorReducer(
      stateWith({ input: "2", status: "error", error: { code: "INTERNAL_ERROR", message: "x" } }),
      { type: "OPERATOR_SELECTED", operation: "add", accumulator: 2, expression: "2 +" },
    );
    expect(next).toMatchObject({
      input: "2",
      accumulator: 2,
      pendingOperation: "add",
      expression: "2 +",
      overwrite: true,
      status: "idle",
      error: null,
    });
  });

  it("CALCULATION_STARTED marks the request in flight", () => {
    const next = calculatorReducer(
      stateWith({ error: { code: "INTERNAL_ERROR", message: "old" }, status: "error" }),
      { type: "CALCULATION_STARTED", expression: "2 + 3 =" },
    );
    expect(next).toMatchObject({ status: "calculating", error: null, expression: "2 + 3 =" });
  });

  describe("CALCULATION_SUCCEEDED", () => {
    it("renders the server result verbatim and records history", () => {
      const next = calculatorReducer(stateWith({ input: "3", status: "calculating" }), {
        type: "CALCULATION_SUCCEEDED",
        result: 42,
        entry: entry("a", "2 + 3", 42),
        accumulator: null,
        pendingOperation: null,
        expression: "2 + 3 =",
      });
      expect(next).toMatchObject({
        input: "42",
        overwrite: true,
        status: "idle",
        accumulator: null,
        pendingOperation: null,
        history: [entry("a", "2 + 3", 42)],
      });
    });

    it("keeps the chain going when another operator is pending", () => {
      const next = calculatorReducer(stateWith({ status: "calculating" }), {
        type: "CALCULATION_SUCCEEDED",
        result: 5,
        entry: entry("a", "2 + 3", 5),
        accumulator: 5,
        pendingOperation: "multiply",
        expression: "5 ×",
      });
      expect(next).toMatchObject({
        input: "5",
        accumulator: 5,
        pendingOperation: "multiply",
        expression: "5 ×",
      });
    });

    it("caps history at HISTORY_LIMIT entries, newest first", () => {
      const seeded = Array.from({ length: HISTORY_LIMIT }, (_, index) =>
        entry(`old-${index}`, "1 + 1", 2),
      );
      const next = calculatorReducer(stateWith({ history: seeded }), {
        type: "CALCULATION_SUCCEEDED",
        result: 9,
        entry: entry("new", "3 × 3", 9),
        accumulator: null,
        pendingOperation: null,
        expression: "3 × 3 =",
      });
      expect(next.history).toHaveLength(HISTORY_LIMIT);
      expect(next.history[0]?.id).toBe("new");
      expect(next.history.at(-1)?.id).toBe(seeded.at(-2)?.id);
    });
  });

  it("CALCULATION_FAILED breaks the chain and shows the error", () => {
    const next = calculatorReducer(
      stateWith({ input: "0", accumulator: 1, pendingOperation: "divide", status: "calculating" }),
      {
        type: "CALCULATION_FAILED",
        error: { code: "DIVISION_BY_ZERO", message: "division by zero is undefined" },
      },
    );
    expect(next).toMatchObject({
      status: "error",
      error: { code: "DIVISION_BY_ZERO", message: "division by zero is undefined" },
      accumulator: null,
      pendingOperation: null,
      overwrite: true,
    });
  });
});
