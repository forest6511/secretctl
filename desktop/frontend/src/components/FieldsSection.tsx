import { FieldEditor } from './FieldEditor'

interface FieldDTO {
  value: string
  sensitive: boolean
  aliases?: string[]
  kind?: string
  hint?: string
}

interface FieldsSectionProps {
  secretKey: string
  fields: Record<string, FieldDTO>
  fieldOrder: string[]
  readOnly?: boolean
}

export function FieldsSection({ secretKey, fields, fieldOrder, readOnly = true }: FieldsSectionProps) {
  // Use fieldOrder if available, otherwise fallback to object keys
  const orderedFieldNames = fieldOrder.length > 0 ? fieldOrder : Object.keys(fields)

  if (orderedFieldNames.length === 0) {
    return (
      <div className="text-sm text-muted-foreground italic">
        No fields defined
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
          />
        )
      })}
    </div>
  )
}
