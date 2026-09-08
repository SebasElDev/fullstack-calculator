import { useCalculator } from "@/features/calculator/context/CalculatorContext";
import { historyLine } from "@/features/calculator/state/formatting";
import { cn } from "@/lib/cn";

export interface CalculatorHistoryProps {
  className?: string;
}

/** Session history of server results, newest first. */
export function CalculatorHistory({ className }: CalculatorHistoryProps) {
  const { state } = useCalculator();

  return (
    <section
      data-testid="history"
      aria-label="History"
      className={cn("mt-3 rounded-2xl bg-display px-4 py-3", className)}
    >
      <h2 className="text-[0.7rem] font-semibold tracking-widest text-slate-500 uppercase">
        History
      </h2>
      {state.history.length === 0 ? (
        <p data-testid="history-empty" className="mt-2 text-sm text-slate-500">
          No calculations yet
        </p>
      ) : (
        <ol className="mt-2 max-h-32 space-y-1 overflow-y-auto font-mono text-sm text-slate-300">
          {state.history.map((entry) => (
            <li key={entry.id} data-testid="history-entry" className="truncate">
              {historyLine(entry.expression, entry.result)}
            </li>
          ))}
        </ol>
      )}
    </section>
  );
}
