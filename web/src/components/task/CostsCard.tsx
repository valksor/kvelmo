import { DollarSign, Loader2, ChevronDown, ChevronRight, Coins, Gauge } from 'lucide-react'
import { useState } from 'react'
import { useTaskCosts } from '@/api/task'

interface CostsCardProps {
  taskId?: string
}

export function CostsCard({ taskId }: CostsCardProps) {
  const { data: costs, isLoading } = useTaskCosts(taskId)
  const [expanded, setExpanded] = useState(false)

  if (isLoading) {
    return (
      <div className="card bg-base-100 shadow-sm">
        <div className="card-body py-4">
          <div className="flex items-center gap-2">
            <Loader2 className="w-4 h-4 animate-spin" />
            <span className="text-sm text-base-content/60">Loading costs...</span>
          </div>
        </div>
      </div>
    )
  }

  if (!costs) {
    return null
  }

  const hasSteps = costs.steps && costs.steps.length > 0
  const hasBudget = costs.budget && costs.budget.max !== ''

  return (
    <div className="card bg-base-100 shadow-sm">
      <div className="card-body py-4">
        <div className="flex items-center justify-between">
          <h3 className="font-medium flex items-center gap-2">
            <DollarSign size={18} className="text-success" />
            Cost Summary
          </h3>
          {hasSteps && (
            <button
              onClick={() => setExpanded(!expanded)}
              className="btn btn-ghost btn-xs"
            >
              {expanded ? <ChevronDown size={16} /> : <ChevronRight size={16} />}
              {expanded ? 'Hide' : 'Show'} breakdown
            </button>
          )}
        </div>

        {/* Total stats */}
        <div className="mt-3 grid grid-cols-2 gap-3">
          <div className="bg-base-200 rounded-lg p-3">
            <div className="text-xs text-base-content/60">Total Cost</div>
            <div className="text-lg font-bold text-success">
              ${costs.total_cost_usd.toFixed(4)}
            </div>
          </div>
          <div className="bg-base-200 rounded-lg p-3">
            <div className="text-xs text-base-content/60 flex items-center gap-1">
              Total Tokens
              <div className="tooltip tooltip-top" data-tip="Tokens are units of text the AI processes. More tokens = more processing.">
                <span className="cursor-help text-base-content/40">ⓘ</span>
              </div>
            </div>
            <div className="text-lg font-bold">
              {costs.total_tokens.toLocaleString()}
            </div>
          </div>
        </div>

        {/* Token breakdown */}
        <div className="mt-2 grid grid-cols-3 gap-2 text-sm">
          <div className="text-center">
            <div className="text-base-content/60 text-xs">Input</div>
            <div>{costs.input_tokens.toLocaleString()}</div>
          </div>
          <div className="text-center">
            <div className="text-base-content/60 text-xs">Output</div>
            <div>{costs.output_tokens.toLocaleString()}</div>
          </div>
          <div className="text-center">
            <div className="text-base-content/60 text-xs">Cached</div>
            <div>
              {costs.cached_tokens.toLocaleString()}
              {costs.cached_percent != null && (
                <span className="text-xs text-base-content/50 ml-1">({costs.cached_percent.toFixed(1)}%)</span>
              )}
            </div>
          </div>
        </div>

        {/* Budget progress */}
        {hasBudget && (
          <div className="mt-3">
            <div className="flex items-center justify-between text-sm mb-1">
              <span className="flex items-center gap-1 text-base-content/70">
                <Gauge size={14} />
                Budget
              </span>
              <span className={costs.budget!.limit_hit ? 'text-error' : ''}>
                {costs.budget!.used} / {costs.budget!.max}
              </span>
            </div>
            <progress
              className={`progress w-full ${
                costs.budget!.limit_hit
                  ? 'progress-error'
                  : costs.budget!.warned
                    ? 'progress-warning'
                    : 'progress-primary'
              }`}
              value={costs.budget!.pct}
              max={100}
            />
            {costs.budget!.limit_hit && (
              <p className="text-xs text-error mt-1">Budget limit reached</p>
            )}
            {costs.budget!.warned && !costs.budget!.limit_hit && (
              <p className="text-xs text-warning mt-1">Approaching budget limit</p>
            )}
          </div>
        )}

        {/* Step-by-step breakdown */}
        {hasSteps && expanded && (
          <div className="mt-3 border-t border-base-300 pt-3">
            <h4 className="text-sm font-medium flex items-center gap-2 mb-2">
              <Coins size={14} />
              Per-Step Costs
            </h4>
            <div className="space-y-2">
              {costs.steps!.map((step, idx) => (
                <div
                  key={idx}
                  className="flex items-center justify-between py-1 px-2 bg-base-200 rounded"
                >
                  <span className="text-sm capitalize">{step.name}</span>
                  <div className="flex items-center gap-3 text-sm">
                    <span className="text-base-content/60">
                      {step.total_tokens.toLocaleString()} tokens
                    </span>
                    <span className="font-mono text-success">{step.cost}</span>
                  </div>
                </div>
              ))}
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
