import { defineConfig } from '@playwright/test';

export default defineConfig({
  testDir: './tests/ui',
  testIgnore: '**/crossbrowser-matrix.spec.mjs', // separate real multi-engine fixture workflow
  timeout: 30000,
  expect: { timeout: 8000 },
  fullyParallel: true,
  retries: 0,
  workers: 2,
  reporter: [['list'], ['html', { outputFolder: 'build/playwright-report', open: 'never' }]],
  outputDir: 'build/playwright-results',
  webServer: {
    command: 'python3 -m http.server 4173 --bind 127.0.0.1 --directory web',
    url: 'http://127.0.0.1:4173',
    reuseExistingServer: !process.env.CI,
    timeout: 12000,
  },
  use: {
    baseURL: 'http://127.0.0.1:4173',
    trace: 'retain-on-failure',
    screenshot: 'only-on-failure',
  },
  projects: [
    { name: 'desktop-chromium', use: { browserName: 'chromium', viewport: { width: 1440, height: 900 } } },
    { name: 'mobile-chromium', use: { browserName: 'chromium', viewport: { width: 390, height: 844 }, isMobile: true, hasTouch: true } },
  ],
});
