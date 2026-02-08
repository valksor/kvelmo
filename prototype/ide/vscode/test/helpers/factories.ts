/**
 * Test data factories for creating mock objects in tests.
 * Provides type-safe factory functions for all API models.
 */

import type {
  StatusResponse,
  TaskResponse,
  TaskInfo,
  TaskWork,
  TaskSummary,
  TaskListResponse,
  WorkflowResponse,
  CostInfo,
  TaskCostResponse,
  StepCost,
  AgentInfo,
  ProviderInfo,
  GuideResponse,
  PendingQuestion,
  InteractiveCommandRequest,
  InteractiveChatRequest,
  InteractiveCommandResponse,
  InteractiveChatResponse,
  StateChangedEvent,
  AgentMessageEvent,
  FindSearchResponse,
  FindMatch,
  MemorySearchResponse,
  MemoryResult,
  MemoryIndexResponse,
  MemoryStatsResponse,
  LibraryListResponse,
  LibraryCollection,
  LibraryShowResponse,
  LibraryStatsResponse,
  LinksListResponse,
  LinkData,
  EntityLinksResponse,
  LinksSearchResponse,
  EntityResult,
  LinksStatsResponse,
  BrowserStatusResponse,
  BrowserTab,
  BrowserTabsResponse,
} from '../../src/api/models';

// Counter for generating unique IDs
let idCounter = 0;

function nextId(): string {
  return `test-${++idCounter}`;
}

/**
 * Reset ID counter (call in test setup).
 */
export function resetFactories(): void {
  idCounter = 0;
}

/**
 * Create a StatusResponse with optional overrides.
 */
export function createStatusResponse(overrides: Partial<StatusResponse> = {}): StatusResponse {
  return {
    mode: 'project',
    running: true,
    port: 3000,
    state: 'idle',
    ...overrides,
  };
}

/**
 * Create a TaskInfo with optional overrides.
 */
export function createTaskInfo(overrides: Partial<TaskInfo> = {}): TaskInfo {
  return {
    id: nextId(),
    state: 'idle',
    ref: 'file:task.md',
    branch: 'feature/test-task',
    ...overrides,
  };
}

/**
 * Create a TaskWork with optional overrides.
 */
export function createTaskWork(overrides: Partial<TaskWork> = {}): TaskWork {
  return {
    title: 'Test Task',
    created_at: new Date().toISOString(),
    ...overrides,
  };
}

/**
 * Create a PendingQuestion with optional overrides.
 */
export function createPendingQuestion(overrides: Partial<PendingQuestion> = {}): PendingQuestion {
  return {
    question: 'What should we do?',
    options: ['Option A', 'Option B'],
    ...overrides,
  };
}

/**
 * Create a TaskResponse with optional overrides.
 */
export function createTaskResponse(overrides: Partial<TaskResponse> = {}): TaskResponse {
  return {
    active: true,
    task: createTaskInfo(),
    work: createTaskWork(),
    ...overrides,
  };
}

/**
 * Create a TaskSummary (for task list) with optional overrides.
 */
export function createTaskListItem(overrides: Partial<TaskSummary> = {}): TaskSummary {
  return {
    id: nextId(),
    state: 'idle',
    title: 'Test Task Item',
    ...overrides,
  };
}

/**
 * Create a TaskListResponse with optional overrides.
 */
export function createTaskListResponse(
  count: number = 2,
  overrides: Partial<TaskListResponse> = {}
): TaskListResponse {
  const tasks = Array.from({ length: count }, (_, i) =>
    createTaskListItem({ title: `Task ${i + 1}` })
  );
  return {
    tasks,
    count: tasks.length,
    ...overrides,
  };
}

/**
 * Create a WorkflowResponse with optional overrides.
 */
export function createWorkflowResponse(
  overrides: Partial<WorkflowResponse> = {}
): WorkflowResponse {
  return {
    success: true,
    state: 'idle',
    ...overrides,
  };
}

/**
 * Create a CostInfo with optional overrides.
 */
export function createCostInfo(overrides: Partial<CostInfo> = {}): CostInfo {
  return {
    input_tokens: 1000,
    output_tokens: 500,
    total_tokens: 1500,
    cached_tokens: 0,
    total_cost_usd: 0.05,
    ...overrides,
  };
}

