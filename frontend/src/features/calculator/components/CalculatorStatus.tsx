import { useEffect, useState } from "react";
import { useApiClient } from "@/features/calculator/context/ApiClientContext";
import { useCalculator } from "@/features/calculator/context/CalculatorContext";
import { cn } from "@/lib/cn";

/** How often `/health` is re-checked. */
export const HEALTH_POLL_INTERVAL_MS = 30_000;

type HealthState = "checking" | "online" | "offline";

const HEALTH_LABELS: Record<HealthState, string> = {
  checking: "Checking API",
  online: "API online",
  offline: "API offline",
};

const HEALTH_DOT_CLASSES: Record<HealthState, string> = {
  checking: "bg-slate-500 animate-pulse",
  online: "bg-emerald-400",
  offline: "bg-danger",
};

/** API reachability, polled on mount and every {@link HEALTH_POLL_INTERVAL_MS}. */
export function CalculatorStatus() {
  const client = useApiClient();
  const { state } = useCalculator();
  const [health, setHealth] = useState<HealthState>("checking");
  const [version, setVersion] = useState<string | null>(null);

  useEffect(() => {
    const controller = new AbortController();

    const check = async () => {
      try {
        const response = await client.health(controller.signal);
        if (controller.signal.aborted) {
          return;
        }
        setHealth("online");
        setVersion(response.version);
      } catch {
        if (controller.signal.aborted) {
          return;
        }
        setHealth("offline");
        setVersion(null);
      }
    };

    void check();
    const timer = setInterval(() => {
      void check();
    }, HEALTH_POLL_INTERVAL_MS);

    return () => {
      controller.abort();
      clearInterval(timer);
    };
  }, [client]);

  return (
    <div
      data-testid="status"
      data-health={health}
      className="mb-3 flex items-center justify-between gap-2 text-xs text-slate-400"
    >
      <span className="inline-flex items-center gap-2">
        <span
          className={cn("size-2 rounded-full", HEALTH_DOT_CLASSES[health])}
          aria-hidden="true"
        />
        <span>{HEALTH_LABELS[health]}</span>
        {version !== null && (
          <span data-testid="status-version" className="text-slate-600">
            {version}
          </span>
        )}
      </span>
      {state.status === "calculating" && (
        <span data-testid="status-calculating" className="text-accent">
          calculating…
        </span>
      )}
    </div>
  );
}
