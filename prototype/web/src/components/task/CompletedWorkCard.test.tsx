import { describe, expect, it, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent } from '@/test/test-utils'
import { CompletedWorkCard } from './CompletedWorkCard'
import type { WorkResponse } from '@/types/api'

// Mock the TaskContentModal component
vi.mock('./TaskContentModal', () => ({
  TaskContentModal: ({ isOpen, title }: { isOpen: boolean; title: string }) =>
    isOpen ? <div data-testid="modal">Modal: {title}</div> : null,
}))

describe('CompletedWorkCard', () => {
  const baseWork: WorkResponse['work'] = {
    metadata: {
      id: 'task-123',
      title: 'Fix authentication bug',
      created_at: new Date(Date.now() - 3600000).toISOString(), // 1 hour ago
      updated_at: new Date().toISOString(),
      external_key: 'JIRA-456',
      labels: ['bug', 'urgent'],
      pull_request: null,
    },
    description: 'This task fixes the authentication flow',
    git: {
      branch: 'fix/auth-bug',
      base_branch: 'main',
    },
    source: {
      ref: 'tasks/auth-fix.md',
    },
  }

  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders completed task with title', () => {
    render(<CompletedWorkCard work={baseWork} />)

    expect(screen.getByText('Fix authentication bug')).toBeInTheDocument()
    // Status badge shows "done"
    expect(screen.getByText('done')).toBeInTheDocument()
  })

  it('renders external key when present', () => {
    render(<CompletedWorkCard work={baseWork} />)

    expect(screen.getByText('JIRA-456')).toBeInTheDocument()
  })

  it('renders labels when present', () => {
    render(<CompletedWorkCard work={baseWork} />)

    expect(screen.getByText('bug')).toBeInTheDocument()
    expect(screen.getByText('urgent')).toBeInTheDocument()
  })

  it('renders description preview when available', () => {
    render(<CompletedWorkCard work={baseWork} />)

    expect(screen.getByText('This task fixes the authentication flow')).toBeInTheDocument()
  })

  it('shows pull request button when PR exists', () => {
    const workWithPR: WorkResponse['work'] = {
      ...baseWork,
      metadata: {
        ...baseWork.metadata,
        pull_request: {
          number: 42,
          url: 'https://github.com/org/repo/pull/42',
        },
      },
    }

    render(<CompletedWorkCard work={workWithPR} />)

    expect(screen.getByText('Completed with PR')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: /View PR #42/i })).toHaveAttribute(
      'href',
      'https://github.com/org/repo/pull/42'
    )
  })

  it('opens modal when View button is clicked', () => {
    render(<CompletedWorkCard work={baseWork} />)

    // Modal should not be visible initially
    expect(screen.queryByTestId('modal')).not.toBeInTheDocument()

    // Click View button
    fireEvent.click(screen.getByRole('button', { name: /view/i }))

    // Modal should be visible
    expect(screen.getByTestId('modal')).toBeInTheDocument()
  })

  it('toggles technical details section', () => {
    render(<CompletedWorkCard work={baseWork} />)

    // Technical details should be hidden initially
    expect(screen.queryByText('fix/auth-bug')).not.toBeInTheDocument()

    // Click to expand
    fireEvent.click(screen.getByRole('button', { name: /technical details/i }))

    // Now branch should be visible
    expect(screen.getByText('fix/auth-bug')).toBeInTheDocument()
    expect(screen.getByText('main')).toBeInTheDocument()
  })

  it('uses task id as title when title is missing', () => {
    const workWithoutTitle: WorkResponse['work'] = {
      ...baseWork,
      metadata: {
        ...baseWork.metadata,
        title: undefined as unknown as string,
      },
    }

    render(<CompletedWorkCard work={workWithoutTitle} />)

    expect(screen.getByText('task-123')).toBeInTheDocument()
  })

  it('hides labels section when no labels exist', () => {
    const workWithoutLabels: WorkResponse['work'] = {
      ...baseWork,
      metadata: {
        ...baseWork.metadata,
        labels: [],
      },
    }

    render(<CompletedWorkCard work={workWithoutLabels} />)

    expect(screen.queryByText('bug')).not.toBeInTheDocument()
  })

  it('hides technical details button when no git info exists', () => {
    const workWithoutGit: WorkResponse['work'] = {
      ...baseWork,
      git: {
        branch: '',
        base_branch: '',
      },
      source: {
        ref: '',
      },
    }

    render(<CompletedWorkCard work={workWithoutGit} />)

    expect(screen.queryByRole('button', { name: /technical details/i })).not.toBeInTheDocument()
  })
})
