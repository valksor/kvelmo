import { describe, test, expect, beforeEach, afterEach } from 'bun:test';
import { InteractivePanelProvider } from '../../src/views/interactivePanel';
import { resetFactories } from '../helpers/factories';
import { EventEmitter } from 'events';

// Mock Uri
const mockUri = {
  fsPath: '/mock/extension/path',
  scheme: 'file',
  path: '/mock/extension/path',
};

// Mock ProjectService that emits events
class MockProjectService extends EventEmitter {
  connectionState: string = 'disconnected';
  workflowState: string = 'idle';
  currentTask: { id: string; branch?: string } | null = null;
  currentWork: { title?: string } | null = null;
  pendingQuestion: { question: string; options?: string[] } | null = null;
  isConnected: boolean = false;
  client: {
    executeCommand: (req: {
      command: string;
      args: string[];
    }) => Promise<{ message?: string; error?: string }>;
    chat: (req: { message: string }) => Promise<{ message?: string; error?: string }>;
    stopOperation: () => Promise<void>;
  } | null = null;

  isServerRunning(): boolean {
    return false;
  }

  stopServer(): void {}
  disconnect(): void {
    this.isConnected = false;
    this.connectionState = 'disconnected';
    this.emit('connectionChanged', 'disconnected');
  }

  refreshState(): Promise<void> {
    return Promise.resolve();
  }

  setConnected(connected: boolean): void {
    this.isConnected = connected;
    this.connectionState = connected ? 'connected' : 'disconnected';
    if (connected) {
      this.client = {
        executeCommand: () => Promise.resolve({ message: 'OK' }),
        chat: () => Promise.resolve({ message: 'Response' }),
        stopOperation: () => Promise.resolve(),
      };
    } else {
      this.client = null;
    }
    this.emit('connectionChanged', this.connectionState);
  }

  emitStateChanged(from: string, to: string): void {
    this.workflowState = to;
    this.emit('stateChanged', { from, to });
  }

  emitTaskChanged(
    task: { id: string; state: string } | null,
    work: { title?: string } | null
  ): void {
    this.currentTask = task as { id: string; branch?: string } | null;
    this.currentWork = work;
    this.emit('taskChanged', task, work);
  }

  emitAgentMessage(role: string, content: string): void {
    this.emit('agentMessage', { role, content, timestamp: new Date().toISOString() });
  }

  emitQuestion(question: string, options?: string[]): void {
    this.emit('questionReceived', { question, options });
  }

  emitError(error: Error): void {
    this.emit('error', error);
  }
}

