import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import { SocketClient } from '../lib/socket'
import { storeName } from '../meta'
import type {
  WorktreeInfo,
  WorkerInfo,
  WorkersStats,
  MemoryStatsResponse,
  TaskListSummary,
} from '../types/socket'

// Re-export types used by components
export type Project = WorktreeInfo
export type Worker = WorkerInfo
export type WorkerStats = WorkersStats
export type MemoryStats = MemoryStatsResponse
export type TaskSummary = TaskListSummary

// Types not yet in Go (UI-only or need backend work)
export interface Job {
  id: string
  type: string
  status: string
  worktree_id: string
  created_at: string
  updated_at?: string
  error?: string
  result?: Record<string, unknown>
}

export interface MemoryResult {
  id: string
  type: string
  content: string
  score: number
  task_id: string
  created_at: string
}

interface GlobalState {
  // Connection
  connected: boolean
  connecting: boolean
  client: SocketClient | null

  // Data
  projects: Project[]
  workers: Worker[]
  workerStats: WorkerStats | null
  memoryStats: MemoryStats | null
  selectedProjectId: string | null
  selectedProject: Project | null
  loading: boolean
  error: string | null

  // Active tasks across all projects
  activeTasks: TaskSummary[]

  // Jobs
  jobs: Job[]

  // Connection
  connect: () => Promise<void>
  disconnect: () => void

  // Projects
  loadProjects: () => Promise<void>
  addProject: (path: string) => Promise<void>
  removeProject: (id: string) => Promise<void>
  selectProject: (project: Project | null) => void

  // Workers
  loadWorkers: () => Promise<void>
  loadWorkerStats: () => Promise<void>
  addWorker: (agent: string) => Promise<void>
  removeWorker: (id: string) => Promise<void>

  // Chat
  stopChat: (worktreeId: string, jobId?: string) => Promise<void>

  // Tasks
  loadActiveTasks: () => Promise<void>

  // Memory
  searchMemory: (query: string, limit?: number) => Promise<MemoryResult[]>
  loadMemoryStats: () => Promise<void>
  clearMemory: () => Promise<void>

  // Jobs
  loadJobs: () => Promise<void>
  loadJob: (id: string) => Promise<Job | null>
}

