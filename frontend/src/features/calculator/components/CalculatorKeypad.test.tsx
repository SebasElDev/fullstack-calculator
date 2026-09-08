import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { Calculator, describeKey, KEYPAD_LAYOUT } from "@/features/calculator";
import { DIGITS } from "@/features/calculator/state/keyboard";
import { OPERATION_NAMES } from "@/lib/api/types";
import { renderWithProviders } from "@/test/utils";

function renderKeypad() {
  return renderWithProviders(
    <Calculator>
      <Calculator.Keypad />
    </Calculator>,
  );
}

describe("KEYPAD_LAYOUT", () => {
  it("lays the keys out in rows of four", () => {
    for (const row of KEYPAD_LAYOUT) {
      expect(row).toHaveLength(4);
    }
  });

  it("reaches every operation, digit, decimal point, clear, backspace and equals", () => {
    const names = KEYPAD_LAYOUT.flat().map((action) => describeKey(action).name);

    expect(names).toEqual([...new Set(names)]);
    for (const expected of [
      ...OPERATION_NAMES,
      ...DIGITS,
      "decimal",
      "clear",
      "backspace",
      "equals",
    ]) {
      expect(names).toContain(expected);
    }
  });
});

describe("<Calculator.Keypad>", () => {
  it("renders the default layout", () => {
    renderKeypad();

    expect(screen.getAllByRole("button")).toHaveLength(KEYPAD_LAYOUT.flat().length);
    expect(screen.getByTestId("keypad")).toHaveAccessibleName("Keypad");
    expect(screen.getByTestId("keypad")).toHaveAttribute("aria-busy", "false");
  });

  it("renders a custom layout instead when given children", () => {
    renderWithProviders(
      <Calculator>
        <Calculator.Keypad className="two-up">
          <Calculator.Key action={{ type: "digit", digit: "7" }} />
          <Calculator.Key action={{ type: "operator", operation: "add" }} />
        </Calculator.Keypad>
      </Calculator>,
    );

    expect(screen.getAllByRole("button")).toHaveLength(2);
    expect(screen.getByTestId("keypad")).toHaveClass("two-up");
    expect(screen.queryByTestId("key-equals")).not.toBeInTheDocument();
  });
});
