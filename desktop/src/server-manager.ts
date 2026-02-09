import { spawn, spawnSync, ChildProcess } from 'child_process';
import { app } from 'electron';
import * as path from 'path';
import * as fs from 'fs';

export interface PrerequisiteResult {
  ok: boolean;
  error?: string;
  needsInstall?: boolean;
}

// Check if this is a nightly build (version contains 'nightly' or commit hash)
function isNightlyBuild(): boolean {
  const version = app.getVersion();
  return version.includes('nightly') || version.includes('-') || !version.match(/^\d+\.\d+\.\d+$/);
}

export class ServerManager {
  private process: ChildProcess | null = null;
  private stopping = false;
  private stopped = false;
  private stopPromise: Promise<void> | null = null;

  /**
   * Clean up all listeners from the child process and reset state.
   */
  private cleanup(): void {
    if (this.process) {
      this.process.stdout?.removeAllListeners();
      this.process.stderr?.removeAllListeners();
      this.process.removeAllListeners();
      this.process = null;
    }
  }

  /**
   * Get the path to the bundled mehr binary (macOS/Linux only).
   */
  private getMehrPath(): string {
    if (app.isPackaged) {
      // In packaged app: resources/bin/mehr
      return path.join(process.resourcesPath, 'bin', 'mehr');
    } else {
      // In dev: look in desktop/resources/bin/ or fall back to PATH
      const devPath = path.join(__dirname, '../../resources/bin', 'mehr');
      if (fs.existsSync(devPath)) {
        return devPath;
      }
      return 'mehr'; // Fall back to PATH for dev
    }
  }

  /**
   * Preflight check - call on app startup.
   * - macOS/Linux: Check bundled binary exists
   * - Windows: Check WSL exists, auto-install mehr if missing
   */
  checkPrerequisites(): PrerequisiteResult {
    const isWindows = process.platform === 'win32';

    if (isWindows) {
      return this.checkWindowsPrerequisites();
    } else {
      return this.checkUnixPrerequisites();
    }
  }

  private checkUnixPrerequisites(): PrerequisiteResult {
    const mehrPath = this.getMehrPath();

    // In dev mode with fallback to PATH, check if mehr exists
    if (mehrPath === 'mehr') {
      const check = spawnSync('which', ['mehr'], { stdio: 'pipe' });
      if (check.status !== 0) {
        return {
          ok: false,
          error: 'mehr binary not found.\n\nThis is a development build issue.',
        };
      }
    } else if (!fs.existsSync(mehrPath)) {
      return {
        ok: false,
        error: 'Bundled mehr binary not found.\n\nPlease reinstall the application.',
      };
    }

    return { ok: true };
  }

  private checkWindowsPrerequisites(): PrerequisiteResult {
    // Check WSL availability
    const wslCheck = spawnSync('wsl', ['--version'], { stdio: 'pipe' });
    if (wslCheck.status !== 0) {
      return {
        ok: false,
        error: 'WSL is not installed.\n\nPlease install WSL2:\nhttps://learn.microsoft.com/en-us/windows/wsl/install',
      };
    }

    // Check if mehr is installed in WSL
    const mehrCheck = spawnSync('wsl', ['which', 'mehr'], { stdio: 'pipe' });
    if (mehrCheck.status !== 0) {
      // mehr not found - needs auto-install
      return { ok: true, needsInstall: true };
    }

    return { ok: true };
  }

  /**
   * Auto-install mehr in WSL (Windows only).
   * Uses the install script with --nightly flag if this is a nightly build.
   */
  async installMehrInWSL(): Promise<void> {
    const nightlyFlag = isNightlyBuild() ? ' -s -- --nightly' : '';
    const installCmd = `curl -fsSL https://raw.githubusercontent.com/valksor/go-mehrhof/master/install.sh | bash${nightlyFlag}`;

    return new Promise((resolve, reject) => {
      const proc = spawn('wsl', ['bash', '-c', installCmd], {
        stdio: ['ignore', 'pipe', 'pipe'],
      });

      let stderr = '';

      proc.stderr?.on('data', (data: Buffer) => {
        stderr += data.toString();
      });

      proc.on('exit', (code) => {
        if (code === 0) {
          resolve();
        } else {
          reject(new Error(`Failed to install mehr in WSL:\n${stderr}`));
        }
      });

      proc.on('error', (err) => {
        reject(new Error(`Failed to run install script: ${err.message}`));
      });
    });
  }

