import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { AlertCircle, Languages, Loader2, RotateCcw, Save } from 'lucide-react'
import { CollapseSection } from '../FormField'
import { TerminologyEditor, type TerminologyEntry, type OverrideScope } from '../TerminologyEditor'
import { KeyOverrideEditor, type KeyOverrideEntry } from '../KeyOverrideEditor'
import {
  useI18nOverridesByScope,
  useI18nKeys,
  useSaveI18nOverrides,
  createEmptyOverrides,
  type I18nOverrides,
} from '@/api/i18n'

interface TranslationSettingsProps {
  /** Current project name for scope display */
  projectName?: string
  /** Whether project context is available */
  hasProject: boolean
}

/**
 * Translation customization settings.
 * Provides terminology replacements and key override editors.
 * Shown in the Advanced/System tab.
 */
export function TranslationSettings({ projectName, hasProject }: TranslationSettingsProps) {
  const { t, i18n } = useTranslation('settings')
  const currentLanguage = i18n.language

  // Fetch overrides by scope
  const { data: overrideData, isLoading, error } = useI18nOverridesByScope()
  const { data: availableKeys = [] } = useI18nKeys()

  // Save mutations for each scope
  const saveGlobal = useSaveI18nOverrides('global')
  const saveProject = useSaveI18nOverrides('project')

  // Derive initial entries from fetched data using useMemo (avoids setState in useEffect)
  const initialEntries = useMemo(() => {
    if (!overrideData) return { terminology: [] as TerminologyEntry[], keys: [] as KeyOverrideEntry[] }

    // Convert terminology maps to entry arrays
    // Use ?? {} to handle undefined/null from API responses
    const termEntries: TerminologyEntry[] = []

    // Global terminology
    for (const [find, replace] of Object.entries(overrideData.global?.terminology ?? {})) {
      termEntries.push({ find, replace, scope: 'global' })
    }
    // Project terminology (may override global)
    for (const [find, replace] of Object.entries(overrideData.project?.terminology ?? {})) {
      // Remove global entry if project overrides it
      const globalIndex = termEntries.findIndex((e) => e.find === find && e.scope === 'global')
      if (globalIndex !== -1) {
        termEntries.splice(globalIndex, 1)
      }
      termEntries.push({ find, replace, scope: 'project' })
    }

    // Convert key maps to entry arrays
    const keyEntriesArr: KeyOverrideEntry[] = []

    // Global keys
    for (const [lang, keys] of Object.entries(overrideData.global?.keys ?? {})) {
      for (const [key, value] of Object.entries(keys ?? {})) {
        keyEntriesArr.push({ key, value, language: lang, scope: 'global' })
      }
    }
    // Project keys
    for (const [lang, keys] of Object.entries(overrideData.project?.keys ?? {})) {
      for (const [key, value] of Object.entries(keys ?? {})) {
        // Remove global entry if project overrides it
        const globalIndex = keyEntriesArr.findIndex(
          (e) => e.key === key && e.language === lang && e.scope === 'global'
        )
        if (globalIndex !== -1) {
          keyEntriesArr.splice(globalIndex, 1)
        }
        keyEntriesArr.push({ key, value, language: lang, scope: 'project' })
      }
    }

    return { terminology: termEntries, keys: keyEntriesArr }
  }, [overrideData])

  // Local state for user edits (null means use initial values)
  const [editedTerminology, setEditedTerminology] = useState<TerminologyEntry[] | null>(null)
  const [editedKeys, setEditedKeys] = useState<KeyOverrideEntry[] | null>(null)
  // Screen reader announcement for save/reset status
  const [statusAnnouncement, setStatusAnnouncement] = useState<string | null>(null)

  // Use edited values if user has made changes, otherwise use computed initial values
  const terminologyEntries = editedTerminology ?? initialEntries.terminology
  const keyEntries = editedKeys ?? initialEntries.keys
  const hasChanges = editedTerminology !== null || editedKeys !== null

  // Convert entries back to override objects
  const buildOverrides = useMemo(() => {
    return (scope: OverrideScope): I18nOverrides => {
      const overrides = createEmptyOverrides()

      // Terminology for this scope
      for (const entry of terminologyEntries) {
        if (entry.scope === scope) {
          overrides.terminology[entry.find] = entry.replace
        }
      }

      // Keys for this scope
      for (const entry of keyEntries) {
        if (entry.scope === scope) {
          if (!overrides.keys[entry.language]) {
            overrides.keys[entry.language] = {}
          }
          overrides.keys[entry.language][entry.key] = entry.value
        }
      }

      return overrides
    }
  }, [terminologyEntries, keyEntries])

  const handleTerminologyChange = (entries: TerminologyEntry[]) => {
    setEditedTerminology(entries)
  }

  const handleKeyChange = (entries: KeyOverrideEntry[]) => {
    setEditedKeys(entries)
  }

  const handleSave = async () => {
    // Save both scopes
    const globalOverrides = buildOverrides('global')
    const projectOverrides = buildOverrides('project')

    try {
      // Save global first
      await saveGlobal.mutateAsync(globalOverrides)
      // If project context exists, save project overrides
      if (hasProject) {
        await saveProject.mutateAsync(projectOverrides)
      }
      // Reset edited state (will refetch and use new initial values)
      setEditedTerminology(null)
      setEditedKeys(null)
      setStatusAnnouncement(t('sections.translations.saved'))
    } catch {
      // Error handling is done by the mutation
    }
  }

  const handleClearGlobal = async () => {
    if (!window.confirm(t('sections.translations.confirmClearGlobal'))) {
      return
    }

    try {
      await saveGlobal.mutateAsync(createEmptyOverrides())
      // Remove global entries from edited state
      setEditedTerminology((prev) =>
        prev ? prev.filter((e) => e.scope !== 'global') : initialEntries.terminology.filter((e) => e.scope !== 'global')
      )
      setEditedKeys((prev) =>
        prev ? prev.filter((e) => e.scope !== 'global') : initialEntries.keys.filter((e) => e.scope !== 'global')
      )
      setStatusAnnouncement(t('sections.translations.clearedGlobal'))
    } catch {
      // Error handling is done by the mutation
    }
  }

  const handleClearProject = async () => {
    if (!window.confirm(t('sections.translations.confirmClearProject'))) {
      return
    }

    try {
      await saveProject.mutateAsync(createEmptyOverrides())
      // Remove project entries from edited state
      setEditedTerminology((prev) =>
        prev ? prev.filter((e) => e.scope !== 'project') : initialEntries.terminology.filter((e) => e.scope !== 'project')
      )
      setEditedKeys((prev) =>
        prev ? prev.filter((e) => e.scope !== 'project') : initialEntries.keys.filter((e) => e.scope !== 'project')
      )
      setStatusAnnouncement(t('sections.translations.clearedProject'))
    } catch {
      // Error handling is done by the mutation
    }
  }

  const isSaving = saveGlobal.isPending || saveProject.isPending
  const saveError = saveGlobal.isError || saveProject.isError

  if (isLoading) {
    return (
      <CollapseSection title={t('sections.translations.title')} defaultOpen={false}>
        <div className="flex items-center justify-center py-8">
          <Loader2 className="w-6 h-6 animate-spin text-primary" aria-hidden="true" />
        </div>
      </CollapseSection>
    )
  }

  if (error) {
    return (
      <CollapseSection title={t('sections.translations.title')} defaultOpen={false}>
        <div className="alert alert-error">
          <AlertCircle size={16} aria-hidden="true" />
          <span>{t('sections.translations.loadError')}</span>
        </div>
      </CollapseSection>
    )
  }

  return (
    <CollapseSection title={t('sections.translations.title')} defaultOpen={false}>
      <div className="space-y-6">
        {/* Description */}
        <p className="text-sm text-base-content/70">
          {t('sections.translations.description')}
        </p>

        {/* Terminology replacements */}
        <div className="space-y-2">
          <h4 className="font-medium flex items-center gap-2">
            <Languages size={16} className="text-base-content/60" aria-hidden="true" />
            {t('sections.translations.terminology')}
          </h4>
          <p className="text-xs text-base-content/60">
            {t('sections.translations.terminologyHint')}
          </p>
          <TerminologyEditor
            entries={terminologyEntries}
            onChange={handleTerminologyChange}
            projectName={projectName}
            hasProject={hasProject}
          />
        </div>

        {/* Key overrides */}
        <div className="space-y-2">
          <h4 className="font-medium flex items-center gap-2">
            <Languages size={16} className="text-base-content/60" aria-hidden="true" />
            {t('sections.translations.keyOverrides')}
          </h4>
          <p className="text-xs text-base-content/60">
            {t('sections.translations.keyOverridesHint')}
          </p>
          <KeyOverrideEditor
            entries={keyEntries}
            onChange={handleKeyChange}
            availableKeys={availableKeys}
            projectName={projectName}
            hasProject={hasProject}
            currentLanguage={currentLanguage}
          />
        </div>

        {/* Action buttons */}
        <div className="flex items-center justify-between pt-4 border-t border-base-300/50">
          <div className="text-sm text-base-content/60">
            {hasChanges && (
              <span className="text-warning">
                {t('sections.translations.unsavedChanges')}
              </span>
            )}
            {saveError && (
              <span className="text-error flex items-center gap-1">
                <AlertCircle size={14} aria-hidden="true" />
                {t('sections.translations.saveError')}
              </span>
            )}
          </div>
          <div className="flex items-center gap-2">
            <button
              type="button"
              className="btn btn-ghost btn-sm text-warning"
              onClick={handleClearGlobal}
              disabled={isSaving}
              aria-label={t('sections.translations.clearGlobal')}
            >
              <RotateCcw size={14} aria-hidden="true" />
              {t('sections.translations.clearGlobal')}
            </button>
            {hasProject && (
              <button
                type="button"
                className="btn btn-ghost btn-sm text-warning"
                onClick={handleClearProject}
                disabled={isSaving}
                aria-label={t('sections.translations.clearProject')}
              >
                <RotateCcw size={14} aria-hidden="true" />
                {t('sections.translations.clearProject')}
              </button>
            )}
            <button
              type="button"
              className="btn btn-primary btn-sm"
              onClick={handleSave}
              disabled={!hasChanges || isSaving}
            >
              {isSaving ? (
                <Loader2 size={14} className="animate-spin" aria-hidden="true" />
              ) : (
                <Save size={14} aria-hidden="true" />
              )}
              {t('sections.translations.saveOverrides')}
            </button>
          </div>
        </div>

        {/* Screen reader announcement for save/reset status */}
        <div aria-live="polite" className="sr-only">
          {statusAnnouncement}
        </div>

        {/* Info about page reload */}
        <div className="alert text-xs">
          <AlertCircle size={14} aria-hidden="true" />
          <span>{t('sections.translations.reloadNotice')}</span>
        </div>
      </div>
    </CollapseSection>
  )
}
