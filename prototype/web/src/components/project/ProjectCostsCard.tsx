import { useQuery } from '@tanstack/react-query'
import { apiRequest } from '@/api/client'
import { Loader2, AlertTriangle, AlertCircle } from 'lucide-react'

interface BudgetStatusResponse {
  enabled: boolean
  max_cost?: number
  spent?: number
  remaining?: number
  currency?: string
  warning_at?: number
  warned?: boolean
  limit_hit?: boolean
}

export function ProjectCostsCard() {
  const { data: budget, isLoading } = useQuery({
    queryKey: ['budget', 'monthly', 'status'],
    queryFn: () => apiRequest<BudgetStatusResponse>('/budget/monthly/status'),
    refetchInterval: 60000, // Refresh every minute
  })

  if (isLoading) {
    return (
      <div className="card bg-base-100 shadow-sm">
        <div className="card-body">
          <div className="flex items-center justify-center py-8">
            <Loader2 className="w-6 h-6 animate-spin text-primary" />
          </div>
        </div>
      </div>
    )
  }

  if (!budget?.enabled) {
    return null // Don't show card if budget tracking not enabled
  }

  const spent = budget.spent || 0
  const maxCost = budget.max_cost || 0
  const remaining = budget.remaining || maxCost - spent
  const currency = budget.currency || 'USD'
  const pct = maxCost > 0 ? (spent / maxCost) * 100 : 0

  const formatCost = (amount: number) => {
    return new Intl.NumberFormat('en-US', {
      style: 'currency',
      currency: currency,
      minimumFractionDigits: 2,
    }).format(amount)
  }

  const budgetColor = budget.limit_hit
    ? 'bg-error'
    : budget.warned
      ? 'bg-warning'
      : pct > 75
        ? 'bg-warning'
        : 'bg-success'

  return (
    <div className="card bg-base-100 shadow-sm">
      <div className="card-body">
        <div className="flex items-center justify-between pb-4 border-b border-base-200">
          <h3 className="text-lg font-bold text-base-content">Monthly Budget</h3>
          <span className="text-sm text-base-content/60">
            {formatCost(spent)} / {formatCost(maxCost)}
          </span>
        </div>

        {/* Progress bar */}
        <div className="mt-4">
          <div className="h-3 bg-base-200 rounded-full overflow-hidden">
            <div
              className={`h-full ${budgetColor} transition-all duration-300`}
              style={{ width: `${Math.min(pct, 100)}%` }}
            />
          </div>
          <p className="text-xs text-base-content/60 mt-1">{Math.round(pct)}% used</p>
        </div>

        {/* Stats */}
        <div className="grid grid-cols-2 gap-4 mt-4">
          <div className="text-center p-3 bg-primary/10 rounded-xl">
            <div className="text-xl font-bold text-primary">{formatCost(spent)}</div>
            <div className="text-xs text-base-content/60 mt-1">Spent</div>
          </div>
          <div className="text-center p-3 bg-success/10 rounded-xl">
            <div className="text-xl font-bold text-success">{formatCost(remaining)}</div>
            <div className="text-xs text-base-content/60 mt-1">Remaining</div>
          </div>
        </div>

        {/* Warnings */}
        {budget.limit_hit && (
          <div className="mt-4 flex items-center gap-2 text-sm text-error">
            <AlertCircle size={16} />
            Budget limit reached - tasks paused
          </div>
        )}
        {budget.warned && !budget.limit_hit && (
          <div className="mt-4 flex items-center gap-2 text-sm text-warning">
            <AlertTriangle size={16} />
            Approaching budget limit
          </div>
        )}
      </div>
    </div>
  )
}
