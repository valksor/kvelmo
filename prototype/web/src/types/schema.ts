/**
 * Schema types for dynamic settings form generation.
 * These types mirror the Go schema package (internal/schema/types.go).
 */

export type FieldType = 'string' | 'boolean' | 'number' | 'select' | 'textarea' | 'password'

export interface SelectOption {
  value: string
  label: string
}

export interface ValidationRules {
  required?: boolean
  min?: number
  max?: number
  maxLength?: number
  pattern?: string
  patternMessage?: string
}

export interface Condition {
  field: string
  equals?: unknown
  notEquals?: unknown
}

export interface FieldSchema {
  path: string
  type: FieldType
  label: string
  description?: string
  placeholder?: string
  default?: unknown
  options?: SelectOption[]
  validation?: ValidationRules
  sensitive?: boolean
  showWhen?: Condition
  advanced?: boolean
  simple?: boolean
}

export interface SectionSchema {
  id: string
  title: string
  description?: string
  icon?: string
  category: 'core' | 'providers' | 'features'
  fields: FieldSchema[]
}

export interface SettingsSchema {
  version: string
  sections: SectionSchema[]
}

/**
 * Validation error returned from the server.
 */
export interface ValidationError {
  path: string
  message: string
}

/**
 * V2 settings response includes both schema and values.
 */
export interface SettingsResponseV2 {
  schema: SettingsSchema
  values: Record<string, unknown>
}
