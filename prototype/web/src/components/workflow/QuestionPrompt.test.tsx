import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import { render, screen } from '@/test/test-utils'
import { QuestionPrompt } from './QuestionPrompt'

const mutateMock = vi.fn()

vi.mock('@/api/workflow', () => ({
  useAnswerQuestion: () => ({
    mutate: mutateMock,
    isPending: false,
  }),
}))

describe('QuestionPrompt', () => {
  it('submits selected quick option immediately', async () => {
    const user = userEvent.setup()
    render(
      <QuestionPrompt
        question={{
          question: 'Choose one option',
          task_id: 'task-1',
          options: [
            { label: 'Yes', value: 'yes', description: 'Proceed' },
            { label: 'No', value: 'no' },
          ],
        }}
      />
    )

    await user.click(screen.getByRole('button', { name: 'Yes' }))
    expect(mutateMock).toHaveBeenCalledWith({ answer: 'yes' })
  })

  it('submits custom free-form answer on form submit', async () => {
    const user = userEvent.setup()
    render(
      <QuestionPrompt
        question={{
          question: 'Provide details',
          task_id: 'task-2',
          options: [],
        }}
      />
    )

    const input = screen.getByPlaceholderText('Type your answer...')
    await user.type(input, 'custom reply')
    await user.click(screen.getByRole('button', { name: /send/i }))

    expect(mutateMock).toHaveBeenCalledWith({ answer: 'custom reply' })
  })
})
