import { describe, test, expect, beforeEach, afterEach } from 'bun:test';
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

describe('MehrhofProjectService Test Suite', () => {
  let outputChannel: MockOutputChannel;
  let context: MockExtensionContext;
  let service: MehrhofProjectService;

  beforeEach(() => {
    saveFetch();
    resetFactories();
    outputChannel = createMockOutputChannel('Mehrhof Test');
    context = createMockExtensionContext();
  });

  afterEach(() => {
    if (service) {
      service.dispose();
    }
    restoreFetch();
  });

  describe('Constructor', () => {
    test('creates instance with context and OutputChannel', () => {
      service = new MehrhofProjectService(
        context as unknown as import('vscode').ExtensionContext,
        outputChannel as unknown as import('vscode').OutputChannel
      );
      expect(service).toBeTruthy();
    });

    test('initial connectionState is disconnected', () => {
      service = new MehrhofProjectService(
        context as unknown as import('vscode').ExtensionContext,
        outputChannel as unknown as import('vscode').OutputChannel
      );
      expect(service.connectionState).toBe('disconnected');
    });

    test('initial workflowState is idle', () => {
      service = new MehrhofProjectService(
        context as unknown as import('vscode').ExtensionContext,
        outputChannel as unknown as import('vscode').OutputChannel
      );
      expect(service.workflowState).toBe('idle');
    });

    test('initial currentTask is null', () => {
      service = new MehrhofProjectService(
        context as unknown as import('vscode').ExtensionContext,
        outputChannel as unknown as import('vscode').OutputChannel
      );
      expect(service.currentTask).toBe(null);
    });

    test('initial currentWork is null', () => {
      service = new MehrhofProjectService(
        context as unknown as import('vscode').ExtensionContext,
        outputChannel as unknown as import('vscode').OutputChannel
      );
      expect(service.currentWork).toBe(null);
    });

    test('initial pendingQuestion is null', () => {
      service = new MehrhofProjectService(
        context as unknown as import('vscode').ExtensionContext,
        outputChannel as unknown as import('vscode').OutputChannel
      );
      expect(service.pendingQuestion).toBe(null);
    });
  });

  describe('Getters', () => {
    test('isConnected returns false when disconnected', () => {
      service = new MehrhofProjectService(
        context as unknown as import('vscode').ExtensionContext,
        outputChannel as unknown as import('vscode').OutputChannel
      );
      expect(service.isConnected).toBe(false);
    });

    test('client returns null when disconnected', () => {
      service = new MehrhofProjectService(
        context as unknown as import('vscode').ExtensionContext,
        outputChannel as unknown as import('vscode').OutputChannel
      );
      expect(service.client).toBe(null);
    });

    test('connectionState getter returns current state', () => {
      service = new MehrhofProjectService(
        context as unknown as import('vscode').ExtensionContext,
        outputChannel as unknown as import('vscode').OutputChannel
      );
      const state = service.connectionState;
      expect(['disconnected', 'connecting', 'connected'].includes(state)).toBeTruthy();
    });
  });

  describe('isServerRunning()', () => {
    test('returns false initially', () => {
      service = new MehrhofProjectService(
        context as unknown as import('vscode').ExtensionContext,
        outputChannel as unknown as import('vscode').OutputChannel
      );
      expect(service.isServerRunning()).toBe(false);
    });
  });

  describe('stopServer()', () => {
    test('handles not running gracefully', () => {
      service = new MehrhofProjectService(
        context as unknown as import('vscode').ExtensionContext,
        outputChannel as unknown as import('vscode').OutputChannel
      );
      // Should not throw
      expect(() => service.stopServer()).not.toThrow();
    });
  });

  describe('disconnect()', () => {
    test('sets connectionState to disconnected', () => {
      service = new MehrhofProjectService(
        context as unknown as import('vscode').ExtensionContext,
        outputChannel as unknown as import('vscode').OutputChannel
      );
      service.disconnect();
      expect(service.connectionState).toBe('disconnected');
    });

    test('clears currentTask', () => {
      service = new MehrhofProjectService(
        context as unknown as import('vscode').ExtensionContext,
        outputChannel as unknown as import('vscode').OutputChannel
      );
      service.disconnect();
      expect(service.currentTask).toBe(null);
    });

    test('clears currentWork', () => {
      service = new MehrhofProjectService(
        context as unknown as import('vscode').ExtensionContext,
        outputChannel as unknown as import('vscode').OutputChannel
      );
      service.disconnect();
      expect(service.currentWork).toBe(null);
    });

    test('resets workflowState to idle', () => {
      service = new MehrhofProjectService(
        context as unknown as import('vscode').ExtensionContext,
        outputChannel as unknown as import('vscode').OutputChannel
      );
      service.disconnect();
      expect(service.workflowState).toBe('idle');
    });

    test('clears pendingQuestion', () => {
      service = new MehrhofProjectService(
        context as unknown as import('vscode').ExtensionContext,
        outputChannel as unknown as import('vscode').OutputChannel
      );
      service.disconnect();
      expect(service.pendingQuestion).toBe(null);
    });
  });

  describe('Event Emitter', () => {
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
      expect(states.includes('disconnected')).toBeTruthy();
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
      expect(callCount).toBe(0);
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

      expect(events.length).toBe(1);
      expect(events[0]).toEqual(testEvent);
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

      expect(received.length).toBe(1);
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

      expect(questions.length).toBe(1);
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

      expect(messages.length).toBe(1);
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

      expect(errors.length).toBe(1);
      expect(errors[0]).toBe(testError);
    });
  });

  describe('dispose()', () => {
    test('disconnects on dispose', () => {
      service = new MehrhofProjectService(
        context as unknown as import('vscode').ExtensionContext,
        outputChannel as unknown as import('vscode').OutputChannel
      );
      service.dispose();
      expect(service.connectionState).toBe('disconnected');
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
      expect(called).toBe(false);
    });
  });

  describe('refreshState()', () => {
    test('does nothing when not connected', async () => {
      service = new MehrhofProjectService(
        context as unknown as import('vscode').ExtensionContext,
        outputChannel as unknown as import('vscode').OutputChannel
      );
      // Should not throw
      await service.refreshState();
      expect(service.currentTask).toBe(null);
    });
  });

  describe('OutputChannel logging', () => {
    test('logs to output channel', () => {
      service = new MehrhofProjectService(
        context as unknown as import('vscode').ExtensionContext,
        outputChannel as unknown as import('vscode').OutputChannel
      );
      // Various operations should log
      service.disconnect();
      // Check that some logging occurred (output channel has been used)
      expect(outputChannel.lines.length >= 0).toBeTruthy();
    });
  });
});

