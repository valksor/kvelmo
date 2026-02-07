import { useId, useState } from 'react'
import { Loader2, BookOpen, FileText, AlertCircle, Search, Tag } from 'lucide-react'
import { useLibraryCollections, useLibraryItems, type LibraryItem } from '@/api/library'
import { useStatus } from '@/api/workflow'
import { ProjectSelector } from '@/components/project/ProjectSelector'

export default function Library() {
  const { data: status, isLoading: statusLoading } = useStatus()
  const collectionSearchInputID = useId()
  const [selectedCollection, setSelectedCollection] = useState<string | undefined>()
  const [selectedItem, setSelectedItem] = useState<LibraryItem | null>(null)
  const [searchQuery, setSearchQuery] = useState('')

  const { data: collectionsData, isLoading: collectionsLoading, error: collectionsError } =
    useLibraryCollections()
  const { data: itemsData, isLoading: itemsLoading } = useLibraryItems(selectedCollection)

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
        <h1 className="text-2xl font-bold">Specification Library</h1>
        <ProjectSelector />
      </div>
    )
  }

  // Check if library is disabled
  if (collectionsData && !collectionsData.enabled) {
    return (
      <div className="space-y-6">
        <h1 className="text-2xl font-bold">Specification Library</h1>
        <div className="card bg-base-100 shadow-sm">
          <div className="card-body text-center py-12">
            <BookOpen aria-hidden="true" className="w-12 h-12 mx-auto text-base-content/30 mb-4" />
            <h2 className="text-lg font-medium">Library Disabled</h2>
            <p className="text-base-content/60 mt-2">
              Enable the library feature in Settings to use specification storage.
            </p>
          </div>
        </div>
      </div>
    )
  }

  const collections = collectionsData?.collections || []
  const items = itemsData?.items || []

  // Filter items by search query
  const filteredItems = searchQuery
    ? items.filter(
        (item) =>
          item.title.toLowerCase().includes(searchQuery.toLowerCase()) ||
          item.content.toLowerCase().includes(searchQuery.toLowerCase())
      )
    : items

  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <h1 className="text-2xl font-bold">Specification Library</h1>
        <p className="text-base-content/60 mt-1">
          Browse and search stored specifications and documentation
        </p>
      </div>

      {collectionsError && (
        <div className="alert alert-error">
          <AlertCircle size={18} aria-hidden="true" />
          <span>Failed to load library: {collectionsError.message}</span>
        </div>
      )}

      {/* Main layout */}
      <div className="grid grid-cols-1 lg:grid-cols-4 gap-6">
        {/* Sidebar: Collections */}
        <div className="lg:col-span-1">
          <div className="card bg-base-100 shadow-sm">
            <div className="card-body py-4">
              <h3 className="font-medium text-sm text-base-content/70 mb-2">Collections</h3>
              {collectionsLoading ? (
                <div className="flex justify-center py-4">
                  <Loader2 aria-hidden="true" className="w-5 h-5 animate-spin" />
                </div>
              ) : collections.length === 0 ? (
                <p className="text-sm text-base-content/50 text-center py-4">
                  No collections yet
                </p>
              ) : (
                <div className="space-y-1">
                  {collections.map((collection) => (
                    <button
                      key={collection.id}
                      onClick={() => {
                        setSelectedCollection(collection.id)
                        setSelectedItem(null)
                      }}
                      className={`w-full text-left px-3 py-2 rounded-lg transition-colors ${
                        selectedCollection === collection.id
                          ? 'bg-primary/10 text-primary'
                          : 'hover:bg-base-200'
                      }`}
                    >
                      <div className="flex items-center justify-between">
                        <span className="font-medium text-sm truncate">{collection.name}</span>
                        <span className="badge badge-sm badge-ghost">{collection.item_count}</span>
                      </div>
                      {collection.description && (
                        <p className="text-xs text-base-content/50 truncate mt-0.5">
                          {collection.description}
                        </p>
                      )}
                    </button>
                  ))}
                </div>
              )}
            </div>
          </div>
        </div>

        {/* Main: Items list or detail view */}
        <div className="lg:col-span-3">
          {selectedItem ? (
            // Item detail view
            <div className="card bg-base-100 shadow-sm">
              <div className="card-body">
                <div className="flex items-start justify-between">
                  <div>
                    <h2 className="text-xl font-bold">{selectedItem.title}</h2>
                    {selectedItem.tags && selectedItem.tags.length > 0 && (
                      <div className="flex items-center gap-1 mt-2">
                        <Tag size={14} aria-hidden="true" className="text-base-content/50" />
                        {selectedItem.tags.map((tag) => (
                          <span key={tag} className="badge badge-sm badge-ghost">
                            {tag}
                          </span>
                        ))}
                      </div>
                    )}
                  </div>
                  <button
                    className="btn btn-ghost btn-sm"
                    onClick={() => setSelectedItem(null)}
                  >
                    Back to list
                  </button>
                </div>
                <div className="divider" />
                <div className="prose prose-sm max-w-none">
                  <MarkdownContent content={selectedItem.content} />
                </div>
              </div>
            </div>
          ) : selectedCollection ? (
            // Items list
            <div className="space-y-4">
              {/* Search within collection */}
              <div className="relative">
                <label htmlFor={collectionSearchInputID} className="sr-only">
                  Search items in selected collection
                </label>
                <Search
                  size={16}
                  aria-hidden="true"
                  className="absolute left-3 top-1/2 -translate-y-1/2 text-base-content/50"
                />
                <input
                  id={collectionSearchInputID}
                  type="text"
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                  placeholder="Search in collection..."
                  className="input input-bordered w-full pl-10"
                />
              </div>

              {itemsLoading ? (
                <div className="flex justify-center py-12">
                  <Loader2 aria-hidden="true" className="w-6 h-6 animate-spin text-primary" />
                </div>
              ) : filteredItems.length === 0 ? (
                <div className="card bg-base-100 shadow-sm">
                  <div className="card-body text-center py-12">
                    <FileText aria-hidden="true" className="w-10 h-10 mx-auto text-base-content/30 mb-2" />
                    <p className="text-base-content/60">
                      {searchQuery ? 'No matching items' : 'No items in this collection'}
                    </p>
                  </div>
                </div>
              ) : (
                <div className="grid gap-3">
                  {filteredItems.map((item) => (
                    <div
                      key={item.id}
                      role="button"
                      tabIndex={0}
                      className="card bg-base-100 shadow-sm cursor-pointer hover:shadow-md transition-shadow"
                      onClick={() => setSelectedItem(item)}
                      onKeyDown={(e) => {
                        if (e.key === 'Enter' || e.key === ' ') {
                          e.preventDefault()
                          setSelectedItem(item)
                        }
                      }}
                    >
                      <div className="card-body py-3 px-4">
                        <div className="flex items-start justify-between">
                          <div>
                            <h3 className="font-medium">{item.title}</h3>
                            <p className="text-sm text-base-content/60 line-clamp-2 mt-1">
                              {item.content.slice(0, 150)}
                              {item.content.length > 150 ? '...' : ''}
                            </p>
                          </div>
                          <FileText size={18} aria-hidden="true" className="text-base-content/30 shrink-0" />
                        </div>
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </div>
          ) : (
            // No collection selected
            <div className="card bg-base-100 shadow-sm">
              <div className="card-body text-center py-12">
                <BookOpen aria-hidden="true" className="w-12 h-12 mx-auto text-base-content/30 mb-4" />
                <p className="text-base-content/60">Select a collection to view its items</p>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}

/**
 * Simple markdown content renderer
 * For full markdown support, consider using react-markdown
 */
function MarkdownContent({ content }: { content: string }) {
  // Basic conversion: code blocks, headers, lists
  const lines = content.split('\n')
  const elements: React.ReactNode[] = []
  let inCodeBlock = false
  let codeContent: string[] = []

  lines.forEach((line, idx) => {
    if (line.startsWith('```')) {
      if (inCodeBlock) {
        // End code block
        elements.push(
          <pre key={`code-${idx}`} className="bg-base-200 p-3 rounded-lg overflow-x-auto">
            <code>{codeContent.join('\n')}</code>
          </pre>
        )
        codeContent = []
        inCodeBlock = false
      } else {
        // Start code block (language hint ignored for now)
        inCodeBlock = true
      }
    } else if (inCodeBlock) {
      codeContent.push(line)
    } else if (line.startsWith('# ')) {
      elements.push(
        <h1 key={idx} className="text-2xl font-bold mt-4 mb-2">
          {line.slice(2)}
        </h1>
      )
    } else if (line.startsWith('## ')) {
      elements.push(
        <h2 key={idx} className="text-xl font-bold mt-4 mb-2">
          {line.slice(3)}
        </h2>
      )
    } else if (line.startsWith('### ')) {
      elements.push(
        <h3 key={idx} className="text-lg font-bold mt-3 mb-1">
          {line.slice(4)}
        </h3>
      )
    } else if (line.startsWith('- ') || line.startsWith('* ')) {
      elements.push(
        <li key={idx} className="ml-4">
          {line.slice(2)}
        </li>
      )
    } else if (line.trim() === '') {
      elements.push(<br key={idx} />)
    } else {
      elements.push(
        <p key={idx} className="my-1">
          {line}
        </p>
      )
    }
  })

  return <>{elements}</>
}
