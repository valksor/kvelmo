import '@testing-library/jest-dom/vitest'
import { cleanup } from '@testing-library/react'
import { afterEach, vi } from 'vitest'

afterEach(() => {
  cleanup()
  vi.clearAllMocks()
})

// JSDOM cannot compute CSS tabbability — FocusTrap throws when it can't find
// focusable elements. Render children passthrough in tests.
vi.mock('focus-trap-react', () => ({
  FocusTrap: ({ children }: { children: unknown }) => children,
}))
