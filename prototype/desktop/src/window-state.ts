import { app } from 'electron';
import * as fs from 'fs';
import * as path from 'path';

export interface WindowState {
  x: number;
  y: number;
  width: number;
  height: number;
  isMaximized: boolean;
  displayId: number;
}

/**
 * WindowStateStore persists window bounds and display across sessions.
 * Stores at: ~/.valksor/mehrhof/window-state.json
 */
export class WindowStateStore {
  private storePath: string;

  constructor() {
    const homeDir = app.getPath('home');
    this.storePath = path.join(homeDir, '.valksor', 'mehrhof', 'window-state.json');
  }

  load(): WindowState | null {
    try {
      if (fs.existsSync(this.storePath)) {
        const content = fs.readFileSync(this.storePath, 'utf-8');
        return JSON.parse(content) as WindowState;
      }
    } catch {
      // First run or corrupted file
    }
    return null;
  }

  save(state: WindowState): void {
    try {
      const dir = path.dirname(this.storePath);
      if (!fs.existsSync(dir)) {
        fs.mkdirSync(dir, { recursive: true });
      }
      fs.writeFileSync(this.storePath, JSON.stringify(state, null, 2));
    } catch (error) {
      console.error('Failed to save window state:', error);
    }
  }
}
