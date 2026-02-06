import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { act, renderHook, waitFor } from '@testing-library/react'
import type { ReactNode } from 'react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import {
  useBrowserClick,
  useBrowserConsole,
  useBrowserCookies,
  useBrowserCoverage,
  useBrowserGoto,
  useBrowserNetwork,
  useBrowserSetCookies,
  useBrowserStatus,
  useBrowserSwitch,
} from './browser'

const apiRequestMock = vi.fn()

vi.mock('./client', () => ({
  apiRequest: (...args: unknown[]) => apiRequestMock(...args),
}))

function createWrapper(queryClient: QueryClient) {
  return function Wrapper({ children }: { children: ReactNode }) {
    return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  }
}

describe('browser api hooks', () => {
  beforeEach(() => {
    apiRequestMock.mockReset()
    apiRequestMock.mockResolvedValue({ success: true })
  })

  it('useBrowserStatus requests browser status', async () => {
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    renderHook(() => useBrowserStatus(), { wrapper: createWrapper(queryClient) })

    await waitFor(() => {
      expect(apiRequestMock).toHaveBeenCalledWith('/browser/status')
    })
  })

  it('useBrowserCookies requests exported cookies', async () => {
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    renderHook(() => useBrowserCookies(), { wrapper: createWrapper(queryClient) })

    await waitFor(() => {
      expect(apiRequestMock).toHaveBeenCalledWith('/browser/cookies')
    })
  })

  it('useBrowserGoto posts url and invalidates browser queries', async () => {
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    const invalidateSpy = vi.spyOn(queryClient, 'invalidateQueries')
    const { result } = renderHook(() => useBrowserGoto(), { wrapper: createWrapper(queryClient) })

    await act(async () => {
      await result.current.mutateAsync('https://example.com')
    })

    expect(apiRequestMock).toHaveBeenCalledWith('/browser/goto', {
      method: 'POST',
      body: JSON.stringify({ url: 'https://example.com' }),
    })
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: ['browser'] })
  })

  it('useBrowserClick posts selector payload', async () => {
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    const { result } = renderHook(() => useBrowserClick(), { wrapper: createWrapper(queryClient) })

    await act(async () => {
      await result.current.mutateAsync({ selector: '#submit' })
    })

    expect(apiRequestMock).toHaveBeenCalledWith('/browser/click', {
      method: 'POST',
      body: JSON.stringify({ selector: '#submit' }),
    })
  })

  it('useBrowserSwitch posts tab id and invalidates browser queries', async () => {
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    const invalidateSpy = vi.spyOn(queryClient, 'invalidateQueries')
    const { result } = renderHook(() => useBrowserSwitch(), { wrapper: createWrapper(queryClient) })

    await act(async () => {
      await result.current.mutateAsync({ tab_id: 'tab-1' })
    })

    expect(apiRequestMock).toHaveBeenCalledWith('/browser/switch', {
      method: 'POST',
      body: JSON.stringify({ tab_id: 'tab-1' }),
    })
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: ['browser'] })
  })

  it('devtools hooks send POST payloads', async () => {
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })

    const network = renderHook(() => useBrowserNetwork(), { wrapper: createWrapper(queryClient) })
    const consoleHook = renderHook(() => useBrowserConsole(), { wrapper: createWrapper(queryClient) })
    const coverage = renderHook(() => useBrowserCoverage(), { wrapper: createWrapper(queryClient) })
    const setCookies = renderHook(() => useBrowserSetCookies(), { wrapper: createWrapper(queryClient) })

    await act(async () => {
      await network.result.current.mutateAsync({ duration: 3 })
      await consoleHook.result.current.mutateAsync({ level: 'warn' })
      await coverage.result.current.mutateAsync({ track_js: true, track_css: false })
      await setCookies.result.current.mutateAsync({
        cookies: [
          {
            name: 'session',
            value: 'abc',
            domain: 'example.com',
            path: '/',
            secure: true,
            http_only: true,
          },
        ],
      })
    })

    expect(apiRequestMock).toHaveBeenCalledWith('/browser/network', {
      method: 'POST',
      body: JSON.stringify({ duration: 3 }),
    })
    expect(apiRequestMock).toHaveBeenCalledWith('/browser/console', {
      method: 'POST',
      body: JSON.stringify({ level: 'warn' }),
    })
    expect(apiRequestMock).toHaveBeenCalledWith('/browser/coverage', {
      method: 'POST',
      body: JSON.stringify({ track_js: true, track_css: false }),
    })
    expect(apiRequestMock).toHaveBeenCalledWith('/browser/cookies', {
      method: 'POST',
      body: JSON.stringify({
        cookies: [
          {
            name: 'session',
            value: 'abc',
            domain: 'example.com',
            path: '/',
            secure: true,
            http_only: true,
          },
        ],
      }),
    })
  })
})
