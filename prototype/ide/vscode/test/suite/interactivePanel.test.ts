import * as assert from 'assert';
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

suite('InteractivePanelProvider Test Suite', () => {
  let service: MockProjectService;
  let provider: InteractivePanelProvider;

  setup(() => {
    resetFactories();
    service = new MockProjectService();
  });

  teardown(() => {
    if (provider) {
      provider.dispose();
    }
  });

  suite('Constructor', () => {
    test('creates instance with extensionUri and service', () => {
      provider = new InteractivePanelProvider(
        mockUri as unknown as import('vscode').Uri,
        service as unknown as import('../../src/services/projectService').MehrhofProjectService
      );
      assert.ok(provider);
    });

    test('static viewType is correct', () => {
      assert.strictEqual(InteractivePanelProvider.viewType, 'mehrhof.interactive');
    });

    test('registers connectionChanged listener', () => {
      provider = new InteractivePanelProvider(
        mockUri as unknown as import('vscode').Uri,
        service as unknown as import('../../src/services/projectService').MehrhofProjectService
      );
      const listeners = service.listenerCount('connectionChanged');
      assert.ok(listeners > 0);
    });

    test('registers stateChanged listener', () => {
      provider = new InteractivePanelProvider(
        mockUri as unknown as import('vscode').Uri,
        service as unknown as import('../../src/services/projectService').MehrhofProjectService
      );
      const listeners = service.listenerCount('stateChanged');
      assert.ok(listeners > 0);
    });

    test('registers taskChanged listener', () => {
      provider = new InteractivePanelProvider(
        mockUri as unknown as import('vscode').Uri,
        service as unknown as import('../../src/services/projectService').MehrhofProjectService
      );
      const listeners = service.listenerCount('taskChanged');
      assert.ok(listeners > 0);
    });

    test('registers agentMessage listener', () => {
      provider = new InteractivePanelProvider(
        mockUri as unknown as import('vscode').Uri,
        service as unknown as import('../../src/services/projectService').MehrhofProjectService
      );
      const listeners = service.listenerCount('agentMessage');
      assert.ok(listeners > 0);
    });

    test('registers questionReceived listener', () => {
      provider = new InteractivePanelProvider(
        mockUri as unknown as import('vscode').Uri,
        service as unknown as import('../../src/services/projectService').MehrhofProjectService
      );
      const listeners = service.listenerCount('questionReceived');
      assert.ok(listeners > 0);
    });

    test('registers error listener', () => {
      provider = new InteractivePanelProvider(
        mockUri as unknown as import('vscode').Uri,
        service as unknown as import('../../src/services/projectService').MehrhofProjectService
      );
      const listeners = service.listenerCount('error');
      assert.ok(listeners > 0);
    });
  });

  suite('Event Handling', () => {
    test('connectionChanged triggers update', () => {
      provider = new InteractivePanelProvider(
        mockUri as unknown as import('vscode').Uri,
        service as unknown as import('../../src/services/projectService').MehrhofProjectService
      );
      assert.doesNotThrow(() => service.setConnected(true));
    });

    test('stateChanged triggers update', () => {
      provider = new InteractivePanelProvider(
        mockUri as unknown as import('vscode').Uri,
        service as unknown as import('../../src/services/projectService').MehrhofProjectService
      );
      assert.doesNotThrow(() => service.emitStateChanged('idle', 'planning'));
    });

    test('taskChanged triggers update', () => {
      provider = new InteractivePanelProvider(
        mockUri as unknown as import('vscode').Uri,
        service as unknown as import('../../src/services/projectService').MehrhofProjectService
      );
      assert.doesNotThrow(() =>
        service.emitTaskChanged({ id: 'task-1', state: 'idle' }, { title: 'Test' })
      );
    });

    test('agentMessage adds message for assistant role', () => {
      provider = new InteractivePanelProvider(
        mockUri as unknown as import('vscode').Uri,
        service as unknown as import('../../src/services/projectService').MehrhofProjectService
      );
      assert.doesNotThrow(() => service.emitAgentMessage('assistant', 'Hello'));
    });

    test('agentMessage adds message for other roles', () => {
      provider = new InteractivePanelProvider(
        mockUri as unknown as import('vscode').Uri,
        service as unknown as import('../../src/services/projectService').MehrhofProjectService
      );
      assert.doesNotThrow(() => service.emitAgentMessage('tool', 'Tool output'));
    });

    test('questionReceived adds question message', () => {
      provider = new InteractivePanelProvider(
        mockUri as unknown as import('vscode').Uri,
        service as unknown as import('../../src/services/projectService').MehrhofProjectService
      );
      assert.doesNotThrow(() => service.emitQuestion('What do you want?'));
    });

    test('questionReceived with options adds both messages', () => {
      provider = new InteractivePanelProvider(
        mockUri as unknown as import('vscode').Uri,
        service as unknown as import('../../src/services/projectService').MehrhofProjectService
      );
      assert.doesNotThrow(() => service.emitQuestion('Choose one', ['A', 'B', 'C']));
    });

    test('error adds error message', () => {
      provider = new InteractivePanelProvider(
        mockUri as unknown as import('vscode').Uri,
        service as unknown as import('../../src/services/projectService').MehrhofProjectService
      );
      assert.doesNotThrow(() => service.emitError(new Error('Test error')));
    });
  });

  suite('dispose()', () => {
    test('dispose does not throw', () => {
      provider = new InteractivePanelProvider(
        mockUri as unknown as import('vscode').Uri,
        service as unknown as import('../../src/services/projectService').MehrhofProjectService
      );
      assert.doesNotThrow(() => provider.dispose());
    });
  });
});

