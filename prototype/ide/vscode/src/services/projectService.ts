import * as vscode from 'vscode';
import { MehrhofApiClient, ApiError } from '../api/client';
import { EventStreamClient } from '../api/events';
import { ServerManager } from './serverManager';
import type {
  TaskInfo,
  TaskWork,
  PendingQuestion,
  StateChangedEvent,
  AgentMessageEvent,
} from '../api/models';

export type ConnectionState = 'disconnected' | 'connecting' | 'connected';

export interface ProjectServiceEvents {
  connectionChanged: (state: ConnectionState) => void;
  stateChanged: (event: StateChangedEvent) => void;
  taskChanged: (task: TaskInfo | null, work: TaskWork | null) => void;
  questionReceived: (question: PendingQuestion) => void;
  agentMessage: (event: AgentMessageEvent) => void;
  error: (error: Error) => void;
}

export class MehrhofProjectService implements vscode.Disposable {
  private readonly context: vscode.ExtensionContext;
  private readonly outputChannel: vscode.OutputChannel;
  private readonly serverManager: ServerManager;
  private apiClient: MehrhofApiClient | null = null;
  private eventStreamClient: EventStreamClient | null = null;

  private _connectionState: ConnectionState = 'disconnected';
  private _currentTask: TaskInfo | null = null;
  private _currentWork: TaskWork | null = null;
  private _workflowState: string = 'idle';
  private _pendingQuestion: PendingQuestion | null = null;

  private reconnectTimer: ReturnType<typeof setTimeout> | null = null;
  private reconnectAttempts = 0;

  private readonly listeners: Map<keyof ProjectServiceEvents, Set<(...args: unknown[]) => void>> =
    new Map();

  private readonly disposables: vscode.Disposable[] = [];

  constructor(context: vscode.ExtensionContext, outputChannel: vscode.OutputChannel) {
    this.context = context;
    this.outputChannel = outputChannel;
    this.serverManager = new ServerManager(outputChannel);

    // Listen for server events
    this.serverManager.on('started', (port) => {
      this.log(`Server started on port ${port}`);
      void this.connectToServer(`http://localhost:${port}`);
    });

    this.serverManager.on('stopped', () => {
      this.log('Server stopped');
      this.disconnect();
    });

    this.serverManager.on('error', (error) => {
      this.log(`Server error: ${error.message}`);
      this.emit('error', error);
    });

    // Listen for configuration changes
    this.disposables.push(
      vscode.workspace.onDidChangeConfiguration((e) => {
        if (e.affectsConfiguration('mehrhof')) {
          this.onConfigurationChanged();
        }
      })
    );
  }

  // Getters
  get connectionState(): ConnectionState {
    return this._connectionState;
  }

  get currentTask(): TaskInfo | null {
    return this._currentTask;
  }

  get currentWork(): TaskWork | null {
    return this._currentWork;
  }

  get workflowState(): string {
    return this._workflowState;
  }

  get pendingQuestion(): PendingQuestion | null {
    return this._pendingQuestion;
  }

  get isConnected(): boolean {
    return this._connectionState === 'connected';
  }

  get client(): MehrhofApiClient | null {
    return this.apiClient;
  }

  // Server management
  async startServer(): Promise<void> {
    const workspaceFolders = vscode.workspace.workspaceFolders;
    if (!workspaceFolders || workspaceFolders.length === 0) {
      throw new Error('No workspace folder open');
    }

    const workspacePath = workspaceFolders[0].uri.fsPath;
    await this.serverManager.start(workspacePath);
  }

  stopServer(): void {
    this.serverManager.stop();
  }

  isServerRunning(): boolean {
    return this.serverManager.isRunning();
  }

  // Connection management
  async connect(): Promise<void> {
    const config = vscode.workspace.getConfiguration('mehrhof');
    const serverUrl = config.get<string>('serverUrl');

    if (serverUrl) {
      await this.connectToServer(serverUrl);
    } else if (this.serverManager.isRunning()) {
      await this.connectToServer(`http://localhost:${this.serverManager.port}`);
    } else {
      await this.startServer();
    }
  }

  disconnect(): void {
    this.cancelReconnect();
    this.eventStreamClient?.disconnect();
    this.eventStreamClient = null;
    this.apiClient = null;
    this.setConnectionState('disconnected');
    this._currentTask = null;
    this._currentWork = null;
    this._workflowState = 'idle';
    this._pendingQuestion = null;
  }

