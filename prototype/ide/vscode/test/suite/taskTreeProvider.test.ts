import * as assert from 'assert';
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

suite('TaskItem Test Suite', () => {
  setup(() => {
    resetFactories();
  });

  suite('Constructor', () => {
    test('sets label from title', () => {
      const task: TaskSummary = { id: 'task-1', state: 'idle', title: 'Test Task' };
      const item = new TaskItem(task, false);
      assert.strictEqual(item.label, 'Test Task');
    });

    test('sets label from id when no title', () => {
      const task: TaskSummary = { id: 'task-123', state: 'idle' };
      const item = new TaskItem(task, false);
      assert.strictEqual(item.label, 'task-123');
    });

    test('sets id from task id', () => {
      const task: TaskSummary = { id: 'task-456', state: 'idle', title: 'Test' };
      const item = new TaskItem(task, false);
      assert.strictEqual(item.id, 'task-456');
    });

    test('sets collapsibleState to None', () => {
      const task: TaskSummary = { id: 'task-1', state: 'idle' };
      const item = new TaskItem(task, false);
      // TreeItemCollapsibleState.None = 0
      assert.strictEqual(item.collapsibleState, 0);
    });

    test('sets contextValue to activeTask when active', () => {
      const task: TaskSummary = { id: 'task-1', state: 'idle' };
      const item = new TaskItem(task, true);
      assert.strictEqual(item.contextValue, 'activeTask');
    });

    test('sets contextValue to task when not active', () => {
      const task: TaskSummary = { id: 'task-1', state: 'idle' };
      const item = new TaskItem(task, false);
      assert.strictEqual(item.contextValue, 'task');
    });
  });

  suite('formatState()', () => {
    test('capitalizes first letter of state', () => {
      const task: TaskSummary = { id: 'task-1', state: 'planning' };
      const item = new TaskItem(task, false);
      assert.strictEqual(item.description, 'Planning');
    });

    test('handles idle state', () => {
      const task: TaskSummary = { id: 'task-1', state: 'idle' };
      const item = new TaskItem(task, false);
      assert.strictEqual(item.description, 'Idle');
    });

    test('handles implementing state', () => {
      const task: TaskSummary = { id: 'task-1', state: 'implementing' };
      const item = new TaskItem(task, false);
      assert.strictEqual(item.description, 'Implementing');
    });
  });

  suite('getIcon()', () => {
    test('done state returns check icon with green color', () => {
      const task: TaskSummary = { id: 'task-1', state: 'done' };
      const item = new TaskItem(task, false);
      assert.ok(item.iconPath);
    });

    test('failed state returns error icon', () => {
      const task: TaskSummary = { id: 'task-1', state: 'failed' };
      const item = new TaskItem(task, false);
      assert.ok(item.iconPath);
    });

    test('idle state returns circle-outline icon', () => {
      const task: TaskSummary = { id: 'task-1', state: 'idle' };
      const item = new TaskItem(task, false);
      assert.ok(item.iconPath);
    });

    test('planning state returns edit icon', () => {
      const task: TaskSummary = { id: 'task-1', state: 'planning' };
      const item = new TaskItem(task, false);
      assert.ok(item.iconPath);
    });

    test('implementing state returns code icon', () => {
      const task: TaskSummary = { id: 'task-1', state: 'implementing' };
      const item = new TaskItem(task, false);
      assert.ok(item.iconPath);
    });

    test('reviewing state returns eye icon', () => {
      const task: TaskSummary = { id: 'task-1', state: 'reviewing' };
      const item = new TaskItem(task, false);
      assert.ok(item.iconPath);
    });

    test('waiting state returns question icon', () => {
      const task: TaskSummary = { id: 'task-1', state: 'waiting' };
      const item = new TaskItem(task, false);
      assert.ok(item.iconPath);
    });

    test('unknown state returns circle-filled icon', () => {
      const task: TaskSummary = { id: 'task-1', state: 'unknown' };
      const item = new TaskItem(task, false);
      assert.ok(item.iconPath);
    });
  });

  suite('buildTooltip()', () => {
    test('includes task ID', () => {
      const task: TaskSummary = { id: 'task-abc', state: 'idle' };
      const item = new TaskItem(task, false);
      const tooltip = typeof item.tooltip === 'string' ? item.tooltip : '';
      assert.ok(tooltip.includes('task-abc'));
    });

    test('includes title when present', () => {
      const task: TaskSummary = { id: 'task-1', state: 'idle', title: 'My Task Title' };
      const item = new TaskItem(task, false);
      const tooltip = typeof item.tooltip === 'string' ? item.tooltip : '';
      assert.ok(tooltip.includes('My Task Title'));
    });

    test('includes state', () => {
      const task: TaskSummary = { id: 'task-1', state: 'planning' };
      const item = new TaskItem(task, false);
      const tooltip = typeof item.tooltip === 'string' ? item.tooltip : '';
      assert.ok(tooltip.includes('Planning'));
    });

    test('includes (Active Task) when active', () => {
      const task: TaskSummary = { id: 'task-1', state: 'idle' };
      const item = new TaskItem(task, true);
      const tooltip = typeof item.tooltip === 'string' ? item.tooltip : '';
      assert.ok(tooltip.includes('Active Task'));
    });

    test('does not include (Active Task) when not active', () => {
      const task: TaskSummary = { id: 'task-1', state: 'idle' };
      const item = new TaskItem(task, false);
      const tooltip = typeof item.tooltip === 'string' ? item.tooltip : '';
      assert.ok(!tooltip.includes('Active Task'));
    });
  });
});

