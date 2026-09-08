import { useCalculator } from "@/features/calculator/context/CalculatorContext";
import { cn } from "@/lib/cn";

const NON_BREAKING_SPACE = " ";

/**
 * The expression line plus the current value.
 *
 * The value is `state.input` verbatim — a digit string the user typed, or
 * `String(result)` exactly as the server sent it.
 */
export function CalculatorDisplay() {
  const { state } = useCalculator();
  // `error` is set only while `status === "error"`, so it alone decides what
  // the value line shows.
  const errorMessage = state.error?.message ?? null;
  const isCalculating = state.status === "calculating";

  return (
    <div
      data-testid="display"
      data-status={state.status}
      aria-live="polite"
      aria-atomic="true"
      aria-busy={isCalculating}
      className="mb-3 rounded-2xl bg-display px-4 py-4 text-right"
    >
      <p
        data-testid="display-expression"
        className="min-h-5 truncate font-mono text-sm text-slate-400"
      >
        {state.expression ?? NON_BREAKING_SPACE}
      </p>
      <p
        data-testid="display-value"
        className={cn(
          "mt-1 font-mono leading-tight break-all",
          errorMessage === null
            ? "text-4xl font-semibold text-slate-50 tabular-nums sm:text-5xl"
            : "text-base font-medium text-danger",
          isCalculating && "opacity-50",
        )}
      >
        {errorMessage ?? state.input}
      </p>
    </div>
  );
}
