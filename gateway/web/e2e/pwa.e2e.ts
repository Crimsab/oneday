import { expect, test } from "@playwright/test";

test("is installable and keeps canonical story data network-only", async ({ page, context }) => {
  await page.goto("/");
  await page.evaluate(() => navigator.serviceWorker.ready.then(() => undefined));
  await page.reload();

  const registration = await page.evaluate(async () => {
    const ready = await navigator.serviceWorker.ready;
    return {
      controlled: Boolean(navigator.serviceWorker.controller),
      scope: ready.scope,
      scriptURL: ready.active?.scriptURL ?? ready.waiting?.scriptURL ?? ready.installing?.scriptURL ?? "",
    };
  });

  expect(registration.controlled).toBe(true);
  expect(registration.scope).toBe("http://127.0.0.1:44174/");
  expect(registration.scriptURL).toMatch(/\/sw\.js$/);

  const cdp = await context.newCDPSession(page);
  await cdp.send("Page.enable");
  const manifest = await cdp.send("Page.getAppManifest");
  expect(manifest.url).toBe("http://127.0.0.1:44174/manifest.webmanifest");
  expect(manifest.errors).toEqual([]);

  const installability = await cdp.send("Page.getInstallabilityErrors");
  expect(installability.installabilityErrors).toEqual([]);

  const cachedPaths = await page.evaluate(async () => {
    const keys = await caches.keys();
    const requests = (await Promise.all(keys.map((key) => caches.open(key).then((cache) => cache.keys())))).flat();
    return requests.map((request) => new URL(request.url).pathname);
  });
  expect(cachedPaths).toContain("/index.html");
  expect(cachedPaths.some((pathname) => pathname.startsWith("/api/"))).toBe(false);
  expect(cachedPaths.some((pathname) => pathname.startsWith("/generated/"))).toBe(false);

  await context.setOffline(true);
  try {
    await page.goto("/stories/story-1/history");
    await expect(page).toHaveTitle("OneDay");
    await expect(page.locator("#root")).not.toBeEmpty();

    const dynamicFetches = await page.evaluate(async () => Promise.all([
      fetch("/api/health").then(() => "resolved", () => "rejected"),
      fetch("/generated/assets/story/scene.png").then(() => "resolved", () => "rejected"),
    ]));
    expect(dynamicFetches).toEqual(["rejected", "rejected"]);
  } finally {
    await context.setOffline(false);
  }
});
