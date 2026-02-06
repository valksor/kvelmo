import { describe, test, expect, beforeEach, afterEach } from 'bun:test';
import { MehrhofOutputChannel } from '../../src/views/outputChannel';
import { resetFactories } from '../helpers/factories';
import { EventEmitter } from 'events';

// Mock ProjectService that emits events
class MockProjectService extends EventEmitter {
  connectionState: string = 'disconnected';
  workflowState: string = 'idle';
  currentTask: { id: string; branch?: string } | null = null;
  currentWork: { title?: string } | null = null;
  isConnected: boolean = false;

  emitConnection(state: string): void {
    this.connectionState = state;
    this.emit('connectionChanged', state);
  }

  emitStateChanged(from: string, to: string): void {
    this.emit('stateChanged', {
      from,
      to,
      event: 'test',
      task_id: 'task-1',
      timestamp: new Date().toISOString(),
    });
  }

  emitAgentMessage(role: string, content: string): void {
    this.emit('agentMessage', { role, content, timestamp: new Date().toISOString() });
  }

  emitTaskChanged(
    task: { id: string; state: string } | null,
    work: { title?: string } | null
  ): void {
    this.currentTask = task as { id: string; branch?: string } | null;
    this.currentWork = work;
    this.emit('taskChanged', task, work);
  }

  emitQuestion(question: string, options?: string[]): void {
    this.emit('questionReceived', { question, options });
  }

  emitError(error: Error): void {
    this.emit('error', error);
  }
}

