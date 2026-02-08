import type { ReactNode } from 'react'
import type { SettingsSchema, FieldSchema, SectionSchema } from '@/types/schema'
import { CollapseSection } from './FormField'
import { FieldRenderer } from './FieldRenderer'
import { getPath, pathToArray, evaluateShowWhen } from './schemaUtils'

/**
 * Props for custom field renderers.
 * Use for complex fields like maps or arrays that need special UI.
 */
export interface CustomFieldProps {
  field: FieldSchema
  value: unknown
  onChange: (value: unknown) => void
}

interface DynamicFormProps {
  /** Schema defining sections and fields */
  schema: SettingsSchema
  /** Current values for all fields */
  values: Record<string, unknown>
  /** Validation errors keyed by field path */
  errors?: Record<string, string>
  /** Callback when a field value changes (receives path array for updateField compatibility) */
  onChange: (path: string[], value: unknown) => void
  /** Filter sections by category */
  category?: 'core' | 'providers' | 'features'
  /** When true, hides advanced fields */
  simpleMode?: boolean
  /** Custom renderers for specific field paths */
  customRenderers?: Record<string, React.ComponentType<CustomFieldProps>>
  /** Default open state for sections (default: first section open) */
  defaultOpen?: boolean | 'first'
  /** Additional content to render after all sections */
  children?: ReactNode
}

/**
 * DynamicForm renders a settings form from a schema definition.
 *
 * This component enables "define once" settings by:
 * 1. Reading field definitions from the schema (generated from Go struct tags)
 * 2. Dynamically rendering appropriate input components
 * 3. Handling conditional visibility (showWhen)
 * 4. Supporting both simple and advanced modes
 *
 * For complex fields (maps, arrays), use customRenderers to provide specialized UI.
 */
export function DynamicForm({
  schema,
  values,
  errors,
  onChange,
  category,
  simpleMode = false,
  customRenderers,
  defaultOpen = 'first',
  children,
}: DynamicFormProps) {
  // Filter sections by category if specified
  const sections = category
    ? schema.sections.filter((s) => s.category === category)
    : schema.sections

  const handleFieldChange = (path: string, value: unknown) => {
    onChange(pathToArray(path), value)
  }

  return (
    <div className="space-y-4">
      {sections.map((section, index) => (
        <DynamicSection
          key={section.id}
          section={section}
          values={values}
          errors={errors}
          onChange={handleFieldChange}
          simpleMode={simpleMode}
          customRenderers={customRenderers}
          defaultOpen={
            defaultOpen === 'first' ? index === 0 : Boolean(defaultOpen)
          }
        />
      ))}

      {children}
    </div>
  )
}

interface DynamicSectionProps {
  section: SectionSchema
  values: Record<string, unknown>
  errors?: Record<string, string>
  onChange: (path: string, value: unknown) => void
  simpleMode: boolean
  customRenderers?: Record<string, React.ComponentType<CustomFieldProps>>
  defaultOpen: boolean
}

function DynamicSection({
  section,
  values,
  errors,
  onChange,
  simpleMode,
  customRenderers,
  defaultOpen,
}: DynamicSectionProps) {
  // Filter fields based on mode and visibility conditions
  const visibleFields = section.fields.filter((field) => {
    // In simple mode, only show fields marked as simple
    if (simpleMode && !field.simple) {
      return false
    }

    // Evaluate showWhen conditions
    if (!evaluateShowWhen(field.showWhen, values)) {
      return false
    }

    return true
  })

  // Don't render empty sections
  if (visibleFields.length === 0) {
    return null
  }

  return (
    <CollapseSection title={section.title} defaultOpen={defaultOpen}>
      {section.description && (
        <p className="text-sm text-base-content/60 mb-4">{section.description}</p>
      )}
      <div className="space-y-4">
        {visibleFields.map((field) => (
          <DynamicField
            key={field.path}
            field={field}
            value={getPath(values, field.path)}
            error={errors?.[field.path]}
            onChange={(v) => onChange(field.path, v)}
            customRenderers={customRenderers}
          />
        ))}
      </div>
    </CollapseSection>
  )
}

interface DynamicFieldProps {
  field: FieldSchema
  value: unknown
  error?: string
  onChange: (value: unknown) => void
  customRenderers?: Record<string, React.ComponentType<CustomFieldProps>>
}

function DynamicField({
  field,
  value,
  error,
  onChange,
  customRenderers,
}: DynamicFieldProps) {
  // Check for custom renderer
  const CustomRenderer = customRenderers?.[field.path]
  if (CustomRenderer) {
    return <CustomRenderer field={field} value={value} onChange={onChange} />
  }

  // Use default field renderer
  return (
    <FieldRenderer
      field={field}
      value={value}
      error={error}
      onChange={onChange}
    />
  )
}
