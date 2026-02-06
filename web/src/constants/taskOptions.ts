export interface TaskOption {
  value: string
  label: string
}

export const TASK_SOURCE_PROVIDERS: TaskOption[] = [
  { value: 'github', label: 'GitHub' },
  { value: 'gitlab', label: 'GitLab' },
  { value: 'jira', label: 'Jira' },
  { value: 'linear', label: 'Linear' },
  { value: 'wrike', label: 'Wrike' },
  { value: 'asana', label: 'Asana' },
  { value: 'clickup', label: 'ClickUp' },
  { value: 'notion', label: 'Notion' },
]

export const TASK_SUBMISSION_PROVIDERS: TaskOption[] = [
  { value: 'github', label: 'GitHub' },
  { value: 'gitlab', label: 'GitLab' },
  { value: 'jira', label: 'Jira' },
  { value: 'linear', label: 'Linear' },
  { value: 'wrike', label: 'Wrike' },
  { value: 'asana', label: 'Asana' },
  { value: 'clickup', label: 'ClickUp' },
]

export const TASK_TEMPLATES: TaskOption[] = [
  { value: '', label: 'No template' },
  { value: 'bug-fix', label: 'Bug Fix' },
  { value: 'feature', label: 'Feature' },
  { value: 'refactor', label: 'Refactor' },
  { value: 'docs', label: 'Documentation' },
  { value: 'test', label: 'Test' },
  { value: 'chore', label: 'Chore' },
]
