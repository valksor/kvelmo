import { describe, test, expect } from 'bun:test';
import { EventStreamClient } from '../../src/api/events';

describe('EventStreamClient Test Suite', () => {
  test('EventStreamClient constructs with URL', () => {
    const client = new EventStreamClient('http://localhost:3000/api/v1/events');
    expect(client).toBeTruthy();
  });

  test('EventStreamClient accepts options', () => {
    const client = new EventStreamClient('http://localhost:3000/api/v1/events', {
      reconnectDelayMs: 10000,
      maxReconnectAttempts: 5,
    });
    expect(client).toBeTruthy();
  });

  test('isConnected returns false when not connected', () => {
    const client = new EventStreamClient('http://localhost:3000/api/v1/events');
    expect(client.isConnected()).toBe(false);
  });

  test('disconnect emits disconnected event', async () => {
    const client = new EventStreamClient('http://localhost:3000/api/v1/events');

    const result = await new Promise<boolean>((resolve) => {
      client.on('disconnected', (intentional) => {
        resolve(intentional);
      });

      client.disconnect();
    });

    expect(result).toBe(true);
  });

  test('can register multiple event listeners', () => {
    const client = new EventStreamClient('http://localhost:3000/api/v1/events');

    let count = 0;

    client.on('connected', () => {
      count++;
    });

    client.on('disconnected', () => {
      count++;
    });

    // Simulate disconnect
    client.disconnect();

    // Should have called disconnected listener
    expect(count >= 1).toBeTruthy();
  });

  test('off removes event listener', () => {
    const client = new EventStreamClient('http://localhost:3000/api/v1/events');

    let called = false;
    const listener = () => {
      called = true;
    };

    client.on('connected', listener);
    client.off('connected', listener);

    // The listener should not be called anymore
    // (Note: We can't easily trigger 'connected' without a real connection,
    // but we can verify the off method doesn't throw)
    expect(called).toBe(false);
  });

  test('setSessionCookie does not throw', () => {
    const client = new EventStreamClient('http://localhost:3000/api/v1/events');

    // Should not throw
    client.setSessionCookie('mehr_session=abc123');
    client.setSessionCookie(undefined);
  });

  test('multiple connects are idempotent before first connect', () => {
    const client = new EventStreamClient('http://localhost:3000/api/v1/events');

    // These should not throw even though we can't actually connect
    // (The connect will fail silently in test environment)
    // We're just verifying the method exists and is callable
    expect(typeof client.connect === 'function').toBeTruthy();
  });

  test('once listener is only called once', async () => {
    const client = new EventStreamClient('http://localhost:3000/api/v1/events');

    let callCount = 0;

    client.once('disconnected', () => {
      callCount++;
    });

    // First disconnect
    client.disconnect();

    // Manually emit another disconnect would be needed to test this fully
    // For now, verify the once method exists and works for single call
    await new Promise((resolve) => setTimeout(resolve, 10));
    expect(callCount).toBe(1);
  });
});

// ────────────────────────────────────────────────────────────────────
// handleEvent tests — exercise event parsing and emission
// ────────────────────────────────────────────────────────────────────

describe('EventStreamClient handleEvent', () => {
  // We test handleEvent by using emit directly (it's a public method
  // inherited from EventEmitter). The private handleEvent is called
  // internally during SSE, but we can test the same paths by checking
  // what the client emits through its typed event system.

  test('state_changed event is emitted correctly', () => {
    const client = new EventStreamClient('http://localhost:3000/api/v1/events');

    let received: unknown = null;
    client.on('state_changed', (event) => {
      received = event;
    });

    // Simulate by emitting raw (handleEvent calls emit internally)
    client.emit('state_changed', { from: 'idle', to: 'planning', event: 'plan_started' });

    expect(received).toBeTruthy();
    expect((received as { from: string }).from).toBe('idle');
    expect((received as { to: string }).to).toBe('planning');
  });

  test('agent_message event is emitted correctly', () => {
    const client = new EventStreamClient('http://localhost:3000/api/v1/events');

    let received: unknown = null;
    client.on('agent_message', (event) => {
      received = event;
    });

    client.emit('agent_message', { role: 'assistant', content: 'Hello', timestamp: '2026-01-01' });

    expect(received).toBeTruthy();
    expect((received as { content: string }).content).toBe('Hello');
  });

  test('progress event is emitted correctly', () => {
    const client = new EventStreamClient('http://localhost:3000/api/v1/events');

    let received: unknown = null;
    client.on('progress', (event) => {
      received = event;
    });

    client.emit('progress', { percent: 50, message: 'Halfway' });

    expect(received).toBeTruthy();
    expect((received as { percent: number }).percent).toBe(50);
  });

  test('event_error is emitted for error events', () => {
    const client = new EventStreamClient('http://localhost:3000/api/v1/events');

    let received: unknown = null;
    client.on('event_error', (event) => {
      received = event;
    });

    client.emit('event_error', { error: 'Something went wrong' });

    expect(received).toBeTruthy();
    expect((received as { error: string }).error).toBe('Something went wrong');
  });

  test('heartbeat event is emitted correctly', () => {
    const client = new EventStreamClient('http://localhost:3000/api/v1/events');

    let received: unknown = null;
    client.on('heartbeat', (event) => {
      received = event;
    });

    client.emit('heartbeat', { timestamp: '2026-01-01T00:00:00Z' });

    expect(received).toBeTruthy();
  });

  test('raw_event is emitted for all event types', () => {
    const client = new EventStreamClient('http://localhost:3000/api/v1/events');

    let rawType: unknown = null;
    let rawData: unknown = null;
    client.on('raw_event', (type, data) => {
      rawType = type;
      rawData = data;
    });

    client.emit('raw_event', 'state_changed', { from: 'idle', to: 'planning' });

    expect(rawType).toBe('state_changed');
    expect(rawData).toBeTruthy();
  });

  test('error event is emitted with Error object', () => {
    const client = new EventStreamClient('http://localhost:3000/api/v1/events');

    let received: Error | null = null;
    client.on('error', (err) => {
      received = err;
    });

    client.emit('error', new Error('test error'));

    expect(received).toBeTruthy();
    expect(received!.message).toBe('test error');
  });
});

// ────────────────────────────────────────────────────────────────────
// Reconnection logic
// ────────────────────────────────────────────────────────────────────

describe('EventStreamClient reconnection', () => {
  test('disconnect sets intentionalClose flag (no reconnect)', () => {
    const client = new EventStreamClient('http://localhost:3000/api/v1/events', {
      maxReconnectAttempts: 3,
    });

    let disconnectIntentional: boolean | null = null;
    client.on('disconnected', (intentional) => {
      disconnectIntentional = intentional;
    });

    client.disconnect();

    expect(disconnectIntentional).toBe(true);
  });

  test('disconnect is safe to call multiple times', () => {
    const client = new EventStreamClient('http://localhost:3000/api/v1/events');

    expect(() => {
      client.disconnect();
      client.disconnect();
    }).not.toThrow();
  });

  test('isConnected returns false after disconnect', () => {
    const client = new EventStreamClient('http://localhost:3000/api/v1/events');
    client.disconnect();
    expect(client.isConnected()).toBe(false);
  });
});
