import { Settings2, Wrench } from 'lucide-react'

interface SettingsModeToggleProps {
  isSimple: boolean
  onToggle: () => void
}

/**
 * Toggle between Simple and Advanced settings views.
 * Simple mode hides technical settings for non-developer users.
 */
export function SettingsModeToggle({ isSimple, onToggle }: SettingsModeToggleProps) {
  return (
    <div className="flex items-center gap-2">
      <span className="text-sm text-base-content/60">{isSimple ? 'Simple' : 'Advanced'}</span>
      <label className="swap swap-rotate">
        <input
          type="checkbox"
          checked={!isSimple}
          onChange={onToggle}
          aria-label={`Switch to ${isSimple ? 'advanced' : 'simple'} settings view`}
        />
        <div className="swap-off" title="Simple mode - essential settings only">
          <Settings2 size={18} className="text-base-content/60" />
        </div>
        <div className="swap-on" title="Advanced mode - all settings">
          <Wrench size={18} className="text-primary" />
        </div>
      </label>
    </div>
  )
}
