import type {
  StatusResponse,
  TaskResponse,
  TaskListResponse,
  WorkflowResponse,
  ContinueResponse,
  GuideResponse,
  AllCostsResponse,
  TaskCostResponse,
  AgentsListResponse,
  ProvidersListResponse,
  InteractiveCommandRequest,
  InteractiveCommandResponse,
  InteractiveChatRequest,
  InteractiveChatResponse,
  InteractiveStateResponse,
  InteractiveStopResponse,
  CommandsResponse,
  StartTaskRequest,
  FinishRequest,
  WorkflowRequest,
  AnswerRequest,
  InteractiveAnswerRequest,
  ErrorResponse,
  AddNoteRequest,
  AddNoteResponse,
  QuestionRequest,
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

export class ApiError extends Error {
  constructor(
    message: string,
    public readonly statusCode: number,
    public readonly response?: ErrorResponse
  ) {
    super(message);
    this.name = 'ApiError';
  }
}

export interface ClientOptions {
  connectTimeoutMs?: number;
  readTimeoutMs?: number;
}

const DEFAULT_CONNECT_TIMEOUT = 10000; // 10 seconds
const DEFAULT_READ_TIMEOUT = 60000; // 60 seconds

export class MehrhofApiClient {
  private readonly baseUrl: string;
  private readonly connectTimeout: number;
  private readonly readTimeout: number;
  private sessionCookie?: string;
  private csrfToken?: string;

  constructor(baseUrl: string, options: ClientOptions = {}) {
    this.baseUrl = baseUrl.replace(/\/$/, '');
    this.connectTimeout = options.connectTimeoutMs ?? DEFAULT_CONNECT_TIMEOUT;
    this.readTimeout = options.readTimeoutMs ?? DEFAULT_READ_TIMEOUT;
  }

  setSessionCookie(cookie: string | undefined): void {
    this.sessionCookie = cookie;
  }

  setCsrfToken(token: string | undefined): void {
    this.csrfToken = token;
  }

  private async request<T>(
    method: string,
    path: string,
    body?: unknown,
    timeout?: number
  ): Promise<T> {
    const url = `${this.baseUrl}${path}`;
    const controller = new AbortController();
    const timeoutMs = timeout ?? this.readTimeout;
    const timeoutId = setTimeout(() => controller.abort(), timeoutMs);

    try {
      const headers: Record<string, string> = {
        'Content-Type': 'application/json',
        Accept: 'application/json',
      };

      if (this.sessionCookie) {
        headers['Cookie'] = this.sessionCookie;
      }

      // Include CSRF token on state-changing requests
      if (this.csrfToken && method !== 'GET' && method !== 'HEAD') {
        headers['X-Csrf-Token'] = this.csrfToken;
      }

      const response = await fetch(url, {
        method,
        headers,
        body: body ? JSON.stringify(body) : undefined,
        signal: controller.signal,
      });

      // Extract session cookie from response if present
      const setCookie = response.headers.get('set-cookie');
      if (setCookie) {
        const match = setCookie.match(/mehr_session=([^;]+)/);
        if (match) {
          this.sessionCookie = `mehr_session=${match[1]}`;
        }
      }

      if (!response.ok) {
        let errorResponse: ErrorResponse | undefined;
        try {
          errorResponse = (await response.json()) as ErrorResponse;
        } catch {
          // Ignore JSON parse errors
        }
        throw new ApiError(
          errorResponse?.error ?? `HTTP ${response.status}: ${response.statusText}`,
          response.status,
          errorResponse
        );
      }

      return (await response.json()) as T;
    } catch (error) {
      if (error instanceof ApiError) {
        throw error;
      }
      if (error instanceof Error) {
        if (error.name === 'AbortError') {
          throw new ApiError(`Request timeout after ${timeoutMs}ms`, 0);
        }
        throw new ApiError(error.message, 0);
      }
      throw new ApiError('Unknown error', 0);
    } finally {
      clearTimeout(timeoutId);
    }
  }

  private get<T>(path: string, timeout?: number): Promise<T> {
    return this.request<T>('GET', path, undefined, timeout);
  }

  private post<T>(path: string, body?: unknown, timeout?: number): Promise<T> {
    return this.request<T>('POST', path, body, timeout);
  }

  // ============================================================================
  // Health & Status
  // ============================================================================

  async health(): Promise<boolean> {
    try {
      await this.get<void>('/health', this.connectTimeout);
      return true;
    } catch {
      return false;
    }
  }

  async getStatus(): Promise<StatusResponse> {
    return this.get<StatusResponse>('/api/v1/status');
  }

  // ============================================================================
  // Task Management
  // ============================================================================

  async getTask(): Promise<TaskResponse> {
    return this.get<TaskResponse>('/api/v1/task');
  }

  async getTasks(): Promise<TaskListResponse> {
    return this.get<TaskListResponse>('/api/v1/tasks');
  }

  async getTaskCosts(taskId: string): Promise<TaskCostResponse> {
    return this.get<TaskCostResponse>(`/api/v1/tasks/${taskId}/costs`);
  }

  async addNote(taskId: string, request: AddNoteRequest): Promise<AddNoteResponse> {
    return this.post<AddNoteResponse>(`/api/v1/tasks/${taskId}/notes`, request);
  }

  // ============================================================================
  // Workflow Operations
  // ============================================================================

  async startTask(request: StartTaskRequest): Promise<WorkflowResponse> {
    return this.post<WorkflowResponse>('/api/v1/workflow/start', request);
  }

  async plan(request?: WorkflowRequest): Promise<WorkflowResponse> {
    return this.post<WorkflowResponse>('/api/v1/workflow/plan', request ?? {});
  }

  async implement(request?: WorkflowRequest): Promise<WorkflowResponse> {
    return this.post<WorkflowResponse>('/api/v1/workflow/implement', request ?? {});
  }

  async review(request?: WorkflowRequest): Promise<WorkflowResponse> {
    return this.post<WorkflowResponse>('/api/v1/workflow/review', request ?? {});
  }

  async finish(request?: FinishRequest): Promise<WorkflowResponse> {
    return this.post<WorkflowResponse>('/api/v1/workflow/finish', request ?? {});
  }

  async continueWorkflow(): Promise<ContinueResponse> {
    return this.post<ContinueResponse>('/api/v1/workflow/continue');
  }

  async abandon(): Promise<WorkflowResponse> {
    return this.post<WorkflowResponse>('/api/v1/workflow/abandon');
  }

  async reset(): Promise<WorkflowResponse> {
    return this.post<WorkflowResponse>('/api/v1/workflow/reset');
  }

  async answer(request: AnswerRequest): Promise<WorkflowResponse> {
    return this.post<WorkflowResponse>('/api/v1/workflow/answer', request);
  }

  async question(request: QuestionRequest): Promise<WorkflowResponse> {
    return this.post<WorkflowResponse>('/api/v1/workflow/question', request);
  }

  // ============================================================================
  // Checkpoints (Undo/Redo)
  // ============================================================================

  async undo(): Promise<WorkflowResponse> {
    return this.post<WorkflowResponse>('/api/v1/workflow/undo');
  }

  async redo(): Promise<WorkflowResponse> {
    return this.post<WorkflowResponse>('/api/v1/workflow/redo');
  }

  // ============================================================================
  // Interactive API
  // ============================================================================

  async executeCommand(request: InteractiveCommandRequest): Promise<InteractiveCommandResponse> {
    return this.post<InteractiveCommandResponse>('/api/v1/interactive/command', request);
  }

  async chat(request: InteractiveChatRequest): Promise<InteractiveChatResponse> {
    return this.post<InteractiveChatResponse>('/api/v1/interactive/chat', request);
  }

  async answerInteractive(request: InteractiveAnswerRequest): Promise<InteractiveCommandResponse> {
    return this.post<InteractiveCommandResponse>('/api/v1/interactive/answer', request);
  }

  async getInteractiveState(): Promise<InteractiveStateResponse> {
    return this.get<InteractiveStateResponse>('/api/v1/interactive/state');
  }

  async stopOperation(): Promise<InteractiveStopResponse> {
    return this.post<InteractiveStopResponse>('/api/v1/interactive/stop');
  }

  async getCommands(): Promise<CommandsResponse> {
    return this.get<CommandsResponse>('/api/v1/interactive/commands');
  }

  // ============================================================================
  // Guide & Info
  // ============================================================================

  async getGuide(): Promise<GuideResponse> {
    return this.get<GuideResponse>('/api/v1/guide');
  }

  async getAllCosts(): Promise<AllCostsResponse> {
    return this.get<AllCostsResponse>('/api/v1/costs');
  }

  async getAgents(): Promise<AgentsListResponse> {
    return this.get<AgentsListResponse>('/api/v1/agents');
  }

  async getProviders(): Promise<ProvidersListResponse> {
    return this.get<ProvidersListResponse>('/api/v1/providers');
  }

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

  // ============================================================================
  // Utility
  // ============================================================================

  getBaseUrl(): string {
    return this.baseUrl;
  }

  getEventsUrl(): string {
    return `${this.baseUrl}/api/v1/events`;
  }
}
