import { describe, expect, it, vi } from "vitest";
import { checkForServiceWorkerUpdate } from "./pwaUpdate";

describe("checkForServiceWorkerUpdate", () => {
  it("bypasses caches before asking the registration to update", async () => {
    const update = vi.fn(async () => undefined);
    const request = vi.fn(async () => new Response("", { status: 200 }));

    await expect(checkForServiceWorkerUpdate(
      "/sw.js",
      { installing: null, update },
      true,
      request as typeof fetch,
    )).resolves.toBe(true);

    expect(request).toHaveBeenCalledWith("/sw.js", {
      cache: "no-store",
      headers: { Cache: "no-store", "Cache-Control": "no-cache" },
    });
    expect(update).toHaveBeenCalledOnce();
  });

  it("does not check while offline or while a worker is installing", async () => {
    const request = vi.fn();
    const update = vi.fn(async () => undefined);
    const installing = {} as ServiceWorker;

    await expect(checkForServiceWorkerUpdate(
      "/sw.js",
      { installing: null, update },
      false,
      request as typeof fetch,
    )).resolves.toBe(false);
    await expect(checkForServiceWorkerUpdate(
      "/sw.js",
      { installing, update },
      true,
      request as typeof fetch,
    )).resolves.toBe(false);

    expect(request).not.toHaveBeenCalled();
    expect(update).not.toHaveBeenCalled();
  });
});
