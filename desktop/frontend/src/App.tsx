import { useState, useEffect } from 'react'
import { AuthPage } from '@/pages/AuthPage'
import { SecretsPage } from '@/pages/SecretsPage'
import { AuditPage } from '@/pages/AuditPage'
import { GetAuthStatus } from '../wailsjs/go/main/App'

type Page = 'secrets' | 'audit'

function App() {
  const [isAuthenticated, setIsAuthenticated] = useState<boolean | null>(null)
  const [currentPage, setCurrentPage] = useState<Page>('secrets')

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

  if (isAuthenticated === null) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <div className="animate-pulse">Loading...</div>
      </div>
    )
  }

  if (!isAuthenticated) {
    return <AuthPage onAuthenticated={() => setIsAuthenticated(true)} />
  }

  if (currentPage === 'audit') {
    return <AuditPage onNavigateBack={() => setCurrentPage('secrets')} />
  }

  return (
    <SecretsPage
      onLocked={() => setIsAuthenticated(false)}
      onNavigateToAudit={() => setCurrentPage('audit')}
    />
  )
}

export default App
