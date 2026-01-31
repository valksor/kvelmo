/**
 * Mock implementations for child_process module used in ServerManager tests.
 * Provides controllable process spawning for testing server lifecycle.
 */

import { EventEmitter } from 'events';
import type { SpawnOptions } from 'child_process';

/**
 * Mock stdout/stderr stream that can emit data events.
 */
export class MockReadable extends EventEmitter {
  setEncoding(_encoding: string): this {
    return this;
  }

  pipe<T extends NodeJS.WritableStream>(destination: T): T {
    return destination;
  }
}

/**
 * Mock ChildProcess that provides full control over process behavior.
 * Extends EventEmitter to support 'error', 'exit', 'close' events.
 */
export class MockChildProcess extends EventEmitter {
  stdin = null;
  stdout: MockReadable;
  stderr: MockReadable;
  killed = false;
  pid = Math.floor(Math.random() * 10000) + 1000;
  connected = true;
  exitCode: number | null = null;
  signalCode: NodeJS.Signals | null = null;
  spawnargs: string[] = [];
  spawnfile = '';

  private _killSignals: string[] = [];

  constructor() {
    super();
    this.stdout = new MockReadable();
    this.stderr = new MockReadable();
  }

  /**
   * Simulate killing the process. Records the signal used.
   */
  kill(signal?: NodeJS.Signals | number): boolean {
    this._killSignals.push(String(signal ?? 'SIGTERM'));
    this.killed = true;
    return true;
  }

  /**
   * Get all signals that were sent to this process.
   */
  getKillSignals(): string[] {
    return [...this._killSignals];
  }

  /**
   * Simulate stdout data output from the process.
   */
  simulateStdout(data: string): void {
    this.stdout.emit('data', Buffer.from(data));
  }

  /**
   * Simulate stderr data output from the process.
   */
  simulateStderr(data: string): void {
    this.stderr.emit('data', Buffer.from(data));
  }

  /**
   * Simulate process exit.
   */
  simulateExit(code: number | null, signal: NodeJS.Signals | null = null): void {
    this.exitCode = code;
    this.signalCode = signal;
    this.emit('exit', code, signal);
    this.emit('close', code, signal);
  }

  /**
   * Simulate process error.
   */
  simulateError(error: Error): void {
    this.emit('error', error);
  }

  // Additional ChildProcess interface methods (stubs)
  ref(): void {}
  unref(): void {}
  disconnect(): void {
    this.connected = false;
  }
  send(): boolean {
    return false;
  }

  [Symbol.dispose](): void {
    this.kill();
  }
}

/**
 * Options for creating a mock spawn function.
 */
export interface MockSpawnOptions {
  /** Delay before process starts emitting output (ms) */
  startDelay?: number;
  /** Auto-emit server startup message with port */
  autoStartPort?: number;
  /** Auto-emit error after delay */
  autoError?: { error: Error; delay: number };
  /** Auto-exit after delay */
  autoExit?: { code: number; delay: number };
}

/**
 * Spawned process record for verification.
 */
export interface SpawnRecord {
  command: string;
  args: string[];
  options?: SpawnOptions;
  process: MockChildProcess;
}

/**
 * Create a mock spawn function that returns controllable MockChildProcess instances.
 */
export function createMockSpawn(options: MockSpawnOptions = {}): {
  spawn: (command: string, args?: string[], spawnOptions?: SpawnOptions) => MockChildProcess;
  getSpawnedProcesses: () => SpawnRecord[];
  getLastProcess: () => MockChildProcess | undefined;
  reset: () => void;
} {
  const spawnedProcesses: SpawnRecord[] = [];

  const spawn = (
    command: string,
    args: string[] = [],
    spawnOptions?: SpawnOptions
  ): MockChildProcess => {
    const process = new MockChildProcess();
    process.spawnfile = command;
    process.spawnargs = [command, ...args];

    spawnedProcesses.push({
      command,
      args,
      options: spawnOptions,
      process,
    });

    // Handle auto-start with port
    if (options.autoStartPort !== undefined) {
      const delay = options.startDelay ?? 10;
      setTimeout(() => {
        process.simulateStdout(`Server running at: http://localhost:${options.autoStartPort}\n`);
      }, delay);
    }

    // Handle auto-error
    if (options.autoError) {
      setTimeout(() => {
        process.simulateError(options.autoError!.error);
      }, options.autoError.delay);
    }

    // Handle auto-exit
    if (options.autoExit) {
      setTimeout(() => {
        process.simulateExit(options.autoExit!.code);
      }, options.autoExit.delay);
    }

    return process;
  };

  return {
    spawn,
    getSpawnedProcesses: () => [...spawnedProcesses],
    getLastProcess: () => spawnedProcesses[spawnedProcesses.length - 1]?.process,
    reset: () => {
      spawnedProcesses.length = 0;
    },
  };
}

/**
 * Create a spawn function that fails to find executable.
 */
export function createFailingSpawn(errorMessage: string = 'spawn ENOENT'): {
  spawn: () => MockChildProcess;
} {
  return {
    spawn: (): MockChildProcess => {
      const process = new MockChildProcess();
      // Emit error on next tick to simulate spawn failure
      setImmediate(() => {
        const error = new Error(errorMessage) as NodeJS.ErrnoException;
        error.code = 'ENOENT';
        process.simulateError(error);
      });
      return process;
    },
  };
}

/**
 * Mock fs.promises.access for testing executable detection.
 */
export function createMockFsAccess(
  executablePaths: string[]
): (path: string, mode?: number) => Promise<void> {
  return (path: string, _mode?: number): Promise<void> => {
    if (executablePaths.includes(path)) {
      return Promise.resolve();
    }
    const error = new Error(`EACCES: permission denied, access '${path}'`) as NodeJS.ErrnoException;
    error.code = 'EACCES';
    return Promise.reject(error);
  };
}

/**
 * Mock os.homedir for testing path resolution.
 */
export function createMockHomedir(homeDir: string = '/home/testuser'): () => string {
  return () => homeDir;
}
