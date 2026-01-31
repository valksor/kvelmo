import * as vscode from 'vscode';
import { MehrhofProjectService } from '../services/projectService';
import type { TaskSummary } from '../api/models';

export class TaskItem extends vscode.TreeItem {
  constructor(
    public readonly task: TaskSummary,
    public readonly isActive: boolean
  ) {
    super(task.title ?? task.id, vscode.TreeItemCollapsibleState.None);

    this.id = task.id;
    this.description = this.formatState(task.state);
    this.tooltip = this.buildTooltip();
    this.iconPath = this.getIcon();
    this.contextValue = isActive ? 'activeTask' : 'task';
  }

  private formatState(state: string): string {
    return state.charAt(0).toUpperCase() + state.slice(1);
  }

  private getIcon(): vscode.ThemeIcon {
    switch (this.task.state) {
      case 'done':
        return new vscode.ThemeIcon('check', new vscode.ThemeColor('charts.green'));
      case 'failed':
        return new vscode.ThemeIcon('error', new vscode.ThemeColor('charts.red'));
      case 'idle':
        return new vscode.ThemeIcon('circle-outline');
      case 'planning':
        return new vscode.ThemeIcon('edit', new vscode.ThemeColor('charts.blue'));
      case 'implementing':
        return new vscode.ThemeIcon('code', new vscode.ThemeColor('charts.orange'));
      case 'reviewing':
        return new vscode.ThemeIcon('eye', new vscode.ThemeColor('charts.purple'));
      case 'waiting':
        return new vscode.ThemeIcon('question', new vscode.ThemeColor('charts.yellow'));
      default:
        return new vscode.ThemeIcon('circle-filled');
    }
  }

  private buildTooltip(): string {
    const lines: string[] = [];
    lines.push(`ID: ${this.task.id}`);
    if (this.task.title) {
      lines.push(`Title: ${this.task.title}`);
    }
    lines.push(`State: ${this.formatState(this.task.state)}`);
    if (this.task.created_at) {
      lines.push(`Created: ${new Date(this.task.created_at).toLocaleString()}`);
    }
    if (this.isActive) {
      lines.push('(Active Task)');
    }
    return lines.join('\n');
  }
}

export class TaskTreeProvider implements vscode.TreeDataProvider<TaskItem>, vscode.Disposable {
  private readonly service: MehrhofProjectService;
  private tasks: TaskSummary[] = [];

  private readonly _onDidChangeTreeData = new vscode.EventEmitter<TaskItem | undefined | null | void>();
  readonly onDidChangeTreeData = this._onDidChangeTreeData.event;

  constructor(service: MehrhofProjectService) {
    this.service = service;

    // Listen for state changes
    this.service.on('connectionChanged', () => {
      void this.refresh();
    });

    this.service.on('taskChanged', () => {
      void this.refresh();
    });
  }

  async refresh(): Promise<void> {
    if (!this.service.isConnected || !this.service.client) {
      this.tasks = [];
      this._onDidChangeTreeData.fire();
      return;
    }

    try {
      const response = await this.service.client.getTasks();
      this.tasks = response.tasks;
      this._onDidChangeTreeData.fire();
    } catch {
      this.tasks = [];
      this._onDidChangeTreeData.fire();
    }
  }

  getTreeItem(element: TaskItem): vscode.TreeItem {
    return element;
  }

  getChildren(_element?: TaskItem): Thenable<TaskItem[]> {
    if (!this.service.isConnected) {
      return Promise.resolve([]);
    }

    const currentTaskId = this.service.currentTask?.id;
    const items = this.tasks.map((task) => new TaskItem(task, task.id === currentTaskId));

    // Sort: active task first, then by created_at descending
    items.sort((a, b) => {
      if (a.isActive && !b.isActive) {
        return -1;
      }
      if (!a.isActive && b.isActive) {
        return 1;
      }
      const aDate = a.task.created_at ? new Date(a.task.created_at).getTime() : 0;
      const bDate = b.task.created_at ? new Date(b.task.created_at).getTime() : 0;
      return bDate - aDate;
    });

    return Promise.resolve(items);
  }

  dispose(): void {
    this._onDidChangeTreeData.dispose();
  }
}
