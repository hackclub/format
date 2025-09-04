'use client'

import { useEffect } from 'react'
import { useLexicalComposerContext } from '@lexical/react/LexicalComposerContext'
import {
  KEY_ARROW_DOWN_COMMAND,
  KEY_ARROW_UP_COMMAND,
  KEY_ENTER_COMMAND,
  COMMAND_PRIORITY_NORMAL,
  FORMAT_TEXT_COMMAND,
} from 'lexical'

export function KeyboardShortcutsPlugin() {
  const [editor] = useLexicalComposerContext()

  useEffect(() => {
    return editor.registerCommand(
      KEY_ENTER_COMMAND,
      (event: KeyboardEvent) => {
        // Handle Enter key behavior
        return false // Let default behavior handle it
      },
      COMMAND_PRIORITY_NORMAL
    )
  }, [editor])

  useEffect(() => {
    const handleKeyDown = (event: KeyboardEvent) => {
      const { metaKey, ctrlKey, key } = event
      const isModifierPressed = metaKey || ctrlKey

      if (isModifierPressed) {
        switch (key) {
          case 'b':
            event.preventDefault()
            editor.dispatchCommand(FORMAT_TEXT_COMMAND, 'bold')
            break
          case 'i':
            event.preventDefault()
            editor.dispatchCommand(FORMAT_TEXT_COMMAND, 'italic')
            break
          case 'u':
            event.preventDefault()
            editor.dispatchCommand(FORMAT_TEXT_COMMAND, 'underline')
            break
          // Note: Cmd+K for links is handled in the toolbar component
        }
      }
    }

    document.addEventListener('keydown', handleKeyDown)
    return () => document.removeEventListener('keydown', handleKeyDown)
  }, [editor])

  return null
}
