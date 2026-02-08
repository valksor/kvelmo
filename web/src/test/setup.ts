import '@testing-library/jest-dom/vitest'
import { cleanup } from '@testing-library/react'
import { afterEach, vi } from 'vitest'

// Clean up after each test
afterEach(() => {
  cleanup()
  vi.clearAllMocks()
})

// Mock window.matchMedia for theme tests
Object.defineProperty(window, 'matchMedia', {
  writable: true,
  value: vi.fn().mockImplementation((query: string) => ({
    matches: false,
    media: query,
    onchange: null,
    addListener: vi.fn(),
    removeListener: vi.fn(),
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
    dispatchEvent: vi.fn(),
  })),
})

// Mock localStorage
const localStorageMock = {
  getItem: vi.fn(),
  setItem: vi.fn(),
  removeItem: vi.fn(),
  clear: vi.fn(),
  length: 0,
  key: vi.fn(),
}
Object.defineProperty(window, 'localStorage', { value: localStorageMock })

// Mock fetch for API tests (can be overridden in individual tests)
global.fetch = vi.fn()

// Mock focus-trap-react: JSDOM cannot compute CSS tabbability, so FocusTrap
// throws when it can't find focusable elements. Render children passthrough.
vi.mock('focus-trap-react', () => ({
  FocusTrap: ({ children }: { children: unknown }) => children,
}))

// Mock react-i18next for tests - returns key as translation value
vi.mock('react-i18next', () => ({
  useTranslation: () => ({
    t: (key: string) => key,
    i18n: {
      changeLanguage: vi.fn(),
      language: 'en',
    },
  }),
  Trans: ({ children }: { children: unknown }) => children,
  initReactI18next: { type: '3rdParty', init: vi.fn() },
  I18nextProvider: ({ children }: { children: unknown }) => children,
}))
