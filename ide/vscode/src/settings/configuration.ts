import * as vscode from 'vscode';

export interface MehrhofConfiguration {
  serverUrl: string;
  mehrExecutable: string;
  showNotifications: boolean;
  defaultAgent: string;
  autoReconnect: boolean;
  reconnectDelaySeconds: number;
  maxReconnectAttempts: number;
}

export function getConfiguration(): MehrhofConfiguration {
  const config = vscode.workspace.getConfiguration('mehrhof');

  return {
    serverUrl: config.get<string>('serverUrl', ''),
    mehrExecutable: config.get<string>('mehrExecutable', ''),
    showNotifications: config.get<boolean>('showNotifications', true),
    defaultAgent: config.get<string>('defaultAgent', ''),
    autoReconnect: config.get<boolean>('autoReconnect', true),
    reconnectDelaySeconds: config.get<number>('reconnectDelaySeconds', 5),
    maxReconnectAttempts: config.get<number>('maxReconnectAttempts', 10),
  };
}

export function getConfigValue<K extends keyof MehrhofConfiguration>(
  key: K
): MehrhofConfiguration[K] {
  return getConfiguration()[key];
}

export function onConfigurationChanged(
  callback: (e: vscode.ConfigurationChangeEvent) => void
): vscode.Disposable {
  return vscode.workspace.onDidChangeConfiguration((e) => {
    if (e.affectsConfiguration('mehrhof')) {
      callback(e);
    }
  });
}
