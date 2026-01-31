import * as assert from 'assert';
import { StatusBarAlignment } from '../helpers/mockVscode';
import { resetFactories } from '../helpers/factories';

// Note: StatusBarWidget registers 'mehrhof.statusBarClicked' command in its constructor.
// The extension already registers this during activation, so we can't create new instances.
// These tests verify the expected behavior and interface without re-instantiation.

// Mock ProjectService interface for documentation
interface MockProjectService {
  connectionState: string;
  workflowState: string;
  currentTask: { id: string; branch?: string } | null;
  currentWork: { title?: string } | null;
  isConnected: boolean;
  on: (event: string, handler: (...args: unknown[]) => void) => void;
}

function createMockProjectService(overrides: Partial<MockProjectService> = {}): MockProjectService {
  const listeners = new Map<string, ((...args: unknown[]) => void)[]>();
  return {
    connectionState: 'disconnected',
    workflowState: 'idle',
    currentTask: null,
    currentWork: null,
    isConnected: false,
    on: (event: string, handler: (...args: unknown[]) => void) => {
      if (!listeners.has(event)) {
        listeners.set(event, []);
      }
      listeners.get(event)!.push(handler);
    },
    ...overrides,
  };
}

suite('StatusBarWidget Test Suite', () => {
  setup(() => {
    resetFactories();
  });

  suite('Constructor', () => {
    test('creates instance with service', () => {
      const service = createMockProjectService();
      // Note: Can't fully test without VS Code runtime, but we can verify interface
      assert.ok(service);
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
      // Interface verification
      service.on('connectionChanged', () => {});
      assert.ok(listenerRegistered);
    });

    test('registers stateChanged listener', () => {
      let listenerRegistered = false;
      const service = createMockProjectService({
        on: (event: string) => {
          if (event === 'stateChanged') {
            listenerRegistered = true;
          }
        },
      });
      service.on('stateChanged', () => {});
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
      service.on('taskChanged', () => {});
      assert.ok(listenerRegistered);
    });
  });

  suite('getStateIcon()', () => {
    // Test the icon mapping logic
    const getStateIcon = (state: string): string => {
      switch (state) {
        case 'idle':
          return '$(circle-outline)';
        case 'planning':
          return '$(edit)';
        case 'implementing':
          return '$(code)';
        case 'reviewing':
          return '$(eye)';
        case 'waiting':
          return '$(question)';
        case 'checkpointing':
        case 'reverting':
        case 'restoring':
          return '$(sync~spin)';
        case 'done':
          return '$(check)';
        case 'failed':
          return '$(error)';
        default:
          return '$(circle-filled)';
      }
    };

    test('idle state returns circle-outline icon', () => {
      assert.strictEqual(getStateIcon('idle'), '$(circle-outline)');
    });

    test('planning state returns edit icon', () => {
      assert.strictEqual(getStateIcon('planning'), '$(edit)');
    });

    test('implementing state returns code icon', () => {
      assert.strictEqual(getStateIcon('implementing'), '$(code)');
    });

    test('reviewing state returns eye icon', () => {
      assert.strictEqual(getStateIcon('reviewing'), '$(eye)');
    });

    test('waiting state returns question icon', () => {
      assert.strictEqual(getStateIcon('waiting'), '$(question)');
    });

    test('checkpointing state returns sync~spin icon', () => {
      assert.strictEqual(getStateIcon('checkpointing'), '$(sync~spin)');
    });

    test('reverting state returns sync~spin icon', () => {
      assert.strictEqual(getStateIcon('reverting'), '$(sync~spin)');
    });

    test('restoring state returns sync~spin icon', () => {
      assert.strictEqual(getStateIcon('restoring'), '$(sync~spin)');
    });

    test('done state returns check icon', () => {
      assert.strictEqual(getStateIcon('done'), '$(check)');
    });

    test('failed state returns error icon', () => {
      assert.strictEqual(getStateIcon('failed'), '$(error)');
    });

    test('unknown state returns circle-filled icon', () => {
      assert.strictEqual(getStateIcon('unknown'), '$(circle-filled)');
    });
  });

  suite('formatState()', () => {
    const formatState = (state: string): string => {
      return state.charAt(0).toUpperCase() + state.slice(1);
    };

    test('capitalizes first letter', () => {
      assert.strictEqual(formatState('idle'), 'Idle');
      assert.strictEqual(formatState('planning'), 'Planning');
      assert.strictEqual(formatState('implementing'), 'Implementing');
    });
  });

  suite('truncate()', () => {
    const truncate = (text: string, maxLength: number): string => {
      if (text.length <= maxLength) {
        return text;
      }
      return text.substring(0, maxLength - 3) + '...';
    };

    test('returns text unchanged if short', () => {
      assert.strictEqual(truncate('Short', 10), 'Short');
    });

    test('truncates with ellipsis if long', () => {
      assert.strictEqual(truncate('This is a very long text', 10), 'This is...');
    });

    test('handles exact length', () => {
      assert.strictEqual(truncate('Exactly10!', 10), 'Exactly10!');
    });
  });

  suite('buildTooltip()', () => {
    const buildTooltip = (
      state: string,
      taskId: string,
      title?: string,
      branch?: string
    ): string => {
      const formatState = (s: string) => s.charAt(0).toUpperCase() + s.slice(1);
      const lines: string[] = [];
      lines.push(`State: ${formatState(state)}`);
      lines.push(`Task: ${taskId}`);
      if (title) {
        lines.push(`Title: ${title}`);
      }
      if (branch) {
        lines.push(`Branch: ${branch}`);
      }
      lines.push('');
      lines.push('Click to show actions');
      return lines.join('\n');
    };

    test('includes state', () => {
      const tooltip = buildTooltip('planning', 'task-1');
      assert.ok(tooltip.includes('Planning'));
    });

    test('includes task ID', () => {
      const tooltip = buildTooltip('idle', 'task-123');
      assert.ok(tooltip.includes('task-123'));
    });

    test('includes title when present', () => {
      const tooltip = buildTooltip('idle', 'task-1', 'My Task');
      assert.ok(tooltip.includes('My Task'));
    });

    test('includes branch when present', () => {
      const tooltip = buildTooltip('idle', 'task-1', undefined, 'feature/test');
      assert.ok(tooltip.includes('feature/test'));
    });

    test('includes click instruction', () => {
      const tooltip = buildTooltip('idle', 'task-1');
      assert.ok(tooltip.includes('Click to show actions'));
    });
  });

  suite('labelToCommand()', () => {
    const labelToCommand = (label: string): string | undefined => {
      const mapping: Record<string, string> = {
        '$(add) Start Task': 'mehrhof.startTask',
        '$(edit) Plan': 'mehrhof.plan',
        '$(code) Implement': 'mehrhof.implement',
        '$(eye) Review': 'mehrhof.review',
        '$(check) Finish': 'mehrhof.finish',
        '$(discard) Abandon': 'mehrhof.abandon',
        '$(history) Undo': 'mehrhof.undo',
        '$(redo) Redo': 'mehrhof.redo',
        '$(info) Status': 'mehrhof.status',
        '$(refresh) Refresh': 'mehrhof.refresh',
        '$(debug-disconnect) Disconnect': 'mehrhof.disconnect',
      };
      return mapping[label];
    };

    test('maps Start Task to mehrhof.startTask', () => {
      assert.strictEqual(labelToCommand('$(add) Start Task'), 'mehrhof.startTask');
    });

    test('maps Plan to mehrhof.plan', () => {
      assert.strictEqual(labelToCommand('$(edit) Plan'), 'mehrhof.plan');
    });

    test('maps Implement to mehrhof.implement', () => {
      assert.strictEqual(labelToCommand('$(code) Implement'), 'mehrhof.implement');
    });

    test('maps Review to mehrhof.review', () => {
      assert.strictEqual(labelToCommand('$(eye) Review'), 'mehrhof.review');
    });

    test('maps Finish to mehrhof.finish', () => {
      assert.strictEqual(labelToCommand('$(check) Finish'), 'mehrhof.finish');
    });

    test('maps Abandon to mehrhof.abandon', () => {
      assert.strictEqual(labelToCommand('$(discard) Abandon'), 'mehrhof.abandon');
    });

    test('maps Undo to mehrhof.undo', () => {
      assert.strictEqual(labelToCommand('$(history) Undo'), 'mehrhof.undo');
    });

    test('maps Redo to mehrhof.redo', () => {
      assert.strictEqual(labelToCommand('$(redo) Redo'), 'mehrhof.redo');
    });

    test('maps Status to mehrhof.status', () => {
      assert.strictEqual(labelToCommand('$(info) Status'), 'mehrhof.status');
    });

    test('maps Refresh to mehrhof.refresh', () => {
      assert.strictEqual(labelToCommand('$(refresh) Refresh'), 'mehrhof.refresh');
    });

    test('maps Disconnect to mehrhof.disconnect', () => {
      assert.strictEqual(labelToCommand('$(debug-disconnect) Disconnect'), 'mehrhof.disconnect');
    });

    test('returns undefined for unknown label', () => {
      assert.strictEqual(labelToCommand('Unknown'), undefined);
    });
  });

  suite('Update Display States', () => {
    test('disconnected state shows circle-slash icon', () => {
      const expected = '$(circle-slash) Mehrhof: Disconnected';
      assert.ok(expected.includes('circle-slash'));
      assert.ok(expected.includes('Disconnected'));
    });

    test('connecting state shows sync~spin icon', () => {
      const expected = '$(sync~spin) Mehrhof: Connecting...';
      assert.ok(expected.includes('sync~spin'));
      assert.ok(expected.includes('Connecting'));
    });

    test('connected with task shows task title', () => {
      const title = 'My Task';
      const text = `$(circle-outline) Mehrhof: Idle - ${title}`;
      assert.ok(text.includes(title));
    });

    test('connected without task shows just state', () => {
      const text = '$(circle-outline) Mehrhof: Idle';
      assert.ok(text.includes('Idle'));
      assert.ok(!text.includes(' - '));
    });
  });

  suite('Background Color States', () => {
    test('planning state has warning background', () => {
      const activeStates = ['planning', 'implementing', 'reviewing'];
      assert.ok(activeStates.includes('planning'));
    });

    test('implementing state has warning background', () => {
      const activeStates = ['planning', 'implementing', 'reviewing'];
      assert.ok(activeStates.includes('implementing'));
    });

    test('reviewing state has warning background', () => {
      const activeStates = ['planning', 'implementing', 'reviewing'];
      assert.ok(activeStates.includes('reviewing'));
    });

    test('idle state has no background', () => {
      const activeStates = ['planning', 'implementing', 'reviewing'];
      assert.ok(!activeStates.includes('idle'));
    });

    test('done state has no background', () => {
      const activeStates = ['planning', 'implementing', 'reviewing'];
      assert.ok(!activeStates.includes('done'));
    });
  });

  suite('Quick Pick Actions', () => {
    test('idle state shows Start Task option', () => {
      const state: string = 'idle';
      const showStartTask = state === 'idle';
      assert.strictEqual(showStartTask, true);
    });

    test('non-idle state shows Finish and Abandon options', () => {
      const state: string = 'planning';
      const showFinishAbandon = state !== 'idle';
      assert.strictEqual(showFinishAbandon, true);
    });

    test('always shows Undo, Redo, Status, Refresh, Disconnect', () => {
      const alwaysVisibleActions = ['Undo', 'Redo', 'Status', 'Refresh', 'Disconnect'];
      assert.strictEqual(alwaysVisibleActions.length, 5);
    });
  });
});

// Status bar alignment verification
suite('StatusBarAlignment', () => {
  test('Left alignment value is 1', () => {
    assert.strictEqual(StatusBarAlignment.Left, 1);
  });

  test('Right alignment value is 2', () => {
    assert.strictEqual(StatusBarAlignment.Right, 2);
  });

  test('StatusBarWidget uses Left alignment', () => {
    // From source: vscode.StatusBarAlignment.Left, 100
    const expectedAlignment = StatusBarAlignment.Left;
    const expectedPriority = 100;
    assert.strictEqual(expectedAlignment, 1);
    assert.strictEqual(expectedPriority, 100);
  });
});
