import * as vscode from 'vscode';
import { MehrhofProjectService } from '../services/projectService';
import { ApiError } from '../api/client';
import type { AgentMessageEvent } from '../api/models';

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

  constructor(extensionUri: vscode.Uri, service: MehrhofProjectService) {
    this.extensionUri = extensionUri;
    this.service = service;

    // Listen for service events
    this.service.on('connectionChanged', () => this.updateView());
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

  private isCommand(input: string): boolean {
    const commands = [
      'start', 'plan', 'implement', 'review', 'finish', 'abandon', 'continue',
      'undo', 'redo', 'status', 'st', 'cost', 'list', 'budget', 'help',
      'note', 'quick', 'label', 'specification', 'spec', 'find', 'memory', 'simplify'
    ];
    const firstWord = input.split(/\s+/)[0].toLowerCase();
    return commands.includes(firstWord);
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
      const msg = error instanceof ApiError ? error.message : error instanceof Error ? error.message : 'Unknown error';
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
      const msg = error instanceof ApiError ? error.message : error instanceof Error ? error.message : 'Unknown error';
      this.addMessage('error', msg);
    }
  }

  private async handleAction(action: string): Promise<void> {
    const commandMap: Record<string, string> = {
      'startTask': 'mehrhof.startTask',
      'plan': 'mehrhof.plan',
      'implement': 'mehrhof.implement',
      'review': 'mehrhof.review',
      'continue': 'mehrhof.continue',
      'finish': 'mehrhof.finish',
      'abandon': 'mehrhof.abandon',
      'undo': 'mehrhof.undo',
      'redo': 'mehrhof.redo',
      'status': 'mehrhof.status',
    };

    const command = commandMap[action];
    if (command) {
      await this.executeCommand(command);
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

  private getHtmlContent(_webview: vscode.Webview): string {
    // Note: escapeHtml function below uses textContent to safely escape HTML
    // before using innerHTML - this is a safe pattern that prevents XSS
    return `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Mehrhof Interactive</title>
  <style>
    :root {
      --container-padding: 8px;
      --input-padding: 6px 8px;
      --border-radius: 4px;
    }

    * {
      box-sizing: border-box;
    }

    body {
      padding: 0;
      margin: 0;
      font-family: var(--vscode-font-family);
      font-size: var(--vscode-font-size);
      color: var(--vscode-foreground);
      background-color: var(--vscode-sideBar-background);
    }

    .container {
      display: flex;
      flex-direction: column;
      height: 100vh;
      padding: var(--container-padding);
    }

    .header {
      display: flex;
      flex-direction: column;
      gap: 8px;
      padding-bottom: 8px;
      border-bottom: 1px solid var(--vscode-panel-border);
    }

    .server-controls {
      display: flex;
      align-items: center;
      gap: 8px;
    }

    .task-info {
      display: flex;
      align-items: center;
      gap: 8px;
      font-size: 12px;
    }

    .state-badge {
      padding: 2px 6px;
      border-radius: var(--border-radius);
      font-size: 11px;
      font-weight: 500;
      text-transform: uppercase;
    }

    .state-idle { background: var(--vscode-badge-background); color: var(--vscode-badge-foreground); }
    .state-planning { background: var(--vscode-charts-blue); color: white; }
    .state-implementing { background: var(--vscode-charts-orange); color: white; }
    .state-reviewing { background: var(--vscode-charts-purple); color: white; }
    .state-waiting { background: var(--vscode-charts-yellow); color: black; }
    .state-done { background: var(--vscode-charts-green); color: white; }
    .state-failed { background: var(--vscode-charts-red); color: white; }

    .messages {
      flex: 1;
      overflow-y: auto;
      padding: 8px 0;
    }

    .message {
      padding: 4px 0;
      line-height: 1.4;
      white-space: pre-wrap;
      word-break: break-word;
    }

    .message-user { color: var(--vscode-terminal-ansiCyan); }
    .message-assistant { color: var(--vscode-foreground); }
    .message-system { color: var(--vscode-descriptionForeground); }
    .message-error { color: var(--vscode-errorForeground); }
    .message-command { color: var(--vscode-terminal-ansiGreen); font-family: monospace; }

    .input-area {
      display: flex;
      gap: 4px;
      padding-top: 8px;
      border-top: 1px solid var(--vscode-panel-border);
    }

    .input-field {
      flex: 1;
      padding: var(--input-padding);
      border: 1px solid var(--vscode-input-border);
      background: var(--vscode-input-background);
      color: var(--vscode-input-foreground);
      border-radius: var(--border-radius);
      font-family: inherit;
      font-size: inherit;
    }

    .input-field:focus {
      outline: 1px solid var(--vscode-focusBorder);
    }

    button {
      padding: var(--input-padding);
      border: none;
      background: var(--vscode-button-background);
      color: var(--vscode-button-foreground);
      border-radius: var(--border-radius);
      cursor: pointer;
      font-family: inherit;
      font-size: inherit;
    }

    button:hover {
      background: var(--vscode-button-hoverBackground);
    }

    button:disabled {
      opacity: 0.5;
      cursor: not-allowed;
    }

    button.secondary {
      background: var(--vscode-button-secondaryBackground);
      color: var(--vscode-button-secondaryForeground);
    }

    button.secondary:hover {
      background: var(--vscode-button-secondaryHoverBackground);
    }

    .actions {
      display: flex;
      flex-wrap: wrap;
      gap: 4px;
      padding-top: 8px;
    }

    .actions button {
      font-size: 11px;
      padding: 4px 8px;
    }

    .disconnected-notice {
      text-align: center;
      padding: 20px;
      color: var(--vscode-descriptionForeground);
    }
  </style>
</head>
<body>
  <div class="container">
    <div class="header">
      <div class="server-controls">
        <button id="serverBtn">Start Server</button>
        <span id="connectionStatus">Disconnected</span>
      </div>
      <div class="task-info" id="taskInfo" style="display: none;">
        <span id="taskTitle">No task</span>
        <span class="state-badge state-idle" id="stateBadge">Idle</span>
      </div>
    </div>

    <div class="messages" id="messages">
      <div class="disconnected-notice" id="disconnectedNotice">
        Start the server or connect to begin.
      </div>
    </div>

    <div class="input-area">
      <input type="text" class="input-field" id="inputField" placeholder="Type a command or message..." disabled>
      <button id="stopBtn" class="secondary" disabled>Stop</button>
      <button id="sendBtn" disabled>Send</button>
    </div>

    <div class="actions">
      <button data-action="startTask" disabled>Start Task</button>
      <button data-action="plan" disabled>Plan</button>
      <button data-action="implement" disabled>Implement</button>
      <button data-action="review" disabled>Review</button>
      <button data-action="continue" disabled>Continue</button>
      <button data-action="finish" disabled>Finish</button>
      <button data-action="abandon" class="secondary" disabled>Abandon</button>
      <button data-action="undo" class="secondary" disabled>Undo</button>
      <button data-action="redo" class="secondary" disabled>Redo</button>
    </div>
  </div>

  <script>
    const vscode = acquireVsCodeApi();

    let state = {
      connected: false,
      connecting: false,
      serverRunning: false,
      workflowState: 'idle',
      task: null,
      work: null,
    };
    let messages = [];
    let commandHistory = [];
    let historyIndex = -1;

    const serverBtn = document.getElementById('serverBtn');
    const connectionStatus = document.getElementById('connectionStatus');
    const taskInfo = document.getElementById('taskInfo');
    const taskTitle = document.getElementById('taskTitle');
    const stateBadge = document.getElementById('stateBadge');
    const messagesEl = document.getElementById('messages');
    const disconnectedNotice = document.getElementById('disconnectedNotice');
    const inputField = document.getElementById('inputField');
    const sendBtn = document.getElementById('sendBtn');
    const stopBtn = document.getElementById('stopBtn');
    const actionBtns = document.querySelectorAll('[data-action]');

    // Event listeners
    serverBtn.addEventListener('click', () => {
      if (state.serverRunning) {
        vscode.postMessage({ type: 'stopServer' });
      } else if (state.connected) {
        vscode.postMessage({ type: 'disconnect' });
      } else {
        vscode.postMessage({ type: 'startServer' });
      }
    });

    sendBtn.addEventListener('click', () => {
      const value = inputField.value.trim();
      if (value) {
        vscode.postMessage({ type: 'input', payload: value });
        inputField.value = '';
        historyIndex = -1;
      }
    });

    stopBtn.addEventListener('click', () => {
      vscode.postMessage({ type: 'stop' });
    });

    inputField.addEventListener('keydown', (e) => {
      if (e.key === 'Enter' && !e.shiftKey) {
        e.preventDefault();
        sendBtn.click();
      } else if (e.key === 'ArrowUp') {
        e.preventDefault();
        if (commandHistory.length > 0) {
          historyIndex = Math.min(historyIndex + 1, commandHistory.length - 1);
          inputField.value = commandHistory[commandHistory.length - 1 - historyIndex] || '';
        }
      } else if (e.key === 'ArrowDown') {
        e.preventDefault();
        if (historyIndex > 0) {
          historyIndex--;
          inputField.value = commandHistory[commandHistory.length - 1 - historyIndex] || '';
        } else {
          historyIndex = -1;
          inputField.value = '';
        }
      }
    });

    actionBtns.forEach(btn => {
      btn.addEventListener('click', () => {
        const action = btn.dataset.action;
        vscode.postMessage({ type: 'action', payload: action });
      });
    });

    // Message handler
    window.addEventListener('message', (event) => {
      const message = event.data;
      switch (message.type) {
        case 'state':
          state = message.payload;
          updateUI();
          break;
        case 'messages':
          messages = message.payload;
          renderMessages();
          break;
        case 'history':
          commandHistory = message.payload;
          break;
      }
    });

    function updateUI() {
      // Server button
      if (state.serverRunning) {
        serverBtn.textContent = 'Stop Server';
      } else if (state.connected) {
        serverBtn.textContent = 'Disconnect';
      } else {
        serverBtn.textContent = 'Start Server';
      }

      // Connection status
      if (state.connecting) {
        connectionStatus.textContent = 'Connecting...';
      } else if (state.connected) {
        connectionStatus.textContent = 'Connected';
      } else {
        connectionStatus.textContent = 'Disconnected';
      }

      // Task info
      if (state.connected && state.task) {
        taskInfo.style.display = 'flex';
        taskTitle.textContent = state.work?.title || state.task.id;
        stateBadge.textContent = state.workflowState;
        stateBadge.className = 'state-badge state-' + state.workflowState;
      } else if (state.connected) {
        taskInfo.style.display = 'flex';
        taskTitle.textContent = 'No active task';
        stateBadge.textContent = 'idle';
        stateBadge.className = 'state-badge state-idle';
      } else {
        taskInfo.style.display = 'none';
      }

      // Input
      inputField.disabled = !state.connected;
      sendBtn.disabled = !state.connected;
      stopBtn.disabled = !state.connected;

      // Action buttons
      actionBtns.forEach(btn => {
        btn.disabled = !state.connected;
      });

      // Disconnected notice
      disconnectedNotice.style.display = state.connected ? 'none' : 'block';
    }

    function renderMessages() {
      if (messages.length === 0) {
        return;
      }

      // Build messages safely using DOM methods
      messagesEl.textContent = ''; // Clear safely
      messages.forEach(msg => {
        const div = document.createElement('div');
        div.className = 'message message-' + msg.role;
        div.textContent = msg.content; // Safe: uses textContent
        messagesEl.appendChild(div);
      });
      messagesEl.scrollTop = messagesEl.scrollHeight;
    }

    // Initial ready signal
    vscode.postMessage({ type: 'ready' });
  </script>
</body>
</html>`;
  }

  dispose(): void {
    // Cleanup
  }
}
