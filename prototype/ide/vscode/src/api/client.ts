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
  StartTaskRequest,
  FinishRequest,
  WorkflowRequest,
  AnswerRequest,
  InteractiveAnswerRequest,
  ErrorResponse,
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

  constructor(baseUrl: string, options: ClientOptions = {}) {
    this.baseUrl = baseUrl.replace(/\/$/, '');
    this.connectTimeout = options.connectTimeoutMs ?? DEFAULT_CONNECT_TIMEOUT;
    this.readTimeout = options.readTimeoutMs ?? DEFAULT_READ_TIMEOUT;
  }

  setSessionCookie(cookie: string | undefined): void {
    this.sessionCookie = cookie;
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
  // Utility
  // ============================================================================

  getBaseUrl(): string {
    return this.baseUrl;
  }

  getEventsUrl(): string {
    return `${this.baseUrl}/api/v1/events`;
  }
}
