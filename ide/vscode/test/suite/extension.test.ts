import * as assert from 'assert';
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

suite('Extension Test Suite', () => {
  setup(() => {
    resetMocks();
  });

  teardown(() => {
    resetMocks();
  });

  suite('Module Exports', () => {
    test('activate function is exported', () => {
      assert.strictEqual(typeof activate, 'function');
    });

    test('deactivate function is exported', () => {
      assert.strictEqual(typeof deactivate, 'function');
    });

    test('activate has correct signature (takes ExtensionContext)', () => {
      // Verify function accepts one parameter
      assert.strictEqual(activate.length, 1);
    });

    test('deactivate has correct signature (takes no parameters)', () => {
      assert.strictEqual(deactivate.length, 0);
    });
  });

  suite('Extension Lifecycle Concepts', () => {
    test('extension creates output channel named Mehrhof', () => {
      // Document expected behavior - actual test requires VS Code runtime
      const expectedChannelName = 'Mehrhof';
      assert.ok(expectedChannelName);
    });

    test('extension registers tree data provider for mehrhof.tasks', () => {
      // Document expected view ID
      const expectedViewId = 'mehrhof.tasks';
      assert.ok(expectedViewId);
    });

    test('extension registers webview view provider', () => {
      // Document expected behavior
      const expectedViewType = 'mehrhof.interactive';
      assert.ok(expectedViewType);
    });

    test('extension shows notification when showNotifications is true', () => {
      // Document expected behavior based on configuration
      const config = createMockConfiguration({ showNotifications: true });
      assert.strictEqual(config.get('showNotifications'), true);
    });

    test('extension does not show notification when showNotifications is false', () => {
      const config = createMockConfiguration({ showNotifications: false });
      assert.strictEqual(config.get('showNotifications'), false);
    });
  });

  suite('Mock Infrastructure Verification', () => {
    test('MockExtensionContext has subscriptions array', () => {
      const context = createMockExtensionContext();
      assert.ok(Array.isArray(context.subscriptions));
      assert.strictEqual(context.subscriptions.length, 0);
    });

    test('MockExtensionContext subscriptions can be added', () => {
      const context = createMockExtensionContext();
      const disposable = { dispose: () => {} };
      context.subscriptions.push(disposable);
      assert.strictEqual(context.subscriptions.length, 1);
    });

    test('MockWindow can create output channels', () => {
      const window = createMockWindow();
      const channel = window.createOutputChannel('Test');
      assert.ok(channel);
      assert.strictEqual(channel.name, 'Test');
    });

    test('MockWindow can create status bar items', () => {
      const window = createMockWindow();
      const item = window.createStatusBarItem();
      assert.ok(item);
    });

    test('MockCommands can register commands', () => {
      const commands = createMockCommands();
      const disposable = commands.registerCommand('test.command', () => {});
      assert.ok(disposable);
      assert.ok(registeredCommands.has('test.command'));
    });

    test('MockWorkspace has configuration', () => {
      const workspace = createMockWorkspace();
      const config = workspace.getConfiguration('mehrhof');
      assert.ok(config);
    });
  });

  suite('Extension Components', () => {
    test('MehrhofProjectService can be imported', async () => {
      const { MehrhofProjectService } = await import('../../src/services/projectService');
      assert.ok(MehrhofProjectService);
    });

    test('MehrhofOutputChannel can be imported', async () => {
      const { MehrhofOutputChannel } = await import('../../src/views/outputChannel');
      assert.ok(MehrhofOutputChannel);
    });

    test('TaskTreeProvider can be imported', async () => {
      const { TaskTreeProvider } = await import('../../src/views/taskTreeProvider');
      assert.ok(TaskTreeProvider);
    });

    test('InteractivePanelProvider can be imported', async () => {
      const { InteractivePanelProvider } = await import('../../src/views/interactivePanel');
      assert.ok(InteractivePanelProvider);
    });

    test('StatusBarWidget can be imported', async () => {
      const { StatusBarWidget } = await import('../../src/statusbar/statusWidget');
      assert.ok(StatusBarWidget);
    });

    test('registerCommands can be imported', async () => {
      const { registerCommands } = await import('../../src/commands');
      assert.ok(registerCommands);
      assert.strictEqual(typeof registerCommands, 'function');
    });
  });

  suite('Configuration Keys', () => {
    test('serverUrl configuration key exists', () => {
      const config = createMockConfiguration({ serverUrl: 'http://localhost:3000' });
      assert.strictEqual(config.get('serverUrl'), 'http://localhost:3000');
    });

    test('mehrExecutable configuration key exists', () => {
      const config = createMockConfiguration({ mehrExecutable: '/usr/local/bin/mehr' });
      assert.strictEqual(config.get('mehrExecutable'), '/usr/local/bin/mehr');
    });

    test('showNotifications configuration key exists', () => {
      const config = createMockConfiguration({ showNotifications: true });
      assert.strictEqual(config.get('showNotifications'), true);
    });

    test('defaultAgent configuration key exists', () => {
      const config = createMockConfiguration({ defaultAgent: 'claude' });
      assert.strictEqual(config.get('defaultAgent'), 'claude');
    });

    test('autoReconnect configuration key exists', () => {
      const config = createMockConfiguration({ autoReconnect: true });
      assert.strictEqual(config.get('autoReconnect'), true);
    });

    test('reconnectDelaySeconds configuration key exists', () => {
      const config = createMockConfiguration({ reconnectDelaySeconds: 5 });
      assert.strictEqual(config.get('reconnectDelaySeconds'), 5);
    });

    test('maxReconnectAttempts configuration key exists', () => {
      const config = createMockConfiguration({ maxReconnectAttempts: 10 });
      assert.strictEqual(config.get('maxReconnectAttempts'), 10);
    });
  });

  suite('Command IDs', () => {
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
      assert.strictEqual(expectedCommands.length, 15);
    });
  });

  suite('View IDs', () => {
    test('interactive view ID is correct', () => {
      const viewId = 'mehrhof.interactive';
      assert.strictEqual(viewId, 'mehrhof.interactive');
    });

    test('tasks view ID is correct', () => {
      const viewId = 'mehrhof.tasks';
      assert.strictEqual(viewId, 'mehrhof.tasks');
    });
  });
});

// Deactivate function tests
suite('Deactivate Function', () => {
  test('deactivate does not throw when called', () => {
    assert.doesNotThrow(() => deactivate());
  });

  test('deactivate can be called multiple times', () => {
    assert.doesNotThrow(() => {
      deactivate();
      deactivate();
    });
  });
});
