import { useState, useMemo } from 'react'
import { useGlobalStore } from '../stores/globalStore'
import { useProjectStore } from '../stores/projectStore'
import { useLayoutStore } from '../stores/layoutStore'
import { useDocsURL } from '../hooks/useDocsURL'
import { Widget, TaskIcon, FilesIcon, ActionsIcon, CheckpointsIcon } from './Widget'
import { PanelLayout } from './PanelLayout'
import { TaskWidget } from './TaskWidget'
import { ActionsWidget } from './ActionsWidget'
import { CheckpointsWidget } from './CheckpointsWidget'
import { ReviewHistoryWidget } from './ReviewHistoryWidget'
import { FileChangesWidget } from './FileChangesWidget'
import { AgentPanel } from './AgentPanel'
import { ThemeToggle } from './ThemeToggle'
import { StatusBadge } from './StatusIndicator'
import { Settings } from './Settings'
import { LogsPanel } from './LogsPanel'

function ReviewIcon() {
  return (
    <svg aria-hidden="true" className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-6 9l2 2 4-4" />
    </svg>
  )
}

// State labels - defined at module level to avoid recreation on every render
const STATE_LABELS: Record<string, string> = {
  'none': 'No Task',
  'loaded': 'Ready',
  'planning': 'Planning...',
  'planned': 'Planned',
  'implementing': 'Implementing...',
  'implemented': 'Implemented',
  'reviewing': 'Reviewing...',
  'reviewed': 'Reviewed',
  'submitted': 'Submitted',
  'failed': 'Failed',
}

