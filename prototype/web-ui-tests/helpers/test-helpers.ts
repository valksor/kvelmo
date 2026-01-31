/**
 * Shared test helper functions for Playwright tests.
 */

import { Page, Locator } from '@playwright/test';

/**
 * Wait for an element with a specific test ID to be visible.
 */
export async function waitForTestId(page: Page, testId: string, timeout = 5000): Promise<Locator> {
  const locator = page.getByTestId(testId);
  await locator.waitFor({ state: 'visible', timeout });
  return locator;
}

/**
 * Navigate to a page and wait for it to be ready.
 */
export async function navigateTo(page: Page, path: string): Promise<void> {
  await page.goto(path);
  await page.waitForLoadState('networkidle');
}

/**
 * Fill a form field identified by test ID.
 */
export async function fillByTestId(page: Page, testId: string, value: string): Promise<void> {
  const locator = page.getByTestId(testId);
  await locator.fill(value);
}

/**
 * Click a button or element identified by test ID.
 */
export async function clickByTestId(page: Page, testId: string): Promise<void> {
  const locator = page.getByTestId(testId);
  await locator.click();
}

/**
 * Get text content of an element identified by test ID.
 */
export async function getTextByTestId(page: Page, testId: string): Promise<string> {
  const locator = page.getByTestId(testId);
  return (await locator.textContent()) || '';
}

/**
 * Wait for a success notification to appear and disappear.
 */
export async function waitForSuccess(page: Page): Promise<void> {
  const locator = page
    .getByTestId(/notification|toast|alert/)
    .filter({ hasText: /success|completed|done/i });
  await locator.waitFor({ state: 'visible', timeout: 5000 });
  await locator.waitFor({ state: 'hidden', timeout: 5000 }).catch(() => {
    // Some notifications might be sticky, don't fail if it doesn't disappear
  });
}

/**
 * Wait for an error notification to appear.
 */
export async function waitForError(page: Page): Promise<Locator> {
  const locator = page.getByTestId(/notification|toast|alert/).filter({ hasText: /error|failed/i });
  await locator.waitFor({ state: 'visible', timeout: 5000 });
  return locator;
}

/**
 * Check if an element with a test ID exists.
 */
export async function hasTestId(page: Page, testId: string): Promise<boolean> {
  const locator = page.getByTestId(testId);
  const count = await locator.count();
  return count > 0;
}

/**
 * Login helper - sets up authentication for tests.
 *
 * For now, this just navigates to the dashboard since auth can be disabled in tests.
 */
export async function login(page: Page, serverUrl: string): Promise<void> {
  await page.goto(`${serverUrl}/dashboard`);
  // Auth is disabled in test mode (MEHR_DISABLE_AUTH=1)
}

/**
 * Select a theme (light/dark).
 */
export async function setTheme(page: Page, theme: 'light' | 'dark'): Promise<void> {
  const themeToggle = page.getByTestId('theme-toggle');
  const currentTheme = await page.locator('html').getAttribute('data-theme');

  // Check if we need to toggle
  const needsToggle =
    (theme === 'dark' && currentTheme !== 'dark') || (theme === 'light' && currentTheme === 'dark');

  if (needsToggle) {
    await themeToggle.click();
  }
}

/**
 * Get the current theme.
 */
export async function getTheme(page: Page): Promise<'light' | 'dark'> {
  const theme = await page.locator('html').getAttribute('data-theme');
  return theme === 'dark' ? 'dark' : 'light';
}

/**
 * Wait for SSE connection to be established.
 */
export async function waitForSSEConnection(page: Page): Promise<void> {
  // Check for SSE indicator or wait for event source to be ready
  // Uses dynamic window properties that can't be statically typed
  await page
    .waitForFunction(
      () =>
        (window as Record<string, unknown>).EventSource &&
        (window as Record<string, unknown>).sseConnected === true,
      { timeout: 5000 }
    )
    .catch(() => {
      // SSE might not be exposed to window, don't fail
    });
}

/**
 * Take a screenshot on test failure.
 */
export async function screenshotOnFailure(page: Page, testName: string): Promise<void> {
  await page.screenshot({
    path: `test-results/${testName}-failure.png`,
    fullPage: true,
  });
}

/**
 * API client for making requests to the server during tests.
 */
export class TestAPIClient {
  constructor(private baseURL: string) {}

  async get(path: string, options?: RequestInit): Promise<Response> {
    return fetch(`${this.baseURL}${path}`, {
      ...options,
      headers: {
        'Content-Type': 'application/json',
        ...options?.headers,
      },
    });
  }

  async post(path: string, body?: unknown, options?: RequestInit): Promise<Response> {
    return fetch(`${this.baseURL}${path}`, {
      ...options,
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        ...options?.headers,
      },
      body: body ? JSON.stringify(body) : undefined,
    });
  }

  async delete(path: string, options?: RequestInit): Promise<Response> {
    return fetch(`${this.baseURL}${path}`, {
      ...options,
      method: 'DELETE',
      headers: {
        'Content-Type': 'application/json',
        ...options?.headers,
      },
    });
  }
}

/**
 * Create a test API client.
 */
export function createTestClient(baseURL: string): TestAPIClient {
  return new TestAPIClient(baseURL);
}
