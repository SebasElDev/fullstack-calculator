import { afterEach, describe, expect, it, vi } from "vitest";
import { createIdFactory } from "@/lib/ids";

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("createIdFactory", () => {
  it("uses crypto.randomUUID when the page is a secure context", () => {
    const randomUUID = vi.fn(() => "11111111-2222-4333-8444-555555555555" as const);
    vi.stubGlobal("crypto", { randomUUID });

    expect(createIdFactory("history")()).toBe("11111111-2222-4333-8444-555555555555");
    expect(randomUUID).toHaveBeenCalledTimes(1);
  });

  it("falls back to a prefixed counter when randomUUID is unavailable", () => {
    vi.stubGlobal("crypto", {});
    const nextId = createIdFactory("history");

    expect([nextId(), nextId()]).toEqual(["history-1", "history-2"]);
  });

  it("falls back when there is no crypto object at all", () => {
    vi.stubGlobal("crypto", undefined);

    expect(createIdFactory("entry")()).toBe("entry-1");
  });

  it("keeps separate factories independent", () => {
    vi.stubGlobal("crypto", {});

    expect([createIdFactory("a")(), createIdFactory("b")()]).toEqual(["a-1", "b-1"]);
  });
});
