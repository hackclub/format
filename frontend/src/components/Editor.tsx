'use client'

import { useCallback, useEffect, useState } from 'react'
import { $generateHtmlFromNodes, $generateNodesFromDOM } from '@lexical/html'
import { $getRoot, $getSelection, COMMAND_PRIORITY_CRITICAL, PASTE_COMMAND, COMMAND_PRIORITY_HIGH } from 'lexical'
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
import { ImageNode, $createImageNode } from './ImageNode'

import DOMPurify from 'dompurify'
import { htmlAPI, assetsAPI, configAPI } from '@/lib/api'
import { parseGmailAttachmentUrl, gmailClient } from '@/lib/gmailAPI'
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
  onProcessAndCopy?: () => void
  transforming?: boolean
  copied?: boolean
  hasContent?: boolean
  hasGmailAccess?: boolean
  onRequestGmailAccess?: () => void
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

// Unified Image Processor Plugin - handles ALL image nodes consistently
function ImageProcessorPlugin() {
  const [editor] = useLexicalComposerContext()
  const [cdnBaseUrl, setCdnBaseUrl] = useState<string | null>(null)
  
  // Load CDN config on mount
  useEffect(() => {
    configAPI.getConfig()
      .then(config => {
        setCdnBaseUrl(config.cdnBaseUrl)
        console.log('📡 CDN Base URL loaded:', config.cdnBaseUrl)
      })
      .catch(error => {
        console.error('❌ Failed to load CDN config:', error)
      })
  }, [])
  
  // Helper to check if URL is from our CDN
  const isFromCDN = (url: string): boolean => {
    if (!cdnBaseUrl) return false
    try {
      const imageUrl = new URL(url)
      const cdnUrl = new URL(cdnBaseUrl)
      return imageUrl.hostname === cdnUrl.hostname
    } catch {
      return false
    }
  }
  
  // Process a single image node - upload if not from CDN
  const processImageNode = async (imageNode: ImageNode) => {
    const src = imageNode.getSrc()
    
    // Skip if already from CDN, blob, or data URI
    if (isFromCDN(src) || src.startsWith('blob:') || src.startsWith('data:')) {
      return
    }
    
    console.log('⬆️ Processing external image:', src)
    
    try {
      // Check if this is a Gmail attachment URL
      const gmailAttachmentInfo = parseGmailAttachmentUrl(src)
      
      if (gmailAttachmentInfo) {
        console.log('📧 Processing Gmail attachment via Gmail API')
        
        // Use Gmail API to fetch the attachment
        const blob = await gmailClient.fetchAttachment(gmailAttachmentInfo)
        
        if (blob) {
          console.log('✅ Gmail attachment fetched successfully')
          
          // Upload the blob via our file upload API
          const file = new File([blob], 'gmail-attachment.jpg', { type: blob.type || 'image/jpeg' })
          const asset = await assetsAPI.uploadFile(file)
          console.log('✅ Gmail attachment processed to CDN:', asset.url)
          
          // Replace the image node with CDN version
          editor.update(() => {
            const newImageNode = $createImageNode({
              src: asset.url,
              altText: imageNode.getAltText(),
              width: asset.width,
              height: asset.height,
            })
            
            imageNode.replace(newImageNode)
            console.log('🔄 Gmail image node replaced with CDN version')
          })
        } else {
          console.error('❌ Failed to fetch Gmail attachment')
        }
        
      } else {
        // Regular external image - use URL upload
        const asset = await assetsAPI.uploadFromURL(src)
        console.log('✅ Image processed successfully:', asset.url)
        
        // Replace the image node with CDN version
        editor.update(() => {
          const newImageNode = $createImageNode({
            src: asset.url,
            altText: imageNode.getAltText(),
            width: asset.width,
            height: asset.height,
          })
          
          imageNode.replace(newImageNode)
          console.log('🔄 Image node replaced with CDN version')
        })
      }
      
    } catch (error) {
      console.error('❌ Failed to process image:', src, error)
    }
  }
  
  // Register node transform to catch ALL image nodes
  useEffect(() => {
    if (!editor || !cdnBaseUrl) return
    
    console.log('✅ Registering image node transform')
    
    const unregister = editor.registerNodeTransform(ImageNode, (node: ImageNode) => {
      // Process this image node (async, but that's ok)
      processImageNode(node)
    })
    
    return unregister
  }, [editor, cdnBaseUrl])
  
  return null
}

export function Editor({ onContentChange, onProcessAndCopy, transforming, copied, hasContent, hasGmailAccess, onRequestGmailAccess, initialContent }: EditorProps) {
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
    <div className="lexical-editor">
      <LexicalComposer initialConfig={initialConfig}>
        {/* Floating Toolbar - Bottom Center */}
        <EditorToolbar 
          onProcessAndCopy={onProcessAndCopy}
          transforming={transforming}
          copied={copied}
          hasContent={hasContent}
          hasGmailAccess={hasGmailAccess}
          onRequestGmailAccess={onRequestGmailAccess}
        />
        
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
            <ImageProcessorPlugin />
          </div>
        </div>
      </LexicalComposer>
    </div>
  )
}

export default Editor
