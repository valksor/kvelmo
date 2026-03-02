import { useState, useEffect } from 'react'
import ReactDiffViewer, { DiffMethod } from 'react-diff-viewer-continued'
import { useLayoutStore } from '../stores/layoutStore'
import { useProjectStore } from '../stores/projectStore'
import { OutputWidget } from './OutputWidget'
import { ChatWidget } from './ChatWidget'
import { ScreenshotsPanel } from './ScreenshotsPanel'
import { JobsPanel } from './JobsPanel'
import { FileBrowserWidget } from './FileBrowserWidget'
import { BrowserPanel } from './BrowserPanel'
import { TaskPanel } from './TaskPanel'
import { ReviewPanel } from './ReviewPanel'
import { FileChangesPanel } from './FileChangesPanel'

interface TabPanelProps {
  className?: string
}

export function TabPanel({ className = '' }: TabPanelProps) {
  const { tabs, activeTabId } = useLayoutStore()
  const activeTab = tabs.find((t) => t.id === activeTabId)

  if (!activeTab) {
    return (
      <div className={`flex items-center justify-center h-full text-base-content/50 ${className}`}>
        <div className="text-center">
          <svg className="w-12 h-12 mx-auto mb-3 opacity-30" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
          </svg>
          <p className="text-sm">No tab selected</p>
        </div>
      </div>
    )
  }

  return (
    <div className={`h-full overflow-hidden ${className}`}>
      {activeTab.type === 'output' && <OutputContent />}
      {activeTab.type === 'file' && <FileContent data={activeTab.data} />}
      {activeTab.type === 'diff' && <DiffContent data={activeTab.data} />}
      {activeTab.type === 'spec' && <SpecContent data={activeTab.data} />}
      {activeTab.type === 'agent' && <AgentContent data={activeTab.data} />}
      {activeTab.type === 'chat' && <ChatContent />}
      {activeTab.type === 'screenshots' && <ScreenshotsContent />}
      {activeTab.type === 'jobs' && <JobsContent />}
      {activeTab.type === 'files' && <FilesContent />}
      {activeTab.type === 'browser' && <BrowserContent />}
      {activeTab.type === 'task' && <TaskContent data={activeTab.data} />}
      {activeTab.type === 'review' && <ReviewContent data={activeTab.data} />}
      {activeTab.type === 'filechanges' && <FileChangesContent data={activeTab.data} />}
    </div>
  )
}

// Tab content components
function OutputContent() {
  return (
    <div className="h-full">
      <OutputWidget embedded />
    </div>
  )
}

function FileContent({ data }: { data?: Record<string, unknown> }) {
  const filePath = data?.path as string | undefined

  return (
    <div className="h-full p-4 overflow-auto">
      <div className="font-mono text-sm">
        {filePath ? (
          <div>
            <div className="text-base-content/60 mb-2">{filePath}</div>
            <pre className="bg-neutral text-neutral-content p-4 rounded-lg overflow-auto">
              {data?.content as string || 'Loading file content...'}
            </pre>
          </div>
        ) : (
          <div className="text-base-content/50">No file selected</div>
        )}
      </div>
    </div>
  )
}

