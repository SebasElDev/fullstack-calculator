import type { ReactNode } from "react";
import { useCalculator } from "@/features/calculator/context/CalculatorContext";
import type { KeyAction } from "@/features/calculator/state/types";
import { cn } from "@/lib/cn";
import { operationLabel, operationSymbol } from "@/lib/operations";

interface KeyDescriptor {
  /** Stable identifier: `data-testid` becomes `key-${name}`. */
  name: string;
  label: string;
  ariaLabel: string;
}

const STATIC_KEYS: Record<"decimal" | "backspace" | "clear" | "equals", KeyDescriptor> = {
  decimal: { name: "decimal", label: ".", ariaLabel: "decimal point" },
  backspace: { name: "backspace", label: "⌫", ariaLabel: "backspace" },
  clear: { name: "clear", label: "AC", ariaLabel: "clear all" },
  equals: { name: "equals", label: "=", ariaLabel: "equals" },
};

/** Label, accessible name and test id for a key — derived, never hard-coded twice. */
export function describeKey(action: KeyAction): KeyDescriptor {
  switch (action.type) {
    case "digit":
      return { name: action.digit, label: action.digit, ariaLabel: action.digit };
    case "operator":
    case "unary":
      return {
        name: action.operation,
        label: operationSymbol(action.operation),
        ariaLabel: operationLabel(action.operation),
      };
    default:
      return STATIC_KEYS[action.type];
  }
}

const BASE_KEY_CLASSES = [
  "flex h-14 items-center justify-center rounded-2xl text-lg font-medium select-none",
  "transition-[background-color,transform,box-shadow] duration-100",
  "focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-accent",
  "active:scale-95 disabled:cursor-not-allowed disabled:opacity-40 disabled:active:scale-100",
].join(" ");

const KEY_VARIANTS: Record<KeyAction["type"], string> = {
  digit: "bg-key text-slate-50 hover:bg-key-hover",
  decimal: "bg-key text-slate-50 hover:bg-key-hover",
  clear: "bg-key-muted text-warn hover:bg-key-hover",
  backspace: "bg-key-muted text-warn hover:bg-key-hover",
  operator: "bg-accent-muted text-accent hover:bg-accent-muted-hover",
  unary: "bg-key-muted text-slate-200 hover:bg-key-hover",
  equals: "bg-accent text-slate-950 hover:bg-accent-strong",
};

export interface CalculatorKeyProps {
  action: KeyAction;
  /** Overrides the derived glyph. */
  children?: ReactNode;
  className?: string;
}

/** One keypad key. Public, so consumers can compose custom layouts. */
export function CalculatorKey({ action, children, className }: CalculatorKeyProps) {
  const { state, press } = useCalculator();
  const descriptor = describeKey(action);
  const isCalculating = state.status === "calculating";
  const isPending = action.type === "operator" && state.pendingOperation === action.operation;

  return (
    <button
      type="button"
      data-testid={`key-${descriptor.name}`}
      aria-label={descriptor.ariaLabel}
      aria-pressed={action.type === "operator" ? isPending : undefined}
      aria-disabled={isCalculating}
      disabled={isCalculating}
      onClick={() => press(action)}
      className={cn(
        BASE_KEY_CLASSES,
        KEY_VARIANTS[action.type],
        isPending && "ring-2 ring-accent ring-offset-2 ring-offset-card",
        className,
      )}
    >
      {children ?? descriptor.label}
    </button>
  );
}
