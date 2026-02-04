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
// Error Response
// ============================================================================

export interface ErrorResponse {
  error: string;
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
