import { useTranslation } from 'react-i18next'
import { ArrowLeft } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { ThemeToggle } from '@/components/ThemeToggle'

interface SettingsPageProps {
  onNavigateBack: () => void
}

export function SettingsPage({ onNavigateBack }: SettingsPageProps) {
  const { t, i18n } = useTranslation()

  const languages = [
    { code: 'en', name: 'English' },
    { code: 'ja', name: '日本語' },
  ]

  const handleLanguageChange = (e: React.ChangeEvent<HTMLSelectElement>) => {
    i18n.changeLanguage(e.target.value)
    localStorage.setItem('secretctl-language', e.target.value)
  }

  return (
    <div className="min-h-screen bg-background">
      {/* Header */}
      <header className="bg-header text-header-foreground px-4 py-3 flex items-center gap-4 macos-titlebar-padding">
        <Button
          variant="ghost"
          size="icon"
          onClick={onNavigateBack}
          className="text-header-foreground hover:bg-header/80"
        >
          <ArrowLeft className="h-5 w-5" />
        </Button>
        <h1 className="text-lg font-semibold">{t('settings.title', 'Settings')}</h1>
      </header>

      {/* Content */}
      <main className="p-6 max-w-2xl mx-auto space-y-8">
        {/* Appearance Section */}
        <section className="space-y-4">
          <h2 className="text-lg font-semibold text-foreground">
            {t('settings.appearance', 'Appearance')}
          </h2>
          <div className="bg-card rounded-lg border border-border p-4">
            <div className="flex items-center justify-between">
              <div>
                <h3 className="font-medium text-card-foreground">
                  {t('settings.theme', 'Theme')}
                </h3>
                <p className="text-sm text-muted-foreground">
                  {t('settings.themeDescription', 'Choose your preferred color scheme')}
                </p>
              </div>
              <ThemeToggle />
            </div>
          </div>
        </section>

        {/* Language Section */}
        <section className="space-y-4">
          <h2 className="text-lg font-semibold text-foreground">
            {t('settings.language', 'Language')}
          </h2>
          <div className="bg-card rounded-lg border border-border p-4">
            <div className="flex items-center justify-between">
              <div>
                <h3 className="font-medium text-card-foreground">
                  {t('settings.displayLanguage', 'Display Language')}
                </h3>
                <p className="text-sm text-muted-foreground">
                  {t('settings.languageDescription', 'Select your preferred language')}
                </p>
              </div>
              <select
                value={i18n.language}
                onChange={handleLanguageChange}
                className="bg-background border border-border rounded-md px-3 py-2 text-foreground focus:outline-none focus:ring-2 focus:ring-ring"
              >
                {languages.map((lang) => (
                  <option key={lang.code} value={lang.code}>
                    {lang.name}
                  </option>
                ))}
              </select>
            </div>
          </div>
        </section>
      </main>
    </div>
  )
}
