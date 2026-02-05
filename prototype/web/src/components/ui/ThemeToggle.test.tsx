import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { ThemeToggle } from './ThemeToggle'

describe('ThemeToggle', () => {
  beforeEach(() => {
    // Reset localStorage mock
    vi.mocked(localStorage.getItem).mockReturnValue(null)
    vi.mocked(localStorage.setItem).mockClear()
    document.documentElement.removeAttribute('data-theme')
  })

  it('renders toggle button', () => {
    render(<ThemeToggle />)

    expect(screen.getByRole('button')).toBeInTheDocument()
  })

  it('shows moon icon in light mode (to switch to dark)', () => {
    vi.mocked(localStorage.getItem).mockReturnValue('light')
    render(<ThemeToggle />)

    expect(screen.getByTitle(/switch to dark mode/i)).toBeInTheDocument()
  })

  it('shows sun icon in dark mode (to switch to light)', () => {
    vi.mocked(localStorage.getItem).mockReturnValue('dark')
    render(<ThemeToggle />)

    expect(screen.getByTitle(/switch to light mode/i)).toBeInTheDocument()
  })

  it('toggles theme when clicked', async () => {
    const user = userEvent.setup()
    vi.mocked(localStorage.getItem).mockReturnValue('light')
    render(<ThemeToggle />)

    // Initially should show "switch to dark" (meaning we're in light mode)
    expect(screen.getByTitle(/switch to dark mode/i)).toBeInTheDocument()

    await user.click(screen.getByRole('button'))

    // After click should show "switch to light" (meaning we're now in dark mode)
    expect(screen.getByTitle(/switch to light mode/i)).toBeInTheDocument()
  })

  it('saves theme preference to localStorage on toggle', async () => {
    const user = userEvent.setup()
    vi.mocked(localStorage.getItem).mockReturnValue('light')
    render(<ThemeToggle />)

    await user.click(screen.getByRole('button'))

    expect(localStorage.setItem).toHaveBeenCalledWith('mehrhof-theme', 'dark')
  })

  it('applies theme to document element', async () => {
    const user = userEvent.setup()
    vi.mocked(localStorage.getItem).mockReturnValue('light')
    render(<ThemeToggle />)

    // After initial render, theme should be applied
    expect(document.documentElement.getAttribute('data-theme')).toBe('winter')

    await user.click(screen.getByRole('button'))

    // After toggle, should switch to dark theme
    expect(document.documentElement.getAttribute('data-theme')).toBe('business')
  })

  it('defaults to light theme when no preference stored', () => {
    vi.mocked(localStorage.getItem).mockReturnValue(null)
    render(<ThemeToggle />)

    expect(screen.getByTitle(/switch to dark mode/i)).toBeInTheDocument()
  })

  it('has accessible aria-label', () => {
    vi.mocked(localStorage.getItem).mockReturnValue('light')
    render(<ThemeToggle />)

    expect(screen.getByRole('button')).toHaveAttribute('aria-label', 'Switch to dark mode')
  })
})
