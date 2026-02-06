import * as vscode from 'vscode';
import { MehrhofProjectService } from '../services/projectService';
import { requireConnection, withProgress } from './helpers';

export function registerWorkflowCommands(
  context: vscode.ExtensionContext,
  service: MehrhofProjectService
): void {
  // Server commands
  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.startServer', async () => {
      await withProgress('Starting server...', async () => {
        await service.startServer();
      });
    })
  );

  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.stopServer', () => {
      service.stopServer();
      void vscode.window.showInformationMessage('Mehrhof: Server stopped');
    })
  );

  // Connection commands
  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.connect', async () => {
      await withProgress('Connecting...', async () => {
        await service.connect();
      });
    })
  );

  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.disconnect', () => {
      service.disconnect();
      void vscode.window.showInformationMessage('Mehrhof: Disconnected');
    })
  );

  // Workflow commands
  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.startTask', async () => {
      if (!requireConnection(service)) {
        return;
      }

      const ref = await vscode.window.showInputBox({
        prompt: 'Enter task reference (e.g., github:123, file:path/to/task.md)',
        placeHolder: 'Task reference',
      });

      if (!ref) {
        return;
      }

      await withProgress('Starting task...', async () => {
        const response = await service.client!.startTask({ ref });
        if (!response.success && response.error) {
          throw new Error(response.error);
        }
        await service.refreshState();
      });
    })
  );

  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.plan', async () => {
      if (!requireConnection(service)) {
        return;
      }

      await withProgress('Planning...', async () => {
        const response = await service.client!.plan();
        if (!response.success && response.error) {
          throw new Error(response.error);
        }
      });
    })
  );

  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.implement', async () => {
      if (!requireConnection(service)) {
        return;
      }

      await withProgress('Implementing...', async () => {
        const response = await service.client!.implement();
        if (!response.success && response.error) {
          throw new Error(response.error);
        }
      });
    })
  );

  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.review', async () => {
      if (!requireConnection(service)) {
        return;
      }

      await withProgress('Reviewing...', async () => {
        const response = await service.client!.review();
        if (!response.success && response.error) {
          throw new Error(response.error);
        }
      });
    })
  );

  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.continue', async () => {
      if (!requireConnection(service)) {
        return;
      }

      await withProgress('Continuing...', async () => {
        const response = await service.client!.continueWorkflow();
        if (!response.success) {
          throw new Error('Continue failed');
        }
      });
    })
  );

  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.finish', async () => {
      if (!requireConnection(service)) {
        return;
      }

      const confirm = await vscode.window.showQuickPick(['Yes', 'No'], {
        placeHolder: 'Finish the current task?',
      });

      if (confirm !== 'Yes') {
        return;
      }

      await withProgress('Finishing...', async () => {
        const response = await service.client!.finish();
        if (!response.success && response.error) {
          throw new Error(response.error);
        }
        await service.refreshState();
      });
    })
  );

  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.abandon', async () => {
      if (!requireConnection(service)) {
        return;
      }

      const confirm = await vscode.window.showWarningMessage(
        'Are you sure you want to abandon the current task? This cannot be undone.',
        { modal: true },
        'Abandon'
      );

      if (confirm !== 'Abandon') {
        return;
      }

      await withProgress('Abandoning...', async () => {
        const response = await service.client!.abandon();
        if (!response.success && response.error) {
          throw new Error(response.error);
        }
        await service.refreshState();
      });
    })
  );

  // Checkpoint commands
  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.undo', async () => {
      if (!requireConnection(service)) {
        return;
      }

      await withProgress('Undoing...', async () => {
        const response = await service.client!.undo();
        if (!response.success && response.error) {
          throw new Error(response.error);
        }
        await service.refreshState();
      });
    })
  );

  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.redo', async () => {
      if (!requireConnection(service)) {
        return;
      }

      await withProgress('Redoing...', async () => {
        const response = await service.client!.redo();
        if (!response.success && response.error) {
          throw new Error(response.error);
        }
        await service.refreshState();
      });
    })
  );
}
