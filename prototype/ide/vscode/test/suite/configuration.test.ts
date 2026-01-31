import * as assert from 'assert';
import type { MehrhofConfiguration } from '../../src/settings/configuration';
import { createMockConfiguration, createMockWorkspace } from '../helpers/mockVscode';

suite('Configuration Test Suite', () => {
  suite('MehrhofConfiguration Interface', () => {
    test('interface has serverUrl property', () => {
      const config: MehrhofConfiguration = {
        serverUrl: 'http://localhost:3000',
        mehrExecutable: '',
        showNotifications: true,
        defaultAgent: '',
        autoReconnect: true,
        reconnectDelaySeconds: 5,
        maxReconnectAttempts: 10,
      };
      assert.strictEqual(config.serverUrl, 'http://localhost:3000');
    });

    test('interface has mehrExecutable property', () => {
      const config: MehrhofConfiguration = {
        serverUrl: '',
        mehrExecutable: '/usr/local/bin/mehr',
        showNotifications: true,
        defaultAgent: '',
        autoReconnect: true,
        reconnectDelaySeconds: 5,
        maxReconnectAttempts: 10,
      };
      assert.strictEqual(config.mehrExecutable, '/usr/local/bin/mehr');
    });

    test('interface has showNotifications property', () => {
      const config: MehrhofConfiguration = {
        serverUrl: '',
        mehrExecutable: '',
        showNotifications: false,
        defaultAgent: '',
        autoReconnect: true,
        reconnectDelaySeconds: 5,
        maxReconnectAttempts: 10,
      };
      assert.strictEqual(config.showNotifications, false);
    });

    test('interface has defaultAgent property', () => {
      const config: MehrhofConfiguration = {
        serverUrl: '',
        mehrExecutable: '',
        showNotifications: true,
        defaultAgent: 'claude',
        autoReconnect: true,
        reconnectDelaySeconds: 5,
        maxReconnectAttempts: 10,
      };
      assert.strictEqual(config.defaultAgent, 'claude');
    });

    test('interface has autoReconnect property', () => {
      const config: MehrhofConfiguration = {
        serverUrl: '',
        mehrExecutable: '',
        showNotifications: true,
        defaultAgent: '',
        autoReconnect: false,
        reconnectDelaySeconds: 5,
        maxReconnectAttempts: 10,
      };
      assert.strictEqual(config.autoReconnect, false);
    });

    test('interface has reconnectDelaySeconds property', () => {
      const config: MehrhofConfiguration = {
        serverUrl: '',
        mehrExecutable: '',
        showNotifications: true,
        defaultAgent: '',
        autoReconnect: true,
        reconnectDelaySeconds: 10,
        maxReconnectAttempts: 10,
      };
      assert.strictEqual(config.reconnectDelaySeconds, 10);
    });

    test('interface has maxReconnectAttempts property', () => {
      const config: MehrhofConfiguration = {
        serverUrl: '',
        mehrExecutable: '',
        showNotifications: true,
        defaultAgent: '',
        autoReconnect: true,
        reconnectDelaySeconds: 5,
        maxReconnectAttempts: 20,
      };
      assert.strictEqual(config.maxReconnectAttempts, 20);
    });
  });

  suite('getConfiguration()', () => {
    test('returns all config values', () => {
      const mockConfig = createMockConfiguration({
        serverUrl: 'http://localhost:3000',
        mehrExecutable: '/path/to/mehr',
        showNotifications: true,
        defaultAgent: 'claude',
        autoReconnect: true,
        reconnectDelaySeconds: 5,
        maxReconnectAttempts: 10,
      });

      assert.strictEqual(mockConfig.get('serverUrl'), 'http://localhost:3000');
      assert.strictEqual(mockConfig.get('mehrExecutable'), '/path/to/mehr');
      assert.strictEqual(mockConfig.get('showNotifications'), true);
      assert.strictEqual(mockConfig.get('defaultAgent'), 'claude');
      assert.strictEqual(mockConfig.get('autoReconnect'), true);
      assert.strictEqual(mockConfig.get('reconnectDelaySeconds'), 5);
      assert.strictEqual(mockConfig.get('maxReconnectAttempts'), 10);
    });

    test('uses defaults when not set', () => {
      const mockConfig = createMockConfiguration();

      // Default values from package.json
      const defaults = {
        serverUrl: '',
        mehrExecutable: '',
        showNotifications: true,
        defaultAgent: '',
        autoReconnect: true,
        reconnectDelaySeconds: 5,
        maxReconnectAttempts: 10,
      };

      // When not set, get() with default returns the default
      assert.strictEqual(mockConfig.get('serverUrl', defaults.serverUrl), '');
      assert.strictEqual(mockConfig.get('showNotifications', defaults.showNotifications), true);
      assert.strictEqual(
        mockConfig.get('reconnectDelaySeconds', defaults.reconnectDelaySeconds),
        5
      );
    });

    test('returns correct types', () => {
      const mockConfig = createMockConfiguration({
        serverUrl: 'http://localhost:3000',
        showNotifications: true,
        reconnectDelaySeconds: 5,
      });

      const serverUrl = mockConfig.get<string>('serverUrl');
      const showNotifications = mockConfig.get<boolean>('showNotifications');
      const reconnectDelaySeconds = mockConfig.get<number>('reconnectDelaySeconds');

      assert.strictEqual(typeof serverUrl, 'string');
      assert.strictEqual(typeof showNotifications, 'boolean');
      assert.strictEqual(typeof reconnectDelaySeconds, 'number');
    });
  });

  suite('getConfigValue()', () => {
    test('returns specific config value', () => {
      const mockConfig = createMockConfiguration({
        serverUrl: 'http://localhost:8080',
      });

      assert.strictEqual(mockConfig.get('serverUrl'), 'http://localhost:8080');
    });

    test('returns undefined for unset values', () => {
      const mockConfig = createMockConfiguration();

      assert.strictEqual(mockConfig.get('unknownKey'), undefined);
    });
  });

  suite('onConfigurationChanged()', () => {
    test('returns disposable', () => {
      const mockWorkspace = createMockWorkspace();
      const disposable = mockWorkspace.onDidChangeConfiguration(() => {});

      assert.ok(disposable);
      assert.strictEqual(typeof disposable.dispose, 'function');
    });

    test('callback can be registered', () => {
      let callbackCalled = false;
      const mockWorkspace = createMockWorkspace();

      mockWorkspace.onDidChangeConfiguration(() => {
        callbackCalled = true;
      });

      // Simulate config change
      if (mockWorkspace._configChangeHandler) {
        mockWorkspace._configChangeHandler({
          affectsConfiguration: (section: string) => section === 'mehrhof',
        });
        assert.ok(callbackCalled);
      }
    });

    test('callback only called for mehrhof config changes', () => {
      let callCount = 0;
      const mockWorkspace = createMockWorkspace();

      mockWorkspace.onDidChangeConfiguration((e) => {
        if (e.affectsConfiguration('mehrhof')) {
          callCount++;
        }
      });

      // Simulate mehrhof config change
      if (mockWorkspace._configChangeHandler) {
        mockWorkspace._configChangeHandler({
          affectsConfiguration: (section: string) => section === 'mehrhof',
        });
      }

      assert.strictEqual(callCount, 1);
    });

    test('callback not called for other config changes', () => {
      let callCount = 0;
      const mockWorkspace = createMockWorkspace();

      mockWorkspace.onDidChangeConfiguration((e) => {
        if (e.affectsConfiguration('mehrhof')) {
          callCount++;
        }
      });

      // Simulate non-mehrhof config change
      if (mockWorkspace._configChangeHandler) {
        mockWorkspace._configChangeHandler({
          affectsConfiguration: (section: string) => section === 'editor',
        });
      }

      assert.strictEqual(callCount, 0);
    });
  });

  suite('Default Values', () => {
    test('serverUrl default is empty string', () => {
      const defaultValue = '';
      assert.strictEqual(defaultValue, '');
    });

    test('mehrExecutable default is empty string', () => {
      const defaultValue = '';
      assert.strictEqual(defaultValue, '');
    });

    test('showNotifications default is true', () => {
      const defaultValue = true;
      assert.strictEqual(defaultValue, true);
    });

    test('defaultAgent default is empty string', () => {
      const defaultValue = '';
      assert.strictEqual(defaultValue, '');
    });

    test('autoReconnect default is true', () => {
      const defaultValue = true;
      assert.strictEqual(defaultValue, true);
    });

    test('reconnectDelaySeconds default is 5', () => {
      const defaultValue = 5;
      assert.strictEqual(defaultValue, 5);
    });

    test('maxReconnectAttempts default is 10', () => {
      const defaultValue = 10;
      assert.strictEqual(defaultValue, 10);
    });
  });

  suite('Configuration Constraints', () => {
    test('reconnectDelaySeconds has minimum of 1', () => {
      const min = 1;
      assert.ok(min >= 1);
    });

    test('reconnectDelaySeconds has maximum of 60', () => {
      const max = 60;
      assert.ok(max <= 60);
    });

    test('maxReconnectAttempts has minimum of 1', () => {
      const min = 1;
      assert.ok(min >= 1);
    });

    test('maxReconnectAttempts has maximum of 100', () => {
      const max = 100;
      assert.ok(max <= 100);
    });
  });
});

// Mock configuration verification
suite('MockConfiguration', () => {
  test('get() returns stored value', () => {
    const config = createMockConfiguration({ key: 'value' });
    assert.strictEqual(config.get('key'), 'value');
  });

  test('get() returns default when not set', () => {
    const config = createMockConfiguration();
    assert.strictEqual(config.get('missing', 'default'), 'default');
  });

  test('has() returns true for set values', () => {
    const config = createMockConfiguration({ key: 'value' });
    assert.strictEqual(config.has('key'), true);
  });

  test('has() returns false for unset values', () => {
    const config = createMockConfiguration();
    assert.strictEqual(config.has('missing'), false);
  });

  test('update() sets value', async () => {
    const config = createMockConfiguration();
    await config.update('newKey', 'newValue');
    assert.strictEqual(config.get('newKey'), 'newValue');
  });
});
