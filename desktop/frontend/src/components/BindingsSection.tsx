import { useTranslation } from 'react-i18next'
import { Trash2, Link } from 'lucide-react'
import { Button } from '@/components/ui/button'

interface BindingsSectionProps {
  bindings: Record<string, string>
  fieldNames: string[]
  readOnly?: boolean
  onDelete?: (envVar: string) => void
}

export function BindingsSection({
  bindings,
  fieldNames,
  readOnly = true,
  onDelete
}: BindingsSectionProps) {
  const { t } = useTranslation()
  const bindingEntries = Object.entries(bindings)

  if (bindingEntries.length === 0) {
    return (
      <div className="text-sm text-muted-foreground italic">
        {t('secrets.noBindings')}
      </div>
    )
  }

  return (
    <div className="space-y-2" data-testid="bindings-section">
      {bindingEntries.map(([envVar, fieldName]) => {
        const fieldExists = fieldNames.includes(fieldName)

        return (
          <div
            key={envVar}
            className="flex items-center gap-2 p-2 bg-muted/50 rounded-md"
            data-testid={`binding-${envVar}`}
          >
            <Link className="w-4 h-4 text-muted-foreground flex-shrink-0" />
            <code className="text-sm font-mono">{envVar}</code>
            <span className="text-sm text-muted-foreground">→</span>
            <code className={`text-sm font-mono ${fieldExists ? '' : 'text-destructive'}`}>
              {fieldName}
            </code>
            {!fieldExists && (
              <span className="text-xs text-destructive">{t('secrets.fieldNotFound')}</span>
            )}
            {!readOnly && onDelete && (
              <Button
                variant="ghost"
                size="icon"
                onClick={() => onDelete(envVar)}
                title={t('common.delete')}
                className="ml-auto h-6 w-6 text-destructive hover:text-destructive"
                data-testid={`delete-binding-${envVar}`}
              >
                <Trash2 className="w-3 h-3" />
              </Button>
            )}
          </div>
        )
      })}
    </div>
  )
}
