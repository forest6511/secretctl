import { useState, useEffect } from 'react'
import { Link } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'

interface AddBindingDialogProps {
  open: boolean
  existingEnvVars: string[]
  fieldNames: string[]
  onAdd: (envVar: string, fieldName: string) => void
  onCancel: () => void
}

// Validation: SCREAMING_SNAKE_CASE
const ENV_VAR_REGEX = /^[A-Z][A-Z0-9_]*$/

export function AddBindingDialog({
  open,
  existingEnvVars,
  fieldNames,
  onAdd,
  onCancel,
}: AddBindingDialogProps) {
  const [envVar, setEnvVar] = useState('')
  const [fieldName, setFieldName] = useState('')
  const [envVarError, setEnvVarError] = useState('')
  const [fieldError, setFieldError] = useState('')

  // Reset state when dialog opens
  useEffect(() => {
    if (open) {
      setEnvVar('')
      setFieldName(fieldNames[0] ?? '')
      setEnvVarError('')
      setFieldError('')
    }
  }, [open, fieldNames])

  // Handle escape key
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (!open) return
      if (e.key === 'Escape') {
        e.preventDefault()
        onCancel()
      }
    }

    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [open, onCancel])

  const validateEnvVar = (value: string): string | null => {
    if (!value) {
      return 'Environment variable name is required'
    }
    if (!ENV_VAR_REGEX.test(value)) {
      return 'Must be SCREAMING_SNAKE_CASE (uppercase letters, numbers, underscores)'
    }
    if (existingEnvVars.includes(value)) {
      return 'This environment variable is already bound'
    }
    return null
  }

  const validateField = (value: string): string | null => {
    if (!value) {
      return 'Field selection is required'
    }
    if (!fieldNames.includes(value)) {
      return 'Selected field does not exist'
    }
    return null
  }

  const handleSubmit = () => {
    const envVarValidation = validateEnvVar(envVar)
    const fieldValidation = validateField(fieldName)

    setEnvVarError(envVarValidation || '')
    setFieldError(fieldValidation || '')

    if (envVarValidation || fieldValidation) {
      return
    }

    onAdd(envVar, fieldName)
  }

  const handleEnvVarChange = (value: string) => {
    setEnvVar(value.toUpperCase())
    if (envVarError) {
      setEnvVarError('')
    }
  }

  if (!open) return null

  return (
    <div
      className="fixed inset-0 bg-black/50 flex items-center justify-center z-50"
      onClick={onCancel}
      data-testid="add-binding-dialog"
    >
      <Card
        className="w-full max-w-md mx-4"
        onClick={e => e.stopPropagation()}
      >
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Link className="w-5 h-5" />
            Add Environment Binding
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="space-y-2">
            <label className="text-sm font-medium">
              Environment Variable <span className="text-destructive">*</span>
            </label>
            <Input
              value={envVar}
              onChange={(e) => handleEnvVarChange(e.target.value)}
              placeholder="e.g. DATABASE_PASSWORD, API_KEY"
              className="font-mono"
              data-testid="binding-envvar-input"
              autoFocus
            />
            {envVarError && (
              <p className="text-sm text-destructive" data-testid="binding-envvar-error">
                {envVarError}
              </p>
            )}
            <p className="text-xs text-muted-foreground">
              Use SCREAMING_SNAKE_CASE (uppercase letters, numbers, underscores)
            </p>
          </div>

          <div className="space-y-2">
            <label className="text-sm font-medium">
              Field <span className="text-destructive">*</span>
            </label>
            {fieldNames.length === 0 ? (
              <p className="text-sm text-muted-foreground italic">
                No fields available. Add a field first.
              </p>
            ) : (
              <select
                value={fieldName}
                onChange={(e) => setFieldName(e.target.value)}
                className="w-full p-2 border rounded-md bg-background font-mono text-sm"
                data-testid="binding-field-select"
              >
                {fieldNames.map((name) => (
                  <option key={name} value={name}>
                    {name}
                  </option>
                ))}
              </select>
            )}
            {fieldError && (
              <p className="text-sm text-destructive" data-testid="binding-field-error">
                {fieldError}
              </p>
            )}
          </div>

          <div className="flex justify-end gap-2">
            <Button
              variant="outline"
              onClick={onCancel}
              data-testid="add-binding-cancel"
            >
              Cancel
            </Button>
            <Button
              onClick={handleSubmit}
              disabled={fieldNames.length === 0}
              data-testid="add-binding-confirm"
            >
              Add Binding
            </Button>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
