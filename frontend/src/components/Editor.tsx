'use client'

import { useCallback, useEffect, useState } from 'react'
import { $generateHtmlFromNodes, $generateNodesFromDOM } from '@lexical/html'
import { $getRoot, $getSelection, COMMAND_PRIORITY_HIGH, PASTE_COMMAND } from 'lexical'
import { LexicalComposer } from '@lexical/react/LexicalComposer'
import { RichTextPlugin } from '@lexical/react/LexicalRichTextPlugin'
import { ContentEditable } from '@lexical/react/LexicalContentEditable'
import { HistoryPlugin } from '@lexical/react/LexicalHistoryPlugin'
import { AutoFocusPlugin } from '@lexical/react/LexicalAutoFocusPlugin'
import { ListPlugin } from '@lexical/react/LexicalListPlugin'
import { LinkPlugin } from '@lexical/react/LexicalLinkPlugin'
import { useLexicalComposerContext } from '@lexical/react/LexicalComposerContext'
import { OnChangePlugin } from '@lexical/react/LexicalOnChangePlugin'
import { RichTextPlugin as LexicalRichTextPlugin } from '@lexical/react/LexicalRichTextPlugin'
import { CheckListPlugin } from '@lexical/react/LexicalCheckListPlugin'
import { TabIndentationPlugin } from '@lexical/react/LexicalTabIndentationPlugin'
import LexicalErrorBoundary from '@lexical/react/LexicalErrorBoundary'

import { HeadingNode, QuoteNode } from '@lexical/rich-text'
import { ListItemNode, ListNode } from '@lexical/list'
import { CodeHighlightNode, CodeNode } from '@lexical/code'
import { LinkNode, AutoLinkNode } from '@lexical/link'
import { ImageNode } from './ImageNode'

import DOMPurify from 'dompurify'
import { htmlAPI } from '@/lib/api'
import { TransformResult } from '@/types'
import { EditorToolbar } from './EditorToolbar'
import { KeyboardShortcutsPlugin } from './KeyboardShortcutsPlugin'
import { DragDropPlugin } from './DragDropPlugin'

const theme = {
  ltr: 'ltr',
  rtl: 'rtl',
  placeholder: 'editor-placeholder',
  paragraph: 'editor-paragraph',
  quote: 'editor-quote',
  heading: {
    h1: 'editor-heading-h1',
    h2: 'editor-heading-h2',
    h3: 'editor-heading-h3',
    h4: 'editor-heading-h4',
    h5: 'editor-heading-h5',
  },
  list: {
    nested: {
      listitem: 'editor-nested-listitem',
    },
    ol: 'editor-list-ol',
    ul: 'editor-list-ul',
    listitem: 'editor-listitem',
  },
  image: 'editor-image',
  link: 'editor-link',
  text: {
    bold: 'editor-text-bold',
    italic: 'editor-text-italic',
    overflowed: 'editor-text-overflowed',
    hashtag: 'editor-text-hashtag',
    underline: 'editor-text-underline',
    strikethrough: 'editor-text-strikethrough',
    underlineStrikethrough: 'editor-text-underlineStrikethrough',
    code: 'editor-text-code',
  },
  code: 'editor-code',
  codeHighlight: {
    atrule: 'editor-tokenAttr',
    attr: 'editor-tokenAttr',
    boolean: 'editor-tokenProperty',
    builtin: 'editor-tokenSelector',
    cdata: 'editor-tokenComment',
    char: 'editor-tokenSelector',
    class: 'editor-tokenFunction',
    'class-name': 'editor-tokenFunction',
    comment: 'editor-tokenComment',
    constant: 'editor-tokenProperty',
    deleted: 'editor-tokenProperty',
    doctype: 'editor-tokenComment',
    entity: 'editor-tokenOperator',
    function: 'editor-tokenFunction',
    important: 'editor-tokenVariable',
    inserted: 'editor-tokenSelector',
    keyword: 'editor-tokenAttr',
    namespace: 'editor-tokenVariable',
    number: 'editor-tokenProperty',
    operator: 'editor-tokenOperator',
    prolog: 'editor-tokenComment',
    property: 'editor-tokenProperty',
    punctuation: 'editor-tokenPunctuation',
    regex: 'editor-tokenVariable',
    selector: 'editor-tokenSelector',
    string: 'editor-tokenSelector',
    symbol: 'editor-tokenProperty',
    tag: 'editor-tokenProperty',
    url: 'editor-tokenOperator',
    variable: 'editor-tokenVariable',
  },
}

function onError(error: Error) {
  console.error(error)
}

interface EditorProps {
  onContentChange: (html: string) => void
  initialContent?: string
}

