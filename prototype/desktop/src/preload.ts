import { contextBridge, ipcRenderer } from 'electron';

/**
 * Preload script exposes IPC methods to the renderer process.
 * These methods are available as window.electron in the web UI.
 */
contextBridge.exposeInMainWorld('electron', {
  /**
   * Open native folder picker dialog.
   * @returns The selected folder path, or null if cancelled.
   */
  openFolder: (): Promise<string | null> => ipcRenderer.invoke('open-folder'),
});
