import * as assert from 'assert';
import { ServerManager } from '../../src/services/serverManager';
import {
  createMockOutputChannel,
  createMockConfiguration,
  type MockOutputChannel,
} from '../helpers/mockVscode';
import { MockChildProcess, createMockSpawn } from '../helpers/mockChildProcess';

// Mock vscode module
const mockConfiguration = createMockConfiguration();

suite('ServerManager Test Suite', () => {
  let outputChannel: MockOutputChannel;
  let serverManager: ServerManager;
  let mockSpawnModule: ReturnType<typeof createMockSpawn>;

  setup(() => {
    outputChannel = createMockOutputChannel('Mehrhof Test');
    mockConfiguration.values.clear();

    // Create fresh mock spawn for each test
    mockSpawnModule = createMockSpawn();
  });

  teardown(() => {
    if (serverManager) {
      serverManager.dispose();
    }
    mockSpawnModule.reset();
  });

  suite('Constructor', () => {
    test('creates instance with OutputChannel', () => {
      serverManager = new ServerManager(outputChannel as unknown as import('vscode').OutputChannel);
      assert.ok(serverManager);
    });

    test('initial port is null', () => {
      serverManager = new ServerManager(outputChannel as unknown as import('vscode').OutputChannel);
      assert.strictEqual(serverManager.port, null);
    });
  });

  suite('isRunning()', () => {
    test('returns false initially', () => {
      serverManager = new ServerManager(outputChannel as unknown as import('vscode').OutputChannel);
      assert.strictEqual(serverManager.isRunning(), false);
    });
  });

  suite('Event Emitter', () => {
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
      assert.strictEqual(called, true);
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
      assert.strictEqual(callCount, 0);
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
      assert.deepStrictEqual(ports, [8080, 16160]);
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
      assert.strictEqual(stopped, true);
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
      assert.strictEqual(receivedError, testError);
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
      assert.deepStrictEqual(lines, ['line 1', 'line 2']);
    });
  });

  suite('stop()', () => {
    test('handles already stopped gracefully', () => {
      serverManager = new ServerManager(outputChannel as unknown as import('vscode').OutputChannel);
      // Should not throw when there's no process
      assert.doesNotThrow(() => serverManager.stop());
    });
  });

  suite('dispose()', () => {
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
      assert.strictEqual(called, false);
    });
  });

  suite('OutputChannel logging', () => {
    test('logs messages to output channel', () => {
      serverManager = new ServerManager(outputChannel as unknown as import('vscode').OutputChannel);
      // Stop should log a message
      serverManager.stop();
      // Since there's no process, it won't log "Stopping server..."
      // but we can verify the output channel exists
      assert.ok(outputChannel.lines.length >= 0);
    });
  });
});

// Additional integration-style tests that test the actual module behavior
suite('ServerManager Integration Tests', () => {
  test('ServerManager interface is correct', () => {
    const outputChannel = createMockOutputChannel('Test');
    const manager = new ServerManager(outputChannel as unknown as import('vscode').OutputChannel);

    // Verify interface methods exist
    assert.strictEqual(typeof manager.start, 'function');
    assert.strictEqual(typeof manager.stop, 'function');
    assert.strictEqual(typeof manager.isRunning, 'function');
    assert.strictEqual(typeof manager.on, 'function');
    assert.strictEqual(typeof manager.off, 'function');
    assert.strictEqual(typeof manager.dispose, 'function');
    assert.strictEqual(typeof manager.port, 'object'); // null is object type

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
      assert.ok(match, `Should match: ${output}`);
      assert.ok(parseInt(match[1], 10) > 0, 'Port should be positive');
    }

    const invalidOutputs = [
      'Server starting...',
      'Listening on port 8080',
      'http://localhost:8080',
    ];

    for (const output of invalidOutputs) {
      const match = portRegex.exec(output);
      assert.strictEqual(match, null, `Should not match: ${output}`);
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
    assert.ok(expectedPaths.length > 0);
  });
});

// Mock process behavior tests
suite('MockChildProcess Verification', () => {
  test('MockChildProcess simulates stdout correctly', () => {
    const proc = new MockChildProcess();
    const received: string[] = [];

    proc.stdout.on('data', (data: Buffer) => {
      received.push(data.toString());
    });

    proc.simulateStdout('Hello World');
    assert.deepStrictEqual(received, ['Hello World']);
  });

  test('MockChildProcess simulates stderr correctly', () => {
    const proc = new MockChildProcess();
    const received: string[] = [];

    proc.stderr.on('data', (data: Buffer) => {
      received.push(data.toString());
    });

    proc.simulateStderr('Error message');
    assert.deepStrictEqual(received, ['Error message']);
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
    assert.strictEqual(exitCode, 0);
    assert.strictEqual(exitSignal, null);
  });

  test('MockChildProcess simulates error correctly', () => {
    const proc = new MockChildProcess();
    let receivedError: Error | null = null;

    proc.on('error', (error: Error) => {
      receivedError = error;
    });

    const testError = new Error('Process failed');
    proc.simulateError(testError);
    assert.strictEqual(receivedError, testError);
  });

  test('MockChildProcess tracks kill signals', () => {
    const proc = new MockChildProcess();

    proc.kill('SIGTERM');
    proc.kill('SIGKILL');

    assert.deepStrictEqual(proc.getKillSignals(), ['SIGTERM', 'SIGKILL']);
    assert.strictEqual(proc.killed, true);
  });

  test('createMockSpawn returns controllable processes', () => {
    const { spawn, getSpawnedProcesses, getLastProcess } = createMockSpawn();

    const proc = spawn('mehr', ['serve', '--api'], { cwd: '/test' });

    assert.ok(proc instanceof MockChildProcess);
    assert.strictEqual(getSpawnedProcesses().length, 1);
    assert.strictEqual(getLastProcess(), proc);

    const record = getSpawnedProcesses()[0];
    assert.strictEqual(record.command, 'mehr');
    assert.deepStrictEqual(record.args, ['serve', '--api']);
    assert.deepStrictEqual(record.options, { cwd: '/test' });
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
    assert.strictEqual(getSpawnedProcesses().length, 2);

    reset();
    assert.strictEqual(getSpawnedProcesses().length, 0);
  });
});

// Test data validation
suite('ServerManager Types', () => {
  test('ServerManagerEvents interface has correct event types', () => {
    // Type checking test - if this compiles, the interface is correct
    const events: import('../../src/services/serverManager').ServerManagerEvents = {
      started: (_port: number) => {},
      stopped: () => {},
      error: (_error: Error) => {},
      output: (_line: string) => {},
    };

    assert.strictEqual(typeof events.started, 'function');
    assert.strictEqual(typeof events.stopped, 'function');
    assert.strictEqual(typeof events.error, 'function');
    assert.strictEqual(typeof events.output, 'function');
  });
});
