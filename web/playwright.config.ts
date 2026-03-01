import { defineConfig, devices } from '@playwright/test'

export default defineConfig({
  testDir: './e2e',
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: process.env.CI ? 1 : undefined,
  reporter: 'html',
  use: {
    headless: true,
    trace: 'on-first-retry',
    screenshot: 'only-on-failure',
  },

  projects: [
    // Demo mode tests - safe, no backend needed
    {
      name: 'demo',
      testDir: './e2e/demo',
      use: {
        ...devices['Desktop Chrome'],
        baseURL: 'http://localhost:5173',
      },
    },
    // Integration tests - requires real backend with Claude
    {
      name: 'integration',
      testDir: './e2e/integration',
      // Run serially - tests may share state
      fullyParallel: false,
      // Longer timeouts for real Claude calls
      timeout: 300_000, // 5 minutes per test
      use: {
        ...devices['Desktop Chrome'],
        // Integration tests hit the real backend (port 6337)
        baseURL: 'http://localhost:6337',
        // Longer action timeout for Claude responses
        actionTimeout: 180_000, // 3 minutes
      },
    },
  ],

  webServer: [
    // Vite dev server for demo tests
    {
      command: 'bun run dev',
      url: 'http://localhost:5173',
      reuseExistingServer: !process.env.CI,
    },
    // Note: Integration tests expect `make run` to be running separately
    // The backend serves the frontend too, so we don't need vite for integration
  ],
})
