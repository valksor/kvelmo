import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { AlertTriangle, X, RefreshCw } from 'lucide-react'
import { useReinitConfig } from '@/api/settings'
import type { ConfigVersionInfo } from '@/types/api'

interface ConfigVersionBannerProps {
  versionInfo: ConfigVersionInfo
  projectId?: string
}

/**
 * Warning banner shown when workspace config is outdated.
 * Offers option to re-initialize config while preserving key settings.
 */
export function ConfigVersionBanner({ versionInfo, projectId }: ConfigVersionBannerProps) {
  const { t } = useTranslation('common')
  const [dismissed, setDismissed] = useState(false)
  const reinitMutation = useReinitConfig(projectId)

  // Don't render if not outdated or dismissed
  if (!versionInfo.is_outdated || dismissed) {
    return null
  }

  const handleReinit = async () => {
    // Confirm before reinit
    const confirmed = window.confirm(
      t('config.reinitConfirm', 'Re-initialize config? Your key settings (agent defaults, git patterns, provider tokens) will be preserved.')
    )
    if (!confirmed) return

    try {
      await reinitMutation.mutateAsync()
    } catch {
      // Error is handled by mutation state
    }
  }

  return (
    <div
      role="alert"
      className="alert alert-warning mb-4 flex flex-wrap items-center justify-between gap-2"
    >
      <div className="flex items-center gap-2">
        <AlertTriangle size={20} aria-hidden="true" className="shrink-0" />
        <span>
          {t('config.outdatedWarning', {
            defaultValue: 'Your config is outdated (v{{current}} → v{{required}}). Some features may not work correctly.',
            current: versionInfo.current,
            required: versionInfo.required,
          })}
        </span>
      </div>
      <div className="flex items-center gap-2">
        <button
          type="button"
          className="btn btn-sm btn-warning gap-1"
          onClick={handleReinit}
          disabled={reinitMutation.isPending}
        >
          {reinitMutation.isPending ? (
            <>
              <RefreshCw size={14} className="animate-spin" aria-hidden="true" />
              {t('config.updating', 'Updating...')}
            </>
          ) : (
            t('config.updateConfig', 'Update Config')
          )}
        </button>
        <button
          type="button"
          className="btn btn-sm btn-ghost"
          onClick={() => setDismissed(true)}
          aria-label={t('common.dismiss', 'Dismiss')}
        >
          <X size={16} aria-hidden="true" />
        </button>
      </div>
      {reinitMutation.isError && (
        <div className="w-full text-sm text-error mt-1">
          {t('config.reinitError', 'Failed to update config. Please try again.')}
        </div>
      )}
    </div>
  )
}
