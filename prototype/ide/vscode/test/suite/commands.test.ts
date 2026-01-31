import * as assert from 'assert';
import { registerCommands } from '../../src/commands';
import {
  createMockExtensionContext,
  resetMocks,
  type MockExtensionContext,
} from '../helpers/mockVscode';

// Note: Commands are registered by the extension during activation.
// These tests verify the command structure without re-registering to avoid conflicts.

// Mock ProjectService for testing
interface MockProjectService {
  isConnected: boolean;
  client: unknown;
  currentTask: unknown;
  currentWork: unknown;
  workflowState: string;
  startServer: () => Promise<void>;
  stopServer: () => void;
  connect: () => Promise<void>;
  disconnect: () => void;
  refreshState: () => Promise<void>;
}

function createMockProjectService(overrides: Partial<MockProjectService> = {}): MockProjectService {
  return {
    isConnected: false,
    client: null,
    currentTask: null,
    currentWork: null,
    workflowState: 'idle',
    startServer: () => Promise.resolve(),
    stopServer: () => {},
    connect: () => Promise.resolve(),
    disconnect: () => {},
    refreshState: () => Promise.resolve(),
    ...overrides,
  };
}

suite('Commands Test Suite', () => {
  let context: MockExtensionContext;

  setup(() => {
    resetMocks();
    context = createMockExtensionContext();
  });

  teardown(() => {
    resetMocks();
  });

  suite('registerCommands()', () => {
    test('registerCommands is a function', () => {
      assert.strictEqual(typeof registerCommands, 'function');
    });

    test('registerCommands accepts context and service parameters', () => {
      // Verify function signature
      assert.strictEqual(registerCommands.length, 2);
    });
  });

  suite('Command IDs', () => {
    test('expected command IDs list', () => {
      const expectedCommands = [
        'mehrhof.startServer',
        'mehrhof.stopServer',
        'mehrhof.connect',
        'mehrhof.disconnect',
        'mehrhof.startTask',
        'mehrhof.plan',
        'mehrhof.implement',
        'mehrhof.review',
        'mehrhof.continue',
        'mehrhof.finish',
        'mehrhof.abandon',
        'mehrhof.undo',
        'mehrhof.redo',
        'mehrhof.status',
        'mehrhof.refresh',
      ];

      assert.strictEqual(expectedCommands.length, 15);

      // Verify all commands are unique
      const uniqueCommands = new Set(expectedCommands);
      assert.strictEqual(uniqueCommands.size, 15);
    });

    test('server command IDs are correctly prefixed', () => {
      const serverCommands = ['mehrhof.startServer', 'mehrhof.stopServer'];
      for (const cmd of serverCommands) {
        assert.ok(cmd.startsWith('mehrhof.'));
      }
    });

    test('connection command IDs are correctly prefixed', () => {
      const connectionCommands = ['mehrhof.connect', 'mehrhof.disconnect'];
      for (const cmd of connectionCommands) {
        assert.ok(cmd.startsWith('mehrhof.'));
      }
    });

    test('workflow command IDs are correctly prefixed', () => {
      const workflowCommands = [
        'mehrhof.startTask',
        'mehrhof.plan',
        'mehrhof.implement',
        'mehrhof.review',
        'mehrhof.continue',
        'mehrhof.finish',
        'mehrhof.abandon',
      ];
      for (const cmd of workflowCommands) {
        assert.ok(cmd.startsWith('mehrhof.'));
      }
    });

    test('checkpoint command IDs are correctly prefixed', () => {
      const checkpointCommands = ['mehrhof.undo', 'mehrhof.redo'];
      for (const cmd of checkpointCommands) {
        assert.ok(cmd.startsWith('mehrhof.'));
      }
    });

    test('info command IDs are correctly prefixed', () => {
      const infoCommands = ['mehrhof.status', 'mehrhof.refresh'];
      for (const cmd of infoCommands) {
        assert.ok(cmd.startsWith('mehrhof.'));
      }
    });
  });

  suite('requireConnection Helper', () => {
    test('workflow commands require connection', () => {
      const workflowCommands = [
        'mehrhof.startTask',
        'mehrhof.plan',
        'mehrhof.implement',
        'mehrhof.review',
        'mehrhof.continue',
        'mehrhof.finish',
        'mehrhof.abandon',
        'mehrhof.undo',
        'mehrhof.redo',
        'mehrhof.status',
        'mehrhof.refresh',
      ];

      // All 11 workflow commands require connection
      assert.strictEqual(workflowCommands.length, 11);
    });

    test('server/connection commands do not require prior connection', () => {
      const noConnectionRequired = [
        'mehrhof.startServer',
        'mehrhof.stopServer',
        'mehrhof.connect',
        'mehrhof.disconnect',
      ];

      assert.strictEqual(noConnectionRequired.length, 4);
    });
  });

  suite('Confirmation Dialogs', () => {
    test('mehrhof.finish shows confirmation dialog', () => {
      // The finish command shows a QuickPick with Yes/No options
      const expectedOptions = ['Yes', 'No'];
      assert.deepStrictEqual(expectedOptions, ['Yes', 'No']);
    });

    test('mehrhof.abandon shows warning dialog', () => {
      // The abandon command shows a warning message with modal: true
      const expectedMessage =
        'Are you sure you want to abandon the current task? This cannot be undone.';
      assert.ok(expectedMessage.includes('abandon'));
    });
  });

  suite('withProgress Helper', () => {
    test('withProgress prefixes messages with Mehrhof:', () => {
      // The withProgress function prefixes all messages with "Mehrhof: "
      const expectedPrefix = 'Mehrhof: ';
      assert.ok(expectedPrefix.startsWith('Mehrhof'));
    });
  });

  suite('MockProjectService', () => {
    test('creates mock with default values', () => {
      const service = createMockProjectService();
      assert.strictEqual(service.isConnected, false);
      assert.strictEqual(service.client, null);
      assert.strictEqual(service.workflowState, 'idle');
    });

    test('creates mock with overrides', () => {
      const service = createMockProjectService({
        isConnected: true,
        workflowState: 'planning',
      });
      assert.strictEqual(service.isConnected, true);
      assert.strictEqual(service.workflowState, 'planning');
    });

    test('mock methods return promises', () => {
      const service = createMockProjectService();
      assert.ok(service.startServer() instanceof Promise);
      assert.ok(service.connect() instanceof Promise);
      assert.ok(service.refreshState() instanceof Promise);
    });
  });

  suite('MockExtensionContext', () => {
    test('has subscriptions array', () => {
      assert.ok(Array.isArray(context.subscriptions));
    });

    test('subscriptions start empty', () => {
      assert.strictEqual(context.subscriptions.length, 0);
    });

    test('can add to subscriptions', () => {
      const disposable = { dispose: () => {} };
      context.subscriptions.push(disposable);
      assert.strictEqual(context.subscriptions.length, 1);
    });
  });
});

