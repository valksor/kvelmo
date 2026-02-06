import * as vscode from 'vscode';

// Note: escapeHtml function below uses textContent to safely escape HTML
// before using innerHTML - this is a safe pattern that prevents XSS
export function getInteractivePanelHtml(
  _webview: vscode.Webview,
  _extensionUri: vscode.Uri
): string {
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
    <div class="actions" style="margin-top: 4px;">
      <button data-action="status" class="secondary" disabled>Status</button>
      <button data-action="cost" class="secondary" disabled>Cost</button>
      <button data-action="budget" class="secondary" disabled>Budget</button>
      <button data-action="specification" class="secondary" disabled>Specs</button>
      <button data-action="find" class="secondary" disabled>Find</button>
      <button data-action="memory" class="secondary" disabled>Memory</button>
      <button data-action="library" class="secondary" disabled>Library</button>
      <button data-action="quick" class="secondary" disabled>Quick Task</button>
      <button data-action="simplify" class="secondary" disabled>Simplify</button>
      <button data-action="note" class="secondary" disabled>Add Note</button>
      <button data-action="list" class="secondary" disabled>List Tasks</button>
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
