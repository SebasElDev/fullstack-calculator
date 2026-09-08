import { type ReactNode, useEffect } from "react";
import { CalculatorDisplay } from "@/features/calculator/components/CalculatorDisplay";
import { CalculatorHistory } from "@/features/calculator/components/CalculatorHistory";
import { CalculatorKey } from "@/features/calculator/components/CalculatorKey";
import { CalculatorKeypad } from "@/features/calculator/components/CalculatorKeypad";
import { CalculatorStatus } from "@/features/calculator/components/CalculatorStatus";
import { CalculatorProvider, useCalculator } from "@/features/calculator/context/CalculatorContext";
import { keyActionFromKeyboardEvent } from "@/features/calculator/state/keyboard";
import { cn } from "@/lib/cn";

/** Physical keyboard support (docs/ARCHITECTURE.md §3.4). */
function useKeyboardControls(): void {
  const { press } = useCalculator();

  useEffect(() => {
    const onKeyDown = (event: KeyboardEvent) => {
      const action = keyActionFromKeyboardEvent(event);
      if (action === null) {
        return;
      }
      event.preventDefault();
      press(action);
    };

    window.addEventListener("keydown", onKeyDown);
    return () => {
      window.removeEventListener("keydown", onKeyDown);
    };
  }, [press]);
}

interface CalculatorShellProps {
  children: ReactNode;
  className?: string;
}

function CalculatorShell({ children, className }: CalculatorShellProps) {
  useKeyboardControls();

  return (
    <section
      data-testid="calculator"
      aria-label="Calculator"
      className={cn(
        "w-full max-w-sm rounded-3xl border border-white/5 bg-card p-4",
        "shadow-2xl shadow-black/60",
        className,
      )}
    >
      {children}
    </section>
  );
}

export interface CalculatorProps {
  /** Compose the parts in any order: Status, Display, Keypad, History. */
  children: ReactNode;
  className?: string;
}

function CalculatorRoot({ children, className }: CalculatorProps) {
  return (
    <CalculatorProvider>
      <CalculatorShell className={className}>{children}</CalculatorShell>
    </CalculatorProvider>
  );
}

/**
 * Compound component root: provides the calculator context and exposes its
 * parts. Each part reads context, so using one outside `<Calculator>` throws.
 */
export const Calculator = Object.assign(CalculatorRoot, {
  Display: CalculatorDisplay,
  Keypad: CalculatorKeypad,
  Key: CalculatorKey,
  History: CalculatorHistory,
  Status: CalculatorStatus,
});
