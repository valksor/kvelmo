import { describe, test, expect, beforeEach, afterEach, mock } from 'bun:test';
import * as vscode from 'vscode';
import { registerTaskCommands } from '../../src/commands/tasks';
import {
  createMockExtensionContext,
  resetMocks,
  registeredCommands,
  type MockExtensionContext,
} from '../helpers/mockVscode';
import {
  createWorkflowResponse,
  createInteractiveCommandResponse,
  createTaskInfo,
  createTaskWork,
  createTaskCostResponse,
  resetFactories,
  saveFetch,
  restoreFetch,
} from '../helpers/factories';
import type { MehrhofProjectService } from '../../src/services/projectService';
import type { MehrhofApiClient } from '../../src/api/client';
import type {
  AddNoteResponse,
  AllCostsResponse,
  DeleteQueueTaskResponse,
  ExportQueueTaskResponse,
  OptimizeQueueTaskResponse,
  SubmitQueueTaskResponse,
  SyncTaskResponse,
} from '../../src/api/models';

// Create a mock API client with all task methods
function createMockApiClient(): Partial<MehrhofApiClient> {
  return {
    addNote: mock(() => Promise.resolve({ success: true, note_number: 1 } as AddNoteResponse)),
    question: mock(() => Promise.resolve(createWorkflowResponse())),
    reset: mock(() => Promise.resolve(createWorkflowResponse())),
    getAllCosts: mock(() =>
      Promise.resolve({
        tasks: [],
        grand_total: {
          input_tokens: 10000,
          output_tokens: 5000,
          total_tokens: 15000,
          cached_tokens: 2000,
          cost_usd: 0.15,
        },
      } as AllCostsResponse)
    ),
    getTaskCosts: mock(() => Promise.resolve(createTaskCostResponse())),
    executeCommand: mock(() => Promise.resolve(createInteractiveCommandResponse())),
    deleteQueueTask: mock(() =>
      Promise.resolve({ success: true, message: 'Task deleted' } as DeleteQueueTaskResponse)
    ),
    exportQueueTask: mock(() =>
      Promise.resolve({
        success: true,
        markdown: '# Exported Task\n\nContent here',
      } as ExportQueueTaskResponse)
    ),
    optimizeQueueTask: mock(() =>
      Promise.resolve({
        success: true,
        original_title: 'Fix bug',
        optimized_title: 'Fix authentication bug',
        added_labels: ['bug', 'auth'],
      } as OptimizeQueueTaskResponse)
    ),
    submitQueueTask: mock(() =>
      Promise.resolve({
        success: true,
        external_id: 'GH-123',
        url: 'https://github.com/owner/repo/issues/123',
      } as SubmitQueueTaskResponse)
    ),
    syncTask: mock(() =>
      Promise.resolve({ success: true, message: 'Task synced' } as SyncTaskResponse)
    ),
  };
}

// Create a mock project service
function createMockProjectService(
  connected: boolean = true,
  hasTask: boolean = false
): { service: Partial<MehrhofProjectService>; client: Partial<MehrhofApiClient> } {
  const client = createMockApiClient();
  const service: Partial<MehrhofProjectService> = {
    isConnected: connected,
    client: connected ? (client as MehrhofApiClient) : null,
    currentTask: hasTask ? createTaskInfo({ id: 'task-123' }) : null,
    currentWork: hasTask ? createTaskWork({ title: 'Test Task' }) : null,
    workflowState: hasTask ? 'planning' : 'idle',
    refreshState: mock(() => Promise.resolve()),
  };
  return { service, client };
}

