import { useId } from 'react'
import { AlertCircle, Loader2 } from 'lucide-react'
import { useAgents, type AgentInfo } from '@/api/settings'
import { FormField } from './FormField'

interface AgentSelectProps {
  label: string
  hint?: string
  error?: string
  value: string | undefined
  onChange: (value: string) => void
  disabled?: boolean
  /** Show "Auto-detect" empty option */
  allowEmpty?: boolean
  /** Custom label for empty option (default: "Auto-detect") */
  emptyLabel?: string
}

/**
 * Agent selection dropdown that fetches available agents from the API.
 * Groups agents by type (built-in first, then aliases).
 * Shows availability status with visual indicators.
 */
export function AgentSelect({
  label,
  hint,
  error,
  value,
  onChange,
  disabled,
  allowEmpty = false,
  emptyLabel = 'Auto-detect',
}: AgentSelectProps) {
  const inputId = useId()
  const hintId = useId()
  const errorId = useId()

  const { data, isLoading, error: fetchError } = useAgents()

  // Group and sort agents
  const agents = data?.agents ?? []
  const builtInAgents = agents
    .filter((a) => a.type === 'built-in')
    .sort((a, b) => a.name.localeCompare(b.name))
  const aliasAgents = agents
    .filter((a) => a.type === 'alias')
    .sort((a, b) => a.name.localeCompare(b.name))

  const descriptionIds = [error && errorId, hint && !error && hintId]
    .filter(Boolean)
    .join(' ') || undefined

  // Loading state
  if (isLoading) {
    return (
      <FormField label={label} hint={hint} inputId={inputId} hintId={hintId}>
        <div className="select select-bordered w-full flex items-center gap-2 text-base-content/60">
          <Loader2 size={16} className="animate-spin" aria-hidden="true" />
          <span>Loading agents...</span>
        </div>
      </FormField>
    )
  }

  // Error state - fall back to text input
  if (fetchError) {
    return (
      <FormField
        label={label}
        hint="Could not load agents. Enter agent name manually."
        error={error}
        inputId={inputId}
        hintId={hintId}
        errorId={errorId}
      >
        <div className="flex gap-2">
          <input
            id={inputId}
            type="text"
            className={`input input-bordered w-full ${error ? 'input-error' : ''}`}
            value={value || ''}
            onChange={(e) => onChange(e.target.value)}
            placeholder="claude"
            disabled={disabled}
            aria-invalid={error ? true : undefined}
            aria-describedby={descriptionIds}
          />
          <div className="tooltip tooltip-left" data-tip="Failed to load agents">
            <span className="btn btn-square btn-ghost btn-sm text-warning">
              <AlertCircle size={16} aria-hidden="true" />
            </span>
          </div>
        </div>
      </FormField>
    )
  }

  // No agents available
  if (agents.length === 0) {
    return (
      <FormField
        label={label}
        hint="No agents available. Configure agents in the Agent section above or add aliases in config.yaml."
        inputId={inputId}
        hintId={hintId}
      >
        <select
          id={inputId}
          className="select select-bordered w-full"
          value=""
          disabled
          aria-describedby={hintId}
        >
          <option value="">No agents available</option>
        </select>
      </FormField>
    )
  }

  return (
    <FormField
      label={label}
      hint={hint}
      error={error}
      inputId={inputId}
      hintId={hintId}
      errorId={errorId}
    >
      <select
        id={inputId}
        className={`select select-bordered w-full ${error ? 'select-error' : ''}`}
        value={value || ''}
        onChange={(e) => onChange(e.target.value)}
        disabled={disabled}
        aria-invalid={error ? true : undefined}
        aria-describedby={descriptionIds}
      >
        {allowEmpty && <option value="">{emptyLabel}</option>}
        {!allowEmpty && <option value="">Select agent...</option>}

        {builtInAgents.length > 0 && (
          <optgroup label="Built-in Agents">
            {builtInAgents.map((agent) => (
              <AgentOption key={agent.name} agent={agent} />
            ))}
          </optgroup>
        )}

        {aliasAgents.length > 0 && (
          <optgroup label="Custom Aliases">
            {aliasAgents.map((agent) => (
              <AgentOption key={agent.name} agent={agent} />
            ))}
          </optgroup>
        )}
      </select>
    </FormField>
  )
}

/**
 * Single agent option with availability indicator
 */
function AgentOption({ agent }: { agent: AgentInfo }) {
  const label = agent.type === 'alias' && agent.extends
    ? `${agent.name} (extends: ${agent.extends})`
    : agent.name

  // Show unavailable agents with indicator but don't disable
  // (user might want to see what's configured even if unavailable)
  const displayLabel = agent.available ? label : `${label} (unavailable)`

  return (
    <option
      value={agent.name}
      disabled={!agent.available}
      className={agent.available ? '' : 'text-base-content/50'}
    >
      {displayLabel}
    </option>
  )
}
