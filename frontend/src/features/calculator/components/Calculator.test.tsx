import { render, screen, waitFor } from "@testing-library/react";
import type { ReactElement } from "react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { Calculator } from "@/features/calculator";
import { CalculatorDisplay } from "@/features/calculator/components/CalculatorDisplay";
import { CalculatorHistory } from "@/features/calculator/components/CalculatorHistory";
import { CalculatorKey } from "@/features/calculator/components/CalculatorKey";
import { CalculatorKeypad } from "@/features/calculator/components/CalculatorKeypad";
import { CalculatorStatus } from "@/features/calculator/components/CalculatorStatus";
import { ApiClientProvider } from "@/features/calculator/context/ApiClientContext";
import { createMockApiClient, type MockApiClient, renderWithProviders } from "@/test/utils";

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

/** React logs the error it re-throws; keep the test output readable. */
function silenceReactErrorLogging() {
  return vi.spyOn(console, "error").mockImplementation(() => {});
}

afterEach(() => {
  vi.restoreAllMocks();
});

describe("<Calculator>", () => {
  it("renders the card shell around its parts", () => {
    renderCalculator(createMockApiClient());

    const card = screen.getByTestId("calculator");
    expect(card).toHaveAccessibleName("Calculator");
    expect(card).toContainElement(screen.getByTestId("display"));
    expect(card).toContainElement(screen.getByTestId("keypad"));
    expect(card).toContainElement(screen.getByTestId("history"));
  });

  it("accepts a custom class name", () => {
    renderWithProviders(
      <Calculator className="custom-shell">
        <Calculator.Display />
      </Calculator>,
    );

    expect(screen.getByTestId("calculator")).toHaveClass("custom-shell");
  });

  it("exposes its parts as static properties", () => {
    expect(Calculator.Display).toBe(CalculatorDisplay);
    expect(Calculator.Keypad).toBe(CalculatorKeypad);
    expect(Calculator.Key).toBe(CalculatorKey);
    expect(Calculator.History).toBe(CalculatorHistory);
    expect(Calculator.Status).toBe(CalculatorStatus);
  });

  it("requires an ApiClientProvider", () => {
    silenceReactErrorLogging();

    expect(() =>
      render(
        <Calculator>
          <Calculator.Display />
        </Calculator>,
      ),
    ).toThrow("useApiClient must be used inside <ApiClientProvider>");
  });

  const parts: [string, () => ReactElement][] = [
    ["Display", () => <Calculator.Display />],
    ["Keypad", () => <Calculator.Keypad />],
    ["Key", () => <Calculator.Key action={{ type: "equals" }} />],
    ["History", () => <Calculator.History />],
    ["Status", () => <Calculator.Status />],
  ];

  it.each(parts)("throws when %s is used outside <Calculator>", (_name, renderPart) => {
    silenceReactErrorLogging();

    expect(() =>
      render(<ApiClientProvider client={createMockApiClient()}>{renderPart()}</ApiClientProvider>),
    ).toThrow("useCalculator must be used inside <Calculator>");
  });

  describe("keyboard support", () => {
    it("types digits and a decimal point", async () => {
      const { user } = renderCalculator(createMockApiClient());

      await user.keyboard("12.5");

      expect(screen.getByTestId("display-value")).toHaveTextContent("12.5");
    });

    it("runs a calculation with +, digits and Enter", async () => {
      const client = createMockApiClient();
      client.calculate.mockResolvedValueOnce({ operation: "add", operands: [2, 3], result: 5 });
      const { user } = renderCalculator(client);

      await user.keyboard("2+3{Enter}");

      expect(client.calculate).toHaveBeenCalledWith({ operation: "add", operands: [2, 3] });
      await waitFor(() => {
        expect(screen.getByTestId("display-value")).toHaveTextContent("5");
      });
    });

    it("maps -, * and / to their operations", async () => {
      const client = createMockApiClient();
      const { user } = renderCalculator(client);

      await user.keyboard("9-");
      expect(screen.getByTestId("display-expression")).toHaveTextContent("9 −");
      await user.keyboard("*");
      expect(screen.getByTestId("display-expression")).toHaveTextContent("9 ×");
      await user.keyboard("/");
      expect(screen.getByTestId("display-expression")).toHaveTextContent("9 ÷");
      expect(client.calculate).not.toHaveBeenCalled();
    });

    it("clears on Escape and deletes on Backspace", async () => {
      const { user } = renderCalculator(createMockApiClient());

      await user.keyboard("123{Backspace}");
      expect(screen.getByTestId("display-value")).toHaveTextContent("12");

      await user.keyboard("{Escape}");
      expect(screen.getByTestId("display-value")).toHaveTextContent("0");
    });

    it("leaves Enter to the browser when a key has focus", async () => {
      const client = createMockApiClient();
      const { user } = renderCalculator(client);

      await user.keyboard("2+3");
      await user.click(screen.getByTestId("key-5"));
      expect(screen.getByTestId("key-5")).toHaveFocus();

      await user.keyboard("{Enter}");

      // The browser turns Enter on a focused button into a click on that
      // button, so the calculator must not also read it as `=`.
      expect(client.calculate).not.toHaveBeenCalled();
    });

    it("stays out of the way while the user types in a text field", async () => {
      const client = createMockApiClient();
      const { user } = renderWithProviders(
        <>
          <input aria-label="note" />
          <Calculator>
            <Calculator.Display />
          </Calculator>
        </>,
        { client },
      );

      await user.click(screen.getByLabelText("note"));
      await user.keyboard("7");

      expect(screen.getByLabelText("note")).toHaveValue("7");
      expect(screen.getByTestId("display-value")).toHaveTextContent("0");
    });

    it("stops listening once unmounted", async () => {
      const { user, unmount } = renderCalculator(createMockApiClient());

      await user.keyboard("7");
      expect(screen.getByTestId("display-value")).toHaveTextContent("7");

      unmount();
      await user.keyboard("8");

      expect(screen.queryByTestId("display-value")).not.toBeInTheDocument();
    });
  });
});
