import { useState, useEffect } from 'react'
import { AuthPage } from '@/pages/AuthPage'
import { SecretsPage } from '@/pages/SecretsPage'
import { GetAuthStatus } from '../wailsjs/go/main/App'

function App() {
  const [isAuthenticated, setIsAuthenticated] = useState<boolean | null>(null)

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

  return <SecretsPage onLocked={() => setIsAuthenticated(false)} />
}

export default App
