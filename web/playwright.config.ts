import { defineConfig, devices } from "@playwright/test";

/**
 * The studio driven in a browser against a real kite binary, each test on a
 * throwaway site of its own (e2e/site.ts), so tests run in any order and in
 * parallel.
 */
export default defineConfig({
  testDir: "e2e",
  globalSetup: "./e2e/global-setup.ts",
  fullyParallel: true,
  forbidOnly: Boolean(process.env.CI),
  reporter: [["list"], ["html", { open: "never" }]],
  expect: { timeout: 10_000 },
  use: {
    // The studio speaks the browser's language, and the tests read English.
    locale: "en-US",
    timezoneId: "UTC",
    trace: "retain-on-failure",
  },
  projects: [{ name: "chromium", use: { ...devices["Desktop Chrome"] } }],
});
