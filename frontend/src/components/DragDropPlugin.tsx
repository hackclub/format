'use client'

import { useEffect } from 'react'
import { useLexicalComposerContext } from '@lexical/react/LexicalComposerContext'
import { $insertNodes, $getSelection } from 'lexical'
import { $createImageNode } from './ImageNode'
import { assetsAPI } from '@/lib/api'

export function DragDropPlugin() {
  const [editor] = useLexicalComposerContext()

  useEffect(() => {
    const handleDragOver = (event: DragEvent) => {
      event.preventDefault()
      event.dataTransfer!.dropEffect = 'copy'
    }

    const handleDragEnter = (event: DragEvent) => {
      event.preventDefault()
      const target = event.target as HTMLElement
      const editorInput = target.closest('.editor-input')
      if (editorInput) {
        editorInput.classList.add('drag-over')
      }
    }

    const handleDragLeave = (event: DragEvent) => {
      event.preventDefault()
      const target = event.target as HTMLElement
      const editorInput = target.closest('.editor-input')
      if (editorInput && !editorInput.contains(event.relatedTarget as Node)) {
        editorInput.classList.remove('drag-over')
      }
    }

    const handleDrop = async (event: DragEvent) => {
      event.preventDefault()
      
      const target = event.target as HTMLElement
      const editorInput = target.closest('.editor-input')
      if (editorInput) {
        editorInput.classList.remove('drag-over')
      }
      
      const files = Array.from(event.dataTransfer?.files || [])
      const imageFiles = files.filter(file => file.type.startsWith('image/'))
      
      if (imageFiles.length === 0) {
        return
      }

      // Process each image file
      for (const file of imageFiles) {
        try {
          console.log('Uploading dropped image:', file.name)
          
          // Upload to backend
          const asset = await assetsAPI.uploadFile(file)
          console.log('Image uploaded successfully:', asset)

          // Insert image node into editor
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
          // Could show a toast notification here
        }
      }
    }

    // Add event listeners to the editor container
    const editorContainer = document.querySelector('.lexical-editor .editor-input')
    if (editorContainer) {
      editorContainer.addEventListener('dragover', handleDragOver)
      editorContainer.addEventListener('dragenter', handleDragEnter)
      editorContainer.addEventListener('dragleave', handleDragLeave)
      editorContainer.addEventListener('drop', handleDrop)

      return () => {
        editorContainer.removeEventListener('dragover', handleDragOver)
        editorContainer.removeEventListener('dragenter', handleDragEnter)
        editorContainer.removeEventListener('dragleave', handleDragLeave)
        editorContainer.removeEventListener('drop', handleDrop)
      }
    }
  }, [editor])

  return null
}