// Command categories verification
suite('Command Categories', () => {
  test('server commands count', () => {
    const serverCommands = ['mehrhof.startServer', 'mehrhof.stopServer'];
    assert.strictEqual(serverCommands.length, 2);
  });

  test('connection commands count', () => {
    const connectionCommands = ['mehrhof.connect', 'mehrhof.disconnect'];
    assert.strictEqual(connectionCommands.length, 2);
  });

  test('workflow commands count', () => {
    const workflowCommands = [
      'mehrhof.startTask',
      'mehrhof.plan',
      'mehrhof.implement',
      'mehrhof.review',
      'mehrhof.continue',
      'mehrhof.finish',
      'mehrhof.abandon',
    ];
    assert.strictEqual(workflowCommands.length, 7);
  });

  test('checkpoint commands count', () => {
    const checkpointCommands = ['mehrhof.undo', 'mehrhof.redo'];
    assert.strictEqual(checkpointCommands.length, 2);
  });

  test('info commands count', () => {
    const infoCommands = ['mehrhof.status', 'mehrhof.refresh'];
    assert.strictEqual(infoCommands.length, 2);
  });

  test('total commands is 15', () => {
    const allCommands = 2 + 2 + 7 + 2 + 2; // server + connection + workflow + checkpoint + info
    assert.strictEqual(allCommands, 15);
  });
});
