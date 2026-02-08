/**
 * API response types for go-mehrhof backend
 * Matches internal/storage/workspace_config.go
 */

// Workflow states
export type WorkflowState =
  | 'idle'
  | 'planning'
  | 'implementing'
  | 'reviewing'
  | 'waiting'
  | 'checkpointing'
  | 'reverting'
  | 'restoring'
  | 'done'
  | 'failed'

// /api/v1/status response
export interface StatusProject {
  id?: string
  name: string
  path: string
  remote_url?: string
}

export interface StatusResponse {
  mode: string
  running: boolean
  port: number
  state?: WorkflowState
  canSwitchToGlobal?: boolean
  project?: StatusProject
}

// Progress phases for context-aware state display (matches internal/display/display.go)
export type ProgressPhase = 'started' | 'planned' | 'implemented' | 'reviewed'

// /api/v1/task response
export interface TaskResponse {
  active: boolean
  task?: {
    id: string
    state: WorkflowState
    progress_phase: ProgressPhase
    ref: string
    branch: string
    worktree_path: string
    started: string
  }
  work?: {
    title: string
    external_key: string
    description?: string
    created_at: string
    updated_at: string
    costs: CostData
    reviews?: Review[]
  }
  pending_question?: PendingQuestion
}

export interface CostData {
  total_input_tokens: number
  total_output_tokens: number
  total_cached_tokens?: number
  total_cost_usd: number
}

export interface PendingQuestion {
  question: string
  options?: QuestionOption[]
  task_id: string
}

export interface QuestionOption {
  label: string
  value: string
  description?: string
}

// /api/v1/tasks response
export interface TaskListResponse {
  tasks: TaskSummary[]
}

export interface TaskSummary {
  id: string
  ref: string
  state: WorkflowState
  title: string
  started: string
  completed?: string
}

// /api/v1/tasks/{id}/specs response
export interface SpecificationsResponse {
  specifications: Specification[]
}

export interface Specification {
  number: number
  name: string
  title: string
  description: string
  component: string
  status: 'pending' | 'in_progress' | 'completed'
  created_at: string
  completed_at?: string
  implemented_files?: string[]
}

export interface SpecificationDiffResponse {
  task_id: string
  specification: number
  file: string
  context: number
  has_diff: boolean
  diff: string
}

export interface AgentLogEntry {
  index: number
  kind?: string
  started_at?: string
  file?: string
  type?: 'output' | 'error' | 'info'
  message: string
}

export interface AgentLogsHistoryResponse {
  logs: AgentLogEntry[]
  task_id?: string
  count?: number
}

// Workflow actions
export type WorkflowAction =
  | 'plan'
  | 'implement'
  | 'review'
  | 'finish'
  | 'sync'
  | 'undo'
  | 'redo'
  | 'abandon'
  | 'reset'

export interface WorkflowSyncResponse {
  success: boolean
  has_changes: boolean
  changes_summary?: string
  spec_generated?: string
  source_updated?: boolean
  previous_snapshot_path?: string
  diff_path?: string
  warnings?: string[]
  message: string
}

// Options for implement action (passed as query params)
export interface ImplementOptions {
  component?: string
  parallel?: number
}

// SSE event types
export interface SSEEvent {
  type: string
  data?: unknown
}

export type SSEEventType =
  | 'state_changed'
  | 'progress'
  | 'agent_message'
  | 'costs_updated'
  | 'spec_updated'
  | 'question_asked'
  | 'connected'
  | 'heartbeat'

// =============================================================================
// Settings / WorkspaceConfig types (matches internal/storage/workspace_config.go)
// =============================================================================

export interface GitSettings {
  commit_prefix: string
  branch_pattern: string
  auto_commit: boolean
  sign_commits: boolean
  stash_on_start: boolean
  auto_pop_stash: boolean
  default_branch?: string
}

export interface StepAgentConfig {
  name?: string
  env?: Record<string, string>
  args?: string[]
  instructions?: string
  optimize_prompts?: boolean
}

