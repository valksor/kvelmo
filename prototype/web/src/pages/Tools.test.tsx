import { describe, expect, it, vi } from 'vitest'
import { render, screen, fireEvent } from '@/test/test-utils'
import Tools from './Tools'

// Mock the ToolPanels components
vi.mock('@/components/tools/ToolPanels', () => ({
  BrowserPanel: () => <div data-testid="browser-panel">Browser Panel</div>,
  MemoryPanel: () => <div data-testid="memory-panel">Memory Panel</div>,
  SecurityPanel: () => <div data-testid="security-panel">Security Panel</div>,
  StackPanel: () => <div data-testid="stack-panel">Stack Panel</div>,
}))

describe('Tools page', () => {
  it('renders page title', () => {
    render(<Tools />)

    expect(screen.getByRole('heading', { name: 'Tools' })).toBeInTheDocument()
  })

  it('renders all tab buttons', () => {
    render(<Tools />)

    expect(screen.getByRole('tab', { name: /browser/i })).toBeInTheDocument()
    expect(screen.getByRole('tab', { name: /memory/i })).toBeInTheDocument()
    expect(screen.getByRole('tab', { name: /security/i })).toBeInTheDocument()
    expect(screen.getByRole('tab', { name: /stack/i })).toBeInTheDocument()
  })

  it('shows browser panel by default', () => {
    render(<Tools />)

    expect(screen.getByTestId('browser-panel')).toBeInTheDocument()
    expect(screen.queryByTestId('memory-panel')).not.toBeInTheDocument()
  })

  it('switches to memory panel when tab is clicked', () => {
    render(<Tools />)

    fireEvent.click(screen.getByRole('tab', { name: /memory/i }))

    expect(screen.getByTestId('memory-panel')).toBeInTheDocument()
    expect(screen.queryByTestId('browser-panel')).not.toBeInTheDocument()
  })

  it('switches to security panel when tab is clicked', () => {
    render(<Tools />)

    fireEvent.click(screen.getByRole('tab', { name: /security/i }))

    expect(screen.getByTestId('security-panel')).toBeInTheDocument()
  })

  it('switches to stack panel when tab is clicked', () => {
    render(<Tools />)

    fireEvent.click(screen.getByRole('tab', { name: /stack/i }))

    expect(screen.getByTestId('stack-panel')).toBeInTheDocument()
  })

  it('has correct ARIA attributes on tabs', () => {
    render(<Tools />)

    const browserTab = screen.getByRole('tab', { name: /browser/i })
    expect(browserTab).toHaveAttribute('aria-selected', 'true')
    expect(browserTab).toHaveAttribute('aria-controls', 'tools-panel-browser')

    const memoryTab = screen.getByRole('tab', { name: /memory/i })
    expect(memoryTab).toHaveAttribute('aria-selected', 'false')
  })

  it('navigates tabs with arrow keys', () => {
    render(<Tools />)

    const browserTab = screen.getByRole('tab', { name: /browser/i })
    browserTab.focus()

    // ArrowRight moves to next tab
    fireEvent.keyDown(browserTab, { key: 'ArrowRight' })
    expect(screen.getByTestId('memory-panel')).toBeInTheDocument()

    // ArrowLeft moves back
    const memoryTab = screen.getByRole('tab', { name: /memory/i })
    fireEvent.keyDown(memoryTab, { key: 'ArrowLeft' })
    expect(screen.getByTestId('browser-panel')).toBeInTheDocument()
  })

  it('navigates to first tab with Home key', () => {
    render(<Tools />)

    // First switch to a different tab
    fireEvent.click(screen.getByRole('tab', { name: /stack/i }))
    expect(screen.getByTestId('stack-panel')).toBeInTheDocument()

    // Press Home
    const stackTab = screen.getByRole('tab', { name: /stack/i })
    fireEvent.keyDown(stackTab, { key: 'Home' })
    expect(screen.getByTestId('browser-panel')).toBeInTheDocument()
  })

  it('navigates to last tab with End key', () => {
    render(<Tools />)

    const browserTab = screen.getByRole('tab', { name: /browser/i })
    fireEvent.keyDown(browserTab, { key: 'End' })

    expect(screen.getByTestId('stack-panel')).toBeInTheDocument()
  })

  it('wraps around when navigating past last tab', () => {
    render(<Tools />)

    // Go to stack tab
    fireEvent.click(screen.getByRole('tab', { name: /stack/i }))

    // Press ArrowRight (should wrap to browser)
    const stackTab = screen.getByRole('tab', { name: /stack/i })
    fireEvent.keyDown(stackTab, { key: 'ArrowRight' })

    expect(screen.getByTestId('browser-panel')).toBeInTheDocument()
  })

  it('wraps around when navigating before first tab', () => {
    render(<Tools />)

    // On browser tab, press ArrowLeft (should wrap to stack)
    const browserTab = screen.getByRole('tab', { name: /browser/i })
    fireEvent.keyDown(browserTab, { key: 'ArrowLeft' })

    expect(screen.getByTestId('stack-panel')).toBeInTheDocument()
  })
})
