import { useState, useMemo, useRef, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { Globe, FolderOpen, Trash2, Search, ArrowRightLeft, ChevronDown } from 'lucide-react'
import type { OverrideScope } from './TerminologyEditor'

export interface KeyOverrideEntry {
  key: string
  value: string
  language: string
  scope: OverrideScope
}

interface KeyOverrideEditorProps {
  /** Combined key override entries from both scopes */
  entries: KeyOverrideEntry[]
  /** Called when entries change */
  onChange: (entries: KeyOverrideEntry[]) => void
  /** Available translation keys for autocomplete */
  availableKeys: string[]
  /** Current project name for display */
  projectName?: string
  /** Whether project context is available */
  hasProject: boolean
  /** Current language code */
  currentLanguage: string
}

/**
 * Editor for direct translation key overrides.
 * Allows power users to override any specific translation key.
 */
export function KeyOverrideEditor({
  entries,
  onChange,
  availableKeys,
  projectName,
  hasProject,
  currentLanguage,
}: KeyOverrideEditorProps) {
  const { t } = useTranslation()
  const [newKey, setNewKey] = useState('')
  const [newValue, setNewValue] = useState('')
  const [searchQuery, setSearchQuery] = useState('')
  const [isDropdownOpen, setIsDropdownOpen] = useState(false)
  const dropdownRef = useRef<HTMLDivElement>(null)
  const inputRef = useRef<HTMLInputElement>(null)

  // Build key-value pairs for the selector
  const keyValuePairs = useMemo(() => {
    return availableKeys.map((key) => ({
      key,
      // Get current translation value - handle namespaced keys
      currentValue: t(key, { defaultValue: key }),
    }))
  }, [availableKeys, t])

  // Filter keys for the dropdown - search by key OR current value
  const filteredKeys = useMemo(() => {
    const query = newKey.toLowerCase().trim()
    return keyValuePairs
      .filter((pair) => {
        if (!query) return true // Show all when empty
        return (
          pair.key.toLowerCase().includes(query) ||
          pair.currentValue.toLowerCase().includes(query)
        )
      })
      .filter((pair) => !entries.some((e) => e.key === pair.key))
      .slice(0, 15)
  }, [newKey, keyValuePairs, entries])

  // Close dropdown when clicking outside
  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (dropdownRef.current && !dropdownRef.current.contains(event.target as Node)) {
        setIsDropdownOpen(false)
      }
    }
    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [])

  // Filter displayed entries
  const displayedEntries = useMemo(() => {
    if (!searchQuery.trim()) return entries
    const query = searchQuery.toLowerCase()
    return entries.filter(
      (e) =>
        e.key.toLowerCase().includes(query) ||
        e.value.toLowerCase().includes(query)
    )
  }, [entries, searchQuery])

  const handleAdd = (scope: OverrideScope) => {
    if (!newKey.trim() || !newValue.trim()) return

    // Check for duplicate key in same scope
    const exists = entries.some((e) => e.key === newKey.trim() && e.scope === scope)
    if (exists) return

    onChange([
      ...entries,
      {
        key: newKey.trim(),
        value: newValue.trim(),
        language: currentLanguage,
        scope,
      },
    ])
    setNewKey('')
    setNewValue('')
  }

  const handleRemove = (index: number) => {
    const actualIndex = entries.indexOf(displayedEntries[index])
    if (actualIndex !== -1) {
      onChange(entries.filter((_, i) => i !== actualIndex))
    }
  }

  const handleScopeChange = (index: number) => {
    const actualIndex = entries.indexOf(displayedEntries[index])
    if (actualIndex !== -1) {
      const updated = entries.map((entry, i) => {
        if (i === actualIndex) {
          return { ...entry, scope: entry.scope === 'global' ? 'project' : 'global' } as KeyOverrideEntry
        }
        return entry
      })
      onChange(updated)
    }
  }

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter') {
      e.preventDefault()
      setIsDropdownOpen(false)
      // Default to project scope if available, otherwise global
      handleAdd(hasProject ? 'project' : 'global')
    } else if (e.key === 'Escape') {
      setIsDropdownOpen(false)
    }
  }

  const handleSelectKey = (key: string, currentValue: string) => {
    setNewKey(key)
    setIsDropdownOpen(false)
    // Pre-fill with current value as starting point
    if (!newValue) {
      setNewValue(currentValue)
    }
  }

  // Get current value for selected key
  const selectedKeyValue = useMemo(() => {
    if (!newKey) return null
    const pair = keyValuePairs.find((p) => p.key === newKey)
    return pair?.currentValue ?? null
  }, [newKey, keyValuePairs])

  return (
    <div className="space-y-3">
      {/* Search filter */}
      {entries.length > 5 && (
        <div className="relative">
          <Search
            size={16}
            className="absolute left-3 top-1/2 -translate-y-1/2 text-base-content/40"
            aria-hidden="true"
          />
          <input
            type="text"
            className="input input-bordered input-sm w-full pl-9"
            placeholder="Filter overrides..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
          />
        </div>
      )}

      {/* Entry list */}
      {displayedEntries.length > 0 && (
        <div className="overflow-x-auto">
          <table className="table table-sm w-full">
            <thead>
              <tr>
                <th className="w-2/5">Key</th>
                <th className="w-2/5">Override Value</th>
                <th className="w-1/5">Scope</th>
                <th className="w-10"></th>
              </tr>
            </thead>
            <tbody>
              {displayedEntries.map((entry, index) => (
                <tr key={`${entry.scope}-${entry.key}`} className="hover">
                  <td className="font-mono text-xs">{entry.key}</td>
                  <td className="text-sm">{entry.value}</td>
                  <td>
                    <ScopeIndicator
                      scope={entry.scope}
                      projectName={projectName}
                      canToggle={hasProject}
                      onToggle={() => handleScopeChange(index)}
                    />
                  </td>
                  <td>
                    <button
                      type="button"
                      className="btn btn-ghost btn-xs text-error"
                      onClick={() => handleRemove(index)}
                      aria-label={`Remove ${entry.key} override`}
                    >
                      <Trash2 size={14} aria-hidden="true" />
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {/* Empty state */}
      {entries.length === 0 && (
        <div className="text-center py-6 text-base-content/60">
          <p className="text-sm">No key overrides configured.</p>
          <p className="text-xs mt-1">
            Override specific translations like "nav.dashboard" → "Home".
          </p>
        </div>
      )}

      {/* Add new entry */}
      <div className="flex flex-wrap gap-2 items-end pt-2 border-t border-base-300/50">
        {/* Key selector with searchable dropdown */}
        <div className="flex-1 min-w-[280px] relative" ref={dropdownRef}>
          <label htmlFor="key-override-key" className="label py-1">
            <span className="label-text text-xs">Search by key or current text</span>
          </label>
          <div className="relative">
            <input
              ref={inputRef}
              id="key-override-key"
              type="text"
              className="input input-bordered input-sm w-full font-mono text-xs pr-8"
              placeholder="Type to search... (e.g., Dashboard or nav.)"
              value={newKey}
              onChange={(e) => {
                setNewKey(e.target.value)
                setIsDropdownOpen(true)
              }}
              onFocus={() => setIsDropdownOpen(true)}
              onKeyDown={handleKeyDown}
              autoComplete="off"
            />
            <button
              type="button"
              className="absolute right-2 top-1/2 -translate-y-1/2 text-base-content/40 hover:text-base-content/70"
              onClick={() => setIsDropdownOpen(!isDropdownOpen)}
              aria-label="Toggle dropdown"
            >
              <ChevronDown size={14} aria-hidden="true" />
            </button>
          </div>
          {/* Dropdown with key + current value */}
          {isDropdownOpen && filteredKeys.length > 0 && (
            <div className="absolute z-50 w-full mt-1 bg-base-100 border border-base-300 rounded-lg shadow-lg max-h-60 overflow-y-auto">
              <table className="table table-xs w-full">
                <thead className="sticky top-0 bg-base-200">
                  <tr>
                    <th className="text-xs">Key</th>
                    <th className="text-xs">Current Text</th>
                  </tr>
                </thead>
                <tbody>
                  {filteredKeys.map((pair) => (
                    <tr
                      key={pair.key}
                      className="hover cursor-pointer"
                      onClick={() => handleSelectKey(pair.key, pair.currentValue)}
                    >
                      <td className="font-mono text-xs text-base-content/70">{pair.key}</td>
                      <td className="text-sm">{pair.currentValue}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
          {/* Show current value when a key is selected */}
          {selectedKeyValue && newKey && (
            <div className="text-xs text-base-content/60 mt-1">
              Currently shows: <span className="font-medium">"{selectedKeyValue}"</span>
            </div>
          )}
        </div>
        <div className="flex-1 min-w-[150px]">
          <label htmlFor="key-override-value" className="label py-1">
            <span className="label-text text-xs">New Value</span>
          </label>
          <input
            id="key-override-value"
            type="text"
            className="input input-bordered input-sm w-full"
            placeholder="Your custom text"
            value={newValue}
            onChange={(e) => setNewValue(e.target.value)}
            onKeyDown={handleKeyDown}
          />
        </div>
        <div className="flex gap-1 items-end">
          <button
            type="button"
            className="btn btn-ghost btn-sm gap-1"
            onClick={() => handleAdd('global')}
            disabled={!newKey.trim() || !newValue.trim()}
          >
            <Globe size={14} aria-hidden="true" />
            Add Global
          </button>
          {hasProject && (
            <button
              type="button"
              className="btn btn-primary btn-sm gap-1"
              onClick={() => handleAdd('project')}
              disabled={!newKey.trim() || !newValue.trim()}
            >
              <FolderOpen size={14} aria-hidden="true" />
              Add to Project
            </button>
          )}
        </div>
      </div>
    </div>
  )
}

interface ScopeIndicatorProps {
  scope: OverrideScope
  projectName?: string
  canToggle?: boolean
  onToggle?: () => void
}

function ScopeIndicator({ scope, projectName, canToggle, onToggle }: ScopeIndicatorProps) {
  const isProject = scope === 'project'
  const icon = isProject ? <FolderOpen size={12} aria-hidden="true" /> : <Globe size={12} aria-hidden="true" />
  const label = isProject ? 'Project' : 'Global'
  const title = isProject ? `Project: ${projectName ?? 'Current'}` : 'Applies to all projects'
  const badgeClass = isProject ? 'badge-primary' : 'badge-ghost'

  if (canToggle && onToggle) {
    const targetScope = isProject ? 'global' : 'project'
    return (
      <button
        type="button"
        className={`badge badge-sm ${badgeClass} gap-1 cursor-pointer hover:opacity-80 transition-opacity group`}
        title={`Click to move to ${targetScope}`}
        aria-label={`Move to ${targetScope} scope`}
        onClick={onToggle}
      >
        {icon}
        {label}
        <ArrowRightLeft size={10} className="opacity-0 group-hover:opacity-100 transition-opacity" aria-hidden="true" />
      </button>
    )
  }

  return (
    <span className={`badge badge-sm ${badgeClass} gap-1`} title={title}>
      {icon}
      {label}
    </span>
  )
}
