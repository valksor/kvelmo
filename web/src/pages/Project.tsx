import { useState } from 'react'
import { Link } from 'react-router-dom'
import { useStatus } from '@/api/workflow'
import { ProjectSelector } from '@/components/project/ProjectSelector'
import { QueuesPanel } from '@/components/project/QueuesPanel'
import { ProjectPlanForm } from '@/components/project/ProjectPlanForm'
import { FolderKanban, FilePlus2, Loader2 } from 'lucide-react'

type Tab = 'create' | 'queues'

export default function Project() {
  const { data: status, isLoading: statusLoading } = useStatus()
  const [activeTab, setActiveTab] = useState<Tab>('create')
  const [selectedQueueId, setSelectedQueueId] = useState<string | undefined>()

  if (statusLoading) {
    return (
      <div className="flex items-center justify-center min-h-[400px]" role="status" aria-label="Loading">
        <Loader2 aria-hidden="true" className="w-8 h-8 animate-spin text-primary" />
      </div>
    )
  }

  if (status?.mode === 'global') {
    return (
      <div className="space-y-4">
        <h1 className="text-2xl font-bold">Project Planning</h1>
        <ProjectSelector />
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">Project Planning</h1>
      </div>

      <div className="card bg-base-100 shadow-sm border border-base-300/70">
        <div className="card-body p-2 sm:p-3">
          <div role="tablist" aria-label="Project planning options" className="grid grid-cols-1 sm:grid-cols-2 gap-2">
            <button
              type="button"
              role="tab"
              id="tab-create"
              aria-selected={activeTab === 'create'}
              aria-controls="tabpanel-create"
              className={`tab h-auto rounded-xl border px-4 py-3 text-left transition-colors ${
                activeTab === 'create'
                  ? 'border-primary bg-primary/10 shadow-sm'
                  : 'border-base-300 bg-base-100 hover:bg-base-200/60'
              }`}
              onClick={() => setActiveTab('create')}
            >
              <div className="flex items-start gap-3">
                <FilePlus2 size={18} className={activeTab === 'create' ? 'text-primary' : 'text-base-content/60'} aria-hidden="true" />
                <div className="space-y-1">
                  <p className="font-semibold">Create Plan</p>
                  <p className="text-xs text-base-content/65">Start from files, directories, or providers</p>
                </div>
              </div>
            </button>
            <button
              type="button"
              role="tab"
              id="tab-queues"
              aria-selected={activeTab === 'queues'}
              aria-controls="tabpanel-queues"
              className={`tab h-auto rounded-xl border px-4 py-3 text-left transition-colors ${
                activeTab === 'queues'
                  ? 'border-primary bg-primary/10 shadow-sm'
                  : 'border-base-300 bg-base-100 hover:bg-base-200/60'
              }`}
              onClick={() => setActiveTab('queues')}
            >
              <div className="flex items-start gap-3">
                <FolderKanban size={18} className={activeTab === 'queues' ? 'text-primary' : 'text-base-content/60'} aria-hidden="true" />
                <div className="space-y-1">
                  <p className="font-semibold">Queues</p>
                  <p className="text-xs text-base-content/65">Browse and manage existing plan queues</p>
                </div>
              </div>
            </button>
          </div>
        </div>
      </div>

      <div className="card bg-base-100 shadow-sm">
        <div className="card-body">
          {activeTab === 'create' && (
            <div role="tabpanel" id="tabpanel-create" aria-labelledby="tab-create" className="max-w-2xl">
              <ProjectPlanForm
                allowAdvancedSources
                description="Create a new project plan from a source file, directory, or provider reference."
              />
            </div>
          )}

          {activeTab === 'queues' && (
            <div role="tabpanel" id="tabpanel-queues" aria-labelledby="tab-queues" className="space-y-4">
              <div className="alert alert-info">
                Manage queue tasks in{' '}
                <Link to="/" className="link link-primary">
                  Dashboard - Tasks - Queue
                </Link>
                .
              </div>
              <QueuesPanel
                onSelectQueue={setSelectedQueueId}
                selectedQueueId={selectedQueueId}
              />
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
