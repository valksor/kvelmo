import { vi } from 'vitest'

// Mock fs functions
export const mockExistsSync = vi.fn(() => true)
export const mockReadFileSync = vi.fn(() => '{}')
export const mockWriteFileSync = vi.fn()
export const mockMkdirSync = vi.fn()

// Combined fs mock for vi.mock()
export const fsMock = {
  existsSync: mockExistsSync,
  readFileSync: mockReadFileSync,
  writeFileSync: mockWriteFileSync,
  mkdirSync: mockMkdirSync,
}
