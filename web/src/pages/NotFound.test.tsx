import { describe, expect, it } from 'vitest'
import { render, screen } from '@/test/test-utils'
import NotFound from './NotFound'

describe('NotFound page', () => {
  it('shows 404 page content and current route', () => {
    render(<NotFound />, { route: '/missing/page' })

    expect(screen.getByRole('heading', { level: 1, name: 'Page Not Found' })).toBeInTheDocument()
    expect(screen.getByText('404')).toBeInTheDocument()
    expect(screen.getByText('/missing/page')).toBeInTheDocument()
  })

  it('links back to dashboard', () => {
    render(<NotFound />, { route: '/bad-url' })

    expect(screen.getByRole('link', { name: /back to dashboard/i })).toHaveAttribute('href', '/')
  })
})