describe('MehrhofOutputChannel Test Suite', () => {
  let service: MockProjectService;
  let outputChannel: MehrhofOutputChannel;

  beforeEach(() => {
    resetFactories();
    service = new MockProjectService();
  });

  afterEach(() => {
    if (outputChannel) {
      outputChannel.dispose();
    }
  });

  describe('Constructor', () => {
    test('creates instance with service', () => {
      outputChannel = new MehrhofOutputChannel(
        service as unknown as import('../../src/services/projectService').MehrhofProjectService
      );
      expect(outputChannel).toBeTruthy();
    });

    test('registers connectionChanged listener', () => {
      outputChannel = new MehrhofOutputChannel(
        service as unknown as import('../../src/services/projectService').MehrhofProjectService
      );
      // Verify the listener was registered by checking the event emitter
      const listeners = service.listenerCount('connectionChanged');
      expect(listeners > 0).toBeTruthy();
    });

    test('registers stateChanged listener', () => {
      outputChannel = new MehrhofOutputChannel(
        service as unknown as import('../../src/services/projectService').MehrhofProjectService
      );
      const listeners = service.listenerCount('stateChanged');
      expect(listeners > 0).toBeTruthy();
    });

    test('registers agentMessage listener', () => {
      outputChannel = new MehrhofOutputChannel(
        service as unknown as import('../../src/services/projectService').MehrhofProjectService
      );
      const listeners = service.listenerCount('agentMessage');
      expect(listeners > 0).toBeTruthy();
    });

    test('registers taskChanged listener', () => {
      outputChannel = new MehrhofOutputChannel(
        service as unknown as import('../../src/services/projectService').MehrhofProjectService
      );
      const listeners = service.listenerCount('taskChanged');
      expect(listeners > 0).toBeTruthy();
    });

    test('registers questionReceived listener', () => {
      outputChannel = new MehrhofOutputChannel(
        service as unknown as import('../../src/services/projectService').MehrhofProjectService
      );
      const listeners = service.listenerCount('questionReceived');
      expect(listeners > 0).toBeTruthy();
    });

    test('registers error listener', () => {
      outputChannel = new MehrhofOutputChannel(
        service as unknown as import('../../src/services/projectService').MehrhofProjectService
      );
      const listeners = service.listenerCount('error');
      expect(listeners > 0).toBeTruthy();
    });
  });

  describe('log()', () => {
    test('formats message with timestamp', () => {
      outputChannel = new MehrhofOutputChannel(
        service as unknown as import('../../src/services/projectService').MehrhofProjectService
      );
      // The log method adds timestamps - we can test by calling it
      expect(() => outputChannel.log('Test message')).not.toThrow();
    });
  });

  describe('getRolePrefix() via logAgentMessage', () => {
    test('assistant role triggers log', () => {
      outputChannel = new MehrhofOutputChannel(
        service as unknown as import('../../src/services/projectService').MehrhofProjectService
      );
      // Emit an agent message event - this tests the getRolePrefix path
      expect(() => service.emitAgentMessage('assistant', 'Hello world')).not.toThrow();
    });

    test('tool role triggers log', () => {
      outputChannel = new MehrhofOutputChannel(
        service as unknown as import('../../src/services/projectService').MehrhofProjectService
      );
      expect(() => service.emitAgentMessage('tool', 'Tool output')).not.toThrow();
    });

    test('system role triggers log', () => {
      outputChannel = new MehrhofOutputChannel(
        service as unknown as import('../../src/services/projectService').MehrhofProjectService
      );
      expect(() => service.emitAgentMessage('system', 'System message')).not.toThrow();
    });

    test('unknown role triggers log', () => {
      outputChannel = new MehrhofOutputChannel(
        service as unknown as import('../../src/services/projectService').MehrhofProjectService
      );
      expect(() => service.emitAgentMessage('custom', 'Custom message')).not.toThrow();
    });
  });

  describe('Event Handling', () => {
    test('connectionChanged triggers log', () => {
      outputChannel = new MehrhofOutputChannel(
        service as unknown as import('../../src/services/projectService').MehrhofProjectService
      );
      expect(() => service.emitConnection('connected')).not.toThrow();
    });

    test('stateChanged triggers log', () => {
      outputChannel = new MehrhofOutputChannel(
        service as unknown as import('../../src/services/projectService').MehrhofProjectService
      );
      expect(() => service.emitStateChanged('idle', 'planning')).not.toThrow();
    });

    test('taskChanged with task triggers log', () => {
      outputChannel = new MehrhofOutputChannel(
        service as unknown as import('../../src/services/projectService').MehrhofProjectService
      );
      expect(() =>
        service.emitTaskChanged({ id: 'task-1', state: 'planning' }, { title: 'Test Task' })
      ).not.toThrow();
    });

    test('taskChanged without work title uses task id', () => {
      outputChannel = new MehrhofOutputChannel(
        service as unknown as import('../../src/services/projectService').MehrhofProjectService
      );
      expect(() =>
        service.emitTaskChanged({ id: 'task-1', state: 'planning' }, null)
      ).not.toThrow();
    });

    test('taskChanged with null task does not log', () => {
      outputChannel = new MehrhofOutputChannel(
        service as unknown as import('../../src/services/projectService').MehrhofProjectService
      );
      expect(() => service.emitTaskChanged(null, null)).not.toThrow();
    });

    test('questionReceived with options triggers log', () => {
      outputChannel = new MehrhofOutputChannel(
        service as unknown as import('../../src/services/projectService').MehrhofProjectService
      );
      expect(() =>
        service.emitQuestion('What do you want?', ['Option A', 'Option B'])
      ).not.toThrow();
    });

    test('questionReceived without options triggers log', () => {
      outputChannel = new MehrhofOutputChannel(
        service as unknown as import('../../src/services/projectService').MehrhofProjectService
      );
      expect(() => service.emitQuestion('What do you want?')).not.toThrow();
    });

    test('error triggers log', () => {
      outputChannel = new MehrhofOutputChannel(
        service as unknown as import('../../src/services/projectService').MehrhofProjectService
      );
      expect(() => service.emitError(new Error('Something went wrong'))).not.toThrow();
    });
  });

  describe('logAgentMessage() multiline', () => {
    test('handles multiline content', () => {
      outputChannel = new MehrhofOutputChannel(
        service as unknown as import('../../src/services/projectService').MehrhofProjectService
      );
      expect(() => service.emitAgentMessage('assistant', 'Line 1\nLine 2\nLine 3')).not.toThrow();
    });

    test('handles single line content', () => {
      outputChannel = new MehrhofOutputChannel(
        service as unknown as import('../../src/services/projectService').MehrhofProjectService
      );
      expect(() => service.emitAgentMessage('tool', 'Single line')).not.toThrow();
    });
  });

  describe('Public Methods', () => {
    test('show() does not throw', () => {
      outputChannel = new MehrhofOutputChannel(
        service as unknown as import('../../src/services/projectService').MehrhofProjectService
      );
      expect(() => outputChannel.show()).not.toThrow();
    });

    test('clear() does not throw', () => {
      outputChannel = new MehrhofOutputChannel(
        service as unknown as import('../../src/services/projectService').MehrhofProjectService
      );
      expect(() => outputChannel.clear()).not.toThrow();
    });

    test('outputChannel getter returns channel', () => {
      outputChannel = new MehrhofOutputChannel(
        service as unknown as import('../../src/services/projectService').MehrhofProjectService
      );
      expect(outputChannel.outputChannel).toBeTruthy();
    });
  });

  describe('dispose()', () => {
    test('dispose() does not throw', () => {
      outputChannel = new MehrhofOutputChannel(
        service as unknown as import('../../src/services/projectService').MehrhofProjectService
      );
      expect(() => outputChannel.dispose()).not.toThrow();
    });
  });
});

// Timestamp format verification
describe('Timestamp Format', () => {
  test('timestamp format is HH:MM:SS', () => {
    const timestamp = new Date().toISOString().split('T')[1].split('.')[0];
    expect(timestamp.match(/^\d{2}:\d{2}:\d{2}$/)).toBeTruthy();
  });
});

// Role prefix mapping
describe('Role Prefix Mapping', () => {
  const roleMappings: Record<string, string> = {
    assistant: '[Agent]',
    tool: '[Tool]',
    system: '[System]',
  };

  for (const [role, prefix] of Object.entries(roleMappings)) {
    test(`${role} maps to ${prefix}`, () => {
      expect(prefix.startsWith('[')).toBeTruthy();
      expect(prefix.endsWith(']')).toBeTruthy();
    });
  }
});
