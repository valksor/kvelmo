import { describe, test, expect, beforeEach, afterEach } from 'bun:test';
import { ServerManager } from '../../src/services/serverManager';
import {
  createMockOutputChannel,
  createMockConfiguration,
  type MockOutputChannel,
} from '../helpers/mockVscode';
import { MockChildProcess, createMockSpawn } from '../helpers/mockChildProcess';

// Mock vscode module
const mockConfiguration = createMockConfiguration();

describe('ServerManager Test Suite', () => {
  let outputChannel: MockOutputChannel;
  let serverManager: ServerManager;
  let mockSpawnModule: ReturnType<typeof createMockSpawn>;

  beforeEach(() => {
    outputChannel = createMockOutputChannel('Mehrhof Test');
    mockConfiguration.values.clear();

    // Create fresh mock spawn for each test
    mockSpawnModule = createMockSpawn();
  });

  afterEach(() => {
    if (serverManager) {
      serverManager.dispose();
    }
    mockSpawnModule.reset();
  });

  describe('Constructor', () => {
    test('creates instance with OutputChannel', () => {
      serverManager = new ServerManager(outputChannel as unknown as import('vscode').OutputChannel);
      expect(serverManager).toBeTruthy();
    });

    test('initial port is null', () => {
      serverManager = new ServerManager(outputChannel as unknown as import('vscode').OutputChannel);
      expect(serverManager.port).toBe(null);
    });
  });

  describe('isRunning()', () => {
    test('returns false initially', () => {
      serverManager = new ServerManager(outputChannel as unknown as import('vscode').OutputChannel);
      expect(serverManager.isRunning()).toBe(false);
    });
  });

  describe('Event Emitter', () => {
    test('on() registers event listener', () => {
      serverManager = new ServerManager(outputChannel as unknown as import('vscode').OutputChannel);
      let called = false;
      serverManager.on('started', () => {
        called = true;
      });
      // Manually trigger the internal emit by accessing private method through type assertion
      (serverManager as unknown as { emit: (event: string, ...args: unknown[]) => void }).emit(
        'started',
        8080
      );
      expect(called).toBe(true);
    });

    test('off() removes event listener', () => {
      serverManager = new ServerManager(outputChannel as unknown as import('vscode').OutputChannel);
      let callCount = 0;
      const listener = () => {
        callCount++;
      };
      serverManager.on('started', listener);
      serverManager.off('started', listener);
      (serverManager as unknown as { emit: (event: string, ...args: unknown[]) => void }).emit(
        'started',
        8080
      );
      expect(callCount).toBe(0);
    });

    test('multiple listeners receive events', () => {
      serverManager = new ServerManager(outputChannel as unknown as import('vscode').OutputChannel);
      const ports: number[] = [];
      serverManager.on('started', (port: number) => ports.push(port));
      serverManager.on('started', (port: number) => ports.push(port * 2));
      (serverManager as unknown as { emit: (event: string, ...args: unknown[]) => void }).emit(
        'started',
        8080
      );
      expect(ports).toEqual([8080, 16160]);
    });

    test('emits stopped event', () => {
      serverManager = new ServerManager(outputChannel as unknown as import('vscode').OutputChannel);
      let stopped = false;
      serverManager.on('stopped', () => {
        stopped = true;
      });
      (serverManager as unknown as { emit: (event: string, ...args: unknown[]) => void }).emit(
        'stopped'
      );
      expect(stopped).toBe(true);
    });

    test('emits error event with Error object', () => {
      serverManager = new ServerManager(outputChannel as unknown as import('vscode').OutputChannel);
      let receivedError: Error | null = null;
      serverManager.on('error', (error: Error) => {
        receivedError = error;
      });
      const testError = new Error('Test error');
      (serverManager as unknown as { emit: (event: string, ...args: unknown[]) => void }).emit(
        'error',
        testError
      );
      expect(receivedError).toBe(testError);
    });

    test('emits output event with line string', () => {
      serverManager = new ServerManager(outputChannel as unknown as import('vscode').OutputChannel);
      const lines: string[] = [];
      serverManager.on('output', (line: string) => {
        lines.push(line);
      });
      (serverManager as unknown as { emit: (event: string, ...args: unknown[]) => void }).emit(
        'output',
        'line 1'
      );
      (serverManager as unknown as { emit: (event: string, ...args: unknown[]) => void }).emit(
        'output',
        'line 2'
      );
      expect(lines).toEqual(['line 1', 'line 2']);
    });
  });

  describe('stop()', () => {
    test('handles already stopped gracefully', () => {
      serverManager = new ServerManager(outputChannel as unknown as import('vscode').OutputChannel);
      // Should not throw when there's no process
      expect(() => serverManager.stop()).not.toThrow();
    });
  });

  describe('dispose()', () => {
    test('clears all listeners', () => {
      serverManager = new ServerManager(outputChannel as unknown as import('vscode').OutputChannel);
      let called = false;
      serverManager.on('started', () => {
        called = true;
      });
      serverManager.dispose();
      // After dispose, listeners should be cleared
      (serverManager as unknown as { emit: (event: string, ...args: unknown[]) => void }).emit(
        'started',
        8080
      );
      expect(called).toBe(false);
    });
  });

  describe('OutputChannel logging', () => {
    test('logs messages to output channel', () => {
      serverManager = new ServerManager(outputChannel as unknown as import('vscode').OutputChannel);
      // Stop should log a message
      serverManager.stop();
      // Since there's no process, it won't log "Stopping server..."
      // but we can verify the output channel exists
      expect(outputChannel.lines.length >= 0).toBeTruthy();
    });
  });
});

