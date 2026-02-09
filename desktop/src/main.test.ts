/**
 * Main process tests.
 *
 * Note: main.ts has module-level side effects (app.whenReady, ipcMain.handle, etc.)
 * that make it challenging to test in isolation. We test the core components
 * (ServerManager, WindowStateStore) directly in their own test files.
 *
 * These tests focus on verifying the mock setup works and key integration points.
 */
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import {
  mockApp,
  mockBrowserWindow,
  mockDialog,
  mockIpcMain,
  mockScreen,
  resetElectronMocks,
} from './test/mocks/electron'
import { createMockChildProcess, mockSpawn, mockSpawnSync } from './test/mocks/child_process'
import { mockExistsSync, mockReadFileSync, mockWriteFileSync, mockMkdirSync } from './test/mocks/fs'

// Store original platform
const originalPlatform = process.platform

// Mock electron
vi.mock('electron', () => ({
  app: mockApp,
  BrowserWindow: mockBrowserWindow,
  dialog: mockDialog,
  ipcMain: mockIpcMain,
  screen: mockScreen,
}))

// Mock child_process
vi.mock('child_process', () => ({
  spawn: mockSpawn,
  spawnSync: mockSpawnSync,
}))

// Mock fs
vi.mock('fs', () => ({
  existsSync: mockExistsSync,
  readFileSync: mockReadFileSync,
  writeFileSync: mockWriteFileSync,
  mkdirSync: mockMkdirSync,
}))

describe('main process mocks', () => {
  beforeEach(() => {
    resetElectronMocks()
    mockSpawn.mockReset()
    mockSpawnSync.mockReset()
    mockExistsSync.mockReset()
    mockExistsSync.mockReturnValue(true)
    mockReadFileSync.mockReset()
    mockWriteFileSync.mockReset()
    mockMkdirSync.mockReset()
    vi.resetModules()

    // Default to Unix platform
    Object.defineProperty(process, 'platform', { value: 'darwin' })
  })

  afterEach(() => {
    Object.defineProperty(process, 'platform', { value: originalPlatform })
  })

  describe('electron mocks', () => {
    it('app.getPath returns mock paths', () => {
      expect(mockApp.getPath('home')).toBe('/mock/home')
      expect(mockApp.getPath('userData')).toBe('/mock/home/.valksor/mehrhof')
    })

    it('app.getVersion returns version', () => {
      expect(mockApp.getVersion()).toBe('0.1.0')
    })

    it('screen.getPrimaryDisplay returns mock display', () => {
      const display = mockScreen.getPrimaryDisplay()
      expect(display).toHaveProperty('id')
      expect(display).toHaveProperty('workArea')
    })

    it('BrowserWindow can be instantiated', () => {
      const window = new mockBrowserWindow({
        width: 800,
        height: 600,
      })
      expect(window).toHaveProperty('loadURL')
      expect(window).toHaveProperty('close')
    })

    it('dialog.showOpenDialog can be called', async () => {
      mockDialog.showOpenDialog.mockResolvedValue({
        canceled: false,
        filePaths: ['/selected/path'],
      })

      const result = await mockDialog.showOpenDialog({}, { properties: ['openDirectory'] })
      expect(result.filePaths[0]).toBe('/selected/path')
    })

    it('dialog.showErrorBox can be called', () => {
      mockDialog.showErrorBox('Title', 'Message')
      expect(mockDialog.showErrorBox).toHaveBeenCalledWith('Title', 'Message')
    })

    it('ipcMain.handle can register handlers', () => {
      const handler = vi.fn()
      mockIpcMain.handle('test-channel', handler)
      expect(mockIpcMain.handle).toHaveBeenCalledWith('test-channel', handler)
    })
  })

  describe('child_process mocks', () => {
    it('spawn returns mock child process', () => {
      const mockProc = createMockChildProcess()
      mockSpawn.mockReturnValue(mockProc)

      const proc = mockSpawn('cmd', ['arg1'])
      expect(proc).toBe(mockProc)
      expect(proc.stdout).toBeDefined()
      expect(proc.stderr).toBeDefined()
    })

    it('spawnSync returns mock result', () => {
      mockSpawnSync.mockReturnValue({
        status: 0,
        stdout: Buffer.from('output'),
        stderr: Buffer.from(''),
      })

      const result = mockSpawnSync('cmd', ['arg1'])
      expect(result.status).toBe(0)
    })
  })

  describe('fs mocks', () => {
    it('existsSync can be configured', () => {
      mockExistsSync.mockReturnValue(false)
      expect(mockExistsSync('/some/path')).toBe(false)

      mockExistsSync.mockReturnValue(true)
      expect(mockExistsSync('/some/path')).toBe(true)
    })

    it('readFileSync returns mock content', () => {
      mockReadFileSync.mockReturnValue('{"key": "value"}')
      const content = mockReadFileSync('/some/file.json')
      expect(content).toBe('{"key": "value"}')
    })

    it('writeFileSync can be called', () => {
      mockWriteFileSync('/some/file.json', '{"key": "value"}')
      expect(mockWriteFileSync).toHaveBeenCalledWith('/some/file.json', '{"key": "value"}')
    })
  })
})

describe('window state store', () => {
  beforeEach(() => {
    resetElectronMocks()
    mockExistsSync.mockReset()
    mockReadFileSync.mockReset()
    mockWriteFileSync.mockReset()
    mockMkdirSync.mockReset()
    vi.resetModules()
  })

  it('loads window state from file', async () => {
    const savedState = {
      x: 100,
      y: 100,
      width: 1200,
      height: 800,
      isMaximized: false,
      displayId: 1,
    }

    mockExistsSync.mockReturnValue(true)
    mockReadFileSync.mockReturnValue(JSON.stringify(savedState))

    const { WindowStateStore } = await import('./window-state')
    const store = new WindowStateStore()
    const state = store.load()

    expect(state).toEqual(savedState)
  })

  it('returns null when no saved state exists', async () => {
    mockExistsSync.mockReturnValue(false)

    const { WindowStateStore } = await import('./window-state')
    const store = new WindowStateStore()
    const state = store.load()

    expect(state).toBeNull()
  })

  it('saves window state to file', async () => {
    mockExistsSync.mockReturnValue(true)

    const { WindowStateStore } = await import('./window-state')
    const store = new WindowStateStore()

    const stateToSave = {
      x: 200,
      y: 200,
      width: 1000,
      height: 600,
      isMaximized: true,
      displayId: 2,
    }

    store.save(stateToSave)

    expect(mockWriteFileSync).toHaveBeenCalled()
    const writtenContent = mockWriteFileSync.mock.calls[0][1] as string
    expect(JSON.parse(writtenContent)).toEqual(stateToSave)
  })

  it('creates directory if it does not exist', async () => {
    mockExistsSync.mockReturnValue(false)

    const { WindowStateStore } = await import('./window-state')
    const store = new WindowStateStore()

    store.save({
      x: 0,
      y: 0,
      width: 800,
      height: 600,
      isMaximized: false,
      displayId: 1,
    })

    expect(mockMkdirSync).toHaveBeenCalledWith(expect.any(String), { recursive: true })
  })

  it('handles corrupted state file gracefully', async () => {
    mockExistsSync.mockReturnValue(true)
    mockReadFileSync.mockReturnValue('not valid json')

    const { WindowStateStore } = await import('./window-state')
    const store = new WindowStateStore()
    const state = store.load()

    expect(state).toBeNull()
  })
})
