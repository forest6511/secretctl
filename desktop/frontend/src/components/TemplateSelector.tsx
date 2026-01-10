import { useTranslation } from 'react-i18next'

import { Key, Database, Globe, Terminal, Lock, Unlock, Link } from 'lucide-react'
import { Card, CardContent } from '@/components/ui/card'
import { main } from '../../wailsjs/go/models'

interface TemplateSelectorProps {
  templates: main.TemplateInfo[]
  selectedTemplate: string | null
  onSelect: (templateId: string | null) => void
}

const iconMap: Record<string, JSX.Element> = {
  key: <Key className="w-8 h-8" />,
  database: <Database className="w-8 h-8" />,
  globe: <Globe className="w-8 h-8" />,
  terminal: <Terminal className="w-8 h-8" />,
}

export function TemplateSelector({ templates, selectedTemplate, onSelect }: TemplateSelectorProps) {
  const { t } = useTranslation()
  const selectedTemplateData = templates.find(t => t.id === selectedTemplate)

  if (templates.length === 0) {
    return (
      <div className="text-center text-muted-foreground py-4">
        {t('common.loading')}
      </div>
    )
  }

  return (
    <div className="space-y-4" data-testid="template-selector">
      <label className="text-sm font-medium">{t('templates.selectTemplate')}</label>

      <div className="grid grid-cols-4 gap-3">
        {templates.map((template) => (
          <Card
            key={template.id}
            className={`cursor-pointer transition-all hover:border-primary hover:bg-sky-100 hover:shadow-md ${
              selectedTemplate === template.id
                ? 'border-2 border-primary bg-primary/5'
                : 'border border-border'
            }`}
            onClick={() => onSelect(selectedTemplate === template.id ? null : template.id)}
            data-testid={`template-${template.id}`}
          >
            <CardContent className="flex flex-col items-center justify-center p-4 text-center">
              <div className={`mb-2 ${selectedTemplate === template.id ? 'text-primary' : 'text-muted-foreground'}`}>
                {iconMap[template.icon] || <Key className="w-8 h-8" />}
              </div>
              <span className={`text-sm font-medium ${selectedTemplate === template.id ? 'text-primary' : ''}`}>
                {template.name}
              </span>
            </CardContent>
          </Card>
        ))}
      </div>

      {selectedTemplateData && (
        <div className="rounded-lg border bg-muted/30 p-4 space-y-3" data-testid="template-preview">
          <div className="text-sm text-muted-foreground">
            {selectedTemplateData.description}
          </div>

          <div className="space-y-2">
            <div className="text-sm font-medium">{t('secrets.fields')}:</div>
            <div className="flex flex-wrap gap-2">
              {selectedTemplateData.fields.map((field) => (
                <div
                  key={field.name}
                  className="flex items-center gap-1 text-xs bg-background rounded px-2 py-1 border"
                >
                  {field.sensitive ? (
                    <Lock className="w-3 h-3 text-muted-foreground" />
                  ) : (
                    <Unlock className="w-3 h-3 text-muted-foreground" />
                  )}
                  <span>{field.name}</span>
                </div>
              ))}
            </div>
          </div>

          {Object.keys(selectedTemplateData.bindings).length > 0 && (
            <div className="space-y-2">
              <div className="text-sm font-medium flex items-center gap-1">
                <Link className="w-4 h-4" />
                Auto-configured Bindings:
              </div>
              <div className="flex flex-wrap gap-2">
                {Object.entries(selectedTemplateData.bindings).map(([envVar, fieldName]) => (
                  <div
                    key={envVar}
                    className="text-xs bg-background rounded px-2 py-1 border font-mono"
                  >
                    {envVar} → {fieldName}
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  )
}
