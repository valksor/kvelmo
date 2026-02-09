import { afterEach, vi } from 'vitest'

// Clean up mocks after each test
afterEach(() => {
  vi.clearAllMocks()
  vi.restoreAllMocks()
})

// Reset modules to ensure clean state between tests
afterEach(() => {
  vi.resetModules()
})
