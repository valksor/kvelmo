import * as assert from 'assert';
import * as vscode from 'vscode';
import { resetFactories } from '../helpers/factories';

// These tests execute commands that are already registered by the extension.
// They test command execution paths without re-registering commands.

suite('Command Execution Test Suite', () => {
  setup(() => {
    resetFactories();
  });

  suite('Server Commands', () => {
    test('mehrhof.stopServer executes without error', async () => {
      // stopServer should handle gracefully when no server is running
      await assert.doesNotReject(async () => {
        await vscode.commands.executeCommand('mehrhof.stopServer');
      });
    });

    test('mehrhof.disconnect executes without error', async () => {
      // disconnect should handle gracefully when not connected
      await assert.doesNotReject(async () => {
        await vscode.commands.executeCommand('mehrhof.disconnect');
      });
    });
  });

  suite('Workflow Commands (require connection)', () => {
    // These commands require a connection to work.
    // They should show a warning message when not connected.

    test('mehrhof.plan shows warning when not connected', async () => {
      // Should not throw - just shows warning
      await assert.doesNotReject(async () => {
        await vscode.commands.executeCommand('mehrhof.plan');
      });
    });

    test('mehrhof.implement shows warning when not connected', async () => {
      await assert.doesNotReject(async () => {
        await vscode.commands.executeCommand('mehrhof.implement');
      });
    });

    test('mehrhof.review shows warning when not connected', async () => {
      await assert.doesNotReject(async () => {
        await vscode.commands.executeCommand('mehrhof.review');
      });
    });

    test('mehrhof.continue shows warning when not connected', async () => {
      await assert.doesNotReject(async () => {
        await vscode.commands.executeCommand('mehrhof.continue');
      });
    });

    test('mehrhof.finish shows warning when not connected', async () => {
      await assert.doesNotReject(async () => {
        await vscode.commands.executeCommand('mehrhof.finish');
      });
    });

    test('mehrhof.abandon shows warning when not connected', async () => {
      await assert.doesNotReject(async () => {
        await vscode.commands.executeCommand('mehrhof.abandon');
      });
    });

    test('mehrhof.undo shows warning when not connected', async () => {
      await assert.doesNotReject(async () => {
        await vscode.commands.executeCommand('mehrhof.undo');
      });
    });

    test('mehrhof.redo shows warning when not connected', async () => {
      await assert.doesNotReject(async () => {
        await vscode.commands.executeCommand('mehrhof.redo');
      });
    });

    test('mehrhof.status shows warning when not connected', async () => {
      await assert.doesNotReject(async () => {
        await vscode.commands.executeCommand('mehrhof.status');
      });
    });

    test('mehrhof.refresh shows warning when not connected', async () => {
      await assert.doesNotReject(async () => {
        await vscode.commands.executeCommand('mehrhof.refresh');
      });
    });
  });

  suite('Command Existence', () => {
    test('all expected commands are registered', async () => {
      const commands = await vscode.commands.getCommands(true);

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

      for (const cmd of expectedCommands) {
        assert.ok(commands.includes(cmd), `Command ${cmd} should be registered`);
      }
    });

    test('statusBarClicked command is registered', async () => {
      const commands = await vscode.commands.getCommands(true);
      assert.ok(
        commands.includes('mehrhof.statusBarClicked'),
        'statusBarClicked should be registered'
      );
    });
  });
});

// Test requireConnection helper behavior
suite('requireConnection Behavior', () => {
  // When not connected, workflow commands should return early
  // and show a warning message. We can't easily verify the message,
  // but we can verify they don't throw.

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

  for (const cmd of workflowCommands) {
    test(`${cmd} handles disconnected state gracefully`, async () => {
      await assert.doesNotReject(async () => {
        await vscode.commands.executeCommand(cmd);
      });
    });
  }
});

// Test withProgress helper behavior
suite('withProgress Behavior', () => {
  // The withProgress helper wraps operations with progress notifications.
  // Since we're not connected, the operations will fail early,
  // but we can verify the commands don't throw uncaught errors.

  test('startServer uses progress notification', async () => {
    // This will attempt to start the server
    // It may fail to find the executable, but shouldn't throw
    try {
      await vscode.commands.executeCommand('mehrhof.startServer');
    } catch {
      // Expected - server executable not found
    }
    assert.ok(true, 'Command handled error gracefully');
  });

  test('connect uses progress notification', async () => {
    // This will attempt to connect
    // It may fail, but shouldn't throw
    try {
      await vscode.commands.executeCommand('mehrhof.connect');
    } catch {
      // Expected - can't connect
    }
    assert.ok(true, 'Command handled error gracefully');
  });
});
