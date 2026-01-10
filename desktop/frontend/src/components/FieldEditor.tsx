import { useState } from 'react'
import { useTranslation } from 'react-i18next'
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
  // Default visibility based on field type:
  // - Input (password): Hidden by default (standard UX)
  // - Textarea (SSH key, etc.): Visible by default (users need to verify pasted content)
  //
  // Note on audit logging: Textarea defaults visible, so initial view is not logged.
  // This is intentional - users pasting SSH keys/certificates need immediate verification.
  // Audit is triggered when user toggles from hidden→visible (active reveal action).
  const isTextarea = field.inputType === 'textarea'
  const [isVisible, setIsVisible] = useState(isTextarea)
  const toast = useToast()
  const { t } = useTranslation()

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
        toast.success(t('secrets.copiedMessage'))
      } catch (err) {
        console.error('Failed to copy:', err)
        toast.error(t('secrets.failedToCopy'))
      }
      return
    }

    try {
      // Security: Value is fetched server-side to prevent caller manipulation
      await CopyFieldValue(secretKey, fieldName)
      toast.success(t('secrets.copiedMessage'))
    } catch (err) {
      console.error('Failed to copy:', err)
      toast.error(t('secrets.failedToCopy'))
    }
  }

  const handleChange = (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => {
    if (onChange) {
      onChange(e.target.value)
    }
  }

  // === Separation of Concerns ===
  // 1. Display masking: When to show '••••••••' instead of actual value
  //    - Only in READ mode (readOnly=true) for sensitive fields when hidden
  // 2. Input blocking: Controlled solely by readOnly prop
  // 3. Visual styling: Applied when display is masked
  // 4. Textarea masking: In EDIT mode, use -webkit-text-security for dot display
  //    - Per ADR-005: "Same UX as single-line Input"
  //    - Works in Chromium (Wails runtime), shows dots like password input
  //    - Eye button toggles masking on/off
  const hasContent = (field.value?.length ?? 0) > 0
  const shouldMaskDisplay = field.sensitive && !isVisible && readOnly && hasContent

  // Display value: masked in read mode when hidden, actual value otherwise
  const displayValue = shouldMaskDisplay ? '••••••••' : field.value

  // Visual styling for masked state (read mode only)
  const maskedStyles = shouldMaskDisplay ? 'cursor-not-allowed bg-muted' : ''

  // For textarea in edit mode, apply text-security for password-like masking
  // Uses -webkit-text-security: disc (Chromium/Wails) to show dots like password input
  // Per ADR-005: "Same UX as single-line Input"
  const textareaEditMask = isTextarea && field.sensitive && !isVisible && !readOnly

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
            title={field.sensitive ? t('fields.markNonSensitive') : t('fields.markSensitive')}
            data-testid={`toggle-sensitive-${fieldName}`}
          >
            {field.sensitive ? t('fields.unmarkSensitive') : t('fields.markSensitive')}
          </Button>
        )}
      </div>
      <div className={`flex ${isTextarea ? 'items-start' : 'items-center'} gap-2`}>
        {isTextarea ? (
          <Textarea
            value={displayValue}
            readOnly={readOnly}
            onChange={handleChange}
            className={`font-mono min-h-[120px] whitespace-pre-wrap ${maskedStyles}`}
            style={textareaEditMask ? { WebkitTextSecurity: 'disc' } as React.CSSProperties : undefined}
            title={shouldMaskDisplay || textareaEditMask ? 'Click 👁 to view' : undefined}
            data-testid={`field-value-${fieldName}`}
            data-masked={textareaEditMask || shouldMaskDisplay ? 'true' : undefined}
          />
        ) : (
          <Input
            type={field.sensitive && !isVisible ? 'password' : 'text'}
            value={displayValue}
            readOnly={readOnly}
            onChange={handleChange}
            className={`font-mono ${maskedStyles}`}
            data-testid={`field-value-${fieldName}`}
          />
        )}
        {field.sensitive && (
          <Button
            variant="ghost"
            size="icon"
            onClick={handleToggleVisibility}
            title={isVisible ? t('common.hide') : t('common.show')}
            data-testid={`toggle-field-${fieldName}`}
          >
            {isVisible ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
          </Button>
        )}
        <Button
          variant="ghost"
          size="icon"
          onClick={handleCopy}
          title={t('common.copy')}
          data-testid={`copy-field-${fieldName}`}
        >
          <Copy className="w-4 h-4" />
        </Button>
        {!readOnly && onDelete && (
          <Button
            variant="ghost"
            size="icon"
            onClick={onDelete}
            title={t('secrets.deleteField')}
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
