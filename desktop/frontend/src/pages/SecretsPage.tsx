import { useState, useEffect, useRef, useCallback } from 'react'
import {
  Search, Plus, Copy, Trash2, Eye, EyeOff, Key,
  Lock, RefreshCw, FileText, ExternalLink, Tag, ClipboardList
} from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { ConfirmDialog } from '@/components/ConfirmDialog'
import { FieldsSection } from '@/components/FieldsSection'
import { useToast } from '@/hooks/useToast'
import {
  ListSecrets, GetSecret, CreateSecret, UpdateSecret,
  DeleteSecret, CopyToClipboard, Lock as LockVault, ResetIdleTimer
} from '../../wailsjs/go/main/App'
import { main } from '../../wailsjs/go/models'
import { EventsOn } from '../../wailsjs/runtime/runtime'

interface SecretsPageProps {
  onLocked: () => void
  onNavigateToAudit: () => void
}

export function SecretsPage({ onLocked, onNavigateToAudit }: SecretsPageProps) {
  const [secrets, setSecrets] = useState<main.SecretListItem[]>([])
  const [searchQuery, setSearchQuery] = useState('')
  const [selectedKey, setSelectedKey] = useState<string | null>(null)
  const [selectedSecret, setSelectedSecret] = useState<main.Secret | null>(null)
  const [showValue, setShowValue] = useState(false)
  const [isEditing, setIsEditing] = useState(false)
  const [isCreating, setIsCreating] = useState(false)
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false)

  // Form state
  const [formKey, setFormKey] = useState('')
  const [formValue, setFormValue] = useState('')
  const [formNotes, setFormNotes] = useState('')
  const [formUrl, setFormUrl] = useState('')
  const [formTags, setFormTags] = useState('')

  // Refs
  const searchInputRef = useRef<HTMLInputElement>(null)

  // Hooks
  const toast = useToast()

  // Keyboard shortcuts handler
  const handleKeyboardShortcuts = useCallback((e: KeyboardEvent) => {
    const isMod = e.metaKey || e.ctrlKey

    if (isMod && e.key === 'n') {
      // Cmd/Ctrl + N: New secret
      e.preventDefault()
      handleStartCreate()
    } else if (isMod && e.key === 'l') {
      // Cmd/Ctrl + L: Lock vault
      e.preventDefault()
      handleLock()
    } else if (isMod && e.key === 'f') {
      // Cmd/Ctrl + F: Focus search
      e.preventDefault()
      searchInputRef.current?.focus()
    } else if (isMod && e.key === 'c' && selectedSecret && !isEditing && !isCreating) {
      // Cmd/Ctrl + C: Copy secret value (only when not editing text)
      const selection = window.getSelection()
      if (!selection || selection.toString() === '') {
        e.preventDefault()
        handleCopy()
      }
    } else if (isMod && e.key === 's' && (isEditing || isCreating)) {
      // Cmd/Ctrl + S: Save
      e.preventDefault()
      handleSave()
    } else if (e.key === 'Escape') {
      // Escape: Cancel editing
      if (isEditing || isCreating) {
        e.preventDefault()
        handleCancel()
      }
    }
  }, [selectedSecret, isEditing, isCreating])

  useEffect(() => {
    loadSecrets()

    // Listen for lock events
    const unlisten = EventsOn('vault:locked', () => {
      onLocked()
    })

    // Reset idle timer on activity
    const handleActivity = () => ResetIdleTimer()
    window.addEventListener('mousemove', handleActivity)
    window.addEventListener('keydown', handleActivity)
    window.addEventListener('keydown', handleKeyboardShortcuts)

    return () => {
      unlisten()
      window.removeEventListener('mousemove', handleActivity)
      window.removeEventListener('keydown', handleActivity)
      window.removeEventListener('keydown', handleKeyboardShortcuts)
    }
  }, [onLocked, handleKeyboardShortcuts])

  const loadSecrets = async () => {
    try {
      const list = await ListSecrets()
      setSecrets(list || [])
    } catch (err) {
      console.error('Failed to load secrets:', err)
    }
  }

  const handleSelectSecret = async (key: string) => {
    setSelectedKey(key)
    setShowValue(false)
    setIsEditing(false)
    setIsCreating(false)
    try {
      const secret = await GetSecret(key)
      setSelectedSecret(secret)
    } catch (err) {
      console.error('Failed to get secret:', err)
    }
  }

  const handleCopy = async () => {
    if (!selectedSecret?.value) return
    try {
      await CopyToClipboard(selectedSecret.value)
      toast.success('Copied! Auto-clears in 30s')
    } catch (err) {
      console.error('Failed to copy:', err)
      toast.error('Failed to copy to clipboard')
    }
  }

  const handleLock = async () => {
    try {
      await LockVault()
      onLocked()
    } catch (err) {
      console.error('Failed to lock:', err)
    }
  }

  const handleDeleteClick = () => {
    if (!selectedKey) return
    setDeleteDialogOpen(true)
  }

  const handleDeleteConfirm = async () => {
    if (!selectedKey) return
    setDeleteDialogOpen(false)

    try {
      await DeleteSecret(selectedKey)
      toast.success('Secret deleted')
      setSelectedKey(null)
      setSelectedSecret(null)
      await loadSecrets()
    } catch (err) {
      console.error('Failed to delete:', err)
      toast.error('Failed to delete secret')
    }
  }

  const handleStartCreate = () => {
    setIsCreating(true)
    setIsEditing(false)
    setSelectedKey(null)
    setSelectedSecret(null)
    setFormKey('')
    setFormValue('')
    setFormNotes('')
    setFormUrl('')
    setFormTags('')
    setShowValue(true)
  }

  const handleStartEdit = () => {
    if (!selectedSecret) return
    setIsEditing(true)
    setFormKey(selectedSecret.key)
    setFormValue(selectedSecret.value || '')
    setFormNotes(selectedSecret.notes || '')
    setFormUrl(selectedSecret.url || '')
    setFormTags(selectedSecret.tags?.join(', ') || '')
    setShowValue(true)
  }

  const handleSave = async () => {
    if (!formKey.trim()) {
      toast.error('Key is required')
      return
    }
    if (!formValue.trim()) {
      toast.error('Value is required')
      return
    }

    const tags = formTags ? formTags.split(',').map(t => t.trim()).filter(Boolean) : []

    try {
      if (isCreating) {
        await CreateSecret(formKey, formValue, formNotes, formUrl, tags)
        toast.success('Secret created')
      } else {
        await UpdateSecret(formKey, formValue, formNotes, formUrl, tags)
        toast.success('Changes saved')
      }
      setIsCreating(false)
      setIsEditing(false)
      await loadSecrets()
      if (formKey) {
        await handleSelectSecret(formKey)
      }
    } catch (err) {
      console.error('Failed to save:', err)
      toast.error(isCreating ? 'Failed to create secret' : 'Failed to save changes')
    }
  }

  const handleCancel = () => {
    setIsCreating(false)
    setIsEditing(false)
    if (selectedKey) {
      handleSelectSecret(selectedKey)
    }
  }

  const filteredSecrets = secrets.filter(s =>
    s.key.toLowerCase().includes(searchQuery.toLowerCase())
  )

  const formatDate = (dateStr: string) => {
    return new Date(dateStr).toLocaleDateString(undefined, {
      year: 'numeric',
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit'
    })
  }

  return (
    <div className="flex h-screen macos-titlebar-padding">
      {/* Sidebar */}
      <div className="w-80 border-r border-border flex flex-col bg-muted/30">
        {/* Header */}
        <div className="p-4 border-b border-border">
          <div className="flex items-center justify-between mb-3">
            <h1 className="text-lg font-semibold">Secrets</h1>
            <div className="flex gap-1">
              <Button variant="ghost" size="icon" onClick={onNavigateToAudit} title="Audit Log" data-testid="audit-button">
                <ClipboardList className="w-4 h-4" />
              </Button>
              <Button variant="ghost" size="icon" onClick={loadSecrets} title="Refresh">
                <RefreshCw className="w-4 h-4" />
              </Button>
              <Button variant="ghost" size="icon" onClick={handleLock} title="Lock">
                <Lock className="w-4 h-4" />
              </Button>
            </div>
          </div>
          <div className="relative">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
            <Input
              ref={searchInputRef}
              placeholder="Search secrets... (⌘F)"
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="pl-9"
              data-testid="search-secrets"
            />
          </div>
        </div>

        {/* Secret List */}
        <div className="flex-1 overflow-y-auto" data-testid="secrets-list">
          {filteredSecrets.map((secret) => (
            <button
              key={secret.key}
              onClick={() => handleSelectSecret(secret.key)}
              className={`w-full p-3 text-left border-b border-border hover:bg-muted/50 transition-colors ${
                selectedKey === secret.key ? 'bg-muted' : ''
              }`}
              data-testid={`secret-item-${secret.key.replace(/\//g, '-')}`}
            >
              <div className="flex items-center gap-2">
                <Key className="w-4 h-4 text-muted-foreground flex-shrink-0" />
                <span className="font-medium truncate">{secret.key}</span>
                {secret.fieldCount > 0 && (
                  <span className="text-xs px-1.5 py-0.5 rounded bg-muted text-muted-foreground">
                    {secret.fieldCount} {secret.fieldCount === 1 ? 'field' : 'fields'}
                  </span>
                )}
              </div>
              {secret.tags && secret.tags.length > 0 && (
                <div className="flex gap-1 mt-1 flex-wrap">
                  {secret.tags.slice(0, 3).map((tag) => (
                    <span key={tag} className="text-xs px-1.5 py-0.5 rounded bg-secondary text-secondary-foreground">
                      {tag}
                    </span>
                  ))}
                </div>
              )}
            </button>
          ))}
          {filteredSecrets.length === 0 && (
            <div className="p-4 text-center text-muted-foreground">
              {searchQuery ? 'No secrets found' : 'No secrets yet'}
            </div>
          )}
        </div>

        {/* Add Button */}
        <div className="p-4 border-t border-border">
          <Button className="w-full" onClick={handleStartCreate} data-testid="add-secret-button">
            <Plus className="w-4 h-4 mr-2" />
            Add Secret
          </Button>
        </div>
      </div>

      {/* Detail Panel */}
      <div className="flex-1 overflow-y-auto p-6">
        {(isCreating || isEditing) ? (
          <Card>
            <CardHeader>
              <CardTitle>{isCreating ? 'New Secret' : 'Edit Secret'}</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="space-y-2">
                <label className="text-sm font-medium">Key</label>
                <Input
                  value={formKey}
                  onChange={(e) => setFormKey(e.target.value)}
                  placeholder="e.g., aws/production/api-key"
                  disabled={isEditing}
                  data-testid="secret-key-input"
                />
              </div>
              <div className="space-y-2">
                <label className="text-sm font-medium">Value</label>
                <div className="relative">
                  <Input
                    type={showValue ? 'text' : 'password'}
                    value={formValue}
                    onChange={(e) => setFormValue(e.target.value)}
                    placeholder="Secret value"
                    className="pr-10"
                    data-testid="secret-value-input"
                  />
                  <button
                    type="button"
                    onClick={() => setShowValue(!showValue)}
                    className="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
                  >
                    {showValue ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
                  </button>
                </div>
              </div>
              <div className="space-y-2">
                <label className="text-sm font-medium">URL (optional)</label>
                <Input
                  value={formUrl}
                  onChange={(e) => setFormUrl(e.target.value)}
                  placeholder="https://example.com"
                  data-testid="secret-url-input"
                />
              </div>
              <div className="space-y-2">
                <label className="text-sm font-medium">Tags (comma-separated)</label>
                <Input
                  value={formTags}
                  onChange={(e) => setFormTags(e.target.value)}
                  placeholder="production, aws, api"
                  data-testid="secret-tags-input"
                />
              </div>
              <div className="space-y-2">
                <label className="text-sm font-medium">Notes (optional)</label>
                <textarea
                  value={formNotes}
                  onChange={(e) => setFormNotes(e.target.value)}
                  placeholder="Additional notes..."
                  className="w-full min-h-[100px] rounded-md border border-border bg-transparent px-3 py-2 text-sm"
                  data-testid="secret-notes-input"
                />
              </div>
              <div className="flex gap-2 pt-4">
                <Button onClick={handleSave} data-testid="save-secret-button">
                  {isCreating ? 'Create' : 'Save'}
                </Button>
                <Button variant="outline" onClick={handleCancel} data-testid="cancel-button">
                  Cancel
                </Button>
              </div>
            </CardContent>
          </Card>
        ) : selectedSecret ? (
          <Card>
            <CardHeader>
              <div className="flex items-center justify-between">
                <CardTitle className="flex items-center gap-2">
                  <Key className="w-5 h-5" />
                  {selectedSecret.key}
                </CardTitle>
                <div className="flex gap-1">
                  <Button variant="ghost" size="icon" onClick={handleStartEdit} title="Edit" data-testid="edit-secret-button">
                    <FileText className="w-4 h-4" />
                  </Button>
                  <Button variant="ghost" size="icon" onClick={handleDeleteClick} title="Delete" data-testid="delete-secret-button">
                    <Trash2 className="w-4 h-4 text-destructive" />
                  </Button>
                </div>
              </div>
            </CardHeader>
            <CardContent className="space-y-6">
              {/* Fields */}
              {selectedSecret.fields && Object.keys(selectedSecret.fields).length > 0 ? (
                <FieldsSection
                  secretKey={selectedSecret.key}
                  fields={selectedSecret.fields}
                  fieldOrder={selectedSecret.fieldOrder || []}
                  readOnly={true}
                />
              ) : selectedSecret.value ? (
                // Legacy single value fallback
                <div className="space-y-2">
                  <label className="text-sm font-medium text-muted-foreground">Value</label>
                  <div className="flex items-center gap-2">
                    <Input
                      type={showValue ? 'text' : 'password'}
                      value={selectedSecret.value || ''}
                      readOnly
                      className="font-mono"
                      data-testid="secret-value-display"
                    />
                    <Button
                      variant="ghost"
                      size="icon"
                      onClick={() => setShowValue(!showValue)}
                      title={showValue ? 'Hide' : 'Show'}
                      data-testid="toggle-value-visibility"
                    >
                      {showValue ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
                    </Button>
                    <Button
                      variant="ghost"
                      size="icon"
                      onClick={handleCopy}
                      title="Copy (⌘C)"
                      data-testid="copy-secret-button"
                    >
                      <Copy className="w-4 h-4" />
                    </Button>
                  </div>
                </div>
              ) : null}

              {/* URL */}
              {selectedSecret.url && (
                <div className="space-y-2">
                  <label className="text-sm font-medium text-muted-foreground flex items-center gap-1">
                    <ExternalLink className="w-3 h-3" />
                    URL
                  </label>
                  <a
                    href={selectedSecret.url}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="text-sm text-primary hover:underline"
                  >
                    {selectedSecret.url}
                  </a>
                </div>
              )}

              {/* Tags */}
              {selectedSecret.tags && selectedSecret.tags.length > 0 && (
                <div className="space-y-2">
                  <label className="text-sm font-medium text-muted-foreground flex items-center gap-1">
                    <Tag className="w-3 h-3" />
                    Tags
                  </label>
                  <div className="flex gap-1 flex-wrap">
                    {selectedSecret.tags.map((tag) => (
                      <span key={tag} className="px-2 py-1 rounded-full bg-secondary text-secondary-foreground text-xs">
                        {tag}
                      </span>
                    ))}
                  </div>
                </div>
              )}

              {/* Notes */}
              {selectedSecret.notes && (
                <div className="space-y-2">
                  <label className="text-sm font-medium text-muted-foreground flex items-center gap-1">
                    <FileText className="w-3 h-3" />
                    Notes
                  </label>
                  <p className="text-sm whitespace-pre-wrap">{selectedSecret.notes}</p>
                </div>
              )}

              {/* Metadata */}
              <div className="pt-4 border-t border-border text-xs text-muted-foreground space-y-1">
                <p>Created: {formatDate(selectedSecret.createdAt)}</p>
                <p>Updated: {formatDate(selectedSecret.updatedAt)}</p>
              </div>
            </CardContent>
          </Card>
        ) : (
          <div className="flex items-center justify-center h-full text-muted-foreground">
            Select a secret or create a new one
          </div>
        )}
      </div>

      {/* Delete Confirmation Dialog */}
      <ConfirmDialog
        open={deleteDialogOpen}
        title="Delete Secret"
        message={`Are you sure you want to delete "${selectedKey}"? This action cannot be undone.`}
        confirmLabel="Delete"
        cancelLabel="Cancel"
        variant="destructive"
        onConfirm={handleDeleteConfirm}
        onCancel={() => setDeleteDialogOpen(false)}
      />
    </div>
  )
}
