import type { ReactNode } from "react";
import { CalculatorKey, describeKey } from "@/features/calculator/components/CalculatorKey";
import { useCalculator } from "@/features/calculator/context/CalculatorContext";
import type { KeyAction } from "@/features/calculator/state/types";
import { cn } from "@/lib/cn";

/**
 * Default 6 × 4 grid (docs/ARCHITECTURE.md §3.4). Every operation, digit, `.`,
 * `AC`, `⌫` and `=` is reachable.
 */
export const KEYPAD_LAYOUT: readonly (readonly KeyAction[])[] = [
  [
    { type: "clear" },
    { type: "backspace" },
    { type: "operator", operation: "modulo" },
    { type: "operator", operation: "divide" },
  ],
  [
    { type: "unary", operation: "sqrt" },
    { type: "unary", operation: "square" },
    { type: "operator", operation: "power" },
    { type: "operator", operation: "multiply" },
  ],
  [
    { type: "digit", digit: "7" },
    { type: "digit", digit: "8" },
    { type: "digit", digit: "9" },
    { type: "operator", operation: "subtract" },
  ],
  [
    { type: "digit", digit: "4" },
    { type: "digit", digit: "5" },
    { type: "digit", digit: "6" },
    { type: "operator", operation: "add" },
  ],
  [
    { type: "digit", digit: "1" },
    { type: "digit", digit: "2" },
    { type: "digit", digit: "3" },
    { type: "equals" },
  ],
  [
    { type: "unary", operation: "negate" },
    { type: "digit", digit: "0" },
    { type: "decimal" },
    { type: "unary", operation: "percent" },
  ],
];

export interface CalculatorKeypadProps {
  /** Custom layout; defaults to {@link KEYPAD_LAYOUT}. */
  children?: ReactNode;
  className?: string;
}

export function CalculatorKeypad({ children, className }: CalculatorKeypadProps) {
  const { state } = useCalculator();

  return (
    <fieldset
      data-testid="keypad"
      aria-label="Keypad"
      aria-busy={state.status === "calculating"}
      className={cn("grid grid-cols-4 gap-2", className)}
    >
      {children ??
        KEYPAD_LAYOUT.flat().map((action) => (
          <CalculatorKey key={describeKey(action).name} action={action} />
        ))}
    </fieldset>
  );
}
