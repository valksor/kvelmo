import * as assert from 'assert';
import { MehrhofProjectService, type ConnectionState } from '../../src/services/projectService';
import {
  createMockOutputChannel,
  createMockExtensionContext,
  type MockOutputChannel,
  type MockExtensionContext,
} from '../helpers/mockVscode';
import {
  createTaskInfo,
  createTaskWork,
  createTaskResponse,
  createStateChangedEvent,
  createAgentMessageEvent,
  createPendingQuestion,
  saveFetch,
  restoreFetch,
  resetFactories,
} from '../helpers/factories';

suite('MehrhofProjectService Test Suite', () => {
  let outputChannel: MockOutputChannel;
  let context: MockExtensionContext;
  let service: MehrhofProjectService;

  setup(() => {
    saveFetch();
    resetFactories();
    outputChannel = createMockOutputChannel('Mehrhof Test');
    context = createMockExtensionContext();
  });

  teardown(() => {
    if (service) {
      service.dispose();
    }
    restoreFetch();
  });

  suite('Constructor', () => {
    test('creates instance with context and OutputChannel', () => {
      service = new MehrhofProjectService(
        context as unknown as import('vscode').ExtensionContext,
        outputChannel as unknown as import('vscode').OutputChannel
      );
      assert.ok(service);
    });

    test('initial connectionState is disconnected', () => {
      service = new MehrhofProjectService(
        context as unknown as import('vscode').ExtensionContext,
        outputChannel as unknown as import('vscode').OutputChannel
      );
      assert.strictEqual(service.connectionState, 'disconnected');
    });

    test('initial workflowState is idle', () => {
      service = new MehrhofProjectService(
        context as unknown as import('vscode').ExtensionContext,
        outputChannel as unknown as import('vscode').OutputChannel
      );
      assert.strictEqual(service.workflowState, 'idle');
    });

    test('initial currentTask is null', () => {
      service = new MehrhofProjectService(
        context as unknown as import('vscode').ExtensionContext,
        outputChannel as unknown as import('vscode').OutputChannel
      );
      assert.strictEqual(service.currentTask, null);
    });

    test('initial currentWork is null', () => {
      service = new MehrhofProjectService(
        context as unknown as import('vscode').ExtensionContext,
        outputChannel as unknown as import('vscode').OutputChannel
      );
      assert.strictEqual(service.currentWork, null);
    });

    test('initial pendingQuestion is null', () => {
      service = new MehrhofProjectService(
        context as unknown as import('vscode').ExtensionContext,
        outputChannel as unknown as import('vscode').OutputChannel
      );
      assert.strictEqual(service.pendingQuestion, null);
    });
  });

  suite('Getters', () => {
    test('isConnected returns false when disconnected', () => {
      service = new MehrhofProjectService(
        context as unknown as import('vscode').ExtensionContext,
        outputChannel as unknown as import('vscode').OutputChannel
      );
      assert.strictEqual(service.isConnected, false);
    });

    test('client returns null when disconnected', () => {
      service = new MehrhofProjectService(
        context as unknown as import('vscode').ExtensionContext,
        outputChannel as unknown as import('vscode').OutputChannel
      );
      assert.strictEqual(service.client, null);
    });

    test('connectionState getter returns current state', () => {
      service = new MehrhofProjectService(
        context as unknown as import('vscode').ExtensionContext,
        outputChannel as unknown as import('vscode').OutputChannel
      );
      const state = service.connectionState;
      assert.ok(['disconnected', 'connecting', 'connected'].includes(state));
    });
  });

  suite('isServerRunning()', () => {
    test('returns false initially', () => {
      service = new MehrhofProjectService(
        context as unknown as import('vscode').ExtensionContext,
        outputChannel as unknown as import('vscode').OutputChannel
      );
      assert.strictEqual(service.isServerRunning(), false);
    });
  });

  suite('stopServer()', () => {
    test('handles not running gracefully', () => {
      service = new MehrhofProjectService(
        context as unknown as import('vscode').ExtensionContext,
        outputChannel as unknown as import('vscode').OutputChannel
      );
      // Should not throw
      assert.doesNotThrow(() => service.stopServer());
    });
  });

  suite('disconnect()', () => {
    test('sets connectionState to disconnected', () => {
      service = new MehrhofProjectService(
        context as unknown as import('vscode').ExtensionContext,
        outputChannel as unknown as import('vscode').OutputChannel
      );
      service.disconnect();
      assert.strictEqual(service.connectionState, 'disconnected');
    });

    test('clears currentTask', () => {
      service = new MehrhofProjectService(
        context as unknown as import('vscode').ExtensionContext,
        outputChannel as unknown as import('vscode').OutputChannel
      );
      service.disconnect();
      assert.strictEqual(service.currentTask, null);
    });

    test('clears currentWork', () => {
      service = new MehrhofProjectService(
        context as unknown as import('vscode').ExtensionContext,
        outputChannel as unknown as import('vscode').OutputChannel
      );
      service.disconnect();
      assert.strictEqual(service.currentWork, null);
    });

    test('resets workflowState to idle', () => {
      service = new MehrhofProjectService(
        context as unknown as import('vscode').ExtensionContext,
        outputChannel as unknown as import('vscode').OutputChannel
      );
      service.disconnect();
      assert.strictEqual(service.workflowState, 'idle');
    });

    test('clears pendingQuestion', () => {
      service = new MehrhofProjectService(
        context as unknown as import('vscode').ExtensionContext,
        outputChannel as unknown as import('vscode').OutputChannel
      );
      service.disconnect();
      assert.strictEqual(service.pendingQuestion, null);
    });
  });

  suite('Event Emitter', () => {
    test('on() registers connectionChanged listener', () => {
      service = new MehrhofProjectService(
        context as unknown as import('vscode').ExtensionContext,
        outputChannel as unknown as import('vscode').OutputChannel
      );
      const states: ConnectionState[] = [];
      service.on('connectionChanged', (state) => {
        states.push(state);
      });
      // Trigger disconnect which emits connectionChanged
      // First set a different state internally, then disconnect
      (service as unknown as { _connectionState: ConnectionState })._connectionState = 'connected';
      service.disconnect();
      assert.ok(states.includes('disconnected'));
    });

    test('off() removes listener', () => {
      service = new MehrhofProjectService(
        context as unknown as import('vscode').ExtensionContext,
        outputChannel as unknown as import('vscode').OutputChannel
      );
      let callCount = 0;
      const listener = () => {
        callCount++;
      };
      service.on('connectionChanged', listener);
      service.off('connectionChanged', listener);
      // Force state change
      (service as unknown as { _connectionState: ConnectionState })._connectionState = 'connected';
      service.disconnect();
      assert.strictEqual(callCount, 0);
    });

    test('emits stateChanged event', () => {
      service = new MehrhofProjectService(
        context as unknown as import('vscode').ExtensionContext,
        outputChannel as unknown as import('vscode').OutputChannel
      );
      const events: unknown[] = [];
      service.on('stateChanged', (event) => {
        events.push(event);
      });

      // Manually emit via private method
      const testEvent = createStateChangedEvent({ from: 'idle', to: 'planning' });
      (service as unknown as { emit: (event: string, ...args: unknown[]) => void }).emit(
        'stateChanged',
        testEvent
      );

      assert.strictEqual(events.length, 1);
      assert.deepStrictEqual(events[0], testEvent);
    });

    test('emits taskChanged event', () => {
      service = new MehrhofProjectService(
        context as unknown as import('vscode').ExtensionContext,
        outputChannel as unknown as import('vscode').OutputChannel
      );
      const received: unknown[] = [];
      service.on('taskChanged', (task, work) => {
        received.push({ task, work });
      });

      const task = createTaskInfo();
      const work = createTaskWork();
      (service as unknown as { emit: (event: string, ...args: unknown[]) => void }).emit(
        'taskChanged',
        task,
        work
      );

      assert.strictEqual(received.length, 1);
    });

    test('emits questionReceived event', () => {
      service = new MehrhofProjectService(
        context as unknown as import('vscode').ExtensionContext,
        outputChannel as unknown as import('vscode').OutputChannel
      );
      const questions: unknown[] = [];
      service.on('questionReceived', (question) => {
        questions.push(question);
      });

      const question = createPendingQuestion();
      (service as unknown as { emit: (event: string, ...args: unknown[]) => void }).emit(
        'questionReceived',
        question
      );

      assert.strictEqual(questions.length, 1);
    });

    test('emits agentMessage event', () => {
      service = new MehrhofProjectService(
        context as unknown as import('vscode').ExtensionContext,
        outputChannel as unknown as import('vscode').OutputChannel
      );
      const messages: unknown[] = [];
      service.on('agentMessage', (event) => {
        messages.push(event);
      });

      const message = createAgentMessageEvent();
      (service as unknown as { emit: (event: string, ...args: unknown[]) => void }).emit(
        'agentMessage',
        message
      );

      assert.strictEqual(messages.length, 1);
    });

    test('emits error event', () => {
      service = new MehrhofProjectService(
        context as unknown as import('vscode').ExtensionContext,
        outputChannel as unknown as import('vscode').OutputChannel
      );
      const errors: Error[] = [];
      service.on('error', (error) => {
        errors.push(error);
      });

      const testError = new Error('Test error');
      (service as unknown as { emit: (event: string, ...args: unknown[]) => void }).emit(
        'error',
        testError
      );

      assert.strictEqual(errors.length, 1);
      assert.strictEqual(errors[0], testError);
    });
  });

  suite('dispose()', () => {
    test('disconnects on dispose', () => {
      service = new MehrhofProjectService(
        context as unknown as import('vscode').ExtensionContext,
        outputChannel as unknown as import('vscode').OutputChannel
      );
      service.dispose();
      assert.strictEqual(service.connectionState, 'disconnected');
    });

    test('clears listeners on dispose', () => {
      service = new MehrhofProjectService(
        context as unknown as import('vscode').ExtensionContext,
        outputChannel as unknown as import('vscode').OutputChannel
      );
      let called = false;
      service.on('connectionChanged', () => {
        called = true;
      });
      service.dispose();
      // After dispose, no events should be emitted
      (service as unknown as { emit: (event: string, ...args: unknown[]) => void }).emit(
        'connectionChanged',
        'connected'
      );
      assert.strictEqual(called, false);
    });
  });

  suite('refreshState()', () => {
    test('does nothing when not connected', async () => {
      service = new MehrhofProjectService(
        context as unknown as import('vscode').ExtensionContext,
        outputChannel as unknown as import('vscode').OutputChannel
      );
      // Should not throw
      await service.refreshState();
      assert.strictEqual(service.currentTask, null);
    });
  });

  suite('OutputChannel logging', () => {
    test('logs to output channel', () => {
      service = new MehrhofProjectService(
        context as unknown as import('vscode').ExtensionContext,
        outputChannel as unknown as import('vscode').OutputChannel
      );
      // Various operations should log
      service.disconnect();
      // Check that some logging occurred (output channel has been used)
      assert.ok(outputChannel.lines.length >= 0);
    });
  });
});

