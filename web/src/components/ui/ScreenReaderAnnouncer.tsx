import { useState, useCallback, useRef, useEffect, type ReactNode } from 'react'
import { AnnouncerContext } from './useAnnouncer'

type Priority = 'polite' | 'assertive'

export function ScreenReaderAnnouncer({ children }: { children: ReactNode }) {
  const [politeMessage, setPoliteMessage] = useState('')
  const [assertiveMessage, setAssertiveMessage] = useState('')

  // Track pending timeouts to prevent race conditions when announce is called rapidly
  // We track both "set" (the 0ms deferred set) and "clear" (the 1000ms auto-clear) timeouts
  const timeoutsRef = useRef<{
    politeSet: ReturnType<typeof setTimeout> | null
    politeClear: ReturnType<typeof setTimeout> | null
    assertiveSet: ReturnType<typeof setTimeout> | null
    assertiveClear: ReturnType<typeof setTimeout> | null
  }>({ politeSet: null, politeClear: null, assertiveSet: null, assertiveClear: null })

  // Cleanup timeouts on unmount
  useEffect(() => {
    const t = timeoutsRef.current
    return () => {
      if (t.politeSet) clearTimeout(t.politeSet)
      if (t.politeClear) clearTimeout(t.politeClear)
      if (t.assertiveSet) clearTimeout(t.assertiveSet)
      if (t.assertiveClear) clearTimeout(t.assertiveClear)
    }
  }, [])

  const announce = useCallback((message: string, priority: Priority = 'polite') => {
    const setter = priority === 'assertive' ? setAssertiveMessage : setPoliteMessage
    const setKey = priority === 'assertive' ? 'assertiveSet' : 'politeSet'
    const clearKey = priority === 'assertive' ? 'assertiveClear' : 'politeClear'
    const t = timeoutsRef.current

    // Cancel any pending set/clear timeouts for this priority level
    if (t[setKey]) {
      clearTimeout(t[setKey])
      t[setKey] = null
    }
    if (t[clearKey]) {
      clearTimeout(t[clearKey])
      t[clearKey] = null
    }

    // Clear first so re-announcing the same message still triggers screen readers.
    // setTimeout(fn, 0) defers the set to the next task, same effect as rAF but
    // testable with vi.runAllTimers() without needing to advance animation frames.
    setter('')
    t[setKey] = setTimeout(() => {
      setter(message)
      t[setKey] = null
    }, 0)

    // Schedule clear and track the timeout
    t[clearKey] = setTimeout(() => {
      setter('')
      t[clearKey] = null
    }, 1000)
  }, [])

  return (
    <AnnouncerContext.Provider value={{ announce }}>
      {children}
      <div role="status" aria-live="polite" aria-atomic="true" className="sr-only">
        {politeMessage}
      </div>
      <div role="alert" aria-live="assertive" aria-atomic="true" className="sr-only">
        {assertiveMessage}
      </div>
    </AnnouncerContext.Provider>
  )
}
