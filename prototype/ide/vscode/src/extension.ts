import * as vscode from 'vscode';
import { MehrhofProjectService } from './services/projectService';
import { MehrhofOutputChannel } from './views/outputChannel';
import { TaskTreeProvider } from './views/taskTreeProvider';
import { InteractivePanelProvider } from './views/interactivePanel';
import { StatusBarWidget } from './statusbar/statusWidget';
import { registerCommands } from './commands';

let projectService: MehrhofProjectService | undefined;
let outputChannel: MehrhofOutputChannel | undefined;
let statusBarWidget: StatusBarWidget | undefined;

export function activate(context: vscode.ExtensionContext): void {
  console.log('Mehrhof extension activating...');

  // Create output channel first (needed by project service)
  const vscodeOutputChannel = vscode.window.createOutputChannel('Mehrhof');
  context.subscriptions.push(vscodeOutputChannel);

  // Create project service
  projectService = new MehrhofProjectService(context, vscodeOutputChannel);
  context.subscriptions.push(projectService);

  // Create output channel wrapper for logging events
  outputChannel = new MehrhofOutputChannel(projectService);
  context.subscriptions.push(outputChannel);

  // Register commands
  registerCommands(context, projectService);

  // Create status bar widget
  statusBarWidget = new StatusBarWidget(projectService);
  context.subscriptions.push(statusBarWidget);

  // Register task tree view
  const taskTreeProvider = new TaskTreeProvider(projectService);
  context.subscriptions.push(taskTreeProvider);
  context.subscriptions.push(
    vscode.window.registerTreeDataProvider('mehrhof.tasks', taskTreeProvider)
  );

  // Register interactive panel webview
  const interactivePanelProvider = new InteractivePanelProvider(
    context.extensionUri,
    projectService
  );
  context.subscriptions.push(interactivePanelProvider);
  context.subscriptions.push(
    vscode.window.registerWebviewViewProvider(
      InteractivePanelProvider.viewType,
      interactivePanelProvider
    )
  );

  // Show notification on activation
  const config = vscode.workspace.getConfiguration('mehrhof');
  if (config.get<boolean>('showNotifications', true)) {
    void vscode.window.showInformationMessage('Mehrhof extension activated');
  }

  console.log('Mehrhof extension activated');
}

export function deactivate(): void {
  console.log('Mehrhof extension deactivating...');

  // Cleanup is handled by subscriptions, but we can do explicit cleanup if needed
  projectService?.dispose();
  outputChannel?.dispose();
  statusBarWidget?.dispose();

  console.log('Mehrhof extension deactivated');
}
