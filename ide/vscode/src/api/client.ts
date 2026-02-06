import { MehrhofApiClientBase } from './clientBase';
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
} from './models';

// Re-export for backward compatibility
export { ApiError, type ClientOptions } from './clientBase';

export class MehrhofApiClient extends MehrhofApiClientBase {
  // ============================================================================
  // Queue Task Operations (via Interactive API)
  // ============================================================================

  async deleteQueueTask(queueId: string, taskId: string): Promise<DeleteQueueTaskResponse> {
    return this.executeCommand({
      command: 'delete',
      args: [`${queueId}/${taskId}`],
    }) as Promise<DeleteQueueTaskResponse>;
  }

  async exportQueueTask(queueId: string, taskId: string): Promise<ExportQueueTaskResponse> {
    return this.executeCommand({
      command: 'export',
      args: [`${queueId}/${taskId}`],
    }) as Promise<ExportQueueTaskResponse>;
  }

  async optimizeQueueTask(queueId: string, taskId: string): Promise<OptimizeQueueTaskResponse> {
    return this.executeCommand({
      command: 'optimize',
      args: [`${queueId}/${taskId}`],
    }) as Promise<OptimizeQueueTaskResponse>;
  }

  async submitQueueTask(
    queueId: string,
    taskId: string,
    provider: string
  ): Promise<SubmitQueueTaskResponse> {
    return this.executeCommand({
      command: 'submit',
      args: [`${queueId}/${taskId}`, provider],
    }) as Promise<SubmitQueueTaskResponse>;
  }

  async syncTask(): Promise<SyncTaskResponse> {
    return this.executeCommand({
      command: 'sync',
      args: [],
    }) as Promise<SyncTaskResponse>;
  }

  // ============================================================================
  // Find & Search
  // ============================================================================

  async find(
    query: string,
    options?: { path?: string; pattern?: string }
  ): Promise<FindSearchResponse> {
    const params = new URLSearchParams({ q: query });
    if (options?.path) params.append('path', options.path);
    if (options?.pattern) params.append('pattern', options.pattern);
    return this.get<FindSearchResponse>(`/api/v1/find?${params.toString()}`);
  }

  // ============================================================================
  // Memory Operations (via Interactive API)
  // ============================================================================

  async memorySearch(query: string): Promise<MemorySearchResponse> {
    // Use direct API for richer response data
    const params = new URLSearchParams({ q: query, limit: '10' });
    return this.get<MemorySearchResponse>(`/api/v1/memory/search?${params.toString()}`);
  }

  async memoryIndex(taskId: string): Promise<MemoryIndexResponse> {
    return this.post<MemoryIndexResponse>('/api/v1/memory/index', { task_id: taskId });
  }

  async memoryStats(): Promise<MemoryStatsResponse> {
    return this.get<MemoryStatsResponse>('/api/v1/memory/stats');
  }

  // ============================================================================
  // Library Operations
  // ============================================================================

  async libraryList(): Promise<LibraryListResponse> {
    return this.get<LibraryListResponse>('/api/v1/library');
  }

  async libraryShow(nameOrId: string): Promise<LibraryShowResponse> {
    return this.get<LibraryShowResponse>(`/api/v1/library/${encodeURIComponent(nameOrId)}`);
  }

  async libraryStats(): Promise<LibraryStatsResponse> {
    return this.get<LibraryStatsResponse>('/api/v1/library/stats');
  }

  async libraryPull(
    source: string,
    options?: { name?: string; shared?: boolean }
  ): Promise<InteractiveCommandResponse> {
    const args = [source];
    if (options?.name) {
      args.push('--name', options.name);
    }
    if (options?.shared) {
      args.push('--shared');
    }
    return this.executeCommand({ command: 'library', args: ['pull', ...args] });
  }

  async libraryRemove(nameOrId: string): Promise<InteractiveCommandResponse> {
    return this.executeCommand({ command: 'library', args: ['remove', nameOrId] });
  }

  // ============================================================================
  // Links Operations
  // ============================================================================

  async linksList(): Promise<LinksListResponse> {
    return this.get<LinksListResponse>('/api/v1/links');
  }

