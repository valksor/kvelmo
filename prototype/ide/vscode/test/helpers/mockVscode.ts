/**
 * Mock implementations for VS Code API used in unit tests.
 * These mocks provide controllable test doubles for vscode module dependencies.
 */

// EventEmitter from events is not needed - we have our own MockEventEmitter

// Track all registered commands for verification
export const registeredCommands: Map<string, (...args: unknown[]) => unknown> = new Map();

// Track all disposables for cleanup verification
export const disposables: { dispose: () => void }[] = [];

// Mock OutputChannel
export interface MockOutputChannel {
  name: string;
  lines: string[];
  appendLine(value: string): void;
  append(value: string): void;
  clear(): void;
  show(preserveFocus?: boolean): void;
  hide(): void;
  dispose(): void;
}

export function createMockOutputChannel(name: string): MockOutputChannel {
  const channel: MockOutputChannel = {
    name,
    lines: [],
    appendLine(value: string) {
      this.lines.push(value);
    },
    append(value: string) {
      if (this.lines.length === 0) {
        this.lines.push(value);
      } else {
        this.lines[this.lines.length - 1] += value;
      }
    },
    clear() {
      this.lines = [];
    },
    show() {},
    hide() {},
    dispose() {},
  };
  return channel;
}

// Mock StatusBarItem
export interface MockStatusBarItem {
  alignment: number;
  priority: number;
  text: string;
  tooltip: string | undefined;
  color: string | undefined;
  backgroundColor: { id: string } | undefined;
  command: string | undefined;
  accessibilityInformation: unknown;
  name: string | undefined;
  show(): void;
  hide(): void;
  dispose(): void;
}

export function createMockStatusBarItem(
  alignment: number = 1,
  priority: number = 0
): MockStatusBarItem {
  return {
    alignment,
    priority,
    text: '',
    tooltip: undefined,
    color: undefined,
    backgroundColor: undefined,
    command: undefined,
    accessibilityInformation: undefined,
    name: undefined,
    show() {},
    hide() {},
    dispose() {},
  };
}

// Mock TreeDataProvider change event
export class MockEventEmitter<T> {
  private listeners: ((e: T) => void)[] = [];

  event = (listener: (e: T) => void) => {
    this.listeners.push(listener);
    return { dispose: () => this.listeners.splice(this.listeners.indexOf(listener), 1) };
  };

  fire(data: T) {
    this.listeners.forEach((l) => l(data));
  }

  dispose() {
    this.listeners = [];
  }
}

// Mock Configuration
export interface MockConfiguration {
  values: Map<string, unknown>;
  get<T>(section: string, defaultValue?: T): T | undefined;
  has(section: string): boolean;
  inspect<T>(
    section: string
  ): { defaultValue?: T; globalValue?: T; workspaceValue?: T } | undefined;
  update(section: string, value: unknown): Promise<void>;
}

export function createMockConfiguration(
  initialValues: Record<string, unknown> = {}
): MockConfiguration {
  const values = new Map(Object.entries(initialValues));
  return {
    values,
    get<T>(section: string, defaultValue?: T): T | undefined {
      if (values.has(section)) {
        return values.get(section) as T;
      }
      return defaultValue;
    },
    has(section: string): boolean {
      return values.has(section);
    },
    inspect<T>(_section: string): { defaultValue?: T } | undefined {
      return undefined;
    },
    update(section: string, value: unknown): Promise<void> {
      values.set(section, value);
      return Promise.resolve();
    },
  };
}

// Mock WebviewView
export interface MockWebviewView {
  webview: {
    options: { enableScripts?: boolean; localResourceRoots?: unknown[] };
    html: string;
    onDidReceiveMessage: (handler: (message: unknown) => void) => { dispose: () => void };
    postMessage: (message: unknown) => Thenable<boolean>;
    asWebviewUri: (uri: unknown) => unknown;
    cspSource: string;
    _messageHandler?: (message: unknown) => void;
  };
  visible: boolean;
  onDidDispose: (handler: () => void) => { dispose: () => void };
  onDidChangeVisibility: (handler: () => void) => { dispose: () => void };
  show(preserveFocus?: boolean): void;
  _disposeHandler?: () => void;
}

export function createMockWebviewView(): MockWebviewView {
  const view: MockWebviewView = {
    webview: {
      options: {},
      html: '',
      onDidReceiveMessage: (handler: (message: unknown) => void) => {
        view.webview._messageHandler = handler;
        return { dispose: () => {} };
      },
      postMessage: () => Promise.resolve(true),
      asWebviewUri: (uri: unknown) => uri,
      cspSource: 'mock-csp-source',
    },
    visible: true,
    onDidDispose: (handler: () => void) => {
      view._disposeHandler = handler;
      return { dispose: () => {} };
    },
    onDidChangeVisibility: () => ({ dispose: () => {} }),
    show() {},
  };
  return view;
}

