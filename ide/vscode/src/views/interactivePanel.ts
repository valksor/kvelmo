import * as vscode from 'vscode';
import { MehrhofProjectService } from '../services/projectService';
import { ApiError } from '../api/client';
import type { AgentMessageEvent } from '../api/models';
import { getInteractivePanelHtml } from './interactivePanelHtml';

interface WebviewMessage {
  type: string;
  payload?: unknown;
}

interface ChatMessage {
  role: 'user' | 'assistant' | 'system' | 'error' | 'command';
  content: string;
  timestamp: string;
}

export class InteractivePanelProvider implements vscode.WebviewViewProvider, vscode.Disposable {
  public static readonly viewType = 'mehrhof.interactive';

  private view?: vscode.WebviewView;
  private readonly service: MehrhofProjectService;
  private readonly extensionUri: vscode.Uri;
  private messages: ChatMessage[] = [];
  private commandHistory: string[] = [];
  private knownCommands: Set<string> = new Set();

  constructor(extensionUri: vscode.Uri, service: MehrhofProjectService) {
    this.extensionUri = extensionUri;
    this.service = service;

    // Listen for service events
    this.service.on('connectionChanged', () => {
      this.updateView();
      // Fetch commands when connection changes
      if (this.service.isConnected) {
        void this.fetchCommands();
      }
    });
    this.service.on('stateChanged', () => this.updateView());
    this.service.on('taskChanged', () => this.updateView());
    this.service.on('agentMessage', (event: AgentMessageEvent) => this.onAgentMessage(event));
    this.service.on('questionReceived', (question) => {
      this.addMessage('system', `Question: ${question.question}`);
      if (question.options?.length) {
        this.addMessage('system', `Options: ${question.options.join(', ')}`);
      }
    });
    this.service.on('error', (error) => {
      this.addMessage('error', error.message);
    });
  }

  resolveWebviewView(
    webviewView: vscode.WebviewView,
    _context: vscode.WebviewViewResolveContext,
    _token: vscode.CancellationToken
  ): void | Thenable<void> {
    this.view = webviewView;

    webviewView.webview.options = {
      enableScripts: true,
      localResourceRoots: [this.extensionUri],
    };

    webviewView.webview.html = this.getHtmlContent(webviewView.webview);

    // Handle messages from webview
    webviewView.webview.onDidReceiveMessage((message: WebviewMessage) => {
      void this.handleMessage(message);
    });

    // Send initial state
    this.updateView();
  }

  private async handleMessage(message: WebviewMessage): Promise<void> {
    switch (message.type) {
      case 'startServer':
        await this.executeCommand('mehrhof.startServer');
        break;
      case 'stopServer':
        this.service.stopServer();
        break;
      case 'connect':
        await this.executeCommand('mehrhof.connect');
        break;
      case 'disconnect':
        this.service.disconnect();
        break;
      case 'input':
        await this.handleInput(message.payload as string);
        break;
      case 'action':
        await this.handleAction(message.payload as string);
        break;
      case 'stop':
        await this.handleStop();
        break;
      case 'ready':
        this.updateView();
        break;
    }
  }

  private async executeCommand(command: string): Promise<void> {
    try {
      await vscode.commands.executeCommand(command);
    } catch (error) {
      const msg = error instanceof Error ? error.message : 'Unknown error';
      this.addMessage('error', msg);
    }
  }

  private async handleInput(input: string): Promise<void> {
    if (!input.trim()) {
      return;
    }

    this.commandHistory.push(input);
    this.addMessage('user', input);

    // Check if it's a command
    const trimmed = input.trim();
    const isCommand = this.isCommand(trimmed);

    if (isCommand) {
      await this.executeInteractiveCommand(trimmed);
    } else {
      await this.sendChat(trimmed);
    }
  }

  private async fetchCommands(): Promise<void> {
    if (!this.service.client) {
      return;
    }

    try {
      const response = await this.service.client.getCommands();
      this.knownCommands.clear();
      for (const cmd of response.commands) {
        this.knownCommands.add(cmd.name.toLowerCase());
        // Also add aliases
        if (cmd.aliases) {
          for (const alias of cmd.aliases) {
            this.knownCommands.add(alias.toLowerCase());
          }
        }
      }
    } catch {
      // API failed - commands will be sent to server anyway, which handles unknown commands
      console.warn('Failed to fetch commands from discovery API');
    }
  }

  private isCommand(input: string): boolean {
    // Check if input starts with a known command from discovery API
    // If discovery failed, knownCommands will be empty and all input goes to server
    const firstWord = input.split(/\s+/)[0].toLowerCase();
    return this.knownCommands.has(firstWord);
  }

