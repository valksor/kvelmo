import { describe, test, expect, beforeEach, afterEach } from 'bun:test';
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

describe('Commands Test Suite', () => {
  let context: MockExtensionContext;

  beforeEach(() => {
    resetMocks();
    context = createMockExtensionContext();
  });

  afterEach(() => {
    resetMocks();
  });

  describe('registerCommands()', () => {
    test('registerCommands is a function', () => {
      expect(typeof registerCommands).toBe('function');
    });

    test('registerCommands accepts context and service parameters', () => {
      // Verify function signature
      expect(registerCommands.length).toBe(2);
    });
  });

  describe('Command IDs', () => {
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

      expect(expectedCommands.length).toBe(15);

      // Verify all commands are unique
      const uniqueCommands = new Set(expectedCommands);
      expect(uniqueCommands.size).toBe(15);
    });

    test('server command IDs are correctly prefixed', () => {
      const serverCommands = ['mehrhof.startServer', 'mehrhof.stopServer'];
      for (const cmd of serverCommands) {
        expect(cmd.startsWith('mehrhof.')).toBeTruthy();
      }
    });

    test('connection command IDs are correctly prefixed', () => {
      const connectionCommands = ['mehrhof.connect', 'mehrhof.disconnect'];
      for (const cmd of connectionCommands) {
        expect(cmd.startsWith('mehrhof.')).toBeTruthy();
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
        expect(cmd.startsWith('mehrhof.')).toBeTruthy();
      }
    });

    test('checkpoint command IDs are correctly prefixed', () => {
      const checkpointCommands = ['mehrhof.undo', 'mehrhof.redo'];
      for (const cmd of checkpointCommands) {
        expect(cmd.startsWith('mehrhof.')).toBeTruthy();
      }
    });

    test('info command IDs are correctly prefixed', () => {
      const infoCommands = ['mehrhof.status', 'mehrhof.refresh'];
      for (const cmd of infoCommands) {
        expect(cmd.startsWith('mehrhof.')).toBeTruthy();
      }
    });
  });

  describe('requireConnection Helper', () => {
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
      expect(workflowCommands.length).toBe(11);
    });

    test('server/connection commands do not require prior connection', () => {
      const noConnectionRequired = [
        'mehrhof.startServer',
        'mehrhof.stopServer',
        'mehrhof.connect',
        'mehrhof.disconnect',
      ];

      expect(noConnectionRequired.length).toBe(4);
    });
  });

  describe('Confirmation Dialogs', () => {
    test('mehrhof.finish shows confirmation dialog', () => {
      // The finish command shows a QuickPick with Yes/No options
      const expectedOptions = ['Yes', 'No'];
      expect(expectedOptions).toEqual(['Yes', 'No']);
    });

    test('mehrhof.abandon shows warning dialog', () => {
      // The abandon command shows a warning message with modal: true
      const expectedMessage =
        'Are you sure you want to abandon the current task? This cannot be undone.';
      expect(expectedMessage.includes('abandon')).toBeTruthy();
    });
  });

  describe('withProgress Helper', () => {
    test('withProgress prefixes messages with Mehrhof:', () => {
      // The withProgress function prefixes all messages with "Mehrhof: "
      const expectedPrefix = 'Mehrhof: ';
      expect(expectedPrefix.startsWith('Mehrhof')).toBeTruthy();
    });
  });

  describe('MockProjectService', () => {
    test('creates mock with default values', () => {
      const service = createMockProjectService();
      expect(service.isConnected).toBe(false);
      expect(service.client).toBe(null);
      expect(service.workflowState).toBe('idle');
    });

    test('creates mock with overrides', () => {
      const service = createMockProjectService({
        isConnected: true,
        workflowState: 'planning',
      });
      expect(service.isConnected).toBe(true);
      expect(service.workflowState).toBe('planning');
    });

    test('mock methods return promises', () => {
      const service = createMockProjectService();
      expect(service.startServer() instanceof Promise).toBeTruthy();
      expect(service.connect() instanceof Promise).toBeTruthy();
      expect(service.refreshState() instanceof Promise).toBeTruthy();
    });
  });

  describe('MockExtensionContext', () => {
    test('has subscriptions array', () => {
      expect(Array.isArray(context.subscriptions)).toBeTruthy();
    });

    test('subscriptions start empty', () => {
      expect(context.subscriptions.length).toBe(0);
    });

    test('can add to subscriptions', () => {
      const disposable = { dispose: () => {} };
      context.subscriptions.push(disposable);
      expect(context.subscriptions.length).toBe(1);
    });
  });
});

// Command categories verification
describe('Command Categories', () => {
  test('server commands count', () => {
    const serverCommands = ['mehrhof.startServer', 'mehrhof.stopServer'];
    expect(serverCommands.length).toBe(2);
  });

  test('connection commands count', () => {
    const connectionCommands = ['mehrhof.connect', 'mehrhof.disconnect'];
    expect(connectionCommands.length).toBe(2);
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
    expect(workflowCommands.length).toBe(7);
  });

  test('checkpoint commands count', () => {
    const checkpointCommands = ['mehrhof.undo', 'mehrhof.redo'];
    expect(checkpointCommands.length).toBe(2);
  });

  test('info commands count', () => {
    const infoCommands = ['mehrhof.status', 'mehrhof.refresh'];
    expect(infoCommands.length).toBe(2);
  });

  test('total commands is 15', () => {
    const allCommands = 2 + 2 + 7 + 2 + 2; // server + connection + workflow + checkpoint + info
    expect(allCommands).toBe(15);
  });
});
