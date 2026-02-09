import { vi } from 'vitest'

// Mock electron app module
export const mockApp = {
  getVersion: vi.fn(() => '0.1.0'),
  getPath: vi.fn((name: string) => {
    if (name === 'home') return '/mock/home'
    if (name === 'userData') return '/mock/home/.valksor/mehrhof'
    return '/mock/' + name
  }),
  isPackaged: false,
  whenReady: vi.fn(() => Promise.resolve()),
  quit: vi.fn(),
  exit: vi.fn(),
  on: vi.fn(),
}

// Mock screen module
export const mockScreen = {
  getAllDisplays: vi.fn(() => [{ id: 1, workArea: { x: 0, y: 0, width: 1920, height: 1080 } }]),
  getPrimaryDisplay: vi.fn(() => ({ id: 1, workArea: { x: 0, y: 0, width: 1920, height: 1080 } })),
  getDisplayMatching: vi.fn(() => ({ id: 1 })),
}

// Mock BrowserWindow - use a class that can be instantiated with `new`
export class MockBrowserWindowClass {
  loadURL = vi.fn()
  close = vi.fn()
  show = vi.fn()
  maximize = vi.fn()
  isMaximized = vi.fn(() => false)
  getBounds = vi.fn(() => ({ x: 0, y: 0, width: 1200, height: 800 }))
  on = vi.fn()
  once = vi.fn()
  webContents = { send: vi.fn() }

  static getAllWindows = vi.fn(() => [])
}

// Export the class directly - it can be used with `new`
export const mockBrowserWindow = MockBrowserWindowClass

// Mock dialog
export const mockDialog = {
  showErrorBox: vi.fn(),
  showOpenDialog: vi.fn(() => Promise.resolve({ canceled: true, filePaths: [] })),
}

// Mock ipcMain
export const mockIpcMain = {
  handle: vi.fn(),
  on: vi.fn(),
  removeHandler: vi.fn(),
}

// Mock ipcRenderer
export const mockIpcRenderer = {
  invoke: vi.fn(),
  on: vi.fn(),
  send: vi.fn(),
}

// Mock contextBridge
export const mockContextBridge = {
  exposeInMainWorld: vi.fn(),
}

// Combined electron mock for vi.mock()
export const electronMock = {
  app: mockApp,
  BrowserWindow: mockBrowserWindow,
  dialog: mockDialog,
  ipcMain: mockIpcMain,
  ipcRenderer: mockIpcRenderer,
  contextBridge: mockContextBridge,
  screen: mockScreen,
}

// Helper to reset all electron mocks
export function resetElectronMocks(): void {
  mockApp.getVersion.mockReturnValue('0.1.0')
  mockApp.getPath.mockImplementation((name: string) => {
    if (name === 'home') return '/mock/home'
    if (name === 'userData') return '/mock/home/.valksor/mehrhof'
    return '/mock/' + name
  })
  mockApp.isPackaged = false
  mockApp.whenReady.mockClear()
  mockApp.quit.mockClear()
  mockApp.exit.mockClear()
  mockApp.on.mockClear()

  MockBrowserWindowClass.getAllWindows.mockClear()
  MockBrowserWindowClass.getAllWindows.mockReturnValue([])

  mockDialog.showErrorBox.mockClear()
  mockDialog.showOpenDialog.mockClear()

  mockIpcMain.handle.mockClear()
  mockIpcMain.on.mockClear()

  mockIpcRenderer.invoke.mockClear()

  mockContextBridge.exposeInMainWorld.mockClear()

  mockScreen.getAllDisplays.mockClear()
  mockScreen.getPrimaryDisplay.mockClear()
  mockScreen.getDisplayMatching.mockClear()
}
