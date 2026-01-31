import * as vscode from 'vscode';
import { spawn, ChildProcess } from 'child_process';
import * as path from 'path';
import * as fs from 'fs';
import * as os from 'os';

export interface ServerManagerEvents {
  started: (port: number) => void;
  stopped: () => void;
  error: (error: Error) => void;
  output: (line: string) => void;
}

export class ServerManager {
  private process: ChildProcess | null = null;
  private _port: number | null = null;
  private outputChannel: vscode.OutputChannel;
  private startPromise: Promise<number> | null = null;
  private readonly listeners: Map<keyof ServerManagerEvents, Set<(...args: unknown[]) => void>> =
    new Map();

  constructor(outputChannel: vscode.OutputChannel) {
    this.outputChannel = outputChannel;
  }

  get port(): number | null {
    return this._port;
  }

  isRunning(): boolean {
    return this.process !== null && this._port !== null;
  }

  async start(workspacePath: string): Promise<number> {
    if (this.startPromise) {
      return this.startPromise;
    }

    if (this.isRunning()) {
      return this._port!;
    }

    this.startPromise = this.doStart(workspacePath);
    try {
      const port = await this.startPromise;
      return port;
    } finally {
      this.startPromise = null;
    }
  }

  private async doStart(workspacePath: string): Promise<number> {
    const executable = await this.findExecutable();
    if (!executable) {
      throw new Error(
        'Could not find mehr executable. Please install it or configure mehrhof.mehrExecutable'
      );
    }

    return new Promise<number>((resolve, reject) => {
      this.outputChannel.appendLine(`Starting server with: ${executable} serve`);
      this.outputChannel.appendLine(`Working directory: ${workspacePath}`);

      const env = { ...process.env };
      // Ensure PATH includes common binary locations
      const additionalPaths = [
        path.join(os.homedir(), '.local', 'bin'),
        path.join(os.homedir(), 'bin'),
        '/usr/local/bin',
        '/opt/homebrew/bin',
      ];
      env.PATH = [...additionalPaths, env.PATH].join(path.delimiter);

      // Using spawn with array arguments - safe from shell injection
      this.process = spawn(executable, ['serve'], {
        cwd: workspacePath,
        env,
        stdio: ['ignore', 'pipe', 'pipe'],
      });

      let portFound = false;
      const portRegex = /Server running at: http:\/\/localhost:(\d+)/;

      const handleOutput = (data: Buffer) => {
        const lines = data.toString().split('\n');
        for (const line of lines) {
          if (line.trim()) {
            this.outputChannel.appendLine(`[server] ${line}`);
            this.emit('output', line);

            if (!portFound) {
              const match = portRegex.exec(line);
              if (match) {
                portFound = true;
                this._port = parseInt(match[1], 10);
                this.outputChannel.appendLine(`Server started on port ${this._port}`);
                this.emit('started', this._port);
                resolve(this._port);
              }
            }
          }
        }
      };

      this.process.stdout?.on('data', handleOutput);
      this.process.stderr?.on('data', handleOutput);

      this.process.on('error', (error) => {
        this.outputChannel.appendLine(`Server error: ${error.message}`);
        this.emit('error', error);
        if (!portFound) {
          reject(error);
        }
      });

      this.process.on('exit', (code, signal) => {
        this.outputChannel.appendLine(`Server exited with code ${code}, signal ${signal}`);
        this._port = null;
        this.process = null;
        this.emit('stopped');
        if (!portFound) {
          reject(new Error(`Server exited before starting (code: ${code})`));
        }
      });

      // Timeout if port not found within 30 seconds
      setTimeout(() => {
        if (!portFound) {
          this.stop();
          reject(new Error('Server startup timed out'));
        }
      }, 30000);
    });
  }

  stop(): void {
    if (this.process) {
      this.outputChannel.appendLine('Stopping server...');
      this.process.kill('SIGTERM');

      // Force kill after 5 seconds if still running
      setTimeout(() => {
        if (this.process) {
          this.process.kill('SIGKILL');
        }
      }, 5000);
    }
  }

  private async findExecutable(): Promise<string | null> {
    // Check user configuration first
    const config = vscode.workspace.getConfiguration('mehrhof');
    const configuredPath = config.get<string>('mehrExecutable');
    if (configuredPath && (await this.isExecutable(configuredPath))) {
      return configuredPath;
    }

    // Check common locations
    const searchPaths = [
      path.join(os.homedir(), '.local', 'bin', 'mehr'),
      path.join(os.homedir(), 'bin', 'mehr'),
      path.join(os.homedir(), 'go', 'bin', 'mehr'),
      '/usr/local/bin/mehr',
      '/opt/homebrew/bin/mehr',
    ];

    for (const searchPath of searchPaths) {
      if (await this.isExecutable(searchPath)) {
        return searchPath;
      }
    }

    // Try to find in PATH
    const pathEnv = process.env.PATH ?? '';
    const pathDirs = pathEnv.split(path.delimiter);
    for (const dir of pathDirs) {
      const candidate = path.join(dir, 'mehr');
      if (await this.isExecutable(candidate)) {
        return candidate;
      }
    }

    return null;
  }

  private async isExecutable(filePath: string): Promise<boolean> {
    try {
      await fs.promises.access(filePath, fs.constants.X_OK);
      return true;
    } catch {
      return false;
    }
  }

  // Event emitter methods
  on<K extends keyof ServerManagerEvents>(event: K, listener: ServerManagerEvents[K]): this {
    if (!this.listeners.has(event)) {
      this.listeners.set(event, new Set());
    }
    this.listeners.get(event)!.add(listener as (...args: unknown[]) => void);
    return this;
  }

  off<K extends keyof ServerManagerEvents>(event: K, listener: ServerManagerEvents[K]): this {
    this.listeners.get(event)?.delete(listener as (...args: unknown[]) => void);
    return this;
  }

  private emit<K extends keyof ServerManagerEvents>(
    event: K,
    ...args: Parameters<ServerManagerEvents[K]>
  ): void {
    const listeners = this.listeners.get(event);
    if (listeners) {
      for (const listener of listeners) {
        listener(...args);
      }
    }
  }

  dispose(): void {
    this.stop();
    this.listeners.clear();
  }
}
