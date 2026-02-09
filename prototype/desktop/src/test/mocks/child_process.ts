import { vi } from 'vitest'
import { EventEmitter } from 'events'
import type { ChildProcess, SpawnSyncReturns } from 'child_process'

/**
 * Create a mock child process that extends EventEmitter.
 * Use this for testing spawn() calls.
 */
export function createMockChildProcess(): ChildProcess {
  const proc = new EventEmitter() as ChildProcess

  // Mock stdout/stderr as EventEmitters
  proc.stdout = new EventEmitter() as NodeJS.ReadableStream
  proc.stderr = new EventEmitter() as NodeJS.ReadableStream
  proc.stdin = null as unknown as NodeJS.WritableStream

  // Mock methods and properties
  proc.kill = vi.fn(() => true)
  proc.pid = 12345
  proc.connected = true
  proc.exitCode = null
  proc.signalCode = null
  proc.killed = false
  proc.spawnargs = []
  proc.spawnfile = ''

  // Add ref/unref methods
  proc.ref = vi.fn(() => proc)
  proc.unref = vi.fn(() => proc)
  proc.disconnect = vi.fn()
  proc.send = vi.fn()

  // Add off method for listener removal (matches Node.js EventEmitter)
  proc.off = vi.fn((event: string, listener: (...args: unknown[]) => void) => {
    proc.removeListener(event, listener)
    return proc
  })

  return proc
}

/**
 * Create a mock spawnSync result.
 * Use this for testing spawnSync() calls.
 */
export function createSpawnSyncResult(
  status: number = 0,
  stdout: string = '',
  stderr: string = ''
): SpawnSyncReturns<Buffer> {
  return {
    status,
    stdout: Buffer.from(stdout),
    stderr: Buffer.from(stderr),
    pid: 12345,
    output: [null, Buffer.from(stdout), Buffer.from(stderr)],
    signal: null,
    error: undefined,
  }
}

// Mock spawn function
export const mockSpawn = vi.fn()

// Mock spawnSync function
export const mockSpawnSync = vi.fn()
