import { useTranslation } from 'react-i18next'
import { FieldEditor, FieldDTO, InputType } from './FieldEditor'

export type { FieldDTO, InputType }

interface FieldsSectionProps {
  secretKey: string
  fields: Record<string, FieldDTO>
  fieldOrder: string[]
  readOnly?: boolean
  onFieldChange?: (fieldName: string, value: string) => void
  onFieldSensitiveToggle?: (fieldName: string) => void
  onFieldDelete?: (fieldName: string) => void
}

export function FieldsSection({
  secretKey,
  fields,
  fieldOrder,
  readOnly = true,
  onFieldChange,
  onFieldSensitiveToggle,
  onFieldDelete
}: FieldsSectionProps) {
  const { t } = useTranslation()
  // Use fieldOrder if available, otherwise fallback to object keys
  const orderedFieldNames = fieldOrder.length > 0 ? fieldOrder : Object.keys(fields)

  if (orderedFieldNames.length === 0) {
    return (
      <div className="text-sm text-muted-foreground italic">
        {t('secrets.noFieldsDefined')}
      </div>
    )
  }

  return (
    <div className="space-y-4" data-testid="fields-section">
      {orderedFieldNames.map((fieldName) => {
        const field = fields[fieldName]
        if (!field) return null

        return (
          <FieldEditor
            key={fieldName}
            secretKey={secretKey}
            fieldName={fieldName}
            field={field}
            readOnly={readOnly}
            onChange={onFieldChange ? (value) => onFieldChange(fieldName, value) : undefined}
            onSensitiveToggle={onFieldSensitiveToggle ? () => onFieldSensitiveToggle(fieldName) : undefined}
            onDelete={onFieldDelete ? () => onFieldDelete(fieldName) : undefined}
          />
        )
      })}
    </div>
  )
}
