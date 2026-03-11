import { useState } from 'react'
import { useGlobalStore } from '../../stores/globalStore'

interface TestResult {
  ok: boolean
  detail: string
}

const PROVIDERS = [
  { name: 'github', label: 'GitHub', envHint: 'GITHUB_TOKEN' },
  { name: 'gitlab', label: 'GitLab', envHint: 'GITLAB_TOKEN' },
  { name: 'linear', label: 'Linear', envHint: 'LINEAR_TOKEN' },
  { name: 'wrike', label: 'Wrike', envHint: 'WRIKE_TOKEN' },
]

export function ProviderTestButtons() {
  const { client } = useGlobalStore()
  const [testing, setTesting] = useState<Record<string, boolean>>({})
  const [results, setResults] = useState<Record<string, TestResult>>({})

  const handleTest = async (provider: string) => {
    if (!client) return

    setTesting(prev => ({ ...prev, [provider]: true }))
    setResults(prev => {
      const next = { ...prev }
      delete next[provider]
      return next
    })

    try {
      // Test with empty token — the backend reads from env/config
      const result = await client.call<TestResult>('providers.test', {
        provider,
        token: '__use_configured__',
      })
      setResults(prev => ({ ...prev, [provider]: result }))
    } catch (err) {
      setResults(prev => ({
        ...prev,
        [provider]: { ok: false, detail: err instanceof Error ? err.message : 'Test failed' },
      }))
    } finally {
      setTesting(prev => ({ ...prev, [provider]: false }))
    }
  }

  return (
    <div className="mt-6 p-4 bg-base-200 rounded-lg">
      <h3 className="font-medium text-sm mb-3">Test Provider Connections</h3>
      <div className="space-y-2">
        {PROVIDERS.map(p => (
          <div key={p.name} className="flex items-center gap-3">
            <span className="text-sm w-16">{p.label}</span>
            <button
              onClick={() => handleTest(p.name)}
              disabled={testing[p.name] || !client}
              className="btn btn-xs btn-outline"
            >
              {testing[p.name] ? (
                <span className="loading loading-spinner loading-xs"></span>
              ) : (
                'Verify'
              )}
            </button>
            {results[p.name] && (
              <span className={`text-xs ${results[p.name].ok ? 'text-success' : 'text-error'}`}>
                {results[p.name].detail}
              </span>
            )}
          </div>
        ))}
      </div>
      <p className="text-xs text-base-content/50 mt-2">
        Tests use tokens from environment variables or saved settings.
      </p>
    </div>
  )
}
