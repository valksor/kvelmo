import * as vscode from 'vscode';
import { MehrhofProjectService } from '../services/projectService';
import { registerBrowserCommands } from './browser';
import { registerProjectCommands } from './project';
import { registerSearchCommands } from './search';
import { registerTaskCommands } from './tasks';
import { registerWorkflowCommands } from './workflow';

export function registerCommands(
  context: vscode.ExtensionContext,
  service: MehrhofProjectService
): void {
  registerWorkflowCommands(context, service);
  registerTaskCommands(context, service);
  registerSearchCommands(context, service);
  registerBrowserCommands(context, service);
  registerProjectCommands(context, service);
}
