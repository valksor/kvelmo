import { describe, expect, it, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@/test/test-utils'
import Commit from './Commit'

// Mock hooks
const useStatusMock = vi.fn()
const useChangesMock = vi.fn()
const useAnalyzeChangesMock = vi.fn()
const useApplyCommitMock = vi.fn()
const analyzeMutateAsyncMock = vi.fn()
const applyMutateAsyncMock = vi.fn()
const refetchMock = vi.fn()

vi.mock('@/api/workflow', () => ({
  useStatus: () => useStatusMock(),
}))

vi.mock('@/api/commit', () => ({
  useChanges: (includeUnstaged: boolean) => useChangesMock(includeUnstaged),
  useAnalyzeChanges: () => useAnalyzeChangesMock(),
  useApplyCommit: () => useApplyCommitMock(),
}))

vi.mock('@/components/project/ProjectSelector', () => ({
  ProjectSelector: () => <div data-testid="project-selector">Project Selector</div>,
}))

describe('Commit page', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    useStatusMock.mockReturnValue({
      data: { mode: 'project' },
      isLoading: false,
    })
    useChangesMock.mockReturnValue({
      data: null,
      isLoading: false,
      error: null,
      refetch: refetchMock,
    })
    useAnalyzeChangesMock.mockReturnValue({
      mutateAsync: analyzeMutateAsyncMock,
      isPending: false,
      isError: false,
      error: null,
    })
    useApplyCommitMock.mockReturnValue({
      mutateAsync: applyMutateAsyncMock,
      isPending: false,
      isError: false,
      isSuccess: false,
      data: null,
      error: null,
    })
  })

  it('renders page title and description', () => {
    render(<Commit />)

    expect(screen.getByRole('heading', { name: 'Commit Generation' })).toBeInTheDocument()
    expect(screen.getByText(/AI-powered commit message/)).toBeInTheDocument()
  })

  it('shows loading spinner while status is loading', () => {
    useStatusMock.mockReturnValue({
      data: null,
      isLoading: true,
    })

    render(<Commit />)

    expect(screen.queryByRole('heading', { name: 'Commit Generation' })).not.toBeInTheDocument()
  })

  it('shows project selector in global mode', () => {
    useStatusMock.mockReturnValue({
      data: { mode: 'global' },
      isLoading: false,
    })

    render(<Commit />)

    expect(screen.getByTestId('project-selector')).toBeInTheDocument()
  })

  it('shows no changes message when no files changed', () => {
    useChangesMock.mockReturnValue({
      data: { files: [] },
      isLoading: false,
      error: null,
      refetch: refetchMock,
    })

    render(<Commit />)

    expect(screen.getByText('No Changes')).toBeInTheDocument()
  })

  it('renders include unstaged checkbox', () => {
    render(<Commit />)

    const checkbox = screen.getByRole('checkbox')
    expect(checkbox).not.toBeChecked()
  })

  it('toggles include unstaged checkbox', () => {
    render(<Commit />)

    const checkbox = screen.getByRole('checkbox')
    fireEvent.click(checkbox)

    expect(useChangesMock).toHaveBeenCalledWith(true)
  })

  it('shows error when changes fail to load', () => {
    useChangesMock.mockReturnValue({
      data: null,
      isLoading: false,
      error: { message: 'Network error' },
      refetch: refetchMock,
    })

    render(<Commit />)

    expect(screen.getByText(/Failed to load changes: Network error/)).toBeInTheDocument()
  })

  it('shows file list when changes exist', () => {
    useChangesMock.mockReturnValue({
      data: {
        files: [
          { path: 'src/main.go', status: 'modified', additions: 10, deletions: 5 },
          { path: 'src/new.go', status: 'added', additions: 50, deletions: 0 },
        ],
        total_additions: 60,
        total_deletions: 5,
      },
      isLoading: false,
      error: null,
      refetch: refetchMock,
    })

    render(<Commit />)

    expect(screen.getByText('src/main.go')).toBeInTheDocument()
    expect(screen.getByText('src/new.go')).toBeInTheDocument()
    expect(screen.getByText('modified')).toBeInTheDocument()
    expect(screen.getByText('added')).toBeInTheDocument()
  })

  it('shows stats when changes exist', () => {
    useChangesMock.mockReturnValue({
      data: {
        files: [{ path: 'src/main.go', status: 'modified', additions: 10, deletions: 5 }],
        total_additions: 10,
        total_deletions: 5,
      },
      isLoading: false,
      error: null,
      refetch: refetchMock,
    })

    render(<Commit />)

    expect(screen.getByText('Files Changed')).toBeInTheDocument()
    expect(screen.getByText('Additions')).toBeInTheDocument()
    expect(screen.getByText('Deletions')).toBeInTheDocument()
  })

  it('shows analyze button when changes exist', () => {
    useChangesMock.mockReturnValue({
      data: {
        files: [{ path: 'src/main.go', status: 'modified', additions: 10, deletions: 5 }],
      },
      isLoading: false,
      error: null,
      refetch: refetchMock,
    })

    render(<Commit />)

    expect(screen.getByRole('button', { name: /analyze changes/i })).toBeInTheDocument()
  })

  it('calls analyze mutation when Analyze button is clicked', async () => {
    analyzeMutateAsyncMock.mockResolvedValue({ message: 'Fix: Update authentication flow' })

    useChangesMock.mockReturnValue({
      data: {
        files: [{ path: 'src/main.go', status: 'modified', additions: 10, deletions: 5 }],
      },
      isLoading: false,
      error: null,
      refetch: refetchMock,
    })

    render(<Commit />)

    fireEvent.click(screen.getByRole('button', { name: /analyze changes/i }))

    await waitFor(() => {
      expect(analyzeMutateAsyncMock).toHaveBeenCalledWith({ include_unstaged: false })
    })
  })

  it('shows commit message editor after analysis', async () => {
    analyzeMutateAsyncMock.mockResolvedValue({ message: 'Fix: Update authentication flow' })

    useChangesMock.mockReturnValue({
      data: {
        files: [{ path: 'src/main.go', status: 'modified', additions: 10, deletions: 5 }],
      },
      isLoading: false,
      error: null,
      refetch: refetchMock,
    })

    render(<Commit />)

    fireEvent.click(screen.getByRole('button', { name: /analyze changes/i }))

    await waitFor(() => {
      expect(screen.getByText('Commit Message')).toBeInTheDocument()
    })
  })

  it('shows error when analysis fails', () => {
    useAnalyzeChangesMock.mockReturnValue({
      mutateAsync: analyzeMutateAsyncMock,
      isPending: false,
      isError: true,
      error: { message: 'Analysis failed' },
    })

    useChangesMock.mockReturnValue({
      data: {
        files: [{ path: 'src/main.go', status: 'modified', additions: 10, deletions: 5 }],
      },
      isLoading: false,
      error: null,
      refetch: refetchMock,
    })

    render(<Commit />)

    expect(screen.getByText(/Failed to analyze: Analysis failed/)).toBeInTheDocument()
  })

  it('shows success message after commit', () => {
    useApplyCommitMock.mockReturnValue({
      mutateAsync: applyMutateAsyncMock,
      isPending: false,
      isError: false,
      isSuccess: true,
      data: { commit_hash: 'abc123def456' },
      error: null,
    })

    useChangesMock.mockReturnValue({
      data: {
        files: [{ path: 'src/main.go', status: 'modified', additions: 10, deletions: 5 }],
      },
      isLoading: false,
      error: null,
      refetch: refetchMock,
    })

    render(<Commit />)

    expect(screen.getByText('abc123d')).toBeInTheDocument()
  })

  it('calls refetch when Refresh button is clicked', () => {
    render(<Commit />)

    fireEvent.click(screen.getByRole('button', { name: /refresh/i }))

    expect(refetchMock).toHaveBeenCalled()
  })

  it('shows loading state while analyzing', () => {
    useAnalyzeChangesMock.mockReturnValue({
      mutateAsync: analyzeMutateAsyncMock,
      isPending: true,
      isError: false,
      error: null,
    })

    useChangesMock.mockReturnValue({
      data: {
        files: [{ path: 'src/main.go', status: 'modified', additions: 10, deletions: 5 }],
      },
      isLoading: false,
      error: null,
      refetch: refetchMock,
    })

    render(<Commit />)

    expect(screen.getByText(/analyzing/i)).toBeInTheDocument()
  })
})
