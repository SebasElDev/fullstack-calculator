import { screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { Calculator } from "@/features/calculator";
import { ApiError } from "@/lib/api/client";
import { createMockApiClient, type MockApiClient, renderWithProviders } from "@/test/utils";

function renderDisplay(client: MockApiClient = createMockApiClient()) {
  return renderWithProviders(
    <Calculator>
      <Calculator.Display />
      <Calculator.Keypad />
    </Calculator>,
    { client },
  );
}

describe("<Calculator.Display>", () => {
  it("starts at zero with an empty expression line", () => {
    renderDisplay();

    expect(screen.getByTestId("display-value")).toHaveTextContent("0");
    expect(screen.getByTestId("display-expression").textContent).toBe("\u00a0");
  });

  it("is an assertive-free live region so results are announced", () => {
    renderDisplay();

    const display = screen.getByTestId("display");
    expect(display).toHaveAttribute("aria-live", "polite");
    expect(display).toHaveAttribute("aria-atomic", "true");
    expect(display).toHaveAttribute("data-status", "idle");
  });

  it("shows the pending expression while a second operand is typed", async () => {
    const { user } = renderDisplay();

    await user.click(screen.getByTestId("key-8"));
    await user.click(screen.getByTestId("key-subtract"));

    expect(screen.getByTestId("display-expression")).toHaveTextContent("8 −");
    expect(screen.getByTestId("display-value")).toHaveTextContent("8");
  });

  it("replaces the value with the server's error message", async () => {
    const client = createMockApiClient();
    client.calculate.mockRejectedValueOnce(
      new ApiError(400, "UNSUPPORTED_OPERATION", 'unsupported operation "add"'),
    );
    const { user } = renderDisplay(client);

    await user.click(screen.getByTestId("key-4"));
    await user.click(screen.getByTestId("key-square"));

    await waitFor(() => {
      expect(screen.getByTestId("display-value")).toHaveTextContent('unsupported operation "add"');
    });
    expect(screen.getByTestId("display")).toHaveAttribute("data-status", "error");
  });
});