export function ProjectView() {
  const { selectedProject, selectProject } = useGlobalStore()
  const { task, state, fileChanges, reviews, output } = useProjectStore()
  const { widgetStates } = useLayoutStore()
  const [showSettings, setShowSettings] = useState(false)
  const [showLogs, setShowLogs] = useState(false)
  const docsData = useDocsURL()

  // Memoize status type to avoid recalculation on every render
  // Must be before early return to satisfy React hooks rules
  const statusType = useMemo(() => {
    if (!task?.state || state === 'none') return 'idle'
    if (state === 'implemented' || state === 'submitted') return 'success'
    if (state === 'planning' || state === 'implementing' || state === 'reviewing') return 'running'
    if (state === 'failed') return 'error'
    return 'idle'
  }, [task?.state, state])

  // Memoize state label
  const stateLabel = useMemo(() => {
    return STATE_LABELS[state || 'none'] || state || 'No Task'
  }, [state])

  if (!selectedProject) return null

  const projectName = selectedProject.path.split('/').pop() || 'Project'

  // Header component
  const header = (
    <header className="header-enhanced sticky top-0 z-10">
      <div className="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-2 sm:gap-0 px-3 sm:px-4 py-2 sm:py-3">
        <div className="flex items-center gap-2 sm:gap-3 w-full sm:w-auto">
          <button
            onClick={() => selectProject(null)}
            aria-label="Back to projects"
            className="flex items-center gap-1 sm:gap-1.5 text-xs sm:text-sm text-base-content/60 hover:text-base-content transition-colors"
          >
            <svg aria-hidden="true" className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 19l-7-7 7-7" />
            </svg>
            <span className="hidden sm:inline">Projects</span>
          </button>
          <div className="w-px h-4 sm:h-5 bg-base-300" />
          <div className="flex items-center gap-2 min-w-0 flex-1">
            <div aria-hidden="true" className="w-6 h-6 sm:w-7 sm:h-7 rounded-lg bg-primary flex items-center justify-center text-primary-content font-semibold text-xs sm:text-sm flex-shrink-0">
              {projectName[0].toUpperCase()}
            </div>
            <div className="min-w-0">
              <h1 className="font-medium text-xs sm:text-sm text-base-content truncate">{projectName}</h1>
              <p className="text-[10px] sm:text-xs text-base-content/50 font-mono truncate max-w-[120px] sm:max-w-[200px]">{selectedProject.path}</p>
            </div>
          </div>
        </div>

        <div className="flex items-center gap-2 sm:gap-3 w-full sm:w-auto justify-between sm:justify-end">
          <div className="flex items-center gap-1 sm:gap-2">
            {/* Documentation link */}
            {docsData?.url && (
              <a
                href={docsData.url}
                target="_blank"
                rel="noopener noreferrer"
                className="btn btn-ghost btn-xs sm:btn-sm btn-circle"
                aria-label="Documentation"
              >
                <svg aria-hidden="true" className="w-4 h-4 sm:w-5 sm:h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8.228 9c.549-1.165 2.03-2 3.772-2 2.21 0 4 1.343 4 3 0 1.4-1.278 2.575-3.006 2.907-.542.104-.994.54-.994 1.093m0 3h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                </svg>
              </a>
            )}
            <button
              onClick={() => setShowLogs(true)}
              className="btn btn-ghost btn-xs sm:btn-sm btn-circle relative"
              aria-label="View logs"
            >
              <svg aria-hidden="true" className="w-4 h-4 sm:w-5 sm:h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6h16M4 12h16M4 18h7" />
              </svg>
              {output.length > 0 && (
                <span className="absolute -top-1 -right-1 w-2 h-2 bg-primary rounded-full" />
              )}
            </button>
            <button
              onClick={() => setShowSettings(true)}
              className="btn btn-ghost btn-xs sm:btn-sm btn-circle"
              aria-label="Settings"
            >
              <svg aria-hidden="true" className="w-4 h-4 sm:w-5 sm:h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
              </svg>
            </button>
            <ThemeToggle />
          </div>
          <StatusBadge status={statusType} label={stateLabel} />
        </div>
      </div>
    </header>
  )

  // Left sidebar content
  const leftContent = (
    <>
      <Widget
        id="task"
        title="Task"
        icon={<TaskIcon />}
        defaultCollapsed={widgetStates.task?.collapsed}
      >
        <TaskWidget embedded />
      </Widget>

      <Widget
        id="files"
        title="File Changes"
        icon={<FilesIcon />}
        defaultCollapsed={widgetStates.files?.collapsed}
        actions={
          fileChanges && fileChanges.length > 0 ? (
            <span className="text-xs text-base-content/50">{fileChanges.length}</span>
          ) : null
        }
      >
        <FileChangesWidget embedded />
      </Widget>
    </>
  )

  // Right panel content
  const rightContent = (
    <>
      <AgentPanel />

      <Widget
        id="actions"
        title="Actions"
        icon={<ActionsIcon />}
        defaultCollapsed={widgetStates.actions?.collapsed}
      >
        <ActionsWidget embedded />
      </Widget>

      <Widget
        id="checkpoints"
        title="Checkpoints"
        icon={<CheckpointsIcon />}
        defaultCollapsed={widgetStates.checkpoints?.collapsed}
      >
        <CheckpointsWidget embedded />
      </Widget>

      {/* Review History widget — defaultCollapsed, not in layout store */}
      <Widget
        id="agents"
        title="Review History"
        icon={<ReviewIcon />}
        defaultCollapsed={true}
        actions={
          reviews && reviews.length > 0 ? (
            <span className="text-xs text-base-content/50">{reviews.length}</span>
          ) : null
        }
      >
        <ReviewHistoryWidget embedded />
      </Widget>
    </>
  )

  return (
    <div className="h-screen flex flex-col bg-base-100">
      <PanelLayout
        header={header}
        leftContent={leftContent}
        rightContent={rightContent}
      />
      <Settings
        isOpen={showSettings}
        onClose={() => setShowSettings(false)}
        defaultScope="project"
      />
      <LogsPanel
        isOpen={showLogs}
        onClose={() => setShowLogs(false)}
      />
    </div>
  )
}