  private async executeInteractiveCommand(input: string): Promise<void> {
    if (!this.service.isConnected || !this.service.client) {
      this.addMessage('error', 'Not connected');
      return;
    }

    const parts = input.trim().split(/\s+/);
    const command = parts[0].toLowerCase();
    const args = parts.slice(1);

    this.addMessage('command', `> ${input}`);

    try {
      const response = await this.service.client.executeCommand({ command, args });
      if (response.message) {
        this.addMessage('system', response.message);
      }
      if (response.error) {
        this.addMessage('error', response.error);
      }
      await this.service.refreshState();
    } catch (error) {
      const msg =
        error instanceof ApiError
          ? error.message
          : error instanceof Error
            ? error.message
            : 'Unknown error';
      this.addMessage('error', msg);
    }
  }

  private async sendChat(message: string): Promise<void> {
    if (!this.service.isConnected || !this.service.client) {
      this.addMessage('error', 'Not connected');
      return;
    }

    try {
      const response = await this.service.client.chat({ message });
      if (response.message) {
        this.addMessage('assistant', response.message);
      }
      if (response.error) {
        this.addMessage('error', response.error);
      }
    } catch (error) {
      const msg =
        error instanceof ApiError
          ? error.message
          : error instanceof Error
            ? error.message
            : 'Unknown error';
      this.addMessage('error', msg);
    }
  }

  private async handleAction(action: string): Promise<void> {
    // Actions that map directly to VS Code commands
    const commandMap: Record<string, string> = {
      startTask: 'mehrhof.startTask',
      plan: 'mehrhof.plan',
      implement: 'mehrhof.implement',
      review: 'mehrhof.review',
      continue: 'mehrhof.continue',
      finish: 'mehrhof.finish',
      abandon: 'mehrhof.abandon',
      undo: 'mehrhof.undo',
      redo: 'mehrhof.redo',
    };

    const command = commandMap[action];
    if (command) {
      await this.executeCommand(command);
      return;
    }

    // Actions that need prompts or direct interactive commands
    switch (action) {
      case 'status':
      case 'cost':
      case 'budget':
      case 'specification':
      case 'simplify':
        await this.executeInteractiveCommand(action);
        break;
      case 'find':
        await this.promptAndExecute('Enter search query:', 'find');
        break;
      case 'memory':
        await this.promptAndExecute('Enter search query:', 'memory');
        break;
      case 'library':
        await this.executeInteractiveCommand('library');
        break;
      case 'quick':
        await this.promptAndExecute('Enter task description:', 'quick');
        break;
      case 'note':
        await this.promptAndExecute('Enter note:', 'note');
        break;
      case 'list':
        await this.executeInteractiveCommand('list');
        break;
    }
  }

  private async promptAndExecute(prompt: string, command: string): Promise<void> {
    const input = await vscode.window.showInputBox({ prompt });
    if (input) {
      await this.executeInteractiveCommand(`${command} ${input}`);
    }
  }

  private async handleStop(): Promise<void> {
    if (!this.service.isConnected || !this.service.client) {
      return;
    }

    try {
      await this.service.client.stopOperation();
      this.addMessage('system', 'Operation cancelled');
    } catch (error) {
      const msg = error instanceof Error ? error.message : 'Unknown error';
      this.addMessage('error', msg);
    }
  }

  private onAgentMessage(event: AgentMessageEvent): void {
    const role = event.role === 'assistant' ? 'assistant' : 'system';
    this.addMessage(role, event.content);
  }

  private addMessage(role: ChatMessage['role'], content: string): void {
    this.messages.push({
      role,
      content,
      timestamp: new Date().toISOString(),
    });

    // Keep last 100 messages
    if (this.messages.length > 100) {
      this.messages = this.messages.slice(-100);
    }

    this.sendToWebview('messages', this.messages);
  }

  private updateView(): void {
    this.sendToWebview('state', {
      connected: this.service.isConnected,
      connecting: this.service.connectionState === 'connecting',
      serverRunning: this.service.isServerRunning(),
      workflowState: this.service.workflowState,
      task: this.service.currentTask,
      work: this.service.currentWork,
      pendingQuestion: this.service.pendingQuestion,
    });
    this.sendToWebview('messages', this.messages);
    this.sendToWebview('history', this.commandHistory);
  }

  private sendToWebview(type: string, payload: unknown): void {
    void this.view?.webview.postMessage({ type, payload });
  }

  private getHtmlContent(webview: vscode.Webview): string {
    return getInteractivePanelHtml(webview, this.extensionUri);
  }

  dispose(): void {
    // Cleanup
  }
}
