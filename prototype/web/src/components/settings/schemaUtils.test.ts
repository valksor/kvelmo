import { describe, it, expect } from 'vitest'
import {
  getPath,
  setPath,
  pathToArray,
  evaluateShowWhen,
  validateField,
  validateSchema,
} from './schemaUtils'
import type { FieldSchema, Condition } from '@/types/schema'

describe('getPath', () => {
  it('gets simple 2-level path: git.commit_prefix', () => {
    const obj = { git: { commit_prefix: '[{key}]' } }
    expect(getPath(obj, 'git.commit_prefix')).toBe('[{key}]')
  })

  it('gets 4-level deep path: security.scanners.sast.enabled', () => {
    const obj = { security: { scanners: { sast: { enabled: true } } } }
    expect(getPath(obj, 'security.scanners.sast.enabled')).toBe(true)
  })

  it('gets 3-level path with map key: agent.steps.planning', () => {
    const obj = { agent: { steps: { planning: { name: 'claude' } } } }
    expect(getPath(obj, 'agent.steps.planning')).toEqual({ name: 'claude' })
  })

  it('gets 3-level path: budget.monthly.limit', () => {
    const obj = { budget: { monthly: { limit: 100 } } }
    expect(getPath(obj, 'budget.monthly.limit')).toBe(100)
  })

  it('gets 3-level path: github.comments.on_pr_created', () => {
    const obj = { github: { comments: { on_pr_created: false } } }
    expect(getPath(obj, 'github.comments.on_pr_created')).toBe(false)
  })

  it('returns undefined for missing intermediate keys', () => {
    const obj = { git: {} }
    expect(getPath(obj, 'git.commit_prefix')).toBeUndefined()
  })

  it('returns undefined for completely missing path', () => {
    const obj = {}
    expect(getPath(obj, 'nonexistent.deep.path')).toBeUndefined()
  })

  it('returns undefined when intermediate is null', () => {
    const obj = { git: null }
    expect(getPath(obj, 'git.commit_prefix')).toBeUndefined()
  })

  it('returns undefined when intermediate is non-object', () => {
    const obj = { git: 'string-value' }
    expect(getPath(obj, 'git.commit_prefix')).toBeUndefined()
  })

  it('handles empty path by returning the object itself', () => {
    const obj = { git: { auto_commit: true } }
    expect(getPath(obj, '')).toEqual(obj)
  })

  it('handles boolean false value correctly', () => {
    const obj = { git: { sign_commits: false } }
    expect(getPath(obj, 'git.sign_commits')).toBe(false)
  })

  it('handles zero value correctly', () => {
    const obj = { budget: { per_task: { max_cost: 0 } } }
    expect(getPath(obj, 'budget.per_task.max_cost')).toBe(0)
  })
})

describe('setPath', () => {
  it('sets simple 2-level path', () => {
    const obj = { git: { auto_commit: true } }
    const result = setPath(obj, 'git.commit_prefix', '[{key}]')
    expect(result.git).toEqual({ auto_commit: true, commit_prefix: '[{key}]' })
  })

  it('creates intermediate objects when missing', () => {
    const obj = {}
    const result = setPath(obj, 'git.commit_prefix', 'test')
    expect(result).toEqual({ git: { commit_prefix: 'test' } })
  })

  it('overwrites existing nested values', () => {
    const obj = { git: { commit_prefix: 'old' } }
    const result = setPath(obj, 'git.commit_prefix', 'new')
    expect(result.git).toEqual({ commit_prefix: 'new' })
  })

  it('sets 4-level deep path', () => {
    const obj = {}
    const result = setPath(obj, 'security.scanners.sast.enabled', true)
    expect(result).toEqual({ security: { scanners: { sast: { enabled: true } } } })
  })

  it('preserves sibling keys when setting nested value', () => {
    const obj = { git: { auto_commit: true, sign_commits: false } }
    const result = setPath(obj, 'git.auto_commit', false)
    expect(result.git).toEqual({ auto_commit: false, sign_commits: false })
  })

  it('is immutable - does not modify original', () => {
    const obj = { git: { auto_commit: true } }
    const result = setPath(obj, 'git.auto_commit', false)
    expect(obj.git.auto_commit).toBe(true)
    expect(result.git.auto_commit).toBe(false)
  })

  it('handles undefined at various levels by creating path', () => {
    const obj = { git: undefined }
    const result = setPath(obj as Record<string, unknown>, 'git.auto_commit', true)
    expect(result).toEqual({ git: { auto_commit: true } })
  })

  it('handles setting null value', () => {
    const obj = { git: { commit_prefix: 'test' } }
    const result = setPath(obj, 'git.commit_prefix', null)
    expect(result.git.commit_prefix).toBeNull()
  })

  it('returns unchanged object for empty path', () => {
    const obj = { git: { auto_commit: true } }
    const result = setPath(obj, '', 'ignored')
    expect(result).toEqual(obj)
  })
})

