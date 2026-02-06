import * as vscode from 'vscode';
import { MehrhofProjectService } from '../services/projectService';
import { ApiError } from '../api/client';

export function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return `${parseFloat((bytes / Math.pow(k, i)).toFixed(1))} ${sizes[i]}`;
}

export function truncateUrl(url: string, maxLen: number): string {
  if (url.length <= maxLen) return url;
  return url.substring(0, maxLen - 3) + '...';
}

export function requireConnection(service: MehrhofProjectService): boolean {
  if (!service.isConnected || !service.client) {
    void vscode.window.showWarningMessage('Mehrhof: Not connected. Please connect first.');
    return false;
  }
  return true;
}

export async function withProgress<T>(
  title: string,
  task: () => Promise<T>
): Promise<T | undefined> {
  try {
    return await vscode.window.withProgress(
      {
        location: vscode.ProgressLocation.Notification,
        title: `Mehrhof: ${title}`,
        cancellable: false,
      },
      async () => {
        return await task();
      }
    );
  } catch (error) {
    const message =
      error instanceof ApiError
        ? error.message
        : error instanceof Error
          ? error.message
          : 'Unknown error';
    void vscode.window.showErrorMessage(`Mehrhof: ${message}`);
    return undefined;
  }
}
