import { afterEach, describe, expect, it } from "vitest";
import {
  DIGITS,
  isEditableTarget,
  keyActionFromKeyboardEvent,
} from "@/features/calculator/state/keyboard";
import type { KeyAction } from "@/features/calculator/state/types";

interface EventOverrides {
  target?: EventTarget | null;
  ctrlKey?: boolean;
  metaKey?: boolean;
  altKey?: boolean;
}

function keydown(key: string, overrides: EventOverrides = {}): KeyboardEvent {
  const { target = null, ...modifiers } = overrides;
  const event = new KeyboardEvent("keydown", { key, ...modifiers });
  // `target` is only populated while an event is being dispatched.
  Object.defineProperty(event, "target", { value: target });
  return event;
}

const created: HTMLElement[] = [];

function mount<T extends HTMLElement>(element: T): T {
  document.body.append(element);
  created.push(element);
  return element;
}

afterEach(() => {
  for (const element of created.splice(0)) {
    element.remove();
  }
});

describe("keyActionFromKeyboardEvent", () => {
  it.each([...DIGITS])("maps the %s key to a digit action", (digit) => {
    expect(keyActionFromKeyboardEvent(keydown(digit))).toEqual({ type: "digit", digit });
  });

  const bindings: [string, KeyAction][] = [
    [".", { type: "decimal" }],
    ["+", { type: "operator", operation: "add" }],
    ["-", { type: "operator", operation: "subtract" }],
    ["*", { type: "operator", operation: "multiply" }],
    ["/", { type: "operator", operation: "divide" }],
    ["=", { type: "equals" }],
    ["Enter", { type: "equals" }],
    ["Backspace", { type: "backspace" }],
    ["Escape", { type: "clear" }],
  ];

  it.each(bindings)("maps %s", (key, expected) => {
    expect(keyActionFromKeyboardEvent(keydown(key))).toEqual(expected);
  });

  it("ignores unbound keys", () => {
    expect(keyActionFromKeyboardEvent(keydown("a"))).toBeNull();
    expect(keyActionFromKeyboardEvent(keydown("F5"))).toBeNull();
  });

  it.each([{ ctrlKey: true }, { metaKey: true }, { altKey: true }])(
    "ignores shortcuts (%o)",
    (modifiers) => {
      expect(keyActionFromKeyboardEvent(keydown("1", modifiers))).toBeNull();
    },
  );

  it.each(["input", "textarea", "select"])("ignores typing inside a <%s>", (tagName) => {
    const field = mount(document.createElement(tagName));
    expect(keyActionFromKeyboardEvent(keydown("1", { target: field }))).toBeNull();
  });

  it("ignores typing inside a contenteditable element", () => {
    const editable = mount(document.createElement("div"));
    // jsdom does not derive `isContentEditable` from the attribute.
    Object.defineProperty(editable, "isContentEditable", { value: true });
    expect(keyActionFromKeyboardEvent(keydown("1", { target: editable }))).toBeNull();
  });

  it("leaves Enter to the browser when a key has focus", () => {
    const button = mount(document.createElement("button"));
    expect(keyActionFromKeyboardEvent(keydown("Enter", { target: button }))).toBeNull();
    expect(keyActionFromKeyboardEvent(keydown("=", { target: button }))).toEqual({
      type: "equals",
    });
  });

  it("still handles digits when a key has focus", () => {
    const button = mount(document.createElement("button"));
    expect(keyActionFromKeyboardEvent(keydown("5", { target: button }))).toEqual({
      type: "digit",
      digit: "5",
    });
  });
});

describe("isEditableTarget", () => {
  it("is false for non-elements", () => {
    expect(isEditableTarget(null)).toBe(false);
    expect(isEditableTarget(new EventTarget())).toBe(false);
  });

  it("is false for a plain element", () => {
    expect(isEditableTarget(mount(document.createElement("div")))).toBe(false);
  });
});
