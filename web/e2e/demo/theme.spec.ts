import { test, expect } from '@playwright/test'

test.describe('Theme switching in demo mode', () => {
  test('page has theme applied', async ({ page }) => {
    await page.goto('/?demo')

    // DaisyUI applies data-theme attribute to html element
    const html = page.locator('html')
    const theme = await html.getAttribute('data-theme')
    expect(theme).toBeTruthy()
  })

  test('theme toggle changes theme', async ({ page }) => {
    await page.goto('/?demo')

    const html = page.locator('html')
    const initialTheme = await html.getAttribute('data-theme')

    // ThemeToggle is a button with title "Switch to light" or "Switch to dark"
    const themeToggle = page.getByRole('button', { name: /switch to/i })
    await expect(themeToggle).toBeVisible()
    await themeToggle.click()

    // Theme should change - use auto-waiting assertion
    await expect(html).not.toHaveAttribute('data-theme', initialTheme!)
  })

  test('theme persists after toggle', async ({ page }) => {
    await page.goto('/?demo')

    const html = page.locator('html')

    // Toggle theme (fail if toggle not found)
    const themeToggle = page.getByRole('button', { name: /switch to/i })
    await expect(themeToggle).toBeVisible()

    const initialTheme = await html.getAttribute('data-theme')
    await themeToggle.click()

    // Wait for theme to change
    await expect(html).not.toHaveAttribute('data-theme', initialTheme!)
    const newTheme = await html.getAttribute('data-theme')

    // Reload page
    await page.reload()

    // Theme should persist (stored in localStorage)
    await expect(html).toHaveAttribute('data-theme', newTheme!)
  })
})