// Mock ExtensionContext
export interface MockExtensionContext {
  subscriptions: { dispose: () => void }[];
  workspaceState: {
    get: <T>(key: string) => T | undefined;
    update: (key: string, value: unknown) => Promise<void>;
  };
  globalState: {
    get: <T>(key: string) => T | undefined;
    update: (key: string, value: unknown) => Promise<void>;
  };
  extensionPath: string;
  extensionUri: { fsPath: string; scheme: string; path: string };
  storagePath: string | undefined;
  globalStoragePath: string;
  logPath: string;
  asAbsolutePath(relativePath: string): string;
  extensionMode: number;
}

export function createMockExtensionContext(): MockExtensionContext {
  const workspaceStorage = new Map<string, unknown>();
  const globalStorage = new Map<string, unknown>();

  return {
    subscriptions: [],
    workspaceState: {
      get: <T>(key: string) => workspaceStorage.get(key) as T | undefined,
      update: (key: string, value: unknown) => {
        workspaceStorage.set(key, value);
        return Promise.resolve();
      },
    },
    globalState: {
      get: <T>(key: string) => globalStorage.get(key) as T | undefined,
      update: (key: string, value: unknown) => {
        globalStorage.set(key, value);
        return Promise.resolve();
      },
    },
    extensionPath: '/mock/extension/path',
    extensionUri: {
      fsPath: '/mock/extension/path',
      scheme: 'file',
      path: '/mock/extension/path',
    },
    storagePath: '/mock/storage/path',
    globalStoragePath: '/mock/global/storage/path',
    logPath: '/mock/log/path',
    asAbsolutePath(relativePath: string) {
      return `/mock/extension/path/${relativePath}`;
    },
    extensionMode: 1, // Development
  };
}

// Mock workspace
export interface MockWorkspace {
  workspaceFolders: { uri: { fsPath: string }; name: string; index: number }[] | undefined;
  getConfiguration(section?: string): MockConfiguration;
  onDidChangeConfiguration: (
    handler: (e: { affectsConfiguration: (section: string) => boolean }) => void
  ) => { dispose: () => void };
  _configChangeHandler?: (e: { affectsConfiguration: (section: string) => boolean }) => void;
  _configurations: Map<string, MockConfiguration>;
}

export function createMockWorkspace(
  folders: { path: string; name?: string }[] = []
): MockWorkspace {
  const configurations = new Map<string, MockConfiguration>();
  configurations.set('mehrhof', createMockConfiguration());

  const workspace: MockWorkspace = {
    workspaceFolders:
      folders.length > 0
        ? folders.map((f, i) => ({
            uri: { fsPath: f.path },
            name: f.name ?? `folder-${i}`,
            index: i,
          }))
        : undefined,
    getConfiguration(section?: string): MockConfiguration {
      if (!section) {
        return createMockConfiguration();
      }
      if (!configurations.has(section)) {
        configurations.set(section, createMockConfiguration());
      }
      return configurations.get(section)!;
    },
    onDidChangeConfiguration: (handler) => {
      workspace._configChangeHandler = handler;
      return { dispose: () => {} };
    },
    _configurations: configurations,
  };
  return workspace;
}

// Mock window
export interface MockWindow {
  showInformationMessage: (...args: unknown[]) => Promise<string | undefined>;
  showWarningMessage: (...args: unknown[]) => Promise<string | undefined>;
  showErrorMessage: (...args: unknown[]) => Promise<string | undefined>;
  showQuickPick: (items: unknown[], options?: unknown) => Promise<unknown>;
  showInputBox: (options?: unknown) => Promise<string | undefined>;
  createOutputChannel: (name: string) => MockOutputChannel;
  createStatusBarItem: (alignment?: number, priority?: number) => MockStatusBarItem;
  registerTreeDataProvider: (viewId: string, provider: unknown) => { dispose: () => void };
  registerWebviewViewProvider: (viewId: string, provider: unknown) => { dispose: () => void };
  withProgress: <T>(
    options: unknown,
    task: (progress: unknown, token: unknown) => Promise<T>
  ) => Promise<T>;
  _outputChannels: MockOutputChannel[];
  _statusBarItems: MockStatusBarItem[];
}

