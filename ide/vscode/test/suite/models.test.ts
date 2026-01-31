import * as assert from 'assert';
import type {
  StatusResponse,
  TaskResponse,
  TaskInfo,
  TaskWork,
  PendingQuestion,
  WorkflowResponse,
  CostInfo,
  TaskCostResponse,
  AgentInfo,
  InteractiveCommandRequest,
  InteractiveStateResponse,
  SSEEventType,
  StateChangedEvent,
  AgentMessageEvent,
} from '../../src/api/models';

suite('API Models Test Suite', () => {
  test('StatusResponse type structure', () => {
    const status: StatusResponse = {
      mode: 'project',
      running: true,
      port: 3000,
      state: 'idle',
    };

    assert.strictEqual(status.mode, 'project');
    assert.strictEqual(status.running, true);
    assert.strictEqual(status.port, 3000);
    assert.strictEqual(status.state, 'idle');
  });

  test('TaskResponse with all fields', () => {
    const task: TaskResponse = {
      active: true,
      task: {
        id: 'task-123',
        state: 'planning',
        ref: 'file:task.md',
        branch: 'feature/task-123',
        worktree_path: '/path/to/worktree',
        started: '2026-01-31T10:00:00Z',
      },
      work: {
        title: 'Test Task',
        external_key: 'GH-123',
        created_at: '2026-01-31T10:00:00Z',
        updated_at: '2026-01-31T11:00:00Z',
        costs: {
          total_tokens: 1000,
          input_tokens: 600,
          output_tokens: 400,
          cached_tokens: 200,
          total_cost_usd: 0.05,
        },
      },
      pending_question: {
        question: 'Which approach should I use?',
        options: ['Option A', 'Option B'],
      },
    };

    assert.strictEqual(task.active, true);
    assert.strictEqual(task.task?.id, 'task-123');
    assert.strictEqual(task.work?.title, 'Test Task');
    assert.strictEqual(task.pending_question?.question, 'Which approach should I use?');
    assert.strictEqual(task.pending_question?.options?.length, 2);
  });

  test('TaskInfo minimal fields', () => {
    const info: TaskInfo = {
      id: 'task-456',
      state: 'idle',
      ref: 'github:456',
    };

    assert.strictEqual(info.id, 'task-456');
    assert.strictEqual(info.state, 'idle');
    assert.strictEqual(info.ref, 'github:456');
    assert.strictEqual(info.branch, undefined);
  });

  test('TaskWork with costs', () => {
    const work: TaskWork = {
      title: 'Feature Implementation',
      costs: {
        total_tokens: 5000,
        input_tokens: 3000,
        output_tokens: 2000,
        cached_tokens: 1000,
        total_cost_usd: 0.25,
      },
    };

    assert.strictEqual(work.costs?.total_tokens, 5000);
    assert.strictEqual(work.costs?.total_cost_usd, 0.25);
  });

  test('PendingQuestion with options', () => {
    const question: PendingQuestion = {
      question: 'Select a framework',
      options: ['React', 'Vue', 'Svelte'],
    };

    assert.strictEqual(question.question, 'Select a framework');
    assert.strictEqual(question.options?.length, 3);
  });

  test('PendingQuestion without options', () => {
    const question: PendingQuestion = {
      question: 'Please describe the expected behavior',
    };

    assert.strictEqual(question.options, undefined);
  });

  test('WorkflowResponse success', () => {
    const response: WorkflowResponse = {
      success: true,
      state: 'planning',
      message: 'Planning started successfully',
    };

    assert.strictEqual(response.success, true);
    assert.strictEqual(response.state, 'planning');
  });

  test('WorkflowResponse error', () => {
    const response: WorkflowResponse = {
      success: false,
      error: 'No active task',
    };

    assert.strictEqual(response.success, false);
    assert.strictEqual(response.error, 'No active task');
  });

  test('CostInfo all fields', () => {
    const cost: CostInfo = {
      total_tokens: 10000,
      input_tokens: 6000,
      output_tokens: 4000,
      cached_tokens: 2000,
      total_cost_usd: 0.50,
    };

    assert.strictEqual(cost.total_tokens, 10000);
    assert.strictEqual(cost.total_cost_usd, 0.50);
  });

  test('TaskCostResponse with by_step', () => {
    const cost: TaskCostResponse = {
      task_id: 'task-789',
      title: 'Complex Task',
      total_tokens: 20000,
      input_tokens: 12000,
      output_tokens: 8000,
      cached_tokens: 4000,
      total_cost_usd: 1.00,
      by_step: {
        planning: {
          input_tokens: 4000,
          output_tokens: 2000,
          cached_tokens: 1000,
          total_tokens: 6000,
          cost_usd: 0.30,
          calls: 2,
        },
        implementing: {
          input_tokens: 8000,
          output_tokens: 6000,
          cached_tokens: 3000,
          total_tokens: 14000,
          cost_usd: 0.70,
          calls: 5,
        },
      },
    };

    assert.strictEqual(cost.task_id, 'task-789');
    assert.strictEqual(cost.by_step?.planning.calls, 2);
    assert.strictEqual(cost.by_step?.implementing.cost_usd, 0.70);
  });

  test('AgentInfo with capabilities', () => {
    const agent: AgentInfo = {
      name: 'claude',
      type: 'cli',
      available: true,
      description: 'Claude CLI agent',
      capabilities: {
        streaming: true,
        tool_use: true,
        file_operations: true,
        code_execution: true,
        multi_turn: true,
        system_prompt: true,
        allowed_tools: ['read', 'write', 'bash'],
      },
      models: [
        {
          id: 'claude-sonnet-4',
          name: 'Claude Sonnet 4',
          default: true,
          max_tokens: 64000,
        },
      ],
    };

    assert.strictEqual(agent.name, 'claude');
    assert.strictEqual(agent.capabilities?.streaming, true);
    assert.strictEqual(agent.models?.length, 1);
  });

  test('InteractiveCommandRequest', () => {
    const request: InteractiveCommandRequest = {
      command: 'plan',
      args: ['--force'],
    };

    assert.strictEqual(request.command, 'plan');
    assert.deepStrictEqual(request.args, ['--force']);
  });

  test('InteractiveStateResponse', () => {
    const state: InteractiveStateResponse = {
      success: true,
      state: 'implementing',
      task_id: 'task-abc',
      title: 'Current Task',
    };

    assert.strictEqual(state.success, true);
    assert.strictEqual(state.state, 'implementing');
    assert.strictEqual(state.task_id, 'task-abc');
  });

  test('SSE event types are valid', () => {
    const eventTypes: SSEEventType[] = [
      'state_changed',
      'progress',
      'error',
      'agent_message',
      'heartbeat',
    ];

    assert.strictEqual(eventTypes.length, 5);
    assert.ok(eventTypes.includes('state_changed'));
    assert.ok(eventTypes.includes('agent_message'));
  });

  test('StateChangedEvent structure', () => {
    const event: StateChangedEvent = {
      from: 'idle',
      to: 'planning',
      event: 'PLAN_REQUESTED',
      task_id: 'task-xyz',
      timestamp: '2026-01-31T12:00:00Z',
    };

    assert.strictEqual(event.from, 'idle');
    assert.strictEqual(event.to, 'planning');
    assert.strictEqual(event.event, 'PLAN_REQUESTED');
  });

  test('AgentMessageEvent structure', () => {
    const event: AgentMessageEvent = {
      task_id: 'task-xyz',
      content: 'Analyzing codebase...',
      role: 'assistant',
      timestamp: '2026-01-31T12:01:00Z',
    };

    assert.strictEqual(event.content, 'Analyzing codebase...');
    assert.strictEqual(event.role, 'assistant');
  });
});