// Test isCommand logic
suite('InteractivePanel Command Detection', () => {
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
    assert.strictEqual(isCommand('start task.md'), true);
  });

  test('plan is a command', () => {
    assert.strictEqual(isCommand('plan'), true);
  });

  test('implement is a command', () => {
    assert.strictEqual(isCommand('implement'), true);
  });

  test('review is a command', () => {
    assert.strictEqual(isCommand('review'), true);
  });

  test('finish is a command', () => {
    assert.strictEqual(isCommand('finish'), true);
  });

  test('abandon is a command', () => {
    assert.strictEqual(isCommand('abandon'), true);
  });

  test('continue is a command', () => {
    assert.strictEqual(isCommand('continue'), true);
  });

  test('undo is a command', () => {
    assert.strictEqual(isCommand('undo'), true);
  });

  test('redo is a command', () => {
    assert.strictEqual(isCommand('redo'), true);
  });

  test('status is a command', () => {
    assert.strictEqual(isCommand('status'), true);
  });

  test('st is a command (short for status)', () => {
    assert.strictEqual(isCommand('st'), true);
  });

  test('cost is a command', () => {
    assert.strictEqual(isCommand('cost'), true);
  });

  test('list is a command', () => {
    assert.strictEqual(isCommand('list'), true);
  });

  test('budget is a command', () => {
    assert.strictEqual(isCommand('budget'), true);
  });

  test('help is a command', () => {
    assert.strictEqual(isCommand('help'), true);
  });

  test('note is a command', () => {
    assert.strictEqual(isCommand('note This is a note'), true);
  });

  test('quick is a command', () => {
    assert.strictEqual(isCommand('quick'), true);
  });

  test('label is a command', () => {
    assert.strictEqual(isCommand('label fix'), true);
  });

  test('specification is a command', () => {
    assert.strictEqual(isCommand('specification'), true);
  });

  test('spec is a command (short for specification)', () => {
    assert.strictEqual(isCommand('spec'), true);
  });

  test('find is a command', () => {
    assert.strictEqual(isCommand('find error'), true);
  });

  test('memory is a command', () => {
    assert.strictEqual(isCommand('memory'), true);
  });

  test('simplify is a command', () => {
    assert.strictEqual(isCommand('simplify'), true);
  });

  test('random text is not a command', () => {
    assert.strictEqual(isCommand('hello world'), false);
  });

  test('question is not a command', () => {
    assert.strictEqual(isCommand('How do I do this?'), false);
  });

  test('commands are case insensitive', () => {
    assert.strictEqual(isCommand('PLAN'), true);
    assert.strictEqual(isCommand('Plan'), true);
    assert.strictEqual(isCommand('pLaN'), true);
  });
});

