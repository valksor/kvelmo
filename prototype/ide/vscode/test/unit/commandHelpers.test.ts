import { describe, test, expect, beforeEach } from 'bun:test';
import * as vscode from 'vscode';
import {
  formatBytes,
  truncateUrl,
  requireConnection,
  withProgress,
} from '../../src/commands/helpers';
import { ApiError } from '../../src/api/client';

describe('formatBytes', () => {
  const cases: { input: number; expected: string }[] = [
    { input: 0, expected: '0 B' },
    { input: 1, expected: '1 B' },
    { input: 512, expected: '512 B' },
    { input: 1023, expected: '1023 B' },
    { input: 1024, expected: '1 KB' },
    { input: 1536, expected: '1.5 KB' },
    { input: 1048576, expected: '1 MB' },
    { input: 1572864, expected: '1.5 MB' },
    { input: 1073741824, expected: '1 GB' },
    { input: 2621440, expected: '2.5 MB' },
  ];

  for (const { input, expected } of cases) {
    test(`${input} -> '${expected}'`, () => {
      expect(formatBytes(input)).toBe(expected);
    });
  }
});

describe('truncateUrl', () => {
  const cases: { url: string; maxLen: number; expected: string }[] = [
    { url: 'https://example.com', maxLen: 50, expected: 'https://example.com' },
    { url: 'https://example.com', maxLen: 19, expected: 'https://example.com' },
    { url: 'https://example.com/very/long/path', maxLen: 20, expected: 'https://example.c...' },
    { url: 'abcdefghij', maxLen: 10, expected: 'abcdefghij' },
    { url: 'abcdefghijk', maxLen: 10, expected: 'abcdefg...' },
    { url: 'ab', maxLen: 10, expected: 'ab' },
  ];

  for (const { url, maxLen, expected } of cases) {
    test(`'${url}' (max ${maxLen}) -> '${expected}'`, () => {
      expect(truncateUrl(url, maxLen)).toBe(expected);
    });
  }
});

describe('requireConnection', () => {
  function createMockService(isConnected: boolean, hasClient: boolean) {
    return {
      isConnected,
      client: hasClient ? {} : null,
    } as unknown as import('../../src/services/projectService').MehrhofProjectService;
  }

  let warningCalls: string[];

  beforeEach(() => {
    warningCalls = [];
    (vscode.window.showWarningMessage as unknown) = (...args: unknown[]) => {
      warningCalls.push(args[0] as string);
      return Promise.resolve(undefined);
    };
  });

  test('returns true when connected with client', () => {
    expect(requireConnection(createMockService(true, true))).toBe(true);
    expect(warningCalls.length).toBe(0);
  });

  test('returns false when not connected', () => {
    expect(requireConnection(createMockService(false, true))).toBe(false);
    expect(warningCalls.length).toBe(1);
    expect(warningCalls[0]).toContain('Not connected');
  });

  test('returns false when client is null', () => {
    expect(requireConnection(createMockService(true, false))).toBe(false);
    expect(warningCalls.length).toBe(1);
  });

  test('returns false when both disconnected and no client', () => {
    expect(requireConnection(createMockService(false, false))).toBe(false);
    expect(warningCalls.length).toBe(1);
  });
});

describe('withProgress', () => {
  let progressOptions: { title: string }[];
  let errorCalls: string[];

  beforeEach(() => {
    progressOptions = [];
    errorCalls = [];

    (vscode.window.withProgress as unknown) = async (
      options: { title: string },
      task: (progress: unknown, token: unknown) => Promise<unknown>
    ) => {
      progressOptions.push(options);
      return task({}, {});
    };

    (vscode.window.showErrorMessage as unknown) = (...args: unknown[]) => {
      errorCalls.push(args[0] as string);
      return Promise.resolve(undefined);
    };
  });

  test('returns the task result on success', async () => {
    const result = await withProgress('Testing...', () => Promise.resolve(42));
    expect(result).toBe(42);
  });

  test('prefixes title with Mehrhof:', async () => {
    await withProgress('Building...', () => Promise.resolve());
    expect(progressOptions[0].title).toBe('Mehrhof: Building...');
  });

  test('returns undefined and shows error on Error', async () => {
    const result = await withProgress('Failing...', () =>
      Promise.reject(new Error('something broke'))
    );
    expect(result).toBeUndefined();
    expect(errorCalls.length).toBe(1);
    expect(errorCalls[0]).toContain('something broke');
  });

  test('returns undefined and shows error on ApiError', async () => {
    const result = await withProgress('API fail...', () =>
      Promise.reject(new ApiError('server error', 500))
    );
    expect(result).toBeUndefined();
    expect(errorCalls.length).toBe(1);
    expect(errorCalls[0]).toContain('server error');
  });

  test('shows Unknown error for non-Error throws', async () => {
    const result = await withProgress(
      'Unknown...',
      () => Promise.reject('string-error') // eslint-disable-line @typescript-eslint/prefer-promise-reject-errors
    );
    expect(result).toBeUndefined();
    expect(errorCalls[0]).toContain('Unknown error');
  });
});