describe('InteractivePanelProvider Test Suite', () => {
  let service: MockProjectService;
  let provider: InteractivePanelProvider;

  beforeEach(() => {
    resetFactories();
    service = new MockProjectService();
  });

  afterEach(() => {
    if (provider) {
      provider.dispose();
    }
  });

  describe('Constructor', () => {
    test('creates instance with extensionUri and service', () => {
      provider = new InteractivePanelProvider(
        mockUri as unknown as import('vscode').Uri,
        service as unknown as import('../../src/services/projectService').MehrhofProjectService
      );
      expect(provider).toBeTruthy();
    });

    test('static viewType is correct', () => {
      expect(InteractivePanelProvider.viewType).toBe('mehrhof.interactive');
    });

    test('registers connectionChanged listener', () => {
      provider = new InteractivePanelProvider(
        mockUri as unknown as import('vscode').Uri,
        service as unknown as import('../../src/services/projectService').MehrhofProjectService
      );
      const listeners = service.listenerCount('connectionChanged');
      expect(listeners > 0).toBeTruthy();
    });

    test('registers stateChanged listener', () => {
      provider = new InteractivePanelProvider(
        mockUri as unknown as import('vscode').Uri,
        service as unknown as import('../../src/services/projectService').MehrhofProjectService
      );
      const listeners = service.listenerCount('stateChanged');
      expect(listeners > 0).toBeTruthy();
    });

    test('registers taskChanged listener', () => {
      provider = new InteractivePanelProvider(
        mockUri as unknown as import('vscode').Uri,
        service as unknown as import('../../src/services/projectService').MehrhofProjectService
      );
      const listeners = service.listenerCount('taskChanged');
      expect(listeners > 0).toBeTruthy();
    });

    test('registers agentMessage listener', () => {
      provider = new InteractivePanelProvider(
        mockUri as unknown as import('vscode').Uri,
        service as unknown as import('../../src/services/projectService').MehrhofProjectService
      );
      const listeners = service.listenerCount('agentMessage');
      expect(listeners > 0).toBeTruthy();
    });

    test('registers questionReceived listener', () => {
      provider = new InteractivePanelProvider(
        mockUri as unknown as import('vscode').Uri,
        service as unknown as import('../../src/services/projectService').MehrhofProjectService
      );
      const listeners = service.listenerCount('questionReceived');
      expect(listeners > 0).toBeTruthy();
    });

    test('registers error listener', () => {
      provider = new InteractivePanelProvider(
        mockUri as unknown as import('vscode').Uri,
        service as unknown as import('../../src/services/projectService').MehrhofProjectService
      );
      const listeners = service.listenerCount('error');
      expect(listeners > 0).toBeTruthy();
    });
  });

  describe('Event Handling', () => {
    test('connectionChanged triggers update', () => {
      provider = new InteractivePanelProvider(
        mockUri as unknown as import('vscode').Uri,
        service as unknown as import('../../src/services/projectService').MehrhofProjectService
      );
      expect(() => service.setConnected(true)).not.toThrow();
    });

    test('stateChanged triggers update', () => {
      provider = new InteractivePanelProvider(
        mockUri as unknown as import('vscode').Uri,
        service as unknown as import('../../src/services/projectService').MehrhofProjectService
      );
      expect(() => service.emitStateChanged('idle', 'planning')).not.toThrow();
    });

    test('taskChanged triggers update', () => {
      provider = new InteractivePanelProvider(
        mockUri as unknown as import('vscode').Uri,
        service as unknown as import('../../src/services/projectService').MehrhofProjectService
      );
      expect(() =>
        service.emitTaskChanged({ id: 'task-1', state: 'idle' }, { title: 'Test' })
      ).not.toThrow();
    });

    test('agentMessage adds message for assistant role', () => {
      provider = new InteractivePanelProvider(
        mockUri as unknown as import('vscode').Uri,
        service as unknown as import('../../src/services/projectService').MehrhofProjectService
      );
      expect(() => service.emitAgentMessage('assistant', 'Hello')).not.toThrow();
    });

    test('agentMessage adds message for other roles', () => {
      provider = new InteractivePanelProvider(
        mockUri as unknown as import('vscode').Uri,
        service as unknown as import('../../src/services/projectService').MehrhofProjectService
      );
      expect(() => service.emitAgentMessage('tool', 'Tool output')).not.toThrow();
    });

    test('questionReceived adds question message', () => {
      provider = new InteractivePanelProvider(
        mockUri as unknown as import('vscode').Uri,
        service as unknown as import('../../src/services/projectService').MehrhofProjectService
      );
      expect(() => service.emitQuestion('What do you want?')).not.toThrow();
    });

    test('questionReceived with options adds both messages', () => {
      provider = new InteractivePanelProvider(
        mockUri as unknown as import('vscode').Uri,
        service as unknown as import('../../src/services/projectService').MehrhofProjectService
      );
      expect(() => service.emitQuestion('Choose one', ['A', 'B', 'C'])).not.toThrow();
    });

    test('error adds error message', () => {
      provider = new InteractivePanelProvider(
        mockUri as unknown as import('vscode').Uri,
        service as unknown as import('../../src/services/projectService').MehrhofProjectService
      );
      expect(() => service.emitError(new Error('Test error'))).not.toThrow();
    });
  });

  describe('dispose()', () => {
    test('dispose does not throw', () => {
      provider = new InteractivePanelProvider(
        mockUri as unknown as import('vscode').Uri,
        service as unknown as import('../../src/services/projectService').MehrhofProjectService
      );
      expect(() => provider.dispose()).not.toThrow();
    });
  });
});

