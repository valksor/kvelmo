import { defineConfig, devices } from '@playwright/test';

/**
 * Playwright configuration for mehrhof Web UI testing.
 *
 * CI-optimized settings:
 * - Chromium only (skip firefox/webkit to reduce overhead)
 * - Single worker to avoid race conditions
 * - Retries in CI for flaky network tests
 * - Video/screenshots on failure only
 */
export default defineConfig({
  testDir: './tests',
  fullyParallel: false,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: 1,
  reporter: [['html', { open: 'never', outputFolder: 'playwright-report' }], ['list']],
  timeout: 30_000, // 30 seconds per test
  expect: {
    timeout: 5_000,
  },
  use: {
    baseURL: process.env.BASE_URL || 'http://localhost:8080',
    headless: true,
    viewport: { width: 1280, height: 720 },
    ignoreHTTPSErrors: true,
    video: 'retain-on-failure',
    screenshot: 'only-on-failure',
    trace: 'retain-on-failure',
    actionTimeout: 10_000,
  },
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],
  // Only run in Chromium for CI - skip firefox/webkit to reduce overhead
});
