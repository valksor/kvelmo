import { describe, expect, it, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@/test/test-utils'
import Simplify from './Simplify'

// Mock hooks
const useStatusMock = vi.fn()
const useStandaloneSimplifyMock = vi.fn()
const mutateAsyncMock = vi.fn()

vi.mock('@/api/workflow', () => ({
  useStatus: () => useStatusMock(),
}))

vi.mock('@/api/standalone', () => ({
  useStandaloneSimplify: () => useStandaloneSimplifyMock(),
}))

vi.mock('@/components/project/ProjectSelector', () => ({
  ProjectSelector: () => <div data-testid="project-selector">Project Selector</div>,
}))

describe('Simplify page', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    useStatusMock.mockReturnValue({
      data: { mode: 'project' },
      isLoading: false,
    })
    useStandaloneSimplifyMock.mockReturnValue({
      mutateAsync: mutateAsyncMock,
      isPending: false,
      isSuccess: false,
      isError: false,
      data: null,
      error: null,
    })
  })

  it('renders page title and description', () => {
    render(<Simplify />)

    expect(screen.getByRole('heading', { name: 'Code Simplifier' })).toBeInTheDocument()
    expect(screen.getByText(/AI-powered code simplification/)).toBeInTheDocument()
  })

  it('shows loading spinner while status is loading', () => {
    useStatusMock.mockReturnValue({
      data: null,
      isLoading: true,
    })

    render(<Simplify />)

    expect(screen.queryByRole('heading', { name: 'Code Simplifier' })).not.toBeInTheDocument()
  })

  it('shows project selector in global mode', () => {
    useStatusMock.mockReturnValue({
      data: { mode: 'global' },
      isLoading: false,
    })

    render(<Simplify />)

    expect(screen.getByTestId('project-selector')).toBeInTheDocument()
  })

  it('renders mode selector with default value', () => {
    render(<Simplify />)

    const select = screen.getByRole('combobox')
    expect(select).toHaveValue('uncommitted')
  })

  it('shows branch input when branch mode is selected', () => {
    render(<Simplify />)

    const select = screen.getByRole('combobox')
    fireEvent.change(select, { target: { value: 'branch' } })

    expect(screen.getByPlaceholderText('main')).toBeInTheDocument()
  })

  it('shows range input when range mode is selected', () => {
    render(<Simplify />)

    const select = screen.getByRole('combobox')
    fireEvent.change(select, { target: { value: 'range' } })

    expect(screen.getByPlaceholderText('HEAD~3..HEAD')).toBeInTheDocument()
  })

  it('shows files textarea when files mode is selected', () => {
    render(<Simplify />)

    const select = screen.getByRole('combobox')
    fireEvent.change(select, { target: { value: 'files' } })

    expect(screen.getByPlaceholderText(/src\/main.go/)).toBeInTheDocument()
  })

  it('renders context lines slider', () => {
    render(<Simplify />)

    const slider = screen.getByRole('slider')
    expect(slider).toHaveValue('3')
  })

  it('renders checkpoint checkbox checked by default', () => {
    render(<Simplify />)

    const checkbox = screen.getByRole('checkbox')
    expect(checkbox).toBeChecked()
  })

  it('submits form with default settings', async () => {
    mutateAsyncMock.mockResolvedValue({ changes: [], summary: '' })

    render(<Simplify />)

    const button = screen.getByRole('button', { name: /run simplify/i })
    fireEvent.click(button)

    await waitFor(() => {
      expect(mutateAsyncMock).toHaveBeenCalledWith({
        mode: 'uncommitted',
        context: 3,
        create_checkpoint: true,
      })
    })
  })

  it('submits with branch settings when in branch mode', async () => {
    mutateAsyncMock.mockResolvedValue({ changes: [], summary: '' })

    render(<Simplify />)

    // Switch to branch mode
    const select = screen.getByRole('combobox')
    fireEvent.change(select, { target: { value: 'branch' } })

    // Change base branch
    const input = screen.getByPlaceholderText('main')
    fireEvent.change(input, { target: { value: 'develop' } })

    // Submit
    const button = screen.getByRole('button', { name: /run simplify/i })
    fireEvent.click(button)

    await waitFor(() => {
      expect(mutateAsyncMock).toHaveBeenCalledWith(
        expect.objectContaining({
          mode: 'branch',
          base_branch: 'develop',
        })
      )
    })
  })

  it('shows loading state while simplifying', () => {
    useStandaloneSimplifyMock.mockReturnValue({
      mutateAsync: mutateAsyncMock,
      isPending: true,
      isSuccess: false,
      isError: false,
      data: null,
      error: null,
    })

    render(<Simplify />)

    expect(screen.getByText('Simplifying...')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /simplifying/i })).toBeDisabled()
  })

  it('shows error message when simplify fails', () => {
    useStandaloneSimplifyMock.mockReturnValue({
      mutateAsync: mutateAsyncMock,
      isPending: false,
      isSuccess: false,
      isError: true,
      data: null,
      error: { message: 'Simplification failed' },
    })

    render(<Simplify />)

    expect(screen.getByText('Simplification failed')).toBeInTheDocument()
  })

  it('shows no changes message when result has no changes', () => {
    useStandaloneSimplifyMock.mockReturnValue({
      mutateAsync: mutateAsyncMock,
      isPending: false,
      isSuccess: true,
      isError: false,
      data: { changes: [], summary: '' },
      error: null,
    })

    render(<Simplify />)

    expect(screen.getByText('No Changes Needed')).toBeInTheDocument()
    expect(screen.getByText(/already simplified/)).toBeInTheDocument()
  })

  it('shows file changes when result has changes', () => {
    useStandaloneSimplifyMock.mockReturnValue({
      mutateAsync: mutateAsyncMock,
      isPending: false,
      isSuccess: true,
      isError: false,
      data: {
        changes: [
          { path: 'src/main.go', operation: 'update' },
          { path: 'src/handler.go', operation: 'create' },
        ],
        summary: 'Simplified 2 files',
      },
      error: null,
    })

    render(<Simplify />)

    expect(screen.getByText('File Changes (2)')).toBeInTheDocument()
    expect(screen.getByText('src/main.go')).toBeInTheDocument()
    expect(screen.getByText('src/handler.go')).toBeInTheDocument()
  })

  it('shows summary when provided', () => {
    useStandaloneSimplifyMock.mockReturnValue({
      mutateAsync: mutateAsyncMock,
      isPending: false,
      isSuccess: true,
      isError: false,
      data: {
        changes: [],
        summary: 'Removed redundant code and improved readability',
      },
      error: null,
    })

    render(<Simplify />)

    expect(screen.getByText('Summary')).toBeInTheDocument()
    expect(screen.getByText('Removed redundant code and improved readability')).toBeInTheDocument()
  })

  it('shows usage statistics when provided', () => {
    useStandaloneSimplifyMock.mockReturnValue({
      mutateAsync: mutateAsyncMock,
      isPending: false,
      isSuccess: true,
      isError: false,
      data: {
        changes: [],
        summary: '',
        usage: {
          input_tokens: 1000,
          output_tokens: 500,
          cached_tokens: 200,
          cost_usd: 0.0025,
        },
      },
      error: null,
    })

    render(<Simplify />)

    expect(screen.getByText('Input Tokens')).toBeInTheDocument()
    expect(screen.getByText('1,000')).toBeInTheDocument()
    expect(screen.getByText('Output Tokens')).toBeInTheDocument()
    expect(screen.getByText('500')).toBeInTheDocument()
  })

  it('shows warning when response contains error', () => {
    useStandaloneSimplifyMock.mockReturnValue({
      mutateAsync: mutateAsyncMock,
      isPending: false,
      isSuccess: true,
      isError: false,
      data: {
        changes: [],
        summary: '',
        error: 'Some warning about the operation',
      },
      error: null,
    })

    render(<Simplify />)

    expect(screen.getByText('Some warning about the operation')).toBeInTheDocument()
  })
})
