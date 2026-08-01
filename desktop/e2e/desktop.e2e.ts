import { expect, test, type Page } from "@playwright/test";
import AxeBuilder from "@axe-core/playwright";

declare global {
  interface Window { __TAURI_INTERNALS__: Record<string, unknown>; __QA_COMMANDS__: string[]; }
}

async function mockDesktop(page: Page, options: { updateAvailable?: boolean } = {}) {
  await page.addInitScript(({ updateAvailable }) => {
    window.__QA_COMMANDS__ = [];
    const values: Record<string, unknown> = {
      desktop_state: {
        profile: { mode: "standalone", profileId: "qa-profile" },
        serverUrl: "http://127.0.0.1:38111/",
        lifecycle: { state: "ready", endpoint: "http://127.0.0.1:38111/" },
        startedMinimized: false,
        updater: {
          enabled: true,
          currentVersion: "0.18.0",
          channel: "Stable",
          reason: "Updates are verified with the OneDay release signing key before installation.",
        },
      },
      list_remote_stories: [{ id: "story-1", name: "The Ashen Harbor" }],
      codex_status: {
        available: true,
        source: "system",
        version: "codex-cli 0.146.0",
        authenticated: true,
        managedVersion: "0.146.0",
        message: "Codex is installed and signed in with a local subscription session.",
      },
      claude_status: {
        available: true,
        version: "2.1.39",
        authenticated: false,
        installSupported: false,
        installMethod: null,
        message: "Claude Code is installed. Sign in before enabling it in OneDay Setup.",
      },
      check_update: updateAvailable
        ? { available: true, version: "0.19.0", notes: "Simpler setup, provider parity, and safer updates.", publishedAt: "2026-08-01T00:00:00Z", message: "OneDay 0.19.0 is available. Review it before installing." }
        : { available: false, version: null, notes: null, publishedAt: null, message: "OneDay is up to date." },
      "plugin:autostart|is_enabled": false,
      "plugin:notification|is_permission_granted": false,
    };
    window.__TAURI_INTERNALS__ = {
      invoke: async (command: string) => {
        window.__QA_COMMANDS__.push(command);
        return values[command] ?? null;
      },
      transformCallback: () => 1,
      unregisterCallback: () => undefined,
      convertFileSrc: (value: string) => value,
      metadata: { currentWindow: { label: "settings" }, currentWebview: { label: "settings" } },
    };
  }, { updateAvailable: Boolean(options.updateAvailable) });
}

test("shows every provider and a separate signed update workflow without accessibility violations", async ({ page }, testInfo) => {
  await mockDesktop(page, { updateAvailable: true });
  await page.goto("/");
  await expect(page.getByRole("heading", { name: "Set up this device" })).toBeVisible();
  await expect(page.getByRole("heading", { name: "Codex" })).toBeVisible();
  await expect(page.getByRole("heading", { name: "Claude Code" })).toBeVisible();
  await expect(page.getByRole("heading", { name: "OpenRouter" })).toBeVisible();
  await expect(page.getByRole("heading", { name: "LiteLLM compatible" })).toBeVisible();
  await expect(page.getByRole("button", { name: "Install and restart" })).toBeVisible();
  await expect(page.getByText("Simpler setup, provider parity, and safer updates.")).toBeVisible();

  const results = await new AxeBuilder({ page }).analyze();
  expect(results.violations).toEqual([]);
  await page.screenshot({ path: testInfo.outputPath("desktop-settings.png"), fullPage: true });
});

test("keeps targets, hover, focus, keyboard accordions, and narrow layouts usable", async ({ page }) => {
  await mockDesktop(page);
  await page.goto("/");
  const local = page.getByRole("button", { name: /Run on this device/ });
  await expect(local).toHaveAttribute("aria-pressed", "true");

  const beforeHover = await local.evaluate((element) => getComputedStyle(element).backgroundColor);
  await local.hover();
  const afterHover = await local.evaluate((element) => getComputedStyle(element).backgroundColor);
  expect(afterHover).not.toBe(beforeHover);
  await local.focus();
  expect(await local.evaluate((element) => getComputedStyle(element).outlineStyle)).toBe("solid");

  const behavior = page.locator("summary").filter({ hasText: "Desktop behavior" });
  await behavior.focus();
  await page.keyboard.press("Enter");
  await expect(page.getByRole("checkbox", { name: "Start with the computer" })).toBeVisible();

  const targetHeights = await page.locator("button:visible, summary:visible").evaluateAll((elements) =>
    elements.map((element) => Math.round(element.getBoundingClientRect().height)),
  );
  expect(Math.min(...targetHeights)).toBeGreaterThanOrEqual(40);

  await page.setViewportSize({ width: 390, height: 844 });
  const layout = await page.evaluate(() => ({ page: document.documentElement.scrollWidth, viewport: innerWidth }));
  expect(layout.page).toBeLessThanOrEqual(layout.viewport + 1);
});

test("checks automatically but installs only after an explicit click", async ({ page }) => {
  await mockDesktop(page, { updateAvailable: true });
  await page.goto("/");
  await expect(page.getByRole("button", { name: "Install and restart" })).toBeVisible();
  expect(await page.evaluate(() => window.__QA_COMMANDS__.filter((command) => command === "check_update").length)).toBe(1);
  expect(await page.evaluate(() => window.__QA_COMMANDS__.includes("install_update"))).toBe(false);
  await page.getByRole("button", { name: "Install and restart" }).click();
  await expect.poll(() => page.evaluate(() => window.__QA_COMMANDS__.includes("install_update"))).toBe(true);
});
