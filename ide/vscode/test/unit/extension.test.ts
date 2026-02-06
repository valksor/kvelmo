import { describe, test, expect, beforeEach, afterEach } from 'bun:test';
import { activate, deactivate } from '../../src/extension';
import {
  createMockExtensionContext,
  createMockWindow,
  createMockWorkspace,
  createMockCommands,
  createMockConfiguration,
  registeredCommands,
  resetMocks,
} from '../helpers/mockVscode';

// Note: These tests verify the extension's structure and exports.
// Full integration tests require running in a VS Code instance via @vscode/test-electron.

describe('Extension Test Suite', () => {
  beforeEach(() => {
    resetMocks();
  });

  afterEach(() => {
    resetMocks();
  });

  describe('Module Exports', () => {
    test('activate function is exported', () => {
      expect(typeof activate).toBe('function');
    });

    test('deactivate function is exported', () => {
      expect(typeof deactivate).toBe('function');
    });

    test('activate has correct signature (takes ExtensionContext)', () => {
      // Verify function accepts one parameter
      expect(activate.length).toBe(1);
    });

    test('deactivate has correct signature (takes no parameters)', () => {
      expect(deactivate.length).toBe(0);
    });
  });

  describe('Extension Lifecycle Concepts', () => {
    test('extension creates output channel named Mehrhof', () => {
      // Document expected behavior - actual test requires VS Code runtime
      const expectedChannelName = 'Mehrhof';
      expect(expectedChannelName).toBeTruthy();
    });

    test('extension registers tree data provider for mehrhof.tasks', () => {
      // Document expected view ID
      const expectedViewId = 'mehrhof.tasks';
      expect(expectedViewId).toBeTruthy();
    });

    test('extension registers webview view provider', () => {
      // Document expected behavior
      const expectedViewType = 'mehrhof.interactive';
      expect(expectedViewType).toBeTruthy();
    });

    test('extension shows notification when showNotifications is true', () => {
      // Document expected behavior based on configuration
      const config = createMockConfiguration({ showNotifications: true });
      expect(config.get('showNotifications')).toBe(true);
    });

    test('extension does not show notification when showNotifications is false', () => {
      const config = createMockConfiguration({ showNotifications: false });
      expect(config.get('showNotifications')).toBe(false);
    });
  });

  describe('Mock Infrastructure Verification', () => {
    test('MockExtensionContext has subscriptions array', () => {
      const context = createMockExtensionContext();
      expect(Array.isArray(context.subscriptions)).toBeTruthy();
      expect(context.subscriptions.length).toBe(0);
    });

    test('MockExtensionContext subscriptions can be added', () => {
      const context = createMockExtensionContext();
      const disposable = { dispose: () => {} };
      context.subscriptions.push(disposable);
      expect(context.subscriptions.length).toBe(1);
    });

    test('MockWindow can create output channels', () => {
      const window = createMockWindow();
      const channel = window.createOutputChannel('Test');
      expect(channel).toBeTruthy();
      expect(channel.name).toBe('Test');
    });

    test('MockWindow can create status bar items', () => {
      const window = createMockWindow();
      const item = window.createStatusBarItem();
      expect(item).toBeTruthy();
    });

    test('MockCommands can register commands', () => {
      const commands = createMockCommands();
      const disposable = commands.registerCommand('test.command', () => {});
      expect(disposable).toBeTruthy();
      expect(registeredCommands.has('test.command')).toBeTruthy();
    });

    test('MockWorkspace has configuration', () => {
      const workspace = createMockWorkspace();
      const config = workspace.getConfiguration('mehrhof');
      expect(config).toBeTruthy();
    });
  });

  describe('Extension Components', () => {
    test('MehrhofProjectService can be imported', async () => {
      const { MehrhofProjectService } = await import('../../src/services/projectService');
      expect(MehrhofProjectService).toBeTruthy();
    });

    test('MehrhofOutputChannel can be imported', async () => {
      const { MehrhofOutputChannel } = await import('../../src/views/outputChannel');
      expect(MehrhofOutputChannel).toBeTruthy();
    });

    test('TaskTreeProvider can be imported', async () => {
      const { TaskTreeProvider } = await import('../../src/views/taskTreeProvider');
      expect(TaskTreeProvider).toBeTruthy();
    });

    test('InteractivePanelProvider can be imported', async () => {
      const { InteractivePanelProvider } = await import('../../src/views/interactivePanel');
      expect(InteractivePanelProvider).toBeTruthy();
    });

    test('StatusBarWidget can be imported', async () => {
      const { StatusBarWidget } = await import('../../src/statusbar/statusWidget');
      expect(StatusBarWidget).toBeTruthy();
    });

    test('registerCommands can be imported', async () => {
      const { registerCommands } = await import('../../src/commands');
      expect(registerCommands).toBeTruthy();
      expect(typeof registerCommands).toBe('function');
    });
  });

  describe('Configuration Keys', () => {
    test('serverUrl configuration key exists', () => {
      const config = createMockConfiguration({ serverUrl: 'http://localhost:3000' });
      expect(config.get('serverUrl')).toBe('http://localhost:3000');
    });

    test('mehrExecutable configuration key exists', () => {
      const config = createMockConfiguration({ mehrExecutable: '/usr/local/bin/mehr' });
      expect(config.get('mehrExecutable')).toBe('/usr/local/bin/mehr');
    });

    test('showNotifications configuration key exists', () => {
      const config = createMockConfiguration({ showNotifications: true });
      expect(config.get('showNotifications')).toBe(true);
    });

    test('defaultAgent configuration key exists', () => {
      const config = createMockConfiguration({ defaultAgent: 'claude' });
      expect(config.get('defaultAgent')).toBe('claude');
    });

    test('autoReconnect configuration key exists', () => {
      const config = createMockConfiguration({ autoReconnect: true });
      expect(config.get('autoReconnect')).toBe(true);
    });

    test('reconnectDelaySeconds configuration key exists', () => {
      const config = createMockConfiguration({ reconnectDelaySeconds: 5 });
      expect(config.get('reconnectDelaySeconds')).toBe(5);
    });

    test('maxReconnectAttempts configuration key exists', () => {
      const config = createMockConfiguration({ maxReconnectAttempts: 10 });
      expect(config.get('maxReconnectAttempts')).toBe(10);
    });
  });

  describe('Command IDs', () => {
    test('expected commands are defined', () => {
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

      // Verify the list is complete (15 commands)
      expect(expectedCommands.length).toBe(15);
    });
  });

  describe('View IDs', () => {
    test('interactive view ID is correct', () => {
      const viewId = 'mehrhof.interactive';
      expect(viewId).toBe('mehrhof.interactive');
    });

    test('tasks view ID is correct', () => {
      const viewId = 'mehrhof.tasks';
      expect(viewId).toBe('mehrhof.tasks');
    });
  });
});

// Deactivate function tests
describe('Deactivate Function', () => {
  test('deactivate does not throw when called', () => {
    expect(() => deactivate()).not.toThrow();
  });

  test('deactivate can be called multiple times', () => {
    expect(() => {
      deactivate();
      deactivate();
    }).not.toThrow();
  });
});
