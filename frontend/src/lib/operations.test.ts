import { describe, expect, it } from "vitest";
import {
  BINARY_OPERATION_NAMES,
  OPERATION_NAMES,
  type OperationName,
  UNARY_OPERATION_NAMES,
} from "@/lib/api/types";
import {
  isBinaryOperation,
  isUnaryOperation,
  OPERATIONS,
  operationLabel,
  operationSymbol,
} from "@/lib/operations";

/** The expected registry table — the twin of `domain.Registry()` on the Go side. */
const EXPECTED: Record<OperationName, { symbol: string; arity: 1 | 2 }> = {
  add: { symbol: "+", arity: 2 },
  subtract: { symbol: "−", arity: 2 },
  multiply: { symbol: "×", arity: 2 },
  divide: { symbol: "÷", arity: 2 },
  power: { symbol: "xʸ", arity: 2 },
  modulo: { symbol: "mod", arity: 2 },
  negate: { symbol: "±", arity: 1 },
  sqrt: { symbol: "√", arity: 1 },
  square: { symbol: "x²", arity: 1 },
  percent: { symbol: "%", arity: 1 },
};

describe("OPERATIONS", () => {
  it("covers exactly the ten operations of the contract", () => {
    expect(Object.keys(OPERATIONS).toSorted()).toEqual([...OPERATION_NAMES].toSorted());
    expect(OPERATION_NAMES).toHaveLength(10);
  });

  it.each([...OPERATION_NAMES])("describes %s with the symbol and arity of the spec", (name) => {
    expect(OPERATIONS[name].symbol).toBe(EXPECTED[name].symbol);
    expect(OPERATIONS[name].arity).toBe(EXPECTED[name].arity);
  });

  it.each([...OPERATION_NAMES])("gives %s a non-empty accessible label", (name) => {
    expect(OPERATIONS[name].label.length).toBeGreaterThan(0);
    expect(operationLabel(name)).toBe(OPERATIONS[name].label);
    expect(operationSymbol(name)).toBe(OPERATIONS[name].symbol);
  });

  it("keeps kind and arity consistent", () => {
    for (const name of OPERATION_NAMES) {
      const { kind, arity } = OPERATIONS[name];
      expect(arity).toBe(kind === "binary" ? 2 : 1);
    }
  });

  it("agrees with the binary and unary name tuples", () => {
    expect(BINARY_OPERATION_NAMES.filter(isBinaryOperation)).toEqual([...BINARY_OPERATION_NAMES]);
    expect(UNARY_OPERATION_NAMES.filter(isUnaryOperation)).toEqual([...UNARY_OPERATION_NAMES]);
    expect(BINARY_OPERATION_NAMES.some(isUnaryOperation)).toBe(false);
    expect(UNARY_OPERATION_NAMES.some(isBinaryOperation)).toBe(false);
  });
});
