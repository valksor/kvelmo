import { app, BrowserWindow, dialog, ipcMain, screen } from 'electron';
import * as path from 'path';
import { ServerManager } from './server-manager.js';
import { WindowStateStore } from './window-state.js';

let mainWindow: BrowserWindow | null = null;
const serverManager = new ServerManager();
const windowState = new WindowStateStore();

// Shutdown coordination
let isQuitting = false;

/**
 * Wrap an async operation with a timeout. If the operation doesn't complete
 * within the timeout, the onTimeout callback is called and the promise resolves.
 */
async function withAppTimeout<T>(
  operation: Promise<T>,
  timeoutMs: number,
  onTimeout: () => void
): Promise<T | void> {
  return Promise.race([
    operation,
    new Promise<void>((resolve) => {
      setTimeout(() => {
        onTimeout();
        resolve();
      }, timeoutMs);
    }),
  ]);
}

app.whenReady().then(async () => {
  // Check prerequisites
  const prereq = serverManager.checkPrerequisites();
  if (!prereq.ok) {
    dialog.showErrorBox('Setup Required', prereq.error!);
    app.quit();
    return;
  }

  // Windows: Auto-install mehr in WSL if needed
  if (prereq.needsInstall) {
    try {
      // Show progress dialog
      const progressWindow = new BrowserWindow({
        width: 400,
        height: 150,
        frame: false,
        resizable: false,
        alwaysOnTop: true,
        webPreferences: { nodeIntegration: false, contextIsolation: true },
      });
      progressWindow.loadURL(
        `data:text/html,<html><body style="font-family:system-ui;display:flex;flex-direction:column;align-items:center;justify-content:center;height:100vh;margin:0;background:#1e1e2e;color:#cdd6f4"><div style="margin-bottom:12px">Installing mehr in WSL...</div><div style="width:80%;height:4px;background:#313244;border-radius:2px;overflow:hidden"><div style="width:100%;height:100%;background:#89b4fa;animation:progress 2s ease-in-out infinite"></div></div><style>@keyframes progress{0%,100%{transform:translateX(-100%)}50%{transform:translateX(100%)}}</style></body></html>`
      );

      await serverManager.installMehrInWSL();
      progressWindow.close();
    } catch (err) {
      dialog.showErrorBox('Installation Failed', String(err));
      app.quit();
      return;
    }
  }

  // Start server in global mode
  try {
    const port = await serverManager.startGlobal();
    createWindow(port);
  } catch (err) {
    dialog.showErrorBox('Failed to Start', String(err));
    app.quit();
  }
});

function createWindow(port: number): void {
  const state = windowState.load();
  const displays = screen.getAllDisplays();

  // Find saved display or fall back to primary
  const targetDisplay = state
    ? displays.find((d) => d.id === state.displayId) || screen.getPrimaryDisplay()
    : screen.getPrimaryDisplay();

  // Restore bounds if display still exists, otherwise use display workArea
  const displayStillExists = state && displays.some((d) => d.id === state.displayId);
  const bounds = displayStillExists
    ? { x: state.x, y: state.y, width: state.width, height: state.height }
    : targetDisplay.workArea;

  mainWindow = new BrowserWindow({
    ...bounds,
    show: false, // Prevent flash while positioning
    title: 'Mehrhof',
    webPreferences: {
      preload: path.join(__dirname, 'preload.js'),
      contextIsolation: true,
      nodeIntegration: false,
    },
  });

  // Show window after ready (maximize on first launch or if was maximized)
  mainWindow.once('ready-to-show', () => {
    if (!state || state.isMaximized) {
      mainWindow!.maximize();
    }
    mainWindow!.show();
  });

  // Save window state on close
  mainWindow.on('close', () => {
    if (!mainWindow) return;
    const currentBounds = mainWindow.getBounds();
    const currentDisplay = screen.getDisplayMatching(currentBounds);
    windowState.save({
      ...currentBounds,
      isMaximized: mainWindow.isMaximized(),
      displayId: currentDisplay.id,
    });
  });

  // Load the live server (global mode shows project picker)
  mainWindow.loadURL(`http://localhost:${port}`);
}

// IPC: Native folder picker for adding new projects
ipcMain.handle('open-folder', async () => {
  const result = await dialog.showOpenDialog(mainWindow!, {
    properties: ['openDirectory'],
    title: 'Select Project Folder',
  });
  return result.canceled ? null : result.filePaths[0];
});

// Graceful shutdown with timeout protection
app.on('before-quit', async (e) => {
  if (isQuitting) return; // Already shutting down
  isQuitting = true;
  e.preventDefault();

  await withAppTimeout(
    serverManager.stop(),
    3000, // 3 second timeout
    () => console.log('[shutdown] Server stop timed out, forcing quit')
  );

  // Use app.exit() to terminate immediately without re-triggering before-quit
  app.exit(0);
});

app.on('window-all-closed', async () => {
  if (isQuitting) return; // Already handled by before-quit
  isQuitting = true;

  await withAppTimeout(
    serverManager.stop(),
    3000,
    () => console.log('[shutdown] Server stop timed out')
  );

  if (process.platform !== 'darwin') {
    app.exit(0);
  }
});

app.on('activate', () => {
  if (BrowserWindow.getAllWindows().length === 0 && mainWindow === null) {
    // Re-launch would need to restart server - for now just quit
    app.quit();
  }
});
