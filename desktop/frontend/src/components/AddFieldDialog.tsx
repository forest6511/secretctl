import { useState, useEffect } from 'react'
import { Plus, Type, AlignLeft } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { InputType } from '@/components/FieldEditor'

interface AddFieldDialogProps {
  open: boolean
  existingFieldNames: string[]
  onAdd: (name: string, value: string, sensitive: boolean, inputType: InputType) => void
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
  const [inputType, setInputType] = useState<InputType>('text')
  const [error, setError] = useState('')

  // Reset state when dialog opens
  useEffect(() => {
    if (open) {
      setName('')
      setValue('')
      setSensitive(false)
      setInputType('text')
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
    onAdd(name, value, sensitive, inputType)
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

          {/* Input Type Selection */}
          <div className="space-y-2">
            <label className="text-sm font-medium">Field Type</label>
            <div className="flex gap-2">
              <button
                type="button"
                onClick={() => setInputType('text')}
                className={`flex-1 flex items-center justify-center gap-2 px-3 py-2 rounded border-2 transition-all ${
                  inputType === 'text'
                    ? 'border-sky-500 bg-sky-50 text-sky-700'
                    : 'border-slate-200 hover:border-slate-300 text-slate-600'
                }`}
                data-testid="input-type-text"
              >
                <Type className="w-4 h-4" />
                <span className="text-sm font-medium">Single Line</span>
              </button>
              <button
                type="button"
                onClick={() => setInputType('textarea')}
                className={`flex-1 flex items-center justify-center gap-2 px-3 py-2 rounded border-2 transition-all ${
                  inputType === 'textarea'
                    ? 'border-sky-500 bg-sky-50 text-sky-700'
                    : 'border-slate-200 hover:border-slate-300 text-slate-600'
                }`}
                data-testid="input-type-textarea"
              >
                <AlignLeft className="w-4 h-4" />
                <span className="text-sm font-medium">Multi-line</span>
              </button>
            </div>
            <p className="text-xs text-muted-foreground">
              {inputType === 'text'
                ? 'For passwords, API keys, usernames, etc.'
                : 'For SSH keys, certificates, JSON, etc.'}
            </p>
          </div>

          <div className="space-y-2">
            <label className="text-sm font-medium">Value</label>
            {inputType === 'text' ? (
              <Input
                type={sensitive ? 'password' : 'text'}
                value={value}
                onChange={(e) => setValue(e.target.value)}
                placeholder="Enter field value"
                className="font-mono"
                data-testid="field-value-input"
              />
            ) : (
              <Textarea
                value={value}
                onChange={(e) => setValue(e.target.value)}
                placeholder="Enter multi-line value (SSH key, certificate, etc.)"
                className="font-mono min-h-[120px]"
                data-testid="field-value-input"
              />
            )}
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
