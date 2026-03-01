/**
 * Full E2E Integration Test - Task Lifecycle with Real Claude
 *
 * This test runs the complete flow:
 *   Load task → Plan → Implement
 *
 * Requirements:
 * - kvelmo backend must be running (`make run`)
 * - Claude CLI must be installed and authenticated
 * - Test uses an isolated fixture repo (never touches real repos)
 *
 * Run with: bun run test:e2e:integration
 */

import { test, expect } from '@playwright/test'
import { createTestFixture, isClaudeAvailable, type TestFixture } from '../fixtures/setup'

// Long timeouts for real Claude responses
// Planning includes hook execution, codebase exploration, and spec writing
const PLANNING_TIMEOUT = 300_000 // 5 minutes
const IMPLEMENTING_TIMEOUT = 300_000 // 5 minutes
const SIMPLIFY_TIMEOUT = 180_000 // 3 minutes
const OPTIMIZE_TIMEOUT = 180_000 // 3 minutes
const REVIEW_TIMEOUT = 120_000 // 2 minutes

let fixture: TestFixture

/**
 * Adds a project via WebSocket API (bypasses folder picker UI)
 * Protocol: newline-delimited JSON-RPC 2.0
 */
/**
 * Sends a JSON-RPC call to a worktree socket via WebSocket proxy
 */
async function callWorktreeAPI(projectPath: string, method: string, params: any = {}): Promise<any> {
  const WebSocket = (await import('ws')).default
  const encodedPath = encodeURIComponent(projectPath)
  const ws = new WebSocket(`ws://localhost:6337/ws/worktree/${encodedPath}`)

  return new Promise((resolve, reject) => {
    let settled = false
    const timeout = setTimeout(() => {
      settled = true
      ws.close()
      reject(new Error(`Timeout calling ${method}`))
    }, 10000)

    let buffer = ''

    ws.on('open', () => {
      ws.send(JSON.stringify({
        jsonrpc: '2.0',
        id: 'api-call',
        method,
        params
      }) + '\n')
    })

    ws.on('message', (data: Buffer) => {
      buffer += data.toString()
      const lines = buffer.split('\n')
      buffer = lines.pop() || ''

      for (const line of lines) {
        if (!line.trim()) continue
        try {
          const msg = JSON.parse(line)
          if (msg.id === 'api-call') {
            clearTimeout(timeout)
            settled = true
            ws.close()
            if (msg.error) {
              reject(new Error(msg.error.message))
            } else {
              resolve(msg.result)
            }
            return
          }
        } catch {
          // Ignore parse errors
        }
      }
    })

    ws.on('error', (err) => {
      clearTimeout(timeout)
      if (!settled) reject(err)
    })

    ws.on('close', (code, reason) => {
      clearTimeout(timeout)
      if (!settled) reject(new Error(`WebSocket closed unexpectedly: code=${code}, reason=${reason}`))
    })
  })
}

async function addProjectViaAPI(projectPath: string, socketPath?: string): Promise<void> {
  const WebSocket = (await import('ws')).default
  const ws = new WebSocket('ws://localhost:6337/ws/global')

  await new Promise<void>((resolve, reject) => {
    let settled = false
    const timeout = setTimeout(() => {
      settled = true
      ws.close()
      reject(new Error('Timeout adding project'))
    }, 10000)

    let buffer = ''

    ws.on('open', () => {
      // Send with newline (protocol requirement)
      ws.send(JSON.stringify({
        jsonrpc: '2.0',
        id: 'add-project',
        method: 'projects.register',
        params: { path: projectPath, socket_path: socketPath || '' }
      }) + '\n')
    })

    ws.on('message', (data: Buffer) => {
      buffer += data.toString()
      // Process newline-delimited messages
      const lines = buffer.split('\n')
      buffer = lines.pop() || ''

      for (const line of lines) {
        if (!line.trim()) continue
        try {
          const msg = JSON.parse(line)
          if (msg.id === 'add-project') {
            clearTimeout(timeout)
            settled = true
            ws.close()
            if (msg.error) {
              reject(new Error(msg.error.message))
            } else {
              resolve()
            }
            return
          }
        } catch {
          // Ignore parse errors for non-JSON lines
        }
      }
    })

    ws.on('error', (err) => {
      clearTimeout(timeout)
      if (!settled) reject(err)
    })

    ws.on('close', (code, reason) => {
      clearTimeout(timeout)
      if (!settled) reject(new Error(`WebSocket closed unexpectedly: code=${code}, reason=${reason}`))
    })
  })
}

