import { describe, expect, it } from "vitest";
import {
  binaryExpression,
  completedExpression,
  formatOperand,
  historyLine,
  pendingExpression,
  unaryExpression,
} from "@/features/calculator/state/formatting";
import { UNARY_OPERATION_NAMES } from "@/lib/api/types";

describe("formatting", () => {
  it("renders an operand exactly as the server sent it", () => {
    expect(formatOperand(0.30000000000000004)).toBe("0.30000000000000004");
    expect(formatOperand(-0.5)).toBe("-0.5");
    expect(formatOperand(1e21)).toBe("1e+21");
  });

  it("builds a pending binary expression", () => {
    expect(pendingExpression(2, "add")).toBe("2 +");
    expect(pendingExpression(7, "modulo")).toBe("7 mod");
  });

  it("builds a complete binary expression", () => {
    expect(binaryExpression(2, "add", 3)).toBe("2 + 3");
    expect(binaryExpression(2, "power", 8)).toBe("2 xʸ 8");
    expect(binaryExpression(10, "divide", 4)).toBe("10 ÷ 4");
  });

  it("builds unary expressions", () => {
    expect(unaryExpression("sqrt", 9)).toBe("√(9)");
    expect(unaryExpression("negate", 5)).toBe("±(5)");
    expect(unaryExpression("square", 4)).toBe("(4)²");
    expect(unaryExpression("percent", 50)).toBe("(50)%");
  });

  it("has a form for every unary operation", () => {
    for (const operation of UNARY_OPERATION_NAMES) {
      expect(unaryExpression(operation, 2)).toContain("2");
    }
  });

  it("marks an expression as evaluated", () => {
    expect(completedExpression("2 + 3")).toBe("2 + 3 =");
  });

  it("builds a history line", () => {
    expect(historyLine("2 + 3", 5)).toBe("2 + 3 = 5");
    expect(historyLine("√(9)", 3)).toBe("√(9) = 3");
  });
});
