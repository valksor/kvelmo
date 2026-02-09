import { describe, expect, it, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@/test/test-utils'
import Review from './Review'

// Mock hooks
const useStatusMock = vi.fn()
const useStandaloneReviewMock = vi.fn()
const mutateAsyncMock = vi.fn()

vi.mock('@/api/workflow', () => ({
  useStatus: () => useStatusMock(),
}))

vi.mock('@/api/standalone', () => ({
  useStandaloneReview: () => useStandaloneReviewMock(),
}))

vi.mock('@/components/project/ProjectSelector', () => ({
  ProjectSelector: () => <div data-testid="project-selector">Project Selector</div>,
}))

describe('Review page', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    useStatusMock.mockReturnValue({
      data: { mode: 'project' },
      isLoading: false,
    })
    useStandaloneReviewMock.mockReturnValue({
      mutateAsync: mutateAsyncMock,
      isPending: false,
      isSuccess: false,
      isError: false,
      data: null,
      error: null,
    })
  })

  it('renders page title and description', () => {
    render(<Review />)

    expect(screen.getByRole('heading', { name: 'Standalone Review' })).toBeInTheDocument()
    expect(screen.getByText(/AI-powered code review/)).toBeInTheDocument()
  })

  it('shows loading spinner while status is loading', () => {
    useStatusMock.mockReturnValue({
      data: null,
      isLoading: true,
    })

    render(<Review />)

    expect(screen.queryByRole('heading', { name: 'Standalone Review' })).not.toBeInTheDocument()
  })

  it('shows project selector in global mode', () => {
    useStatusMock.mockReturnValue({
      data: { mode: 'global' },
      isLoading: false,
    })

    render(<Review />)

    expect(screen.getByTestId('project-selector')).toBeInTheDocument()
  })

  it('renders mode selector with default value', () => {
    render(<Review />)

    const select = screen.getByRole('combobox')
    expect(select).toHaveValue('uncommitted')
  })

  it('shows branch input when branch mode is selected', () => {
    render(<Review />)

    const select = screen.getByRole('combobox')
    fireEvent.change(select, { target: { value: 'branch' } })

    expect(screen.getByPlaceholderText('main')).toBeInTheDocument()
  })

  it('shows range input when range mode is selected', () => {
    render(<Review />)

    const select = screen.getByRole('combobox')
    fireEvent.change(select, { target: { value: 'range' } })

    expect(screen.getByPlaceholderText('HEAD~3..HEAD')).toBeInTheDocument()
  })

  it('shows files textarea when files mode is selected', () => {
    render(<Review />)

    const select = screen.getByRole('combobox')
    fireEvent.change(select, { target: { value: 'files' } })

    expect(screen.getByPlaceholderText(/src\/main.go/)).toBeInTheDocument()
  })

  it('renders checkpoint checkbox checked by default', () => {
    render(<Review />)

    const checkbox = screen.getByRole('checkbox')
    expect(checkbox).toBeChecked()
  })

  it('submits form with default settings', async () => {
    mutateAsyncMock.mockResolvedValue({ issues: [], total_issues: 0 })

    render(<Review />)

    const button = screen.getByRole('button', { name: /run review/i })
    fireEvent.click(button)

    await waitFor(() => {
      expect(mutateAsyncMock).toHaveBeenCalledWith({
        mode: 'uncommitted',
        create_checkpoint: true,
      })
    })
  })

  it('shows loading state while reviewing', () => {
    useStandaloneReviewMock.mockReturnValue({
      mutateAsync: mutateAsyncMock,
      isPending: true,
      isSuccess: false,
      isError: false,
      data: null,
      error: null,
    })

    render(<Review />)

    expect(screen.getByText(/running review/i)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /running review/i })).toBeDisabled()
  })

  it('shows error message when review fails', () => {
    useStandaloneReviewMock.mockReturnValue({
      mutateAsync: mutateAsyncMock,
      isPending: false,
      isSuccess: false,
      isError: true,
      data: null,
      error: { message: 'Review failed' },
    })

    render(<Review />)

    expect(screen.getByText('Review failed')).toBeInTheDocument()
  })

  it('shows no issues message when review finds nothing', () => {
    useStandaloneReviewMock.mockReturnValue({
      mutateAsync: mutateAsyncMock,
      isPending: false,
      isSuccess: true,
      isError: false,
      data: { issues: [], total_issues: 0 },
      error: null,
    })

    render(<Review />)

    expect(screen.getByText('No Issues Found')).toBeInTheDocument()
    expect(screen.getByText(/review completed without finding/)).toBeInTheDocument()
  })

  it('shows issues list when review finds issues', () => {
    useStandaloneReviewMock.mockReturnValue({
      mutateAsync: mutateAsyncMock,
      isPending: false,
      isSuccess: true,
      isError: false,
      data: {
        issues: [
          { severity: 'error', message: 'Missing error handling', file: 'main.go', line: 10 },
          { severity: 'warning', message: 'Unused variable', file: 'utils.go', line: 20 },
        ],
        total_issues: 2,
      },
      error: null,
    })

    render(<Review />)

    expect(screen.getByText('Issues (2)')).toBeInTheDocument()
    expect(screen.getByText('Missing error handling')).toBeInTheDocument()
    expect(screen.getByText('Unused variable')).toBeInTheDocument()
    expect(screen.getByText('main.go')).toBeInTheDocument()
  })

  it('shows issue statistics', () => {
    useStandaloneReviewMock.mockReturnValue({
      mutateAsync: mutateAsyncMock,
      isPending: false,
      isSuccess: true,
      isError: false,
      data: {
        issues: [
          { severity: 'error', message: 'Error 1', file: 'a.go' },
          { severity: 'warning', message: 'Warning 1', file: 'b.go' },
          { severity: 'info', message: 'Info 1', file: 'c.go' },
        ],
        total_issues: 3,
      },
      error: null,
    })

    render(<Review />)

    expect(screen.getByText('Total Issues')).toBeInTheDocument()
    expect(screen.getByText('Errors')).toBeInTheDocument()
    expect(screen.getByText('Warnings')).toBeInTheDocument()
  })

  it('shows summary when provided', () => {
    useStandaloneReviewMock.mockReturnValue({
      mutateAsync: mutateAsyncMock,
      isPending: false,
      isSuccess: true,
      isError: false,
      data: {
        issues: [],
        total_issues: 0,
        summary: 'Code looks good overall',
      },
      error: null,
    })

    render(<Review />)

    expect(screen.getByText('Summary')).toBeInTheDocument()
    expect(screen.getByText('Code looks good overall')).toBeInTheDocument()
  })

  it('shows rule badge when issue has a rule', () => {
    useStandaloneReviewMock.mockReturnValue({
      mutateAsync: mutateAsyncMock,
      isPending: false,
      isSuccess: true,
      isError: false,
      data: {
        issues: [
          { severity: 'error', message: 'Error', file: 'a.go', rule: 'errcheck' },
        ],
        total_issues: 1,
      },
      error: null,
    })

    render(<Review />)

    expect(screen.getByText('errcheck')).toBeInTheDocument()
  })
})
