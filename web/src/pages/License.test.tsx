import { describe, expect, it, vi } from 'vitest'
import { render, screen, waitFor } from '@/test/test-utils'
import License from './License'

describe('License page', () => {
  it('renders project license and grouped dependency licenses', async () => {
    vi.stubGlobal(
      'fetch',
      vi
        .fn()
        .mockResolvedValueOnce({
          text: async () => 'MIT License Text',
        })
        .mockResolvedValueOnce({
          ok: true,
          json: async () => ({
            licenses: [
              { path: 'a/pkg', license: 'MIT', unknown: false },
              { path: 'b/pkg', license: 'Apache-2.0', unknown: false },
              { path: 'c/pkg', license: 'MIT', unknown: true },
            ],
          }),
        })
    )

    render(<License />)

    await waitFor(() => {
      expect(screen.getByRole('heading', { name: /open source licenses/i })).toBeInTheDocument()
    })

    expect(screen.getByText('MIT License Text')).toBeInTheDocument()
    expect(screen.getByText(/this software includes 3 open source packages/i)).toBeInTheDocument()
    expect(screen.getAllByText('MIT').length).toBeGreaterThan(0)
    expect(screen.getByText('Apache-2.0')).toBeInTheDocument()
  })

  it('shows error message when dependency license fetch fails', async () => {
    vi.stubGlobal(
      'fetch',
      vi
        .fn()
        .mockResolvedValueOnce({
          text: async () => 'MIT',
        })
        .mockResolvedValueOnce({
          ok: false,
        })
    )

    render(<License />)

    await waitFor(() => {
      expect(screen.getByText('Failed to load licenses')).toBeInTheDocument()
    })
  })
})
