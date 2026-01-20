import { useState, useEffect, useRef } from 'react'
import { useTranslation } from 'react-i18next'
import { Input } from '@/components/ui/input'
import { X, Plus, Settings, Lock, Search, HelpCircle } from 'lucide-react'

interface Command {
  id: string
  name: string
  shortcut?: string
  action: () => void
  icon: React.ReactNode
}

interface CommandPaletteProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  onNavigate: (page: 'secrets' | 'settings' | 'new-secret') => void
  onLock: () => void
  onFocusSearch: () => void
  onShowHelp: () => void
}

export function CommandPalette({
  open,
  onOpenChange,
  onNavigate,
  onLock,
  onFocusSearch,
  onShowHelp
}: CommandPaletteProps) {
  const { t } = useTranslation()
  const [search, setSearch] = useState('')
  const inputRef = useRef<HTMLInputElement>(null)

  const commands: Command[] = [
    {
      id: 'new',
      name: t('commandPalette.newSecret', 'New Secret'),
      shortcut: '⌘N',
      icon: <Plus className="h-4 w-4" />,
      action: () => onNavigate('new-secret')
    },
    {
      id: 'search',
      name: t('commandPalette.searchSecrets', 'Search Secrets'),
      shortcut: '⌘F',
      icon: <Search className="h-4 w-4" />,
      action: () => onFocusSearch()
    },
    {
      id: 'settings',
      name: t('commandPalette.settings', 'Settings'),
      shortcut: '⌘,',
      icon: <Settings className="h-4 w-4" />,
      action: () => onNavigate('settings')
    },
    {
      id: 'lock',
      name: t('commandPalette.lockVault', 'Lock Vault'),
      shortcut: '⌘L',
      icon: <Lock className="h-4 w-4" />,
      action: () => onLock()
    },
    {
      id: 'help',
      name: t('commandPalette.keyboardShortcuts', 'Keyboard Shortcuts'),
      shortcut: '⌘/',
      icon: <HelpCircle className="h-4 w-4" />,
      action: () => onShowHelp()
    },
  ]

  const filteredCommands = commands.filter(cmd =>
    cmd.name.toLowerCase().includes(search.toLowerCase())
  )

  // Reset search and focus input when dialog opens
  useEffect(() => {
    if (open) {
      setSearch('')
      setTimeout(() => inputRef.current?.focus(), 0)
    }
  }, [open])

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

  const handleSelect = (cmd: Command) => {
    onOpenChange(false)
    cmd.action()
  }

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && filteredCommands.length > 0) {
      const firstCommand = filteredCommands[0]
      if (firstCommand) {
        handleSelect(firstCommand)
      }
    }
  }

  if (!open) return null

  return (
    <div className="fixed inset-0 z-50 flex items-start justify-center pt-20">
      {/* Backdrop */}
      <div
        className="absolute inset-0 bg-black/50"
        onClick={() => onOpenChange(false)}
      />
      {/* Dialog */}
      <div className="relative w-full max-w-md bg-card rounded-lg shadow-lg border border-border overflow-hidden">
        <div className="flex items-center justify-between px-4 py-3 border-b border-border">
          <h2 className="text-lg font-semibold text-card-foreground">
            {t('commandPalette.title', 'Command Palette')}
          </h2>
          <button
            onClick={() => onOpenChange(false)}
            className="text-muted-foreground hover:text-foreground"
          >
            <X className="h-4 w-4" />
          </button>
        </div>
        <div className="p-4 space-y-4">
          <Input
            ref={inputRef}
            placeholder={t('commandPalette.placeholder', 'Type a command...')}
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            onKeyDown={handleKeyDown}
            className="bg-background"
          />
          <div className="space-y-1 max-h-64 overflow-y-auto">
            {filteredCommands.map((cmd) => (
              <button
                key={cmd.id}
                onClick={() => handleSelect(cmd)}
                className="w-full flex items-center gap-3 p-2 rounded-md
                           text-foreground hover:bg-muted transition-colors"
              >
                <span className="text-muted-foreground">{cmd.icon}</span>
                <span className="flex-1 text-left">{cmd.name}</span>
                {cmd.shortcut && (
                  <kbd className="text-xs bg-muted text-muted-foreground px-2 py-1 rounded">
                    {cmd.shortcut}
                  </kbd>
                )}
              </button>
            ))}
            {filteredCommands.length === 0 && (
              <p className="text-muted-foreground text-center py-4">
                {t('commandPalette.noCommands', 'No commands found')}
              </p>
            )}
          </div>
        </div>
      </div>
    </div>
  )
}
