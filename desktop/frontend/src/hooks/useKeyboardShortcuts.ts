import { useEffect, useCallback } from 'react'

export interface ShortcutDefinition {
  key: string
  ctrl?: boolean
  meta?: boolean  // Cmd on macOS
  shift?: boolean
  alt?: boolean
  handler: () => void
  description: string
}

const isMac = typeof navigator !== 'undefined'
  ? navigator.platform.toUpperCase().indexOf('MAC') >= 0
  : false

export function useKeyboardShortcuts(shortcuts: ShortcutDefinition[]) {
  const handleKeyDown = useCallback((event: KeyboardEvent) => {
    // Don't trigger shortcuts when typing in inputs
    const target = event.target as HTMLElement
    if (target.tagName === 'INPUT' || target.tagName === 'TEXTAREA') {
      // Allow specific shortcuts even in inputs
      const allowedInInputs = ['Escape', 'Enter']
      if (!allowedInInputs.includes(event.key)) {
        return
      }
    }

    for (const shortcut of shortcuts) {
      const metaOrCtrl = isMac ? event.metaKey : event.ctrlKey
      const expectedMetaOrCtrl = shortcut.meta || shortcut.ctrl

      if (
        event.key.toLowerCase() === shortcut.key.toLowerCase() &&
        metaOrCtrl === !!expectedMetaOrCtrl &&
        event.shiftKey === !!shortcut.shift &&
        event.altKey === !!shortcut.alt
      ) {
        event.preventDefault()
        shortcut.handler()
        return
      }
    }
  }, [shortcuts])

  useEffect(() => {
    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [handleKeyDown])
}

// Helper to format shortcut for display
export function formatShortcut(shortcut: ShortcutDefinition): string {
  const parts: string[] = []
  if (shortcut.ctrl || shortcut.meta) {
    parts.push(isMac ? '⌘' : 'Ctrl')
  }
  if (shortcut.shift) {
    parts.push(isMac ? '⇧' : 'Shift')
  }
  if (shortcut.alt) {
    parts.push(isMac ? '⌥' : 'Alt')
  }
  parts.push(shortcut.key.toUpperCase())
  return parts.join(isMac ? '' : '+')
}
