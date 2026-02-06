import { describe, test, expect } from 'bun:test';
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

describe('API Models Test Suite', () => {
  test('StatusResponse type structure', () => {
    const status: StatusResponse = {
      mode: 'project',
      running: true,
      port: 3000,
      state: 'idle',
    };

    expect(status.mode).toBe('project');
    expect(status.running).toBe(true);
    expect(status.port).toBe(3000);
    expect(status.state).toBe('idle');
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

    expect(task.active).toBe(true);
    expect(task.task?.id).toBe('task-123');
    expect(task.work?.title).toBe('Test Task');
    expect(task.pending_question?.question).toBe('Which approach should I use?');
    expect(task.pending_question?.options?.length).toBe(2);
  });

  test('TaskInfo minimal fields', () => {
    const info: TaskInfo = {
      id: 'task-456',
      state: 'idle',
      ref: 'github:456',
    };

    expect(info.id).toBe('task-456');
    expect(info.state).toBe('idle');
    expect(info.ref).toBe('github:456');
    expect(info.branch).toBe(undefined);
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

    expect(work.costs?.total_tokens).toBe(5000);
    expect(work.costs?.total_cost_usd).toBe(0.25);
  });

  test('PendingQuestion with options', () => {
    const question: PendingQuestion = {
      question: 'Select a framework',
      options: ['React', 'Vue', 'Svelte'],
    };

    expect(question.question).toBe('Select a framework');
    expect(question.options?.length).toBe(3);
  });

  test('PendingQuestion without options', () => {
    const question: PendingQuestion = {
      question: 'Please describe the expected behavior',
    };

    expect(question.options).toBe(undefined);
  });

  test('WorkflowResponse success', () => {
    const response: WorkflowResponse = {
      success: true,
      state: 'planning',
      message: 'Planning started successfully',
    };

    expect(response.success).toBe(true);
    expect(response.state).toBe('planning');
  });

  test('WorkflowResponse error', () => {
    const response: WorkflowResponse = {
      success: false,
      error: 'No active task',
    };

    expect(response.success).toBe(false);
    expect(response.error).toBe('No active task');
  });

  test('CostInfo all fields', () => {
    const cost: CostInfo = {
      total_tokens: 10000,
      input_tokens: 6000,
      output_tokens: 4000,
      cached_tokens: 2000,
      total_cost_usd: 0.5,
    };

    expect(cost.total_tokens).toBe(10000);
    expect(cost.total_cost_usd).toBe(0.5);
  });

  test('TaskCostResponse with by_step', () => {
    const cost: TaskCostResponse = {
      task_id: 'task-789',
      title: 'Complex Task',
      total_tokens: 20000,
      input_tokens: 12000,
      output_tokens: 8000,
      cached_tokens: 4000,
      total_cost_usd: 1.0,
      by_step: {
        planning: {
          input_tokens: 4000,
          output_tokens: 2000,
          cached_tokens: 1000,
          total_tokens: 6000,
          cost_usd: 0.3,
          calls: 2,
        },
        implementing: {
          input_tokens: 8000,
          output_tokens: 6000,
          cached_tokens: 3000,
          total_tokens: 14000,
          cost_usd: 0.7,
          calls: 5,
        },
      },
    };

    expect(cost.task_id).toBe('task-789');
    expect(cost.by_step?.planning.calls).toBe(2);
    expect(cost.by_step?.implementing.cost_usd).toBe(0.7);
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

    expect(agent.name).toBe('claude');
    expect(agent.capabilities?.streaming).toBe(true);
    expect(agent.models?.length).toBe(1);
  });

  test('InteractiveCommandRequest', () => {
    const request: InteractiveCommandRequest = {
      command: 'plan',
      args: ['--force'],
    };

    expect(request.command).toBe('plan');
    expect(request.args).toEqual(['--force']);
  });

  test('InteractiveStateResponse', () => {
    const state: InteractiveStateResponse = {
      success: true,
      state: 'implementing',
      task_id: 'task-abc',
      title: 'Current Task',
    };

    expect(state.success).toBe(true);
    expect(state.state).toBe('implementing');
    expect(state.task_id).toBe('task-abc');
  });

  test('SSE event types are valid', () => {
    const eventTypes: SSEEventType[] = [
      'state_changed',
      'progress',
      'error',
      'agent_message',
      'heartbeat',
    ];

    expect(eventTypes.length).toBe(5);
    expect(eventTypes.includes('state_changed')).toBeTruthy();
    expect(eventTypes.includes('agent_message')).toBeTruthy();
  });

  test('StateChangedEvent structure', () => {
    const event: StateChangedEvent = {
      from: 'idle',
      to: 'planning',
      event: 'PLAN_REQUESTED',
      task_id: 'task-xyz',
      timestamp: '2026-01-31T12:00:00Z',
    };

    expect(event.from).toBe('idle');
    expect(event.to).toBe('planning');
    expect(event.event).toBe('PLAN_REQUESTED');
  });

  test('AgentMessageEvent structure', () => {
    const event: AgentMessageEvent = {
      task_id: 'task-xyz',
      content: 'Analyzing codebase...',
      role: 'assistant',
      timestamp: '2026-01-31T12:01:00Z',
    };

    expect(event.content).toBe('Analyzing codebase...');
    expect(event.role).toBe('assistant');
  });
});
