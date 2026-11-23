import {defineConfig} from "@playwright/test";

// Actual compiled Go server with a localhost synthetic GraphQL provider, not a mocked web API.
export default defineConfig({
  testDir: "./tests/ui",
  testMatch: "**/integrated-service.spec.mjs",
  timeout: 45000,
  expect: {timeout: 12000},
  workers: 1,
  retries: 0,
  outputDir: "build/playwright-integrated-service-results",
  reporter: [["list"], ["html", {outputFolder: "build/playwright-integrated-service-report", open: "never"}]],
  webServer: {
    command: "node scripts/ux/integrated-service-fixture.mjs",
    url: "http://127.0.0.1:4178/healthz",
    reuseExistingServer: !process.env.CI,
    timeout: 30000
  },
  use: {
    baseURL: "http://127.0.0.1:4178",
    browserName: "chromium",
    viewport: {width: 1280, height: 800},
    trace: "retain-on-failure",
    screenshot: "only-on-failure"
  }
});