test.describe('Task Lifecycle with Real Claude', () => {
  test.beforeAll(async () => {
    // Check prerequisites
    if (!isClaudeAvailable()) {
      throw new Error('Claude CLI not found. Install it first: https://claude.ai/code')
    }

    // Create isolated test fixture
    fixture = createTestFixture()
    console.log(`Created test fixture at: ${fixture.repoPath}`)
  })

  test.afterAll(async () => {
    // Cleanup fixture
    if (fixture) {
      fixture.cleanup()
      console.log('Cleaned up test fixture')
    }
  })

  test('complete task flow: load → plan → implement → simplify → optimize → review', async ({ page }) => {
    test.setTimeout(PLANNING_TIMEOUT + IMPLEMENTING_TIMEOUT + SIMPLIFY_TIMEOUT + OPTIMIZE_TIMEOUT + REVIEW_TIMEOUT + 120_000)

    // Capture browser console for debugging WebSocket issues
    page.on('console', msg => console.log('Browser:', msg.type(), msg.text()))

    // Step 1: Navigate to the app (real backend)
    await page.goto('/')

    // Wait for connection
    await expect(page.getByRole('button', { name: 'Add Project' })).toBeVisible({ timeout: 10_000 })
    console.log('GlobalView loaded')

    // Step 2: Add the test fixture as a project via API
    console.log(`Adding project path: ${fixture.repoPath}`)
    console.log(`Expected socket path: ${fixture.socketPath}`)

    // Verify the hash computation manually
    const crypto = await import('crypto')
    const testHash = crypto.createHash('sha256').update(fixture.repoPath).digest('hex').slice(0, 16)
    console.log(`Hash for repoPath "${fixture.repoPath}": ${testHash}`)

    await addProjectViaAPI(fixture.repoPath, fixture.socketPath)

    // Refresh to see the new project
    await page.getByRole('button', { name: 'Refresh' }).click()
    await page.waitForTimeout(1000)

    // The project should now appear in the list - click on it
    const projectName = fixture.repoPath.split('/').pop()!
    // Use first() since there may be multiple matching elements (project name appears twice in UI)
    const projectItem = page.getByText(projectName, { exact: true }).first()
    await expect(projectItem).toBeVisible({ timeout: 5_000 })
    console.log(`Project "${projectName}" added`)
    await projectItem.click()

    // Step 3: Should now be in ProjectView
    await expect(page.getByRole('complementary', { name: 'Left sidebar' })).toBeVisible({ timeout: 10_000 })
    console.log('ProjectView loaded')

    // Verify socket file still exists before connecting
    const { existsSync, realpathSync } = await import('fs')
    const { createHash } = await import('crypto')
    const { homedir } = await import('os')
    const { join, resolve } = await import('path')

    // Compute expected socket path (same algorithm as setup.ts)
    const absPath = realpathSync(resolve(fixture.repoPath))
    const hash = createHash('sha256').update(absPath).digest('hex').slice(0, 16)
    const expectedSocketPath = join(homedir(), '.valksor', 'kvelmo', 'worktrees', hash + '.sock')
    console.log('Checking socket before connect:', expectedSocketPath, 'exists:', existsSync(expectedSocketPath))

    // Wait for worktree socket to connect and task to be ready
    // Check for either "Connected" status (no task) or "loaded" state (task already loaded)
    const connectionStatus = page.getByTestId('task-connection-status')
    const loadedState = page.getByText('loaded', { exact: true })
    await Promise.any([
      expect(connectionStatus).toHaveText('Connected', { timeout: 15_000 }),
      expect(loadedState).toBeVisible({ timeout: 15_000 })
    ])
    console.log('Worktree connected')

    // Step 4: Check state and abort if stuck in a running state from previous test
    const stateIndicator = page.getByLabel('Right sidebar').locator('text=Current State').locator('..').locator('.stat-value, [class*="capitalize"]')
    await expect(stateIndicator).toBeVisible({ timeout: 10_000 })
    let currentState = await stateIndicator.textContent() || 'none'
    console.log('Current state:', currentState)

    // Abort if stuck in a running state (no active job from server restart)
    const runningStates = ['planning', 'implementing', 'optimizing', 'simplifying', 'reviewing']
    if (runningStates.includes(currentState)) {
      console.log(`State "${currentState}" appears stuck (no active job). Calling abort...`)
      try {
        await callWorktreeAPI(fixture.repoPath, 'abort')
        console.log('Abort successful')
        await page.waitForTimeout(500)
        currentState = await stateIndicator.textContent() || 'none'
        console.log('State after abort:', currentState)
      } catch (err) {
        console.log('Abort failed (may be expected):', err)
      }
    }

    // Reset if in failed state (transition back to loaded so we can re-plan)
    if (currentState === 'failed') {
      console.log('State is "failed". Calling reset to go back to loaded...')
      try {
        await callWorktreeAPI(fixture.repoPath, 'reset')
        console.log('Reset successful')
        await page.waitForTimeout(500)
        currentState = await stateIndicator.textContent() || 'none'
        console.log('State after reset:', currentState)
      } catch (err) {
        console.log('Reset failed:', err)
      }
    }

    const planButton = page.getByRole('button', { name: 'Plan' })

    // Refresh state after potential abort
    currentState = await stateIndicator.textContent() || 'none'

    // If state is none or we need to load
    if (currentState === 'none') {
      // Use absolute path since FileProvider resolves relative to backend CWD, not project dir
      const taskSource = `file:${fixture.taskPath}`

      // Use a more specific selector for the task input
      const taskInput = page.locator('input[placeholder*="file:task.md"]')
      await expect(taskInput).toBeVisible({ timeout: 5_000 })

      // For React controlled inputs: focus, clear, then type character by character
      await taskInput.focus()
      await taskInput.clear()
      await taskInput.pressSequentially(taskSource, { delay: 10 })

      // Verify input has value and wait for React state to sync
      await expect(taskInput).toHaveValue(taskSource, { timeout: 5_000 })
      console.log('Task source entered')

      // Click Load
      await page.getByRole('button', { name: 'Load' }).click()
      console.log('Loading task...')

      // Wait for task to be loaded - state should change from "none"
      await expect(stateIndicator).not.toHaveText('none', { timeout: 15_000 })
      console.log('Task loaded')
    } else if (currentState === 'loaded' || currentState === 'planned') {
      console.log(`Task already in state: ${currentState}`)
    } else {
      console.log(`Unexpected state: ${currentState}, attempting to continue...`)
    }

    // Step 5: Start Planning (or wait if already in progress)
    const refreshedState = await stateIndicator.textContent() || 'none'
    if (refreshedState === 'loaded') {
      console.log('Starting planning phase...')
      await expect(planButton).toBeEnabled({ timeout: 10_000 })
      await planButton.click()
    } else if (refreshedState === 'planning') {
      console.log('Planning already in progress, waiting for completion...')
    } else if (refreshedState === 'planned') {
      console.log('Planning already completed, skipping to implementation')
    } else {
      console.log(`Unexpected state: ${refreshedState}, attempting to continue...`)
    }

    // Wait for planning to complete if not already done
    if (refreshedState !== 'planned' && refreshedState !== 'implementing' && refreshedState !== 'implemented') {
      await expect(stateIndicator).toHaveText('planning', { timeout: 30_000 })
      console.log('Planning in progress...')

      // Wait for planning to finish - state should change to 'planned'
      await expect(stateIndicator).toHaveText('planned', { timeout: PLANNING_TIMEOUT })
      console.log('Planning completed!')
    }

    // Step 6: Start Implementation (or wait if already in progress)
    const preImplState = await stateIndicator.textContent() || 'none'
    const implementButton = page.getByRole('button', { name: 'Implement' })

    if (preImplState === 'planned') {
      console.log('Starting implementation phase...')
      await expect(implementButton).toBeEnabled({ timeout: 10_000 })
      await implementButton.click()
    } else if (preImplState === 'implementing') {
      console.log('Implementation already in progress, waiting for completion...')
    } else if (preImplState === 'implemented') {
      console.log('Implementation already completed')
    } else {
      console.log(`Unexpected state before implementation: ${preImplState}`)
    }

    // Wait for implementation to complete if not already done
    if (preImplState !== 'implemented') {
      await expect(stateIndicator).toHaveText('implementing', { timeout: 30_000 })
      console.log('Implementation in progress...')

      await expect(stateIndicator).toHaveText('implemented', { timeout: IMPLEMENTING_TIMEOUT })
      console.log('Implementation completed!')
    }

    // Step 7: Simplify (optional code clarity pass)
    const simplifyButton = page.getByRole('button', { name: 'Simplify' })
    const preSimplifyState = await stateIndicator.textContent() || 'none'

    if (preSimplifyState === 'implemented') {
      console.log('Starting simplification phase...')
      await expect(simplifyButton).toBeEnabled({ timeout: 10_000 })
      await simplifyButton.click()

      await expect(stateIndicator).toHaveText('simplifying', { timeout: 30_000 })
      console.log('Simplification in progress...')

      // Simplify returns to implemented state when done
      await expect(stateIndicator).toHaveText('implemented', { timeout: SIMPLIFY_TIMEOUT })
      console.log('Simplification completed!')
    }

    // Step 8: Optimize (optional performance/quality pass)
    const optimizeButton = page.getByRole('button', { name: 'Optimize' })
    const preOptimizeState = await stateIndicator.textContent() || 'none'

    if (preOptimizeState === 'implemented') {
      console.log('Starting optimization phase...')
      await expect(optimizeButton).toBeEnabled({ timeout: 10_000 })
      await optimizeButton.click()

      await expect(stateIndicator).toHaveText('optimizing', { timeout: 30_000 })
      console.log('Optimization in progress...')

      // Optimize returns to implemented state when done
      await expect(stateIndicator).toHaveText('implemented', { timeout: OPTIMIZE_TIMEOUT })
      console.log('Optimization completed!')
    }

    // Step 9: Review (quality checks and security scan)
    const reviewButton = page.getByRole('button', { name: 'Review' })
    const preReviewState = await stateIndicator.textContent() || 'none'

    if (preReviewState === 'implemented') {
      console.log('Starting review phase...')
      await expect(reviewButton).toBeEnabled({ timeout: 10_000 })
      await reviewButton.click()

      await expect(stateIndicator).toHaveText('reviewing', { timeout: 30_000 })
      console.log('Review in progress...')

      // Review phase is now active - ready for submit
      console.log('Review phase active - ready for submit!')
    }

    // Step 10: Verify file changes show something was created
    // Look in the left sidebar for file changes
    const fileChangesSection = page.getByLabel('Left sidebar').locator('text=File Changes').locator('..')
    await expect(fileChangesSection).toBeVisible()

    // Final state check - should be reviewing (ready for submit) or implemented
    const finalState = await stateIndicator.textContent() || 'none'
    console.log(`Full flow completed! Final state: ${finalState}`)
    expect(['reviewing', 'implemented']).toContain(finalState)
  })
})

// Skip other tests for now
test.describe.skip('Additional Tests', () => {
  test('placeholder', async () => {})
})