export function createMockWindow(): MockWindow {
  const outputChannels: MockOutputChannel[] = [];
  const statusBarItems: MockStatusBarItem[] = [];

  return {
    showInformationMessage: () => Promise.resolve(undefined),
    showWarningMessage: () => Promise.resolve(undefined),
    showErrorMessage: () => Promise.resolve(undefined),
    showQuickPick: () => Promise.resolve(undefined),
    showInputBox: () => Promise.resolve(undefined),
    createOutputChannel: (name: string) => {
      const channel = createMockOutputChannel(name);
      outputChannels.push(channel);
      return channel;
    },
    createStatusBarItem: (alignment?: number, priority?: number) => {
      const item = createMockStatusBarItem(alignment, priority);
      statusBarItems.push(item);
      return item;
    },
    registerTreeDataProvider: () => ({ dispose: () => {} }),
    registerWebviewViewProvider: () => ({ dispose: () => {} }),
    withProgress: async <T>(
      _options: unknown,
      task: (progress: unknown, token: unknown) => Promise<T>
    ): Promise<T> => {
      const progress = { report: () => {} };
      const token = {
        isCancellationRequested: false,
        onCancellationRequested: () => ({ dispose: () => {} }),
      };
      return task(progress, token);
    },
    _outputChannels: outputChannels,
    _statusBarItems: statusBarItems,
  };
}

// Mock commands
export interface MockCommands {
  registerCommand: (
    command: string,
    callback: (...args: unknown[]) => unknown
  ) => { dispose: () => void };
  executeCommand: <T>(command: string, ...rest: unknown[]) => Promise<T | undefined>;
  getCommands: (filterInternal?: boolean) => Promise<string[]>;
}

export function createMockCommands(): MockCommands {
  return {
    registerCommand: (command: string, callback: (...args: unknown[]) => unknown) => {
      registeredCommands.set(command, callback);
      const disposable = {
        dispose: () => {
          registeredCommands.delete(command);
        },
      };
      disposables.push(disposable);
      return disposable;
    },
    executeCommand: <T>(command: string, ...rest: unknown[]): Promise<T | undefined> => {
      const handler = registeredCommands.get(command);
      if (handler) {
        return Promise.resolve(handler(...rest) as T);
      }
      return Promise.resolve(undefined);
    },
    getCommands: () => Promise.resolve(Array.from(registeredCommands.keys())),
  };
}

// Mock Uri
export const Uri = {
  file: (path: string) => ({ fsPath: path, scheme: 'file', path }),
  parse: (value: string) => ({ fsPath: value, scheme: 'file', path: value }),
  joinPath: (base: { path: string }, ...pathSegments: string[]) => ({
    fsPath: [base.path, ...pathSegments].join('/'),
    scheme: 'file',
    path: [base.path, ...pathSegments].join('/'),
  }),
};

// Mock ThemeIcon
export class ThemeIcon {
  constructor(
    public readonly id: string,
    public readonly color?: { id: string }
  ) {}
}

// Mock ThemeColor
export class ThemeColor {
  constructor(public readonly id: string) {}
}

// Mock TreeItem
export class TreeItem {
  label?: string;
  description?: string;
  tooltip?: string;
  iconPath?: ThemeIcon | { light: string; dark: string };
  contextValue?: string;
  command?: { command: string; title: string; arguments?: unknown[] };
  collapsibleState?: number;

  constructor(label: string, collapsibleState?: number) {
    this.label = label;
    this.collapsibleState = collapsibleState;
  }
}

// TreeItemCollapsibleState enum
export const TreeItemCollapsibleState = {
  None: 0,
  Collapsed: 1,
  Expanded: 2,
};

// StatusBarAlignment enum
export const StatusBarAlignment = {
  Left: 1,
  Right: 2,
};

// ProgressLocation enum
export const ProgressLocation = {
  SourceControl: 1,
  Window: 10,
  Notification: 15,
};

// Reset all mocks (call in teardown)
export function resetMocks(): void {
  registeredCommands.clear();
  disposables.length = 0;
}

// Complete mock vscode module
export function createMockVscode(workspaceFolders: { path: string; name?: string }[] = []): {
  window: MockWindow;
  workspace: MockWorkspace;
  commands: MockCommands;
  Uri: typeof Uri;
  ThemeIcon: typeof ThemeIcon;
  ThemeColor: typeof ThemeColor;
  TreeItem: typeof TreeItem;
  TreeItemCollapsibleState: typeof TreeItemCollapsibleState;
  StatusBarAlignment: typeof StatusBarAlignment;
  ProgressLocation: typeof ProgressLocation;
  EventEmitter: typeof MockEventEmitter;
} {
  return {
    window: createMockWindow(),
    workspace: createMockWorkspace(workspaceFolders),
    commands: createMockCommands(),
    Uri,
    ThemeIcon,
    ThemeColor,
    TreeItem,
    TreeItemCollapsibleState,
    StatusBarAlignment,
    ProgressLocation,
    EventEmitter: MockEventEmitter,
  };
}
