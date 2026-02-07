import { useId, useState } from 'react'
import { Loader2, AlertCircle, Sparkles, FileText, Plus, Pencil, Trash2, CheckCircle, DollarSign } from 'lucide-react'
import { useStandaloneSimplify, type StandaloneMode } from '@/api/standalone'
import { useStatus } from '@/api/workflow'
import { ProjectSelector } from '@/components/project/ProjectSelector'
import { Checkbox, FormField, TextArea, TextInput } from '@/components/settings/FormField'

export default function Simplify() {
  const { data: status, isLoading: statusLoading } = useStatus()
  const id = useId()
  const [mode, setMode] = useState<StandaloneMode>('uncommitted')
  const [baseBranch, setBaseBranch] = useState('main')
  const [range, setRange] = useState('')
  const [files, setFiles] = useState('')
  const [context, setContext] = useState(3)
  const [agent, setAgent] = useState('')
  const [createCheckpoint, setCreateCheckpoint] = useState(true)

  const simplifyMutation = useStandaloneSimplify()

  if (statusLoading) {
    return (
      <div className="flex items-center justify-center min-h-[400px]">
        <Loader2 aria-hidden="true" className="w-8 h-8 animate-spin text-primary" />
      </div>
    )
  }

  // Global mode: show project selector
  if (status?.mode === 'global') {
    return (
      <div className="space-y-4">
        <h1 className="text-2xl font-bold">Code Simplifier</h1>
        <ProjectSelector />
      </div>
    )
  }

  const handleRun = async () => {
    const request: Parameters<typeof simplifyMutation.mutateAsync>[0] = {
      mode,
      context,
      create_checkpoint: createCheckpoint,
    }

    if (mode === 'branch') request.base_branch = baseBranch
    if (mode === 'range') request.range = range
    if (mode === 'files') request.files = files.split(',').map((f) => f.trim()).filter(Boolean)
    if (agent) request.agent = agent

    await simplifyMutation.mutateAsync(request)
  }

  const getOperationIcon = (op: string) => {
    switch (op) {
      case 'create':
        return <Plus size={14} aria-hidden="true" className="text-success" />
      case 'update':
        return <Pencil size={14} aria-hidden="true" className="text-info" />
      case 'delete':
        return <Trash2 size={14} aria-hidden="true" className="text-error" />
      default:
        return <FileText size={14} aria-hidden="true" />
    }
  }

  const getOperationBadge = (op: string) => {
    switch (op) {
      case 'create':
        return 'badge-success'
      case 'update':
        return 'badge-info'
      case 'delete':
        return 'badge-error'
      default:
        return 'badge-ghost'
    }
  }

  const changes = simplifyMutation.data?.changes || []
  const usage = simplifyMutation.data?.usage

  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <h1 className="text-2xl font-bold">Code Simplifier</h1>
        <p className="text-base-content/60 mt-1">
          AI-powered code simplification and cleanup without an active task
        </p>
      </div>

      {/* Configuration */}
      <div className="card bg-base-100 shadow-sm">
        <div className="card-body">
          <h3 className="font-medium mb-4">Simplify Configuration</h3>

          <form
            className="space-y-4"
            onSubmit={(e) => {
              e.preventDefault()
              void handleRun()
            }}
          >
            <FormField label="Target Mode" inputId={`${id}-mode`}>
              <select
                id={`${id}-mode`}
                value={mode}
                onChange={(e) => setMode(e.target.value as StandaloneMode)}
                className="select select-bordered w-full"
              >
                <option value="uncommitted">Uncommitted Changes</option>
                <option value="branch">Branch Comparison</option>
                <option value="range">Commit Range</option>
                <option value="files">Specific Files</option>
              </select>
            </FormField>

            {mode === 'branch' && (
              <TextInput
                label="Base Branch"
                value={baseBranch}
                onChange={setBaseBranch}
                placeholder="main"
              />
            )}

            {mode === 'range' && (
              <TextInput
                label="Commit Range"
                value={range}
                onChange={setRange}
                placeholder="HEAD~3..HEAD"
              />
            )}

            {mode === 'files' && (
              <TextArea
                label="File Paths"
                hint="Comma-separated paths"
                value={files}
                onChange={setFiles}
                placeholder="src/main.go, internal/handler.go"
                rows={4}
              />
            )}

            <FormField label="Context Lines" hint={`${context} lines (0 to 10)`} inputId={`${id}-context`}>
              <input
                id={`${id}-context`}
                type="range"
                min={0}
                max={10}
                value={context}
                onChange={(e) => setContext(parseInt(e.target.value, 10))}
                className="range range-primary"
              />
              <div className="w-full flex justify-between text-xs px-2 mt-1 text-base-content/60">
                <span>0</span>
                <span>5</span>
                <span>10</span>
              </div>
            </FormField>

            <TextInput
              label="Agent (optional)"
              value={agent}
              onChange={setAgent}
              placeholder="Leave empty for default"
            />

            <Checkbox
              label="Create checkpoint before simplifying (recommended)"
              checked={createCheckpoint}
              onChange={setCreateCheckpoint}
            />

            <button
              type="submit"
              disabled={simplifyMutation.isPending}
              className="btn btn-primary w-full"
            >
              {simplifyMutation.isPending ? (
                <>
                  <Loader2 aria-hidden="true" className="w-4 h-4 animate-spin" />
                  Simplifying...
                </>
              ) : (
                <>
                  <Sparkles size={18} aria-hidden="true" />
                  Run Simplify
                </>
              )}
            </button>
          </form>
        </div>
      </div>

      {/* Error */}
      {simplifyMutation.isError && (
        <div className="alert alert-error">
          <AlertCircle size={18} aria-hidden="true" />
          <span>{simplifyMutation.error.message}</span>
        </div>
      )}

      {/* Results */}
      {simplifyMutation.isSuccess && (
        <div className="space-y-4">
          {/* Usage stats */}
          {usage && (
            <div className="stats shadow w-full">
              <div className="stat">
                <div className="stat-title">Input Tokens</div>
                <div className="stat-value text-sm">{usage.input_tokens.toLocaleString()}</div>
              </div>
              <div className="stat">
                <div className="stat-title">Output Tokens</div>
                <div className="stat-value text-sm">{usage.output_tokens.toLocaleString()}</div>
              </div>
              <div className="stat">
                <div className="stat-title">Cached</div>
                <div className="stat-value text-sm">{usage.cached_tokens.toLocaleString()}</div>
              </div>
              <div className="stat">
                <div className="stat-title">Cost</div>
                <div className="stat-value text-success flex items-center gap-1">
                  <DollarSign size={20} aria-hidden="true" />
                  {usage.cost_usd.toFixed(4)}
                </div>
              </div>
            </div>
          )}

          {/* Summary text */}
          {simplifyMutation.data.summary && (
            <div className="card bg-base-100 shadow-sm">
              <div className="card-body">
                <h3 className="font-medium">Summary</h3>
                <p className="text-base-content/80 whitespace-pre-wrap">{simplifyMutation.data.summary}</p>
              </div>
            </div>
          )}

          {/* Changes list */}
          {changes.length === 0 ? (
            <div className="card bg-base-100 shadow-sm">
              <div className="card-body text-center py-12">
                <CheckCircle aria-hidden="true" className="w-12 h-12 mx-auto text-success mb-4" />
                <h2 className="text-lg font-medium">No Changes Needed</h2>
                <p className="text-base-content/60 mt-2">
                  The code is already simplified or no improvements were found.
                </p>
              </div>
            </div>
          ) : (
            <div className="card bg-base-100 shadow-sm">
              <div className="card-body">
                <h3 className="font-medium mb-4">File Changes ({changes.length})</h3>
                <div className="space-y-2">
                  {changes.map((change, idx) => (
                    <div
                      key={idx}
                      className="flex items-center justify-between py-2 px-3 bg-base-200 rounded-lg"
                    >
                      <div className="flex items-center gap-2">
                        {getOperationIcon(change.operation)}
                        <span className="font-mono text-sm">{change.path}</span>
                      </div>
                      <span className={`badge badge-sm ${getOperationBadge(change.operation)}`}>
                        {change.operation}
                      </span>
                    </div>
                  ))}
                </div>
              </div>
            </div>
          )}

          {/* Error from response */}
          {simplifyMutation.data.error && (
            <div className="alert alert-warning">
              <AlertCircle size={18} aria-hidden="true" />
              <span>{simplifyMutation.data.error}</span>
            </div>
          )}
        </div>
      )}
    </div>
  )
}
