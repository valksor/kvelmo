import { Loader2, AlertCircle } from 'lucide-react'
import { useSettings } from '@/api/settings'
import { DynamicForm } from './DynamicForm'

interface DynamicSettingsProps {
  /** Project ID for global mode */
  projectId?: string
  /** Filter to specific section IDs */
  sectionIds?: string[]
  /** Callback when a field value changes */
  onChange: (path: string[], value: unknown) => void
  /** Current form data (for conditional visibility) */
  values: Record<string, unknown>
  /** When true, hides advanced fields */
  simpleMode?: boolean
  /** Validation errors keyed by field path */
  errors?: Record<string, string>
}

/**
 * DynamicSettings renders settings sections from the v2 schema API.
 *
 * Use this component to progressively migrate from hardcoded settings
 * components to schema-driven rendering. Sections are defined by struct
 * tags in Go (internal/storage/workspace_config.go) and rendered dynamically.
 *
 * Example usage:
 * ```tsx
 * // Render only the Git section dynamically
 * <DynamicSettings
 *   sectionIds={['git']}
 *   values={formData}
 *   onChange={updateField}
 * />
 * ```
 */
export function DynamicSettings({
  projectId,
  sectionIds,
  onChange,
  values,
  simpleMode = false,
  errors,
}: DynamicSettingsProps) {
  const { data, isLoading, error } = useSettings(projectId)

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-8" role="status" aria-label="Loading">
        <Loader2 className="w-6 h-6 animate-spin text-primary" aria-hidden="true" />
      </div>
    )
  }

  if (error) {
    return (
      <div className="alert alert-error" role="alert">
        <AlertCircle size={20} aria-hidden="true" />
        <span>Failed to load schema: {error.message}</span>
      </div>
    )
  }

  if (!data?.schema) {
    return (
      <div className="alert alert-warning" role="alert">
        <AlertCircle size={20} aria-hidden="true" />
        <span>Settings schema not available</span>
      </div>
    )
  }

  // Filter sections if sectionIds specified
  const filteredSchema = sectionIds
    ? {
        ...data.schema,
        sections: data.schema.sections.filter((s) => sectionIds.includes(s.id)),
      }
    : data.schema

  // If no sections match, render nothing
  if (filteredSchema.sections.length === 0) {
    return null
  }

  return (
    <DynamicForm
      schema={filteredSchema}
      values={values}
      errors={errors}
      onChange={onChange}
      simpleMode={simpleMode}
      defaultOpen="first"
    />
  )
}

/**
 * Hook to check if a section has schema support.
 * Use this to conditionally render DynamicSettings vs hardcoded components.
 */
export function useSectionHasSchema(projectId?: string, sectionId?: string) {
  const { data } = useSettings(projectId)

  if (!data?.schema || !sectionId) {
    return false
  }

  return data.schema.sections.some((s) => s.id === sectionId)
}
