import { defineConfig, devices } from '@playwright/test';

const isCoverage = process.env.COVERAGE === 'true';

/**
 * Playwright configuration for mehrhof Web UI testing.
 *
 * CI-optimized settings:
 * - Chromium only (skip firefox/webkit to reduce overhead)
 * - Single worker to avoid race conditions
 * - Retries in CI for flaky network tests
 * - Video/screenshots on failure only
 * - V8 coverage collection when COVERAGE=true
 */
export default defineConfig({
  testDir: './tests',
  fullyParallel: false,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: 1,
  reporter: isCoverage
    ? [
        [
          'monocart-reporter',
          {
            name: 'Mehrhof Web UI Coverage',
            outputFile: './coverage/report.html',
            coverage: {
              entryFilter: (entry: { url: string }) => {
                // Only cover our app's JS, not third-party libs
                return entry.url.includes('/static/js/') && !entry.url.includes('.min.js');
              },
              sourceFilter: (sourcePath: string) => {
                // Include our source files
                return sourcePath.includes('/static/js/');
              },
              sourcePath: (filePath: string) => {
                // V8 coverage produces URL-based paths like "localhost-PORT/static/js/actions.js"
                // Remap to repo-relative paths so Coveralls can match them to source files
                const match = filePath.match(/static\/js\/(.*)/);
                if (match) {
                  let fileName = match[1];
                  // Strip query parameter artifacts (e.g., "app.js-v=2" → "app.js")
                  fileName = fileName.replace(/(\.\w+)-.+$/, '$1');
                  return `internal/server/static/js/${fileName}`;
                }
                return filePath;
              },
              reports: [['lcovonly', { file: 'lcov.info' }], ['text-summary'], ['html']],
              outputDir: './coverage',
            },
          },
        ],
        ['list'],
      ]
    : [['html', { open: 'never', outputFolder: 'playwright-report' }], ['list']],
  timeout: 30_000,
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
});
