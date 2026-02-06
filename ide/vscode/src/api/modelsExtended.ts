// ============================================================================
// Find Search Models
// ============================================================================

export interface FindSearchResponse {
  query: string;
  count: number;
  matches: FindMatch[];
}

export interface FindMatch {
  file: string;
  line: number;
  snippet: string;
  context?: string;
  reason?: string;
}

// ============================================================================
// Memory Models
// ============================================================================

export interface MemorySearchResponse {
  results: MemoryResult[];
  count: number;
}

export interface MemoryResult {
  task_id: string;
  type: string;
  score: number;
  content: string;
  metadata?: Record<string, unknown>;
}

export interface MemoryIndexResponse {
  success: boolean;
  message?: string;
  task_id?: string;
  error?: string;
}

export interface MemoryStatsResponse {
  total_documents: number;
  by_type: Record<string, number>;
  enabled: boolean;
}

// ============================================================================
// Library Models
// ============================================================================

export interface LibraryListResponse {
  collections: LibraryCollection[];
  count: number;
}

export interface LibraryCollection {
  id: string;
  name: string;
  source: string;
  source_type: string;
  include_mode: string;
  page_count: number;
  total_size: number;
  location: string;
  pulled_at?: string;
  tags?: string[];
  paths?: string[];
}

export interface LibraryShowResponse {
  collection: LibraryCollection;
  pages: string[];
}

export interface LibraryStatsResponse {
  total_collections: number;
  total_pages: number;
  total_size: number;
  project_count: number;
  shared_count: number;
  by_mode: Record<string, number>;
  enabled: boolean;
}

export interface LibraryPullResponse {
  success: boolean;
  message?: string;
  collection?: LibraryCollection;
  error?: string;
}

export interface LibraryRemoveResponse {
  success: boolean;
  message?: string;
  error?: string;
}

// ============================================================================
// Links Models
// ============================================================================

export interface LinksListResponse {
  links: LinkData[];
  count: number;
}

export interface LinkData {
  source: string;
  target: string;
  context: string;
  created_at: string;
}

export interface EntityLinksResponse {
  entity_id: string;
  outgoing: LinkData[];
  incoming: LinkData[];
}

export interface LinksSearchResponse {
  query: string;
  results: EntityResult[];
  count: number;
}

export interface EntityResult {
  entity_id: string;
  type: string;
  name?: string;
  task_id?: string;
  id?: string;
  full_type?: string;
  total_links?: number;
}

export interface LinksStatsResponse {
  total_links: number;
  total_sources: number;
  total_targets: number;
  orphan_entities: number;
  most_linked: EntityResult[];
  enabled: boolean;
}

export interface LinksRebuildResponse {
  success: boolean;
  message?: string;
  total_links?: number;
  total_sources?: number;
  total_targets?: number;
  error?: string;
}

// ============================================================================
// Project Models
// ============================================================================

export interface ProjectPlanRequest {
  source: string;
  title?: string;
  instructions?: string;
}

export interface ProjectPlanResponse {
  success: boolean;
  queue_id?: string;
  task_count?: number;
  questions?: string[];
  error?: string;
}

export interface ProjectTasksResponse {
  queue_id: string;
  tasks: ProjectQueueTask[];
  count: number;
}

export interface ProjectQueueTask {
  id: string;
  title: string;
  status: string;
  priority: number;
  parent_id?: string;
  depends_on?: string[];
}

export interface ProjectSubmitRequest {
  provider: string;
  queue_id?: string;
  create_epic?: boolean;
  labels?: string[];
}

export interface ProjectSubmitResponse {
  success: boolean;
  submitted_count?: number;
  tasks?: ProjectSubmittedTask[];
  error?: string;
}

export interface ProjectSubmittedTask {
  local_id: string;
  external_id: string;
  title: string;
}

export interface ProjectSyncRequest {
  reference: string;
}

export interface ProjectSyncResponse {
  success: boolean;
  queue_id?: string;
  queue_title?: string;
  tasks_synced?: number;
  error?: string;
}

// ============================================================================
// Stack Models
// ============================================================================

export interface StackListResponse {
  stacks: StackInfo[];
  count: number;
}

export interface StackInfo {
  id: string;
  task_count: number;
  tasks: StackTask[];
}

export interface StackTask {
  id: string;
  branch: string;
  state: string;
  depends_on?: string;
  pr_number?: number;
}

export interface StackRebaseResponse {
  success: boolean;
  rebased_count?: number;
  rebased_tasks?: string[];
  error?: string;
}

export interface StackSyncResponse {
  success: boolean;
  updated_count?: number;
  error?: string;
}

// ============================================================================
// SSE Event Types
// ============================================================================

export type SSEEventType =
  | 'state_changed'
  | 'progress'
  | 'error'
  | 'file_changed'
  | 'agent_message'
  | 'checkpoint'
  | 'blueprint_ready'
  | 'branch_created'
  | 'plan_completed'
  | 'implement_done'
  | 'pr_created'
  | 'browser_action'
  | 'browser_tab_opened'
  | 'browser_screenshot'
  | 'sandbox_status_changed'
  | 'heartbeat';

export interface SSEEvent {
  type: SSEEventType;
  data: unknown;
  timestamp?: string;
}

export interface StateChangedEvent {
  from: string;
  to: string;
  event: string;
  task_id?: string;
  timestamp: string;
}

export interface AgentMessageEvent {
  task_id?: string;
  content: string;
  role: 'assistant' | 'tool' | 'system';
  timestamp: string;
}

export interface ProgressEvent {
  task_id?: string;
  phase: string;
  message: string;
  current?: number;
  total?: number;
  timestamp: string;
}

export interface ErrorEvent {
  task_id?: string;
  error: string;
  fatal: boolean;
  timestamp: string;
}

export interface QuestionEvent {
  task_id?: string;
  question: string;
  options?: string[];
  timestamp: string;
}

export interface HeartbeatEvent {
  state?: string;
  state_changed?: boolean;
  task_id?: string;
  specs?: number;
  checkpoints?: number;
  agent?: string;
  timestamp: number;
}
