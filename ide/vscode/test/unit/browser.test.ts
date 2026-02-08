import { describe, test, expect, beforeEach, afterEach, mock } from 'bun:test';
import * as vscode from 'vscode';
import { registerBrowserCommands } from '../../src/commands/browser';
import {
  createMockExtensionContext,
  resetMocks,
  registeredCommands,
  type MockExtensionContext,
} from '../helpers/mockVscode';
import {
  createBrowserStatusResponse,
  createBrowserTabsResponse,
  resetFactories,
  saveFetch,
  restoreFetch,
} from '../helpers/factories';
import type { MehrhofProjectService } from '../../src/services/projectService';
import type { MehrhofApiClient } from '../../src/api/client';
import type {
  BrowserGotoResponse,
  BrowserNavigateResponse,
  BrowserReloadResponse,
  BrowserScreenshotResponse,
  BrowserClickResponse,
  BrowserTypeResponse,
  BrowserEvalResponse,
  BrowserConsoleResponse,
  BrowserNetworkResponse,
} from '../../src/api/models';

// Create a mock API client with all browser methods
function createMockApiClient(): Partial<MehrhofApiClient> {
  return {
    browserStatus: mock(() => Promise.resolve(createBrowserStatusResponse())),
    browserTabs: mock(() => Promise.resolve(createBrowserTabsResponse())),
    browserGoto: mock(() =>
      Promise.resolve({
        success: true,
        tab: { id: 'tab-1', title: 'Test Page', url: 'https://example.com' },
      } as BrowserGotoResponse)
    ),
    browserNavigate: mock(() =>
      Promise.resolve({ success: true, message: 'Navigated' } as BrowserNavigateResponse)
    ),
    browserReload: mock(() =>
      Promise.resolve({ success: true, message: 'Page reloaded' } as BrowserReloadResponse)
    ),
    browserScreenshot: mock(() =>
      Promise.resolve({
        success: true,
        data: 'base64data',
        format: 'png',
        size: 12345,
      } as BrowserScreenshotResponse)
    ),
    browserClick: mock(() =>
      Promise.resolve({ success: true, selector: '#button' } as BrowserClickResponse)
    ),
    browserType: mock(() =>
      Promise.resolve({ success: true, selector: '#input' } as BrowserTypeResponse)
    ),
    browserEval: mock(() =>
      Promise.resolve({ success: true, result: 'Document Title' } as BrowserEvalResponse)
    ),
    browserConsole: mock(() =>
      Promise.resolve({
        success: true,
        messages: [{ level: 'error', text: 'Test error', timestamp: '2026-01-01T00:00:00Z' }],
        count: 1,
      } as BrowserConsoleResponse)
    ),
    browserNetwork: mock(() =>
      Promise.resolve({
        success: true,
        requests: [
          {
            method: 'GET',
            url: 'https://api.example.com/data',
            status: 200,
            timestamp: '2026-01-01T00:00:00Z',
          },
        ],
        count: 1,
      } as BrowserNetworkResponse)
    ),
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

describe('Browser Commands Test Suite', () => {
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

  describe('registerBrowserCommands', () => {
    test('registers all expected commands', () => {
      const { service } = createMockProjectService();
      registerBrowserCommands(context, service as MehrhofProjectService);

      const expectedCommands = [
        'mehrhof.browserStatus',
        'mehrhof.browserTabs',
        'mehrhof.browserGoto',
        'mehrhof.browserNavigate',
        'mehrhof.browserReload',
        'mehrhof.browserScreenshot',
        'mehrhof.browserClick',
        'mehrhof.browserType',
        'mehrhof.browserEval',
        'mehrhof.browserConsole',
        'mehrhof.browserNetwork',
      ];

      for (const cmd of expectedCommands) {
        expect(registeredCommands.has(cmd)).toBe(true);
      }
    });

    test('adds disposables to context subscriptions', () => {
      const { service } = createMockProjectService();
      registerBrowserCommands(context, service as MehrhofProjectService);
      expect(context.subscriptions.length).toBe(11);
    });
  });

  describe('mehrhof.browserStatus command', () => {
    test('returns early when not connected', async () => {
      const { service } = createMockProjectService(false);
      registerBrowserCommands(context, service as MehrhofProjectService);

      const handler = registeredCommands.get('mehrhof.browserStatus');
      await handler!();
      // Should not throw
    });

    test('shows connected status in modal', async () => {
      const { service } = createMockProjectService();
      registerBrowserCommands(context, service as MehrhofProjectService);

      let infoMessage: string | undefined;
      let modalOptions: { modal?: boolean } | undefined;
      (vscode.window.showInformationMessage as ReturnType<typeof mock>) = mock(
        (msg: string, options?: { modal?: boolean }) => {
          infoMessage = msg;
          modalOptions = options;
          return Promise.resolve(undefined);
        }
      );

      const handler = registeredCommands.get('mehrhof.browserStatus');
      await handler!();

      expect(infoMessage).toContain('Connected to localhost:9222');
      expect(modalOptions?.modal).toBe(true);
    });

    test('shows not connected message when browser is disconnected', async () => {
      const { service, client } = createMockProjectService();
      (client.browserStatus as ReturnType<typeof mock>) = mock(() =>
        Promise.resolve(createBrowserStatusResponse({ connected: false, error: 'Not running' }))
      );
      registerBrowserCommands(context, service as MehrhofProjectService);

      let infoMessage: string | undefined;
      (vscode.window.showInformationMessage as ReturnType<typeof mock>) = mock((msg: string) => {
        infoMessage = msg;
        return Promise.resolve(undefined);
      });

      const handler = registeredCommands.get('mehrhof.browserStatus');
      await handler!();

      expect(infoMessage).toContain('Not connected');
      expect(infoMessage).toContain('Not running');
    });
  });

  describe('mehrhof.browserTabs command', () => {
    test('calls browserTabs API', async () => {
      const { service, client } = createMockProjectService();
      registerBrowserCommands(context, service as MehrhofProjectService);

      (vscode.window.showQuickPick as ReturnType<typeof mock>) = mock(() =>
        Promise.resolve(undefined)
      );

      const handler = registeredCommands.get('mehrhof.browserTabs');
      await handler!();

      expect(client.browserTabs).toHaveBeenCalled();
    });

    test('shows no tabs message when empty', async () => {
      const { service, client } = createMockProjectService();
      (client.browserTabs as ReturnType<typeof mock>) = mock(() =>
        Promise.resolve(createBrowserTabsResponse({ count: 0, tabs: [] }))
      );
      registerBrowserCommands(context, service as MehrhofProjectService);

      let infoMessage: string | undefined;
      (vscode.window.showInformationMessage as ReturnType<typeof mock>) = mock((msg: string) => {
        infoMessage = msg;
        return Promise.resolve(undefined);
      });

      const handler = registeredCommands.get('mehrhof.browserTabs');
      await handler!();

      expect(infoMessage).toBe('No browser tabs open');
    });
  });

  describe('mehrhof.browserGoto command', () => {
    test('calls browserGoto API with URL', async () => {
      const { service, client } = createMockProjectService();
      registerBrowserCommands(context, service as MehrhofProjectService);

      (vscode.window.showInputBox as ReturnType<typeof mock>) = mock(() =>
        Promise.resolve('https://example.com')
      );

      const handler = registeredCommands.get('mehrhof.browserGoto');
      await handler!();

      expect(client.browserGoto).toHaveBeenCalledWith('https://example.com');
    });

    test('returns early when user cancels input', async () => {
      const { service, client } = createMockProjectService();
      registerBrowserCommands(context, service as MehrhofProjectService);

      (vscode.window.showInputBox as ReturnType<typeof mock>) = mock(() =>
        Promise.resolve(undefined)
      );

      const handler = registeredCommands.get('mehrhof.browserGoto');
      await handler!();

      expect(client.browserGoto).not.toHaveBeenCalled();
    });

    test('shows opened message on success', async () => {
      const { service } = createMockProjectService();
      registerBrowserCommands(context, service as MehrhofProjectService);

      (vscode.window.showInputBox as ReturnType<typeof mock>) = mock(() =>
        Promise.resolve('https://example.com')
      );
      let infoMessage: string | undefined;
      (vscode.window.showInformationMessage as ReturnType<typeof mock>) = mock((msg: string) => {
        infoMessage = msg;
        return Promise.resolve(undefined);
      });

      const handler = registeredCommands.get('mehrhof.browserGoto');
      await handler!();

      expect(infoMessage).toContain('Opened: Test Page');
    });
  });

  describe('mehrhof.browserNavigate command', () => {
    test('calls browserNavigate API with URL', async () => {
      const { service, client } = createMockProjectService();
      registerBrowserCommands(context, service as MehrhofProjectService);

      (vscode.window.showInputBox as ReturnType<typeof mock>) = mock(() =>
        Promise.resolve('https://test.com')
      );

      const handler = registeredCommands.get('mehrhof.browserNavigate');
      await handler!();

      expect(client.browserNavigate).toHaveBeenCalledWith('https://test.com');
    });

    test('shows error when navigation fails', async () => {
      const { service, client } = createMockProjectService();
      (client.browserNavigate as ReturnType<typeof mock>) = mock(() =>
        Promise.resolve({ success: false } as BrowserNavigateResponse)
      );
      registerBrowserCommands(context, service as MehrhofProjectService);

      (vscode.window.showInputBox as ReturnType<typeof mock>) = mock(() =>
        Promise.resolve('https://test.com')
      );
      let errorMessage: string | undefined;
      (vscode.window.showErrorMessage as ReturnType<typeof mock>) = mock((msg: string) => {
        errorMessage = msg;
        return Promise.resolve(undefined);
      });

      const handler = registeredCommands.get('mehrhof.browserNavigate');
      await handler!();

      expect(errorMessage).toContain('Navigation failed');
    });
  });

  describe('mehrhof.browserReload command', () => {
    test('calls browserReload API', async () => {
      const { service, client } = createMockProjectService();
      registerBrowserCommands(context, service as MehrhofProjectService);

      const handler = registeredCommands.get('mehrhof.browserReload');
      await handler!();

      expect(client.browserReload).toHaveBeenCalled();
    });

    test('shows success message', async () => {
      const { service } = createMockProjectService();
      registerBrowserCommands(context, service as MehrhofProjectService);

      let infoMessage: string | undefined;
      (vscode.window.showInformationMessage as ReturnType<typeof mock>) = mock((msg: string) => {
        infoMessage = msg;
        return Promise.resolve(undefined);
      });

      const handler = registeredCommands.get('mehrhof.browserReload');
      await handler!();

      expect(infoMessage).toBe('Page reloaded');
    });
  });

  describe('mehrhof.browserScreenshot command', () => {
    test('calls browserScreenshot API', async () => {
      const { service, client } = createMockProjectService();
      registerBrowserCommands(context, service as MehrhofProjectService);

      const handler = registeredCommands.get('mehrhof.browserScreenshot');
      await handler!();

      expect(client.browserScreenshot).toHaveBeenCalled();
    });

    test('shows screenshot info on success', async () => {
      const { service } = createMockProjectService();
      registerBrowserCommands(context, service as MehrhofProjectService);

      let infoMessage: string | undefined;
      (vscode.window.showInformationMessage as ReturnType<typeof mock>) = mock((msg: string) => {
        infoMessage = msg;
        return Promise.resolve(undefined);
      });

      const handler = registeredCommands.get('mehrhof.browserScreenshot');
      await handler!();

      expect(infoMessage).toContain('Screenshot captured');
      expect(infoMessage).toContain('png');
    });

    test('shows error when screenshot fails', async () => {
      const { service, client } = createMockProjectService();
      (client.browserScreenshot as ReturnType<typeof mock>) = mock(() =>
        Promise.resolve({ success: false } as BrowserScreenshotResponse)
      );
      registerBrowserCommands(context, service as MehrhofProjectService);

      let errorMessage: string | undefined;
      (vscode.window.showErrorMessage as ReturnType<typeof mock>) = mock((msg: string) => {
        errorMessage = msg;
        return Promise.resolve(undefined);
      });

      const handler = registeredCommands.get('mehrhof.browserScreenshot');
      await handler!();

      expect(errorMessage).toContain('Screenshot failed');
    });
  });

  describe('mehrhof.browserClick command', () => {
    test('calls browserClick API with selector', async () => {
      const { service, client } = createMockProjectService();
      registerBrowserCommands(context, service as MehrhofProjectService);

      (vscode.window.showInputBox as ReturnType<typeof mock>) = mock(() =>
        Promise.resolve('#submit-btn')
      );

      const handler = registeredCommands.get('mehrhof.browserClick');
      await handler!();

      expect(client.browserClick).toHaveBeenCalledWith('#submit-btn');
    });

    test('shows clicked message on success', async () => {
      const { service } = createMockProjectService();
      registerBrowserCommands(context, service as MehrhofProjectService);

      (vscode.window.showInputBox as ReturnType<typeof mock>) = mock(() =>
        Promise.resolve('#submit-btn')
      );
      let infoMessage: string | undefined;
      (vscode.window.showInformationMessage as ReturnType<typeof mock>) = mock((msg: string) => {
        infoMessage = msg;
        return Promise.resolve(undefined);
      });

      const handler = registeredCommands.get('mehrhof.browserClick');
      await handler!();

      expect(infoMessage).toContain('Clicked');
    });
  });

  describe('mehrhof.browserType command', () => {
    test('calls browserType API with selector and text', async () => {
      const { service, client } = createMockProjectService();
      registerBrowserCommands(context, service as MehrhofProjectService);

      let inputCount = 0;
      (vscode.window.showInputBox as ReturnType<typeof mock>) = mock(() => {
        inputCount++;
        if (inputCount === 1) return Promise.resolve('#search');
        return Promise.resolve('search query');
      });

      const handler = registeredCommands.get('mehrhof.browserType');
      await handler!();

      expect(client.browserType).toHaveBeenCalledWith('#search', 'search query');
    });

    test('returns early when selector is cancelled', async () => {
      const { service, client } = createMockProjectService();
      registerBrowserCommands(context, service as MehrhofProjectService);

      (vscode.window.showInputBox as ReturnType<typeof mock>) = mock(() =>
        Promise.resolve(undefined)
      );

      const handler = registeredCommands.get('mehrhof.browserType');
      await handler!();

      expect(client.browserType).not.toHaveBeenCalled();
    });
  });

  describe('mehrhof.browserEval command', () => {
    test('calls browserEval API with expression', async () => {
      const { service, client } = createMockProjectService();
      registerBrowserCommands(context, service as MehrhofProjectService);

      (vscode.window.showInputBox as ReturnType<typeof mock>) = mock(() =>
        Promise.resolve('document.title')
      );

      const handler = registeredCommands.get('mehrhof.browserEval');
      await handler!();

      expect(client.browserEval).toHaveBeenCalledWith('document.title');
    });

    test('shows result in modal', async () => {
      const { service } = createMockProjectService();
      registerBrowserCommands(context, service as MehrhofProjectService);

      (vscode.window.showInputBox as ReturnType<typeof mock>) = mock(() =>
        Promise.resolve('document.title')
      );
      let infoMessage: string | undefined;
      let modalOptions: { modal?: boolean } | undefined;
      (vscode.window.showInformationMessage as ReturnType<typeof mock>) = mock(
        (msg: string, options?: { modal?: boolean }) => {
          infoMessage = msg;
          modalOptions = options;
          return Promise.resolve(undefined);
        }
      );

      const handler = registeredCommands.get('mehrhof.browserEval');
      await handler!();

      expect(infoMessage).toContain('Result:');
      expect(modalOptions?.modal).toBe(true);
    });
  });

  describe('mehrhof.browserConsole command', () => {
    test('calls browserConsole API', async () => {
      const { service, client } = createMockProjectService();
      registerBrowserCommands(context, service as MehrhofProjectService);

      (vscode.window.showQuickPick as ReturnType<typeof mock>) = mock(() =>
        Promise.resolve(undefined)
      );

      const handler = registeredCommands.get('mehrhof.browserConsole');
      await handler!();

      expect(client.browserConsole).toHaveBeenCalled();
    });

    test('shows no messages when empty', async () => {
      const { service, client } = createMockProjectService();
      (client.browserConsole as ReturnType<typeof mock>) = mock(() =>
        Promise.resolve({ success: true, messages: [] } as BrowserConsoleResponse)
      );
      registerBrowserCommands(context, service as MehrhofProjectService);

      let infoMessage: string | undefined;
      (vscode.window.showInformationMessage as ReturnType<typeof mock>) = mock((msg: string) => {
        infoMessage = msg;
        return Promise.resolve(undefined);
      });

      const handler = registeredCommands.get('mehrhof.browserConsole');
      await handler!();

      expect(infoMessage).toBe('No console messages');
    });
  });

  describe('mehrhof.browserNetwork command', () => {
    test('calls browserNetwork API', async () => {
      const { service, client } = createMockProjectService();
      registerBrowserCommands(context, service as MehrhofProjectService);

      (vscode.window.showQuickPick as ReturnType<typeof mock>) = mock(() =>
        Promise.resolve(undefined)
      );

      const handler = registeredCommands.get('mehrhof.browserNetwork');
      await handler!();

      expect(client.browserNetwork).toHaveBeenCalled();
    });

    test('shows no requests message when empty', async () => {
      const { service, client } = createMockProjectService();
      (client.browserNetwork as ReturnType<typeof mock>) = mock(() =>
        Promise.resolve({ success: true, requests: [] } as BrowserNetworkResponse)
      );
      registerBrowserCommands(context, service as MehrhofProjectService);

      let infoMessage: string | undefined;
      (vscode.window.showInformationMessage as ReturnType<typeof mock>) = mock((msg: string) => {
        infoMessage = msg;
        return Promise.resolve(undefined);
      });

      const handler = registeredCommands.get('mehrhof.browserNetwork');
      await handler!();

      expect(infoMessage).toBe('No network requests');
    });
  });
});
