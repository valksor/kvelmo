import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@/test/test-utils'
import userEvent from '@testing-library/user-event'
import Login from './Login'

describe('Login', () => {
  beforeEach(() => {
    // Mock CSRF and login endpoints
    global.fetch = vi.fn().mockImplementation((url: string, options?: RequestInit) => {
      if (url.includes('/auth/csrf')) {
        return Promise.resolve({
          ok: true,
          json: () => Promise.resolve({ csrf_token: 'test-token' }),
        })
      }
      if (url.includes('/auth/login') && options?.method === 'POST') {
        const body = JSON.parse(options.body as string)
        if (body.username === 'admin' && body.password === 'password') {
          return Promise.resolve({
            ok: true,
            json: () => Promise.resolve({ success: true, username: 'admin', role: 'admin' }),
          })
        }
        return Promise.resolve({
          ok: false,
          json: () => Promise.resolve({ error: 'Invalid credentials' }),
        })
      }
      return Promise.resolve({ ok: false })
    })
  })

  it('renders login form with title', () => {
    render(<Login />, { route: '/login' })

    expect(screen.getByRole('heading', { name: /mehrhof/i })).toBeInTheDocument()
    expect(screen.getByText(/sign in to continue/i)).toBeInTheDocument()
  })

  it('renders username and password inputs', () => {
    render(<Login />, { route: '/login' })

    expect(screen.getByLabelText(/username/i)).toBeInTheDocument()
    expect(screen.getByLabelText(/password/i)).toBeInTheDocument()
  })

  it('renders sign in button disabled when fields are empty', () => {
    render(<Login />, { route: '/login' })

    const submitButton = screen.getByRole('button', { name: /sign in/i })
    expect(submitButton).toBeDisabled()
  })

  it('enables submit button when fields are filled', async () => {
    const user = userEvent.setup()
    render(<Login />, { route: '/login' })

    await user.type(screen.getByLabelText(/username/i), 'admin')
    await user.type(screen.getByLabelText(/password/i), 'password')

    const submitButton = screen.getByRole('button', { name: /sign in/i })
    expect(submitButton).toBeEnabled()
  })

  it('shows loading state during submission', async () => {
    const user = userEvent.setup()

    // Make login hang to see loading state
    global.fetch = vi.fn().mockImplementation((url: string) => {
      if (url.includes('/auth/login')) {
        return new Promise(() => {}) // Never resolves
      }
      return Promise.resolve({
        ok: true,
        json: () => Promise.resolve({}),
      })
    })

    render(<Login />, { route: '/login' })

    await user.type(screen.getByLabelText(/username/i), 'admin')
    await user.type(screen.getByLabelText(/password/i), 'password')
    await user.click(screen.getByRole('button', { name: /sign in/i }))

    // Button should show loading text
    await waitFor(() => {
      expect(screen.getByText(/signing in/i)).toBeInTheDocument()
    })
  })

  it('shows help text about config file', () => {
    render(<Login />, { route: '/login' })

    expect(screen.getByText(/use credentials from your config file/i)).toBeInTheDocument()
  })
})
