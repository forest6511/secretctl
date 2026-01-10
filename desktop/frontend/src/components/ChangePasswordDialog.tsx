import { useState, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { Key, AlertTriangle, CheckCircle } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { ChangePassword } from '../../wailsjs/go/main/App'

interface ChangePasswordDialogProps {
  open: boolean
  onClose: () => void
  onSuccess?: () => void
}

interface PasswordChangeResult {
  success: boolean
  message: string
  strength?: string
  warnings?: string[]
}

export function ChangePasswordDialog({
  open,
  onClose,
  onSuccess,
}: ChangePasswordDialogProps) {
  const { t } = useTranslation()

  const [currentPassword, setCurrentPassword] = useState('')
  const [newPassword, setNewPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const [success, setSuccess] = useState(false)

  // Reset state when dialog opens
  useEffect(() => {
    if (open) {
      setCurrentPassword('')
      setNewPassword('')
      setConfirmPassword('')
      setError('')
      setLoading(false)
      setSuccess(false)
    }
  }, [open])

  // Handle escape key
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (!open) return
      if (e.key === 'Escape') {
        e.preventDefault()
        if (!loading) {
          onClose()
        }
      }
    }

    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [open, loading, onClose])

  const handleSubmit = async () => {
    // Clear previous error
    setError('')

    // Validate inputs
    if (!currentPassword) {
      setError(t('changePassword.currentRequired'))
      return
    }
    if (!newPassword) {
      setError(t('changePassword.newRequired'))
      return
    }
    if (!confirmPassword) {
      setError(t('changePassword.confirmRequired'))
      return
    }
    if (newPassword !== confirmPassword) {
      setError(t('changePassword.newDoNotMatch'))
      return
    }
    if (currentPassword === newPassword) {
      setError(t('changePassword.mustBeDifferent'))
      return
    }

    setLoading(true)

    try {
      const result: PasswordChangeResult = await ChangePassword(
        currentPassword,
        newPassword,
        confirmPassword
      )

      if (result.success) {
        setSuccess(true)
        // Wait a moment to show success message
        setTimeout(() => {
          onSuccess?.()
          onClose()
        }, 1500)
      } else {
        setError(result.message)
      }
    } catch (err) {
      setError(`Failed to change password: ${err}`)
    } finally {
      setLoading(false)
    }
  }

  if (!open) return null

  return (
    <div
      className="fixed inset-0 bg-black/50 flex items-center justify-center z-50"
      onClick={() => !loading && onClose()}
      data-testid="change-password-dialog"
    >
      <Card
        className="w-full max-w-md mx-4"
        onClick={e => e.stopPropagation()}
      >
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Key className="w-5 h-5" />
            {t('changePassword.title')}
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          {success ? (
            <div className="flex flex-col items-center py-6 space-y-4">
              <CheckCircle className="w-12 h-12 text-green-500" />
              <p className="text-lg font-medium">{t('changePassword.successTitle')}</p>
              <p className="text-sm text-muted-foreground text-center">
                {t('changePassword.successMessage')}
              </p>
            </div>
          ) : (
            <>
              {error && (
                <div className="flex items-center gap-2 p-3 bg-destructive/10 border border-destructive/20 rounded-md">
                  <AlertTriangle className="w-4 h-4 text-destructive" />
                  <p className="text-sm text-destructive" data-testid="change-password-error">
                    {error}
                  </p>
                </div>
              )}

              <div className="space-y-2">
                <label className="text-sm font-medium">
                  {t('changePassword.currentPassword')} <span className="text-destructive">*</span>
                </label>
                <Input
                  type="password"
                  value={currentPassword}
                  onChange={(e) => setCurrentPassword(e.target.value)}
                  placeholder={t('changePassword.currentPasswordPlaceholder')}
                  disabled={loading}
                  data-testid="current-password-input"
                  autoFocus
                />
              </div>

              <div className="space-y-2">
                <label className="text-sm font-medium">
                  {t('changePassword.newPassword')} <span className="text-destructive">*</span>
                </label>
                <Input
                  type="password"
                  value={newPassword}
                  onChange={(e) => setNewPassword(e.target.value)}
                  placeholder={t('changePassword.newPasswordPlaceholder')}
                  disabled={loading}
                  data-testid="new-password-input"
                />
                <p className="text-xs text-muted-foreground">
                  {t('changePassword.newPasswordHint')}
                </p>
              </div>

              <div className="space-y-2">
                <label className="text-sm font-medium">
                  {t('changePassword.confirmNewPassword')} <span className="text-destructive">*</span>
                </label>
                <Input
                  type="password"
                  value={confirmPassword}
                  onChange={(e) => setConfirmPassword(e.target.value)}
                  placeholder={t('changePassword.confirmNewPasswordPlaceholder')}
                  disabled={loading}
                  data-testid="confirm-password-input"
                />
              </div>

              <div className="flex justify-end gap-2 pt-4">
                <Button
                  variant="outline"
                  onClick={onClose}
                  disabled={loading}
                  data-testid="change-password-cancel"
                >
                  {t('common.cancel')}
                </Button>
                <Button
                  onClick={handleSubmit}
                  disabled={loading}
                  data-testid="change-password-confirm"
                >
                  {loading ? t('changePassword.changing') : t('changePassword.changePassword')}
                </Button>
              </div>
            </>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
