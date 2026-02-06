// ============================================================================
// Status & Task Models
// ============================================================================

export interface StatusResponse {
  mode: string;
  running: boolean;
  port: number;
  state?: string;
}

export interface TaskResponse {
  active: boolean;
  task?: TaskInfo;
  work?: TaskWork;
  pending_question?: PendingQuestion;
}

export interface TaskInfo {
  id: string;
  state: string;
  ref: string;
  branch?: string;
  worktree_path?: string;
  started?: string;
}

export interface TaskWork {
  title?: string;
  external_key?: string;
  created_at?: string;
  updated_at?: string;
  costs?: CostInfo;
}

export interface PendingQuestion {
  question: string;
  options?: string[];
}

export interface TaskListResponse {
  tasks: TaskSummary[];
  count: number;
}

export interface TaskSummary {
  id: string;
  title?: string;
  state: string;
  created_at?: string;
  worktree_path?: string;
}

// ============================================================================
// Workflow Models
// ============================================================================

export interface WorkflowResponse {
  success: boolean;
  state?: string;
  message?: string;
  error?: string;
}

export interface ContinueResponse {
  success: boolean;
  state: string;
  action?: string;
  next_actions: string[];
  message: string;
}

export interface WorkflowRequest {
  agent?: string;
}

export interface FinishRequest {
  squash_merge?: boolean;
  delete_branch?: boolean;
  target_branch?: string;
  push_after?: boolean;
  force_merge?: boolean;
  draft_pr?: boolean;
  pr_title?: string;
  pr_body?: string;
}

export interface AnswerRequest {
  answer: string;
}

export interface AddNoteRequest {
  message: string;
}

export interface AddNoteResponse {
  success: boolean;
  note_number?: number;
  error?: string;
}

export interface QuestionRequest {
  message: string;
}

export interface StartTaskRequest {
  ref?: string;
  content?: string;
}

// ============================================================================
// Guide Models
// ============================================================================

export interface GuideResponse {
  has_task: boolean;
  task_id?: string;
  title?: string;
  state?: string;
  specifications: number;
  pending_question?: PendingQuestionInfo;
  next_actions: GuideAction[];
}

export interface PendingQuestionInfo {
  question: string;
  options?: string[];
}

export interface GuideAction {
  command: string;
  description: string;
  endpoint?: string;
}

// ============================================================================
// Cost Models
// ============================================================================

export interface CostInfo {
  total_tokens: number;
  input_tokens: number;
  output_tokens: number;
  cached_tokens: number;
  total_cost_usd: number;
}

export interface TaskCostResponse {
  task_id: string;
  title?: string;
  total_tokens: number;
  input_tokens: number;
  output_tokens: number;
  cached_tokens: number;
  cached_percent?: number;
  total_cost_usd: number;
  by_step?: Record<string, StepCost>;
}

export interface StepCost {
  input_tokens: number;
  output_tokens: number;
  cached_tokens: number;
  total_tokens: number;
  cost_usd: number;
  calls: number;
}

export interface AllCostsResponse {
  tasks: TaskCostResponse[];
  grand_total: GrandTotal;
  monthly?: MonthlyBudgetInfo;
}

export interface GrandTotal {
  input_tokens: number;
  output_tokens: number;
  total_tokens: number;
  cached_tokens: number;
  cost_usd: number;
}

export interface MonthlyBudgetInfo {
  month: string;
  spent: number;
  max_cost?: number;
  warning_at?: number;
  warning_sent?: boolean;
}

// ============================================================================
// Specification Models
// ============================================================================

export interface SpecificationsResponse {
  specifications: Specification[];
}

export interface Specification {
  id: number;
  title: string;
  content: string;
  created_at?: string;
  status?: string;
  implemented_files?: string[];
}

// ============================================================================
// Session Models
// ============================================================================

export interface SessionsResponse {
  sessions: Session[];
}

export interface Session {
  id: string;
  step: string;
  started_at: string;
  ended_at?: string;
  status?: string;
}

// ============================================================================
// Agent & Provider Models
// ============================================================================

export interface AgentsListResponse {
  agents: AgentInfo[];
  count: number;
}

export interface AgentInfo {
  name: string;
  type: string;
  extends?: string;
  description?: string;
  version?: string;
  available: boolean;
  capabilities?: AgentCapabilities;
  models?: AgentModel[];
}

export interface AgentCapabilities {
  streaming: boolean;
  tool_use: boolean;
  file_operations: boolean;
  code_execution: boolean;
  multi_turn: boolean;
  system_prompt: boolean;
  allowed_tools?: string[];
}

export interface AgentModel {
  id: string;
  name: string;
  default: boolean;
  max_tokens?: number;
  input_cost_usd?: number;
  output_cost_usd?: number;
}

export interface ProvidersListResponse {
  providers: ProviderInfo[];
  count: number;
}

export interface ProviderInfo {
  scheme: string;
  shorthand?: string;
  name: string;
  description: string;
  env_vars?: string[];
}

// ============================================================================
// Interactive API Models
// ============================================================================

export interface InteractiveCommandRequest {
  command: string;
  args?: string[];
}

