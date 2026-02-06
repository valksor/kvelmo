import { describe, test, expect, beforeEach } from 'bun:test';
import * as vscode from 'vscode';
import { StatusBarWidget } from '../../src/statusbar/statusWidget';
import type { MehrhofProjectService } from '../../src/services/projectService';
import { resetMocks } from '../helpers/mockVscode';

// Helper type for our mock service
type EventHandler = (...args: unknown[]) => void;

interface MockService {
  connectionState: string;
  workflowState: string;
  currentTask: { id: string; branch?: string } | null;
  currentWork: { title?: string } | null;
  isConnected: boolean;
  _listeners: Map<string, EventHandler[]>;
  on(event: string, handler: EventHandler): MockService;
  _emit(event: string): void;
}

function createMockService(overrides: Partial<MockService> = {}): MockService {
  const listeners = new Map<string, EventHandler[]>();
  const service: MockService = {
    connectionState: 'disconnected',
    workflowState: 'idle',
    currentTask: null,
    currentWork: null,
    isConnected: false,
    _listeners: listeners,
    on(event: string, handler: EventHandler) {
      if (!listeners.has(event)) {
        listeners.set(event, []);
      }
      listeners.get(event)!.push(handler);
      return this;
    },
    _emit(event: string) {
      for (const h of listeners.get(event) ?? []) {
        h();
      }
    },
    ...overrides,
  };
  return service;
}

