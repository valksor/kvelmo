import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mockContextBridge, mockIpcRenderer, resetElectronMocks } from './test/mocks/electron'

// Mock electron before importing preload
vi.mock('electron', () => ({
  contextBridge: mockContextBridge,
  ipcRenderer: mockIpcRenderer,
}))

describe('preload', () => {
  beforeEach(() => {
    resetElectronMocks()
    vi.resetModules()
  })

  it('exposes electron API to renderer via contextBridge', async () => {
    await import('./preload')

    expect(mockContextBridge.exposeInMainWorld).toHaveBeenCalledTimes(1)
    expect(mockContextBridge.exposeInMainWorld).toHaveBeenCalledWith(
      'electron',
      expect.objectContaining({
        openFolder: expect.any(Function),
      })
    )
  })

  it('openFolder calls ipcRenderer.invoke with open-folder channel', async () => {
    mockIpcRenderer.invoke.mockResolvedValue('/path/to/folder')

    await import('./preload')

    // Get the exposed API
    const exposedApi = mockContextBridge.exposeInMainWorld.mock.calls[0][1] as {
      openFolder: () => Promise<string | null>
    }

    const result = await exposedApi.openFolder()

    expect(mockIpcRenderer.invoke).toHaveBeenCalledWith('open-folder')
    expect(result).toBe('/path/to/folder')
  })

  it('openFolder returns null when dialog is cancelled', async () => {
    mockIpcRenderer.invoke.mockResolvedValue(null)

    await import('./preload')

    const exposedApi = mockContextBridge.exposeInMainWorld.mock.calls[0][1] as {
      openFolder: () => Promise<string | null>
    }

    const result = await exposedApi.openFolder()

    expect(result).toBeNull()
  })
})
