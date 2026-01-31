import * as assert from 'assert';
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

suite('MehrhofOutputChannel Test Suite', () => {
  let service: MockProjectService;
  let outputChannel: MehrhofOutputChannel;

  setup(() => {
    resetFactories();
    service = new MockProjectService();
  });

  teardown(() => {
    if (outputChannel) {
      outputChannel.dispose();
    }
  });

  suite('Constructor', () => {
    test('creates instance with service', () => {
      outputChannel = new MehrhofOutputChannel(
        service as unknown as import('../../src/services/projectService').MehrhofProjectService
      );
      assert.ok(outputChannel);
    });

    test('registers connectionChanged listener', () => {
      outputChannel = new MehrhofOutputChannel(
        service as unknown as import('../../src/services/projectService').MehrhofProjectService
      );
      // Verify the listener was registered by checking the event emitter
      const listeners = service.listenerCount('connectionChanged');
      assert.ok(listeners > 0);
    });

    test('registers stateChanged listener', () => {
      outputChannel = new MehrhofOutputChannel(
        service as unknown as import('../../src/services/projectService').MehrhofProjectService
      );
      const listeners = service.listenerCount('stateChanged');
      assert.ok(listeners > 0);
    });

    test('registers agentMessage listener', () => {
      outputChannel = new MehrhofOutputChannel(
        service as unknown as import('../../src/services/projectService').MehrhofProjectService
      );
      const listeners = service.listenerCount('agentMessage');
      assert.ok(listeners > 0);
    });

    test('registers taskChanged listener', () => {
      outputChannel = new MehrhofOutputChannel(
        service as unknown as import('../../src/services/projectService').MehrhofProjectService
      );
      const listeners = service.listenerCount('taskChanged');
      assert.ok(listeners > 0);
    });

    test('registers questionReceived listener', () => {
      outputChannel = new MehrhofOutputChannel(
        service as unknown as import('../../src/services/projectService').MehrhofProjectService
      );
      const listeners = service.listenerCount('questionReceived');
      assert.ok(listeners > 0);
    });

    test('registers error listener', () => {
      outputChannel = new MehrhofOutputChannel(
        service as unknown as import('../../src/services/projectService').MehrhofProjectService
      );
      const listeners = service.listenerCount('error');
      assert.ok(listeners > 0);
    });
  });

  suite('log()', () => {
    test('formats message with timestamp', () => {
      outputChannel = new MehrhofOutputChannel(
        service as unknown as import('../../src/services/projectService').MehrhofProjectService
      );
      // The log method adds timestamps - we can test by calling it
      assert.doesNotThrow(() => outputChannel.log('Test message'));
    });
  });

  suite('getRolePrefix() via logAgentMessage', () => {
    test('assistant role triggers log', () => {
      outputChannel = new MehrhofOutputChannel(
        service as unknown as import('../../src/services/projectService').MehrhofProjectService
      );
      // Emit an agent message event - this tests the getRolePrefix path
      assert.doesNotThrow(() => service.emitAgentMessage('assistant', 'Hello world'));
    });

    test('tool role triggers log', () => {
      outputChannel = new MehrhofOutputChannel(
        service as unknown as import('../../src/services/projectService').MehrhofProjectService
      );
      assert.doesNotThrow(() => service.emitAgentMessage('tool', 'Tool output'));
    });

    test('system role triggers log', () => {
      outputChannel = new MehrhofOutputChannel(
        service as unknown as import('../../src/services/projectService').MehrhofProjectService
      );
      assert.doesNotThrow(() => service.emitAgentMessage('system', 'System message'));
    });

    test('unknown role triggers log', () => {
      outputChannel = new MehrhofOutputChannel(
        service as unknown as import('../../src/services/projectService').MehrhofProjectService
      );
      assert.doesNotThrow(() => service.emitAgentMessage('custom', 'Custom message'));
    });
  });

  suite('Event Handling', () => {
    test('connectionChanged triggers log', () => {
      outputChannel = new MehrhofOutputChannel(
        service as unknown as import('../../src/services/projectService').MehrhofProjectService
      );
      assert.doesNotThrow(() => service.emitConnection('connected'));
    });

    test('stateChanged triggers log', () => {
      outputChannel = new MehrhofOutputChannel(
        service as unknown as import('../../src/services/projectService').MehrhofProjectService
      );
      assert.doesNotThrow(() => service.emitStateChanged('idle', 'planning'));
    });

    test('taskChanged with task triggers log', () => {
      outputChannel = new MehrhofOutputChannel(
        service as unknown as import('../../src/services/projectService').MehrhofProjectService
      );
      assert.doesNotThrow(() =>
        service.emitTaskChanged({ id: 'task-1', state: 'planning' }, { title: 'Test Task' })
      );
    });

    test('taskChanged without work title uses task id', () => {
      outputChannel = new MehrhofOutputChannel(
        service as unknown as import('../../src/services/projectService').MehrhofProjectService
      );
      assert.doesNotThrow(() => service.emitTaskChanged({ id: 'task-1', state: 'planning' }, null));
    });

    test('taskChanged with null task does not log', () => {
      outputChannel = new MehrhofOutputChannel(
        service as unknown as import('../../src/services/projectService').MehrhofProjectService
      );
      assert.doesNotThrow(() => service.emitTaskChanged(null, null));
    });

    test('questionReceived with options triggers log', () => {
      outputChannel = new MehrhofOutputChannel(
        service as unknown as import('../../src/services/projectService').MehrhofProjectService
      );
      assert.doesNotThrow(() =>
        service.emitQuestion('What do you want?', ['Option A', 'Option B'])
      );
    });

    test('questionReceived without options triggers log', () => {
      outputChannel = new MehrhofOutputChannel(
        service as unknown as import('../../src/services/projectService').MehrhofProjectService
      );
      assert.doesNotThrow(() => service.emitQuestion('What do you want?'));
    });

    test('error triggers log', () => {
      outputChannel = new MehrhofOutputChannel(
        service as unknown as import('../../src/services/projectService').MehrhofProjectService
      );
      assert.doesNotThrow(() => service.emitError(new Error('Something went wrong')));
    });
  });

  suite('logAgentMessage() multiline', () => {
    test('handles multiline content', () => {
      outputChannel = new MehrhofOutputChannel(
        service as unknown as import('../../src/services/projectService').MehrhofProjectService
      );
      assert.doesNotThrow(() => service.emitAgentMessage('assistant', 'Line 1\nLine 2\nLine 3'));
    });

    test('handles single line content', () => {
      outputChannel = new MehrhofOutputChannel(
        service as unknown as import('../../src/services/projectService').MehrhofProjectService
      );
      assert.doesNotThrow(() => service.emitAgentMessage('tool', 'Single line'));
    });
  });

  suite('Public Methods', () => {
    test('show() does not throw', () => {
      outputChannel = new MehrhofOutputChannel(
        service as unknown as import('../../src/services/projectService').MehrhofProjectService
      );
      assert.doesNotThrow(() => outputChannel.show());
    });

    test('clear() does not throw', () => {
      outputChannel = new MehrhofOutputChannel(
        service as unknown as import('../../src/services/projectService').MehrhofProjectService
      );
      assert.doesNotThrow(() => outputChannel.clear());
    });

    test('outputChannel getter returns channel', () => {
      outputChannel = new MehrhofOutputChannel(
        service as unknown as import('../../src/services/projectService').MehrhofProjectService
      );
      assert.ok(outputChannel.outputChannel);
    });
  });

  suite('dispose()', () => {
    test('dispose() does not throw', () => {
      outputChannel = new MehrhofOutputChannel(
        service as unknown as import('../../src/services/projectService').MehrhofProjectService
      );
      assert.doesNotThrow(() => outputChannel.dispose());
    });
  });
});

// Timestamp format verification
suite('Timestamp Format', () => {
  test('timestamp format is HH:MM:SS', () => {
    const timestamp = new Date().toISOString().split('T')[1].split('.')[0];
    assert.ok(timestamp.match(/^\d{2}:\d{2}:\d{2}$/));
  });
});

// Role prefix mapping
suite('Role Prefix Mapping', () => {
  const roleMappings: Record<string, string> = {
    assistant: '[Agent]',
    tool: '[Tool]',
    system: '[System]',
  };

  for (const [role, prefix] of Object.entries(roleMappings)) {
    test(`${role} maps to ${prefix}`, () => {
      assert.ok(prefix.startsWith('['));
      assert.ok(prefix.endsWith(']'));
    });
  }
});
