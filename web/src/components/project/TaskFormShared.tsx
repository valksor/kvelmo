import { useId, useRef } from 'react'
import { Upload } from 'lucide-react'
import { TASK_SOURCE_PROVIDERS } from '@/constants/taskOptions'

interface SourceTypeButtonProps {
  active: boolean
  onClick: () => void
  icon: React.ReactNode
  label: string
}

export function SourceTypeButton({ active, onClick, icon, label }: SourceTypeButtonProps) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={`p-3 rounded-xl border-2 flex flex-col items-center gap-1 transition-all ${
        active
          ? 'border-primary bg-primary/10 text-primary'
          : 'border-base-300 hover:border-primary/50 text-base-content/60 hover:text-base-content'
      }`}
    >
      {icon}
      <span className="text-xs font-medium">{label}</span>
    </button>
  )
}

interface FileInputProps {
  file: File | null
  onFileChange: (file: File | null) => void
}

export function FileInput({ file, onFileChange }: FileInputProps) {
  const fileInputRef = useRef<HTMLInputElement>(null)
  const fileInputID = useId()

  const handleDrop = (e: React.DragEvent) => {
    e.preventDefault()
    const droppedFile = e.dataTransfer.files[0]
    if (droppedFile) onFileChange(droppedFile)
  }

  return (
    <div
      role="button"
      tabIndex={0}
      className="border-2 border-dashed border-base-300 rounded-xl p-6 text-center hover:border-primary hover:bg-primary/5 transition-all cursor-pointer"
      onClick={() => fileInputRef.current?.click()}
      onKeyDown={(e) => {
        if (e.key === 'Enter' || e.key === ' ') {
          e.preventDefault()
          fileInputRef.current?.click()
        }
      }}
      onDragOver={(e) => e.preventDefault()}
      onDrop={handleDrop}
    >
      <input
        ref={fileInputRef}
        type="file"
        id={fileInputID}
        accept=".md,.txt,.markdown,.zip,.tar.gz"
        className="hidden"
        onChange={(e) => onFileChange(e.target.files?.[0] || null)}
      />
      <Upload className="w-10 h-10 mx-auto text-base-content/40 mb-2" aria-hidden="true" />
      <p className="text-sm text-base-content/60">
        Drop file here or <span className="text-primary font-medium">browse</span>
      </p>
      <p className="text-xs text-base-content/40 mt-1">.md, .txt, .zip, .tar.gz (max 10MB)</p>
      {file && <p className="text-sm text-primary mt-2 font-medium">Selected: {file.name}</p>}
    </div>
  )
}

interface TextInputProps {
  value: string
  onChange: (value: string) => void
}

export function TextInput({ value, onChange }: TextInputProps) {
  return (
    <div className="form-control">
      <label className="label py-1" htmlFor="task-text-source">
        <span className="label-text">Task content</span>
      </label>
      <textarea
        id="task-text-source"
        rows={6}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        placeholder="# Task Title&#10;&#10;Describe what you want to accomplish...&#10;&#10;## Requirements&#10;- First requirement&#10;- Second requirement"
        className="textarea textarea-bordered w-full font-mono text-sm"
      />
      <p className="label py-1">
        <span className="label-text-alt text-base-content/60">Use Markdown for better structure</span>
      </p>
    </div>
  )
}

interface ReferenceInputProps {
  provider: string
  onProviderChange: (value: string) => void
  referenceId: string
  onReferenceIdChange: (value: string) => void
}

export function ReferenceInput({
  provider,
  onProviderChange,
  referenceId,
  onReferenceIdChange,
}: ReferenceInputProps) {
  return (
    <div className="grid grid-cols-2 gap-3">
      <div className="form-control">
        <label className="label py-1" htmlFor="ref-provider">
          <span className="label-text">Provider</span>
        </label>
        <select
          id="ref-provider"
          value={provider}
          onChange={(e) => onProviderChange(e.target.value)}
          className="select select-bordered w-full"
        >
          {TASK_SOURCE_PROVIDERS.map((p) => (
            <option key={p.value} value={p.value}>
              {p.label}
            </option>
          ))}
        </select>
      </div>
      <div className="form-control">
        <label className="label py-1" htmlFor="ref-id">
          <span className="label-text">Reference ID</span>
        </label>
        <input
          id="ref-id"
          type="text"
          value={referenceId}
          onChange={(e) => onReferenceIdChange(e.target.value)}
          placeholder="123 or PROJECT-123"
          className="input input-bordered w-full font-mono text-sm"
        />
      </div>
    </div>
  )
}

export function ErrorMessage({ message }: { message: string }) {
  return (
    <div className="alert alert-error text-sm">
      <span>{message}</span>
    </div>
  )
}
