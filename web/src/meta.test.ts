import { describe, it, expect } from 'vitest'
import { name, storeName, envVar, cliCmd } from './meta'

describe('meta', () => {
  describe('name', () => {
    it('is kvelmo', () => {
      expect(name).toBe('kvelmo')
    })
  })

  describe('storeName', () => {
    it('returns kvelmo-{suffix}', () => {
      expect(storeName('theme')).toBe('kvelmo-theme')
      expect(storeName('layout')).toBe('kvelmo-layout')
      expect(storeName('global')).toBe('kvelmo-global')
    })
  })

  describe('envVar', () => {
    it('returns KVELMO_{SUFFIX}', () => {
      expect(envVar('HOME')).toBe('KVELMO_HOME')
      expect(envVar('socket_dir')).toBe('KVELMO_socket_dir')
    })
  })

  describe('cliCmd', () => {
    it("returns 'kvelmo {sub}'", () => {
      expect(cliCmd('serve')).toBe("'kvelmo serve'")
      expect(cliCmd('config show')).toBe("'kvelmo config show'")
    })
  })
})
