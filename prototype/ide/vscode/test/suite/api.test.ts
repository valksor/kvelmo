import * as assert from 'assert';
import { MehrhofApiClient, ApiError } from '../../src/api/client';
import type {
  StatusResponse,
  TaskResponse,
  TaskListResponse,
  WorkflowResponse,
  GuideResponse,
} from '../../src/api/models';

suite('API Client Test Suite', () => {
  // Mock fetch for testing
  let originalFetch: typeof global.fetch;

  setup(() => {
    originalFetch = global.fetch;
  });

  teardown(() => {
    global.fetch = originalFetch;
  });

  function setupMockFetch(response: unknown, status = 200): void {
    global.fetch = (): Promise<Response> =>
      Promise.resolve({
        ok: status >= 200 && status < 300,
        status,
        statusText: status === 200 ? 'OK' : 'Error',
        json: () => Promise.resolve(response),
        headers: new Headers(),
      } as Response);
  }

  test('MehrhofApiClient constructs with base URL', () => {
    const client = new MehrhofApiClient('http://localhost:3000');
    assert.strictEqual(client.getBaseUrl(), 'http://localhost:3000');
  });

  test('MehrhofApiClient removes trailing slash from base URL', () => {
    const client = new MehrhofApiClient('http://localhost:3000/');
    assert.strictEqual(client.getBaseUrl(), 'http://localhost:3000');
  });

  test('getEventsUrl returns correct SSE endpoint', () => {
    const client = new MehrhofApiClient('http://localhost:3000');
    assert.strictEqual(client.getEventsUrl(), 'http://localhost:3000/api/v1/events');
  });

  test('health returns true on success', async () => {
    setupMockFetch({});
    const client = new MehrhofApiClient('http://localhost:3000');
    const result = await client.health();
    assert.strictEqual(result, true);
  });

  test('health returns false on failure', async () => {
    global.fetch = (): Promise<Response> => Promise.reject(new Error('Connection refused'));
    const client = new MehrhofApiClient('http://localhost:3000');
    const result = await client.health();
    assert.strictEqual(result, false);
  });

  test('getStatus returns StatusResponse', async () => {
    const mockResponse: StatusResponse = {
      mode: 'project',
      running: true,
      port: 3000,
      state: 'idle',
    };
    setupMockFetch(mockResponse);

    const client = new MehrhofApiClient('http://localhost:3000');
    const result = await client.getStatus();

    assert.strictEqual(result.mode, 'project');
    assert.strictEqual(result.running, true);
    assert.strictEqual(result.port, 3000);
  });

  test('getTask returns TaskResponse', async () => {
    const mockResponse: TaskResponse = {
      active: true,
      task: {
        id: 'task-123',
        state: 'planning',
        ref: 'file:task.md',
        branch: 'feature/task-123',
      },
      work: {
        title: 'Test Task',
        created_at: '2026-01-31T10:00:00Z',
      },
    };
    setupMockFetch(mockResponse);

    const client = new MehrhofApiClient('http://localhost:3000');
    const result = await client.getTask();

    assert.strictEqual(result.active, true);
    assert.strictEqual(result.task?.id, 'task-123');
    assert.strictEqual(result.task?.state, 'planning');
    assert.strictEqual(result.work?.title, 'Test Task');
  });

  test('getTasks returns TaskListResponse', async () => {
    const mockResponse: TaskListResponse = {
      tasks: [
        { id: 'task-1', state: 'done', title: 'Task 1' },
        { id: 'task-2', state: 'planning', title: 'Task 2' },
      ],
      count: 2,
    };
    setupMockFetch(mockResponse);

    const client = new MehrhofApiClient('http://localhost:3000');
    const result = await client.getTasks();

    assert.strictEqual(result.count, 2);
    assert.strictEqual(result.tasks.length, 2);
    assert.strictEqual(result.tasks[0].id, 'task-1');
  });

  test('plan returns WorkflowResponse', async () => {
    const mockResponse: WorkflowResponse = {
      success: true,
      state: 'planning',
      message: 'Planning started',
    };
    setupMockFetch(mockResponse);

    const client = new MehrhofApiClient('http://localhost:3000');
    const result = await client.plan();

    assert.strictEqual(result.success, true);
    assert.strictEqual(result.state, 'planning');
  });

  test('implement returns WorkflowResponse', async () => {
    const mockResponse: WorkflowResponse = {
      success: true,
      state: 'implementing',
    };
    setupMockFetch(mockResponse);

    const client = new MehrhofApiClient('http://localhost:3000');
    const result = await client.implement();

    assert.strictEqual(result.success, true);
    assert.strictEqual(result.state, 'implementing');
  });

  test('review returns WorkflowResponse', async () => {
    const mockResponse: WorkflowResponse = {
      success: true,
      state: 'reviewing',
    };
    setupMockFetch(mockResponse);

    const client = new MehrhofApiClient('http://localhost:3000');
    const result = await client.review();

    assert.strictEqual(result.success, true);
  });

  test('undo returns WorkflowResponse', async () => {
    const mockResponse: WorkflowResponse = {
      success: true,
      message: 'Reverted to previous checkpoint',
    };
    setupMockFetch(mockResponse);

    const client = new MehrhofApiClient('http://localhost:3000');
    const result = await client.undo();

    assert.strictEqual(result.success, true);
  });

  test('redo returns WorkflowResponse', async () => {
    const mockResponse: WorkflowResponse = {
      success: true,
      message: 'Restored to next checkpoint',
    };
    setupMockFetch(mockResponse);

    const client = new MehrhofApiClient('http://localhost:3000');
    const result = await client.redo();

    assert.strictEqual(result.success, true);
  });

  test('getGuide returns GuideResponse', async () => {
    const mockResponse: GuideResponse = {
      has_task: true,
      task_id: 'task-123',
      title: 'Test Task',
      state: 'planning',
      specifications: 2,
      next_actions: [
        { command: 'implement', description: 'Execute the specifications' },
      ],
    };
    setupMockFetch(mockResponse);

    const client = new MehrhofApiClient('http://localhost:3000');
    const result = await client.getGuide();

    assert.strictEqual(result.has_task, true);
    assert.strictEqual(result.task_id, 'task-123');
    assert.strictEqual(result.next_actions.length, 1);
  });

  test('throws ApiError on HTTP error', async () => {
    setupMockFetch({ error: 'Not found' }, 404);

    const client = new MehrhofApiClient('http://localhost:3000');

    try {
      await client.getTask();
      assert.fail('Expected ApiError to be thrown');
    } catch (error) {
      assert.ok(error instanceof ApiError);
      assert.strictEqual(error.statusCode, 404);
    }
  });

  test('startTask sends correct request body', async () => {
    let capturedBody: unknown;
    global.fetch = ((_url: unknown, init?: RequestInit): Promise<Response> => {
      capturedBody = init?.body ? JSON.parse(init.body as string) as unknown : undefined;
      return Promise.resolve({
        ok: true,
        status: 200,
        json: () => Promise.resolve({ success: true }),
        headers: new Headers(),
      } as Response);
    }) as typeof fetch;

    const client = new MehrhofApiClient('http://localhost:3000');
    await client.startTask({ ref: 'github:123' });

    assert.deepStrictEqual(capturedBody, { ref: 'github:123' });
  });

  test('executeCommand sends correct request body', async () => {
    let capturedBody: unknown;
    global.fetch = ((_url: unknown, init?: RequestInit): Promise<Response> => {
      capturedBody = init?.body ? JSON.parse(init.body as string) as unknown : undefined;
      return Promise.resolve({
        ok: true,
        status: 200,
        json: () => Promise.resolve({ success: true }),
        headers: new Headers(),
      } as Response);
    }) as typeof fetch;

    const client = new MehrhofApiClient('http://localhost:3000');
    await client.executeCommand({ command: 'plan', args: ['--force'] });

    assert.deepStrictEqual(capturedBody, { command: 'plan', args: ['--force'] });
  });

  test('chat sends correct request body', async () => {
    let capturedBody: unknown;
    global.fetch = ((_url: unknown, init?: RequestInit): Promise<Response> => {
      capturedBody = init?.body ? JSON.parse(init.body as string) as unknown : undefined;
      return Promise.resolve({
        ok: true,
        status: 200,
        json: () => Promise.resolve({ success: true, message: 'Response' }),
        headers: new Headers(),
      } as Response);
    }) as typeof fetch;

    const client = new MehrhofApiClient('http://localhost:3000');
    await client.chat({ message: 'Hello' });

    assert.deepStrictEqual(capturedBody, { message: 'Hello' });
  });

  test('session cookie is set and sent', async () => {
    let capturedHeaders: Headers | undefined;
    global.fetch = ((_url: unknown, init?: RequestInit): Promise<Response> => {
      capturedHeaders = new Headers(init?.headers);
      return Promise.resolve({
        ok: true,
        status: 200,
        json: () => Promise.resolve({}),
        headers: new Headers([['set-cookie', 'mehr_session=abc123; Path=/']]),
      } as Response);
    }) as typeof fetch;

    const client = new MehrhofApiClient('http://localhost:3000');
    await client.getStatus(); // First call sets the cookie

    // Second call should send the cookie
    await client.getTask();

    assert.ok(capturedHeaders?.get('Cookie')?.includes('mehr_session'));
  });
});