  /**
   * Start mehr serve in global mode.
   * Shows project picker UI.
   * @returns The port number mehr is listening on.
   */
  async startGlobal(): Promise<number> {
    // Stop any existing process first
    await this.stop();
    this.stopped = false; // Reset for new server instance

    const isWindows = process.platform === 'win32';

    let cmd: string;
    let args: string[];

    if (isWindows) {
      cmd = 'wsl';
      args = ['mehr', 'serve', '--global', '--port', '0'];
    } else {
      cmd = this.getMehrPath();
      args = ['serve', '--global', '--port', '0'];
    }

    return new Promise((resolve, reject) => {
      this.process = spawn(cmd, args, { shell: false });
      let resolved = false;

      const timeout = setTimeout(() => {
        if (!resolved) {
          resolved = true;
          removeListeners();
          reject(new Error('Server startup timed out after 30 seconds.'));
        }
      }, 30000);

      // Define named listeners for proper cleanup
      const onStdout = (data: Buffer) => {
        const match = data.toString().match(/localhost:(\d+)/);
        if (match && !resolved) {
          resolved = true;
          clearTimeout(timeout);
          // Keep stderr listener for logging, remove startup listeners
          this.process?.off('error', onError);
          this.process?.off('exit', onExit);
          this.process?.stdout?.off('data', onStdout);
          resolve(parseInt(match[1], 10));
        }
      };

      const onStderr = (data: Buffer) => {
        console.error('[mehr stderr]', data.toString());
      };

      const onError = (err: Error) => {
        if (!resolved) {
          resolved = true;
          clearTimeout(timeout);
          removeListeners();
          this.cleanup();
          reject(new Error(`Failed to start mehr: ${err.message}`));
        }
      };

      const onExit = (code: number | null) => {
        if (!resolved && code !== 0 && code !== null) {
          resolved = true;
          clearTimeout(timeout);
          removeListeners();
          this.cleanup();
          reject(new Error(`mehr exited with code ${code}`));
        }
      };

      const removeListeners = () => {
        this.process?.stdout?.off('data', onStdout);
        this.process?.stderr?.off('data', onStderr);
        this.process?.off('error', onError);
        this.process?.off('exit', onExit);
      };

      this.process.stdout?.on('data', onStdout);
      this.process.stderr?.on('data', onStderr);
      this.process.on('error', onError);
      this.process.on('exit', onExit);
    });
  }