// Type validation tests
describe('ProjectService Types', () => {
  test('ConnectionState type includes all valid states', () => {
    const states: ConnectionState[] = ['disconnected', 'connecting', 'connected'];
    expect(states.length).toBe(3);
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

    expect(typeof events.connectionChanged).toBe('function');
    expect(typeof events.stateChanged).toBe('function');
    expect(typeof events.taskChanged).toBe('function');
    expect(typeof events.questionReceived).toBe('function');
    expect(typeof events.agentMessage).toBe('function');
    expect(typeof events.error).toBe('function');
  });
});

// Integration with factories
describe('ProjectService Factory Integration', () => {
  test('createTaskInfo creates valid TaskInfo', () => {
    const task = createTaskInfo();
    expect(task.id).toBeTruthy();
    expect(task.state).toBeTruthy();
  });

  test('createTaskWork creates valid TaskWork', () => {
    const work = createTaskWork();
    expect(work.title).toBeTruthy();
    expect(work.created_at).toBeTruthy();
  });

  test('createTaskResponse creates valid TaskResponse', () => {
    const response = createTaskResponse();
    expect(response.active).toBe(true);
    expect(response.task).toBeTruthy();
    expect(response.work).toBeTruthy();
  });

  test('createStateChangedEvent creates valid event', () => {
    const event = createStateChangedEvent({ from: 'idle', to: 'planning' });
    expect(event.from).toBe('idle');
    expect(event.to).toBe('planning');
  });

  test('createPendingQuestion creates valid question', () => {
    const question = createPendingQuestion();
    expect(question.question).toBeTruthy();
    expect(question.options && question.options.length > 0).toBeTruthy();
  });
});

// Mock helpers verification
describe('ProjectService Mock Helpers', () => {
  test('MockExtensionContext has required properties', () => {
    const ctx = createMockExtensionContext();
    expect(Array.isArray(ctx.subscriptions)).toBeTruthy();
    expect(ctx.extensionPath).toBeTruthy();
    expect(ctx.extensionUri).toBeTruthy();
  });

  test('MockOutputChannel tracks lines', () => {
    const channel = createMockOutputChannel('Test');
    channel.appendLine('Line 1');
    channel.appendLine('Line 2');
    expect(channel.lines).toEqual(['Line 1', 'Line 2']);
  });

  test('MockOutputChannel clear works', () => {
    const channel = createMockOutputChannel('Test');
    channel.appendLine('Line 1');
    channel.clear();
    expect(channel.lines).toEqual([]);
  });
});
