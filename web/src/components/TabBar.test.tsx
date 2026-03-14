import { render } from '@testing-library/react'
import { describe, it, expect, vi } from 'vitest'
import { TabBar } from './TabBar'

vi.mock('../stores/layoutStore', () => ({
  useLayoutStore: () => ({
    tabs: [
      { id: 'tab-1', type: 'chat', title: 'Chat', closeable: true },
      { id: 'tab-2', type: 'spec', title: 'Spec', closeable: false },
    ],
    activeTabId: 'tab-1',
    setActiveTab: vi.fn(),
    closeTab: vi.fn(),
    openTab: vi.fn(),
  }),
}))

describe('TabBar', () => {
  it('has role="tablist" on the tab container', () => {
    const { getByRole } = render(<TabBar />)
    expect(getByRole('tablist')).toBeInTheDocument()
  })

  it('tab buttons have role="tab"', () => {
    const { getAllByRole } = render(<TabBar />)
    expect(getAllByRole('tab')).toHaveLength(2)
  })

  it('active tab has aria-selected="true"', () => {
    const { getAllByRole } = render(<TabBar />)
    expect(getAllByRole('tab')[0]).toHaveAttribute('aria-selected', 'true')
  })

  it('inactive tab has aria-selected="false"', () => {
    const { getAllByRole } = render(<TabBar />)
    expect(getAllByRole('tab')[1]).toHaveAttribute('aria-selected', 'false')
  })

  it('add-tab button has aria-label', () => {
    const { getByRole } = render(<TabBar />)
    expect(getByRole('button', { name: /add new tab/i })).toBeInTheDocument()
  })

  it('add-tab dropdown has aria-expanded', () => {
    const { getByRole } = render(<TabBar />)
    const btn = getByRole('button', { name: /add new tab/i })
    expect(btn).toHaveAttribute('aria-expanded', 'false')
  })

  it('tab SVG icons are aria-hidden', () => {
    const { getAllByRole } = render(<TabBar />)
    const tabs = getAllByRole('tab')
    // Each tab should contain an svg — verify no svg has a non-hidden role
    tabs.forEach((tab: HTMLElement) => {
      const svgs = tab.querySelectorAll('svg')
      svgs.forEach((svg: SVGSVGElement) => {
        expect(svg).toHaveAttribute('aria-hidden', 'true')
      })
    })
  })
})
