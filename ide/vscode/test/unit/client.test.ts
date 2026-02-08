import { describe, test, expect, beforeEach, afterEach } from 'bun:test';
import { MehrhofApiClient } from '../../src/api/client';
import type {
  DeleteQueueTaskResponse,
  ExportQueueTaskResponse,
  OptimizeQueueTaskResponse,
  SubmitQueueTaskResponse,
  SyncTaskResponse,
  FindSearchResponse,
  MemorySearchResponse,
  MemoryIndexResponse,
  MemoryStatsResponse,
  LibraryListResponse,
  LibraryShowResponse,
  LibraryStatsResponse,
  InteractiveCommandResponse,
  LinksListResponse,
  EntityLinksResponse,
  LinksSearchResponse,
  LinksStatsResponse,
  BrowserStatusResponse,
  BrowserTabsResponse,
  BrowserGotoResponse,
  BrowserNavigateResponse,
  BrowserClickResponse,
  BrowserTypeResponse,
  BrowserEvalResponse,
  BrowserDOMResponse,
  BrowserScreenshotResponse,
  BrowserReloadResponse,
  BrowserCloseResponse,
  BrowserConsoleResponse,
  BrowserNetworkResponse,
} from '../../src/api/models';

describe('MehrhofApiClient Extended Methods', () => {
  let originalFetch: typeof global.fetch;
  let capturedUrl: string | undefined;
  let capturedBody: unknown;
  let capturedMethod: string | undefined;

  beforeEach(() => {
    originalFetch = global.fetch;
    capturedUrl = undefined;
    capturedBody = undefined;
    capturedMethod = undefined;
  });

  afterEach(() => {
    global.fetch = originalFetch;
  });

  function setupMockFetch(response: unknown, status = 200): void {
    global.fetch = ((url: string | URL, init?: RequestInit): Promise<Response> => {
      capturedUrl = url.toString();
      capturedMethod = init?.method ?? 'GET';
      capturedBody = init?.body ? (JSON.parse(init.body as string) as unknown) : undefined;
      return Promise.resolve({
        ok: status >= 200 && status < 300,
        status,
        statusText: status === 200 ? 'OK' : 'Error',
        json: () => Promise.resolve(response),
        headers: new Headers(),
      } as Response);
    }) as typeof fetch;
  }

  // ============================================================================
  // Queue Task Operations
  // ============================================================================

  describe('Queue Task Operations', () => {
    test('deleteQueueTask sends correct command', async () => {
      const mockResponse: DeleteQueueTaskResponse = {
        success: true,
        message: 'Task deleted',
      };
      setupMockFetch(mockResponse);

      const client = new MehrhofApiClient('http://localhost:3000');
      const result = await client.deleteQueueTask('queue-1', 'task-1');

      expect(result.success).toBe(true);
      expect(capturedBody).toEqual({
        command: 'delete',
        args: ['queue-1/task-1'],
      });
    });

    test('exportQueueTask sends correct command', async () => {
      const mockResponse: ExportQueueTaskResponse = {
        success: true,
        markdown: '# Exported Task',
      };
      setupMockFetch(mockResponse);

      const client = new MehrhofApiClient('http://localhost:3000');
      const result = await client.exportQueueTask('queue-1', 'task-2');

      expect(result.success).toBe(true);
      expect(result.markdown).toBe('# Exported Task');
      expect(capturedBody).toEqual({
        command: 'export',
        args: ['queue-1/task-2'],
      });
    });

    test('optimizeQueueTask sends correct command', async () => {
      const mockResponse: OptimizeQueueTaskResponse = {
        success: true,
        original_title: 'Fix bug',
        optimized_title: 'Fix authentication bug in login flow',
        added_labels: ['bug', 'auth'],
      };
      setupMockFetch(mockResponse);

      const client = new MehrhofApiClient('http://localhost:3000');
      const result = await client.optimizeQueueTask('queue-1', 'task-3');

      expect(result.success).toBe(true);
      expect(result.optimized_title).toBe('Fix authentication bug in login flow');
      expect(capturedBody).toEqual({
        command: 'optimize',
        args: ['queue-1/task-3'],
      });
    });

    test('submitQueueTask sends correct command with provider', async () => {
      const mockResponse: SubmitQueueTaskResponse = {
        success: true,
        external_id: 'GH-123',
        url: 'https://github.com/owner/repo/issues/123',
      };
      setupMockFetch(mockResponse);

      const client = new MehrhofApiClient('http://localhost:3000');
      const result = await client.submitQueueTask('queue-1', 'task-4', 'github');

      expect(result.success).toBe(true);
      expect(result.external_id).toBe('GH-123');
      expect(capturedBody).toEqual({
        command: 'submit',
        args: ['queue-1/task-4', 'github'],
      });
    });

    test('syncTask sends correct command', async () => {
      const mockResponse: SyncTaskResponse = {
        success: true,
        message: 'Task synced',
      };
      setupMockFetch(mockResponse);

      const client = new MehrhofApiClient('http://localhost:3000');
      const result = await client.syncTask();

      expect(result.success).toBe(true);
      expect(capturedBody).toEqual({
        command: 'sync',
        args: [],
      });
    });
  });

  // ============================================================================
  // Find & Search
  // ============================================================================

  describe('Find & Search', () => {
    test('find sends query parameter', async () => {
      const mockResponse: FindSearchResponse = {
        query: 'TODO',
        count: 2,
        matches: [
          { file: 'src/app.ts', line: 10, snippet: '// TODO: implement' },
          { file: 'src/utils.ts', line: 25, snippet: '// TODO: refactor' },
        ],
      };
      setupMockFetch(mockResponse);

      const client = new MehrhofApiClient('http://localhost:3000');
      const result = await client.find('TODO');

      expect(result.count).toBe(2);
      expect(result.matches.length).toBe(2);
      expect(capturedUrl).toContain('/api/v1/find?q=TODO');
      expect(capturedMethod).toBe('GET');
    });

    test('find with path option appends path parameter', async () => {
      const mockResponse: FindSearchResponse = {
        query: 'error',
        count: 1,
        matches: [{ file: 'src/handlers/error.ts', line: 5, snippet: 'function handleError' }],
      };
      setupMockFetch(mockResponse);

      const client = new MehrhofApiClient('http://localhost:3000');
      await client.find('error', { path: 'src/handlers' });

      expect(capturedUrl).toContain('q=error');
      expect(capturedUrl).toContain('path=src%2Fhandlers');
    });

    test('find with pattern option appends pattern parameter', async () => {
      const mockResponse: FindSearchResponse = {
        query: 'test',
        count: 0,
        matches: [],
      };
      setupMockFetch(mockResponse);

      const client = new MehrhofApiClient('http://localhost:3000');
      await client.find('test', { pattern: '*.test.ts' });

      expect(capturedUrl).toContain('pattern=*.test.ts');
    });

    test('find with both path and pattern options', async () => {
      const mockResponse: FindSearchResponse = {
        query: 'describe',
        count: 5,
        matches: [],
      };
      setupMockFetch(mockResponse);

      const client = new MehrhofApiClient('http://localhost:3000');
      await client.find('describe', { path: 'test', pattern: '*.spec.ts' });

      expect(capturedUrl).toContain('q=describe');
      expect(capturedUrl).toContain('path=test');
      expect(capturedUrl).toContain('pattern=*.spec.ts');
    });
  });

  // ============================================================================
  // Memory Operations
  // ============================================================================

  describe('Memory Operations', () => {
    test('memorySearch sends query with limit', async () => {
      const mockResponse: MemorySearchResponse = {
        results: [
          { task_id: 'task-1', type: 'spec', score: 0.95, content: 'Authentication spec' },
          { task_id: 'task-2', type: 'note', score: 0.87, content: 'Login flow notes' },
        ],
        count: 2,
      };
      setupMockFetch(mockResponse);

      const client = new MehrhofApiClient('http://localhost:3000');
      const result = await client.memorySearch('authentication');

      expect(result.count).toBe(2);
      expect(result.results[0].score).toBe(0.95);
      expect(capturedUrl).toContain('/api/v1/memory/search');
      expect(capturedUrl).toContain('q=authentication');
      expect(capturedUrl).toContain('limit=10');
    });

    test('memoryIndex sends task_id', async () => {
      const mockResponse: MemoryIndexResponse = {
        success: true,
        message: 'Indexed 5 documents',
        task_id: 'task-123',
      };
      setupMockFetch(mockResponse);

      const client = new MehrhofApiClient('http://localhost:3000');
      const result = await client.memoryIndex('task-123');

      expect(result.success).toBe(true);
      expect(capturedBody).toEqual({ task_id: 'task-123' });
      expect(capturedMethod).toBe('POST');
    });

    test('memoryStats returns stats', async () => {
      const mockResponse: MemoryStatsResponse = {
        total_documents: 150,
        by_type: { spec: 50, note: 30, session: 70 },
        enabled: true,
      };
      setupMockFetch(mockResponse);

      const client = new MehrhofApiClient('http://localhost:3000');
      const result = await client.memoryStats();

      expect(result.total_documents).toBe(150);
      expect(result.enabled).toBe(true);
      expect(capturedUrl).toContain('/api/v1/memory/stats');
      expect(capturedMethod).toBe('GET');
    });
  });

  // ============================================================================
  // Library Operations
  // ============================================================================

  describe('Library Operations', () => {
    test('libraryList returns collections', async () => {
      const mockResponse: LibraryListResponse = {
        collections: [
          {
            id: 'lib-1',
            name: 'React Docs',
            source: 'https://react.dev',
            source_type: 'web',
            include_mode: 'auto',
            page_count: 100,
            total_size: 5000000,
            location: 'project',
          },
        ],
        count: 1,
      };
      setupMockFetch(mockResponse);

      const client = new MehrhofApiClient('http://localhost:3000');
      const result = await client.libraryList();

      expect(result.count).toBe(1);
      expect(result.collections[0].name).toBe('React Docs');
      expect(capturedUrl).toContain('/api/v1/library');
    });

    test('libraryShow returns collection details', async () => {
      const mockResponse: LibraryShowResponse = {
        collection: {
          id: 'lib-1',
          name: 'React Docs',
          source: 'https://react.dev',
          source_type: 'web',
          include_mode: 'auto',
          page_count: 3,
          total_size: 50000,
          location: 'project',
        },
        pages: ['hooks.md', 'components.md', 'state.md'],
      };
      setupMockFetch(mockResponse);

      const client = new MehrhofApiClient('http://localhost:3000');
      const result = await client.libraryShow('lib-1');

      expect(result.collection.name).toBe('React Docs');
      expect(result.pages.length).toBe(3);
      expect(capturedUrl).toContain('/api/v1/library/lib-1');
    });

    test('libraryShow encodes special characters in name', async () => {
      const mockResponse: LibraryShowResponse = {
        collection: {
          id: 'lib-1',
          name: 'My Library',
          source: 'local',
          source_type: 'file',
          include_mode: 'manual',
          page_count: 1,
          total_size: 1000,
          location: 'project',
        },
        pages: [],
      };
      setupMockFetch(mockResponse);

      const client = new MehrhofApiClient('http://localhost:3000');
      await client.libraryShow('lib/with/slashes');

      expect(capturedUrl).toContain('lib%2Fwith%2Fslashes');
    });

    test('libraryStats returns statistics', async () => {
      const mockResponse: LibraryStatsResponse = {
        total_collections: 5,
        total_pages: 500,
        total_size: 10000000,
        project_count: 3,
        shared_count: 2,
        by_mode: { auto: 3, manual: 2 },
        enabled: true,
      };
      setupMockFetch(mockResponse);

      const client = new MehrhofApiClient('http://localhost:3000');
      const result = await client.libraryStats();

      expect(result.total_collections).toBe(5);
      expect(result.enabled).toBe(true);
      expect(capturedUrl).toContain('/api/v1/library/stats');
    });

    test('libraryPull sends source', async () => {
      const mockResponse: InteractiveCommandResponse = {
        success: true,
        message: 'Library pulled successfully',
      };
      setupMockFetch(mockResponse);

      const client = new MehrhofApiClient('http://localhost:3000');
      const result = await client.libraryPull('https://docs.example.com');

      expect(result.success).toBe(true);
      expect(capturedBody).toEqual({
        command: 'library',
        args: ['pull', 'https://docs.example.com'],
      });
    });

    test('libraryPull with name option', async () => {
      const mockResponse: InteractiveCommandResponse = {
        success: true,
        message: 'Library pulled',
      };
      setupMockFetch(mockResponse);

      const client = new MehrhofApiClient('http://localhost:3000');
      await client.libraryPull('https://docs.example.com', { name: 'example-docs' });

      expect(capturedBody).toEqual({
        command: 'library',
        args: ['pull', 'https://docs.example.com', '--name', 'example-docs'],
      });
    });

    test('libraryPull with shared option', async () => {
      const mockResponse: InteractiveCommandResponse = {
        success: true,
        message: 'Library pulled to shared location',
      };
      setupMockFetch(mockResponse);

      const client = new MehrhofApiClient('http://localhost:3000');
      await client.libraryPull('https://docs.example.com', { shared: true });

      expect(capturedBody).toEqual({
        command: 'library',
        args: ['pull', 'https://docs.example.com', '--shared'],
      });
    });

    test('libraryPull with name and shared options', async () => {
      const mockResponse: InteractiveCommandResponse = {
        success: true,
        message: 'Library pulled',
      };
      setupMockFetch(mockResponse);

      const client = new MehrhofApiClient('http://localhost:3000');
      await client.libraryPull('https://docs.example.com', { name: 'my-lib', shared: true });

      expect(capturedBody).toEqual({
        command: 'library',
        args: ['pull', 'https://docs.example.com', '--name', 'my-lib', '--shared'],
      });
    });

    test('libraryRemove sends name', async () => {
      const mockResponse: InteractiveCommandResponse = {
        success: true,
        message: 'Library removed',
      };
      setupMockFetch(mockResponse);

      const client = new MehrhofApiClient('http://localhost:3000');
      const result = await client.libraryRemove('lib-1');

      expect(result.success).toBe(true);
      expect(capturedBody).toEqual({
        command: 'library',
        args: ['remove', 'lib-1'],
      });
    });
  });

  // ============================================================================
  // Links Operations
  // ============================================================================

  describe('Links Operations', () => {
    test('linksList returns all links', async () => {
      const mockResponse: LinksListResponse = {
        links: [
          {
            source: 'spec:1',
            target: 'decision:cache',
            context: 'specification',
            created_at: '2026-01-01T00:00:00Z',
          },
        ],
        count: 1,
      };
      setupMockFetch(mockResponse);

      const client = new MehrhofApiClient('http://localhost:3000');
      const result = await client.linksList();

      expect(result.count).toBe(1);
      expect(result.links[0].source).toBe('spec:1');
      expect(capturedUrl).toContain('/api/v1/links');
    });

    test('linksGet returns entity links', async () => {
      const mockResponse: EntityLinksResponse = {
        entity_id: 'spec:1',
        outgoing: [
          {
            source: 'spec:1',
            target: 'decision:cache',
            context: 'spec',
            created_at: '2026-01-01T00:00:00Z',
          },
        ],
        incoming: [
          {
            source: 'note:2',
            target: 'spec:1',
            context: 'note',
            created_at: '2026-01-02T00:00:00Z',
          },
        ],
      };
      setupMockFetch(mockResponse);

      const client = new MehrhofApiClient('http://localhost:3000');
      const result = await client.linksGet('spec:1');

      expect(result.entity_id).toBe('spec:1');
      expect(result.outgoing.length).toBe(1);
      expect(result.incoming.length).toBe(1);
      expect(capturedUrl).toContain('/api/v1/links/spec%3A1');
    });

    test('linksSearch returns search results', async () => {
      const mockResponse: LinksSearchResponse = {
        query: 'cache',
        results: [
          { entity_id: 'decision:cache', type: 'decision', name: 'Cache Strategy', total_links: 5 },
        ],
        count: 1,
      };
      setupMockFetch(mockResponse);

      const client = new MehrhofApiClient('http://localhost:3000');
      const result = await client.linksSearch('cache');

      expect(result.count).toBe(1);
      expect(result.results[0].entity_id).toBe('decision:cache');
      expect(capturedUrl).toContain('/api/v1/links/search?q=cache');
    });

    test('linksStats returns statistics', async () => {
      const mockResponse: LinksStatsResponse = {
        total_links: 50,
        total_sources: 20,
        total_targets: 30,
        orphan_entities: 5,
        most_linked: [{ entity_id: 'spec:main', type: 'spec', total_links: 10 }],
        enabled: true,
      };
      setupMockFetch(mockResponse);

      const client = new MehrhofApiClient('http://localhost:3000');
      const result = await client.linksStats();

      expect(result.total_links).toBe(50);
      expect(result.enabled).toBe(true);
      expect(capturedUrl).toContain('/api/v1/links/stats');
    });

    test('linksRebuild sends rebuild command', async () => {
      const mockResponse: InteractiveCommandResponse = {
        success: true,
        message: 'Links rebuilt: 100 total',
      };
      setupMockFetch(mockResponse);

      const client = new MehrhofApiClient('http://localhost:3000');
      const result = await client.linksRebuild();

      expect(result.success).toBe(true);
      expect(capturedBody).toEqual({
        command: 'links',
        args: ['rebuild'],
      });
    });
  });

  // ============================================================================
  // Browser Operations
  // ============================================================================

  describe('Browser Operations', () => {
    test('browserStatus returns browser status', async () => {
      const mockResponse: BrowserStatusResponse = {
        connected: true,
        host: 'localhost',
        port: 9222,
        tabs: [{ id: 'tab-1', title: 'Google', url: 'https://google.com' }],
      };
      setupMockFetch(mockResponse);

      const client = new MehrhofApiClient('http://localhost:3000');
      const result = await client.browserStatus();

      expect(result.connected).toBe(true);
      expect(result.tabs?.length).toBe(1);
      expect(capturedUrl).toContain('/api/v1/browser/status');
      expect(capturedMethod).toBe('GET');
    });

    test('browserTabs returns tabs list', async () => {
      const mockResponse: BrowserTabsResponse = {
        tabs: [
          { id: 'tab-1', title: 'Google', url: 'https://google.com' },
          { id: 'tab-2', title: 'GitHub', url: 'https://github.com' },
        ],
        count: 2,
      };
      setupMockFetch(mockResponse);

      const client = new MehrhofApiClient('http://localhost:3000');
      const result = await client.browserTabs();

      expect(result.count).toBe(2);
      expect(result.tabs[0].title).toBe('Google');
      expect(capturedUrl).toContain('/api/v1/browser/tabs');
    });

    test('browserGoto navigates to URL', async () => {
      const mockResponse: BrowserGotoResponse = {
        success: true,
        tab: { id: 'tab-1', title: 'Example', url: 'https://example.com' },
      };
      setupMockFetch(mockResponse);

      const client = new MehrhofApiClient('http://localhost:3000');
      const result = await client.browserGoto('https://example.com');

      expect(result.success).toBe(true);
      expect(result.tab?.url).toBe('https://example.com');
      expect(capturedBody).toEqual({ url: 'https://example.com' });
      expect(capturedMethod).toBe('POST');
    });

    test('browserNavigate sends url and optional tabId', async () => {
      const mockResponse: BrowserNavigateResponse = {
        success: true,
        message: 'Navigated',
      };
      setupMockFetch(mockResponse);

      const client = new MehrhofApiClient('http://localhost:3000');
      await client.browserNavigate('https://test.com', 'tab-2');

      expect(capturedBody).toEqual({
        url: 'https://test.com',
        tab_id: 'tab-2',
      });
    });

    test('browserNavigate without tabId', async () => {
      const mockResponse: BrowserNavigateResponse = { success: true };
      setupMockFetch(mockResponse);

      const client = new MehrhofApiClient('http://localhost:3000');
      await client.browserNavigate('https://test.com');

      expect(capturedBody).toEqual({
        url: 'https://test.com',
        tab_id: undefined,
      });
    });

    test('browserClick sends selector and optional tabId', async () => {
      const mockResponse: BrowserClickResponse = {
        success: true,
        selector: '#submit-btn',
      };
      setupMockFetch(mockResponse);

      const client = new MehrhofApiClient('http://localhost:3000');
      const result = await client.browserClick('#submit-btn', 'tab-1');

      expect(result.success).toBe(true);
      expect(capturedBody).toEqual({
        selector: '#submit-btn',
        tab_id: 'tab-1',
      });
    });

    test('browserType sends selector, text, and options', async () => {
      const mockResponse: BrowserTypeResponse = {
        success: true,
        selector: '#input-field',
      };
      setupMockFetch(mockResponse);

      const client = new MehrhofApiClient('http://localhost:3000');
      await client.browserType('#input-field', 'Hello World', { tabId: 'tab-1', clear: true });

      expect(capturedBody).toEqual({
        selector: '#input-field',
        text: 'Hello World',
        tab_id: 'tab-1',
        clear: true,
      });
    });

    test('browserType with minimal options', async () => {
      const mockResponse: BrowserTypeResponse = { success: true };
      setupMockFetch(mockResponse);

      const client = new MehrhofApiClient('http://localhost:3000');
      await client.browserType('.search', 'query');

      expect(capturedBody).toEqual({
        selector: '.search',
        text: 'query',
        tab_id: undefined,
        clear: undefined,
      });
    });

    test('browserEval sends expression', async () => {
      const mockResponse: BrowserEvalResponse = {
        success: true,
        result: { title: 'Page Title' },
      };
      setupMockFetch(mockResponse);

      const client = new MehrhofApiClient('http://localhost:3000');
      const result = await client.browserEval('document.title');

      expect(result.success).toBe(true);
      expect(capturedBody).toEqual({
        expression: 'document.title',
        tab_id: undefined,
      });
    });

    test('browserDom sends selector and options', async () => {
      const mockResponse: BrowserDOMResponse = {
        success: true,
        elements: [{ tag_name: 'div', text_content: 'Content', visible: true }],
        count: 1,
      };
      setupMockFetch(mockResponse);

      const client = new MehrhofApiClient('http://localhost:3000');
      await client.browserDom('.container', { all: true, html: true, limit: 10 });

      expect(capturedBody).toEqual({
        selector: '.container',
        tab_id: undefined,
        all: true,
        html: true,
        limit: 10,
      });
    });

    test('browserScreenshot sends options', async () => {
      const mockResponse: BrowserScreenshotResponse = {
        success: true,
        format: 'png',
        data: 'base64data...',
        size: 12345,
      };
      setupMockFetch(mockResponse);

      const client = new MehrhofApiClient('http://localhost:3000');
      const result = await client.browserScreenshot({
        format: 'jpeg',
        quality: 80,
        fullPage: true,
      });

      expect(result.success).toBe(true);
      expect(capturedBody).toEqual({
        tab_id: undefined,
        format: 'jpeg',
        quality: 80,
        full_page: true,
      });
    });

    test('browserScreenshot with no options', async () => {
      const mockResponse: BrowserScreenshotResponse = { success: true };
      setupMockFetch(mockResponse);

      const client = new MehrhofApiClient('http://localhost:3000');
      await client.browserScreenshot();

      expect(capturedBody).toEqual({
        tab_id: undefined,
        format: undefined,
        quality: undefined,
        full_page: undefined,
      });
    });

    test('browserReload sends options', async () => {
      const mockResponse: BrowserReloadResponse = {
        success: true,
        message: 'Page reloaded',
      };
      setupMockFetch(mockResponse);

      const client = new MehrhofApiClient('http://localhost:3000');
      await client.browserReload({ tabId: 'tab-1', hard: true });

      expect(capturedBody).toEqual({
        tab_id: 'tab-1',
        hard: true,
      });
    });

    test('browserClose sends tab_id', async () => {
      const mockResponse: BrowserCloseResponse = {
        success: true,
        message: 'Tab closed',
      };
      setupMockFetch(mockResponse);

      const client = new MehrhofApiClient('http://localhost:3000');
      const result = await client.browserClose('tab-1');

      expect(result.success).toBe(true);
      expect(capturedBody).toEqual({ tab_id: 'tab-1' });
    });

    test('browserConsole sends options', async () => {
      const mockResponse: BrowserConsoleResponse = {
        success: true,
        messages: [
          { level: 'error', text: 'Failed to load resource', timestamp: '2026-01-01T00:00:00Z' },
        ],
        count: 1,
      };
      setupMockFetch(mockResponse);

      const client = new MehrhofApiClient('http://localhost:3000');
      const result = await client.browserConsole({ duration: 5000, level: 'error' });

      expect(result.count).toBe(1);
      expect(capturedBody).toEqual({
        tab_id: undefined,
        duration: 5000,
        level: 'error',
      });
    });

    test('browserNetwork sends options', async () => {
      const mockResponse: BrowserNetworkResponse = {
        success: true,
        requests: [
          {
            method: 'GET',
            url: 'https://api.example.com/data',
            status: 200,
            status_text: 'OK',
            timestamp: '2026-01-01T00:00:00Z',
          },
        ],
        count: 1,
      };
      setupMockFetch(mockResponse);

      const client = new MehrhofApiClient('http://localhost:3000');
      const result = await client.browserNetwork({
        duration: 10000,
        captureBody: true,
        maxBodySize: 1024,
      });

      expect(result.count).toBe(1);
      expect(capturedBody).toEqual({
        tab_id: undefined,
        duration: 10000,
        capture_body: true,
        max_body_size: 1024,
      });
    });
  });
});
