import { useTranslation } from 'react-i18next'
import { Globe } from 'lucide-react'
import { CollapseSection, Select } from '../FormField'
import { SUPPORTED_LANGUAGES, STORAGE_KEY } from '@/i18n'

/**
 * Language labels for each supported language.
 * Add new languages here when adding translations.
 */
const languageLabels: Record<string, string> = {
  en: 'English',
  lv: 'Latviešu',
}

interface AppearanceSettingsProps {
  /** Whether to show in simple mode (just the essentials) */
  simpleMode?: boolean
}

/**
 * Appearance settings section including language selection.
 * Shown in the Work tab since language is a commonly used setting.
 */
export function AppearanceSettings({ simpleMode }: AppearanceSettingsProps) {
  const { i18n, t } = useTranslation('settings')

  const languageOptions = SUPPORTED_LANGUAGES.map((lang) => ({
    value: lang,
    label: languageLabels[lang] ?? lang,
  }))

  const handleLanguageChange = (value: string) => {
    // Save to localStorage and change language
    localStorage.setItem(STORAGE_KEY, value)
    i18n.changeLanguage(value)
  }

  // In simple mode, skip if only one language available
  if (simpleMode && SUPPORTED_LANGUAGES.length <= 1) {
    return null
  }

  return (
    <CollapseSection title={t('sections.appearance.title')} defaultOpen={false}>
      <div className="space-y-4">
        {/* Language selector */}
        <div className="flex items-start gap-3">
          <div className="mt-1">
            <Globe size={18} className="text-base-content/60" aria-hidden="true" />
          </div>
          <div className="flex-1">
            <Select
              label={t('sections.appearance.language')}
              hint={t('sections.appearance.languageHint')}
              value={i18n.language}
              onChange={handleLanguageChange}
              options={languageOptions}
            />
          </div>
        </div>

        {/* Info about adding languages */}
        {SUPPORTED_LANGUAGES.length <= 1 && !simpleMode && (
          <div className="alert alert-info text-sm">
            <Globe size={16} aria-hidden="true" />
            <span>{t('sections.appearance.moreLanguagesComing')}</span>
          </div>
        )}
      </div>
    </CollapseSection>
  )
}
