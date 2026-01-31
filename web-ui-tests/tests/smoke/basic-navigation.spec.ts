/**
 * Smoke tests for the mehrhof Web UI.
 *
 * These are fast, basic tests that verify the application loads
 * and core functionality is accessible. They run in CI as part
 * of the smoke test suite.
 */

import { test, expect } from '@playwright/test';
import { startTestServer } from '../../helpers/server';

let server: Awaited<ReturnType<typeof import('../../helpers/server').startTestServer>>;

test.describe.configure({ mode: 'serial' });

test.beforeAll(async () => {
  server = await startTestServer();
});

test.afterAll(async () => {
  if (server) {
    await server.stop();
  }
});

// Helper function to get server URL
const baseURL = () => (server ? server.url : 'http://localhost:8080');

test.describe('Smoke Tests', () => {
  test('homepage loads', async ({ page }) => {
    await page.goto(baseURL());

    // Check that we get a successful response
    const title = await page.title();
    expect(title).toBeTruthy();
  });

  test('dashboard page loads', async ({ page }) => {
    await page.goto(baseURL());

    // Should have the main navigation
    const nav = page.locator('nav');
    await expect(nav).toBeVisible();

    // Should have the main content area
    const main = page.getByTestId('main-content');
    await expect(main).toBeVisible();
  });

  test('navigation menu works', async ({ page }) => {
    await page.goto(baseURL());

    // Find navigation links
    const navLinks = page.locator('nav a');

    // Should have at least some navigation
    const count = await navLinks.count();
    expect(count).toBeGreaterThan(0);
  });

  test('API health endpoint responds', async ({ request }) => {
    const response = await request.get(`${baseURL()}/api/v1/health`);
    expect(response.status()).toBe(200);
  });

  test('status endpoint returns current state', async ({ request }) => {
    const response = await request.get(`${baseURL()}/api/v1/status`);
    expect(response.status()).toBe(200);

    const data = (await response.json()) as { state: string };
    expect(data).toHaveProperty('state');
  });

  test('theme toggle works', async ({ page }) => {
    await page.goto(baseURL());

    // Get initial theme
    const html = page.locator('html');
    const initialTheme = await html.getAttribute('data-theme');

    // Find and click theme toggle
    const themeToggle = page.getByTestId('theme-toggle');
    await themeToggle.click();

    // Theme should have changed
    const newTheme = await html.getAttribute('data-theme');
    expect(newTheme).not.toBe(initialTheme);
  });

  test('interactive page loads', async ({ page }) => {
    await page.goto(`${baseURL()}/interactive`);

    // Should have interactive mode container
    const interactive = page.getByTestId('interactive-container');
    await expect(interactive).toBeVisible();
  });

  test('settings page loads', async ({ page }) => {
    await page.goto(`${baseURL()}/settings`);

    // Should have settings container
    const settings = page.getByTestId('settings-container');
    await expect(settings).toBeVisible();
  });

  test('workflow diagram endpoint returns valid response', async ({ request }) => {
    const response = await request.get(`${baseURL()}/api/v1/workflow/diagram`);
    // Should return either 200 (SVG) or 401/503 (no conductor/task)
    expect([200, 401, 503]).toContain(response.status());
  });
});
