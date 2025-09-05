'use client'

import { useCallback, useEffect, useState } from 'react'
import {
  $getSelection,
  $isRangeSelection,
  FORMAT_TEXT_COMMAND,
  SELECTION_CHANGE_COMMAND,
  COMMAND_PRIORITY_CRITICAL,
} from 'lexical'
import { $isLinkNode, TOGGLE_LINK_COMMAND } from '@lexical/link'
import {
  INSERT_ORDERED_LIST_COMMAND,
  INSERT_UNORDERED_LIST_COMMAND,
  REMOVE_LIST_COMMAND,
  $isListNode,
} from '@lexical/list'
import { $isHeadingNode } from '@lexical/rich-text'
import { $getNearestNodeOfType, mergeRegister } from '@lexical/utils'
import { useLexicalComposerContext } from '@lexical/react/LexicalComposerContext'
import { $createImageNode } from './ImageNode'
import { assetsAPI } from '@/lib/api'
import { LoadingSpinner } from './LoadingSpinner'

interface ToolbarState {
  isBold: boolean
  isItalic: boolean
  isUnderline: boolean
  isStrikethrough: boolean
  isLink: boolean
  isOrderedList: boolean
  isUnorderedList: boolean
  blockType: string
}

interface EditorToolbarProps {
  onProcessAndCopy?: () => void
  transforming?: boolean
  copied?: boolean
  hasContent?: boolean
  hasGmailAccess?: boolean
  onRequestGmailAccess?: () => void
}

