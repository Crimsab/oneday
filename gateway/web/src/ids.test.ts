import { afterEach, describe, expect, it } from "vitest";
import { clientId } from "./ids";

const originalCrypto = Object.getOwnPropertyDescriptor(globalThis, "crypto");

describe("clientId", () => {
  afterEach(() => {
    if (originalCrypto) {
      Object.defineProperty(globalThis, "crypto", originalCrypto);
    } else {
      Reflect.deleteProperty(globalThis, "crypto");
    }
  });

  it("uses crypto.randomUUID when available", () => {
    Object.defineProperty(globalThis, "crypto", {
      configurable: true,
      value: { randomUUID: () => "00000000-0000-4000-8000-000000000000" },
    });
    expect(clientId("turn")).toBe("turn-00000000-0000-4000-8000-000000000000");
  });

  it("falls back to getRandomValues when randomUUID is unavailable", () => {
    Object.defineProperty(globalThis, "crypto", {
      configurable: true,
      value: {
        getRandomValues: (bytes: Uint8Array) => {
          bytes.fill(1);
          return bytes;
        },
      },
    });
    expect(clientId("turn")).toMatch(/^turn-[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/);
  });

  it("still returns an id without Web Crypto", () => {
    Reflect.deleteProperty(globalThis, "crypto");
    expect(clientId("command")).toMatch(/^command-[0-9a-f-]+$/);
  });
});