export interface PRReviewConfig {
  enabled?: boolean
  format?: string
  scope?: string
  fail_on_issues?: boolean
  max_comments?: number
  exclude_patterns?: string[]
  acknowledge_fixes?: boolean
  update_existing?: boolean
}

export interface AgentSettings {
  default: string
  timeout: number
  max_retries: number
  instructions?: string
  optimize_prompts?: boolean
  steps?: Record<string, StepAgentConfig>
  pr_review?: PRReviewConfig
}

export interface SimplifySettings {
  instructions?: string
}

export interface WorkflowSettings {
  auto_init: boolean
  session_retention_days: number
  delete_work_on_finish: boolean
  delete_work_on_abandon: boolean
  prefer_local_merge: boolean
  simplify?: SimplifySettings
}

export interface BudgetConfig {
  max_tokens?: number
  max_cost?: number
  currency?: string
  on_limit?: string
  warning_at?: number
}

export interface MonthlyBudgetSettings {
  max_cost?: number
  currency?: string
  warning_at?: number
}

export interface BudgetSettings {
  enabled?: boolean
  per_task?: BudgetConfig
  monthly?: MonthlyBudgetSettings
  exchange_rates?: Record<string, number>
}

export interface ProvidersSettings {
  default?: string
  default_mention?: string
}

export interface ProjectSettings {
  code_dir?: string
}

export interface StorageSettings {
  home_dir?: string
  save_in_project?: boolean
  project_dir?: string
}

export interface UpdateSettings {
  enabled: boolean
  check_interval: number
}

export interface SpecificationSettings {
  filename_pattern: string
}

export interface ReviewSettings {
  filename_pattern: string
}

export interface StackSettings {
  auto_rebase?: string
  block_on_conflicts?: boolean
}

// Provider settings
export interface GitHubCommentsSettings {
  enabled: boolean
  on_branch_created: boolean
  on_plan_done: boolean
  on_implement_done: boolean
  on_pr_created: boolean
}

export interface GitHubSettings {
  token?: string
  owner?: string
  repo?: string
  branch_pattern?: string
  commit_prefix?: string
  target_branch?: string
  draft_pr?: boolean
  comments?: GitHubCommentsSettings
}

export interface GitLabSettings {
  token?: string
  host?: string
  project_path?: string
  branch_pattern?: string
  commit_prefix?: string
}

export interface JiraSettings {
  token?: string
  email?: string
  base_url?: string
  project?: string
}

export interface LinearSettings {
  token?: string
  team?: string
}

export interface NotionSettings {
  token?: string
  database_id?: string
  status_property?: string
  description_property?: string
  labels_property?: string
}

export interface YouTrackSettings {
  token?: string
  host?: string
}

export interface BitbucketSettings {
  username?: string
  app_password?: string
  workspace?: string
  repo?: string
  branch_pattern?: string
  commit_prefix?: string
  target_branch?: string
  close_source_branch?: boolean
}

export interface AsanaSettings {
  token?: string
  workspace_gid?: string
  default_project?: string
  branch_pattern?: string
  commit_prefix?: string
}

export interface ClickUpSettings {
  token?: string
  team_id?: string
  default_list?: string
  branch_pattern?: string
  commit_prefix?: string
}

export interface AzureDevOpsSettings {
  token?: string
  organization?: string
  project?: string
  area_path?: string
  iteration_path?: string
  repo_name?: string
  target_branch?: string
  branch_pattern?: string
  commit_prefix?: string
}

export interface TrelloSettings {
  api_key?: string
  token?: string
  board?: string
}

export interface WrikeSettings {
  token?: string
  host?: string
  space?: string
  folder?: string
  project?: string
}

// Feature settings
export interface BrowserSettings {
  enabled?: boolean
  host?: string
  port?: number
  headless?: boolean
  ignore_cert_errors?: boolean
  timeout?: number
  screenshot_dir?: string
  cookie_profile?: string
  cookie_auto_load?: boolean
  cookie_auto_save?: boolean
  cookie_dir?: string
}

export interface RateLimitSettings {
  rate?: number
  burst?: number
}

export interface MCPSettings {
  enabled?: boolean
  tools?: string[]
  rate_limit?: RateLimitSettings
}

