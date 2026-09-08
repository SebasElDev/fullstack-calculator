import { screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { Calculator } from "@/features/calculator";
import { createMockApiClient, renderWithProviders } from "@/test/utils";

function renderHistory() {
  return renderWithProviders(
    <Calculator>
      <Calculator.Display />
      <Calculator.Keypad />
      <Calculator.History className="tall" />
    </Calculator>,
    { client: createMockApiClient() },
  );
}

describe("<Calculator.History>", () => {
  it("explains that nothing has been calculated yet", () => {
    renderHistory();

    expect(screen.getByTestId("history")).toHaveAccessibleName("History");
    expect(screen.getByTestId("history")).toHaveClass("tall");
    expect(screen.getByTestId("history-empty")).toHaveTextContent("No calculations yet");
    expect(screen.queryAllByTestId("history-entry")).toHaveLength(0);
  });

  it("lists server results newest first", async () => {
    const { client, user } = renderHistory();
    client.calculate
      .mockResolvedValueOnce({ operation: "sqrt", operands: [9], result: 3 })
      .mockResolvedValueOnce({ operation: "add", operands: [3, 1], result: 4 });

    await user.click(screen.getByTestId("key-9"));
    await user.click(screen.getByTestId("key-sqrt"));
    await waitFor(() => {
      expect(screen.getByTestId("display-value")).toHaveTextContent("3");
    });

    await user.click(screen.getByTestId("key-add"));
    await user.click(screen.getByTestId("key-1"));
    await user.click(screen.getByTestId("key-equals"));
    await waitFor(() => {
      expect(screen.getByTestId("display-value")).toHaveTextContent("4");
    });

    expect(screen.getAllByTestId("history-entry").map((item) => item.textContent)).toEqual([
      "3 + 1 = 4",
      "√(9) = 3",
    ]);
    expect(screen.queryByTestId("history-empty")).not.toBeInTheDocument();
  });
});
