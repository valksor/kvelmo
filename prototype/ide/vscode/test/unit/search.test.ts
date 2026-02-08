import { describe, test, expect, beforeEach, afterEach, mock } from 'bun:test';
import * as vscode from 'vscode';
import { registerSearchCommands } from '../../src/commands/search';
import {
  createMockExtensionContext,
  resetMocks,
  registeredCommands,
  type MockExtensionContext,
} from '../helpers/mockVscode';
import {
  createFindSearchResponse,
  createMemorySearchResponse,
  createMemoryIndexResponse,
  createMemoryStatsResponse,
  createLibraryListResponse,
  createLibraryShowResponse,
  createLibraryStatsResponse,
  createLinksListResponse,
  createLinksSearchResponse,
  createLinksStatsResponse,
  createInteractiveCommandResponse,
  resetFactories,
  saveFetch,
  restoreFetch,
} from '../helpers/factories';
import type { MehrhofProjectService } from '../../src/services/projectService';
import type { MehrhofApiClient } from '../../src/api/client';

// Create a mock API client with all methods
function createMockApiClient(): Partial<MehrhofApiClient> {
  return {
    find: mock(() => Promise.resolve(createFindSearchResponse())),
    memorySearch: mock(() => Promise.resolve(createMemorySearchResponse())),
    memoryIndex: mock(() => Promise.resolve(createMemoryIndexResponse())),
    memoryStats: mock(() => Promise.resolve(createMemoryStatsResponse())),
    libraryList: mock(() => Promise.resolve(createLibraryListResponse())),
    libraryShow: mock(() => Promise.resolve(createLibraryShowResponse())),
    libraryPull: mock(() => Promise.resolve(createInteractiveCommandResponse())),
    libraryRemove: mock(() => Promise.resolve(createInteractiveCommandResponse())),
    libraryStats: mock(() => Promise.resolve(createLibraryStatsResponse())),
    linksList: mock(() => Promise.resolve(createLinksListResponse())),
    linksSearch: mock(() => Promise.resolve(createLinksSearchResponse())),
    linksStats: mock(() => Promise.resolve(createLinksStatsResponse())),
    linksRebuild: mock(() => Promise.resolve(createInteractiveCommandResponse())),
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

describe('Search Commands Test Suite', () => {
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

  describe('registerSearchCommands', () => {
    test('registers all expected commands', () => {
      const { service } = createMockProjectService();
      registerSearchCommands(context, service as MehrhofProjectService);

      const expectedCommands = [
        'mehrhof.find',
        'mehrhof.memorySearch',
        'mehrhof.memoryIndex',
        'mehrhof.memoryStats',
        'mehrhof.libraryList',
        'mehrhof.libraryShow',
        'mehrhof.libraryPull',
        'mehrhof.libraryRemove',
        'mehrhof.libraryStats',
        'mehrhof.linksList',
        'mehrhof.linksSearch',
        'mehrhof.linksStats',
        'mehrhof.linksRebuild',
      ];

      for (const cmd of expectedCommands) {
        expect(registeredCommands.has(cmd)).toBe(true);
      }
    });

    test('adds disposables to context subscriptions', () => {
      const { service } = createMockProjectService();
      registerSearchCommands(context, service as MehrhofProjectService);
      expect(context.subscriptions.length).toBe(13);
    });
  });

  describe('mehrhof.find command', () => {
    test('returns early when not connected', async () => {
      const { service } = createMockProjectService(false);
      registerSearchCommands(context, service as MehrhofProjectService);

      const handler = registeredCommands.get('mehrhof.find');
      expect(handler).toBeDefined();

      await handler!();
      // Should not throw, just return early
    });

    test('returns early when user cancels input', async () => {
      const { service, client } = createMockProjectService();
      registerSearchCommands(context, service as MehrhofProjectService);

      // Mock showInputBox to return undefined (cancelled)
      (vscode.window.showInputBox as ReturnType<typeof mock>) = mock(() =>
        Promise.resolve(undefined)
      );

      const handler = registeredCommands.get('mehrhof.find');
      await handler!();

      // API should not be called
      expect(client.find).not.toHaveBeenCalled();
    });

    test('calls find API with query', async () => {
      const { service, client } = createMockProjectService();
      registerSearchCommands(context, service as MehrhofProjectService);

      // Mock user input
      (vscode.window.showInputBox as ReturnType<typeof mock>) = mock(() => Promise.resolve('TODO'));
      (vscode.window.showQuickPick as ReturnType<typeof mock>) = mock(() =>
        Promise.resolve(undefined)
      );

      const handler = registeredCommands.get('mehrhof.find');
      await handler!();

      expect(client.find).toHaveBeenCalledWith('TODO');
    });

    test('shows no matches message when count is 0', async () => {
      const { service, client } = createMockProjectService();
      (client.find as ReturnType<typeof mock>) = mock(() =>
        Promise.resolve(createFindSearchResponse({ count: 0, matches: [] }))
      );
      registerSearchCommands(context, service as MehrhofProjectService);

      (vscode.window.showInputBox as ReturnType<typeof mock>) = mock(() =>
        Promise.resolve('nonexistent')
      );
      let infoMessage: string | undefined;
      (vscode.window.showInformationMessage as ReturnType<typeof mock>) = mock((msg: string) => {
        infoMessage = msg;
        return Promise.resolve(undefined);
      });

      const handler = registeredCommands.get('mehrhof.find');
      await handler!();

      expect(infoMessage).toBe('No matches found');
    });

    test('opens file when user selects a match', async () => {
      const { service, client } = createMockProjectService();
      (client.find as ReturnType<typeof mock>) = mock(() =>
        Promise.resolve(
          createFindSearchResponse({
            count: 1,
            matches: [{ file: '/path/to/file.ts', line: 10, snippet: 'test code' }],
          })
        )
      );
      registerSearchCommands(context, service as MehrhofProjectService);

      (vscode.window.showInputBox as ReturnType<typeof mock>) = mock(() => Promise.resolve('test'));
      (vscode.window.showQuickPick as ReturnType<typeof mock>) = mock(() =>
        Promise.resolve({ file: '/path/to/file.ts', line: 10 })
      );

      let openedUri: string | undefined;
      (vscode.workspace.openTextDocument as ReturnType<typeof mock>) = mock(
        (uri: { fsPath: string }) => {
          openedUri = uri.fsPath;
          return Promise.resolve({ getText: () => '', lineCount: 100, uri });
        }
      );
      (vscode.window.showTextDocument as ReturnType<typeof mock>) = mock(() => Promise.resolve({}));

      const handler = registeredCommands.get('mehrhof.find');
      await handler!();

      expect(openedUri).toBe('/path/to/file.ts');
    });
  });

  describe('mehrhof.memorySearch command', () => {
    test('calls memorySearch API with query', async () => {
      const { service, client } = createMockProjectService();
      registerSearchCommands(context, service as MehrhofProjectService);

      (vscode.window.showInputBox as ReturnType<typeof mock>) = mock(() =>
        Promise.resolve('authentication')
      );
      (vscode.window.showQuickPick as ReturnType<typeof mock>) = mock(() =>
        Promise.resolve(undefined)
      );

      const handler = registeredCommands.get('mehrhof.memorySearch');
      await handler!();

      expect(client.memorySearch).toHaveBeenCalledWith('authentication');
    });

    test('shows no results message when count is 0', async () => {
      const { service, client } = createMockProjectService();
      (client.memorySearch as ReturnType<typeof mock>) = mock(() =>
        Promise.resolve(createMemorySearchResponse({ count: 0, results: [] }))
      );
      registerSearchCommands(context, service as MehrhofProjectService);

      (vscode.window.showInputBox as ReturnType<typeof mock>) = mock(() =>
        Promise.resolve('nonexistent')
      );
      let infoMessage: string | undefined;
      (vscode.window.showInformationMessage as ReturnType<typeof mock>) = mock((msg: string) => {
        infoMessage = msg;
        return Promise.resolve(undefined);
      });

      const handler = registeredCommands.get('mehrhof.memorySearch');
      await handler!();

      expect(infoMessage).toBe('No similar tasks found');
    });
  });

  describe('mehrhof.memoryIndex command', () => {
    test('calls memoryIndex API with task ID', async () => {
      const { service, client } = createMockProjectService();
      registerSearchCommands(context, service as MehrhofProjectService);

      (vscode.window.showInputBox as ReturnType<typeof mock>) = mock(() =>
        Promise.resolve('task-123')
      );

      const handler = registeredCommands.get('mehrhof.memoryIndex');
      await handler!();

      expect(client.memoryIndex).toHaveBeenCalledWith('task-123');
    });

    test('shows success message', async () => {
      const { service } = createMockProjectService();
      registerSearchCommands(context, service as MehrhofProjectService);

      (vscode.window.showInputBox as ReturnType<typeof mock>) = mock(() =>
        Promise.resolve('task-123')
      );
      let infoMessage: string | undefined;
      (vscode.window.showInformationMessage as ReturnType<typeof mock>) = mock((msg: string) => {
        infoMessage = msg;
        return Promise.resolve(undefined);
      });

      const handler = registeredCommands.get('mehrhof.memoryIndex');
      await handler!();

      expect(infoMessage).toBe('Indexed 5 documents');
    });

    test('shows error message when API fails', async () => {
      const { service, client } = createMockProjectService();
      (client.memoryIndex as ReturnType<typeof mock>) = mock(() =>
        Promise.resolve(createMemoryIndexResponse({ success: false, error: 'Index failed' }))
      );
      registerSearchCommands(context, service as MehrhofProjectService);

      (vscode.window.showInputBox as ReturnType<typeof mock>) = mock(() =>
        Promise.resolve('task-123')
      );
      let errorMessage: string | undefined;
      (vscode.window.showErrorMessage as ReturnType<typeof mock>) = mock((msg: string) => {
        errorMessage = msg;
        return Promise.resolve(undefined);
      });

      const handler = registeredCommands.get('mehrhof.memoryIndex');
      await handler!();

      expect(errorMessage).toContain('Index failed');
    });
  });

  describe('mehrhof.memoryStats command', () => {
    test('shows stats in modal', async () => {
      const { service } = createMockProjectService();
      registerSearchCommands(context, service as MehrhofProjectService);

      let infoMessage: string | undefined;
      let modalOptions: { modal?: boolean } | undefined;
      (vscode.window.showInformationMessage as ReturnType<typeof mock>) = mock(
        (msg: string, options?: { modal?: boolean }) => {
          infoMessage = msg;
          modalOptions = options;
          return Promise.resolve(undefined);
        }
      );

      const handler = registeredCommands.get('mehrhof.memoryStats');
      await handler!();

      expect(infoMessage).toContain('Total documents: 100');
      expect(modalOptions?.modal).toBe(true);
    });

    test('shows disabled message when memory is not enabled', async () => {
      const { service, client } = createMockProjectService();
      (client.memoryStats as ReturnType<typeof mock>) = mock(() =>
        Promise.resolve(createMemoryStatsResponse({ enabled: false }))
      );
      registerSearchCommands(context, service as MehrhofProjectService);

      let infoMessage: string | undefined;
      (vscode.window.showInformationMessage as ReturnType<typeof mock>) = mock((msg: string) => {
        infoMessage = msg;
        return Promise.resolve(undefined);
      });

      const handler = registeredCommands.get('mehrhof.memoryStats');
      await handler!();

      expect(infoMessage).toBe('Memory system is not enabled');
    });
  });

  describe('mehrhof.libraryList command', () => {
    test('calls libraryList API', async () => {
      const { service, client } = createMockProjectService();
      registerSearchCommands(context, service as MehrhofProjectService);

      (vscode.window.showQuickPick as ReturnType<typeof mock>) = mock(() =>
        Promise.resolve(undefined)
      );

      const handler = registeredCommands.get('mehrhof.libraryList');
      await handler!();

      expect(client.libraryList).toHaveBeenCalled();
    });

    test('shows no collections message when empty', async () => {
      const { service, client } = createMockProjectService();
      (client.libraryList as ReturnType<typeof mock>) = mock(() =>
        Promise.resolve(createLibraryListResponse({ count: 0, collections: [] }))
      );
      registerSearchCommands(context, service as MehrhofProjectService);

      let infoMessage: string | undefined;
      (vscode.window.showInformationMessage as ReturnType<typeof mock>) = mock((msg: string) => {
        infoMessage = msg;
        return Promise.resolve(undefined);
      });

      const handler = registeredCommands.get('mehrhof.libraryList');
      await handler!();

      expect(infoMessage).toBe('No library collections');
    });
  });

  describe('mehrhof.libraryShow command', () => {
    test('calls libraryShow API with name', async () => {
      const { service, client } = createMockProjectService();
      registerSearchCommands(context, service as MehrhofProjectService);

      (vscode.window.showInputBox as ReturnType<typeof mock>) = mock(() =>
        Promise.resolve('my-collection')
      );

      const handler = registeredCommands.get('mehrhof.libraryShow');
      await handler!();

      expect(client.libraryShow).toHaveBeenCalledWith('my-collection');
    });
  });

  describe('mehrhof.libraryPull command', () => {
    test('calls libraryPull API with source', async () => {
      const { service, client } = createMockProjectService();
      registerSearchCommands(context, service as MehrhofProjectService);

      (vscode.window.showInputBox as ReturnType<typeof mock>) = mock(() =>
        Promise.resolve('https://docs.example.com')
      );

      const handler = registeredCommands.get('mehrhof.libraryPull');
      await handler!();

      expect(client.libraryPull).toHaveBeenCalledWith('https://docs.example.com');
    });
  });

  describe('mehrhof.libraryRemove command', () => {
    test('calls libraryRemove API after confirmation', async () => {
      const { service, client } = createMockProjectService();
      registerSearchCommands(context, service as MehrhofProjectService);

      (vscode.window.showInputBox as ReturnType<typeof mock>) = mock(() =>
        Promise.resolve('old-collection')
      );
      (vscode.window.showWarningMessage as ReturnType<typeof mock>) = mock(() =>
        Promise.resolve('Remove')
      );

      const handler = registeredCommands.get('mehrhof.libraryRemove');
      await handler!();

      expect(client.libraryRemove).toHaveBeenCalledWith('old-collection');
    });

    test('does not remove when user cancels', async () => {
      const { service, client } = createMockProjectService();
      registerSearchCommands(context, service as MehrhofProjectService);

      (vscode.window.showInputBox as ReturnType<typeof mock>) = mock(() =>
        Promise.resolve('old-collection')
      );
      (vscode.window.showWarningMessage as ReturnType<typeof mock>) = mock(() =>
        Promise.resolve(undefined)
      );

      const handler = registeredCommands.get('mehrhof.libraryRemove');
      await handler!();

      expect(client.libraryRemove).not.toHaveBeenCalled();
    });
  });

  describe('mehrhof.libraryStats command', () => {
    test('shows stats in modal', async () => {
      const { service } = createMockProjectService();
      registerSearchCommands(context, service as MehrhofProjectService);

      let infoMessage: string | undefined;
      (vscode.window.showInformationMessage as ReturnType<typeof mock>) = mock((msg: string) => {
        infoMessage = msg;
        return Promise.resolve(undefined);
      });

      const handler = registeredCommands.get('mehrhof.libraryStats');
      await handler!();

      expect(infoMessage).toContain('Collections: 5');
    });
  });

  describe('mehrhof.linksList command', () => {
    test('calls linksList API', async () => {
      const { service, client } = createMockProjectService();
      registerSearchCommands(context, service as MehrhofProjectService);

      (vscode.window.showQuickPick as ReturnType<typeof mock>) = mock(() =>
        Promise.resolve(undefined)
      );

      const handler = registeredCommands.get('mehrhof.linksList');
      await handler!();

      expect(client.linksList).toHaveBeenCalled();
    });
  });

  describe('mehrhof.linksSearch command', () => {
    test('calls linksSearch API with query', async () => {
      const { service, client } = createMockProjectService();
      registerSearchCommands(context, service as MehrhofProjectService);

      (vscode.window.showInputBox as ReturnType<typeof mock>) = mock(() =>
        Promise.resolve('cache')
      );
      (vscode.window.showQuickPick as ReturnType<typeof mock>) = mock(() =>
        Promise.resolve(undefined)
      );

      const handler = registeredCommands.get('mehrhof.linksSearch');
      await handler!();

      expect(client.linksSearch).toHaveBeenCalledWith('cache');
    });
  });

  describe('mehrhof.linksStats command', () => {
    test('shows stats in modal', async () => {
      const { service } = createMockProjectService();
      registerSearchCommands(context, service as MehrhofProjectService);

      let infoMessage: string | undefined;
      (vscode.window.showInformationMessage as ReturnType<typeof mock>) = mock((msg: string) => {
        infoMessage = msg;
        return Promise.resolve(undefined);
      });

      const handler = registeredCommands.get('mehrhof.linksStats');
      await handler!();

      expect(infoMessage).toContain('Total links: 50');
    });
  });

  describe('mehrhof.linksRebuild command', () => {
    test('calls linksRebuild API after confirmation', async () => {
      const { service, client } = createMockProjectService();
      registerSearchCommands(context, service as MehrhofProjectService);

      (vscode.window.showWarningMessage as ReturnType<typeof mock>) = mock(() =>
        Promise.resolve('Rebuild')
      );

      const handler = registeredCommands.get('mehrhof.linksRebuild');
      await handler!();

      expect(client.linksRebuild).toHaveBeenCalled();
    });

    test('does not rebuild when user cancels', async () => {
      const { service, client } = createMockProjectService();
      registerSearchCommands(context, service as MehrhofProjectService);

      (vscode.window.showWarningMessage as ReturnType<typeof mock>) = mock(() =>
        Promise.resolve(undefined)
      );

      const handler = registeredCommands.get('mehrhof.linksRebuild');
      await handler!();

      expect(client.linksRebuild).not.toHaveBeenCalled();
    });
  });
});