describe('pathToArray', () => {
  it('converts dot-notation to array', () => {
    expect(pathToArray('git.commit_prefix')).toEqual(['git', 'commit_prefix'])
  })

  it('handles single segment', () => {
    expect(pathToArray('agent')).toEqual(['agent'])
  })

  it('handles deep path', () => {
    expect(pathToArray('a.b.c.d.e')).toEqual(['a', 'b', 'c', 'd', 'e'])
  })
})

describe('evaluateShowWhen', () => {
  it('returns true when condition is undefined', () => {
    expect(evaluateShowWhen(undefined, {})).toBe(true)
  })

  it('evaluates equals condition correctly', () => {
    const condition: Condition = { field: 'git.auto_commit', equals: true }
    expect(evaluateShowWhen(condition, { git: { auto_commit: true } })).toBe(true)
    expect(evaluateShowWhen(condition, { git: { auto_commit: false } })).toBe(false)
  })

  it('evaluates notEquals condition correctly', () => {
    const condition: Condition = { field: 'mode', notEquals: 'simple' }
    expect(evaluateShowWhen(condition, { mode: 'advanced' })).toBe(true)
    expect(evaluateShowWhen(condition, { mode: 'simple' })).toBe(false)
  })

  it('evaluates truthiness when only field specified', () => {
    const condition: Condition = { field: 'budget.enabled' }
    expect(evaluateShowWhen(condition, { budget: { enabled: true } })).toBe(true)
    expect(evaluateShowWhen(condition, { budget: { enabled: false } })).toBe(false)
    expect(evaluateShowWhen(condition, { budget: {} })).toBe(false)
  })

  it('handles nested field paths', () => {
    const condition: Condition = { field: 'security.enabled', equals: true }
    expect(evaluateShowWhen(condition, { security: { enabled: true } })).toBe(true)
    expect(evaluateShowWhen(condition, {})).toBe(false)
  })
})

describe('validateField', () => {
  it('returns undefined when no validation rules', () => {
    const field: FieldSchema = { path: 'test', type: 'string', label: 'Test' }
    expect(validateField(field, 'any value')).toBeUndefined()
  })

  it('validates required fields', () => {
    const field: FieldSchema = {
      path: 'test',
      type: 'string',
      label: 'Name',
      validation: { required: true },
    }
    expect(validateField(field, '')).toBe('Name is required')
    expect(validateField(field, null)).toBe('Name is required')
    expect(validateField(field, 'value')).toBeUndefined()
  })

  it('validates string maxLength', () => {
    const field: FieldSchema = {
      path: 'test',
      type: 'string',
      label: 'Code',
      validation: { maxLength: 5 },
    }
    expect(validateField(field, 'toolong')).toBe('Maximum 5 characters')
    expect(validateField(field, 'ok')).toBeUndefined()
  })

  it('validates string pattern', () => {
    const field: FieldSchema = {
      path: 'test',
      type: 'string',
      label: 'Email',
      validation: { pattern: '^[^@]+@[^@]+$', patternMessage: 'Must be a valid email' },
    }
    expect(validateField(field, 'invalid')).toBe('Must be a valid email')
    expect(validateField(field, 'valid@example.com')).toBeUndefined()
  })

  it('validates number min', () => {
    const field: FieldSchema = {
      path: 'test',
      type: 'number',
      label: 'Count',
      validation: { min: 1 },
    }
    expect(validateField(field, 0)).toBe('Minimum value is 1')
    expect(validateField(field, 1)).toBeUndefined()
  })

  it('validates number max', () => {
    const field: FieldSchema = {
      path: 'test',
      type: 'number',
      label: 'Count',
      validation: { max: 100 },
    }
    expect(validateField(field, 101)).toBe('Maximum value is 100')
    expect(validateField(field, 100)).toBeUndefined()
  })

  it('skips validation for empty non-required fields', () => {
    const field: FieldSchema = {
      path: 'test',
      type: 'string',
      label: 'Optional',
      validation: { maxLength: 5 },
    }
    expect(validateField(field, '')).toBeUndefined()
    expect(validateField(field, null)).toBeUndefined()
  })
})

describe('validateSchema', () => {
  it('returns empty object when all valid', () => {
    const fields: FieldSchema[] = [
      { path: 'name', type: 'string', label: 'Name', validation: { required: true } },
    ]
    const errors = validateSchema(fields, { name: 'Test' })
    expect(errors).toEqual({})
  })

  it('returns errors for invalid fields', () => {
    const fields: FieldSchema[] = [
      { path: 'name', type: 'string', label: 'Name', validation: { required: true } },
      { path: 'count', type: 'number', label: 'Count', validation: { min: 1 } },
    ]
    const errors = validateSchema(fields, { name: '', count: 0 })
    expect(errors).toEqual({
      name: 'Name is required',
      count: 'Minimum value is 1',
    })
  })
})
