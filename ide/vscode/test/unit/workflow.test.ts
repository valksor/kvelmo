import { describe, test, expect, beforeEach, afterEach, mock } from 'bun:test';
import * as vscode from 'vscode';
import { registerWorkflowCommands } from '../../src/commands/workflow';
import {
  createMockExtensionContext,
  resetMocks,
  registeredCommands,
  type MockExtensionContext,
} from '../helpers/mockVscode';
import {
  createWorkflowResponse,
  resetFactories,
  saveFetch,
  restoreFetch,
} from '../helpers/factories';
import type { MehrhofProjectService } from '../../src/services/projectService';
import type { MehrhofApiClient } from '../../src/api/client';
import type { ContinueResponse } from '../../src/api/models';

// Create a mock API client with all workflow methods
function createMockApiClient(): Partial<MehrhofApiClient> {
  return {
    startTask: mock(() => Promise.resolve(createWorkflowResponse())),
    plan: mock(() => Promise.resolve(createWorkflowResponse())),
    implement: mock(() => Promise.resolve(createWorkflowResponse())),
    review: mock(() => Promise.resolve(createWorkflowResponse())),
    continueWorkflow: mock(() =>
      Promise.resolve({
        success: true,
        state: 'implementing',
        next_actions: ['review'],
        message: 'Continuing',
      } as ContinueResponse)
    ),
    finish: mock(() => Promise.resolve(createWorkflowResponse({ message: 'Task finished' }))),
    abandon: mock(() => Promise.resolve(createWorkflowResponse({ message: 'Task abandoned' }))),
    undo: mock(() => Promise.resolve(createWorkflowResponse({ message: 'Reverted' }))),
    redo: mock(() => Promise.resolve(createWorkflowResponse({ message: 'Restored' }))),
  };
}

// Create a mock project service
function createMockProjectService(connected: boolean = true): {
  service: Partial<MehrhofProjectService>;
  client: Partial<MehrhofApiClient>;
} {
  const client = createMockApiClient();
  const service: Partial<MehrhofProjectService> = {
    isConnected: connected,
    client: connected ? (client as MehrhofApiClient) : null,
    currentTask: null,
    startServer: mock(() => Promise.resolve()),
    stopServer: mock(() => {}),
    connect: mock(() => Promise.resolve()),
    disconnect: mock(() => {}),
    refreshState: mock(() => Promise.resolve()),
  };
  return { service, client };
}