/**
 * Create a StepCost with optional overrides.
 */
export function createStepCost(overrides: Partial<StepCost> = {}): StepCost {
  return {
    input_tokens: 500,
    output_tokens: 250,
    cached_tokens: 0,
    total_tokens: 750,
    cost_usd: 0.025,
    calls: 1,
    ...overrides,
  };
}

/**
 * Create a TaskCostResponse with optional overrides.
 */
export function createTaskCostResponse(
  overrides: Partial<TaskCostResponse> = {}
): TaskCostResponse {
  return {
    task_id: nextId(),
    total_tokens: 1500,
    input_tokens: 1000,
    output_tokens: 500,
    cached_tokens: 0,
    total_cost_usd: 0.05,
    by_step: {
      planning: createStepCost(),
      implementing: createStepCost(),
    },
    ...overrides,
  };
}

/**
 * Create an AgentInfo with optional overrides.
 */
export function createAgentInfo(overrides: Partial<AgentInfo> = {}): AgentInfo {
  return {
    name: 'test-agent',
    type: 'claude',
    available: true,
    ...overrides,
  };
}

/**
 * Create a ProviderInfo with optional overrides.
 */
export function createProviderInfo(overrides: Partial<ProviderInfo> = {}): ProviderInfo {
  return {
    name: 'test-provider',
    scheme: 'file',
    description: 'Test provider',
    ...overrides,
  };
}

/**
 * Create a GuideResponse with optional overrides.
 */
export function createGuideResponse(overrides: Partial<GuideResponse> = {}): GuideResponse {
  return {
    has_task: true,
    task_id: nextId(),
    title: 'Test Task',
    state: 'planning',
    specifications: 2,
    next_actions: [
      { command: 'implement', description: 'Execute the specifications' },
      { command: 'review', description: 'Review the implementation' },
    ],
    ...overrides,
  };
}

/**
 * Create an InteractiveCommandRequest with optional overrides.
 */
export function createInteractiveCommandRequest(
  overrides: Partial<InteractiveCommandRequest> = {}
): InteractiveCommandRequest {
  return {
    command: 'plan',
    args: [],
    ...overrides,
  };
}

/**
 * Create an InteractiveChatRequest with optional overrides.
 */
export function createInteractiveChatRequest(
  overrides: Partial<InteractiveChatRequest> = {}
): InteractiveChatRequest {
  return {
    message: 'Hello, how can I help?',
    ...overrides,
  };
}

/**
 * Create an InteractiveCommandResponse with optional overrides.
 */
export function createInteractiveCommandResponse(
  overrides: Partial<InteractiveCommandResponse> = {}
): InteractiveCommandResponse {
  return {
    success: true,
    message: 'Command executed successfully',
    ...overrides,
  };
}

/**
 * Create an InteractiveChatResponse with optional overrides.
 */
export function createInteractiveChatResponse(
  overrides: Partial<InteractiveChatResponse> = {}
): InteractiveChatResponse {
  return {
    success: true,
    message: 'Here is my response',
    ...overrides,
  };
}

/**
 * Create a StateChangedEvent with optional overrides.
 */
export function createStateChangedEvent(
  overrides: Partial<StateChangedEvent> = {}
): StateChangedEvent {
  return {
    from: 'idle',
    to: 'planning',
    event: 'plan_started',
    task_id: nextId(),
    timestamp: new Date().toISOString(),
    ...overrides,
  };
}

/**
 * Create an AgentMessageEvent with optional overrides.
 */
export function createAgentMessageEvent(
  overrides: Partial<AgentMessageEvent> = {}
): AgentMessageEvent {
  return {
    role: 'assistant',
    content: 'This is a test message from the agent.',
    timestamp: new Date().toISOString(),
    ...overrides,
  };
}

// ============================================================================
// Find & Search Factories
// ============================================================================

/**
 * Create a FindMatch with optional overrides.
 */
export function createFindMatch(overrides: Partial<FindMatch> = {}): FindMatch {
  return {
    file: 'src/app.ts',
    line: 10,
    snippet: 'function handleRequest() {',
    ...overrides,
  };
}

/**
 * Create a FindSearchResponse with optional overrides.
 */