export function EditorToolbar({ onProcessAndCopy, transforming, copied, hasContent, hasGmailAccess, onRequestGmailAccess }: EditorToolbarProps) {
  const [editor] = useLexicalComposerContext()
  const [toolbarState, setToolbarState] = useState<ToolbarState>({
    isBold: false,
    isItalic: false,
    isUnderline: false,
    isStrikethrough: false,
    isLink: false,
    isOrderedList: false,
    isUnorderedList: false,
    blockType: 'paragraph',
  })
  const [showLinkInput, setShowLinkInput] = useState(false)
  const [linkUrl, setLinkUrl] = useState('')
  const [uploading, setUploading] = useState(false)

  const updateToolbar = useCallback(() => {
    const selection = $getSelection()
    if ($isRangeSelection(selection)) {
      const anchorNode = selection.anchor.getNode()
      const element = anchorNode.getKey() === 'root' ? anchorNode : anchorNode.getTopLevelElementOrThrow()

      // Update formatting state
      setToolbarState(prev => ({
        ...prev,
        isBold: selection.hasFormat('bold'),
        isItalic: selection.hasFormat('italic'),
        isUnderline: selection.hasFormat('underline'),
        isStrikethrough: selection.hasFormat('strikethrough'),
      }))

      // Update link state 
      const node = anchorNode
      const parent = node.getParent()
      const linkNode = $isLinkNode(parent) ? parent : ($isLinkNode(node) ? node : null)
      
      const isOnLink = !!linkNode
      setToolbarState(prev => ({ ...prev, isLink: isOnLink }))

      // Update list state
      if ($isListNode(element)) {
        const listType = element.getTag()
        setToolbarState(prev => ({
          ...prev,
          isOrderedList: listType === 'ol',
          isUnorderedList: listType === 'ul',
          blockType: 'list',
        }))
      } else {
        setToolbarState(prev => ({
          ...prev,
          isOrderedList: false,
          isUnorderedList: false,
          blockType: $isHeadingNode(element) ? element.getTag() : 'paragraph',
        }))
      }
    }
  }, [editor])
  
  // Separate effect to handle link editor show/hide
  useEffect(() => {
    if (toolbarState.isLink) {
      // Show link editor when on a link
      editor.getEditorState().read(() => {
        const selection = $getSelection()
        if ($isRangeSelection(selection)) {
          const node = selection.anchor.getNode()
          const parent = node.getParent()
          const linkNode = $isLinkNode(parent) ? parent : ($isLinkNode(node) ? node : null)
          
          if (linkNode) {
            const currentUrl = linkNode.getURL()
            setLinkUrl(currentUrl)
            setShowLinkInput(true)
            console.log('🔗 Showing link editor for:', currentUrl)
          }
        }
      })
    } else {
      // Hide link editor when not on a link
      if (showLinkInput) {
        setShowLinkInput(false)
        setLinkUrl('')
        console.log('🔗 Hiding link editor - moved away from link')
      }
    }
  }, [toolbarState.isLink, editor, showLinkInput])

  useEffect(() => {
    return mergeRegister(
      editor.registerUpdateListener(({ editorState }) => {
        editorState.read(() => {
          updateToolbar()
        })
      }),
      editor.registerCommand(
        SELECTION_CHANGE_COMMAND,
        () => {
          updateToolbar()
          return false
        },
        COMMAND_PRIORITY_CRITICAL,
      ),
    )
  }, [editor, updateToolbar])

  const formatText = (format: string) => {
    editor.dispatchCommand(FORMAT_TEXT_COMMAND, format as any)
  }

  const insertLink = () => {
    if (!toolbarState.isLink) {
      // Creating new link
      setLinkUrl('')
      setShowLinkInput(true)
    } else {
      // Editing existing link - get current URL first
      editor.getEditorState().read(() => {
        const selection = $getSelection()
        if ($isRangeSelection(selection)) {
          const node = selection.anchor.getNode()
          const parent = node.getParent()
          const linkNode = $isLinkNode(parent) ? parent : ($isLinkNode(node) ? node : null)
          
          if (linkNode) {
            const currentUrl = linkNode.getURL()
            setLinkUrl(currentUrl)
            setShowLinkInput(true)
            console.log('📝 Editing existing link:', currentUrl)
          }
        }
      })
    }
  }

  const handleLinkSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    if (linkUrl) {
      editor.dispatchCommand(TOGGLE_LINK_COMMAND, linkUrl.startsWith('http') ? linkUrl : `https://${linkUrl}`)
    }
    setShowLinkInput(false)
    setLinkUrl('')
  }

  const insertList = (listType: 'ul' | 'ol') => {
    if (listType === 'ul') {
      if (toolbarState.isUnorderedList) {
        editor.dispatchCommand(REMOVE_LIST_COMMAND, undefined)
      } else {
        editor.dispatchCommand(INSERT_UNORDERED_LIST_COMMAND, undefined)
      }
    } else {
      if (toolbarState.isOrderedList) {
        editor.dispatchCommand(REMOVE_LIST_COMMAND, undefined)
      } else {
        editor.dispatchCommand(INSERT_ORDERED_LIST_COMMAND, undefined)
      }
    }
  }

  const insertImage = () => {
    const input = document.createElement('input')
    input.type = 'file'
    input.accept = 'image/*'
    input.onchange = async (e) => {
      const file = (e.target as HTMLInputElement).files?.[0]
      if (!file) return

      try {
        setUploading(true)
        console.log('Uploading image:', file.name)
        
        const asset = await assetsAPI.uploadFile(file)
        console.log('Image uploaded:', asset)

        editor.update(() => {
          const imageNode = $createImageNode({
            src: asset.url,
            altText: file.name,
            width: asset.width,
            height: asset.height,
          })

          const selection = $getSelection()
          if (selection) {
            selection.insertNodes([imageNode])
          }
        })
      } catch (error) {
        console.error('Failed to upload image:', error)
        // Could show error state here
      } finally {
        setUploading(false)
      }
    }
    input.click()
  }

  // Keyboard shortcuts
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
        e.preventDefault()
        insertLink()
      }
    }

    document.addEventListener('keydown', handleKeyDown)
    return () => document.removeEventListener('keydown', handleKeyDown)
  }, [])

  return (
    <div className="fixed bottom-20 left-1/2 transform -translate-x-1/2 z-10 bg-white border border-gray-300 rounded-full px-4 py-2 shadow-lg flex items-center gap-2">
      {/* Format buttons */}
      <button
        onClick={() => formatText('bold')}
        className={`px-3 py-1 rounded text-sm font-medium ${
          toolbarState.isBold ? 'bg-blue-100 text-blue-800' : 'bg-white hover:bg-gray-100'
        } border`}
        title="Bold (Cmd+B)"
      >
        <strong>B</strong>
      </button>

      <button
        onClick={() => formatText('italic')}
        className={`px-3 py-1 rounded text-sm font-medium ${
          toolbarState.isItalic ? 'bg-blue-100 text-blue-800' : 'bg-white hover:bg-gray-100'
        } border`}
        title="Italic (Cmd+I)"
      >
        <em>I</em>
      </button>

      <button
        onClick={() => formatText('underline')}
        className={`px-3 py-1 rounded text-sm font-medium ${
          toolbarState.isUnderline ? 'bg-blue-100 text-blue-800' : 'bg-white hover:bg-gray-100'
        } border`}
        title="Underline (Cmd+U)"
      >
        <u>U</u>
      </button>

      <button
        onClick={() => formatText('strikethrough')}
        className={`px-3 py-1 rounded text-sm font-medium ${
          toolbarState.isStrikethrough ? 'bg-blue-100 text-blue-800' : 'bg-white hover:bg-gray-100'
        } border`}
        title="Strikethrough"
      >
        <s>S</s>
      </button>

      <div className="w-px h-4 bg-gray-300" />

      {/* List buttons */}
      <button
        onClick={() => insertList('ul')}
        className={`px-3 py-1 rounded text-sm font-medium ${
          toolbarState.isUnorderedList ? 'bg-blue-100 text-blue-800' : 'bg-white hover:bg-gray-100'
        } border`}
        title="Bullet List"
      >
        •
      </button>

      <button
        onClick={() => insertList('ol')}
        className={`px-3 py-1 rounded text-sm font-medium ${
          toolbarState.isOrderedList ? 'bg-blue-100 text-blue-800' : 'bg-white hover:bg-gray-100'
        } border`}
        title="Numbered List"
      >
        1.
      </button>

      <div className="w-px h-4 bg-gray-300" />

      {/* Link button */}
      <button
        onClick={insertLink}
        className={`px-3 py-1 rounded text-sm font-medium ${
          toolbarState.isLink ? 'bg-blue-100 text-blue-800' : 'bg-white hover:bg-gray-100'
        } border`}
        title="Insert/Edit Link (Cmd+K)"
      >
        🔗
      </button>

      {/* Image button */}
      <button
        onClick={insertImage}
        disabled={uploading}
        className={`px-3 py-1 rounded text-sm font-medium ${
          uploading ? 'bg-gray-100 cursor-not-allowed' : 'bg-white hover:bg-gray-100'
        } border`}
        title="Insert Image"
      >
        {uploading ? '⏳' : '🖼️'}
      </button>

      <div className="w-px h-4 bg-gray-300" />

      {/* Gmail Access Button */}
      {!hasGmailAccess && onRequestGmailAccess && (
        <button
          onClick={onRequestGmailAccess}
          className="px-3 py-1 rounded text-xs font-medium bg-blue-500 text-white hover:bg-blue-600 border"
          title="Enable Gmail image auto-fetch"
        >
          📧
        </button>
      )}

      {/* Copy to Clipboard Button */}
      {onProcessAndCopy && (
        <button
          onClick={onProcessAndCopy}
          disabled={transforming || !hasContent}
          className={`px-4 py-1 rounded text-sm font-medium border ${
            copied ? 'bg-green-100 text-green-800 border-green-300' :
            transforming ? 'bg-gray-100 text-gray-600 border-gray-300' :
            'bg-hack-green text-white border-hack-green hover:bg-green-600'
          }`}
        >
          {copied ? (
            '✅ Copied!'
          ) : transforming ? (
            <div className="flex items-center space-x-1">
              <LoadingSpinner size="sm" />
              <span>Processing...</span>
            </div>
          ) : (
            '📋 Copy'
          )}
        </button>
      )}

      {/* Link input modal */}
      {showLinkInput && (
        <div className="absolute -top-32 left-1/2 transform -translate-x-1/2 z-20 bg-white border border-gray-300 rounded-md shadow-lg p-3 min-w-80">
          <form onSubmit={handleLinkSubmit} className="flex flex-col gap-2">
            <input
              type="url"
              placeholder="Enter URL..."
              value={linkUrl}
              onChange={(e) => setLinkUrl(e.target.value)}
              className="w-full px-3 py-2 border border-gray-300 rounded text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
              autoFocus
            />
            <div className="flex gap-2">
              <button
                type="submit"
                className="px-3 py-1 bg-blue-500 text-white text-sm rounded hover:bg-blue-600"
              >
                {toolbarState.isLink ? 'Update' : 'Add'}
              </button>
              {toolbarState.isLink && (
                <button
                  type="button"
                  onClick={() => {
                    editor.dispatchCommand(TOGGLE_LINK_COMMAND, null)
                    setShowLinkInput(false)
                    setLinkUrl('')
                  }}
                  className="px-3 py-1 bg-red-500 text-white text-sm rounded hover:bg-red-600"
                >
                  Remove
                </button>
              )}
              <div className="flex-1" />
              {toolbarState.isLink && (
                <button
                  type="button"
                  onClick={() => {
                    if (linkUrl) {
                      window.open(linkUrl, '_blank', 'noopener,noreferrer')
                    }
                  }}
                  className="px-3 py-1 bg-white border border-gray-300 text-gray-700 text-sm rounded hover:bg-gray-50"
                >
                  ↗
                </button>
              )}
            </div>
          </form>
        </div>
      )}
    </div>
  )
}
