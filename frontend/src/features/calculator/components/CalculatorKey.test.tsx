import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { Calculator, describeKey, type KeyAction } from "@/features/calculator";
import { createMockApiClient, renderWithProviders } from "@/test/utils";

function renderKeys(...actions: KeyAction[]) {
  return renderWithProviders(
    <Calculator>
      <Calculator.Display />
      <Calculator.Keypad>
        {actions.map((action) => (
          <Calculator.Key key={describeKey(action).name} action={action} />
        ))}
      </Calculator.Keypad>
    </Calculator>,
    { client: createMockApiClient() },
  );
}

describe("describeKey", () => {
  const cases: [KeyAction, { name: string; label: string; ariaLabel: string }][] = [
    [
      { type: "digit", digit: "7" },
      { name: "7", label: "7", ariaLabel: "7" },
    ],
    [{ type: "decimal" }, { name: "decimal", label: ".", ariaLabel: "decimal point" }],
    [{ type: "backspace" }, { name: "backspace", label: "⌫", ariaLabel: "backspace" }],
    [{ type: "clear" }, { name: "clear", label: "AC", ariaLabel: "clear all" }],
    [{ type: "equals" }, { name: "equals", label: "=", ariaLabel: "equals" }],
    [
      { type: "operator", operation: "divide" },
      { name: "divide", label: "÷", ariaLabel: "divide" },
    ],
    [
      { type: "unary", operation: "sqrt" },
      { name: "sqrt", label: "√", ariaLabel: "square root" },
    ],
  ];

  it.each(cases)("describes %o", (action, expected) => {
    expect(describeKey(action)).toEqual(expected);
  });
});

describe("<Calculator.Key>", () => {
  it("renders the derived glyph, accessible name and test id", () => {
    renderKeys({ type: "operator", operation: "divide" }, { type: "unary", operation: "sqrt" });

    const divide = screen.getByTestId("key-divide");
    expect(divide).toHaveTextContent("÷");
    expect(divide).toHaveAccessibleName("divide");
    expect(divide).toHaveAttribute("type", "button");
    expect(screen.getByTestId("key-sqrt")).toHaveAccessibleName("square root");
  });

  it("accepts custom content and classes", () => {
    renderWithProviders(
      <Calculator>
        <Calculator.Key action={{ type: "clear" }} className="wide">
          reset
        </Calculator.Key>
      </Calculator>,
    );

    const key = screen.getByTestId("key-clear");
    expect(key).toHaveTextContent("reset");
    expect(key).toHaveClass("wide");
  });

  it("presses the action it was given", async () => {
    const { user } = renderKeys({ type: "digit", digit: "7" });

    await user.click(screen.getByTestId("key-7"));

    expect(screen.getByTestId("display-value")).toHaveTextContent("7");
  });

  it("marks only binary operators as pressable, and highlights the pending one", async () => {
    const { user } = renderKeys(
      { type: "digit", digit: "7" },
      { type: "operator", operation: "add" },
      { type: "unary", operation: "sqrt" },
    );

    expect(screen.getByTestId("key-add")).toHaveAttribute("aria-pressed", "false");
    expect(screen.getByTestId("key-sqrt")).not.toHaveAttribute("aria-pressed");

    await user.click(screen.getByTestId("key-7"));
    await user.click(screen.getByTestId("key-add"));

    expect(screen.getByTestId("key-add")).toHaveAttribute("aria-pressed", "true");
  });
});
