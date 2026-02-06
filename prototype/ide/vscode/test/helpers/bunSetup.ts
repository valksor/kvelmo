/**
 * Bun test preload: intercepts `import * as vscode from 'vscode'` so that
 * all source files resolve to the mock instead of the real VS Code module.
 * This eliminates the need for @vscode/test-electron entirely.
 */
import { mock } from 'bun:test';
import { createMockVscode } from './mockVscode';

void mock.module('vscode', () => createMockVscode());