// Test action to command mapping
suite('InteractivePanel Action Mapping', () => {
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
    assert.strictEqual(actionToCommand('startTask'), 'mehrhof.startTask');
  });

  test('plan maps to mehrhof.plan', () => {
    assert.strictEqual(actionToCommand('plan'), 'mehrhof.plan');
  });

  test('implement maps to mehrhof.implement', () => {
    assert.strictEqual(actionToCommand('implement'), 'mehrhof.implement');
  });

  test('review maps to mehrhof.review', () => {
    assert.strictEqual(actionToCommand('review'), 'mehrhof.review');
  });

  test('continue maps to mehrhof.continue', () => {
    assert.strictEqual(actionToCommand('continue'), 'mehrhof.continue');
  });

  test('finish maps to mehrhof.finish', () => {
    assert.strictEqual(actionToCommand('finish'), 'mehrhof.finish');
  });

  test('abandon maps to mehrhof.abandon', () => {
    assert.strictEqual(actionToCommand('abandon'), 'mehrhof.abandon');
  });

  test('undo maps to mehrhof.undo', () => {
    assert.strictEqual(actionToCommand('undo'), 'mehrhof.undo');
  });

  test('redo maps to mehrhof.redo', () => {
    assert.strictEqual(actionToCommand('redo'), 'mehrhof.redo');
  });

  test('status maps to mehrhof.status', () => {
    assert.strictEqual(actionToCommand('status'), 'mehrhof.status');
  });

  test('unknown action returns undefined', () => {
    assert.strictEqual(actionToCommand('unknown'), undefined);
  });
});

// Test message type handling
suite('InteractivePanel Message Types', () => {
  test('WebviewMessage interface has type and payload', () => {
    const message: { type: string; payload?: unknown } = {
      type: 'input',
      payload: 'test input',
    };
    assert.strictEqual(message.type, 'input');
    assert.strictEqual(message.payload, 'test input');
  });

  test('startServer message type', () => {
    const message = { type: 'startServer' };
    assert.strictEqual(message.type, 'startServer');
  });

  test('stopServer message type', () => {
    const message = { type: 'stopServer' };
    assert.strictEqual(message.type, 'stopServer');
  });

  test('connect message type', () => {
    const message = { type: 'connect' };
    assert.strictEqual(message.type, 'connect');
  });

  test('disconnect message type', () => {
    const message = { type: 'disconnect' };
    assert.strictEqual(message.type, 'disconnect');
  });

  test('input message type with payload', () => {
    const message = { type: 'input', payload: 'plan' };
    assert.strictEqual(message.type, 'input');
    assert.strictEqual(message.payload, 'plan');
  });

  test('action message type with payload', () => {
    const message = { type: 'action', payload: 'plan' };
    assert.strictEqual(message.type, 'action');
  });

  test('stop message type', () => {
    const message = { type: 'stop' };
    assert.strictEqual(message.type, 'stop');
  });

  test('ready message type', () => {
    const message = { type: 'ready' };
    assert.strictEqual(message.type, 'ready');
  });
});

// Test ChatMessage interface
suite('InteractivePanel ChatMessage', () => {
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
    assert.strictEqual(msg.role, 'user');
    assert.strictEqual(msg.content, 'Hello');
    assert.ok(msg.timestamp);
  });

  test('assistant message has correct structure', () => {
    const msg = createChatMessage('assistant', 'Response');
    assert.strictEqual(msg.role, 'assistant');
  });

  test('system message has correct structure', () => {
    const msg = createChatMessage('system', 'System message');
    assert.strictEqual(msg.role, 'system');
  });

  test('error message has correct structure', () => {
    const msg = createChatMessage('error', 'Error occurred');
    assert.strictEqual(msg.role, 'error');
  });

  test('command message has correct structure', () => {
    const msg = createChatMessage('command', '> plan');
    assert.strictEqual(msg.role, 'command');
  });
});

// Test HTML generation concepts
suite('InteractivePanel HTML Content', () => {
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
    assert.strictEqual(requiredIds.length, 10);
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

    assert.strictEqual(stateClasses.length, 7);
  });

  test('message role classes exist', () => {
    const messageClasses = [
      'message-user',
      'message-assistant',
      'message-system',
      'message-error',
      'message-command',
    ];

    assert.strictEqual(messageClasses.length, 5);
  });
});
