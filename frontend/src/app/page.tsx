'use client'

import { useState, useCallback } from 'react'
import dynamic from 'next/dynamic'
import { AuthGuard } from '@/components/AuthGuard'
import { LoadingSpinner } from '@/components/LoadingSpinner'
import { useAuth } from '@/hooks/useAuth'
import { htmlAPI } from '@/lib/api'
import { TransformResult } from '@/types'
import { convertGmailAttachmentsToDataUris, hasGmailAttachments } from '@/lib/gmailAPI'
import { useGmailAPI } from '@/hooks/useGmailAPI'
import { useOAuthTokens } from '@/hooks/useOAuthTokens'




// Dynamically import Editor to avoid SSR issues
const Editor = dynamic(() => import('@/components/Editor'), {
  ssr: false,
  loading: () => <div className="h-64 bg-gray-100 animate-pulse rounded-lg" />
})

export default function HomePage() {
  useOAuthTokens() // Capture tokens from OAuth redirect
  const { user, logout } = useAuth()
  const { hasGmailAccess, requestGmailAccess } = useGmailAPI()
  const [content, setContent] = useState('')
  const [transforming, setTransforming] = useState(false)
  const [transformResult, setTransformResult] = useState<TransformResult | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [copied, setCopied] = useState(false)

  const handleContentChange = useCallback((html: string) => {
    setContent(html)
    setError(null)
    if (transformResult) {
      setTransformResult(null) // Clear previous results when content changes
    }
  }, [transformResult])

  const handleProcessAndCopy = async () => {
    if (!content.trim()) {
      setError('No content to process')
      return
    }

    try {
      setTransforming(true)
      setError(null)
      
      // Step 1: Try Gmail API for attachment images if available
      let htmlToProcess = content
      
      if (hasGmailAccess && hasGmailAttachments(content)) {
        console.log('🔍 Gmail access available, processing attachments...')
        try {
          const gmailResult = await convertGmailAttachmentsToDataUris(content)
          htmlToProcess = gmailResult.html
          console.log(`📧 Gmail API: ${gmailResult.processed} processed, ${gmailResult.failed} failed`)
        } catch (gmailError) {
          console.error('Gmail API processing failed:', gmailError)
          
          // Show helpful error for permission issues
          if (gmailError instanceof Error && gmailError.message.includes('403')) {
            setError('Gmail API permissions needed - please sign out and sign in again to enable automatic image processing')
          }
          
          // Continue with original HTML
        }
      } else if (hasGmailAttachments(content) && !hasGmailAccess) {
        console.log('📧 Gmail attachments detected but no API access')
      }
      
      // Step 2: Process and clean HTML
      console.log('Processing HTML for copy:', htmlToProcess.substring(0, 200) + '...')
      const result = await htmlAPI.transform(htmlToProcess)
      console.log('Transform result:', result)
      
      // Ensure the result has the expected structure
      const normalizedResult = {
        html: result.html || content,
        messages: result.messages || [],
        stats: result.stats || { images_processed: 0, images_rehosted: 0, styles_removed: 0, scripts_removed: 0 }
      }
      
      setTransformResult(normalizedResult)
      
      // Step 2: Immediately copy to clipboard
      console.log('=== COPYING PROCESSED HTML ===')
      console.log('Backend HTML length:', normalizedResult.html.length)
      console.log('Backend has <b> tags:', normalizedResult.html.includes('<b>'))
      console.log('Backend has <i> tags:', normalizedResult.html.includes('<i>'))
      
      await navigator.clipboard.write([
        new ClipboardItem({
          'text/html': new Blob([normalizedResult.html], { type: 'text/html' }),
          'text/plain': new Blob([normalizedResult.html.replace(/<[^>]*>/g, '')], { type: 'text/plain' }),
        }),
      ])
      
      // Show success feedback using React state
      setCopied(true)
      setTimeout(() => {
        setCopied(false)
      }, 2000)
    } catch (err) {
      console.error('Process and copy error:', err)
      setError(err instanceof Error ? err.message : 'Failed to process and copy HTML')
    } finally {
      setTransforming(false)
    }
  }

  return (
    <AuthGuard>
      <div className="min-h-screen bg-white relative">
        {/* Floating Sign Out Button */}
        {user && (
          <button
            onClick={logout}
            className="fixed bottom-4 right-4 z-30 bg-white border border-gray-300 text-gray-600 px-3 py-2 rounded-lg shadow-lg hover:bg-gray-50 text-sm"
          >
            Sign out
          </button>
        )}

        {/* Main Content - Full Screen Editor */}
        <main className="h-screen">
          <Editor 
            onContentChange={handleContentChange}
            onProcessAndCopy={handleProcessAndCopy}
            transforming={transforming}
            copied={copied}
            hasContent={!!content.trim()}
            hasGmailAccess={hasGmailAccess}
            onRequestGmailAccess={requestGmailAccess}
            initialContent={content}
          />

          {/* Floating Error Messages */}
          {error && (
            <div className="fixed bottom-20 left-4 z-10 bg-red-50 border border-red-200 rounded-lg p-3 shadow-lg max-w-md">
              <p className="text-sm text-red-700">{error}</p>
            </div>
          )}

          {/* Floating Success Messages */}
          {transformResult?.messages && transformResult.messages.length > 0 && (
            <div className="fixed bottom-20 left-4 z-10 bg-yellow-50 border border-yellow-200 rounded-lg p-3 shadow-lg max-w-md">
              <div className="text-xs text-yellow-800">
                {transformResult.messages.map((message, index) => (
                  <div key={index}>• {message}</div>
                ))}
              </div>
            </div>
          )}
        </main>
      </div>
    </AuthGuard>
  )
}
