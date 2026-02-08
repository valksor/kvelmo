import type { Condition, FieldSchema } from '@/types/schema'

/**
 * Get nested value by dot-notation path.
 * Returns undefined for missing or invalid paths.
 */
export function getPath(obj: Record<string, unknown>, path: string): unknown {
  if (!path) return obj
  const keys = path.split('.')
  let current: unknown = obj

  for (const key of keys) {
    if (current == null || typeof current !== 'object') {
      return undefined
    }
    current = (current as Record<string, unknown>)[key]
  }

  return current
}

/**
 * Set nested value by dot-notation path (immutable).
 * Creates intermediate objects as needed.
 */
export function setPath(
  obj: Record<string, unknown>,
  path: string,
  value: unknown
): Record<string, unknown> {
  if (!path) return obj

  const keys = path.split('.')
  const result = { ...obj }
  let current: Record<string, unknown> = result

  for (let i = 0; i < keys.length - 1; i++) {
    const key = keys[i]
    const existing = current[key]
    current[key] = existing != null && typeof existing === 'object'
      ? { ...(existing as Record<string, unknown>) }
      : {}
    current = current[key] as Record<string, unknown>
  }

  const finalKey = keys[keys.length - 1]
  current[finalKey] = value

  return result
}

/**
 * Convert dot-notation path to array format for updateField compatibility.
 */
export function pathToArray(path: string): string[] {
  return path.split('.')
}

/**
 * Evaluate showWhen condition against current values.
 */
export function evaluateShowWhen(
  condition: Condition | undefined,
  values: Record<string, unknown>
): boolean {
  if (!condition) return true

  const fieldValue = getPath(values, condition.field)

  if (condition.equals !== undefined) {
    return fieldValue === condition.equals
  }

  if (condition.notEquals !== undefined) {
    return fieldValue !== condition.notEquals
  }

  // If only field is specified, check truthiness
  return Boolean(fieldValue)
}

/**
 * Validate a field value against its validation rules.
 * Returns error message or undefined if valid.
 */
export function validateField(field: FieldSchema, value: unknown): string | undefined {
  const rules = field.validation
  if (!rules) return undefined

  // Check required
  if (rules.required && isEmpty(value)) {
    return `${field.label} is required`
  }

  // Skip further validation if value is empty and not required
  if (isEmpty(value)) {
    return undefined
  }

  // String validation
  if (typeof value === 'string') {
    if (rules.maxLength != null && value.length > rules.maxLength) {
      return `Maximum ${rules.maxLength} characters`
    }
    if (rules.pattern) {
      try {
        if (!new RegExp(rules.pattern).test(value)) {
          return rules.patternMessage ?? `${field.label} has an invalid format`
        }
      } catch {
        // Invalid regex pattern - skip validation
      }
    }
  }

  // Numeric validation
  if (typeof value === 'number') {
    if (rules.min != null && value < rules.min) {
      return `Minimum value is ${rules.min}`
    }
    if (rules.max != null && value > rules.max) {
      return `Maximum value is ${rules.max}`
    }
  }

  return undefined
}

/**
 * Check if a value is considered empty.
 */
function isEmpty(value: unknown): boolean {
  if (value == null) return true
  if (typeof value === 'string') return value === ''
  if (typeof value === 'number') return false // 0 is not empty for numbers
  if (typeof value === 'boolean') return false // false is not empty for booleans
  return false
}

/**
 * Validate all fields in the schema against current values.
 * Returns a map of path -> error message for invalid fields.
 */
export function validateSchema(
  fields: FieldSchema[],
  values: Record<string, unknown>
): Record<string, string> {
  const errors: Record<string, string> = {}

  for (const field of fields) {
    const value = getPath(values, field.path)
    const error = validateField(field, value)
    if (error) {
      errors[field.path] = error
    }
  }

  return errors
}
