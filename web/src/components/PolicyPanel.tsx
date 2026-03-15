import { useState, useCallback, useEffect } from 'react'
import { useProjectStore } from '../stores/projectStore'
import { AccessibleModal } from './ui/AccessibleModal'

interface PolicyPanelProps {
  isOpen: boolean
  onClose: () => void
}

interface PolicyResult {
  name: string
  status: 'pass' | 'fail'
  message: string
}

export function PolicyPanel({ isOpen, onClose }: PolicyPanelProps) {
  const { client, connected } = useProjectStore()

  const [policies, setPolicies] = useState<PolicyResult[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const checkPolicies = useCallback(async () => {
    if (!client || !connected) return

    setLoading(true)
    setError(null)

    try {
      const result = await client.call<{ results: PolicyResult[] }>('policy.check', {})
      setPolicies(result.results || [])
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to check policies')
      setPolicies([])
    } finally {
      setLoading(false)
    }
  }, [client, connected])

  useEffect(() => {
    if (isOpen && connected) {
      checkPolicies()
    }
  }, [isOpen, connected, checkPolicies])

  return (
    <AccessibleModal isOpen={isOpen} onClose={onClose} title="Policy Checks" size="2xl">
      <div className="max-h-[70vh] flex flex-col">
        {/* Toolbar */}
        <div className="flex items-center justify-end mb-4">
          <button
            onClick={checkPolicies}
            disabled={loading || !connected}
            className="btn btn-ghost btn-sm"
            aria-label="Check policies"
          >
            {loading ? (
              <span className="loading loading-spinner loading-xs"></span>
            ) : (
              <svg aria-hidden="true" className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z" />
              </svg>
            )}
            Check Policies
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
          {loading && policies.length === 0 ? (
            <div className="flex items-center justify-center py-12">
              <span className="loading loading-spinner loading-lg text-primary"></span>
            </div>
          ) : policies.length === 0 ? (
            <div className="text-center py-12 text-base-content/50">
              <svg aria-hidden="true" className="w-10 h-10 mx-auto mb-3 opacity-30" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z" />
              </svg>
              <p>No policy data available</p>
            </div>
          ) : (
            <div className="overflow-x-auto">
              <table className="table table-sm table-zebra w-full">
                <thead>
                  <tr>
                    <th>Policy</th>
                    <th>Status</th>
                    <th>Message</th>
                  </tr>
                </thead>
                <tbody>
                  {policies.map((policy, i) => (
                    <tr key={`${policy.name}-${i}`}>
                      <td className="font-mono text-xs">{policy.name}</td>
                      <td>
                        <span className={`badge badge-sm ${policy.status === 'pass' ? 'badge-success' : 'badge-error'}`}>
                          {policy.status === 'pass' ? 'Pass' : 'Fail'}
                        </span>
                      </td>
                      <td className="text-xs text-base-content/70">{policy.message}</td>
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