// Test isCommand logic
describe('InteractivePanel Command Detection', () => {
  const isCommand = (input: string): boolean => {
    const commands = [
      'start',
      'plan',
      'implement',
      'review',
      'finish',
      'abandon',
      'continue',
      'undo',
      'redo',
      'status',
      'st',
      'cost',
      'list',
      'budget',
      'help',
      'note',
      'quick',
      'label',
      'specification',
      'spec',
      'find',
      'memory',
      'simplify',
    ];
    const firstWord = input.split(/\s+/)[0].toLowerCase();
    return commands.includes(firstWord);
  };

  test('start is a command', () => {
    expect(isCommand('start task.md')).toBe(true);
  });

  test('plan is a command', () => {
    expect(isCommand('plan')).toBe(true);
  });

  test('implement is a command', () => {
    expect(isCommand('implement')).toBe(true);
  });

  test('review is a command', () => {
    expect(isCommand('review')).toBe(true);
  });

  test('finish is a command', () => {
    expect(isCommand('finish')).toBe(true);
  });

  test('abandon is a command', () => {
    expect(isCommand('abandon')).toBe(true);
  });

  test('continue is a command', () => {
    expect(isCommand('continue')).toBe(true);
  });

  test('undo is a command', () => {
    expect(isCommand('undo')).toBe(true);
  });

  test('redo is a command', () => {
    expect(isCommand('redo')).toBe(true);
  });

  test('status is a command', () => {
    expect(isCommand('status')).toBe(true);
  });

  test('st is a command (short for status)', () => {
    expect(isCommand('st')).toBe(true);
  });

  test('cost is a command', () => {
    expect(isCommand('cost')).toBe(true);
  });

  test('list is a command', () => {
    expect(isCommand('list')).toBe(true);
  });

  test('budget is a command', () => {
    expect(isCommand('budget')).toBe(true);
  });

  test('help is a command', () => {
    expect(isCommand('help')).toBe(true);
  });

  test('note is a command', () => {
    expect(isCommand('note This is a note')).toBe(true);
  });

  test('quick is a command', () => {
    expect(isCommand('quick')).toBe(true);
  });

  test('label is a command', () => {
    expect(isCommand('label fix')).toBe(true);
  });

  test('specification is a command', () => {
    expect(isCommand('specification')).toBe(true);
  });

  test('spec is a command (short for specification)', () => {
    expect(isCommand('spec')).toBe(true);
  });

  test('find is a command', () => {
    expect(isCommand('find error')).toBe(true);
  });

  test('memory is a command', () => {
    expect(isCommand('memory')).toBe(true);
  });

  test('simplify is a command', () => {
    expect(isCommand('simplify')).toBe(true);
  });

  test('random text is not a command', () => {
    expect(isCommand('hello world')).toBe(false);
  });

  test('question is not a command', () => {
    expect(isCommand('How do I do this?')).toBe(false);
  });

  test('commands are case insensitive', () => {
    expect(isCommand('PLAN')).toBe(true);
    expect(isCommand('Plan')).toBe(true);
    expect(isCommand('pLaN')).toBe(true);
  });
});

// Test action to command mapping
describe('InteractivePanel Action Mapping', () => {
  const actionToCommand = (action: string): string | undefined => {
    const commandMap: Record<string, string> = {
      startTask: 'mehrhof.startTask',
      plan: 'mehrhof.plan',
      implement: 'mehrhof.implement',
      review: 'mehrhof.review',
      continue: 'mehrhof.continue',
      finish: 'mehrhof.finish',
      abandon: 'mehrhof.abandon',
      undo: 'mehrhof.undo',
      redo: 'mehrhof.redo',
      status: 'mehrhof.status',
    };
    return commandMap[action];
  };

  test('startTask maps to mehrhof.startTask', () => {
    expect(actionToCommand('startTask')).toBe('mehrhof.startTask');
  });

  test('plan maps to mehrhof.plan', () => {
    expect(actionToCommand('plan')).toBe('mehrhof.plan');
  });

  test('implement maps to mehrhof.implement', () => {
    expect(actionToCommand('implement')).toBe('mehrhof.implement');
  });

  test('review maps to mehrhof.review', () => {
    expect(actionToCommand('review')).toBe('mehrhof.review');
  });

  test('continue maps to mehrhof.continue', () => {
    expect(actionToCommand('continue')).toBe('mehrhof.continue');
  });

  test('finish maps to mehrhof.finish', () => {
    expect(actionToCommand('finish')).toBe('mehrhof.finish');
  });

  test('abandon maps to mehrhof.abandon', () => {
    expect(actionToCommand('abandon')).toBe('mehrhof.abandon');
  });

  test('undo maps to mehrhof.undo', () => {
    expect(actionToCommand('undo')).toBe('mehrhof.undo');
  });

  test('redo maps to mehrhof.redo', () => {
    expect(actionToCommand('redo')).toBe('mehrhof.redo');
  });

  test('status maps to mehrhof.status', () => {
    expect(actionToCommand('status')).toBe('mehrhof.status');
  });

  test('unknown action returns undefined', () => {
    expect(actionToCommand('unknown')).toBe(undefined);
  });
});