export const useGlobalStore = create<GlobalState>()(
  persist(
    (set, get) => ({
      connected: false,
      connecting: false,
      client: null,

      projects: [],
      workers: [],
      workerStats: null,
      memoryStats: null,
      jobs: [],
      selectedProjectId: null,
      selectedProject: null,
      loading: false,
      error: null,
      activeTasks: [],

      connect: async () => {
        if (get().connected || get().connecting) return

        set({ connecting: true, error: null })

        // Support dynamic port injection for Tauri desktop app
        // In dev mode (Vite), connect directly to Go server on 6337
        // In production (Tauri), use ?port= param or window.location.host
        const getBackendHost = () => {
          const params = new URLSearchParams(window.location.search)
          const port = params.get('port')
          if (port) return `localhost:${port}`
          // Dev mode: Vite runs on different port, connect directly to Go server
          if (import.meta.env.DEV) return 'localhost:6337'
          return window.location.host
        }

        const url = `ws://${getBackendHost()}/ws/global`
        console.log('[kvelmo] Connecting to:', url, 'DEV:', import.meta.env.DEV)
        const client = new SocketClient(url)

        // Handle disconnection with auto-reconnect
        client.setOnDisconnect(() => {
          set({
            connected: false,
            connecting: false,
            client: null,
            error: 'Connection lost - reconnecting...'
          })
          // Auto-reconnect after a delay
          setTimeout(() => {
            get().connect()
          }, 2000)
        })

        try {
          await client.connect()
          set({ client, connected: true, connecting: false })

          // Load initial data
          await get().loadProjects()
          await get().loadWorkers()
          await get().loadActiveTasks()
        } catch (err) {
          set({
            connected: false,
            connecting: false,
            error: err instanceof Error ? err.message : 'Connection failed'
          })
        }
      },

      disconnect: () => {
        const client = get().client
        if (client) {
          client.close()
        }
        set({
          connected: false,
          connecting: false,
          client: null
        })
      },

      loadProjects: async () => {
        const client = get().client
        if (!client) {
          set({ error: 'Not connected' })
          return
        }

        set({ loading: true, error: null })
        try {
          const result = await client.call<{ projects: Project[] }>('projects.list')
          const projects = result.projects || []

          set({ projects, loading: false })
        } catch (err) {
          set({
            error: err instanceof Error ? err.message : 'Failed to load projects',
            loading: false
          })
        }
      },

      addProject: async (path: string) => {
        const client = get().client
        if (!client) {
          set({ error: 'Not connected' })
          return
        }

        set({ loading: true, error: null })
        try {
          await client.call<{ id: string }>('projects.register', { path })
          // Reload projects
          await get().loadProjects()
        } catch (err) {
          set({
            error: err instanceof Error ? err.message : 'Failed to add project',
            loading: false
          })
        }
      },

      removeProject: async (id: string) => {
        const client = get().client
        if (!client) {
          set({ error: 'Not connected' })
          return
        }

        set({ loading: true, error: null })
        try {
          await client.call('projects.unregister', { id })
          // Clear selection if removed
          const selectedId = get().selectedProjectId
          if (selectedId === id) {
            set({ selectedProject: null, selectedProjectId: null })
          }
          // Reload projects
          await get().loadProjects()
        } catch (err) {
          set({
            error: err instanceof Error ? err.message : 'Failed to remove project',
            loading: false
          })
        }
      },

      selectProject: (project) => {
        set({
          selectedProject: project,
          selectedProjectId: project?.path || null
        })
        // Persist to sessionStorage for page refresh survival (cleared on tab close)
        if (project) {
          sessionStorage.setItem('kvelmo-selectedProjectId', project.path)
        } else {
          sessionStorage.removeItem('kvelmo-selectedProjectId')
        }
      },

      loadWorkers: async () => {
        const client = get().client
        if (!client) return

        try {
          const result = await client.call<{ workers: Worker[]; stats: WorkerStats }>('workers.list')
          set({
            workers: result.workers || [],
            workerStats: result.stats || null
          })
        } catch (err) {
          console.warn('Failed to load workers:', err)
        }
      },

      loadWorkerStats: async () => {
        const client = get().client
        if (!client) return

        try {
          const result = await client.call<WorkerStats>('workers.stats', {})
          set({ workerStats: result })
        } catch (err) {
          console.warn('Failed to load worker stats:', err)
        }
      },

      addWorker: async (agent: string) => {
        const client = get().client
        if (!client) {
          set({ error: 'Not connected' })
          return
        }

        set({ loading: true, error: null })
        try {
          await client.call('workers.add', { agent })
          await get().loadWorkers()
          set({ loading: false })
        } catch (err) {
          set({
            error: err instanceof Error ? err.message : 'Failed to add worker',
            loading: false
          })
        }
      },

      removeWorker: async (id: string) => {
        const client = get().client
        if (!client) {
          set({ error: 'Not connected' })
          return
        }

        set({ loading: true, error: null })
        try {
          await client.call('workers.remove', { id })
          await get().loadWorkers()
          set({ loading: false })
        } catch (err) {
          set({
            error: err instanceof Error ? err.message : 'Failed to remove worker',
            loading: false
          })
        }
      },

      stopChat: async (worktreeId: string, jobId?: string) => {
        const client = get().client
        if (!client) {
          set({ error: 'Not connected' })
          return
        }

        try {
          const params: Record<string, string> = { worktree_id: worktreeId }
          if (jobId) {
            params.job_id = jobId
          }
          await client.call('chat.stop', params)
        } catch (err) {
          set({ error: err instanceof Error ? err.message : 'Failed to stop chat' })
        }
      },

      loadActiveTasks: async () => {
        const client = get().client
        if (!client) return

        try {
          const result = await client.call<{ tasks: TaskSummary[] }>('tasks.list')
          set({ activeTasks: result.tasks || [] })
        } catch (err) {
          console.warn('Failed to load active tasks:', err)
        }
      },

      searchMemory: async (query: string, limit: number = 10): Promise<MemoryResult[]> => {
        const client = get().client
        if (!client) return []

        try {
          const result = await client.call<{ results: MemoryResult[] }>('memory.search', { query, limit })
          return result.results || []
        } catch (err) {
          console.warn('Failed to search memory:', err)
          return []
        }
      },

      loadMemoryStats: async () => {
        const client = get().client
        if (!client) return

        try {
          const result = await client.call<MemoryStats>('memory.stats', {})
          set({ memoryStats: result })
        } catch (err) {
          console.warn('Failed to load memory stats:', err)
        }
      },

      clearMemory: async () => {
        const client = get().client
        if (!client) return

        try {
          await client.call('memory.clear', {})
          set({ memoryStats: null })
        } catch (err) {
          console.warn('Failed to clear memory:', err)
        }
      },

      loadJobs: async () => {
        const client = get().client
        if (!client) return

        try {
          const result = await client.call<{ jobs: Job[] }>('jobs.list', {})
          set({ jobs: result.jobs || [] })
        } catch (err) {
          console.warn('Failed to load jobs:', err)
        }
      },

      loadJob: async (id: string): Promise<Job | null> => {
        const client = get().client
        if (!client) return null

        try {
          const result = await client.call<Job>('jobs.get', { id })
          return result
        } catch (err) {
          console.warn('Failed to load job:', err)
          return null
        }
      }
    }),
    {
      name: storeName('global'),
      // Project selection persisted via sessionStorage in selectProject() - survives refresh but not tab close
      partialize: () => ({})
    }
  )
)