export function createFindSearchResponse(
  overrides: Partial<FindSearchResponse> = {}
): FindSearchResponse {
  return {
    query: 'test',
    count: 2,
    matches: [
      createFindMatch({ file: 'src/app.ts', line: 10 }),
      createFindMatch({ file: 'src/utils.ts', line: 25 }),
    ],
    ...overrides,
  };
}

// ============================================================================
// Memory Factories
// ============================================================================

/**
 * Create a MemoryResult with optional overrides.
 */
export function createMemoryResult(overrides: Partial<MemoryResult> = {}): MemoryResult {
  return {
    task_id: nextId(),
    type: 'spec',
    score: 0.95,
    content: 'Test memory content',
    ...overrides,
  };
}

/**
 * Create a MemorySearchResponse with optional overrides.
 */
export function createMemorySearchResponse(
  overrides: Partial<MemorySearchResponse> = {}
): MemorySearchResponse {
  return {
    results: [createMemoryResult()],
    count: 1,
    ...overrides,
  };
}

/**
 * Create a MemoryIndexResponse with optional overrides.
 */
export function createMemoryIndexResponse(
  overrides: Partial<MemoryIndexResponse> = {}
): MemoryIndexResponse {
  return {
    success: true,
    message: 'Indexed 5 documents',
    task_id: nextId(),
    ...overrides,
  };
}

/**
 * Create a MemoryStatsResponse with optional overrides.
 */
export function createMemoryStatsResponse(
  overrides: Partial<MemoryStatsResponse> = {}
): MemoryStatsResponse {
  return {
    total_documents: 100,
    by_type: { spec: 50, note: 30, session: 20 },
    enabled: true,
    ...overrides,
  };
}

// ============================================================================
// Library Factories
// ============================================================================

/**
 * Create a LibraryCollection with optional overrides.
 */
export function createLibraryCollection(
  overrides: Partial<LibraryCollection> = {}
): LibraryCollection {
  return {
    id: nextId(),
    name: 'Test Collection',
    source: 'https://docs.example.com',
    source_type: 'web',
    include_mode: 'auto',
    page_count: 50,
    total_size: 500000,
    location: 'project',
    ...overrides,
  };
}

/**
 * Create a LibraryListResponse with optional overrides.
 */
export function createLibraryListResponse(
  overrides: Partial<LibraryListResponse> = {}
): LibraryListResponse {
  return {
    collections: [createLibraryCollection()],
    count: 1,
    ...overrides,
  };
}

/**
 * Create a LibraryShowResponse with optional overrides.
 */
export function createLibraryShowResponse(
  overrides: Partial<LibraryShowResponse> = {}
): LibraryShowResponse {
  return {
    collection: createLibraryCollection(),
    pages: ['page1.md', 'page2.md', 'page3.md'],
    ...overrides,
  };
}

/**
 * Create a LibraryStatsResponse with optional overrides.
 */
export function createLibraryStatsResponse(
  overrides: Partial<LibraryStatsResponse> = {}
): LibraryStatsResponse {
  return {
    total_collections: 5,
    total_pages: 250,
    total_size: 2500000,
    project_count: 3,
    shared_count: 2,
    by_mode: { auto: 3, manual: 2 },
    enabled: true,
    ...overrides,
  };
}

// ============================================================================
// Links Factories
// ============================================================================

/**
 * Create a LinkData with optional overrides.
 */
export function createLinkData(overrides: Partial<LinkData> = {}): LinkData {
  return {
    source: 'spec:1',
    target: 'decision:cache',
    context: 'specification',
    created_at: new Date().toISOString(),
    ...overrides,
  };
}

/**
 * Create a LinksListResponse with optional overrides.
 */
export function createLinksListResponse(
  overrides: Partial<LinksListResponse> = {}
): LinksListResponse {
  return {
    links: [createLinkData()],
    count: 1,
    ...overrides,
  };
}

/**
 * Create an EntityLinksResponse with optional overrides.
 */
export function createEntityLinksResponse(
  overrides: Partial<EntityLinksResponse> = {}
): EntityLinksResponse {
  return {
    entity_id: 'spec:1',
    outgoing: [createLinkData()],
    incoming: [],
    ...overrides,
  };
}

/**
 * Create an EntityResult with optional overrides.
 */
export function createEntityResult(overrides: Partial<EntityResult> = {}): EntityResult {
  return {
    entity_id: 'spec:main',
    type: 'spec',
    name: 'Main Specification',
    total_links: 5,
    ...overrides,
  };
}

