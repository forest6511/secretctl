import { useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { X } from 'lucide-react'

interface KeyboardShortcutsHelpProps {
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function KeyboardShortcutsHelp({ open, onOpenChange }: KeyboardShortcutsHelpProps) {
  const { t } = useTranslation()

  const shortcuts = [
    {
      category: t('shortcuts.general', 'General'),
      items: [
        { keys: '⌘K', description: t('shortcuts.commandPalette', 'Command Palette') },
        { keys: '⌘N', description: t('shortcuts.newSecret', 'New Secret') },
        { keys: '⌘F', description: t('shortcuts.searchSecrets', 'Search Secrets') },
        { keys: '⌘,', description: t('shortcuts.settings', 'Settings') },
        { keys: '⌘L', description: t('shortcuts.lockVault', 'Lock Vault') },
        { keys: '⌘/', description: t('shortcuts.showHelp', 'Show Shortcuts') },
      ]
    }
  ]

  // Close on Escape
  useEffect(() => {
    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && open) {
        onOpenChange(false)
      }
    }
    window.addEventListener('keydown', handleEscape)
    return () => window.removeEventListener('keydown', handleEscape)
  }, [open, onOpenChange])

  if (!open) return null

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      {/* Backdrop */}
      <div
        className="absolute inset-0 bg-black/50"
        onClick={() => onOpenChange(false)}
      />
      {/* Dialog */}
      <div className="relative w-full max-w-sm bg-card rounded-lg shadow-lg border border-border overflow-hidden">
        <div className="flex items-center justify-between px-4 py-3 border-b border-border">
          <h2 className="text-lg font-semibold text-card-foreground">
            {t('shortcuts.title', 'Keyboard Shortcuts')}
          </h2>
          <button
            onClick={() => onOpenChange(false)}
            className="text-muted-foreground hover:text-foreground"
          >
            <X className="h-4 w-4" />
          </button>
        </div>
        <div className="p-4 space-y-6">
          {shortcuts.map((section) => (
            <div key={section.category} className="space-y-2">
              <h3 className="text-sm font-medium text-muted-foreground">
                {section.category}
              </h3>
              <div className="space-y-1">
                {section.items.map((item) => (
                  <div
                    key={item.keys}
                    className="flex items-center justify-between py-1"
                  >
                    <span className="text-sm">{item.description}</span>
                    <kbd className="text-xs bg-muted text-muted-foreground px-2 py-1 rounded font-mono">
                      {item.keys}
                    </kbd>
                  </div>
                ))}
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}
