import { ChevronRight } from 'lucide-react'
import { useId, useState, type ReactNode } from 'react'

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
  const [isOpen, setIsOpen] = useState(defaultOpen ?? false)
  const contentID = useId()

  return (
    <section className="rounded-xl border border-base-300/70 bg-base-100 shadow-sm overflow-hidden">
      <button
        type="button"
        className="w-full px-4 py-3 flex items-center justify-between gap-3 text-left hover:bg-base-200/60 transition-colors"
        onClick={() => setIsOpen((prev) => !prev)}
        aria-expanded={isOpen}
        aria-controls={contentID}
      >
        <span className="font-medium text-base-content">{title}</span>
        <span className="inline-flex items-center justify-center rounded-md bg-base-200 px-1.5 py-1 text-base-content/70">
          <ChevronRight
            size={16}
            className={`transition-transform ${isOpen ? 'rotate-90' : 'rotate-0'}`}
            aria-hidden="true"
          />
        </span>
      </button>

      {isOpen && (
        <div id={contentID} className="border-t border-base-300/70 px-4 pb-4">
          <div className="pt-4 space-y-4">{children}</div>
        </div>
      )}
    </section>
  )
}