describe('Task Commands Test Suite', () => {
  let context: MockExtensionContext;

  beforeEach(() => {
    resetMocks();
    resetFactories();
    saveFetch();
    context = createMockExtensionContext();
  });

  afterEach(() => {
    resetMocks();
    restoreFetch();
  });

  describe('registerTaskCommands', () => {
    test('registers all expected commands', () => {
      const { service } = createMockProjectService();
      registerTaskCommands(context, service as MehrhofProjectService);

      const expectedCommands = [
        'mehrhof.status',
        'mehrhof.refresh',
        'mehrhof.note',
        'mehrhof.question',
        'mehrhof.reset',
        'mehrhof.cost',
        'mehrhof.quick',
        'mehrhof.deleteTask',
        'mehrhof.exportTask',
        'mehrhof.optimizeTask',
        'mehrhof.submitTask',
        'mehrhof.syncTask',
      ];

      for (const cmd of expectedCommands) {
        expect(registeredCommands.has(cmd)).toBe(true);
      }
    });

    test('adds disposables to context subscriptions', () => {
      const { service } = createMockProjectService();
      registerTaskCommands(context, service as MehrhofProjectService);
      expect(context.subscriptions.length).toBe(12);
    });
  });

  describe('mehrhof.status command', () => {
    test('returns early when not connected', async () => {
      const { service } = createMockProjectService(false);
      registerTaskCommands(context, service as MehrhofProjectService);

      const handler = registeredCommands.get('mehrhof.status');
      await handler!();
    });

    test('shows task info when task is active', async () => {
      const { service } = createMockProjectService(true, true);
      registerTaskCommands(context, service as MehrhofProjectService);

      let infoMessage: string | undefined;
      let modalOptions: { modal?: boolean } | undefined;
      (vscode.window.showInformationMessage as ReturnType<typeof mock>) = mock(
        (msg: string, options?: { modal?: boolean }) => {
          infoMessage = msg;
          modalOptions = options;
          return Promise.resolve(undefined);
        }
      );

      const handler = registeredCommands.get('mehrhof.status');
      await handler!();

      expect(infoMessage).toContain('Test Task');
      expect(infoMessage).toContain('planning');
      expect(modalOptions?.modal).toBe(true);
    });

    test('shows no active task message when no task', async () => {
      const { service } = createMockProjectService(true, false);
      registerTaskCommands(context, service as MehrhofProjectService);

      let infoMessage: string | undefined;
      (vscode.window.showInformationMessage as ReturnType<typeof mock>) = mock((msg: string) => {
        infoMessage = msg;
        return Promise.resolve(undefined);
      });

      const handler = registeredCommands.get('mehrhof.status');
      await handler!();

      expect(infoMessage).toBe('No active task');
    });
  });

  describe('mehrhof.refresh command', () => {
    test('calls refreshState', async () => {
      const { service } = createMockProjectService();
      registerTaskCommands(context, service as MehrhofProjectService);

      const handler = registeredCommands.get('mehrhof.refresh');
      await handler!();

      expect(service.refreshState).toHaveBeenCalled();
    });
  });

  describe('mehrhof.note command', () => {
    test('shows warning when no active task', async () => {
      const { service } = createMockProjectService(true, false);
      registerTaskCommands(context, service as MehrhofProjectService);

      let warningMessage: string | undefined;
      (vscode.window.showWarningMessage as ReturnType<typeof mock>) = mock((msg: string) => {
        warningMessage = msg;
        return Promise.resolve(undefined);
      });

      const handler = registeredCommands.get('mehrhof.note');
      await handler!();

      expect(warningMessage).toContain('No active task');
    });

    test('calls addNote API with message', async () => {
      const { service, client } = createMockProjectService(true, true);
      registerTaskCommands(context, service as MehrhofProjectService);

      (vscode.window.showInputBox as ReturnType<typeof mock>) = mock(() =>
        Promise.resolve('This is my note')
      );

      const handler = registeredCommands.get('mehrhof.note');
      await handler!();

      expect(client.addNote).toHaveBeenCalledWith('task-123', { message: 'This is my note' });
    });
  });

  describe('mehrhof.question command', () => {
    test('calls question API with message', async () => {
      const { service, client } = createMockProjectService(true, true);
      registerTaskCommands(context, service as MehrhofProjectService);

      (vscode.window.showInputBox as ReturnType<typeof mock>) = mock(() =>
        Promise.resolve('What is the best approach?')
      );

      const handler = registeredCommands.get('mehrhof.question');
      await handler!();

      expect(client.question).toHaveBeenCalledWith({ message: 'What is the best approach?' });
    });
  });

  describe('mehrhof.reset command', () => {
    test('calls reset API after confirmation', async () => {
      const { service, client } = createMockProjectService();
      registerTaskCommands(context, service as MehrhofProjectService);

      (vscode.window.showWarningMessage as ReturnType<typeof mock>) = mock(() =>
        Promise.resolve('Reset')
      );

      const handler = registeredCommands.get('mehrhof.reset');
      await handler!();

      expect(client.reset).toHaveBeenCalled();
      expect(service.refreshState).toHaveBeenCalled();
    });

    test('does not reset when user cancels', async () => {
      const { service, client } = createMockProjectService();
      registerTaskCommands(context, service as MehrhofProjectService);

      (vscode.window.showWarningMessage as ReturnType<typeof mock>) = mock(() =>
        Promise.resolve(undefined)
      );

      const handler = registeredCommands.get('mehrhof.reset');
      await handler!();

      expect(client.reset).not.toHaveBeenCalled();
    });
  });

  describe('mehrhof.cost command', () => {
    test('shows all costs when no active task', async () => {
      const { service, client } = createMockProjectService(true, false);
      registerTaskCommands(context, service as MehrhofProjectService);

      let infoMessage: string | undefined;
      (vscode.window.showInformationMessage as ReturnType<typeof mock>) = mock((msg: string) => {
        infoMessage = msg;
        return Promise.resolve(undefined);
      });

      const handler = registeredCommands.get('mehrhof.cost');
      await handler!();

      expect(client.getAllCosts).toHaveBeenCalled();
      expect(infoMessage).toContain('Total Cost');
    });

    test('shows task costs when task is active', async () => {
      const { service, client } = createMockProjectService(true, true);
      registerTaskCommands(context, service as MehrhofProjectService);

      let infoMessage: string | undefined;
      (vscode.window.showInformationMessage as ReturnType<typeof mock>) = mock((msg: string) => {
        infoMessage = msg;
        return Promise.resolve(undefined);
      });

      const handler = registeredCommands.get('mehrhof.cost');
      await handler!();

      expect(client.getTaskCosts).toHaveBeenCalledWith('task-123');
      expect(infoMessage).toContain('Cost');
    });
  });

  describe('mehrhof.quick command', () => {
    test('calls executeCommand with description', async () => {
      const { service, client } = createMockProjectService();
      registerTaskCommands(context, service as MehrhofProjectService);

      (vscode.window.showInputBox as ReturnType<typeof mock>) = mock(() =>
        Promise.resolve('Add unit tests')
      );

      const handler = registeredCommands.get('mehrhof.quick');
      await handler!();

      expect(client.executeCommand).toHaveBeenCalledWith({
        command: 'quick',
        args: ['Add unit tests'],
      });
    });
  });

  describe('mehrhof.deleteTask command', () => {
    test('calls deleteQueueTask after confirmation', async () => {
      const { service, client } = createMockProjectService();
      registerTaskCommands(context, service as MehrhofProjectService);

      (vscode.window.showInputBox as ReturnType<typeof mock>) = mock(() =>
        Promise.resolve('quick-tasks/task-1')
      );
      (vscode.window.showWarningMessage as ReturnType<typeof mock>) = mock(() =>
        Promise.resolve('Delete')
      );

      const handler = registeredCommands.get('mehrhof.deleteTask');
      await handler!();

      expect(client.deleteQueueTask).toHaveBeenCalledWith('quick-tasks', 'task-1');
    });

    test('shows error for invalid task reference', async () => {
      const { service, client } = createMockProjectService();
      registerTaskCommands(context, service as MehrhofProjectService);

      (vscode.window.showInputBox as ReturnType<typeof mock>) = mock(() =>
        Promise.resolve('invalid-ref')
      );
      (vscode.window.showWarningMessage as ReturnType<typeof mock>) = mock(() =>
        Promise.resolve('Delete')
      );

      let errorMessage: string | undefined;
      (vscode.window.showErrorMessage as ReturnType<typeof mock>) = mock((msg: string) => {
        errorMessage = msg;
        return Promise.resolve(undefined);
      });

      const handler = registeredCommands.get('mehrhof.deleteTask');
      await handler!();

      expect(client.deleteQueueTask).not.toHaveBeenCalled();
      expect(errorMessage).toContain('Invalid task reference');
    });
  });

  describe('mehrhof.exportTask command', () => {
    test('calls exportQueueTask and opens document', async () => {
      const { service, client } = createMockProjectService();
      registerTaskCommands(context, service as MehrhofProjectService);

      (vscode.window.showInputBox as ReturnType<typeof mock>) = mock(() =>
        Promise.resolve('quick-tasks/task-1')
      );

      let openedContent: string | undefined;
      (vscode.workspace.openTextDocument as ReturnType<typeof mock>) = mock(
        (options: { content: string; language: string }) => {
          openedContent = options.content;
          return Promise.resolve({ getText: () => options.content });
        }
      );
      (vscode.window.showTextDocument as ReturnType<typeof mock>) = mock(() => Promise.resolve({}));

      const handler = registeredCommands.get('mehrhof.exportTask');
      await handler!();

      expect(client.exportQueueTask).toHaveBeenCalledWith('quick-tasks', 'task-1');
      expect(openedContent).toContain('# Exported Task');
    });
  });

  describe('mehrhof.optimizeTask command', () => {
    test('calls optimizeQueueTask and shows result', async () => {
      const { service, client } = createMockProjectService();
      registerTaskCommands(context, service as MehrhofProjectService);

      (vscode.window.showInputBox as ReturnType<typeof mock>) = mock(() =>
        Promise.resolve('quick-tasks/task-1')
      );

      let infoMessage: string | undefined;
      (vscode.window.showInformationMessage as ReturnType<typeof mock>) = mock((msg: string) => {
        infoMessage = msg;
        return Promise.resolve(undefined);
      });

      const handler = registeredCommands.get('mehrhof.optimizeTask');
      await handler!();

      expect(client.optimizeQueueTask).toHaveBeenCalledWith('quick-tasks', 'task-1');
      expect(infoMessage).toContain('Fix authentication bug');
      expect(infoMessage).toContain('bug, auth');
    });
  });

  describe('mehrhof.submitTask command', () => {
    test('calls submitQueueTask with provider', async () => {
      const { service, client } = createMockProjectService();
      registerTaskCommands(context, service as MehrhofProjectService);

      let inputCount = 0;
      (vscode.window.showInputBox as ReturnType<typeof mock>) = mock(() => {
        inputCount++;
        if (inputCount === 1) return Promise.resolve('quick-tasks/task-1');
        return Promise.resolve('github');
      });

      let infoMessage: string | undefined;
      (vscode.window.showInformationMessage as ReturnType<typeof mock>) = mock((msg: string) => {
        infoMessage = msg;
        return Promise.resolve(undefined);
      });

      const handler = registeredCommands.get('mehrhof.submitTask');
      await handler!();

      expect(client.submitQueueTask).toHaveBeenCalledWith('quick-tasks', 'task-1', 'github');
      expect(infoMessage).toContain('GH-123');
    });
  });

  describe('mehrhof.syncTask command', () => {
    test('shows warning when no active task', async () => {
      const { service } = createMockProjectService(true, false);
      registerTaskCommands(context, service as MehrhofProjectService);

      let warningMessage: string | undefined;
      (vscode.window.showWarningMessage as ReturnType<typeof mock>) = mock((msg: string) => {
        warningMessage = msg;
        return Promise.resolve(undefined);
      });

      const handler = registeredCommands.get('mehrhof.syncTask');
      await handler!();

      expect(warningMessage).toContain('No active task');
    });

    test('calls syncTask API when task is active', async () => {
      const { service, client } = createMockProjectService(true, true);
      registerTaskCommands(context, service as MehrhofProjectService);

      const handler = registeredCommands.get('mehrhof.syncTask');
      await handler!();

      expect(client.syncTask).toHaveBeenCalled();
    });
  });
});
