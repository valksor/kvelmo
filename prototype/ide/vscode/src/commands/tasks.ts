import * as vscode from 'vscode';
import { MehrhofProjectService } from '../services/projectService';
import { requireConnection, withProgress } from './helpers';

export function registerTaskCommands(
  context: vscode.ExtensionContext,
  service: MehrhofProjectService
): void {
  // Info commands
  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.status', async () => {
      if (!requireConnection(service)) {
        return;
      }

      await service.refreshState();

      const task = service.currentTask;
      const work = service.currentWork;
      const state = service.workflowState;

      if (task) {
        const message = `Task: ${work?.title ?? task.id}\nState: ${state}\nBranch: ${task.branch ?? 'N/A'}`;
        void vscode.window.showInformationMessage(message, { modal: true });
      } else {
        void vscode.window.showInformationMessage('No active task');
      }
    })
  );

  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.refresh', async () => {
      if (!requireConnection(service)) {
        return;
      }

      await service.refreshState();
      void vscode.window.showInformationMessage('Mehrhof: Refreshed');
    })
  );

  // Note command
  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.note', async () => {
      if (!requireConnection(service)) {
        return;
      }

      const task = service.currentTask;
      if (!task) {
        void vscode.window.showWarningMessage('Mehrhof: No active task');
        return;
      }

      const message = await vscode.window.showInputBox({
        prompt: 'Enter note message',
        placeHolder: 'Note content...',
      });

      if (!message) {
        return;
      }

      await withProgress('Adding note...', async () => {
        const response = await service.client!.addNote(task.id, { message });
        if (!response.success && response.error) {
          throw new Error(response.error);
        }
        void vscode.window.showInformationMessage(
          `Mehrhof: Note #${response.note_number ?? 'N/A'} added`
        );
      });
    })
  );

  // Question command
  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.question', async () => {
      if (!requireConnection(service)) {
        return;
      }

      const task = service.currentTask;
      if (!task) {
        void vscode.window.showWarningMessage('Mehrhof: No active task');
        return;
      }

      const message = await vscode.window.showInputBox({
        prompt: 'Enter question for the agent',
        placeHolder: 'Your question...',
      });

      if (!message) {
        return;
      }

      await withProgress('Asking question...', async () => {
        const response = await service.client!.question({ message });
        if (!response.success && response.error) {
          throw new Error(response.error);
        }
      });
    })
  );

  // Reset command
  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.reset', async () => {
      if (!requireConnection(service)) {
        return;
      }

      const confirm = await vscode.window.showWarningMessage(
        'Reset workflow state to idle? This will not lose your work.',
        { modal: true },
        'Reset'
      );

      if (confirm !== 'Reset') {
        return;
      }

      await withProgress('Resetting...', async () => {
        const response = await service.client!.reset();
        if (!response.success && response.error) {
          throw new Error(response.error);
        }
        await service.refreshState();
        void vscode.window.showInformationMessage('Mehrhof: Workflow reset to idle');
      });
    })
  );

  // Cost command
  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.cost', async () => {
      if (!requireConnection(service)) {
        return;
      }

      const task = service.currentTask;
      if (!task) {
        // Show all costs if no active task
        await withProgress('Fetching costs...', async () => {
          const response = await service.client!.getAllCosts();
          const total = response.grand_total;
          const message = `Total Cost: $${total.cost_usd.toFixed(4)}\nTokens: ${total.total_tokens.toLocaleString()} (${total.input_tokens.toLocaleString()} in, ${total.output_tokens.toLocaleString()} out)\nCached: ${total.cached_tokens.toLocaleString()}`;
          void vscode.window.showInformationMessage(message, { modal: true });
        });
        return;
      }

      await withProgress('Fetching task costs...', async () => {
        const response = await service.client!.getTaskCosts(task.id);
        const message = `Task: ${response.title ?? task.id}\nCost: $${response.total_cost_usd.toFixed(4)}\nTokens: ${response.total_tokens.toLocaleString()} (${response.input_tokens.toLocaleString()} in, ${response.output_tokens.toLocaleString()} out)\nCached: ${response.cached_tokens.toLocaleString()} (${response.cached_percent?.toFixed(1) ?? 0}%)`;
        void vscode.window.showInformationMessage(message, { modal: true });
      });
    })
  );

  // Quick task command
  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.quick', async () => {
      if (!requireConnection(service)) {
        return;
      }

      const description = await vscode.window.showInputBox({
        prompt: 'Enter task description',
        placeHolder: 'Task description...',
      });

      if (!description) {
        return;
      }

      await withProgress('Creating quick task...', async () => {
        const response = await service.client!.executeCommand({
          command: 'quick',
          args: [description],
        });
        if (!response.success && response.error) {
          throw new Error(response.error);
        }
        void vscode.window.showInformationMessage(response.message ?? 'Quick task created');
      });
    })
  );

  // Delete queue task command
  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.deleteTask', async () => {
      if (!requireConnection(service)) {
        return;
      }

      const taskRef = await vscode.window.showInputBox({
        prompt: 'Enter task reference (queue/task-id)',
        placeHolder: 'quick-tasks/task-1',
      });

      if (!taskRef) {
        return;
      }

      const confirm = await vscode.window.showWarningMessage(
        `Delete task ${taskRef}? This cannot be undone.`,
        { modal: true },
        'Delete'
      );

      if (confirm !== 'Delete') {
        return;
      }

      const [queueId, taskId] = taskRef.split('/');
      if (!queueId || !taskId) {
        void vscode.window.showErrorMessage('Invalid task reference format');
        return;
      }

      await withProgress('Deleting task...', async () => {
        const response = await service.client!.deleteQueueTask(queueId, taskId);
        if (!response.success && response.error) {
          throw new Error(response.error);
        }
        void vscode.window.showInformationMessage(response.message ?? 'Task deleted');
      });
    })
  );

  // Export queue task command
  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.exportTask', async () => {
      if (!requireConnection(service)) {
        return;
      }

      const taskRef = await vscode.window.showInputBox({
        prompt: 'Enter task reference (queue/task-id)',
        placeHolder: 'quick-tasks/task-1',
      });

      if (!taskRef) {
        return;
      }

      const [queueId, taskId] = taskRef.split('/');
      if (!queueId || !taskId) {
        void vscode.window.showErrorMessage('Invalid task reference format');
        return;
      }

      await withProgress('Exporting task...', async () => {
        const response = await service.client!.exportQueueTask(queueId, taskId);
        if (!response.success && response.error) {
          throw new Error(response.error);
        }
        if (response.markdown) {
          // Open the markdown in a new document
          const doc = await vscode.workspace.openTextDocument({
            content: response.markdown,
            language: 'markdown',
          });
          await vscode.window.showTextDocument(doc);
        }
        void vscode.window.showInformationMessage(response.message ?? 'Task exported');
      });
    })
  );

  // Optimize queue task command
  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.optimizeTask', async () => {
      if (!requireConnection(service)) {
        return;
      }

      const taskRef = await vscode.window.showInputBox({
        prompt: 'Enter task reference (queue/task-id)',
        placeHolder: 'quick-tasks/task-1',
      });

      if (!taskRef) {
        return;
      }

      const [queueId, taskId] = taskRef.split('/');
      if (!queueId || !taskId) {
        void vscode.window.showErrorMessage('Invalid task reference format');
        return;
      }

      await withProgress('Optimizing task with AI...', async () => {
        const response = await service.client!.optimizeQueueTask(queueId, taskId);
        if (!response.success && response.error) {
          throw new Error(response.error);
        }
        let message = 'Task optimized';
        if (response.original_title !== response.optimized_title) {
          message += `\nTitle: ${response.original_title} → ${response.optimized_title}`;
        }
        if (response.added_labels && response.added_labels.length > 0) {
          message += `\nAdded labels: ${response.added_labels.join(', ')}`;
        }
        void vscode.window.showInformationMessage(message, { modal: true });
      });
    })
  );

  // Submit queue task command
  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.submitTask', async () => {
      if (!requireConnection(service)) {
        return;
      }

      const taskRef = await vscode.window.showInputBox({
        prompt: 'Enter task reference (queue/task-id)',
        placeHolder: 'quick-tasks/task-1',
      });

      if (!taskRef) {
        return;
      }

      const provider = await vscode.window.showInputBox({
        prompt: 'Enter provider name',
        placeHolder: 'github, jira, wrike, linear, etc.',
      });

      if (!provider) {
        return;
      }

      const [queueId, taskId] = taskRef.split('/');
      if (!queueId || !taskId) {
        void vscode.window.showErrorMessage('Invalid task reference format');
        return;
      }

      await withProgress('Submitting task...', async () => {
        const response = await service.client!.submitQueueTask(queueId, taskId, provider);
        if (!response.success && response.error) {
          throw new Error(response.error);
        }
        let message = response.message ?? 'Task submitted';
        if (response.external_id) {
          message += `\nExternal ID: ${response.external_id}`;
        }
        if (response.url) {
          message += `\nURL: ${response.url}`;
        }
        void vscode.window.showInformationMessage(message, { modal: true });
      });
    })
  );

  // Sync task command
  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.syncTask', async () => {
      if (!requireConnection(service)) {
        return;
      }

      const task = service.currentTask;
      if (!task) {
        void vscode.window.showWarningMessage('Mehrhof: No active task to sync');
        return;
      }

      await withProgress('Syncing task...', async () => {
        const response = await service.client!.syncTask();
        if (!response.success && response.error) {
          throw new Error(response.error);
        }
        void vscode.window.showInformationMessage(response.message ?? 'Task synced');
      });
    })
  );
}
