import { act, screen, waitFor } from "@testing-library/react";
import type { UserEvent } from "@testing-library/user-event";
import { describe, expect, it } from "vitest";
import { Calculator } from "@/features/calculator";
import { ApiError } from "@/lib/api/client";
import type { CalculateResponse } from "@/lib/api/types";
import {
  createDeferred,
  createMockApiClient,
  type MockApiClient,
  renderWithProviders,
} from "@/test/utils";

function renderCalculator(client: MockApiClient) {
  return renderWithProviders(
    <Calculator>
      <Calculator.Display />
      <Calculator.Keypad />
      <Calculator.History />
    </Calculator>,
    { client },
  );
}

/** Clicks keypad keys in order, e.g. `press(user, "2", "add", "3", "equals")`. */
async function press(user: UserEvent, ...keys: string[]): Promise<void> {
  for (const key of keys) {
    await user.click(screen.getByTestId(`key-${key}`));
  }
}

function displayValue(): string {
  return screen.getByTestId("display-value").textContent ?? "";
}

function historyLines(): string[] {
  return screen.getAllByTestId("history-entry").map((entry) => entry.textContent ?? "");
}

describe("CalculatorProvider", () => {
  it("displays what the server returned, even when the answer is wrong", async () => {
    const client = createMockApiClient();
    client.calculate.mockResolvedValueOnce({ operation: "add", operands: [2, 3], result: 42 });
    const { user } = renderCalculator(client);

    await press(user, "2", "add", "3", "equals");

    expect(client.calculate).toHaveBeenCalledWith({ operation: "add", operands: [2, 3] });
    await waitFor(() => {
      expect(displayValue()).toBe("42");
    });
    expect(historyLines()).toEqual(["2 + 3 = 42"]);
    expect(screen.getByTestId("display-expression")).toHaveTextContent("2 + 3 =");
  });

  it("sends one request per binary operation when operators are chained", async () => {
    const client = createMockApiClient();
    client.calculate
      .mockResolvedValueOnce({ operation: "add", operands: [2, 3], result: 10 })
      .mockResolvedValueOnce({ operation: "multiply", operands: [10, 4], result: 99 });
    const { user } = renderCalculator(client);

    await press(user, "2", "add", "3", "multiply");
    await waitFor(() => {
      expect(displayValue()).toBe("10");
    });
    await press(user, "4", "equals");
    await waitFor(() => {
      expect(displayValue()).toBe("99");
    });

    expect(client.calculate.mock.calls).toEqual([
      [{ operation: "add", operands: [2, 3] }],
      [{ operation: "multiply", operands: [10, 4] }],
    ]);
    expect(historyLines()).toEqual(["10 × 4 = 99", "2 + 3 = 10"]);
  });

  it("feeds a result back into the next operation as the left operand", async () => {
    const client = createMockApiClient();
    client.calculate
      .mockResolvedValueOnce({ operation: "add", operands: [2, 3], result: 5 })
      .mockResolvedValueOnce({ operation: "add", operands: [5, 1], result: 6 });
    const { user } = renderCalculator(client);

    await press(user, "2", "add", "3", "equals");
    await waitFor(() => {
      expect(displayValue()).toBe("5");
    });
    await press(user, "add", "1", "equals");
    await waitFor(() => {
      expect(displayValue()).toBe("6");
    });

    expect(client.calculate).toHaveBeenLastCalledWith({ operation: "add", operands: [5, 1] });
  });

  it("sends a single operand for unary operations", async () => {
    const client = createMockApiClient();
    client.calculate.mockResolvedValueOnce({ operation: "sqrt", operands: [9], result: 3 });
    const { user } = renderCalculator(client);

    await press(user, "9", "sqrt");

    expect(client.calculate).toHaveBeenCalledWith({ operation: "sqrt", operands: [9] });
    await waitFor(() => {
      expect(displayValue()).toBe("3");
    });
    expect(historyLines()).toEqual(["√(9) = 3"]);
    expect(screen.getByTestId("display-expression")).toHaveTextContent("√(9) =");
  });

  it("applies a unary operation to the right operand without breaking the chain", async () => {
    const client = createMockApiClient();
    client.calculate
      .mockResolvedValueOnce({ operation: "sqrt", operands: [9], result: 3 })
      .mockResolvedValueOnce({ operation: "add", operands: [2, 3], result: 5 });
    const { user } = renderCalculator(client);

    await press(user, "2", "add", "9", "sqrt");
    await waitFor(() => {
      expect(displayValue()).toBe("3");
    });
    expect(screen.getByTestId("display-expression")).toHaveTextContent("2 +");

    await press(user, "equals");
    await waitFor(() => {
      expect(displayValue()).toBe("5");
    });
    expect(client.calculate).toHaveBeenLastCalledWith({ operation: "add", operands: [2, 3] });
  });

  it("settles the pending operation when an operator follows a unary result", async () => {
    const client = createMockApiClient();
    client.calculate
      .mockResolvedValueOnce({ operation: "sqrt", operands: [9], result: 3 })
      .mockResolvedValueOnce({ operation: "add", operands: [2, 3], result: 5 });
    const { user } = renderCalculator(client);

    await press(user, "2", "add", "9", "sqrt");
    await waitFor(() => {
      expect(displayValue()).toBe("3");
    });

    await press(user, "multiply");
    await waitFor(() => {
      expect(displayValue()).toBe("5");
    });

    expect(client.calculate.mock.calls).toEqual([
      [{ operation: "sqrt", operands: [9] }],
      [{ operation: "add", operands: [2, 3] }],
    ]);
    expect(historyLines()).toEqual(["2 + 3 = 5", "\u221a(9) = 3"]);
    expect(screen.getByTestId("display-expression")).toHaveTextContent("5 \u00d7");
  });

  it("replaces the pending operator when two are pressed in a row", async () => {
    const client = createMockApiClient();
    const { user } = renderCalculator(client);

    await press(user, "2", "add", "multiply");

    expect(client.calculate).not.toHaveBeenCalled();
    expect(screen.getByTestId("display-expression")).toHaveTextContent("2 ×");
    expect(screen.getByTestId("key-multiply")).toHaveAttribute("aria-pressed", "true");
    expect(screen.getByTestId("key-add")).toHaveAttribute("aria-pressed", "false");
  });

  it("ignores equals until an operation is pending", async () => {
    const client = createMockApiClient();
    const { user } = renderCalculator(client);

    await press(user, "5", "equals");

    expect(client.calculate).not.toHaveBeenCalled();
    expect(displayValue()).toBe("5");
  });

  it("shows the server's error message and recovers on the next digit", async () => {
    const client = createMockApiClient();
    client.calculate.mockRejectedValueOnce(
      new ApiError(422, "DIVISION_BY_ZERO", "division by zero is undefined"),
    );
    const { user } = renderCalculator(client);

    await press(user, "1", "divide", "0", "equals");

    await waitFor(() => {
      expect(displayValue()).toBe("division by zero is undefined");
    });
    expect(screen.getByTestId("display")).toHaveAttribute("data-status", "error");

    await press(user, "7");
    expect(displayValue()).toBe("7");
    expect(screen.getByTestId("display")).toHaveAttribute("data-status", "idle");
  });

  it("ignores operators while an error is on screen", async () => {
    const client = createMockApiClient();
    client.calculate.mockRejectedValueOnce(new ApiError(422, "UNDEFINED_RESULT", "no answer"));
    const { user } = renderCalculator(client);

    await press(user, "9", "sqrt");
    await waitFor(() => {
      expect(displayValue()).toBe("no answer");
    });
    client.calculate.mockClear();

    await press(user, "add", "square", "equals");

    expect(client.calculate).not.toHaveBeenCalled();
    expect(displayValue()).toBe("no answer");
  });

  it("reports a transport failure that carries no server message", async () => {
    const client = createMockApiClient();
    client.calculate.mockRejectedValueOnce("connection reset");
    const { user } = renderCalculator(client);

    await press(user, "4", "square");

    await waitFor(() => {
      expect(displayValue()).toBe("The calculation could not be performed");
    });
  });

  it("reports the message of an unexpected Error", async () => {
    const client = createMockApiClient();
    client.calculate.mockRejectedValueOnce(new Error("client exploded"));
    const { user } = renderCalculator(client);

    await press(user, "4", "square");

    await waitFor(() => {
      expect(displayValue()).toBe("client exploded");
    });
  });

  it("ignores clicks while a calculation is in flight without moving focus", async () => {
    const deferred = createDeferred<CalculateResponse>();
    const client = createMockApiClient();
    client.calculate.mockReturnValueOnce(deferred.promise);
    const { user } = renderCalculator(client);

    await press(user, "2", "add", "3", "equals");

    const digitKey = screen.getByTestId("key-5");
    expect(digitKey).toHaveAttribute("aria-disabled", "true");
    expect(screen.getByTestId("keypad")).toHaveAttribute("aria-busy", "true");
    // Not the native attribute: a disabled button leaves the accessibility tree
    // and drops the focused key to <body> on every calculation.
    expect(digitKey).not.toBeDisabled();

    await user.click(digitKey);
    expect(displayValue()).toBe("3");

    await act(async () => {
      deferred.resolve({ operation: "add", operands: [2, 3], result: 5 });
    });

    await waitFor(() => {
      expect(displayValue()).toBe("5");
    });
    expect(screen.getByTestId("key-5")).toHaveAttribute("aria-disabled", "false");
    expect(client.calculate).toHaveBeenCalledTimes(1);
  });

  it("keeps up with keys pressed faster than React re-renders", async () => {
    const client = createMockApiClient();
    client.calculate.mockResolvedValueOnce({ operation: "add", operands: [2, 3], result: 5 });
    renderCalculator(client);

    // All four presses land in a single tick, before any re-render.
    await act(async () => {
      for (const key of ["2", "add", "3", "equals"]) {
        screen.getByTestId(`key-${key}`).click();
      }
    });

    expect(client.calculate).toHaveBeenCalledWith({ operation: "add", operands: [2, 3] });
    await waitFor(() => {
      expect(displayValue()).toBe("5");
    });
  });

  it("ignores keystrokes while a calculation is in flight", async () => {
    const deferred = createDeferred<CalculateResponse>();
    const client = createMockApiClient();
    client.calculate.mockReturnValueOnce(deferred.promise);
    const { user } = renderCalculator(client);

    await press(user, "2", "add", "3", "equals");
    await user.keyboard("7{Escape}");

    expect(displayValue()).toBe("3");
    expect(client.calculate).toHaveBeenCalledTimes(1);

    await act(async () => {
      deferred.resolve({ operation: "add", operands: [2, 3], result: 5 });
    });
    await waitFor(() => {
      expect(displayValue()).toBe("5");
    });
  });

  it("clears everything but the history on AC", async () => {
    const client = createMockApiClient();
    client.calculate.mockResolvedValueOnce({ operation: "add", operands: [2, 3], result: 5 });
    const { user } = renderCalculator(client);

    await press(user, "2", "add", "3", "equals");
    await waitFor(() => {
      expect(displayValue()).toBe("5");
    });

    await press(user, "clear");

    expect(displayValue()).toBe("0");
    expect(screen.getByTestId("display-expression").textContent).toBe("\u00a0");
    expect(historyLines()).toEqual(["2 + 3 = 5"]);
  });

  it("edits the typed number without contacting the server", async () => {
    const client = createMockApiClient();
    const { user } = renderCalculator(client);

    await press(user, "1", "2", "decimal", "5", "backspace");

    expect(displayValue()).toBe("12.");
    expect(client.calculate).not.toHaveBeenCalled();
  });
});
