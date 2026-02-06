import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { render, screen, waitFor } from '@/test/test-utils'
import { mockApiEndpoints } from '@/test/mocks'
import Chat from './Chat'

describe('Chat', () => {
  beforeEach(() => {
    Object.defineProperty(window.HTMLElement.prototype, 'scrollIntoView', {
      configurable: true,
      value: vi.fn(),
      writable: true,
    })
  })

  it('renders detailed help output from command metadata', async () => {
    mockApiEndpoints({
      '/api/v1/interactive/command': {
        success: true,
        message: 'Available commands',
        commands: [
          {
            name: 'plan',
            description: 'Generate specifications',
            requires_task: true,
          },
          {
            name: 'help',
            aliases: ['?'],
            description: 'Show available commands',
            requires_task: false,
          },
        ],
      },
    })

    const user = userEvent.setup()
    render(<Chat />)

    await user.type(screen.getByRole('textbox'), 'help{enter}')

    await waitFor(() => {
      expect(screen.getByText(/Available commands:/i)).toBeInTheDocument()
    })
    expect(screen.getByText(/- help \(aliases: \?\): Show available commands/i)).toBeInTheDocument()
    expect(
      screen.getByText(/- plan: Generate specifications \[active task required\]/i)
    ).toBeInTheDocument()
  })

  it('falls back to response message for non-help commands', async () => {
    mockApiEndpoints({
      '/api/v1/interactive/command': {
        success: true,
        message: 'Task status: idle',
      },
    })

    const user = userEvent.setup()
    render(<Chat />)

    await user.type(screen.getByRole('textbox'), 'status{enter}')

    await waitFor(() => {
      expect(screen.getByText('Task status: idle')).toBeInTheDocument()
    })
  })

  it('shows command errors once in chat without duplicate alert banner', async () => {
    global.fetch = vi.fn().mockImplementation((url: string) => {
      const path = url.replace(/^.*\/api\/v1/, '/api/v1').split('?')[0]

      if (path === '/api/v1/auth/csrf') {
        return Promise.resolve({
          ok: true,
          status: 200,
          json: () => Promise.resolve({ csrf_token: 'test-csrf-token' }),
          text: () => Promise.resolve('{"csrf_token":"test-csrf-token"}'),
        })
      }

      if (path === '/api/v1/interactive/command') {
        return Promise.resolve({
          ok: false,
          status: 400,
          json: () => Promise.resolve({ error: 'no active task' }),
          text: () => Promise.resolve('no active task'),
        })
      }

      return Promise.resolve({
        ok: false,
        status: 404,
        json: () => Promise.resolve({ error: 'Not found' }),
        text: () => Promise.resolve('Not found'),
      })
    })

    const user = userEvent.setup()
    render(<Chat />)

    await user.type(screen.getByRole('textbox'), 'status{enter}')

    await waitFor(() => {
      expect(screen.getByText('no active task')).toBeInTheDocument()
    })
    expect(screen.getAllByText('no active task')).toHaveLength(1)
    expect(screen.queryByText('Dismiss')).not.toBeInTheDocument()
  })
})