function DiffContent({ data }: { data?: Record<string, unknown> }) {
  const filePath = data?.path as string | undefined
  const getGitDiff = useProjectStore((state) => state.getGitDiff)
  const [diff, setDiff] = useState<{ oldValue: string; newValue: string } | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  // Reactive theme state via MutationObserver
  const [isDarkTheme, setIsDarkTheme] = useState(() => {
    return document.documentElement.getAttribute('data-theme')?.includes('dark') ?? false
  })

  // Watch for theme changes
  useEffect(() => {
    const observer = new MutationObserver((mutations) => {
      for (const mutation of mutations) {
        if (mutation.attributeName === 'data-theme') {
          const theme = document.documentElement.getAttribute('data-theme')
          setIsDarkTheme(theme?.includes('dark') ?? false)
        }
      }
    })

    observer.observe(document.documentElement, { attributes: true })
    return () => observer.disconnect()
  }, [])

  // Fetch diff with cancellation guard
  useEffect(() => {
    if (!filePath) return

    let cancelled = false
    // Schedule state updates outside synchronous effect execution
    queueMicrotask(() => {
      if (cancelled) return
      setLoading(true)
      setError(null)
    })

    getGitDiff(false)
      .then((fullDiff) => {
        if (cancelled) return
        const parsed = parseDiffForFile(fullDiff, filePath)
        setDiff(parsed)
      })
      .catch((err) => {
        if (cancelled) return
        setError(err instanceof Error ? err.message : 'Failed to load diff')
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })

    return () => {
      cancelled = true
    }
  }, [filePath, getGitDiff])

  if (!filePath) {
    return (
      <div className="flex items-center justify-center h-full text-base-content/50">
        <p className="text-sm">No file selected</p>
      </div>
    )
  }

  if (loading) {
    return (
      <div className="flex items-center justify-center h-full">
        <span className="loading loading-spinner loading-md"></span>
      </div>
    )
  }

  if (error) {
    return (
      <div className="flex items-center justify-center h-full text-error">
        <p className="text-sm">{error}</p>
      </div>
    )
  }

  return (
    <div className="h-full overflow-auto">
      <div className="sticky top-0 z-10 bg-base-100 border-b border-base-300 px-4 py-2">
        <span className="text-sm font-medium">{filePath}</span>
        <span className="text-xs text-base-content/50 ml-2">
          ({data?.status as string || 'modified'})
        </span>
      </div>
      {diff ? (
        <ReactDiffViewer
          oldValue={diff.oldValue}
          newValue={diff.newValue}
          splitView={true}
          compareMethod={DiffMethod.WORDS}
          useDarkTheme={isDarkTheme}
          styles={{
            variables: {
              dark: {
                diffViewerBackground: 'var(--fallback-b1,oklch(var(--b1)/1))',
                addedBackground: 'oklch(var(--su)/0.2)',
                removedBackground: 'oklch(var(--er)/0.2)',
                wordAddedBackground: 'oklch(var(--su)/0.4)',
                wordRemovedBackground: 'oklch(var(--er)/0.4)',
                addedGutterBackground: 'oklch(var(--su)/0.3)',
                removedGutterBackground: 'oklch(var(--er)/0.3)',
                gutterBackground: 'var(--fallback-b2,oklch(var(--b2)/1))',
                gutterBackgroundDark: 'var(--fallback-b3,oklch(var(--b3)/1))',
                codeFoldGutterBackground: 'var(--fallback-b2,oklch(var(--b2)/1))',
                codeFoldBackground: 'var(--fallback-b2,oklch(var(--b2)/1))',
              },
              light: {
                diffViewerBackground: 'var(--fallback-b1,oklch(var(--b1)/1))',
                addedBackground: 'oklch(var(--su)/0.15)',
                removedBackground: 'oklch(var(--er)/0.15)',
                wordAddedBackground: 'oklch(var(--su)/0.3)',
                wordRemovedBackground: 'oklch(var(--er)/0.3)',
                addedGutterBackground: 'oklch(var(--su)/0.2)',
                removedGutterBackground: 'oklch(var(--er)/0.2)',
                gutterBackground: 'var(--fallback-b2,oklch(var(--b2)/1))',
                gutterBackgroundDark: 'var(--fallback-b3,oklch(var(--b3)/1))',
                codeFoldGutterBackground: 'var(--fallback-b2,oklch(var(--b2)/1))',
                codeFoldBackground: 'var(--fallback-b2,oklch(var(--b2)/1))',
              },
            },
          }}
        />
      ) : (
        <div className="flex items-center justify-center h-64 text-base-content/50">
          <p className="text-sm">No changes to display</p>
        </div>
      )}
    </div>
  )
}