export interface SecurityRunOnConfig {
  planning?: boolean
  implementing?: boolean
  reviewing?: boolean
}

export interface SecurityFailOnConfig {
  level?: string
  block_finish?: boolean
}

export interface SASTScannerConfig {
  enabled?: boolean
  tools?: Record<string, unknown>[]
}

export interface SecretScannerConfig {
  enabled?: boolean
  tools?: Record<string, unknown>[]
}

export interface DependencyScannerConfig {
  enabled?: boolean
  tools?: Record<string, unknown>[]
}

export interface LicenseScannerConfig {
  enabled?: boolean
  allowlist?: string[]
}

export interface SecurityScannersConfig {
  sast?: SASTScannerConfig
  secrets?: SecretScannerConfig
  dependencies?: DependencyScannerConfig
  license?: LicenseScannerConfig
}

export interface SecurityOutputConfig {
  format?: string
  file?: string
  include_suggestions?: boolean
}

export interface SecurityToolsConfig {
  auto_download?: boolean
  cache_dir?: string
  timeout?: number
}

export interface SecuritySettings {
  enabled?: boolean
  run_on?: SecurityRunOnConfig
  fail_on?: SecurityFailOnConfig
  scanners?: SecurityScannersConfig
  output?: SecurityOutputConfig
  tools?: SecurityToolsConfig
}

export interface VectorDBSettings {
  backend?: string
  connection_string?: string
  collection?: string
  embedding_model?: string
  onnx?: ONNXSettings
}

export interface ONNXSettings {
  model?: string
  cache_path?: string
  max_length?: number
}

export interface MemoryRetentionConfig {
  max_days?: number
  max_tasks?: number
}

export interface MemorySearchConfig {
  similarity_threshold?: number
  max_results?: number
  include_code?: boolean
  include_specs?: boolean
  include_sessions?: boolean
}

export interface MemoryLearningConfig {
  auto_store?: boolean
  learn_from_corrections?: boolean
  suggest_similar?: boolean
}

export interface MemorySettings {
  enabled?: boolean
  vector_db?: VectorDBSettings
  retention?: MemoryRetentionConfig
  search?: MemorySearchConfig
  learning?: MemoryLearningConfig
}

export interface LibrarySettings {
  auto_include_max?: number
  max_pages_per_prompt?: number
  max_crawl_pages?: number
  max_crawl_depth?: number
  max_page_size_bytes?: number
  lock_timeout?: string
  max_token_budget?: number
  domain_scope?: string
  version_filter?: boolean
  version_path?: string
}

export interface OrchestrationAgentConfig {
  name: string
  agent: string
  model?: string
  role: string
  input?: string[]
  output?: string
  depends?: string[]
  env?: Record<string, string>
  args?: string[]
  timeout?: number
}

export interface StepConsensusConfig {
  mode?: string
  min_votes?: number
  synthesizer?: string
}

export interface StepOrchestratorConfig {
  mode?: string
  agents?: OrchestrationAgentConfig[]
  consensus?: StepConsensusConfig
}

export interface OrchestrationSettings {
  enabled?: boolean
  steps?: Record<string, StepOrchestratorConfig>
}

export interface MLTelemetryConfig {
  enabled?: boolean
  anonymize?: boolean
  sample_rate?: number
  storage?: string
}

export interface MLModelConfig {
  type?: string
  retrain_interval?: string
  min_samples?: number
}

export interface MLPredictionsConfig {
  next_action?: boolean
  duration?: boolean
  complexity?: boolean
  agent_selection?: boolean
  risk_assessment?: boolean
}

export interface MLSettings {
  enabled?: boolean
  telemetry?: MLTelemetryConfig
  model?: MLModelConfig
  predictions?: MLPredictionsConfig
}

export interface SandboxSettings {
  enabled?: boolean
  network?: boolean
  tmp_dir?: string
  tools?: string[]
}

export interface LabelDefinition {
  name: string
  color?: string
}

export interface LabelSettings {
  enabled?: boolean
  defined?: LabelDefinition[]
  suggestions?: string[]
}

