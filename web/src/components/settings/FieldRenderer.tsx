import type { FieldSchema } from '@/types/schema'
import { TextInput, NumberInput, Checkbox, Select, TextArea } from './FormField'

interface FieldRendererProps {
  field: FieldSchema
  value: unknown
  error?: string
  onChange: (value: unknown) => void
  disabled?: boolean
}

/**
 * Renders a single field based on its schema type.
 * Maps schema field types to existing FormField components.
 */
export function FieldRenderer({ field, value, error, onChange, disabled }: FieldRendererProps) {
  switch (field.type) {
    case 'string':
      return (
        <TextInput
          label={field.label}
          hint={field.description}
          error={error}
          value={value as string | undefined}
          onChange={onChange}
          placeholder={field.placeholder}
          disabled={disabled}
          required={field.validation?.required}
        />
      )

    case 'password':
      return (
        <TextInput
          label={field.label}
          hint={field.description}
          error={error}
          value={value as string | undefined}
          onChange={onChange}
          placeholder={field.placeholder}
          type="password"
          disabled={disabled}
          required={field.validation?.required}
        />
      )

    case 'textarea':
      return (
        <TextArea
          label={field.label}
          hint={field.description}
          error={error}
          value={value as string | undefined}
          onChange={onChange}
          placeholder={field.placeholder}
          disabled={disabled}
        />
      )

    case 'number':
      return (
        <NumberInput
          label={field.label}
          hint={field.description}
          error={error}
          value={value as number | undefined}
          onChange={onChange}
          min={field.validation?.min}
          max={field.validation?.max}
          disabled={disabled}
        />
      )

    case 'boolean':
      return (
        <Checkbox
          label={field.label}
          hint={field.description}
          checked={value as boolean | undefined}
          onChange={onChange}
          disabled={disabled}
        />
      )

    case 'select':
      return (
        <Select
          label={field.label}
          hint={field.description}
          error={error}
          value={value as string | undefined}
          onChange={onChange}
          options={field.options ?? []}
          disabled={disabled}
        />
      )

    default:
      // Fallback to text input for unknown types
      return (
        <TextInput
          label={field.label}
          hint={field.description}
          error={error}
          value={String(value ?? '')}
          onChange={onChange}
          placeholder={field.placeholder}
          disabled={disabled}
        />
      )
  }
}
