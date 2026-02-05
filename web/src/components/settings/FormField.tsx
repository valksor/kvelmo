import type { ReactNode } from 'react'

interface FormFieldProps {
  label: string
  hint?: string
  children: ReactNode
}

export function FormField({ label, hint, children }: FormFieldProps) {
  return (
    <div className="form-control">
      <label className="label">
        <span className="label-text">{label}</span>
      </label>
      {children}
      {hint && (
        <label className="label">
          <span className="label-text-alt text-base-content/60">{hint}</span>
        </label>
      )}
    </div>
  )
}

interface TextInputProps {
  label: string
  hint?: string
  value: string | undefined
  onChange: (value: string) => void
  placeholder?: string
  type?: 'text' | 'password' | 'number'
  disabled?: boolean
}

export function TextInput({
  label,
  hint,
  value,
  onChange,
  placeholder,
  type = 'text',
  disabled,
}: TextInputProps) {
  return (
    <FormField label={label} hint={hint}>
      <input
        type={type}
        className="input input-bordered w-full"
        value={value || ''}
        onChange={(e) => onChange(e.target.value)}
        placeholder={placeholder}
        disabled={disabled}
      />
    </FormField>
  )
}

interface NumberInputProps {
  label: string
  hint?: string
  value: number | undefined
  onChange: (value: number) => void
  min?: number
  max?: number
  step?: number
  disabled?: boolean
}

export function NumberInput({
  label,
  hint,
  value,
  onChange,
  min,
  max,
  step,
  disabled,
}: NumberInputProps) {
  return (
    <FormField label={label} hint={hint}>
      <input
        type="number"
        className="input input-bordered w-full"
        value={value ?? ''}
        onChange={(e) => onChange(Number(e.target.value))}
        min={min}
        max={max}
        step={step}
        disabled={disabled}
      />
    </FormField>
  )
}

interface CheckboxProps {
  label: string
  hint?: string
  checked: boolean | undefined
  onChange: (checked: boolean) => void
  disabled?: boolean
}

export function Checkbox({ label, hint, checked, onChange, disabled }: CheckboxProps) {
  return (
    <div className="form-control">
      <label className="label cursor-pointer justify-start gap-3">
        <input
          type="checkbox"
          className="checkbox checkbox-primary"
          checked={checked || false}
          onChange={(e) => onChange(e.target.checked)}
          disabled={disabled}
        />
        <span className="label-text">{label}</span>
      </label>
      {hint && (
        <span className="text-xs text-base-content/60 ml-10">{hint}</span>
      )}
    </div>
  )
}

interface SelectProps {
  label: string
  hint?: string
  value: string | undefined
  onChange: (value: string) => void
  options: { value: string; label: string }[]
  disabled?: boolean
}

export function Select({ label, hint, value, onChange, options, disabled }: SelectProps) {
  return (
    <FormField label={label} hint={hint}>
      <select
        className="select select-bordered w-full"
        value={value || ''}
        onChange={(e) => onChange(e.target.value)}
        disabled={disabled}
      >
        <option value="">Select...</option>
        {options.map((opt) => (
          <option key={opt.value} value={opt.value}>
            {opt.label}
          </option>
        ))}
      </select>
    </FormField>
  )
}

interface TextAreaProps {
  label: string
  hint?: string
  value: string | undefined
  onChange: (value: string) => void
  placeholder?: string
  rows?: number
  disabled?: boolean
}

export function TextArea({
  label,
  hint,
  value,
  onChange,
  placeholder,
  rows = 3,
  disabled,
}: TextAreaProps) {
  return (
    <FormField label={label} hint={hint}>
      <textarea
        className="textarea textarea-bordered w-full"
        value={value || ''}
        onChange={(e) => onChange(e.target.value)}
        placeholder={placeholder}
        rows={rows}
        disabled={disabled}
      />
    </FormField>
  )
}

interface CollapseSectionProps {
  title: string
  defaultOpen?: boolean
  children: ReactNode
}

export function CollapseSection({ title, defaultOpen, children }: CollapseSectionProps) {
  return (
    <div className="collapse collapse-arrow bg-base-200">
      <input type="checkbox" defaultChecked={defaultOpen} />
      <div className="collapse-title font-medium">{title}</div>
      <div className="collapse-content">
        <div className="pt-2 space-y-4">{children}</div>
      </div>
    </div>
  )
}
