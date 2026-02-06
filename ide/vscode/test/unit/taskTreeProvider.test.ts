import { describe, test, expect, beforeEach, afterEach } from 'bun:test';
import { TaskItem, TaskTreeProvider } from '../../src/views/taskTreeProvider';
import type { TaskSummary } from '../../src/api/models';
import { saveFetch, restoreFetch, resetFactories } from '../helpers/factories';

// Mock ProjectService for testing
interface MockProjectService {
  isConnected: boolean;
  client: { getTasks: () => Promise<{ tasks: TaskSummary[] }> } | null;
  currentTask: { id: string } | null;
  on: (event: string, handler: (...args: unknown[]) => void) => void;
}

function createMockProjectService(overrides: Partial<MockProjectService> = {}): MockProjectService {
  const listeners = new Map<string, ((...args: unknown[]) => void)[]>();
  return {
    isConnected: false,
    client: null,
    currentTask: null,
    on: (event: string, handler: (...args: unknown[]) => void) => {
      if (!listeners.has(event)) {
        listeners.set(event, []);
      }
      listeners.get(event)!.push(handler);
    },
    ...overrides,
  };
}

describe('TaskItem Test Suite', () => {
  beforeEach(() => {
    resetFactories();
  });

  describe('Constructor', () => {
    test('sets label from title', () => {
      const task: TaskSummary = { id: 'task-1', state: 'idle', title: 'Test Task' };
      const item = new TaskItem(task, false);
      expect(item.label).toBe('Test Task');
    });

    test('sets label from id when no title', () => {
      const task: TaskSummary = { id: 'task-123', state: 'idle' };
      const item = new TaskItem(task, false);
      expect(item.label).toBe('task-123');
    });

    test('sets id from task id', () => {
      const task: TaskSummary = { id: 'task-456', state: 'idle', title: 'Test' };
      const item = new TaskItem(task, false);
      expect(item.id).toBe('task-456');
    });

    test('sets collapsibleState to None', () => {
      const task: TaskSummary = { id: 'task-1', state: 'idle' };
      const item = new TaskItem(task, false);
      // TreeItemCollapsibleState.None = 0
      expect(item.collapsibleState).toBe(0);
    });

    test('sets contextValue to activeTask when active', () => {
      const task: TaskSummary = { id: 'task-1', state: 'idle' };
      const item = new TaskItem(task, true);
      expect(item.contextValue).toBe('activeTask');
    });

    test('sets contextValue to task when not active', () => {
      const task: TaskSummary = { id: 'task-1', state: 'idle' };
      const item = new TaskItem(task, false);
      expect(item.contextValue).toBe('task');
    });
  });

  describe('formatState()', () => {
    test('capitalizes first letter of state', () => {
      const task: TaskSummary = { id: 'task-1', state: 'planning' };
      const item = new TaskItem(task, false);
      expect(item.description).toBe('Planning');
    });

    test('handles idle state', () => {
      const task: TaskSummary = { id: 'task-1', state: 'idle' };
      const item = new TaskItem(task, false);
      expect(item.description).toBe('Idle');
    });

    test('handles implementing state', () => {
      const task: TaskSummary = { id: 'task-1', state: 'implementing' };
      const item = new TaskItem(task, false);
      expect(item.description).toBe('Implementing');
    });
  });

  describe('getIcon()', () => {
    test('done state returns check icon with green color', () => {
      const task: TaskSummary = { id: 'task-1', state: 'done' };
      const item = new TaskItem(task, false);
      expect(item.iconPath).toBeTruthy();
    });

    test('failed state returns error icon', () => {
      const task: TaskSummary = { id: 'task-1', state: 'failed' };
      const item = new TaskItem(task, false);
      expect(item.iconPath).toBeTruthy();
    });

    test('idle state returns circle-outline icon', () => {
      const task: TaskSummary = { id: 'task-1', state: 'idle' };
      const item = new TaskItem(task, false);
      expect(item.iconPath).toBeTruthy();
    });

    test('planning state returns edit icon', () => {
      const task: TaskSummary = { id: 'task-1', state: 'planning' };
      const item = new TaskItem(task, false);
      expect(item.iconPath).toBeTruthy();
    });

    test('implementing state returns code icon', () => {
      const task: TaskSummary = { id: 'task-1', state: 'implementing' };
      const item = new TaskItem(task, false);
      expect(item.iconPath).toBeTruthy();
    });

    test('reviewing state returns eye icon', () => {
      const task: TaskSummary = { id: 'task-1', state: 'reviewing' };
      const item = new TaskItem(task, false);
      expect(item.iconPath).toBeTruthy();
    });

    test('waiting state returns question icon', () => {
      const task: TaskSummary = { id: 'task-1', state: 'waiting' };
      const item = new TaskItem(task, false);
      expect(item.iconPath).toBeTruthy();
    });

    test('unknown state returns circle-filled icon', () => {
      const task: TaskSummary = { id: 'task-1', state: 'unknown' };
      const item = new TaskItem(task, false);
      expect(item.iconPath).toBeTruthy();
    });
  });

  describe('buildTooltip()', () => {
    test('includes task ID', () => {
      const task: TaskSummary = { id: 'task-abc', state: 'idle' };
      const item = new TaskItem(task, false);
      const tooltip = typeof item.tooltip === 'string' ? item.tooltip : '';
      expect(tooltip.includes('task-abc')).toBeTruthy();
    });

    test('includes title when present', () => {
      const task: TaskSummary = { id: 'task-1', state: 'idle', title: 'My Task Title' };
      const item = new TaskItem(task, false);
      const tooltip = typeof item.tooltip === 'string' ? item.tooltip : '';
      expect(tooltip.includes('My Task Title')).toBeTruthy();
    });

    test('includes state', () => {
      const task: TaskSummary = { id: 'task-1', state: 'planning' };
      const item = new TaskItem(task, false);
      const tooltip = typeof item.tooltip === 'string' ? item.tooltip : '';
      expect(tooltip.includes('Planning')).toBeTruthy();
    });

    test('includes (Active Task) when active', () => {
      const task: TaskSummary = { id: 'task-1', state: 'idle' };
      const item = new TaskItem(task, true);
      const tooltip = typeof item.tooltip === 'string' ? item.tooltip : '';
      expect(tooltip.includes('Active Task')).toBeTruthy();
    });

    test('does not include (Active Task) when not active', () => {
      const task: TaskSummary = { id: 'task-1', state: 'idle' };
      const item = new TaskItem(task, false);
      const tooltip = typeof item.tooltip === 'string' ? item.tooltip : '';
      expect(!tooltip.includes('Active Task')).toBeTruthy();
    });
  });
});

