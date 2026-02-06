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
      <div className="flex items-center justify-center min-h-[400px]">
        <Loader2 className="w-8 h-8 animate-spin text-primary" />
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

      <div className="tabs tabs-boxed bg-base-200 p-1 inline-flex">
        <button
          className={`tab gap-2 ${activeTab === 'create' ? 'tab-active' : ''}`}
          onClick={() => setActiveTab('create')}
        >
          <FilePlus2 size={16} />
          Create Plan
        </button>
        <button
          className={`tab gap-2 ${activeTab === 'queues' ? 'tab-active' : ''}`}
          onClick={() => setActiveTab('queues')}
        >
          <FolderKanban size={16} />
          Queues
        </button>
      </div>

      <div className="card bg-base-100 shadow-sm">
        <div className="card-body">
          {activeTab === 'create' && (
            <div className="max-w-2xl">
              <ProjectPlanForm
                allowAdvancedSources
                description="Create a new project plan from a source file, directory, or provider reference."
              />
            </div>
          )}

          {activeTab === 'queues' && (
            <div className="space-y-4">
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