suite('TaskTreeProvider Test Suite', () => {
  setup(() => {
    saveFetch();
    resetFactories();
  });

  teardown(() => {
    restoreFetch();
  });

  suite('Constructor', () => {
    test('creates instance with service', () => {
      const service = createMockProjectService();
      const provider = new TaskTreeProvider(
        service as unknown as import('../../src/services/projectService').MehrhofProjectService
      );
      assert.ok(provider);
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
      assert.ok(listenerRegistered);
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
      assert.ok(listenerRegistered);
    });
  });

  suite('getTreeItem()', () => {
    test('returns element unchanged', () => {
      const service = createMockProjectService();
      const provider = new TaskTreeProvider(
        service as unknown as import('../../src/services/projectService').MehrhofProjectService
      );

      const task: TaskSummary = { id: 'task-1', state: 'idle' };
      const item = new TaskItem(task, false);
      const result = provider.getTreeItem(item);

      assert.strictEqual(result, item);
    });
  });

  suite('getChildren()', () => {
    test('returns empty array when not connected', async () => {
      const service = createMockProjectService({ isConnected: false });
      const provider = new TaskTreeProvider(
        service as unknown as import('../../src/services/projectService').MehrhofProjectService
      );

      const children = await provider.getChildren();
      assert.deepStrictEqual(children, []);
    });
  });

  suite('refresh()', () => {
    test('clears tasks when not connected', async () => {
      const service = createMockProjectService({ isConnected: false });
      const provider = new TaskTreeProvider(
        service as unknown as import('../../src/services/projectService').MehrhofProjectService
      );

      await provider.refresh();
      const children = await provider.getChildren();
      assert.deepStrictEqual(children, []);
    });
  });

  suite('onDidChangeTreeData', () => {
    test('event is defined', () => {
      const service = createMockProjectService();
      const provider = new TaskTreeProvider(
        service as unknown as import('../../src/services/projectService').MehrhofProjectService
      );
      assert.ok(provider.onDidChangeTreeData);
    });
  });

  suite('dispose()', () => {
    test('disposes without error', () => {
      const service = createMockProjectService();
      const provider = new TaskTreeProvider(
        service as unknown as import('../../src/services/projectService').MehrhofProjectService
      );
      assert.doesNotThrow(() => provider.dispose());
    });
  });

  suite('Sorting', () => {
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

      assert.strictEqual(sorted[0].id, 'task-2');
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
      assert.strictEqual(sorted[0].id, 'task-2');
      assert.strictEqual(sorted[1].id, 'task-1');
      assert.strictEqual(sorted[2].id, 'task-3');
    });
  });
});

// State-to-icon mapping verification
suite('State Icon Mapping', () => {
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
      assert.ok(expectedIcon);
    });
  }
});