export interface LinterConfig {
  enabled?: boolean
  command?: string[]
  args?: string[]
  extensions?: string[]
}

export interface QualitySettings {
  enabled?: boolean
  use_defaults?: boolean
  linters?: Record<string, LinterConfig>
}

export interface LinksSettings {
  enabled?: boolean
  auto_index?: boolean
  case_sensitive?: boolean
  max_context_length?: number
}

export interface ContextSettings {
  include_parent?: boolean
  include_siblings?: boolean
  max_siblings?: number
  description_limit?: number
}

export interface DisplaySettings {
  timezone?: string
}

// Plugins
export interface PluginsConfig {
  enabled?: string[]
  config?: Record<string, Record<string, unknown>>
}

// Agent aliases
export interface AgentAliasConfig {
  extends: string
  binary_path?: string
  description?: string
  components?: string[]
  env?: Record<string, string>
  args?: string[]
}

// Complete WorkspaceConfig
export interface WorkspaceConfig {
  git: GitSettings
  agent: AgentSettings
  workflow: WorkflowSettings
  budget?: BudgetSettings
  providers?: ProvidersSettings
  env?: Record<string, string>
  agents?: Record<string, AgentAliasConfig>
  github?: GitHubSettings
  gitlab?: GitLabSettings
  notion?: NotionSettings
  jira?: JiraSettings
  linear?: LinearSettings
  wrike?: WrikeSettings
  youtrack?: YouTrackSettings
  bitbucket?: BitbucketSettings
  asana?: AsanaSettings
  clickup?: ClickUpSettings
  azure_devops?: AzureDevOpsSettings
  trello?: TrelloSettings
  plugins?: PluginsConfig
  update?: UpdateSettings
  storage?: StorageSettings
  browser?: BrowserSettings
  mcp?: MCPSettings
  specification?: SpecificationSettings
  review?: ReviewSettings
  security?: SecuritySettings
  memory?: MemorySettings
  library?: LibrarySettings
  orchestration?: OrchestrationSettings
  ml?: MLSettings
  sandbox?: SandboxSettings
  labels?: LabelSettings
  quality?: QualitySettings
  links?: LinksSettings
  context?: ContextSettings
  project?: ProjectSettings
  stack?: StackSettings
  display?: DisplaySettings
}

// /api/v1/tasks list item (from handleListTasks)
export interface TaskHistoryItem {
  id: string
  title: string
  state: WorkflowState
  progress_phase?: ProgressPhase
  created_at: string
  worktree_path?: string
}

// =============================================================================
// Task-specific responses
// =============================================================================

// /api/v1/tasks/{id}/notes response
export interface NotesResponse {
  notes: Note[]
  count: number
}

export interface Note {
  number: number
  content: string
  timestamp: string
  state?: string
}

// /api/v1/tasks/{id}/costs response
export interface CostsResponse {
  total_cost_usd: number
  total_tokens: number
  input_tokens: number
  output_tokens: number
  cached_tokens: number
  cached_percent?: number
  budget?: BudgetProgressStatus
  steps?: StepCost[]
}

export interface BudgetProgressStatus {
  type: string
  used: string
  max: string
  pct: number
  warned: boolean
  limit_hit: boolean
}

export interface StepCost {
  name: string
  total_tokens: number
  cost: string
}

// Reviews from task work
export interface Review {
  number: number
  status: string
  summary?: string
  issue_count: number
}

// PR info for completed tasks
export interface PullRequestInfo {
  number: number
  url: string
  created_at: string
}

// /api/v1/work/{id} response
export interface WorkResponse {
  active: boolean
  task?: {
    id: string
    state: WorkflowState
    progress_phase: ProgressPhase
    ref: string
    branch: string
    worktree_path: string
    started: string
  }
  work: {
    metadata: {
      id: string
      title: string
      state: WorkflowState
      external_key: string
      task_type?: string
      labels?: string[]
      created_at: string
      updated_at: string
      pull_request?: PullRequestInfo
    }
    source: {
      type: string
      ref: string
      read_at: string
    }
    git: {
      branch: string
      base_branch: string
      worktree_path?: string
    }
    costs: CostData
    description?: string
  }
}
