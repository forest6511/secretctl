import { useState } from 'react'
import { Copy, Eye, EyeOff, Lock, Unlock } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { ViewSensitiveField, CopyFieldValue } from '../../wailsjs/go/main/App'
import { useToast } from '@/hooks/useToast'

interface FieldDTO {
  value: string
  sensitive: boolean
  aliases?: string[]
  kind?: string
  hint?: string
}

interface FieldEditorProps {
  secretKey: string
  fieldName: string
  field: FieldDTO
  readOnly?: boolean
}

export function FieldEditor({ secretKey, fieldName, field, readOnly = true }: FieldEditorProps) {
  const [isVisible, setIsVisible] = useState(false)
  const toast = useToast()

  const handleToggleVisibility = async () => {
    if (field.sensitive && !isVisible) {
      // Log view action before showing
      try {
        await ViewSensitiveField(secretKey, fieldName)
      } catch (err) {
        console.error('Failed to log field view:', err)
      }
    }
    setIsVisible(!isVisible)
  }

  const handleCopy = async () => {
    try {
      // Security: Value is fetched server-side to prevent caller manipulation
      await CopyFieldValue(secretKey, fieldName)
      toast.success('Copied! Auto-clears in 30s')
    } catch (err) {
      console.error('Failed to copy:', err)
      toast.error('Failed to copy to clipboard')
    }
  }

  const displayValue = field.sensitive && !isVisible
    ? '••••••••'
    : field.value

  return (
    <div className="space-y-1">
      <div className="flex items-center gap-2">
        <label className="text-sm font-medium text-muted-foreground flex items-center gap-1">
          {field.sensitive ? (
            <Lock className="w-3 h-3" />
          ) : (
            <Unlock className="w-3 h-3" />
          )}
          {fieldName}
        </label>
        {field.hint && (
          <span className="text-xs text-muted-foreground">({field.hint})</span>
        )}
      </div>
      <div className="flex items-center gap-2">
        <Input
          type={field.sensitive && !isVisible ? 'password' : 'text'}
          value={displayValue}
          readOnly={readOnly}
          className="font-mono"
          data-testid={`field-value-${fieldName}`}
        />
        {field.sensitive && (
          <Button
            variant="ghost"
            size="icon"
            onClick={handleToggleVisibility}
            title={isVisible ? 'Hide' : 'Show'}
            data-testid={`toggle-field-${fieldName}`}
          >
            {isVisible ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
          </Button>
        )}
        <Button
          variant="ghost"
          size="icon"
          onClick={handleCopy}
          title="Copy"
          data-testid={`copy-field-${fieldName}`}
        >
          <Copy className="w-4 h-4" />
        </Button>
      </div>
    </div>
  )
}
