import { defineConfig } from '@playwright/test';

// Pairwise supplemental coverage, not physical Android-device certification.
// Reuses the same synthetic source-shaped real-browser journeys and service.
export default defineConfig({
  testDir: './tests/ui',
  testMatch: ['**/crossbrowser-matrix.spec.mjs', '**/accessibility-recovery.spec.mjs'],
  timeout: 30000,
  expect: { timeout: 10000 },
  workers: 2,
  retries: 0,
  reporter: [['list'], ['html', { outputFolder: 'build/playwright-edge-mobile-report', open: 'never' }]],
  outputDir: 'build/playwright-edge-mobile-results',
  webServer: {
    command: 'node scripts/ux/fixture-server.mjs',
    url: 'http://127.0.0.1:4175/healthz',
    reuseExistingServer: !process.env.CI,
    timeout: 12000,
  },
  use: {
    baseURL: 'http://127.0.0.1:4175',
    trace: 'retain-on-failure',
    screenshot: 'only-on-failure',
  },
  projects: [
    { name: 'edge-desktop', use: { browserName: 'chromium', channel: 'msedge', viewport: { width: 1366, height: 768 } } },
    { name: 'chromium-android-small-emulated', use: {
      browserName: 'chromium', viewport: { width: 360, height: 740 },
      isMobile: true, hasTouch: true, deviceScaleFactor: 3,
    } },
  ],
});