  /**
   * Start mehr serve for a project.
   * Uses random port (--port 0) to avoid conflicts.
   * @returns The port number mehr is listening on.
   */
  async start(projectPath: string): Promise<number> {
    // Stop any existing process first
    await this.stop();
    this.stopped = false; // Reset for new server instance

    const isWindows = process.platform === 'win32';

    // Validate path
    const validation = this.validatePath(projectPath);
    if (!validation.ok) {
      throw new Error(validation.error);
    }

    let cmd: string;
    let args: string[];
    let cwd: string | undefined;

    if (isWindows) {
      // Convert Windows path → WSL path, use cd to set working directory
      const wslPath = this.toWslPath(projectPath);
      cmd = 'wsl';
      args = ['bash', '-c', `cd '${wslPath}' && mehr serve --port 0`];
      cwd = undefined;
    } else {
      cmd = this.getMehrPath();
      args = ['serve', '--port', '0'];
      cwd = projectPath;
    }

    return new Promise((resolve, reject) => {
      this.process = spawn(cmd, args, { cwd, shell: false });
      let resolved = false;

      const timeout = setTimeout(() => {
        if (!resolved) {
          resolved = true;
          removeListeners();
          reject(
            new Error(
              'Server startup timed out after 30 seconds.\n\nCheck that mehr is working: mehr --version'
            )
          );
        }
      }, 30000);

      // Define named listeners for proper cleanup
      const onStdout = (data: Buffer) => {
        const match = data.toString().match(/localhost:(\d+)/);
        if (match && !resolved) {
          resolved = true;
          clearTimeout(timeout);
          // Keep stderr listener for logging, remove startup listeners
          this.process?.off('error', onError);
          this.process?.off('exit', onExit);
          this.process?.stdout?.off('data', onStdout);
          resolve(parseInt(match[1], 10));
        }
      };

      const onStderr = (data: Buffer) => {
        console.error('[mehr stderr]', data.toString());
      };

      const onError = (err: Error) => {
        if (!resolved) {
          resolved = true;
          clearTimeout(timeout);
          removeListeners();
          this.cleanup();
          reject(new Error(`Failed to start mehr: ${err.message}`));
        }
      };

      const onExit = (code: number | null) => {
        if (!resolved && code !== 0 && code !== null) {
          resolved = true;
          clearTimeout(timeout);
          removeListeners();
          this.cleanup();
          reject(new Error(`mehr exited with code ${code}`));
        }
      };

      const removeListeners = () => {
        this.process?.stdout?.off('data', onStdout);
        this.process?.stderr?.off('data', onStderr);
        this.process?.off('error', onError);
        this.process?.off('exit', onExit);
      };

      this.process.stdout?.on('data', onStdout);
      this.process.stderr?.on('data', onStderr);
      this.process.on('error', onError);
      this.process.on('exit', onExit);
    });
  }

  /**
   * Validate a project path.
   * Rejects UNC paths on Windows.
   */
  private validatePath(winPath: string): PrerequisiteResult {
    if (process.platform === 'win32') {
      // Reject network paths (UNC)
      if (winPath.startsWith('\\\\')) {
        return {
          ok: false,
          error: 'Network paths (\\\\server\\share) are not supported.\n\nPlease use a local drive.',
        };
      }
      // Validate drive letter format
      if (!/^[A-Za-z]:\\/.test(winPath)) {
        return {
          ok: false,
          error: 'Invalid path format.\n\nExpected: C:\\path\\to\\project',
        };
      }
    }
    return { ok: true };
  }

  /**
   * Convert Windows path to WSL path.
   * C:\Users\foo -> /mnt/c/Users/foo
   */
  private toWslPath(winPath: string): string {
    return winPath
      .replace(/^([A-Za-z]):\\/, (_, drive: string) => `/mnt/${drive.toLowerCase()}/`)
      .replace(/\\/g, '/');
  }

  /**
   * Graceful shutdown: SIGTERM -> wait 5s -> SIGKILL.
   * Safe to call multiple times - subsequent calls wait for the first to complete.
   */
  async stop(): Promise<void> {
    // Already stopped
    if (this.stopped || !this.process) return;

    // If already stopping, wait for that operation to complete
    if (this.stopping && this.stopPromise) {
      return this.stopPromise;
    }

    this.stopping = true;
    const proc = this.process;

    this.stopPromise = new Promise((resolve) => {
      const forceKill = setTimeout(() => {
        proc.kill('SIGKILL');
        this.cleanup();
        this.stopping = false;
        this.stopped = true;
        this.stopPromise = null;
        resolve();
      }, 5000);

      // Use once() to auto-remove listener after first call
      proc.once('exit', () => {
        clearTimeout(forceKill);
        this.cleanup();
        this.stopping = false;
        this.stopped = true;
        this.stopPromise = null;
        resolve();
      });

      proc.kill('SIGTERM');
    });

    return this.stopPromise;
  }
}
