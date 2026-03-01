import { test, expect } from '@playwright/test'

test.describe('Accessibility in demo mode', () => {
  test('main content has proper landmark', async ({ page }) => {
    await page.goto('/?demo')

    const main = page.locator('main#main-content')
    await expect(main).toBeVisible()
    await expect(main).toHaveAttribute('tabindex', '-1')
  })

  test('sidebar landmarks are present', async ({ page }) => {
    await page.goto('/?demo')

    // Left and right sidebars should have proper landmark roles
    await expect(page.getByRole('complementary', { name: 'Left sidebar' })).toBeVisible()
    await expect(page.getByRole('complementary', { name: 'Right sidebar' })).toBeVisible()
  })

  test('settings button has accessible name', async ({ page }) => {
    await page.goto('/?demo')

    const settingsButton = page.getByRole('button', { name: /settings/i })
    await expect(settingsButton).toBeVisible()
  })

  test('chat input has placeholder for screen readers', async ({ page }) => {
    await page.goto('/?demo')

    const chatInput = page.getByPlaceholder(/type a message/i)
    await expect(chatInput).toBeVisible()
  })

  test('no duplicate IDs on page', async ({ page }) => {
    await page.goto('/?demo')

    // Check for duplicate IDs which is a common accessibility issue
    const ids = await page.evaluate(() => {
      const elements = document.querySelectorAll('[id]')
      const idList = Array.from(elements).map(el => el.id).filter(id => id)
      const duplicates = idList.filter((id, index) => idList.indexOf(id) !== index)
      return duplicates
    })

    expect(ids).toHaveLength(0)
  })

  test('interactive elements are focusable', async ({ page }) => {
    await page.goto('/?demo')

    // Tab through and ensure we can reach key elements
    await page.keyboard.press('Tab')

    // Should be able to focus an interactive element (not body)
    const focusedElement = await page.evaluate(() => {
      const el = document.activeElement
      return {
        tagName: el?.tagName,
        isInteractive: el !== document.body && el !== document.documentElement
      }
    })
    expect(focusedElement.isInteractive, 'Expected Tab to focus an interactive element').toBe(true)
  })

  test('action buttons have accessible names', async ({ page }) => {
    await page.goto('/?demo')

    // All action buttons should have clear names
    await expect(page.getByRole('button', { name: 'Plan' })).toBeVisible()
    await expect(page.getByRole('button', { name: 'Implement' })).toBeVisible()
  })
})