  private async connectToServer(serverUrl: string): Promise<void> {
    this.setConnectionState('connecting');
    this.log(`Connecting to server at ${serverUrl}`);

    try {
      this.apiClient = new MehrhofApiClient(serverUrl);

      // Check if server is healthy
      const healthy = await this.apiClient.health();
      if (!healthy) {
        throw new Error('Server health check failed');
      }

      // Connect to event stream
      this.eventStreamClient = new EventStreamClient(this.apiClient.getEventsUrl(), {
        reconnectDelayMs: this.getConfig('reconnectDelaySeconds', 5) * 1000,
        maxReconnectAttempts: this.getConfig('maxReconnectAttempts', 10),
      });

      this.setupEventStreamListeners();
      this.eventStreamClient.connect();

      // Fetch initial state
      await this.refreshState();

      this.setConnectionState('connected');
      this.reconnectAttempts = 0;
      this.log('Connected to server');
    } catch (error) {
      this.log(`Connection failed: ${error instanceof Error ? error.message : 'Unknown error'}`);
      this.setConnectionState('disconnected');

      if (error instanceof Error) {
        this.emit('error', error);
      }

      // Schedule reconnect if auto-reconnect is enabled
      if (this.getConfig('autoReconnect', true)) {
        this.scheduleReconnect(serverUrl);
      }
    }
  }

  private setupEventStreamListeners(): void {
    if (!this.eventStreamClient) {
      return;
    }

    this.eventStreamClient.on('connected', () => {
      this.log('Event stream connected');
    });

    this.eventStreamClient.on('disconnected', (intentional) => {
      if (!intentional && this._connectionState === 'connected') {
        this.log('Event stream disconnected unexpectedly');
        this.setConnectionState('disconnected');

        if (this.getConfig('autoReconnect', true) && this.apiClient) {
          this.scheduleReconnect(this.apiClient.getBaseUrl());
        }
      }
    });

    this.eventStreamClient.on('error', (error) => {
      this.log(`Event stream error: ${error.message}`);
    });

    this.eventStreamClient.on('state_changed', (event) => {
      this._workflowState = event.to;
      this.emit('stateChanged', event);
      // Refresh task state after state change
      void this.refreshState();
    });

    this.eventStreamClient.on('agent_message', (event) => {
      this.emit('agentMessage', event);
    });

    this.eventStreamClient.on('event_error', (event) => {
      this.emit('error', new Error(event.error));
    });
  }

  async refreshState(): Promise<void> {
    if (!this.apiClient) {
      return;
    }

    try {
      const response = await this.apiClient.getTask();
      this._currentTask = response.task ?? null;
      this._currentWork = response.work ?? null;
      this._pendingQuestion = response.pending_question ?? null;

      if (response.task) {
        this._workflowState = response.task.state;
      }

      this.emit('taskChanged', this._currentTask, this._currentWork);

      if (this._pendingQuestion) {
        this.emit('questionReceived', this._pendingQuestion);
      }
    } catch (error) {
      if (error instanceof ApiError) {
        this.log(`Failed to refresh state: ${error.message}`);
      }
    }
  }

  private scheduleReconnect(serverUrl: string): void {
    const maxAttempts = this.getConfig('maxReconnectAttempts', 10);
    if (this.reconnectAttempts >= maxAttempts) {
      this.log(`Max reconnection attempts (${maxAttempts}) reached`);
      return;
    }

    const delay = this.getConfig('reconnectDelaySeconds', 5) * 1000;
    const backoffDelay = delay * Math.min(this.reconnectAttempts + 1, 5);

    this.log(
      `Scheduling reconnect in ${backoffDelay / 1000}s (attempt ${this.reconnectAttempts + 1}/${maxAttempts})`
    );

    this.reconnectTimer = setTimeout(() => {
      this.reconnectAttempts++;
      void this.connectToServer(serverUrl);
    }, backoffDelay);
  }

  private cancelReconnect(): void {
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }
    this.reconnectAttempts = 0;
  }

  private setConnectionState(state: ConnectionState): void {
    if (this._connectionState !== state) {
      this._connectionState = state;
      this.emit('connectionChanged', state);
    }
  }

  private onConfigurationChanged(): void {
    // Could reconnect with new settings if needed
    this.log('Configuration changed');
  }

  private getConfig<T>(key: string, defaultValue: T): T {
    const config = vscode.workspace.getConfiguration('mehrhof');
    return config.get<T>(key, defaultValue);
  }

  private log(message: string): void {
    const timestamp = new Date().toISOString();
    this.outputChannel.appendLine(`[${timestamp}] ${message}`);
  }

  // Event emitter methods
  on<K extends keyof ProjectServiceEvents>(event: K, listener: ProjectServiceEvents[K]): this {
    if (!this.listeners.has(event)) {
      this.listeners.set(event, new Set());
    }
    this.listeners.get(event)!.add(listener as (...args: unknown[]) => void);
    return this;
  }

  off<K extends keyof ProjectServiceEvents>(event: K, listener: ProjectServiceEvents[K]): this {
    this.listeners.get(event)?.delete(listener as (...args: unknown[]) => void);
    return this;
  }

  private emit<K extends keyof ProjectServiceEvents>(
    event: K,
    ...args: Parameters<ProjectServiceEvents[K]>
  ): void {
    const listeners = this.listeners.get(event);
    if (listeners) {
      for (const listener of listeners) {
        listener(...args);
      }
    }
  }

  dispose(): void {
    this.disconnect();
    this.serverManager.dispose();
    for (const d of this.disposables) {
      d.dispose();
    }
    this.listeners.clear();
  }
}
