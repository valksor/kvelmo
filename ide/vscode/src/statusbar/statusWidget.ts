import * as vscode from 'vscode';
import { MehrhofProjectService } from '../services/projectService';

export class StatusBarWidget implements vscode.Disposable {
  private readonly statusBarItem: vscode.StatusBarItem;
  private readonly service: MehrhofProjectService;

  constructor(service: MehrhofProjectService) {
    this.service = service;

    this.statusBarItem = vscode.window.createStatusBarItem(
      vscode.StatusBarAlignment.Left,
      100
    );
    this.statusBarItem.command = 'mehrhof.statusBarClicked';
    this.statusBarItem.show();

    // Register click command
    vscode.commands.registerCommand('mehrhof.statusBarClicked', () => {
      this.onClicked();
    });

    // Listen for state changes
    this.service.on('connectionChanged', () => this.update());
    this.service.on('stateChanged', () => this.update());
    this.service.on('taskChanged', () => this.update());

    // Initial update
    this.update();
  }

  private update(): void {
    const connectionState = this.service.connectionState;
    const workflowState = this.service.workflowState;
    const task = this.service.currentTask;
    const work = this.service.currentWork;

    if (connectionState === 'disconnected') {
      this.statusBarItem.text = '$(circle-slash) Mehrhof: Disconnected';
      this.statusBarItem.tooltip = 'Click to connect';
      this.statusBarItem.backgroundColor = undefined;
      return;
    }

    if (connectionState === 'connecting') {
      this.statusBarItem.text = '$(sync~spin) Mehrhof: Connecting...';
      this.statusBarItem.tooltip = 'Connecting to server';
      this.statusBarItem.backgroundColor = undefined;
      return;
    }

    // Connected
    const stateIcon = this.getStateIcon(workflowState);
    const stateDisplay = this.formatState(workflowState);

    if (task && work?.title) {
      this.statusBarItem.text = `${stateIcon} Mehrhof: ${stateDisplay} - ${this.truncate(work.title, 30)}`;
      this.statusBarItem.tooltip = this.buildTooltip(workflowState, task.id, work.title, task.branch);
    } else if (task) {
      this.statusBarItem.text = `${stateIcon} Mehrhof: ${stateDisplay} - ${this.truncate(task.id, 10)}`;
      this.statusBarItem.tooltip = this.buildTooltip(workflowState, task.id, undefined, task.branch);
    } else {
      this.statusBarItem.text = `${stateIcon} Mehrhof: ${stateDisplay}`;
      this.statusBarItem.tooltip = 'Click to show actions';
    }

    // Set background color for active states
    if (workflowState === 'planning' || workflowState === 'implementing' || workflowState === 'reviewing') {
      this.statusBarItem.backgroundColor = new vscode.ThemeColor('statusBarItem.warningBackground');
    } else {
      this.statusBarItem.backgroundColor = undefined;
    }
  }

  private getStateIcon(state: string): string {
    switch (state) {
      case 'idle':
        return '$(circle-outline)';
      case 'planning':
        return '$(edit)';
      case 'implementing':
        return '$(code)';
      case 'reviewing':
        return '$(eye)';
      case 'waiting':
        return '$(question)';
      case 'checkpointing':
      case 'reverting':
      case 'restoring':
        return '$(sync~spin)';
      case 'done':
        return '$(check)';
      case 'failed':
        return '$(error)';
      default:
        return '$(circle-filled)';
    }
  }

  private formatState(state: string): string {
    return state.charAt(0).toUpperCase() + state.slice(1);
  }

  private truncate(text: string, maxLength: number): string {
    if (text.length <= maxLength) {
      return text;
    }
    return text.substring(0, maxLength - 3) + '...';
  }

  private buildTooltip(state: string, taskId: string, title?: string, branch?: string): string {
    const lines: string[] = [];
    lines.push(`State: ${this.formatState(state)}`);
    lines.push(`Task: ${taskId}`);
    if (title) {
      lines.push(`Title: ${title}`);
    }
    if (branch) {
      lines.push(`Branch: ${branch}`);
    }
    lines.push('');
    lines.push('Click to show actions');
    return lines.join('\n');
  }

  private onClicked(): void {
    const connectionState = this.service.connectionState;

    if (connectionState === 'disconnected') {
      void vscode.commands.executeCommand('mehrhof.connect');
      return;
    }

    // Show quick pick with available actions
    const items: vscode.QuickPickItem[] = [];

    if (this.service.isConnected) {
      const state = this.service.workflowState;

      if (state === 'idle') {
        items.push({ label: '$(add) Start Task', description: 'Start a new task' });
      }

      if (state === 'idle' || state === 'planning') {
        items.push({ label: '$(edit) Plan', description: 'Start planning' });
      }

      if (state === 'implementing' || state === 'planning') {
        items.push({ label: '$(code) Implement', description: 'Start implementation' });
      }

      if (state === 'implementing' || state === 'reviewing') {
        items.push({ label: '$(eye) Review', description: 'Start review' });
      }

      if (state !== 'idle') {
        items.push({ label: '$(check) Finish', description: 'Complete the task' });
        items.push({ label: '$(discard) Abandon', description: 'Abandon the task' });
      }

      items.push({ label: '$(history) Undo', description: 'Undo last checkpoint' });
      items.push({ label: '$(redo) Redo', description: 'Redo checkpoint' });
      items.push({ label: '$(info) Status', description: 'Show task status' });
      items.push({ label: '$(refresh) Refresh', description: 'Refresh state' });
      items.push({ label: '$(debug-disconnect) Disconnect', description: 'Disconnect from server' });
    }

    void vscode.window.showQuickPick(items, { placeHolder: 'Mehrhof Actions' }).then((selected) => {
      if (!selected) {
        return;
      }

      const command = this.labelToCommand(selected.label);
      if (command) {
        void vscode.commands.executeCommand(command);
      }
    });
  }

  private labelToCommand(label: string): string | undefined {
    const mapping: Record<string, string> = {
      '$(add) Start Task': 'mehrhof.startTask',
      '$(edit) Plan': 'mehrhof.plan',
      '$(code) Implement': 'mehrhof.implement',
      '$(eye) Review': 'mehrhof.review',
      '$(check) Finish': 'mehrhof.finish',
      '$(discard) Abandon': 'mehrhof.abandon',
      '$(history) Undo': 'mehrhof.undo',
      '$(redo) Redo': 'mehrhof.redo',
      '$(info) Status': 'mehrhof.status',
      '$(refresh) Refresh': 'mehrhof.refresh',
      '$(debug-disconnect) Disconnect': 'mehrhof.disconnect',
    };
    return mapping[label];
  }

  dispose(): void {
    this.statusBarItem.dispose();
  }
}
