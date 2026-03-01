import { test, expect } from '@playwright/test'

test.describe('ProjectView in demo mode', () => {
  test('shows project header with mock project name', async ({ page }) => {
    await page.goto('/?demo')

    // Demo mode injects a mock project at /Users/demo/workspace/my-project
    await expect(page.getByRole('heading', { name: 'my-project' })).toBeVisible()
  })

  test('displays back button to projects', async ({ page }) => {
    await page.goto('/?demo')

    // Back link shows "Projects" text
    await expect(page.getByText('Projects').first()).toBeVisible()
  })

  test('shows task widget in left sidebar', async ({ page }) => {
    await page.goto('/?demo')

    // Widget title "Task" in the left sidebar (rendered as span)
    const leftSidebar = page.getByRole('complementary', { name: 'Left sidebar' })
    await expect(leftSidebar.getByText('Task', { exact: true })).toBeVisible()
  })

  test('shows actions widget in right sidebar', async ({ page }) => {
    await page.goto('/?demo')

    const rightSidebar = page.getByRole('complementary', { name: 'Right sidebar' })
    await expect(rightSidebar.getByText('Actions', { exact: true })).toBeVisible()
  })

  test('shows file changes widget', async ({ page }) => {
    await page.goto('/?demo')

    await expect(page.getByText('FILE CHANGES')).toBeVisible()
  })

  test('shows agents widget', async ({ page }) => {
    await page.goto('/?demo')

    await expect(page.getByText('AGENTS')).toBeVisible()
  })

  test('displays status badge with No Task state', async ({ page }) => {
    await page.goto('/?demo')

    // Demo mode sets state to 'idle', which shows 'No Task'
    await expect(page.getByText('No Task')).toBeVisible()
  })

  test('shows chat area with placeholder', async ({ page }) => {
    await page.goto('/?demo')

    await expect(page.getByText('Start a conversation')).toBeVisible()
    await expect(page.getByPlaceholder(/type a message/i)).toBeVisible()
  })

  test('settings button is present', async ({ page }) => {
    await page.goto('/?demo')

    const settingsButton = page.getByRole('button', { name: /settings/i })
    await expect(settingsButton).toBeVisible()
  })

  test('action buttons are present', async ({ page }) => {
    await page.goto('/?demo')

    // Action buttons in the ACTIONS widget
    await expect(page.getByRole('button', { name: 'Plan' })).toBeVisible()
    await expect(page.getByRole('button', { name: 'Implement' })).toBeVisible()
  })

  test('output panel is present', async ({ page }) => {
    await page.goto('/?demo')

    await expect(page.getByText('Output', { exact: true })).toBeVisible()
    await expect(page.getByText('No output yet')).toBeVisible()
  })
})