describe('Workflow Commands Test Suite', () => {
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

  describe('registerWorkflowCommands', () => {
    test('registers all expected commands', () => {
      const { service } = createMockProjectService();
      registerWorkflowCommands(context, service as MehrhofProjectService);

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
      ];

      for (const cmd of expectedCommands) {
        expect(registeredCommands.has(cmd)).toBe(true);
      }
    });

    test('adds disposables to context subscriptions', () => {
      const { service } = createMockProjectService();
      registerWorkflowCommands(context, service as MehrhofProjectService);
      expect(context.subscriptions.length).toBe(13);
    });
  });

  describe('Server Commands', () => {
    test('mehrhof.startServer calls service.startServer', async () => {
      const { service } = createMockProjectService();
      registerWorkflowCommands(context, service as MehrhofProjectService);

      const handler = registeredCommands.get('mehrhof.startServer');
      await handler!();

      expect(service.startServer).toHaveBeenCalled();
    });

    test('mehrhof.stopServer calls service.stopServer', () => {
      const { service } = createMockProjectService();
      registerWorkflowCommands(context, service as MehrhofProjectService);

      let infoMessage: string | undefined;
      (vscode.window.showInformationMessage as ReturnType<typeof mock>) = mock((msg: string) => {
        infoMessage = msg;
        return Promise.resolve(undefined);
      });

      const handler = registeredCommands.get('mehrhof.stopServer');
      handler!();

      expect(service.stopServer).toHaveBeenCalled();
      expect(infoMessage).toContain('Server stopped');
    });
  });

  describe('Connection Commands', () => {
    test('mehrhof.connect calls service.connect', async () => {
      const { service } = createMockProjectService();
      registerWorkflowCommands(context, service as MehrhofProjectService);

      const handler = registeredCommands.get('mehrhof.connect');
      await handler!();

      expect(service.connect).toHaveBeenCalled();
    });

    test('mehrhof.disconnect calls service.disconnect', () => {
      const { service } = createMockProjectService();
      registerWorkflowCommands(context, service as MehrhofProjectService);

      let infoMessage: string | undefined;
      (vscode.window.showInformationMessage as ReturnType<typeof mock>) = mock((msg: string) => {
        infoMessage = msg;
        return Promise.resolve(undefined);
      });

      const handler = registeredCommands.get('mehrhof.disconnect');
      handler!();

      expect(service.disconnect).toHaveBeenCalled();
      expect(infoMessage).toContain('Disconnected');
    });
  });

  describe('mehrhof.startTask command', () => {
    test('returns early when not connected', async () => {
      const { service } = createMockProjectService(false);
      registerWorkflowCommands(context, service as MehrhofProjectService);

      const handler = registeredCommands.get('mehrhof.startTask');
      await handler!();
      // Should not throw
    });

    test('calls startTask API with ref', async () => {
      const { service, client } = createMockProjectService();
      registerWorkflowCommands(context, service as MehrhofProjectService);

      (vscode.window.showInputBox as ReturnType<typeof mock>) = mock(() =>
        Promise.resolve('github:123')
      );

      const handler = registeredCommands.get('mehrhof.startTask');
      await handler!();

      expect(client.startTask).toHaveBeenCalledWith({ ref: 'github:123' });
      expect(service.refreshState).toHaveBeenCalled();
    });

    test('returns early when user cancels input', async () => {
      const { service, client } = createMockProjectService();
      registerWorkflowCommands(context, service as MehrhofProjectService);

      (vscode.window.showInputBox as ReturnType<typeof mock>) = mock(() =>
        Promise.resolve(undefined)
      );

      const handler = registeredCommands.get('mehrhof.startTask');
      await handler!();

      expect(client.startTask).not.toHaveBeenCalled();
    });

    test('shows error when API fails', async () => {
      const { service, client } = createMockProjectService();
      (client.startTask as ReturnType<typeof mock>) = mock(() =>
        Promise.resolve(createWorkflowResponse({ success: false, error: 'Task not found' }))
      );
      registerWorkflowCommands(context, service as MehrhofProjectService);

      (vscode.window.showInputBox as ReturnType<typeof mock>) = mock(() =>
        Promise.resolve('invalid:ref')
      );
      let errorMessage: string | undefined;
      (vscode.window.showErrorMessage as ReturnType<typeof mock>) = mock((msg: string) => {
        errorMessage = msg;
        return Promise.resolve(undefined);
      });

      const handler = registeredCommands.get('mehrhof.startTask');
      await handler!();

      expect(errorMessage).toContain('Task not found');
    });
  });

  describe('mehrhof.plan command', () => {
    test('returns early when not connected', async () => {
      const { service } = createMockProjectService(false);
      registerWorkflowCommands(context, service as MehrhofProjectService);

      const handler = registeredCommands.get('mehrhof.plan');
      await handler!();
    });

    test('calls plan API', async () => {
      const { service, client } = createMockProjectService();
      registerWorkflowCommands(context, service as MehrhofProjectService);

      const handler = registeredCommands.get('mehrhof.plan');
      await handler!();

      expect(client.plan).toHaveBeenCalled();
    });

    test('shows error when API fails', async () => {
      const { service, client } = createMockProjectService();
      (client.plan as ReturnType<typeof mock>) = mock(() =>
        Promise.resolve(createWorkflowResponse({ success: false, error: 'No active task' }))
      );
      registerWorkflowCommands(context, service as MehrhofProjectService);

      let errorMessage: string | undefined;
      (vscode.window.showErrorMessage as ReturnType<typeof mock>) = mock((msg: string) => {
        errorMessage = msg;
        return Promise.resolve(undefined);
      });

      const handler = registeredCommands.get('mehrhof.plan');
      await handler!();

      expect(errorMessage).toContain('No active task');
    });
  });

  describe('mehrhof.implement command', () => {
    test('calls implement API', async () => {
      const { service, client } = createMockProjectService();
      registerWorkflowCommands(context, service as MehrhofProjectService);

      const handler = registeredCommands.get('mehrhof.implement');
      await handler!();

      expect(client.implement).toHaveBeenCalled();
    });
  });

  describe('mehrhof.review command', () => {
    test('calls review API', async () => {
      const { service, client } = createMockProjectService();
      registerWorkflowCommands(context, service as MehrhofProjectService);

      const handler = registeredCommands.get('mehrhof.review');
      await handler!();

      expect(client.review).toHaveBeenCalled();
    });
  });

  describe('mehrhof.continue command', () => {
    test('calls continueWorkflow API', async () => {
      const { service, client } = createMockProjectService();
      registerWorkflowCommands(context, service as MehrhofProjectService);

      const handler = registeredCommands.get('mehrhof.continue');
      await handler!();

      expect(client.continueWorkflow).toHaveBeenCalled();
    });

    test('shows error when continue fails', async () => {
      const { service, client } = createMockProjectService();
      (client.continueWorkflow as ReturnType<typeof mock>) = mock(() =>
        Promise.resolve({ success: false } as ContinueResponse)
      );
      registerWorkflowCommands(context, service as MehrhofProjectService);

      let errorMessage: string | undefined;
      (vscode.window.showErrorMessage as ReturnType<typeof mock>) = mock((msg: string) => {
        errorMessage = msg;
        return Promise.resolve(undefined);
      });

      const handler = registeredCommands.get('mehrhof.continue');
      await handler!();

      expect(errorMessage).toContain('Continue failed');
    });
  });

  describe('mehrhof.finish command', () => {
    test('calls finish API after confirmation', async () => {
      const { service, client } = createMockProjectService();
      registerWorkflowCommands(context, service as MehrhofProjectService);

      (vscode.window.showQuickPick as ReturnType<typeof mock>) = mock(() => Promise.resolve('Yes'));

      const handler = registeredCommands.get('mehrhof.finish');
      await handler!();

      expect(client.finish).toHaveBeenCalled();
      expect(service.refreshState).toHaveBeenCalled();
    });

    test('does not call finish when user cancels', async () => {
      const { service, client } = createMockProjectService();
      registerWorkflowCommands(context, service as MehrhofProjectService);

      (vscode.window.showQuickPick as ReturnType<typeof mock>) = mock(() => Promise.resolve('No'));

      const handler = registeredCommands.get('mehrhof.finish');
      await handler!();

      expect(client.finish).not.toHaveBeenCalled();
    });
  });

  describe('mehrhof.abandon command', () => {
    test('calls abandon API after confirmation', async () => {
      const { service, client } = createMockProjectService();
      registerWorkflowCommands(context, service as MehrhofProjectService);

      (vscode.window.showWarningMessage as ReturnType<typeof mock>) = mock(() =>
        Promise.resolve('Abandon')
      );

      const handler = registeredCommands.get('mehrhof.abandon');
      await handler!();

      expect(client.abandon).toHaveBeenCalled();
      expect(service.refreshState).toHaveBeenCalled();
    });

    test('does not call abandon when user cancels', async () => {
      const { service, client } = createMockProjectService();
      registerWorkflowCommands(context, service as MehrhofProjectService);

      (vscode.window.showWarningMessage as ReturnType<typeof mock>) = mock(() =>
        Promise.resolve(undefined)
      );

      const handler = registeredCommands.get('mehrhof.abandon');
      await handler!();

      expect(client.abandon).not.toHaveBeenCalled();
    });
  });

  describe('mehrhof.undo command', () => {
    test('calls undo API', async () => {
      const { service, client } = createMockProjectService();
      registerWorkflowCommands(context, service as MehrhofProjectService);

      const handler = registeredCommands.get('mehrhof.undo');
      await handler!();

      expect(client.undo).toHaveBeenCalled();
      expect(service.refreshState).toHaveBeenCalled();
    });

    test('shows error when undo fails', async () => {
      const { service, client } = createMockProjectService();
      (client.undo as ReturnType<typeof mock>) = mock(() =>
        Promise.resolve(createWorkflowResponse({ success: false, error: 'No checkpoint' }))
      );
      registerWorkflowCommands(context, service as MehrhofProjectService);

      let errorMessage: string | undefined;
      (vscode.window.showErrorMessage as ReturnType<typeof mock>) = mock((msg: string) => {
        errorMessage = msg;
        return Promise.resolve(undefined);
      });

      const handler = registeredCommands.get('mehrhof.undo');
      await handler!();

      expect(errorMessage).toContain('No checkpoint');
    });
  });

  describe('mehrhof.redo command', () => {
    test('calls redo API', async () => {
      const { service, client } = createMockProjectService();
      registerWorkflowCommands(context, service as MehrhofProjectService);

      const handler = registeredCommands.get('mehrhof.redo');
      await handler!();

      expect(client.redo).toHaveBeenCalled();
      expect(service.refreshState).toHaveBeenCalled();
    });

    test('shows error when redo fails', async () => {
      const { service, client } = createMockProjectService();
      (client.redo as ReturnType<typeof mock>) = mock(() =>
        Promise.resolve(createWorkflowResponse({ success: false, error: 'No redo available' }))
      );
      registerWorkflowCommands(context, service as MehrhofProjectService);

      let errorMessage: string | undefined;
      (vscode.window.showErrorMessage as ReturnType<typeof mock>) = mock((msg: string) => {
        errorMessage = msg;
        return Promise.resolve(undefined);
      });

      const handler = registeredCommands.get('mehrhof.redo');
      await handler!();

      expect(errorMessage).toContain('No redo available');
    });
  });
});