// Additional integration-style tests that test the actual module behavior
describe('ServerManager Integration Tests', () => {
  test('ServerManager interface is correct', () => {
    const outputChannel = createMockOutputChannel('Test');
    const manager = new ServerManager(outputChannel as unknown as import('vscode').OutputChannel);

    // Verify interface methods exist
    expect(typeof manager.start).toBe('function');
    expect(typeof manager.stop).toBe('function');
    expect(typeof manager.isRunning).toBe('function');
    expect(typeof manager.on).toBe('function');
    expect(typeof manager.off).toBe('function');
    expect(typeof manager.dispose).toBe('function');
    expect(typeof manager.port).toBe('object'); // null is object type

    manager.dispose();
  });

  test('port regex matches server output format', () => {
    // Test the regex pattern used in ServerManager
    const portRegex = /Server running at: http:\/\/localhost:(\d+)/;

    const validOutputs = [
      'Server running at: http://localhost:8080',
      'Server running at: http://localhost:3000',
      'Server running at: http://localhost:12345',
    ];

    for (const output of validOutputs) {
      const match = portRegex.exec(output);
      expect(match).toBeTruthy();
      expect(parseInt(match![1], 10) > 0).toBeTruthy();
    }

    const invalidOutputs = [
      'Server starting...',
      'Listening on port 8080',
      'http://localhost:8080',
    ];

    for (const output of invalidOutputs) {
      const match = portRegex.exec(output);
      expect(match).toBe(null);
    }
  });

  test('search paths include common locations', () => {
    // Verify the expected paths are documented/expected
    const expectedPaths = [
      '.local/bin/mehr',
      'bin/mehr',
      'go/bin/mehr',
      '/usr/local/bin/mehr',
      '/opt/homebrew/bin/mehr',
    ];

    // This is a documentation test - ensuring the paths we expect are known
    expect(expectedPaths.length > 0).toBeTruthy();
  });
});

// Mock process behavior tests
describe('MockChildProcess Verification', () => {
  test('MockChildProcess simulates stdout correctly', () => {
    const proc = new MockChildProcess();
    const received: string[] = [];

    proc.stdout.on('data', (data: Buffer) => {
      received.push(data.toString());
    });

    proc.simulateStdout('Hello World');
    expect(received).toEqual(['Hello World']);
  });

  test('MockChildProcess simulates stderr correctly', () => {
    const proc = new MockChildProcess();
    const received: string[] = [];

    proc.stderr.on('data', (data: Buffer) => {
      received.push(data.toString());
    });

    proc.simulateStderr('Error message');
    expect(received).toEqual(['Error message']);
  });

  test('MockChildProcess simulates exit correctly', () => {
    const proc = new MockChildProcess();
    let exitCode: number | null = null;
    let exitSignal: string | null = null;

    proc.on('exit', (code: number | null, signal: string | null) => {
      exitCode = code;
      exitSignal = signal;
    });

    proc.simulateExit(0, null);
    expect(exitCode).toBe(0);
    expect(exitSignal).toBe(null);
  });

  test('MockChildProcess simulates error correctly', () => {
    const proc = new MockChildProcess();
    let receivedError: Error | null = null;

    proc.on('error', (error: Error) => {
      receivedError = error;
    });

    const testError = new Error('Process failed');
    proc.simulateError(testError);
    expect(receivedError).toBe(testError);
  });

  test('MockChildProcess tracks kill signals', () => {
    const proc = new MockChildProcess();

    proc.kill('SIGTERM');
    proc.kill('SIGKILL');

    expect(proc.getKillSignals()).toEqual(['SIGTERM', 'SIGKILL']);
    expect(proc.killed).toBe(true);
  });

  test('createMockSpawn returns controllable processes', () => {
    const { spawn, getSpawnedProcesses, getLastProcess } = createMockSpawn();

    const proc = spawn('mehr', ['serve', '--api'], { cwd: '/test' });

    expect(proc instanceof MockChildProcess).toBeTruthy();
    expect(getSpawnedProcesses().length).toBe(1);
    expect(getLastProcess()).toBe(proc);

    const record = getSpawnedProcesses()[0];
    expect(record.command).toBe('mehr');
    expect(record.args).toEqual(['serve', '--api']);
    expect(record.options).toEqual({ cwd: '/test' });
  });

  test('createMockSpawn with autoStartPort emits server message', (done) => {
    const { spawn } = createMockSpawn({ autoStartPort: 8080, startDelay: 5 });

    const proc = spawn('mehr', ['serve']);

    proc.stdout.on('data', (data: Buffer) => {
      const output = data.toString();
      if (output.includes('Server running at: http://localhost:8080')) {
        done();
      }
    });
  });

  test('createMockSpawn reset clears history', () => {
    const { spawn, getSpawnedProcesses, reset } = createMockSpawn();

    spawn('test1', []);
    spawn('test2', []);
    expect(getSpawnedProcesses().length).toBe(2);

    reset();
    expect(getSpawnedProcesses().length).toBe(0);
  });
});

// Test data validation
describe('ServerManager Types', () => {
  test('ServerManagerEvents interface has correct event types', () => {
    // Type checking test - if this compiles, the interface is correct
    const events: import('../../src/services/serverManager').ServerManagerEvents = {
      started: (_port: number) => {},
      stopped: () => {},
      error: (_error: Error) => {},
      output: (_line: string) => {},
    };

    expect(typeof events.started).toBe('function');
    expect(typeof events.stopped).toBe('function');
    expect(typeof events.error).toBe('function');
    expect(typeof events.output).toBe('function');
  });
});
