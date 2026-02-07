import { useId, useState } from 'react'
import { Search, FileCode, Copy, Check, Loader2, AlertCircle } from 'lucide-react'
import { useFindCode, type FindResult } from '@/api/find'
import { useStatus } from '@/api/workflow'
import { ProjectSelector } from '@/components/project/ProjectSelector'

export default function Find() {
  const { data: status, isLoading: statusLoading } = useStatus()
  const id = useId()
  const [query, setQuery] = useState('')
  const [searchQuery, setSearchQuery] = useState('')

  const { data, isLoading, error } = useFindCode(searchQuery)

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
        <h1 className="text-2xl font-bold">Code Search</h1>
        <ProjectSelector />
      </div>
    )
  }

  const handleSearch = (e: React.FormEvent) => {
    e.preventDefault()
    if (query.trim().length >= 3) {
      setSearchQuery(query.trim())
    }
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <h1 className="text-2xl font-bold">Code Search</h1>
        <p className="text-base-content/60 mt-1">
          AI-powered semantic search across your codebase
        </p>
      </div>

      <div className="card bg-base-100 shadow-sm border border-base-300/70">
        <div className="card-body">
          <form onSubmit={handleSearch} className="flex flex-col gap-3 sm:flex-row sm:items-end">
            <div className="form-control flex-1">
              <label className="label py-1" htmlFor={`${id}-query`}>
                <span className="label-text">Search query</span>
              </label>
              <div className="relative">
                <Search
                  size={18}
                  aria-hidden="true"
                  className="absolute left-3 top-1/2 -translate-y-1/2 text-base-content/50"
                />
                <input
                  id={`${id}-query`}
                  type="text"
                  value={query}
                  onChange={(e) => setQuery(e.target.value)}
                  placeholder="Search code... (min 3 characters)"
                  className="input input-bordered w-full pl-10"
                />
              </div>
            </div>
            <button
              type="submit"
              className="btn btn-primary"
              disabled={query.trim().length < 3 || isLoading}
            >
              {isLoading ? <Loader2 size={18} aria-hidden="true" className="animate-spin" /> : 'Search'}
            </button>
          </form>
        </div>
      </div>

      {/* Results */}
      {error && (
        <div className="alert alert-error">
          <AlertCircle size={18} aria-hidden="true" />
          <span>Search failed: {error.message}</span>
        </div>
      )}

      {data && (
        <div className="space-y-4">
          <div className="text-sm text-base-content/60">
            Found {data.total} result{data.total !== 1 ? 's' : ''} for "{data.query}"
          </div>

          {data.results.length === 0 ? (
            <div className="card bg-base-100 shadow-sm">
              <div className="card-body text-center py-12">
                <p className="text-base-content/60">No results found</p>
                <p className="text-sm text-base-content/40">
                  Try different keywords or phrases
                </p>
              </div>
            </div>
          ) : (
            <div className="space-y-3">
              {data.results.map((result, idx) => (
                <ResultCard key={idx} result={result} />
              ))}
            </div>
          )}
        </div>
      )}

      {!data && !isLoading && searchQuery && (
        <div className="text-center py-12 text-base-content/60">
          Enter a search query to find code
        </div>
      )}
    </div>
  )
}

function ResultCard({ result }: { result: FindResult }) {
  const [copied, setCopied] = useState(false)

  const copyPath = async () => {
    await navigator.clipboard.writeText(`${result.file}:${result.line}`)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  return (
    <div className="card bg-base-100 shadow-sm">
      <div className="card-body py-3 px-4">
        {/* Header */}
        <div className="flex items-start justify-between gap-2">
          <div className="flex items-center gap-2 min-w-0">
            <FileCode size={16} aria-hidden="true" className="text-primary shrink-0" />
            <span className="font-mono text-sm truncate">{result.file}</span>
            <span className="text-xs text-base-content/50">:{result.line}</span>
          </div>
          <button
            className="btn btn-ghost btn-xs"
            onClick={copyPath}
            title="Copy path"
          >
            {copied ? <Check size={14} aria-hidden="true" className="text-success" /> : <Copy size={14} aria-hidden="true" />}
          </button>
        </div>

        {/* Code snippet */}
        <div className="mt-2 bg-base-200 rounded-lg overflow-x-auto">
          <pre className="p-3 text-xs font-mono">
            {result.context_before?.map((line, i) => (
              <div key={`before-${i}`} className="text-base-content/40">
                {line}
              </div>
            ))}
            <div className="text-base-content bg-primary/10 -mx-3 px-3 py-0.5">
              {result.content}
            </div>
            {result.context_after?.map((line, i) => (
              <div key={`after-${i}`} className="text-base-content/40">
                {line}
              </div>
            ))}
          </pre>
        </div>

        {/* Score */}
        {result.score !== undefined && (
          <div className="text-xs text-base-content/50 mt-1">
            Relevance: {(result.score * 100).toFixed(0)}%
          </div>
        )}
      </div>
    </div>
  )
}
