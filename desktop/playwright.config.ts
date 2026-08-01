import { defineConfig, devices } from "@playwright/test";

export default defineConfig({
  testDir: "./e2e",
  testMatch: "**/*.e2e.ts",
  fullyParallel: false,
  forbidOnly: true,
  retries: 0,
  reporter: "line",
  use: {
    baseURL: "http://127.0.0.1:44201",
    trace: "retain-on-failure",
    screenshot: "only-on-failure",
  },
  projects: [
    { name: "chromium", use: { ...devices["Desktop Chrome"], viewport: { width: 860, height: 860 } } },
  ],
  webServer: {
    command: "bun run dev -- --host 127.0.0.1 --port 44201",
    url: "http://127.0.0.1:44201",
    reuseExistingServer: false,
  },
});