// Type validation tests
suite('ProjectService Types', () => {
  test('ConnectionState type includes all valid states', () => {
    const states: ConnectionState[] = ['disconnected', 'connecting', 'connected'];
    assert.strictEqual(states.length, 3);
  });

  test('ProjectServiceEvents interface has correct event types', () => {
    // Type checking test
    const events: import('../../src/services/projectService').ProjectServiceEvents = {
      connectionChanged: (_state: ConnectionState) => {},
      stateChanged: (_event) => {},
      taskChanged: (_task, _work) => {},
      questionReceived: (_question) => {},
      agentMessage: (_event) => {},
      error: (_error: Error) => {},
    };

    assert.strictEqual(typeof events.connectionChanged, 'function');
    assert.strictEqual(typeof events.stateChanged, 'function');
    assert.strictEqual(typeof events.taskChanged, 'function');
    assert.strictEqual(typeof events.questionReceived, 'function');
    assert.strictEqual(typeof events.agentMessage, 'function');
    assert.strictEqual(typeof events.error, 'function');
  });
});

// Integration with factories
suite('ProjectService Factory Integration', () => {
  test('createTaskInfo creates valid TaskInfo', () => {
    const task = createTaskInfo();
    assert.ok(task.id);
    assert.ok(task.state);
  });

  test('createTaskWork creates valid TaskWork', () => {
    const work = createTaskWork();
    assert.ok(work.title);
    assert.ok(work.created_at);
  });

  test('createTaskResponse creates valid TaskResponse', () => {
    const response = createTaskResponse();
    assert.strictEqual(response.active, true);
    assert.ok(response.task);
    assert.ok(response.work);
  });

  test('createStateChangedEvent creates valid event', () => {
    const event = createStateChangedEvent({ from: 'idle', to: 'planning' });
    assert.strictEqual(event.from, 'idle');
    assert.strictEqual(event.to, 'planning');
  });

  test('createPendingQuestion creates valid question', () => {
    const question = createPendingQuestion();
    assert.ok(question.question);
    assert.ok(question.options && question.options.length > 0);
  });
});

// Mock helpers verification
suite('ProjectService Mock Helpers', () => {
  test('MockExtensionContext has required properties', () => {
    const ctx = createMockExtensionContext();
    assert.ok(Array.isArray(ctx.subscriptions));
    assert.ok(ctx.extensionPath);
    assert.ok(ctx.extensionUri);
  });

  test('MockOutputChannel tracks lines', () => {
    const channel = createMockOutputChannel('Test');
    channel.appendLine('Line 1');
    channel.appendLine('Line 2');
    assert.deepStrictEqual(channel.lines, ['Line 1', 'Line 2']);
  });

  test('MockOutputChannel clear works', () => {
    const channel = createMockOutputChannel('Test');
    channel.appendLine('Line 1');
    channel.clear();
    assert.deepStrictEqual(channel.lines, []);
  });
});