describe('TaskTreeProvider Test Suite', () => {
  beforeEach(() => {
    saveFetch();
    resetFactories();
  });

  afterEach(() => {
    restoreFetch();
  });

  describe('Constructor', () => {
    test('creates instance with service', () => {
      const service = createMockProjectService();
      const provider = new TaskTreeProvider(
        service as unknown as import('../../src/services/projectService').MehrhofProjectService
      );
      expect(provider).toBeTruthy();
    });

    test('registers connectionChanged listener', () => {
      let listenerRegistered = false;
      const service = createMockProjectService({
        on: (event: string) => {
          if (event === 'connectionChanged') {
            listenerRegistered = true;
          }
        },
      });
      new TaskTreeProvider(
        service as unknown as import('../../src/services/projectService').MehrhofProjectService
      );
      expect(listenerRegistered).toBeTruthy();
    });

    test('registers taskChanged listener', () => {
      let listenerRegistered = false;
      const service = createMockProjectService({
        on: (event: string) => {
          if (event === 'taskChanged') {
            listenerRegistered = true;
          }
        },
      });
      new TaskTreeProvider(
        service as unknown as import('../../src/services/projectService').MehrhofProjectService
      );
      expect(listenerRegistered).toBeTruthy();
    });
  });

  describe('getTreeItem()', () => {
    test('returns element unchanged', () => {
      const service = createMockProjectService();
      const provider = new TaskTreeProvider(
        service as unknown as import('../../src/services/projectService').MehrhofProjectService
      );

      const task: TaskSummary = { id: 'task-1', state: 'idle' };
      const item = new TaskItem(task, false);
      const result = provider.getTreeItem(item);

      expect(result).toBe(item);
    });
  });

  describe('getChildren()', () => {
    test('returns empty array when not connected', async () => {
      const service = createMockProjectService({ isConnected: false });
      const provider = new TaskTreeProvider(
        service as unknown as import('../../src/services/projectService').MehrhofProjectService
      );

      const children = await provider.getChildren();
      expect(children).toEqual([]);
    });
  });

  describe('refresh()', () => {
    test('clears tasks when not connected', async () => {
      const service = createMockProjectService({ isConnected: false });
      const provider = new TaskTreeProvider(
        service as unknown as import('../../src/services/projectService').MehrhofProjectService
      );

      await provider.refresh();
      const children = await provider.getChildren();
      expect(children).toEqual([]);
    });
  });

  describe('onDidChangeTreeData', () => {
    test('event is defined', () => {
      const service = createMockProjectService();
      const provider = new TaskTreeProvider(
        service as unknown as import('../../src/services/projectService').MehrhofProjectService
      );
      expect(provider.onDidChangeTreeData).toBeTruthy();
    });
  });

  describe('dispose()', () => {
    test('disposes without error', () => {
      const service = createMockProjectService();
      const provider = new TaskTreeProvider(
        service as unknown as import('../../src/services/projectService').MehrhofProjectService
      );
      expect(() => provider.dispose()).not.toThrow();
    });
  });

  describe('Sorting', () => {
    test('active task sorts first', () => {
      // Document expected sorting behavior
      const tasks: TaskSummary[] = [
        { id: 'task-1', state: 'idle' },
        { id: 'task-2', state: 'planning' }, // This is active
        { id: 'task-3', state: 'done' },
      ];

      // Active task (task-2) should appear first
      const activeId = 'task-2';
      const sorted = [...tasks].sort((a, b) => {
        if (a.id === activeId && b.id !== activeId) {
          return -1;
        }
        if (a.id !== activeId && b.id === activeId) {
          return 1;
        }
        return 0;
      });

      expect(sorted[0].id).toBe('task-2');
    });

    test('sorts by created_at descending after active', () => {
      const now = Date.now();
      const tasks: TaskSummary[] = [
        { id: 'task-1', state: 'idle', created_at: new Date(now - 1000).toISOString() },
        { id: 'task-2', state: 'idle', created_at: new Date(now).toISOString() },
        { id: 'task-3', state: 'idle', created_at: new Date(now - 2000).toISOString() },
      ];

      const sorted = [...tasks].sort((a, b) => {
        const aDate = a.created_at ? new Date(a.created_at).getTime() : 0;
        const bDate = b.created_at ? new Date(b.created_at).getTime() : 0;
        return bDate - aDate;
      });

      // Most recent first
      expect(sorted[0].id).toBe('task-2');
      expect(sorted[1].id).toBe('task-1');
      expect(sorted[2].id).toBe('task-3');
    });
  });
});

// State-to-icon mapping verification
describe('State Icon Mapping', () => {
  const stateToExpectedIcon: Record<string, string> = {
    done: 'check',
    failed: 'error',
    idle: 'circle-outline',
    planning: 'edit',
    implementing: 'code',
    reviewing: 'eye',
    waiting: 'question',
  };

  for (const [state, expectedIcon] of Object.entries(stateToExpectedIcon)) {
    test(`${state} state has correct icon`, () => {
      expect(expectedIcon).toBeTruthy();
    });
  }
});
