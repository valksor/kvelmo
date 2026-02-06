import { act, renderHook } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { useDebounce } from './useDebounce'

describe('useDebounce', () => {
  beforeEach(() => {
    vi.useFakeTimers()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('keeps old value until delay passes', () => {
    const { result, rerender } = renderHook(
      ({ value, delay }) => useDebounce(value, delay),
      { initialProps: { value: 'alpha', delay: 200 } }
    )

    expect(result.current).toBe('alpha')

    rerender({ value: 'beta', delay: 200 })
    expect(result.current).toBe('alpha')

    act(() => {
      vi.advanceTimersByTime(199)
    })
    expect(result.current).toBe('alpha')

    act(() => {
      vi.advanceTimersByTime(1)
    })
    expect(result.current).toBe('beta')
  })

  it('resets timer when value changes quickly', () => {
    const { result, rerender } = renderHook(
      ({ value, delay }) => useDebounce(value, delay),
      { initialProps: { value: 'a', delay: 100 } }
    )

    rerender({ value: 'b', delay: 100 })
    act(() => {
      vi.advanceTimersByTime(50)
    })

    rerender({ value: 'c', delay: 100 })
    act(() => {
      vi.advanceTimersByTime(99)
    })
    expect(result.current).toBe('a')

    act(() => {
      vi.advanceTimersByTime(1)
    })
    expect(result.current).toBe('c')
  })
})
