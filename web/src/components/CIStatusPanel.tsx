import { useState, useCallback, useEffect } from 'react'
import { useProjectStore } from '../stores/projectStore'
import { AccessibleModal } from './ui/AccessibleModal'

interface CIStatusPanelProps {
  isOpen: boolean
  onClose: () => void
}

interface CICheck {
  name: string
  status: 'pass' | 'fail' | 'pending'
  url?: string
}

const STATUS_BADGE: Record<string, string> = {
  pass: 'badge-success',
  fail: 'badge-error',
  pending: 'badge-warning',
}

const STATUS_LABEL: Record<string, string> = {
  pass: 'Pass',
  fail: 'Fail',
  pending: 'Pending',
}

export function CIStatusPanel({ isOpen, onClose }: CIStatusPanelProps) {
  const { client, connected } = useProjectStore()

  const [checks, setChecks] = useState<CICheck[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const loadCIStatus = useCallback(async () => {
    if (!client || !connected) return

    setLoading(true)
    setError(null)

    try {
      const result = await client.call<{ checks: CICheck[] }>('ci.status', {})
      setChecks(result.checks || [])
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load CI status')
      setChecks([])
    } finally {
      setLoading(false)
    }
  }, [client, connected])

  useEffect(() => {
    if (isOpen && connected) {
      loadCIStatus()
    }
  }, [isOpen, connected, loadCIStatus])

  return (
    <AccessibleModal isOpen={isOpen} onClose={onClose} title="CI Status" size="2xl">
      <div className="max-h-[70vh] flex flex-col">
        {/* Toolbar */}
        <div className="flex items-center justify-end mb-4">
          <button
            onClick={loadCIStatus}
            disabled={loading || !connected}
            className="btn btn-ghost btn-sm"
            aria-label="Refresh CI status"
          >
            {loading ? (
              <span className="loading loading-spinner loading-xs"></span>
            ) : (
              <svg aria-hidden="true" className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
              </svg>
            )}
            Refresh
          </button>
        </div>

        {/* Error */}
        {error && (
          <div className="alert alert-error py-2 mb-4">
            <span className="text-sm">{error}</span>
          </div>
        )}

        {/* Content */}
        <div className="flex-1 overflow-y-auto">
          {loading && checks.length === 0 ? (
            <div className="flex items-center justify-center py-12">
              <span className="loading loading-spinner loading-lg text-primary"></span>
            </div>
          ) : checks.length === 0 ? (
            <div className="text-center py-12 text-base-content/50">
              <svg aria-hidden="true" className="w-10 h-10 mx-auto mb-3 opacity-30" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M9 3v2m6-2v2M9 19v2m6-2v2M5 9H3m2 6H3m18-6h-2m2 6h-2M7 19h10a2 2 0 002-2V7a2 2 0 00-2-2H7a2 2 0 00-2 2v10a2 2 0 002 2z" />
              </svg>
              <p>No CI data available</p>
            </div>
          ) : (
            <div className="overflow-x-auto">
              <table className="table table-sm table-zebra w-full">
                <thead>
                  <tr>
                    <th>Check</th>
                    <th>Status</th>
                    <th></th>
                  </tr>
                </thead>
                <tbody>
                  {checks.map((check, i) => (
                    <tr key={`${check.name}-${i}`}>
                      <td className="font-mono text-xs">{check.name}</td>
                      <td>
                        <span className={`badge badge-sm ${STATUS_BADGE[check.status] || 'badge-ghost'}`}>
                          {STATUS_LABEL[check.status] || check.status}
                        </span>
                      </td>
                      <td className="text-right">
                        {check.url && (
                          <a
                            href={check.url}
                            target="_blank"
                            rel="noopener noreferrer"
                            className="link link-primary text-xs"
                          >
                            View
                          </a>
                        )}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>
      </div>
    </AccessibleModal>
  )
}