describe('StatusBarWidget (Source Import)', () => {
  let service: MockService;
  let widget: StatusBarWidget;
  let statusBarItem: {
    text: string;
    tooltip: string | undefined;
    backgroundColor: { id: string } | undefined;
    command: string | undefined;
  };

  beforeEach(() => {
    resetMocks();
    // Clear accumulated status bar items
    const win = vscode.window as unknown as { _statusBarItems: unknown[] };
    win._statusBarItems.length = 0;

    service = createMockService();
    widget = new StatusBarWidget(service as unknown as MehrhofProjectService);

    // Grab the status bar item that was just created
    statusBarItem = win._statusBarItems[win._statusBarItems.length - 1] as typeof statusBarItem;
  });

  // ────────────────────────────────────────────────────────────────────
  // Constructor
  // ────────────────────────────────────────────────────────────────────

  describe('constructor', () => {
    test('creates a status bar item', () => {
      expect(statusBarItem).toBeTruthy();
    });

    test('sets command to mehrhof.statusBarClicked', () => {
      expect(statusBarItem.command).toBe('mehrhof.statusBarClicked');
    });

    test('registers connectionChanged listener on service', () => {
      expect(service._listeners.has('connectionChanged')).toBe(true);
    });

    test('registers stateChanged listener on service', () => {
      expect(service._listeners.has('stateChanged')).toBe(true);
    });

    test('registers taskChanged listener on service', () => {
      expect(service._listeners.has('taskChanged')).toBe(true);
    });

    test('performs initial update showing disconnected', () => {
      // Default service state is 'disconnected'
      expect(statusBarItem.text).toContain('Disconnected');
    });
  });

  // ────────────────────────────────────────────────────────────────────
  // update() — disconnected state
  // ────────────────────────────────────────────────────────────────────

  describe('update() — disconnected', () => {
    test('shows circle-slash icon', () => {
      expect(statusBarItem.text).toContain('circle-slash');
    });

    test('sets tooltip to click to connect', () => {
      expect(statusBarItem.tooltip).toBe('Click to connect');
    });

    test('clears background color', () => {
      expect(statusBarItem.backgroundColor).toBeUndefined();
    });
  });

  // ────────────────────────────────────────────────────────────────────
  // update() — connecting state
  // ────────────────────────────────────────────────────────────────────

  describe('update() — connecting', () => {
    beforeEach(() => {
      service.connectionState = 'connecting';
      service._emit('connectionChanged');
    });

    test('shows sync~spin icon', () => {
      expect(statusBarItem.text).toContain('sync~spin');
    });

    test('shows Connecting text', () => {
      expect(statusBarItem.text).toContain('Connecting');
    });

    test('sets tooltip to connecting to server', () => {
      expect(statusBarItem.tooltip).toBe('Connecting to server');
    });
  });

  // ────────────────────────────────────────────────────────────────────
  // update() — connected, idle, no task
  // ────────────────────────────────────────────────────────────────────

  describe('update() — connected idle no task', () => {
    beforeEach(() => {
      service.connectionState = 'connected';
      service.workflowState = 'idle';
      service._emit('connectionChanged');
    });

    test('shows circle-outline icon for idle', () => {
      expect(statusBarItem.text).toContain('circle-outline');
    });

    test('shows Idle text', () => {
      expect(statusBarItem.text).toContain('Idle');
    });

    test('tooltip says click to show actions', () => {
      expect(statusBarItem.tooltip).toBe('Click to show actions');
    });

    test('no background color for idle', () => {
      expect(statusBarItem.backgroundColor).toBeUndefined();
    });
  });

  // ────────────────────────────────────────────────────────────────────
  // update() — connected with task and work title
  // ────────────────────────────────────────────────────────────────────

  describe('update() — connected with task', () => {
    beforeEach(() => {
      service.connectionState = 'connected';
      service.workflowState = 'planning';
      service.currentTask = { id: 'task-42', branch: 'feature/test' };
      service.currentWork = { title: 'Fix login bug' };
      service._emit('connectionChanged');
    });

    test('shows edit icon for planning', () => {
      expect(statusBarItem.text).toContain('edit');
    });

    test('shows task title in text', () => {
      expect(statusBarItem.text).toContain('Fix login bug');
    });

    test('tooltip includes state', () => {
      expect(statusBarItem.tooltip).toContain('Planning');
    });

    test('tooltip includes task ID', () => {
      expect(statusBarItem.tooltip).toContain('task-42');
    });

    test('tooltip includes title', () => {
      expect(statusBarItem.tooltip).toContain('Fix login bug');
    });

    test('tooltip includes branch', () => {
      expect(statusBarItem.tooltip).toContain('feature/test');
    });

    test('planning state has warning background', () => {
      expect(statusBarItem.backgroundColor).toBeTruthy();
    });
  });

  // ────────────────────────────────────────────────────────────────────
  // update() — connected with task but no work title
  // ────────────────────────────────────────────────────────────────────

  describe('update() — task without title', () => {
    beforeEach(() => {
      service.connectionState = 'connected';
      service.workflowState = 'implementing';
      service.currentTask = { id: 'task-99' };
      service.currentWork = null;
      service._emit('connectionChanged');
    });

    test('shows task ID in text', () => {
      expect(statusBarItem.text).toContain('task-99');
    });

    test('shows code icon for implementing', () => {
      expect(statusBarItem.text).toContain('code');
    });

    test('implementing state has warning background', () => {
      expect(statusBarItem.backgroundColor).toBeTruthy();
    });
  });

  // ────────────────────────────────────────────────────────────────────
  // getStateIcon() — all states (exercised through update)
  // ────────────────────────────────────────────────────────────────────

  describe('getStateIcon() via update', () => {
    const stateIconCases: { state: string; iconPart: string }[] = [
      { state: 'idle', iconPart: 'circle-outline' },
      { state: 'planning', iconPart: 'edit' },
      { state: 'implementing', iconPart: 'code' },
      { state: 'reviewing', iconPart: 'eye' },
      { state: 'waiting', iconPart: 'question' },
      { state: 'checkpointing', iconPart: 'sync~spin' },
      { state: 'reverting', iconPart: 'sync~spin' },
      { state: 'restoring', iconPart: 'sync~spin' },
      { state: 'done', iconPart: 'check' },
      { state: 'failed', iconPart: 'error' },
      { state: 'unknown_state', iconPart: 'circle-filled' },
    ];

    for (const { state, iconPart } of stateIconCases) {
      test(`${state} → $(${iconPart})`, () => {
        service.connectionState = 'connected';
        service.workflowState = state;
        service._emit('stateChanged');
        expect(statusBarItem.text).toContain(iconPart);
      });
    }
  });

  // ────────────────────────────────────────────────────────────────────
  // Background color for active states
  // ────────────────────────────────────────────────────────────────────

  describe('background color', () => {
    test('reviewing has warning background', () => {
      service.connectionState = 'connected';
      service.workflowState = 'reviewing';
      service._emit('stateChanged');
      expect(statusBarItem.backgroundColor).toBeTruthy();
    });

    test('done has no background', () => {
      service.connectionState = 'connected';
      service.workflowState = 'done';
      service._emit('stateChanged');
      expect(statusBarItem.backgroundColor).toBeUndefined();
    });

    test('waiting has no background', () => {
      service.connectionState = 'connected';
      service.workflowState = 'waiting';
      service._emit('stateChanged');
      expect(statusBarItem.backgroundColor).toBeUndefined();
    });
  });

  // ────────────────────────────────────────────────────────────────────
  // truncate() — long titles
  // ────────────────────────────────────────────────────────────────────

  describe('title truncation', () => {
    test('long title is truncated with ellipsis', () => {
      service.connectionState = 'connected';
      service.workflowState = 'implementing';
      service.currentTask = { id: 'task-1' };
      service.currentWork = { title: 'This is a very long task title that exceeds the limit' };
      service._emit('taskChanged');
      // Title is truncated to 30 chars: first 27 + '...'
      expect(statusBarItem.text.length).toBeLessThan(100);
      expect(statusBarItem.text).toContain('...');
    });

    test('short title is not truncated', () => {
      service.connectionState = 'connected';
      service.workflowState = 'idle';
      service.currentTask = { id: 'task-1' };
      service.currentWork = { title: 'Short' };
      service._emit('taskChanged');
      expect(statusBarItem.text).toContain('Short');
      expect(statusBarItem.text).not.toContain('...');
    });
  });

  // ────────────────────────────────────────────────────────────────────
  // onClicked() — disconnected triggers connect
  // ────────────────────────────────────────────────────────────────────

  describe('onClicked()', () => {
    test('when disconnected, triggers mehrhof.connect', async () => {
      let connectCalled = false;
      (
        vscode.commands as unknown as {
          registerCommand: (cmd: string, cb: () => void) => { dispose: () => void };
        }
      ).registerCommand('mehrhof.connect', () => {
        connectCalled = true;
      });

      // Execute the statusBarClicked command
      await vscode.commands.executeCommand('mehrhof.statusBarClicked');
      expect(connectCalled).toBe(true);
    });

    test('when connected, shows quick pick', async () => {
      service.connectionState = 'connected';
      service.isConnected = true;
      service.workflowState = 'idle';

      let quickPickItems: unknown[] = [];
      (
        vscode.window as unknown as {
          showQuickPick: (items: unknown[], opts?: unknown) => Promise<unknown>;
        }
      ).showQuickPick = (items: unknown[]) => {
        quickPickItems = items;
        return Promise.resolve(undefined); // user cancels
      };

      await vscode.commands.executeCommand('mehrhof.statusBarClicked');

      // Should have presented quick pick items
      expect(quickPickItems.length).toBeGreaterThan(0);
    });

    test('idle state includes Start Task in quick pick', async () => {
      service.connectionState = 'connected';
      service.isConnected = true;
      service.workflowState = 'idle';

      let items: { label: string }[] = [];
      (
        vscode.window as unknown as {
          showQuickPick: (items: unknown[], opts?: unknown) => Promise<unknown>;
        }
      ).showQuickPick = (i: unknown[]) => {
        items = i as { label: string }[];
        return Promise.resolve(undefined);
      };

      await vscode.commands.executeCommand('mehrhof.statusBarClicked');

      const labels = items.map((i) => i.label);
      expect(labels).toContain('$(add) Start Task');
    });

    test('non-idle state includes Finish and Abandon', async () => {
      service.connectionState = 'connected';
      service.isConnected = true;
      service.workflowState = 'implementing';

      let items: { label: string }[] = [];
      (
        vscode.window as unknown as {
          showQuickPick: (items: unknown[], opts?: unknown) => Promise<unknown>;
        }
      ).showQuickPick = (i: unknown[]) => {
        items = i as { label: string }[];
        return Promise.resolve(undefined);
      };

      await vscode.commands.executeCommand('mehrhof.statusBarClicked');

      const labels = items.map((i) => i.label);
      expect(labels).toContain('$(check) Finish');
      expect(labels).toContain('$(discard) Abandon');
    });

    test('selecting an action executes the command', async () => {
      service.connectionState = 'connected';
      service.isConnected = true;
      service.workflowState = 'idle';

      let executedCommand = '';
      (
        vscode.window as unknown as {
          showQuickPick: (items: unknown[], opts?: unknown) => Promise<unknown>;
        }
      ).showQuickPick = () => {
        return Promise.resolve({ label: '$(info) Status', description: 'Show task status' });
      };

      (
        vscode.commands as unknown as {
          registerCommand: (cmd: string, cb: () => void) => { dispose: () => void };
        }
      ).registerCommand('mehrhof.status', () => {
        executedCommand = 'mehrhof.status';
      });

      await vscode.commands.executeCommand('mehrhof.statusBarClicked');
      // Allow the then() promise to resolve
      await new Promise((resolve) => setTimeout(resolve, 10));
      expect(executedCommand).toBe('mehrhof.status');
    });
  });

  // ────────────────────────────────────────────────────────────────────
  // dispose()
  // ────────────────────────────────────────────────────────────────────

  describe('dispose', () => {
    test('does not throw', () => {
      expect(() => widget.dispose()).not.toThrow();
    });
  });
});
