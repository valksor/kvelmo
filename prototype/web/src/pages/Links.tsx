import { useState } from 'react'
import { Loader2, Link2, Search, FileText, ArrowRight } from 'lucide-react'
import { useLinksStatus, useSearchLinks, useBacklinks } from '@/api/links'
import { useStatus } from '@/api/workflow'
import { ProjectSelector } from '@/components/project/ProjectSelector'
import { useDebounce } from '@/hooks/useDebounce'

export default function Links() {
  const { data: status, isLoading: statusLoading } = useStatus()
  const [searchQuery, setSearchQuery] = useState('')
  const [selectedRef, setSelectedRef] = useState<string | null>(null)
  const debouncedQuery = useDebounce(searchQuery, 300)

  const { data: linksStatus, isLoading: linksStatusLoading } = useLinksStatus()
  const { data: searchData, isLoading: searchLoading } = useSearchLinks(debouncedQuery)
  const { data: backlinksData, isLoading: backlinksLoading } = useBacklinks(selectedRef || '')

  if (statusLoading || linksStatusLoading) {
    return (
      <div className="flex items-center justify-center min-h-[400px]">
        <Loader2 className="w-8 h-8 animate-spin text-primary" />
      </div>
    )
  }

  // Global mode: show project selector
  if (status?.mode === 'global') {
    return (
      <div className="space-y-4">
        <h1 className="text-2xl font-bold">Knowledge Links</h1>
        <ProjectSelector />
      </div>
    )
  }

  // Check if links feature is disabled
  if (linksStatus && !linksStatus.enabled) {
    return (
      <div className="space-y-6">
        <h1 className="text-2xl font-bold">Knowledge Links</h1>
        <div className="card bg-base-100 shadow-sm">
          <div className="card-body text-center py-12">
            <Link2 className="w-12 h-12 mx-auto text-base-content/30 mb-4" />
            <h2 className="text-lg font-medium">Links Disabled</h2>
            <p className="text-base-content/60 mt-2">
              Enable the links feature in Settings to use bidirectional linking.
            </p>
            <p className="text-sm text-base-content/50 mt-4">
              Links allow you to create <code className="bg-base-200 px-1 rounded">[[references]]</code> in your
              documents that connect related content.
            </p>
          </div>
        </div>
      </div>
    )
  }

  const links = searchData?.links || []

  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <h1 className="text-2xl font-bold">Knowledge Links</h1>
        <p className="text-base-content/60 mt-1">
          Search and explore bidirectional links using <code className="bg-base-200 px-1 rounded">[[reference]]</code>{' '}
          syntax
        </p>
      </div>

      {/* Search */}
      <div className="relative">
        <Search size={16} className="absolute left-3 top-1/2 -translate-y-1/2 text-base-content/50" />
        <input
          type="text"
          value={searchQuery}
          onChange={(e) => {
            setSearchQuery(e.target.value)
            setSelectedRef(null)
          }}
          placeholder="Search links (e.g., spec:1, decision:cache)..."
          className="input input-bordered w-full pl-10"
        />
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Search Results */}
        <div className="space-y-4">
          <h3 className="font-medium text-base-content/70">
            {debouncedQuery ? `Results for "${debouncedQuery}"` : 'Search Results'}
          </h3>

          {!debouncedQuery ? (
            <div className="card bg-base-100 shadow-sm">
              <div className="card-body text-center py-12">
                <Search className="w-10 h-10 mx-auto text-base-content/30 mb-2" />
                <p className="text-base-content/60">Enter a search term to find links</p>
              </div>
            </div>
          ) : searchLoading ? (
            <div className="flex justify-center py-12">
              <Loader2 className="w-6 h-6 animate-spin text-primary" />
            </div>
          ) : links.length === 0 ? (
            <div className="card bg-base-100 shadow-sm">
              <div className="card-body text-center py-8">
                <p className="text-base-content/60">No links found matching "{debouncedQuery}"</p>
              </div>
            </div>
          ) : (
            <div className="space-y-2">
              {links.map((link, idx) => (
                <div
                  key={`${link.ref}-${idx}`}
                  className={`card bg-base-100 shadow-sm cursor-pointer transition-all ${
                    selectedRef === link.ref ? 'ring-2 ring-primary' : 'hover:shadow-md'
                  }`}
                  onClick={() => setSelectedRef(link.ref)}
                >
                  <div className="card-body py-3 px-4">
                    <div className="flex items-start justify-between">
                      <div>
                        <div className="flex items-center gap-2">
                          <code className="text-primary font-mono text-sm">[[{link.ref}]]</code>
                          <span className="badge badge-sm badge-ghost">{link.type}</span>
                        </div>
                        {link.title && (
                          <p className="text-sm font-medium mt-1">{link.title}</p>
                        )}
                        <div className="flex items-center gap-1 text-xs text-base-content/50 mt-1">
                          <FileText size={12} />
                          <span className="font-mono">{link.file}</span>
                          {link.line && <span>:{link.line}</span>}
                        </div>
                      </div>
                      <ArrowRight size={16} className="text-base-content/30" />
                    </div>
                  </div>
                </div>
              ))}
              {searchData?.total && searchData.total > links.length && (
                <p className="text-sm text-base-content/50 text-center">
                  Showing {links.length} of {searchData.total} results
                </p>
              )}
            </div>
          )}
        </div>

        {/* Backlinks Panel */}
        <div className="space-y-4">
          <h3 className="font-medium text-base-content/70">
            {selectedRef ? `Backlinks to [[${selectedRef}]]` : 'Backlinks'}
          </h3>

          {!selectedRef ? (
            <div className="card bg-base-100 shadow-sm">
              <div className="card-body text-center py-12">
                <Link2 className="w-10 h-10 mx-auto text-base-content/30 mb-2" />
                <p className="text-base-content/60">Select a link to view its backlinks</p>
              </div>
            </div>
          ) : backlinksLoading ? (
            <div className="flex justify-center py-12">
              <Loader2 className="w-6 h-6 animate-spin text-primary" />
            </div>
          ) : !backlinksData?.backlinks?.length ? (
            <div className="card bg-base-100 shadow-sm">
              <div className="card-body text-center py-8">
                <p className="text-base-content/60">No backlinks found for [[{selectedRef}]]</p>
              </div>
            </div>
          ) : (
            <div className="space-y-2">
              {backlinksData.backlinks.map((backlink, idx) => (
                <div key={`${backlink.ref}-${idx}`} className="card bg-base-100 shadow-sm">
                  <div className="card-body py-3 px-4">
                    <div className="flex items-center gap-2">
                      <code className="text-secondary font-mono text-sm">[[{backlink.ref}]]</code>
                      <span className="badge badge-sm badge-ghost">{backlink.type}</span>
                    </div>
                    {backlink.title && (
                      <p className="text-sm font-medium mt-1">{backlink.title}</p>
                    )}
                    <div className="flex items-center gap-1 text-xs text-base-content/50 mt-1">
                      <FileText size={12} />
                      <span className="font-mono">{backlink.file}</span>
                      {backlink.line && <span>:{backlink.line}</span>}
                    </div>
                  </div>
                </div>
              ))}
              {backlinksData.total > backlinksData.backlinks.length && (
                <p className="text-sm text-base-content/50 text-center">
                  Showing {backlinksData.backlinks.length} of {backlinksData.total} backlinks
                </p>
              )}
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
