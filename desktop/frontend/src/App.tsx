import { useState, useEffect } from 'react'
import { AuthPage } from '@/pages/AuthPage'
import { SecretsPage } from '@/pages/SecretsPage'
import { AuditPage } from '@/pages/AuditPage'
import { SettingsPage } from '@/pages/SettingsPage'
import { CommandPalette } from '@/components/CommandPalette'
import { KeyboardShortcutsHelp } from '@/components/KeyboardShortcutsHelp'
import { ToastProvider } from '@/hooks/useToast'
import { useKeyboardShortcuts } from '@/hooks/useKeyboardShortcuts'
import { GetAuthStatus, Lock } from '../wailsjs/go/main/App'

type Page = 'secrets' | 'audit' | 'settings'

function App() {
  const [isAuthenticated, setIsAuthenticated] = useState<boolean | null>(null)
  const [currentPage, setCurrentPage] = useState<Page>('secrets')
  const [commandPaletteOpen, setCommandPaletteOpen] = useState(false)
  const [shortcutsHelpOpen, setShortcutsHelpOpen] = useState(false)

  useEffect(() => {
    checkAuth()
  }, [])

  const checkAuth = async () => {
    try {
      const status = await GetAuthStatus()
      setIsAuthenticated(status.unlocked)
    } catch (err) {
      setIsAuthenticated(false)
    }
  }

  const handleLock = async () => {
    try {
      await Lock()
      setIsAuthenticated(false)
    } catch (err) {
      console.error('Failed to lock vault:', err)
    }
  }

  const handleNavigate = (page: 'secrets' | 'settings' | 'new-secret') => {
    // 'new-secret' maps to 'secrets' page (user creates via UI)
    if (page === 'new-secret') {
      setCurrentPage('secrets')
    } else {
      setCurrentPage(page)
    }
  }

  const handleFocusSearch = () => {
    // Focus search is handled by SecretsPage directly via ⌘F
    // This is a placeholder for CommandPalette integration
    setCurrentPage('secrets')
  }

  // Keyboard shortcuts - only active when authenticated
  useKeyboardShortcuts(
    isAuthenticated
      ? [
          {
            key: 'k',
            meta: true,
            handler: () => setCommandPaletteOpen(true),
            description: 'Command Palette'
          },
          {
            key: 'n',
            meta: true,
            handler: () => setCurrentPage('secrets'),
            description: 'New Secret'
          },
          {
            key: 'f',
            meta: true,
            handler: handleFocusSearch,
            description: 'Search Secrets'
          },
          {
            key: ',',
            meta: true,
            handler: () => setCurrentPage('settings'),
            description: 'Settings'
          },
          {
            key: 'l',
            meta: true,
            handler: handleLock,
            description: 'Lock Vault'
          },
          {
            key: '/',
            meta: true,
            handler: () => setShortcutsHelpOpen(true),
            description: 'Show Shortcuts'
          }
        ]
      : []
  )

  if (isAuthenticated === null) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <div className="animate-pulse">Loading...</div>
      </div>
    )
  }

  if (!isAuthenticated) {
    return (
      <ToastProvider>
        <AuthPage onAuthenticated={() => setIsAuthenticated(true)} />
      </ToastProvider>
    )
  }

  if (currentPage === 'audit') {
    return (
      <ToastProvider>
        <AuditPage onNavigateBack={() => setCurrentPage('secrets')} />
      </ToastProvider>
    )
  }

  if (currentPage === 'settings') {
    return (
      <ToastProvider>
        <SettingsPage onNavigateBack={() => setCurrentPage('secrets')} />
      </ToastProvider>
    )
  }

  return (
    <ToastProvider>
      <SecretsPage
        onLocked={() => setIsAuthenticated(false)}
        onNavigateToAudit={() => setCurrentPage('audit')}
        onNavigateToSettings={() => setCurrentPage('settings')}
      />
      <CommandPalette
        open={commandPaletteOpen}
        onOpenChange={setCommandPaletteOpen}
        onNavigate={handleNavigate}
        onLock={handleLock}
        onFocusSearch={handleFocusSearch}
        onShowHelp={() => setShortcutsHelpOpen(true)}
      />
      <KeyboardShortcutsHelp
        open={shortcutsHelpOpen}
        onOpenChange={setShortcutsHelpOpen}
      />
    </ToastProvider>
  )
}

export default App
