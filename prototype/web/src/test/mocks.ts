import { vi } from 'vitest'
import type {
  StatusResponse,
  TaskResponse,
  TaskHistoryItem,
  WorkspaceConfig,
} from '@/types/api'
import type { SettingsResponseV2 } from '@/types/schema'

// =============================================================================
// Mock Status Responses
// =============================================================================

export const mockProjectModeStatus: StatusResponse = {
  mode: 'project',
  running: true,
  port: 8080,
  state: 'idle',
  project: {
    id: 'github.com-acme-repo',
    name: 'acme/repo',
    path: '/tmp/acme/repo',
    remote_url: 'https://github.com/acme/repo.git',
  },
}

export const mockGlobalModeStatus: StatusResponse = {
  mode: 'global',
  running: true,
  port: 8080,
}

// =============================================================================
// Mock Task Responses
// =============================================================================

export const mockActiveTask: TaskResponse = {
  active: true,
  task: {
    id: 'task-123',
    state: 'implementing',
    ref: 'github:456',
    branch: 'feature/test-branch',
    worktree_path: '/tmp/worktree',
    started: new Date().toISOString(),
  },
  work: {
    title: 'Test Task',
    external_key: 'ISSUE-123',
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
    costs: {
      total_input_tokens: 10000,
      total_output_tokens: 5000,
      total_cost_usd: 0.15,
    },
  },
}

export const mockNoActiveTask: TaskResponse = {
  active: false,
}

// =============================================================================
// Mock Task History
// =============================================================================

export const mockTaskHistory: TaskHistoryItem[] = [
  {
    id: 'task-1',
    title: 'First Task',
    state: 'done',
    created_at: '2026-01-01T12:00:00Z',
  },
  {
    id: 'task-2',
    title: 'Second Task',
    state: 'implementing',
    created_at: '2026-01-02T12:00:00Z',
    worktree_path: '/tmp/worktree/task-2',
  },
]

// =============================================================================
// Mock Settings
// =============================================================================

export const mockSettingsValues: Partial<WorkspaceConfig> = {
  git: {
    commit_prefix: '[MEHR-{key}]',
    branch_pattern: 'mehr/{type}/{slug}',
    auto_commit: true,
    sign_commits: false,
    stash_on_start: false,
    auto_pop_stash: false,
  },
  agent: {
    default: 'claude',
    timeout: 300,
    max_retries: 3,
  },
  workflow: {
    auto_init: true,
    session_retention_days: 30,
    delete_work_on_finish: false,
    delete_work_on_abandon: false,
    prefer_local_merge: false,
  },
}

// Schema-driven settings response format
export const mockSettings: SettingsResponseV2 = {
  schema: {
    version: '1',
    sections: [
      {
        id: 'git',
        title: 'Git',
        description: 'Version control settings',
        icon: 'git-branch',
        category: 'core',
        fields: [
          { path: 'git.auto_commit', type: 'boolean', label: 'Auto Commit', simple: true },
          { path: 'git.commit_prefix', type: 'string', label: 'Commit Prefix', simple: true },
        ],
      },
      {
        id: 'agent',
        title: 'Agent',
        description: 'AI agent configuration',
        icon: 'bot',
        category: 'core',
        fields: [
          { path: 'agent.default', type: 'string', label: 'Default Agent', simple: true },
        ],
      },
      {
        id: 'budget',
        title: 'Budget',
        description: 'Cost and budget controls',
        icon: 'wallet',
        category: 'core',
        fields: [
          { path: 'budget.per_task.max_cost', type: 'number', label: 'Max Cost Per Task' },
        ],
      },
      {
        id: 'browser',
        title: 'Browser Automation',
        description: 'Chrome DevTools Protocol settings',
        icon: 'globe',
        category: 'features',
        fields: [
          { path: 'browser.enabled', type: 'boolean', label: 'Enable Browser', advanced: true },
        ],
      },
    ],
  },
  values: mockSettingsValues as Record<string, unknown>,
}

// =============================================================================
// Mock Fetch Helpers
// =============================================================================

export function mockFetchResponse(data: unknown, ok = true) {
  return vi.fn().mockResolvedValue({
    ok,
    status: ok ? 200 : 400,
    json: () => Promise.resolve(data),
    text: () => Promise.resolve(JSON.stringify(data)),
  })
}

export function mockApiEndpoints(overrides: Record<string, unknown> = {}) {
  const defaults: Record<string, unknown> = {
    '/api/v1/status': mockProjectModeStatus,
    '/api/v1/task': mockNoActiveTask,
    '/api/v1/tasks': { tasks: mockTaskHistory, count: mockTaskHistory.length },
    '/api/v1/auth/csrf': { csrf_token: 'test-csrf-token' },
    '/api/v1/settings': mockSettings,
    '/api/v1/agents': { agents: [{ name: 'claude', type: 'cli', available: true }], count: 1 },
    '/api/v1/budget': { monthly: { used: 0, max: 100, currency: 'USD' } },
  }

  const endpoints = { ...defaults, ...overrides }

  global.fetch = vi.fn().mockImplementation((url: string) => {
    // Normalize URL to path
    const path = url.replace(/^.*\/api\/v1/, '/api/v1').split('?')[0]

    const data = endpoints[path]

    if (data) {
      return Promise.resolve({
        ok: true,
        status: 200,
        json: () => Promise.resolve(data),
        text: () => Promise.resolve(JSON.stringify(data)),
      })
    }

    return Promise.resolve({
      ok: false,
      status: 404,
      json: () => Promise.resolve({ error: 'Not found' }),
      text: () => Promise.resolve('Not found'),
    })
  })
}

// =============================================================================
// Mock Workflow SSE Hook
// =============================================================================

export const mockUseWorkflowSSE = {
  connected: true,
  lastEvent: null,
  error: null,
}
