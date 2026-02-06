import { describe, test, expect, beforeEach } from 'bun:test';
import * as vscode from 'vscode';
import {
  getConfiguration,
  getConfigValue,
  onConfigurationChanged,
} from '../../src/settings/configuration';
import type { MehrhofConfiguration } from '../../src/settings/configuration';
import { createMockConfiguration, createMockWorkspace } from '../helpers/mockVscode';

// ────────────────────────────────────────────────────────────────────
// Source-imported tests — these exercise the actual source functions
// ────────────────────────────────────────────────────────────────────

describe('Configuration (Source Import)', () => {
  beforeEach(() => {
    // Reset the mock mehrhof config values before each test
    const config = vscode.workspace.getConfiguration('mehrhof') as unknown as {
      values: Map<string, unknown>;
    };
    config.values.clear();
  });

  describe('getConfiguration()', () => {
    test('returns all default values when nothing is set', () => {
      const config = getConfiguration();
      expect(config.serverUrl).toBe('');
      expect(config.mehrExecutable).toBe('');
      expect(config.showNotifications).toBe(true);
      expect(config.defaultAgent).toBe('');
      expect(config.autoReconnect).toBe(true);
      expect(config.reconnectDelaySeconds).toBe(5);
      expect(config.maxReconnectAttempts).toBe(10);
    });

    test('returns stored values when set', () => {
      const mockConfig = vscode.workspace.getConfiguration('mehrhof') as unknown as {
        values: Map<string, unknown>;
      };
      mockConfig.values.set('serverUrl', 'http://localhost:8080');
      mockConfig.values.set('mehrExecutable', '/usr/bin/mehr');
      mockConfig.values.set('showNotifications', false);
      mockConfig.values.set('defaultAgent', 'claude');
      mockConfig.values.set('autoReconnect', false);
      mockConfig.values.set('reconnectDelaySeconds', 15);
      mockConfig.values.set('maxReconnectAttempts', 20);

      const config = getConfiguration();

      expect(config.serverUrl).toBe('http://localhost:8080');
      expect(config.mehrExecutable).toBe('/usr/bin/mehr');
      expect(config.showNotifications).toBe(false);
      expect(config.defaultAgent).toBe('claude');
      expect(config.autoReconnect).toBe(false);
      expect(config.reconnectDelaySeconds).toBe(15);
      expect(config.maxReconnectAttempts).toBe(20);
    });

    test('returns correct MehrhofConfiguration type', () => {
      const config: MehrhofConfiguration = getConfiguration();
      expect(typeof config.serverUrl).toBe('string');
      expect(typeof config.showNotifications).toBe('boolean');
      expect(typeof config.reconnectDelaySeconds).toBe('number');
    });
  });

  describe('getConfigValue()', () => {
    test('returns specific value for serverUrl', () => {
      const mockConfig = vscode.workspace.getConfiguration('mehrhof') as unknown as {
        values: Map<string, unknown>;
      };
      mockConfig.values.set('serverUrl', 'http://custom:9000');

      expect(getConfigValue('serverUrl')).toBe('http://custom:9000');
    });

    test('returns default for unset key', () => {
      expect(getConfigValue('serverUrl')).toBe('');
      expect(getConfigValue('showNotifications')).toBe(true);
      expect(getConfigValue('reconnectDelaySeconds')).toBe(5);
    });

    test('returns defaultAgent value', () => {
      const mockConfig = vscode.workspace.getConfiguration('mehrhof') as unknown as {
        values: Map<string, unknown>;
      };
      mockConfig.values.set('defaultAgent', 'opus');

      expect(getConfigValue('defaultAgent')).toBe('opus');
    });
  });

  describe('onConfigurationChanged()', () => {
    test('returns a disposable', () => {
      const disposable = onConfigurationChanged(() => {});
      expect(disposable).toBeTruthy();
      expect(typeof disposable.dispose).toBe('function');
    });

    test('calls callback when mehrhof config changes', () => {
      let callbackCalled = false;

      onConfigurationChanged(() => {
        callbackCalled = true;
      });

      // Trigger the config change through the mock workspace
      const ws = vscode.workspace as unknown as {
        _configChangeHandler?: (e: { affectsConfiguration: (s: string) => boolean }) => void;
      };
      if (ws._configChangeHandler) {
        ws._configChangeHandler({
          affectsConfiguration: (section: string) => section === 'mehrhof',
        });
      }

      expect(callbackCalled).toBe(true);
    });

    test('does not call callback for other config changes', () => {
      let callbackCalled = false;

      onConfigurationChanged(() => {
        callbackCalled = true;
      });

      const ws = vscode.workspace as unknown as {
        _configChangeHandler?: (e: { affectsConfiguration: (s: string) => boolean }) => void;
      };
      if (ws._configChangeHandler) {
        ws._configChangeHandler({
          affectsConfiguration: (section: string) => section === 'editor.fontSize',
        });
      }

      expect(callbackCalled).toBe(false);
    });
  });
});

// ────────────────────────────────────────────────────────────────────
// Mock infrastructure tests — verify the mock helpers themselves
// ────────────────────────────────────────────────────────────────────

describe('MockConfiguration', () => {
  test('get() returns stored value', () => {
    const config = createMockConfiguration({ key: 'value' });
    expect(config.get('key')).toBe('value');
  });

  test('get() returns default when not set', () => {
    const config = createMockConfiguration();
    expect(config.get('missing', 'default')).toBe('default');
  });

  test('has() returns true for set values', () => {
    const config = createMockConfiguration({ key: 'value' });
    expect(config.has('key')).toBe(true);
  });

  test('has() returns false for unset values', () => {
    const config = createMockConfiguration();
    expect(config.has('missing')).toBe(false);
  });

  test('update() sets value', async () => {
    const config = createMockConfiguration();
    await config.update('newKey', 'newValue');
    expect(config.get('newKey')).toBe('newValue');
  });
});

describe('MockWorkspace', () => {
  test('getConfiguration returns same instance for same section', () => {
    const ws = createMockWorkspace();
    const c1 = ws.getConfiguration('mehrhof');
    const c2 = ws.getConfiguration('mehrhof');
    expect(c1).toBe(c2);
  });

  test('onDidChangeConfiguration returns disposable', () => {
    const ws = createMockWorkspace();
    const d = ws.onDidChangeConfiguration(() => {});
    expect(typeof d.dispose).toBe('function');
  });
});
