import { useState, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { KeyRound, Lock, Eye, EyeOff, AlertCircle } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { CheckVaultExists, InitVault, Unlock } from '../../wailsjs/go/main/App'

interface AuthPageProps {
  onAuthenticated: () => void
}

export function AuthPage({ onAuthenticated }: AuthPageProps) {
  const { t } = useTranslation()
  const [vaultExists, setVaultExists] = useState<boolean | null>(null)
  const [password, setPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [showPassword, setShowPassword] = useState(false)
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  useEffect(() => {
    CheckVaultExists().then(setVaultExists)
  }, [])

  const handleUnlock = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!password) return

    setLoading(true)
    setError('')

    try {
      await Unlock(password)
      onAuthenticated()
    } catch (err) {
      setError(t('auth.invalidPassword'))
    } finally {
      setLoading(false)
    }
  }

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!password || !confirmPassword) return

    if (password !== confirmPassword) {
      setError(t('auth.passwordsDoNotMatch'))
      return
    }

    if (password.length < 8) {
      setError(t('auth.passwordTooShort'))
      return
    }

    setLoading(true)
    setError('')

    try {
      await InitVault(password)
      onAuthenticated()
    } catch (err) {
      setError(t('auth.failedToCreateVault'))
    } finally {
      setLoading(false)
    }
  }

  if (vaultExists === null) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <div className="animate-pulse">{t('common.loading')}</div>
      </div>
    )
  }

  return (
    <div className="flex items-center justify-center min-h-screen bg-muted/30 macos-titlebar-padding">
      <Card className="w-full max-w-md mx-4">
        <CardHeader className="space-y-1 text-center">
          <div className="flex justify-center mb-4">
            <div className="p-3 rounded-full bg-primary/10">
              <KeyRound className="w-8 h-8 text-primary" />
            </div>
          </div>
          <CardTitle className="text-2xl">
            {vaultExists ? t('auth.unlockVault') : t('auth.createVault')}
          </CardTitle>
          <CardDescription>
            {vaultExists
              ? t('auth.unlockDescription')
              : t('auth.createDescription')}
          </CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={vaultExists ? handleUnlock : handleCreate} className="space-y-4">
            <div className="space-y-2">
              <div className="relative">
                <Input
                  type={showPassword ? 'text' : 'password'}
                  placeholder={t('auth.masterPassword')}
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  className="pr-10"
                  autoFocus
                  data-testid="master-password"
                />
                <button
                  type="button"
                  onClick={() => setShowPassword(!showPassword)}
                  className="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
                >
                  {showPassword ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
                </button>
              </div>
            </div>

            {!vaultExists && (
              <div className="space-y-2">
                <Input
                  type={showPassword ? 'text' : 'password'}
                  placeholder={t('auth.confirmPassword')}
                  value={confirmPassword}
                  onChange={(e) => setConfirmPassword(e.target.value)}
                  data-testid="confirm-password"
                />
              </div>
            )}

            {error && (
              <div className="flex items-center gap-2 text-sm text-destructive">
                <AlertCircle className="w-4 h-4" />
                {error}
              </div>
            )}

            <Button type="submit" className="w-full" disabled={loading} data-testid="unlock-button">
              {loading ? (
                <span className="animate-pulse">{t('common.processing')}</span>
              ) : (
                <>
                  <Lock className="w-4 h-4 mr-2" />
                  {vaultExists ? t('auth.unlock') : t('auth.createVault')}
                </>
              )}
            </Button>
          </form>
        </CardContent>
      </Card>
    </div>
  )
}
