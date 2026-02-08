import { Bot } from 'lucide-react'
import { CollapseSection } from '../FormField'
import { AgentSelect } from '../AgentSelect'

/** Step agent configuration from workspace config */
interface StepAgentConfig {
  name?: string
  // env, args, instructions exist but we only expose name in this UI
}

interface AgentStepsSettingsProps {
  /** Per-step agent configuration values */
  values: {
    planning?: StepAgentConfig
    implementing?: StepAgentConfig
    reviewing?: StepAgentConfig
  }
  /** Callback when a step's agent changes */
  onChange: (step: string, name: string | undefined) => void
  /** Current default agent name (shown when step has no specific agent) */
  defaultAgent: string
}

const WORKFLOW_STEPS = [
  {
    id: 'planning',
    label: 'Planning',
    hint: 'Agent used for mehr plan - generates specifications from task descriptions',
  },
  {
    id: 'implementing',
    label: 'Implementing',
    hint: 'Agent used for mehr implement - executes specifications and writes code',
  },
  {
    id: 'reviewing',
    label: 'Reviewing',
    hint: 'Agent used for mehr review - reviews implementation against specifications',
  },
] as const

/**
 * Per-step agent configuration section.
 * Allows setting different agents for planning, implementing, and reviewing steps.
 * Only shown in advanced mode.
 */
export function AgentStepsSettings({
  values,
  onChange,
  defaultAgent,
}: AgentStepsSettingsProps) {
  return (
    <CollapseSection title="Per-Step Agents" defaultOpen={false}>
      <div className="space-y-4">
        <p className="text-sm text-base-content/70">
          Override the default agent for specific workflow steps. Leave empty to use the default agent.
        </p>

        {WORKFLOW_STEPS.map((step) => {
          const stepConfig = values[step.id as keyof typeof values]
          const currentValue = stepConfig?.name ?? ''

          return (
            <AgentSelect
              key={step.id}
              label={step.label}
              hint={step.hint}
              value={currentValue}
              onChange={(name) => onChange(step.id, name || undefined)}
              allowEmpty
              emptyLabel={`Use default (${defaultAgent})`}
            />
          )
        })}

        <div className="alert text-sm">
          <Bot size={16} className="text-base-content/60" aria-hidden="true" />
          <span>
            Settings here override the default agent. Command-line flags and task settings take precedence.
          </span>
        </div>
      </div>
    </CollapseSection>
  )
}
