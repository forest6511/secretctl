import { useState } from 'react'
import { Copy, Eye, EyeOff, Lock, Unlock, Trash2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { ViewSensitiveField, CopyFieldValue } from '../../wailsjs/go/main/App'
import { useToast } from '@/hooks/useToast'

// InputType for UI rendering per ADR-005
export type InputType = 'text' | 'textarea'

export interface FieldDTO {
  value: string
  sensitive: boolean
  aliases?: string[]
  kind?: string
  inputType?: InputType // "text" (default) | "textarea" per ADR-005
  hint?: string
}

interface FieldEditorProps {
  secretKey: string
  fieldName: string
  field: FieldDTO
  readOnly?: boolean
  onChange?: (value: string) => void
  onSensitiveToggle?: () => void
  onDelete?: () => void
}

export function FieldEditor({
  secretKey,
  fieldName,
  field,
  readOnly = true,
  onChange,
  onSensitiveToggle,
  onDelete
}: FieldEditorProps) {
  const [isVisible, setIsVisible] = useState(false)
  const toast = useToast()

  const handleToggleVisibility = async () => {
    if (field.sensitive && !isVisible) {
      // Log view action before showing (only in read mode for existing secrets)
      if (readOnly && secretKey) {
        try {
          await ViewSensitiveField(secretKey, fieldName)
        } catch (err) {
          console.error('Failed to log field view:', err)
        }
      }
    }
    setIsVisible(!isVisible)
  }

  const handleCopy = async () => {
    if (!secretKey) {
      // For new secrets, use clipboard API directly
      try {
        await navigator.clipboard.writeText(field.value)
        toast.success('Copied!')
      } catch (err) {
        console.error('Failed to copy:', err)
        toast.error('Failed to copy to clipboard')
      }
      return
    }

    try {
      // Security: Value is fetched server-side to prevent caller manipulation
      await CopyFieldValue(secretKey, fieldName)
      toast.success('Copied! Auto-clears in 30s')
    } catch (err) {
      console.error('Failed to copy:', err)
      toast.error('Failed to copy to clipboard')
    }
  }

  const handleChange = (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => {
    if (onChange) {
      onChange(e.target.value)
    }
  }

  // Determine if this field should use textarea per ADR-005
  const isTextarea = field.inputType === 'textarea'

  // Security: For sensitive fields, mask value until visible
  // - Textarea: Must show masked value when hidden (no type="password" equivalent)
  //   Only mask when field has content (allows input for new/empty fields)
  // - Input in read mode: Show masked value (type="password" only works for visual masking in edit mode)
  // - Input in edit mode: type="password" handles visual masking, actual value preserved for editing
  const shouldMaskTextarea = isTextarea && field.sensitive && !isVisible && (field.value?.length ?? 0) > 0
  const shouldMaskInput = !isTextarea && field.sensitive && !isVisible && readOnly
  const displayValue = (shouldMaskTextarea || shouldMaskInput)
    ? '••••••••'
    : field.value

  return (
    <div className="space-y-1" data-testid={`field-${fieldName}`}>
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
        {!readOnly && onSensitiveToggle && (
          <Button
            variant="ghost"
            size="sm"
            onClick={onSensitiveToggle}
            className="h-6 px-2 text-xs"
            title={field.sensitive ? 'Mark as non-sensitive' : 'Mark as sensitive'}
            data-testid={`toggle-sensitive-${fieldName}`}
          >
            {field.sensitive ? 'Unmark sensitive' : 'Mark sensitive'}
          </Button>
        )}
      </div>
      <div className={`flex ${isTextarea ? 'items-start' : 'items-center'} gap-2`}>
        {isTextarea ? (
          <Textarea
            value={displayValue}
            readOnly={readOnly || shouldMaskTextarea}
            onChange={handleChange}
            className={`font-mono min-h-[120px] whitespace-pre-wrap ${
              shouldMaskTextarea ? 'cursor-not-allowed bg-muted' : ''
            }`}
            title={shouldMaskTextarea
              ? (readOnly ? 'Click 👁 to view' : 'Click 👁 to edit')
              : undefined
            }
            data-testid={`field-value-${fieldName}`}
          />
        ) : (
          <Input
            type={field.sensitive && !isVisible ? 'password' : 'text'}
            value={displayValue}
            readOnly={readOnly}
            onChange={handleChange}
            className="font-mono"
            data-testid={`field-value-${fieldName}`}
          />
        )}
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
        {!readOnly && onDelete && (
          <Button
            variant="ghost"
            size="icon"
            onClick={onDelete}
            title="Delete field"
            className="text-destructive hover:text-destructive"
            data-testid={`delete-field-${fieldName}`}
          >
            <Trash2 className="w-4 h-4" />
          </Button>
        )}
      </div>
    </div>
  )
}
