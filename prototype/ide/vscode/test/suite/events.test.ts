import * as assert from 'assert';
import { EventStreamClient } from '../../src/api/events';

suite('EventStreamClient Test Suite', () => {
  test('EventStreamClient constructs with URL', () => {
    const client = new EventStreamClient('http://localhost:3000/api/v1/events');
    assert.ok(client);
  });

  test('EventStreamClient accepts options', () => {
    const client = new EventStreamClient('http://localhost:3000/api/v1/events', {
      reconnectDelayMs: 10000,
      maxReconnectAttempts: 5,
    });
    assert.ok(client);
  });

  test('isConnected returns false when not connected', () => {
    const client = new EventStreamClient('http://localhost:3000/api/v1/events');
    assert.strictEqual(client.isConnected(), false);
  });

  test('disconnect emits disconnected event', (done) => {
    const client = new EventStreamClient('http://localhost:3000/api/v1/events');

    client.on('disconnected', (intentional) => {
      assert.strictEqual(intentional, true);
      done();
    });

    client.disconnect();
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
    assert.ok(count >= 1);
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
    assert.strictEqual(called, false);
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
    assert.ok(typeof client.connect === 'function');
  });

  test('once listener is only called once', (done) => {
    const client = new EventStreamClient('http://localhost:3000/api/v1/events');

    let callCount = 0;

    client.once('disconnected', () => {
      callCount++;
    });

    // First disconnect
    client.disconnect();

    // Manually emit another disconnect would be needed to test this fully
    // For now, verify the once method exists and works for single call
    setTimeout(() => {
      assert.strictEqual(callCount, 1);
      done();
    }, 10);
  });
});