export interface InteractiveCommandResponse {
  success: boolean;
  message?: string;
  state?: string;
  error?: string;
}

export interface InteractiveChatRequest {
  message: string;
}

export interface InteractiveChatMessage {
  role: string;
  content: string;
  timestamp?: string;
}

export interface InteractiveChatResponse {
  success: boolean;
  message?: string;
  messages?: InteractiveChatMessage[];
  error?: string;
}

export interface InteractiveAnswerRequest {
  response: string;
}

export interface InteractiveStateResponse {
  success: boolean;
  state?: string;
  task_id?: string;
  title?: string;
  error?: string;
}

export interface InteractiveStopResponse {
  success: boolean;
  message?: string;
  error?: string;
}

// ============================================================================
// Command Discovery Models
// ============================================================================

export interface CommandArg {
  name: string;
  required: boolean;
  description?: string;
}

export interface CommandInfo {
  name: string;
  aliases?: string[];
  description: string;
  category: string;
  args?: CommandArg[];
  requires_task: boolean;
  subcommands?: string[];
}

export interface CommandsResponse {
  commands: CommandInfo[];
}

// ============================================================================
// Error Response
// ============================================================================

export interface ErrorResponse {
  error: string;
}

// ============================================================================
// Queue Task Models (for quick tasks)
// ============================================================================

export interface DeleteQueueTaskResponse {
  success: boolean;
  message?: string;
  error?: string;
}

export interface ExportQueueTaskResponse {
  success: boolean;
  message?: string;
  markdown?: string;
  error?: string;
}

export interface OptimizeQueueTaskResponse {
  success: boolean;
  message?: string;
  original_title?: string;
  optimized_title?: string;
  added_labels?: string[];
  improvement_notes?: string[];
  error?: string;
}

export interface SubmitQueueTaskRequest {
  provider: string;
}

export interface SubmitQueueTaskResponse {
  success: boolean;
  message?: string;
  external_id?: string;
  url?: string;
  error?: string;
}

export interface SyncTaskResponse {
  success: boolean;
  message?: string;
  error?: string;
}

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
// Browser Models
// ============================================================================

export interface BrowserStatusResponse {
  connected: boolean;
  host?: string;
  port?: number;
  tabs?: BrowserTab[];
  error?: string;
}

export interface BrowserTab {
  id: string;
  title: string;
  url: string;
}

export interface BrowserTabsResponse {
  tabs: BrowserTab[];
  count: number;
}

export interface BrowserGotoRequest {
  url: string;
}

export interface BrowserGotoResponse {
  success: boolean;
  tab?: BrowserTab;
}

export interface BrowserNavigateRequest {
  tab_id?: string;
  url: string;
}

export interface BrowserNavigateResponse {
  success: boolean;
  message?: string;
}

export interface BrowserClickRequest {
  tab_id?: string;
  selector: string;
}

export interface BrowserClickResponse {
  success: boolean;
  selector?: string;
}

export interface BrowserTypeRequest {
  tab_id?: string;
  selector: string;
  text: string;
  clear?: boolean;
}

export interface BrowserTypeResponse {
  success: boolean;
  selector?: string;
}

export interface BrowserEvalRequest {
  tab_id?: string;
  expression: string;
}

export interface BrowserEvalResponse {
  success: boolean;
  result?: unknown;
}

export interface BrowserDOMRequest {
  tab_id?: string;
  selector: string;
  all?: boolean;
  html?: boolean;
  limit?: number;
}

export interface BrowserDOMElement {
  tag_name: string;
  text_content?: string;
  outer_html?: string;
  visible: boolean;
}

export interface BrowserDOMResponse {
  success: boolean;
  element?: BrowserDOMElement;
  elements?: BrowserDOMElement[];
  count?: number;
  showing?: number;
}

export interface BrowserScreenshotRequest {
  tab_id?: string;
  format?: string;
  quality?: number;
  full_page?: boolean;
}

export interface BrowserScreenshotResponse {
  success: boolean;
  format?: string;
  data?: string;
  size?: number;
  encoding?: string;
}

export interface BrowserReloadRequest {
  tab_id?: string;
  hard?: boolean;
}

export interface BrowserReloadResponse {
  success: boolean;
  message?: string;
}

export interface BrowserCloseRequest {
  tab_id: string;
}

export interface BrowserCloseResponse {
  success: boolean;
  message?: string;
}

export interface BrowserConsoleRequest {
  tab_id?: string;
  duration?: number;
  level?: string;
}

export interface BrowserConsoleMessage {
  level: string;
  text: string;
  timestamp?: string;
}

export interface BrowserConsoleResponse {
  success: boolean;
  messages?: BrowserConsoleMessage[];
  count?: number;
}

export interface BrowserNetworkRequest {
  tab_id?: string;
  duration?: number;
  capture_body?: boolean;
  max_body_size?: number;
}

export interface BrowserNetworkEntry {
  method: string;
  url: string;
  status?: number;
  status_text?: string;
  timestamp: string;
  request_body?: string;
  response_body?: string;
}

export interface BrowserNetworkResponse {
  success: boolean;
  requests?: BrowserNetworkEntry[];
  count?: number;
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