/**
 * Create a LinksSearchResponse with optional overrides.
 */
export function createLinksSearchResponse(
  overrides: Partial<LinksSearchResponse> = {}
): LinksSearchResponse {
  return {
    query: 'cache',
    results: [createEntityResult()],
    count: 1,
    ...overrides,
  };
}

/**
 * Create a LinksStatsResponse with optional overrides.
 */
export function createLinksStatsResponse(
  overrides: Partial<LinksStatsResponse> = {}
): LinksStatsResponse {
  return {
    total_links: 50,
    total_sources: 20,
    total_targets: 30,
    orphan_entities: 5,
    most_linked: [createEntityResult()],
    enabled: true,
    ...overrides,
  };
}

// ============================================================================
// Browser Factories
// ============================================================================

/**
 * Create a BrowserTab with optional overrides.
 */
export function createBrowserTab(overrides: Partial<BrowserTab> = {}): BrowserTab {
  return {
    id: nextId(),
    title: 'Test Page',
    url: 'https://example.com',
    ...overrides,
  };
}

/**
 * Create a BrowserStatusResponse with optional overrides.
 */
export function createBrowserStatusResponse(
  overrides: Partial<BrowserStatusResponse> = {}
): BrowserStatusResponse {
  return {
    connected: true,
    host: 'localhost',
    port: 9222,
    tabs: [createBrowserTab()],
    ...overrides,
  };
}

/**
 * Create a BrowserTabsResponse with optional overrides.
 */
export function createBrowserTabsResponse(
  overrides: Partial<BrowserTabsResponse> = {}
): BrowserTabsResponse {
  return {
    tabs: [createBrowserTab()],
    count: 1,
    ...overrides,
  };
}

// ============================================================================
// Mock Fetch Utilities
// ============================================================================

/**
 * Create a mock fetch response.
 */
export function createMockFetchResponse<T>(
  data: T,
  status: number = 200,
  headers: Record<string, string> = {}
): Response {
  return {
    ok: status >= 200 && status < 300,
    status,
    statusText: status === 200 ? 'OK' : 'Error',
    json: () => Promise.resolve(data),
    text: () => Promise.resolve(JSON.stringify(data)),
    headers: new Headers(headers),
    clone: function () {
      return this;
    },
    body: null,
    bodyUsed: false,
    arrayBuffer: () => Promise.resolve(new ArrayBuffer(0)),
    blob: () => Promise.resolve(new Blob()),
    formData: () => Promise.resolve(new FormData()),
    redirected: false,
    type: 'basic' as const,
    url: '',
  } as Response;
}

/**
 * Setup mock fetch with a response.
 */
export function setupMockFetch<T>(response: T, status: number = 200): void {
  global.fetch = (): Promise<Response> =>
    Promise.resolve(createMockFetchResponse(response, status));
}

/**
 * Setup mock fetch that captures request details.
 */
export function setupCapturingMockFetch<T>(
  response: T,
  status: number = 200
): {
  getLastRequest: () => { url: string; init?: RequestInit } | undefined;
  getCapturedBody: () => unknown;
} {
  let lastRequest: { url: string; init?: RequestInit } | undefined;
  let capturedBody: unknown;

  global.fetch = ((url: string | URL | Request, init?: RequestInit): Promise<Response> => {
    lastRequest = {
      url: typeof url === 'string' ? url : url instanceof URL ? url.href : 'unknown',
      init,
    };
    if (init?.body) {
      try {
        capturedBody = JSON.parse(init.body as string);
      } catch {
        capturedBody = init.body;
      }
    }
    return Promise.resolve(createMockFetchResponse(response, status));
  }) as typeof fetch;

  return {
    getLastRequest: () => lastRequest,
    getCapturedBody: () => capturedBody,
  };
}

/**
 * Setup mock fetch that fails.
 */
export function setupFailingMockFetch(error: Error = new Error('Network error')): void {
  global.fetch = (): Promise<Response> => Promise.reject(error);
}

/**
 * Restore original fetch (call in teardown).
 */
let originalFetch: typeof fetch | undefined;

export function saveFetch(): void {
  originalFetch = global.fetch;
}

export function restoreFetch(): void {
  if (originalFetch) {
    global.fetch = originalFetch;
  }
}
