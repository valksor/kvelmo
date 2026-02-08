import { describe, test, expect, beforeEach, afterEach, mock } from 'bun:test';
import * as vscode from 'vscode';
import { registerProjectCommands } from '../../src/commands/project';
import {
  createMockExtensionContext,
  resetMocks,
  registeredCommands,
  type MockExtensionContext,
} from '../helpers/mockVscode';
import {
  createInteractiveCommandResponse,
  resetFactories,
  saveFetch,
  restoreFetch,
} from '../helpers/factories';
import type { MehrhofProjectService } from '../../src/services/projectService';
import type { MehrhofApiClient } from '../../src/api/client';

// Create a mock API client
function createMockApiClient(): Partial<MehrhofApiClient> {
  return {
    executeCommand: mock(() => Promise.resolve(createInteractiveCommandResponse())),
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
  };
  return { service, client };
}

describe('Project Commands Test Suite', () => {
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

  describe('registerProjectCommands', () => {
    test('registers all expected commands', () => {
      const { service } = createMockProjectService();
      registerProjectCommands(context, service as MehrhofProjectService);

      const expectedCommands = [
        'mehrhof.projectPlan',
        'mehrhof.projectTasks',
        'mehrhof.projectEdit',
        'mehrhof.projectSubmit',
        'mehrhof.projectStart',
        'mehrhof.projectSync',
        'mehrhof.stackList',
        'mehrhof.stackRebase',
        'mehrhof.stackSync',
        'mehrhof.configValidate',
        'mehrhof.agentsList',
        'mehrhof.agentsExplain',
        'mehrhof.providersList',
        'mehrhof.providersInfo',
        'mehrhof.templatesList',
        'mehrhof.templatesShow',
        'mehrhof.scan',
        'mehrhof.commit',
      ];

      for (const cmd of expectedCommands) {
        expect(registeredCommands.has(cmd)).toBe(true);
      }
    });

    test('adds disposables to context subscriptions', () => {
      const { service } = createMockProjectService();
      registerProjectCommands(context, service as MehrhofProjectService);
      expect(context.subscriptions.length).toBe(18);
    });
  });

  describe('Project Commands', () => {
    describe('mehrhof.projectPlan', () => {
      test('returns early when not connected', async () => {
        const { service } = createMockProjectService(false);
        registerProjectCommands(context, service as MehrhofProjectService);

        const handler = registeredCommands.get('mehrhof.projectPlan');
        await handler!();
      });

      test('calls executeCommand with source', async () => {
        const { service, client } = createMockProjectService();
        registerProjectCommands(context, service as MehrhofProjectService);

        (vscode.window.showInputBox as ReturnType<typeof mock>) = mock(() =>
          Promise.resolve('./roadmap.md')
        );

        const handler = registeredCommands.get('mehrhof.projectPlan');
        await handler!();

        expect(client.executeCommand).toHaveBeenCalledWith({
          command: 'project',
          args: ['plan', './roadmap.md'],
        });
      });

      test('returns early when user cancels input', async () => {
        const { service, client } = createMockProjectService();
        registerProjectCommands(context, service as MehrhofProjectService);

        (vscode.window.showInputBox as ReturnType<typeof mock>) = mock(() =>
          Promise.resolve(undefined)
        );

        const handler = registeredCommands.get('mehrhof.projectPlan');
        await handler!();

        expect(client.executeCommand).not.toHaveBeenCalled();
      });
    });

    describe('mehrhof.projectTasks', () => {
      test('calls executeCommand with tasks', async () => {
        const { service, client } = createMockProjectService();
        registerProjectCommands(context, service as MehrhofProjectService);

        const handler = registeredCommands.get('mehrhof.projectTasks');
        await handler!();

        expect(client.executeCommand).toHaveBeenCalledWith({
          command: 'project',
          args: ['tasks'],
        });
      });
    });

    describe('mehrhof.projectEdit', () => {
      test('calls executeCommand with edit params', async () => {
        const { service, client } = createMockProjectService();
        registerProjectCommands(context, service as MehrhofProjectService);

        let inputCount = 0;
        (vscode.window.showInputBox as ReturnType<typeof mock>) = mock(() => {
          inputCount++;
          if (inputCount === 1) return Promise.resolve('task-1');
          return Promise.resolve('New Title');
        });
        (vscode.window.showQuickPick as ReturnType<typeof mock>) = mock(() =>
          Promise.resolve({ label: 'title', description: 'Edit task title' })
        );

        const handler = registeredCommands.get('mehrhof.projectEdit');
        await handler!();

        expect(client.executeCommand).toHaveBeenCalledWith({
          command: 'project',
          args: ['edit', 'task-1', '--title', 'New Title'],
        });
      });

      test('returns early when user cancels task ID', async () => {
        const { service, client } = createMockProjectService();
        registerProjectCommands(context, service as MehrhofProjectService);

        (vscode.window.showInputBox as ReturnType<typeof mock>) = mock(() =>
          Promise.resolve(undefined)
        );

        const handler = registeredCommands.get('mehrhof.projectEdit');
        await handler!();

        expect(client.executeCommand).not.toHaveBeenCalled();
      });

      test('returns early when user cancels field selection', async () => {
        const { service, client } = createMockProjectService();
        registerProjectCommands(context, service as MehrhofProjectService);

        (vscode.window.showInputBox as ReturnType<typeof mock>) = mock(() =>
          Promise.resolve('task-1')
        );
        (vscode.window.showQuickPick as ReturnType<typeof mock>) = mock(() =>
          Promise.resolve(undefined)
        );

        const handler = registeredCommands.get('mehrhof.projectEdit');
        await handler!();

        expect(client.executeCommand).not.toHaveBeenCalled();
      });
    });

    describe('mehrhof.projectSubmit', () => {
      test('calls executeCommand with provider', async () => {
        const { service, client } = createMockProjectService();
        registerProjectCommands(context, service as MehrhofProjectService);

        (vscode.window.showQuickPick as ReturnType<typeof mock>) = mock(() =>
          Promise.resolve({ label: 'github', description: 'Submit to GitHub Issues' })
        );

        const handler = registeredCommands.get('mehrhof.projectSubmit');
        await handler!();

        expect(client.executeCommand).toHaveBeenCalledWith({
          command: 'project',
          args: ['submit', 'github'],
        });
      });
    });

    describe('mehrhof.projectStart', () => {
      test('calls executeCommand with start', async () => {
        const { service, client } = createMockProjectService();
        registerProjectCommands(context, service as MehrhofProjectService);

        const handler = registeredCommands.get('mehrhof.projectStart');
        await handler!();

        expect(client.executeCommand).toHaveBeenCalledWith({
          command: 'project',
          args: ['start'],
        });
      });
    });

    describe('mehrhof.projectSync', () => {
      test('calls executeCommand with sync reference', async () => {
        const { service, client } = createMockProjectService();
        registerProjectCommands(context, service as MehrhofProjectService);

        (vscode.window.showInputBox as ReturnType<typeof mock>) = mock(() =>
          Promise.resolve('owner/repo#123')
        );

        const handler = registeredCommands.get('mehrhof.projectSync');
        await handler!();

        expect(client.executeCommand).toHaveBeenCalledWith({
          command: 'project',
          args: ['sync', 'owner/repo#123'],
        });
      });
    });
  });

  describe('Stack Commands', () => {
    describe('mehrhof.stackList', () => {
      test('calls executeCommand with stack list', async () => {
        const { service, client } = createMockProjectService();
        registerProjectCommands(context, service as MehrhofProjectService);

        const handler = registeredCommands.get('mehrhof.stackList');
        await handler!();

        expect(client.executeCommand).toHaveBeenCalledWith({
          command: 'stack',
          args: ['list'],
        });
      });
    });

    describe('mehrhof.stackRebase', () => {
      test('calls executeCommand with rebase for specific task', async () => {
        const { service, client } = createMockProjectService();
        registerProjectCommands(context, service as MehrhofProjectService);

        (vscode.window.showInputBox as ReturnType<typeof mock>) = mock(() =>
          Promise.resolve('task-1')
        );

        const handler = registeredCommands.get('mehrhof.stackRebase');
        await handler!();

        expect(client.executeCommand).toHaveBeenCalledWith({
          command: 'stack',
          args: ['rebase', 'task-1'],
        });
      });

      test('calls executeCommand with rebase all when empty', async () => {
        const { service, client } = createMockProjectService();
        registerProjectCommands(context, service as MehrhofProjectService);

        (vscode.window.showInputBox as ReturnType<typeof mock>) = mock(() => Promise.resolve(''));

        const handler = registeredCommands.get('mehrhof.stackRebase');
        await handler!();

        expect(client.executeCommand).toHaveBeenCalledWith({
          command: 'stack',
          args: ['rebase'],
        });
      });
    });

    describe('mehrhof.stackSync', () => {
      test('calls executeCommand with stack sync', async () => {
        const { service, client } = createMockProjectService();
        registerProjectCommands(context, service as MehrhofProjectService);

        const handler = registeredCommands.get('mehrhof.stackSync');
        await handler!();

        expect(client.executeCommand).toHaveBeenCalledWith({
          command: 'stack',
          args: ['sync'],
        });
      });
    });
  });

  describe('Configuration Commands', () => {
    describe('mehrhof.configValidate', () => {
      test('calls executeCommand with config validate', async () => {
        const { service, client } = createMockProjectService();
        registerProjectCommands(context, service as MehrhofProjectService);

        const handler = registeredCommands.get('mehrhof.configValidate');
        await handler!();

        expect(client.executeCommand).toHaveBeenCalledWith({
          command: 'config',
          args: ['validate'],
        });
      });
    });

    describe('mehrhof.agentsList', () => {
      test('calls executeCommand with agents list', async () => {
        const { service, client } = createMockProjectService();
        registerProjectCommands(context, service as MehrhofProjectService);

        const handler = registeredCommands.get('mehrhof.agentsList');
        await handler!();

        expect(client.executeCommand).toHaveBeenCalledWith({
          command: 'agents',
          args: ['list'],
        });
      });
    });

    describe('mehrhof.agentsExplain', () => {
      test('calls executeCommand with agent name', async () => {
        const { service, client } = createMockProjectService();
        registerProjectCommands(context, service as MehrhofProjectService);

        (vscode.window.showInputBox as ReturnType<typeof mock>) = mock(() =>
          Promise.resolve('claude')
        );

        const handler = registeredCommands.get('mehrhof.agentsExplain');
        await handler!();

        expect(client.executeCommand).toHaveBeenCalledWith({
          command: 'agents',
          args: ['explain', 'claude'],
        });
      });
    });

    describe('mehrhof.providersList', () => {
      test('calls executeCommand with providers list', async () => {
        const { service, client } = createMockProjectService();
        registerProjectCommands(context, service as MehrhofProjectService);

        const handler = registeredCommands.get('mehrhof.providersList');
        await handler!();

        expect(client.executeCommand).toHaveBeenCalledWith({
          command: 'providers',
          args: ['list'],
        });
      });
    });

    describe('mehrhof.providersInfo', () => {
      test('calls executeCommand with provider name', async () => {
        const { service, client } = createMockProjectService();
        registerProjectCommands(context, service as MehrhofProjectService);

        (vscode.window.showInputBox as ReturnType<typeof mock>) = mock(() =>
          Promise.resolve('github')
        );

        const handler = registeredCommands.get('mehrhof.providersInfo');
        await handler!();

        expect(client.executeCommand).toHaveBeenCalledWith({
          command: 'providers',
          args: ['info', 'github'],
        });
      });
    });

    describe('mehrhof.templatesList', () => {
      test('calls executeCommand with templates list', async () => {
        const { service, client } = createMockProjectService();
        registerProjectCommands(context, service as MehrhofProjectService);

        const handler = registeredCommands.get('mehrhof.templatesList');
        await handler!();

        expect(client.executeCommand).toHaveBeenCalledWith({
          command: 'templates',
          args: ['list'],
        });
      });
    });

    describe('mehrhof.templatesShow', () => {
      test('calls executeCommand with template name', async () => {
        const { service, client } = createMockProjectService();
        registerProjectCommands(context, service as MehrhofProjectService);

        (vscode.window.showInputBox as ReturnType<typeof mock>) = mock(() =>
          Promise.resolve('bug-fix')
        );

        const handler = registeredCommands.get('mehrhof.templatesShow');
        await handler!();

        expect(client.executeCommand).toHaveBeenCalledWith({
          command: 'templates',
          args: ['show', 'bug-fix'],
        });
      });
    });
  });

  describe('Utility Commands', () => {
    describe('mehrhof.scan', () => {
      test('calls executeCommand with scan', async () => {
        const { service, client } = createMockProjectService();
        registerProjectCommands(context, service as MehrhofProjectService);

        const handler = registeredCommands.get('mehrhof.scan');
        await handler!();

        expect(client.executeCommand).toHaveBeenCalledWith({
          command: 'scan',
          args: [],
        });
      });
    });

    describe('mehrhof.commit', () => {
      test('calls executeCommand with commit', async () => {
        const { service, client } = createMockProjectService();
        registerProjectCommands(context, service as MehrhofProjectService);

        const handler = registeredCommands.get('mehrhof.commit');
        await handler!();

        expect(client.executeCommand).toHaveBeenCalledWith({
          command: 'commit',
          args: [],
        });
      });
    });
  });

  describe('Error Handling', () => {
    test('shows error when command fails', async () => {
      const { service, client } = createMockProjectService();
      (client.executeCommand as ReturnType<typeof mock>) = mock(() =>
        Promise.resolve(createInteractiveCommandResponse({ success: false, error: 'Failed' }))
      );
      registerProjectCommands(context, service as MehrhofProjectService);

      let errorMessage: string | undefined;
      (vscode.window.showErrorMessage as ReturnType<typeof mock>) = mock((msg: string) => {
        errorMessage = msg;
        return Promise.resolve(undefined);
      });

      const handler = registeredCommands.get('mehrhof.projectTasks');
      await handler!();

      expect(errorMessage).toContain('Failed');
    });
  });
});
