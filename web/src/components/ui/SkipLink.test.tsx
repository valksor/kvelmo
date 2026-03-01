import { render } from '@testing-library/react'
import { describe, it, expect } from 'vitest'
import { SkipLink } from './SkipLink'

describe('SkipLink', () => {
  it('renders a link to #main-content', () => {
    const { getByRole } = render(<SkipLink />)
    const link = getByRole('link', { name: /skip to main content/i })
    expect(link).toHaveAttribute('href', '#main-content')
  })
})