function MyOnChangePlugin({ onChange }: { onChange: (html: string) => void }) {
  const [editor] = useLexicalComposerContext()
  
  useEffect(() => {
    return editor.registerUpdateListener(({ editorState }) => {
      editorState.read(() => {
        const htmlString = $generateHtmlFromNodes(editor, null)
        onChange(htmlString)
      })
    })
  }, [editor, onChange])
  
  return null
}

function PastePlugin() {
  const [editor] = useLexicalComposerContext()
  
  useEffect(() => {
    return editor.registerCommand(
      PASTE_COMMAND,
      (event: ClipboardEvent) => {
        const clipboardData = event.clipboardData
        if (!clipboardData) return false
        
        const htmlData = clipboardData.getData('text/html')
        if (htmlData) {
          console.log('Pasting HTML with length:', htmlData.length)
          
          // Preprocess to preserve blank line paragraphs properly
          let preprocessedHtml = htmlData
          
          // Remove Gmail wrapper spans that can interfere with structure
          preprocessedHtml = preprocessedHtml.replace(/<span[^>]*class="im"[^>]*>/g, '')
          preprocessedHtml = preprocessedHtml.replace(/<\/span>/g, '')
          
          // Handle Gmail signature pattern: remove <br> immediately after '--'
          preprocessedHtml = preprocessedHtml.replace(/<span[^>]*>--<\/span><br>/g, '<span>--</span>')
          
          // Also handle the div structure: <div><span>--</span><br></div><div>content</div>
          preprocessedHtml = preprocessedHtml.replace(
            /<div[^>]*><span[^>]*>--<\/span><br><\/div>\s*<div/g, 
            '<div><span>--</span><br>'
          )
          
          // Convert all <div><br></div> to proper paragraph breaks that Lexical understands
          preprocessedHtml = preprocessedHtml.replace(/<div[^>]*><br><\/div>/g, '<p><br></p>')
          
          console.log('Preprocessed HTML length:', preprocessedHtml.length)
          console.log('Signature area:', preprocessedHtml.match(/.{100}--{1,3}.{100}/)?.[0] || 'no -- found')
          
          // Sanitize the HTML
          const sanitizedHtml = DOMPurify.sanitize(preprocessedHtml, {
            ALLOWED_TAGS: [
              'p', 'div', 'br', 'strong', 'em', 's', 'u', 'a', 'ul', 'ol', 'li',
              'blockquote', 'hr', 'pre', 'code', 'img', 'h1', 'h2', 'h3', 'h4', 'h5', 'h6', 'b', 'i', 'span', 'font'
            ],
            ALLOWED_ATTR: ['href', 'src', 'alt', 'title', 'target', 'width', 'height', 'style', 'class', 'dir', 'face', 'color'],
          })
          

          
          // Parse the HTML and insert it
          const parser = new DOMParser()
          const dom = parser.parseFromString(sanitizedHtml, 'text/html')
          
          editor.update(() => {
            const nodes = $generateNodesFromDOM(editor, dom)
            const root = $getRoot()
            const selection = $getSelection()
            
            if (selection) {
              selection.insertNodes(nodes)
            } else {
              root.clear()
              root.append(...nodes)
            }
          })
          
          event.preventDefault()
          return true
        }
        
        return false
      },
      COMMAND_PRIORITY_HIGH
    )
  }, [editor])
  
  return null
}

export function Editor({ onContentChange, initialContent }: EditorProps) {
  const initialConfig = {
    namespace: 'FormatEditor',
    theme,
    onError,
    nodes: [
      HeadingNode,
      ListNode,
      ListItemNode,
      QuoteNode,
      CodeNode,
      CodeHighlightNode,
      LinkNode,
      AutoLinkNode,
      ImageNode,
    ],
  }

  return (
    <div className="lexical-editor border rounded-lg">
      <LexicalComposer initialConfig={initialConfig}>
        <EditorToolbar />
        <div className="editor-container">
          <div className="editor-inner">
            <RichTextPlugin
              contentEditable={
                <ContentEditable className="editor-input" />
              }
              placeholder={
                <div className="editor-placeholder">
                  Paste your rich text content here...
                </div>
              }
              ErrorBoundary={LexicalErrorBoundary}
            />
            <HistoryPlugin />
            <AutoFocusPlugin />
            <ListPlugin />
            <LinkPlugin />
            <CheckListPlugin />
            <TabIndentationPlugin />
            <KeyboardShortcutsPlugin />
            <DragDropPlugin />
            <MyOnChangePlugin onChange={onContentChange} />
            <PastePlugin />
          </div>
        </div>
      </LexicalComposer>
    </div>
  )
}

export default Editor
