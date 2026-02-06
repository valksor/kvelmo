import { describe, expect, it } from 'vitest'
import { getStateConfig, getStateConfigWithProgress } from './stateConfig'

describe('getStateConfig', () => {
  it('returns known state config', () => {
    const config = getStateConfig('implementing')
    expect(config.icon).toBe('🔨')
    expect(config.badge).toBe('badge-primary')
  })

  it('falls back to idle for unknown state', () => {
    const config = getStateConfig('unknown-state')
    expect(config.icon).toBe('⏸️')
    expect(config.badge).toBe('badge-ghost')
  })
})

describe('getStateConfigWithProgress', () => {
  it('uses state directly when non-idle', () => {
    const config = getStateConfigWithProgress('reviewing', 'implemented')
    expect(config.displayState).toBe('reviewing')
    expect(config.icon).toBe('🔍')
  })

  it('uses progress phase for idle state when phase is beyond started', () => {
    const config = getStateConfigWithProgress('idle', 'planned')
    expect(config.displayState).toBe('planned')
    expect(config.icon).toBe('📋')
  })

  it('keeps idle display for started/no phase', () => {
    expect(getStateConfigWithProgress('idle', 'started').displayState).toBe('idle')
    expect(getStateConfigWithProgress('idle').displayState).toBe('idle')
  })
})