// Parse unified diff to extract old/new content for a specific file
function parseDiffForFile(fullDiff: string, filePath: string): { oldValue: string; newValue: string } | null {
  if (!fullDiff) return null

  // Find the section for this file in the unified diff
  const lines = fullDiff.split('\n')
  let inTargetFile = false
  let oldLines: string[] = []
  let newLines: string[] = []

  for (const line of lines) {
    // Check for file header (diff --git a/path b/path)
    if (line.startsWith('diff --git')) {
      // Parse exact paths from "diff --git a/path b/path" header
      // Match pattern: diff --git a/<path> b/<path>
      const match = line.match(/^diff --git a\/(.+?) b\/(.+)$/)
      if (match) {
        const [, leftPath, rightPath] = match
        // Exact match instead of includes()
        inTargetFile = leftPath === filePath || rightPath === filePath
      } else {
        inTargetFile = false
      }
      if (inTargetFile) {
        oldLines = []
        newLines = []
      }
      continue
    }

    if (!inTargetFile) continue

    // Skip metadata lines
    if (line.startsWith('index ') || line.startsWith('--- ') || line.startsWith('+++ ') || line.startsWith('@@')) {
      continue
    }

    // Parse diff content
    if (line.startsWith('-')) {
      oldLines.push(line.slice(1))
    } else if (line.startsWith('+')) {
      newLines.push(line.slice(1))
    } else if (line.startsWith(' ')) {
      // Context line - appears in both
      oldLines.push(line.slice(1))
      newLines.push(line.slice(1))
    }
  }

  if (oldLines.length === 0 && newLines.length === 0) {
    return null
  }

  return {
    oldValue: oldLines.join('\n'),
    newValue: newLines.join('\n'),
  }
}

function SpecContent({ data }: { data?: Record<string, unknown> }) {
  const title = data?.title as string | undefined
  const content = data?.content as string | undefined

  return (
    <div className="h-full p-4 overflow-auto prose prose-sm max-w-none">
      {title && <h2>{title}</h2>}
      {content ? (
        <div className="whitespace-pre-wrap">{content}</div>
      ) : (
        <div className="text-base-content/50">No spec content</div>
      )}
    </div>
  )
}

function AgentContent({ data }: { data?: Record<string, unknown> }) {
  const agentId = data?.agentId as string | undefined
  const agentName = data?.name as string | undefined

  return (
    <div className="h-full p-4 overflow-auto">
      <div className="space-y-4">
        {agentName ? (
          <>
            <div className="flex items-center gap-3">
              <span className="status-dot status-running" />
              <span className="font-medium">{agentName}</span>
              <span className="text-xs text-base-content/50">ID: {agentId}</span>
            </div>
            <div className="bg-neutral text-neutral-content p-4 rounded-lg font-mono text-sm">
              Agent output will appear here...
            </div>
          </>
        ) : (
          <div className="text-base-content/50">No agent selected</div>
        )}
      </div>
    </div>
  )
}

function ChatContent() {
  return (
    <div className="h-full">
      <ChatWidget embedded />
    </div>
  )
}

function ScreenshotsContent() {
  return (
    <div className="h-full">
      <ScreenshotsPanel />
    </div>
  )
}

function JobsContent() {
  return (
    <div className="h-full">
      <JobsPanel />
    </div>
  )
}

function FilesContent() {
  return (
    <div className="h-full">
      <FileBrowserWidget />
    </div>
  )
}

function BrowserContent() {
  return (
    <div className="h-full">
      <BrowserPanel />
    </div>
  )
}

function TaskContent({ data }: { data?: Record<string, unknown> }) {
  return (
    <div className="h-full">
      <TaskPanel data={data} />
    </div>
  )
}

function ReviewContent({ data }: { data?: Record<string, unknown> }) {
  return (
    <div className="h-full">
      <ReviewPanel data={data} />
    </div>
  )
}

function FileChangesContent({ data }: { data?: Record<string, unknown> }) {
  return (
    <div className="h-full">
      <FileChangesPanel data={data} />
    </div>
  )
}
