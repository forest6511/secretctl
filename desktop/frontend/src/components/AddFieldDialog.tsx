import { useState, useEffect } from 'react'
import { Plus } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'

interface AddFieldDialogProps {
  open: boolean
  existingFieldNames: string[]
  onAdd: (name: string, value: string, sensitive: boolean) => void
  onCancel: () => void
}

// Validation: snake_case, max 64 chars
const FIELD_NAME_REGEX = /^[a-z][a-z0-9_]*$/

export function AddFieldDialog({
  open,
  existingFieldNames,
  onAdd,
  onCancel,
}: AddFieldDialogProps) {
  const [name, setName] = useState('')
  const [value, setValue] = useState('')
  const [sensitive, setSensitive] = useState(false)
  const [error, setError] = useState('')

  // Reset state when dialog opens
  useEffect(() => {
    if (open) {
      setName('')
      setValue('')
      setSensitive(false)
      setError('')
    }
  }, [open])

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

  const validateName = (fieldName: string): string | null => {
    if (!fieldName) {
      return 'Field name is required'
    }
    if (fieldName.length > 64) {
      return 'Field name must be 64 characters or less'
    }
    if (!FIELD_NAME_REGEX.test(fieldName)) {
      return 'Field name must be lowercase letters, numbers, and underscores (snake_case)'
    }
    if (existingFieldNames.includes(fieldName)) {
      return 'A field with this name already exists'
    }
    return null
  }

  const handleSubmit = () => {
    const validationError = validateName(name)
    if (validationError) {
      setError(validationError)
      return
    }
    onAdd(name, value, sensitive)
  }

  const handleNameChange = (newName: string) => {
    setName(newName)
    // Clear error when user starts typing
    if (error) {
      setError('')
    }
  }

  if (!open) return null

  return (
    <div
      className="fixed inset-0 bg-black/50 flex items-center justify-center z-50"
      onClick={onCancel}
      data-testid="add-field-dialog"
    >
      <Card
        className="w-full max-w-md mx-4"
        onClick={e => e.stopPropagation()}
      >
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Plus className="w-5 h-5" />
            Add Field
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="space-y-2">
            <label className="text-sm font-medium">
              Field Name <span className="text-destructive">*</span>
            </label>
            <Input
              value={name}
              onChange={(e) => handleNameChange(e.target.value)}
              placeholder="e.g. username, api_key, password"
              data-testid="field-name-input"
              autoFocus
            />
            {error && (
              <p className="text-sm text-destructive" data-testid="field-name-error">
                {error}
              </p>
            )}
            <p className="text-xs text-muted-foreground">
              Use snake_case (lowercase letters, numbers, underscores)
            </p>
          </div>

          <div className="space-y-2">
            <label className="text-sm font-medium">Value</label>
            <Input
              type={sensitive ? 'password' : 'text'}
              value={value}
              onChange={(e) => setValue(e.target.value)}
              placeholder="Enter field value"
              className="font-mono"
              data-testid="field-value-input"
            />
          </div>

          <div className="flex items-center gap-2">
            <input
              type="checkbox"
              id="sensitive-checkbox"
              checked={sensitive}
              onChange={(e) => setSensitive(e.target.checked)}
              className="w-4 h-4"
              data-testid="field-sensitive-checkbox"
            />
            <label htmlFor="sensitive-checkbox" className="text-sm">
              Sensitive field (will be masked by default)
            </label>
          </div>

          <div className="flex justify-end gap-2">
            <Button
              variant="outline"
              onClick={onCancel}
              data-testid="add-field-cancel"
            >
              Cancel
            </Button>
            <Button
              onClick={handleSubmit}
              data-testid="add-field-confirm"
            >
              Add Field
            </Button>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