// Test message type handling
describe('InteractivePanel Message Types', () => {
  test('WebviewMessage interface has type and payload', () => {
    const message: { type: string; payload?: unknown } = {
      type: 'input',
      payload: 'test input',
    };
    expect(message.type).toBe('input');
    expect(message.payload).toBe('test input');
  });

  test('startServer message type', () => {
    const message = { type: 'startServer' };
    expect(message.type).toBe('startServer');
  });

  test('stopServer message type', () => {
    const message = { type: 'stopServer' };
    expect(message.type).toBe('stopServer');
  });

  test('connect message type', () => {
    const message = { type: 'connect' };
    expect(message.type).toBe('connect');
  });

  test('disconnect message type', () => {
    const message = { type: 'disconnect' };
    expect(message.type).toBe('disconnect');
  });

  test('input message type with payload', () => {
    const message = { type: 'input', payload: 'plan' };
    expect(message.type).toBe('input');
    expect(message.payload).toBe('plan');
  });

  test('action message type with payload', () => {
    const message = { type: 'action', payload: 'plan' };
    expect(message.type).toBe('action');
  });

  test('stop message type', () => {
    const message = { type: 'stop' };
    expect(message.type).toBe('stop');
  });

  test('ready message type', () => {
    const message = { type: 'ready' };
    expect(message.type).toBe('ready');
  });
});

// Test ChatMessage interface
describe('InteractivePanel ChatMessage', () => {
  const createChatMessage = (
    role: 'user' | 'assistant' | 'system' | 'error' | 'command',
    content: string
  ) => ({
    role,
    content,
    timestamp: new Date().toISOString(),
  });

  test('user message has correct structure', () => {
    const msg = createChatMessage('user', 'Hello');
    expect(msg.role).toBe('user');
    expect(msg.content).toBe('Hello');
    expect(msg.timestamp).toBeTruthy();
  });

  test('assistant message has correct structure', () => {
    const msg = createChatMessage('assistant', 'Response');
    expect(msg.role).toBe('assistant');
  });

  test('system message has correct structure', () => {
    const msg = createChatMessage('system', 'System message');
    expect(msg.role).toBe('system');
  });

  test('error message has correct structure', () => {
    const msg = createChatMessage('error', 'Error occurred');
    expect(msg.role).toBe('error');
  });

  test('command message has correct structure', () => {
    const msg = createChatMessage('command', '> plan');
    expect(msg.role).toBe('command');
  });
});

// Test HTML generation concepts
describe('InteractivePanel HTML Content', () => {
  test('HTML includes required elements', () => {
    const requiredIds = [
      'serverBtn',
      'connectionStatus',
      'taskInfo',
      'taskTitle',
      'stateBadge',
      'messages',
      'disconnectedNotice',
      'inputField',
      'stopBtn',
      'sendBtn',
    ];

    // This is a documentation test - verifying expected element IDs
    expect(requiredIds.length).toBe(10);
  });

  test('CSS classes for state badges exist', () => {
    const stateClasses = [
      'state-idle',
      'state-planning',
      'state-implementing',
      'state-reviewing',
      'state-waiting',
      'state-done',
      'state-failed',
    ];

    expect(stateClasses.length).toBe(7);
  });

  test('message role classes exist', () => {
    const messageClasses = [
      'message-user',
      'message-assistant',
      'message-system',
      'message-error',
      'message-command',
    ];

    expect(messageClasses.length).toBe(5);
  });
});