  async linksGet(entityId: string): Promise<EntityLinksResponse> {
    return this.get<EntityLinksResponse>(`/api/v1/links/${encodeURIComponent(entityId)}`);
  }

  async linksSearch(query: string): Promise<LinksSearchResponse> {
    const params = new URLSearchParams({ q: query });
    return this.get<LinksSearchResponse>(`/api/v1/links/search?${params.toString()}`);
  }

  async linksStats(): Promise<LinksStatsResponse> {
    return this.get<LinksStatsResponse>('/api/v1/links/stats');
  }

  async linksRebuild(): Promise<InteractiveCommandResponse> {
    return this.executeCommand({ command: 'links', args: ['rebuild'] });
  }

  // ============================================================================
  // Browser Operations
  // ============================================================================

  async browserStatus(): Promise<BrowserStatusResponse> {
    return this.get<BrowserStatusResponse>('/api/v1/browser/status');
  }

  async browserTabs(): Promise<BrowserTabsResponse> {
    return this.get<BrowserTabsResponse>('/api/v1/browser/tabs');
  }

  async browserGoto(url: string): Promise<BrowserGotoResponse> {
    return this.post<BrowserGotoResponse>('/api/v1/browser/goto', { url });
  }

  async browserNavigate(url: string, tabId?: string): Promise<BrowserNavigateResponse> {
    return this.post<BrowserNavigateResponse>('/api/v1/browser/navigate', {
      url,
      tab_id: tabId,
    });
  }

  async browserClick(selector: string, tabId?: string): Promise<BrowserClickResponse> {
    return this.post<BrowserClickResponse>('/api/v1/browser/click', {
      selector,
      tab_id: tabId,
    });
  }

  async browserType(
    selector: string,
    text: string,
    options?: { tabId?: string; clear?: boolean }
  ): Promise<BrowserTypeResponse> {
    return this.post<BrowserTypeResponse>('/api/v1/browser/type', {
      selector,
      text,
      tab_id: options?.tabId,
      clear: options?.clear,
    });
  }

  async browserEval(expression: string, tabId?: string): Promise<BrowserEvalResponse> {
    return this.post<BrowserEvalResponse>('/api/v1/browser/eval', {
      expression,
      tab_id: tabId,
    });
  }

  async browserDom(
    selector: string,
    options?: { tabId?: string; all?: boolean; html?: boolean; limit?: number }
  ): Promise<BrowserDOMResponse> {
    return this.post<BrowserDOMResponse>('/api/v1/browser/dom', {
      selector,
      tab_id: options?.tabId,
      all: options?.all,
      html: options?.html,
      limit: options?.limit,
    });
  }

  async browserScreenshot(options?: {
    tabId?: string;
    format?: string;
    quality?: number;
    fullPage?: boolean;
  }): Promise<BrowserScreenshotResponse> {
    return this.post<BrowserScreenshotResponse>('/api/v1/browser/screenshot', {
      tab_id: options?.tabId,
      format: options?.format,
      quality: options?.quality,
      full_page: options?.fullPage,
    });
  }

  async browserReload(options?: {
    tabId?: string;
    hard?: boolean;
  }): Promise<BrowserReloadResponse> {
    return this.post<BrowserReloadResponse>('/api/v1/browser/reload', {
      tab_id: options?.tabId,
      hard: options?.hard,
    });
  }

  async browserClose(tabId: string): Promise<BrowserCloseResponse> {
    return this.post<BrowserCloseResponse>('/api/v1/browser/close', { tab_id: tabId });
  }

  async browserConsole(options?: {
    tabId?: string;
    duration?: number;
    level?: string;
  }): Promise<BrowserConsoleResponse> {
    return this.post<BrowserConsoleResponse>('/api/v1/browser/console', {
      tab_id: options?.tabId,
      duration: options?.duration,
      level: options?.level,
    });
  }

  async browserNetwork(options?: {
    tabId?: string;
    duration?: number;
    captureBody?: boolean;
    maxBodySize?: number;
  }): Promise<BrowserNetworkResponse> {
    return this.post<BrowserNetworkResponse>('/api/v1/browser/network', {
      tab_id: options?.tabId,
      duration: options?.duration,
      capture_body: options?.captureBody,
      max_body_size: options?.maxBodySize,
    });
  }
}
