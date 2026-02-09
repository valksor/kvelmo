import { describe, expect, it, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent } from '@/test/test-utils'
import Library from './Library'

// Mock hooks
const useStatusMock = vi.fn()
const useLibraryCollectionsMock = vi.fn()
const useLibraryItemsMock = vi.fn()

vi.mock('@/api/workflow', () => ({
  useStatus: () => useStatusMock(),
}))

vi.mock('@/api/library', () => ({
  useLibraryCollections: () => useLibraryCollectionsMock(),
  useLibraryItems: (collectionId?: string) => useLibraryItemsMock(collectionId),
}))

vi.mock('@/components/project/ProjectSelector', () => ({
  ProjectSelector: () => <div data-testid="project-selector">Project Selector</div>,
}))

describe('Library page', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    useStatusMock.mockReturnValue({
      data: { mode: 'project' },
      isLoading: false,
    })
    useLibraryCollectionsMock.mockReturnValue({
      data: { enabled: true, collections: [] },
      isLoading: false,
      error: null,
    })
    useLibraryItemsMock.mockReturnValue({
      data: null,
      isLoading: false,
    })
  })

  it('renders page title and description', () => {
    render(<Library />)

    expect(screen.getByRole('heading', { name: 'Specification Library' })).toBeInTheDocument()
    expect(screen.getByText(/Browse and search/)).toBeInTheDocument()
  })

  it('shows loading spinner while status is loading', () => {
    useStatusMock.mockReturnValue({
      data: null,
      isLoading: true,
    })

    render(<Library />)

    expect(screen.queryByRole('heading', { name: 'Specification Library' })).not.toBeInTheDocument()
  })

  it('shows project selector in global mode', () => {
    useStatusMock.mockReturnValue({
      data: { mode: 'global' },
      isLoading: false,
    })

    render(<Library />)

    expect(screen.getByTestId('project-selector')).toBeInTheDocument()
  })

  it('shows disabled message when library is disabled', () => {
    useLibraryCollectionsMock.mockReturnValue({
      data: { enabled: false, collections: [] },
      isLoading: false,
      error: null,
    })

    render(<Library />)

    expect(screen.getByText('Library Disabled')).toBeInTheDocument()
    expect(screen.getByText(/Enable the library feature/)).toBeInTheDocument()
  })

  it('shows empty collections message when no collections', () => {
    render(<Library />)

    expect(screen.getByText('No collections yet')).toBeInTheDocument()
  })

  it('shows collections list when available', () => {
    useLibraryCollectionsMock.mockReturnValue({
      data: {
        enabled: true,
        collections: [
          { id: 'col-1', name: 'API Specs', item_count: 5, description: 'API specifications' },
          { id: 'col-2', name: 'Design Docs', item_count: 3 },
        ],
      },
      isLoading: false,
      error: null,
    })

    render(<Library />)

    expect(screen.getByText('API Specs')).toBeInTheDocument()
    expect(screen.getByText('Design Docs')).toBeInTheDocument()
    expect(screen.getByText('5')).toBeInTheDocument()
  })

  it('shows error message when collections fail to load', () => {
    useLibraryCollectionsMock.mockReturnValue({
      data: null,
      isLoading: false,
      error: { message: 'Network error' },
    })

    render(<Library />)

    expect(screen.getByText(/Failed to load library: Network error/)).toBeInTheDocument()
  })

  it('shows prompt to select a collection when none selected', () => {
    useLibraryCollectionsMock.mockReturnValue({
      data: {
        enabled: true,
        collections: [{ id: 'col-1', name: 'API Specs', item_count: 5 }],
      },
      isLoading: false,
      error: null,
    })

    render(<Library />)

    expect(screen.getByText('Select a collection to view its items')).toBeInTheDocument()
  })

  it('shows items when a collection is selected', () => {
    useLibraryCollectionsMock.mockReturnValue({
      data: {
        enabled: true,
        collections: [{ id: 'col-1', name: 'API Specs', item_count: 2 }],
      },
      isLoading: false,
      error: null,
    })
    useLibraryItemsMock.mockReturnValue({
      data: {
        items: [
          { id: 'item-1', title: 'Auth API', content: 'Authentication API documentation' },
          { id: 'item-2', title: 'User API', content: 'User management API documentation' },
        ],
      },
      isLoading: false,
    })

    render(<Library />)

    // Click on collection
    fireEvent.click(screen.getByText('API Specs'))

    // Items should appear
    expect(screen.getByText('Auth API')).toBeInTheDocument()
    expect(screen.getByText('User API')).toBeInTheDocument()
  })

  it('shows empty items message when collection has no items', () => {
    useLibraryCollectionsMock.mockReturnValue({
      data: {
        enabled: true,
        collections: [{ id: 'col-1', name: 'Empty Collection', item_count: 0 }],
      },
      isLoading: false,
      error: null,
    })
    useLibraryItemsMock.mockReturnValue({
      data: { items: [] },
      isLoading: false,
    })

    render(<Library />)

    fireEvent.click(screen.getByText('Empty Collection'))

    expect(screen.getByText('No items in this collection')).toBeInTheDocument()
  })

  it('filters items by search query', () => {
    useLibraryCollectionsMock.mockReturnValue({
      data: {
        enabled: true,
        collections: [{ id: 'col-1', name: 'API Specs', item_count: 2 }],
      },
      isLoading: false,
      error: null,
    })
    useLibraryItemsMock.mockReturnValue({
      data: {
        items: [
          { id: 'item-1', title: 'Auth API', content: 'Authentication API' },
          { id: 'item-2', title: 'User API', content: 'User management' },
        ],
      },
      isLoading: false,
    })

    render(<Library />)

    // Select collection
    fireEvent.click(screen.getByText('API Specs'))

    // Search for "Auth"
    const input = screen.getByPlaceholderText(/search in collection/i)
    fireEvent.change(input, { target: { value: 'Auth' } })

    // Only Auth API should be visible
    expect(screen.getByText('Auth API')).toBeInTheDocument()
    expect(screen.queryByText('User API')).not.toBeInTheDocument()
  })

  it('shows item detail when item is clicked', () => {
    useLibraryCollectionsMock.mockReturnValue({
      data: {
        enabled: true,
        collections: [{ id: 'col-1', name: 'API Specs', item_count: 1 }],
      },
      isLoading: false,
      error: null,
    })
    useLibraryItemsMock.mockReturnValue({
      data: {
        items: [
          { id: 'item-1', title: 'Auth API', content: 'Full authentication documentation here', tags: ['api', 'auth'] },
        ],
      },
      isLoading: false,
    })

    render(<Library />)

    // Select collection
    fireEvent.click(screen.getByText('API Specs'))

    // Click on item
    const itemCard = screen.getByText('Auth API').closest('[role="button"]')!
    fireEvent.click(itemCard)

    // Detail view should show
    expect(screen.getByText('Full authentication documentation here')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /back to list/i })).toBeInTheDocument()
  })

  it('shows tags in item detail view', () => {
    useLibraryCollectionsMock.mockReturnValue({
      data: {
        enabled: true,
        collections: [{ id: 'col-1', name: 'API Specs', item_count: 1 }],
      },
      isLoading: false,
      error: null,
    })
    useLibraryItemsMock.mockReturnValue({
      data: {
        items: [
          { id: 'item-1', title: 'Auth API', content: 'Content', tags: ['api', 'auth'] },
        ],
      },
      isLoading: false,
    })

    render(<Library />)

    fireEvent.click(screen.getByText('API Specs'))
    const itemCard = screen.getByText('Auth API').closest('[role="button"]')!
    fireEvent.click(itemCard)

    expect(screen.getByText('api')).toBeInTheDocument()
    expect(screen.getByText('auth')).toBeInTheDocument()
  })

  it('returns to items list when Back to list is clicked', () => {
    useLibraryCollectionsMock.mockReturnValue({
      data: {
        enabled: true,
        collections: [{ id: 'col-1', name: 'API Specs', item_count: 1 }],
      },
      isLoading: false,
      error: null,
    })
    useLibraryItemsMock.mockReturnValue({
      data: {
        items: [
          { id: 'item-1', title: 'Auth API', content: 'Content' },
        ],
      },
      isLoading: false,
    })

    render(<Library />)

    fireEvent.click(screen.getByText('API Specs'))
    const itemCard = screen.getByText('Auth API').closest('[role="button"]')!
    fireEvent.click(itemCard)

    // Click back
    fireEvent.click(screen.getByRole('button', { name: /back to list/i }))

    // Should be back to items list
    expect(screen.queryByText('Back to list')).not.toBeInTheDocument()
    expect(screen.getByPlaceholderText(/search in collection/i)).toBeInTheDocument()
  })
})
