import { EventSource } from 'eventsource';
import { EventEmitter } from 'events';
import type {
  SSEEventType,
  StateChangedEvent,
  AgentMessageEvent,
  ProgressEvent,
  ErrorEvent as MehrhofErrorEvent,
  QuestionEvent,
} from './models';

// eventsource v4 readyState constants
const ES_OPEN = 1;

export interface EventStreamOptions {
  reconnectDelayMs?: number;
  maxReconnectAttempts?: number;
}

const DEFAULT_RECONNECT_DELAY = 5000; // 5 seconds
const DEFAULT_MAX_RECONNECT_ATTEMPTS = 10;

export interface EventStreamClientEvents {
  connected: () => void;
  disconnected: (intentional: boolean) => void;
  error: (error: Error) => void;
  state_changed: (event: StateChangedEvent) => void;
  agent_message: (event: AgentMessageEvent) => void;
  progress: (event: ProgressEvent) => void;
  event_error: (event: MehrhofErrorEvent) => void;
  question: (event: QuestionEvent) => void;
  heartbeat: () => void;
  raw_event: (type: SSEEventType, data: unknown) => void;
}

export class EventStreamClient extends EventEmitter {
  private eventSource: EventSource | null = null;
  private readonly eventsUrl: string;
  private readonly reconnectDelay: number;
  private readonly maxReconnectAttempts: number;
  private reconnectAttempts = 0;
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null;
  private intentionalClose = false;
  private sessionCookie?: string;

  constructor(eventsUrl: string, options: EventStreamOptions = {}) {
    super();
    this.eventsUrl = eventsUrl;
    this.reconnectDelay = options.reconnectDelayMs ?? DEFAULT_RECONNECT_DELAY;
    this.maxReconnectAttempts = options.maxReconnectAttempts ?? DEFAULT_MAX_RECONNECT_ATTEMPTS;
  }

  setSessionCookie(cookie: string | undefined): void {
    this.sessionCookie = cookie;
  }

  connect(): void {
    if (this.eventSource) {
      return;
    }

    this.intentionalClose = false;
    this.createEventSource();
  }

  disconnect(): void {
    this.intentionalClose = true;
    this.cleanup();
    this.emit('disconnected', true);
  }

  isConnected(): boolean {
    return this.eventSource?.readyState === ES_OPEN;
  }

  private createEventSource(): void {
    // eventsource v4 uses custom fetch for headers
    const sessionCookie = this.sessionCookie;
    const customFetch: typeof fetch = (url, init) => {
      const headers = new Headers(init?.headers);
      if (sessionCookie) {
        headers.set('Cookie', sessionCookie);
      }
      return fetch(url, { ...init, headers });
    };

    this.eventSource = new EventSource(this.eventsUrl, {
      withCredentials: true,
      fetch: customFetch,
    });

    this.eventSource.onopen = () => {
      this.reconnectAttempts = 0;
      this.emit('connected');
    };

    this.eventSource.onerror = (error) => {
      if (this.intentionalClose) {
        return;
      }

      // Extract message from error event if available
      const errorWithMessage = error as { message?: string };
      this.emit('error', new Error(errorWithMessage.message ?? 'EventSource error'));
      this.handleDisconnect();
    };

    // Listen for all event types
    const eventTypes: SSEEventType[] = [
      'state_changed',
      'progress',
      'error',
      'file_changed',
      'agent_message',
      'checkpoint',
      'blueprint_ready',
      'branch_created',
      'plan_completed',
      'implement_done',
      'pr_created',
      'browser_action',
      'browser_tab_opened',
      'browser_screenshot',
      'sandbox_status_changed',
      'heartbeat',
    ];

    for (const eventType of eventTypes) {
      this.eventSource.addEventListener(eventType, (event) => {
        this.handleEvent(eventType, String(event.data));
      });
    }

    // Also listen for generic message events
    this.eventSource.onmessage = (event) => {
      // Try to parse as JSON to determine event type
      try {
        const rawData = String(event.data);
        const data = JSON.parse(rawData) as { type?: SSEEventType };
        if (data.type) {
          this.handleEvent(data.type, rawData);
        }
      } catch {
        // Ignore non-JSON messages (e.g., keepalive comments)
      }
    };
  }

  private handleEvent(type: SSEEventType, rawData: string): void {
    try {
      const data: unknown = rawData ? JSON.parse(rawData) : {};
      this.emit('raw_event', type, data);

      switch (type) {
        case 'state_changed':
          this.emit('state_changed', data as StateChangedEvent);
          break;
        case 'agent_message':
          this.emit('agent_message', data as AgentMessageEvent);
          break;
        case 'progress':
          this.emit('progress', data as ProgressEvent);
          break;
        case 'error':
          this.emit('event_error', data as MehrhofErrorEvent);
          break;
        case 'heartbeat':
          this.emit('heartbeat');
          break;
        default:
          // For other event types, just emit raw_event
          break;
      }
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : String(error);
      this.emit('error', new Error(`Failed to parse event data: ${errorMessage}`));
    }
  }

  private handleDisconnect(): void {
    this.cleanup();
    this.emit('disconnected', false);

    if (!this.intentionalClose && this.reconnectAttempts < this.maxReconnectAttempts) {
      this.scheduleReconnect();
    }
  }

  private scheduleReconnect(): void {
    if (this.reconnectTimer) {
      return;
    }

    this.reconnectAttempts++;
    const delay = this.reconnectDelay * Math.min(this.reconnectAttempts, 5); // Cap exponential backoff

    this.reconnectTimer = setTimeout(() => {
      this.reconnectTimer = null;
      if (!this.intentionalClose) {
        this.createEventSource();
      }
    }, delay);
  }

  private cleanup(): void {
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }

    if (this.eventSource) {
      this.eventSource.close();
      this.eventSource = null;
    }
  }

  // Type-safe event emitter overrides
  override on<K extends keyof EventStreamClientEvents>(
    event: K,
    listener: EventStreamClientEvents[K]
  ): this {
    return super.on(event, listener as (...args: unknown[]) => void);
  }

  override once<K extends keyof EventStreamClientEvents>(
    event: K,
    listener: EventStreamClientEvents[K]
  ): this {
    return super.once(event, listener as (...args: unknown[]) => void);
  }

  override off<K extends keyof EventStreamClientEvents>(
    event: K,
    listener: EventStreamClientEvents[K]
  ): this {
    return super.off(event, listener as (...args: unknown[]) => void);
  }

  override emit<K extends keyof EventStreamClientEvents>(
    event: K,
    ...args: Parameters<EventStreamClientEvents[K]>
  ): boolean {
    return super.emit(event, ...args);
  }
}
