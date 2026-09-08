import type { Digit, KeyAction } from "@/features/calculator/state/types";

export const DIGITS: readonly Digit[] = ["0", "1", "2", "3", "4", "5", "6", "7", "8", "9"];

/** Physical keys that map onto a keypad action (docs/ARCHITECTURE.md §3.4). */
const KEY_BINDINGS: Record<string, KeyAction> = {
  ".": { type: "decimal" },
  "+": { type: "operator", operation: "add" },
  "-": { type: "operator", operation: "subtract" },
  "*": { type: "operator", operation: "multiply" },
  "/": { type: "operator", operation: "divide" },
  "=": { type: "equals" },
  Enter: { type: "equals" },
  Backspace: { type: "backspace" },
  Escape: { type: "clear" },
};

const EDITABLE_TAGS = ["INPUT", "TEXTAREA", "SELECT"];

function isDigit(key: string): key is Digit {
  return (DIGITS as readonly string[]).includes(key);
}

/** Typing into a form control must never be hijacked by the calculator. */
export function isEditableTarget(target: EventTarget | null): boolean {
  if (!(target instanceof HTMLElement)) {
    return false;
  }
  return EDITABLE_TAGS.includes(target.tagName) || target.isContentEditable === true;
}

/**
 * Translates a keydown into a {@link KeyAction}, or `null` when the calculator
 * should stay out of the way.
 *
 * `Enter` on a focused key is left to the browser, which turns it into a click
 * on that key — handling it here as well would press two keys at once.
 */
export function keyActionFromKeyboardEvent(event: KeyboardEvent): KeyAction | null {
  if (event.ctrlKey || event.metaKey || event.altKey) {
    return null;
  }
  if (isEditableTarget(event.target)) {
    return null;
  }
  if (event.key === "Enter" && event.target instanceof HTMLButtonElement) {
    return null;
  }
  if (isDigit(event.key)) {
    return { type: "digit", digit: event.key };
  }
  return KEY_BINDINGS[event.key] ?? null;
}
