import { describe, it, expect, vi } from 'vitest'
import { render } from '@/test/test-utils'
import { WorkflowDiagram } from './WorkflowDiagram'

vi.mock('@/api/task', () => ({
  useWorkflowDiagram: vi.fn(() => ({
    data: `
      <svg xmlns="http://www.w3.org/2000/svg">
        <rect class="state-box current" />
        <text class="state-text">idle</text>
        <rect class="state-box" />
        <text class="state-text">planning</text>
      </svg>
    `,
    isLoading: false,
    error: null,
  })),
}))

describe('WorkflowDiagram', () => {
  it('highlights planning node when idle state has planned progress', () => {
    const { container } = render(<WorkflowDiagram currentState="idle" progressPhase="planned" />)
    const labels = Array.from(container.querySelectorAll('text.state-text'))
    const planningLabel = labels.find((label) => label.textContent?.trim() === 'planning')
    const idleLabel = labels.find((label) => label.textContent?.trim() === 'idle')

    expect(planningLabel).toBeTruthy()
    expect(idleLabel).toBeTruthy()

    const planningRect = planningLabel?.previousElementSibling as SVGRectElement | null
    const idleRect = idleLabel?.previousElementSibling as SVGRectElement | null

    expect(planningRect?.classList.contains('workflow-current')).toBe(true)
    expect(idleRect?.classList.contains('workflow-current')).toBe(false)
  })

  it('highlights idle node when idle state has started progress', () => {
    const { container } = render(<WorkflowDiagram currentState="idle" progressPhase="started" />)
    const labels = Array.from(container.querySelectorAll('text.state-text'))
    const idleLabel = labels.find((label) => label.textContent?.trim() === 'idle')
    const idleRect = idleLabel?.previousElementSibling as SVGRectElement | null

    expect(idleRect?.classList.contains('workflow-current')).toBe(true)
  })
})
