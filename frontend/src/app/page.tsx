'use client'

import { useState, useCallback } from 'react'
import dynamic from 'next/dynamic'
import { AuthGuard } from '@/components/AuthGuard'
import { LoadingSpinner } from '@/components/LoadingSpinner'
import { useAuth } from '@/hooks/useAuth'
import { htmlAPI } from '@/lib/api'
import { TransformResult } from '@/types'

// Dynamically import Editor to avoid SSR issues
const Editor = dynamic(() => import('@/components/Editor'), {
  ssr: false,
  loading: () => <div className="h-64 bg-gray-100 animate-pulse rounded-lg" />
})

export default function HomePage() {
  const { user, logout } = useAuth()
  const [content, setContent] = useState('')
  const [transforming, setTransforming] = useState(false)
  const [transformResult, setTransformResult] = useState<TransformResult | null>(null)
  const [error, setError] = useState<string | null>(null)

  const handleContentChange = useCallback((html: string) => {
    setContent(html)
    setError(null)
    if (transformResult) {
      setTransformResult(null) // Clear previous results when content changes
    }
  }, [transformResult])

  const handleRehostImages = async () => {
    if (!content.trim()) {
      setError('No content to process')
      return
    }

    try {
      setTransforming(true)
      setError(null)
      console.log('Sending HTML to transform:', content.substring(0, 200) + '...')
      const result = await htmlAPI.transform(content)
      console.log('Transform result:', result) // Debug log
      
      // Ensure the result has the expected structure
      const normalizedResult = {
        html: result.html || content,
        messages: result.messages || [],
        stats: result.stats || { images_processed: 0, images_rehosted: 0, styles_removed: 0, scripts_removed: 0 }
      }
      
      setTransformResult(normalizedResult)
    } catch (err) {
      console.error('Transform error:', err) // Debug log
      setError(err instanceof Error ? err.message : 'Failed to transform HTML')
    } finally {
      setTransforming(false)
    }
  }

  const handleCopy = async () => {
    if (!transformResult) return

    try {
      console.log('=== COPY DEBUG ===')
      console.log('Backend HTML length:', transformResult.html.length)
      console.log('Backend HTML first 200 chars:', transformResult.html.substring(0, 200))
      console.log('Backend has <b> tags:', transformResult.html.includes('<b>'))
      console.log('Backend has <i> tags:', transformResult.html.includes('<i>'))
      
      // Use the backend-transformed HTML directly, not what's in the editor
      await navigator.clipboard.write([
        new ClipboardItem({
          'text/html': new Blob([transformResult.html], { type: 'text/html' }),
          'text/plain': new Blob([transformResult.html.replace(/<[^>]*>/g, '')], { type: 'text/plain' }),
        }),
      ])
      
      // Show success feedback
      const button = document.querySelector('[data-copy-button]') as HTMLElement
      const originalText = button?.textContent
      if (button && originalText) {
        button.textContent = 'Copied!'
        setTimeout(() => {
          button.textContent = originalText
        }, 2000)
      }
    } catch (err) {
      setError('Failed to copy to clipboard')
    }
  }

  return (
    <AuthGuard>
      <div className="min-h-screen bg-gray-50">
        {/* Header */}
        <header className="bg-white shadow-sm">
          <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
            <div className="flex justify-between items-center py-6">
              <div>
                <h1 className="text-2xl font-bold text-gray-900">Format</h1>
                <p className="text-sm text-gray-600">Clean HTML for Gmail</p>
              </div>
              
              <div className="flex items-center space-x-4">
                {user && (
                  <>
                    <div className="flex items-center space-x-2">
                      {/* eslint-disable-next-line @next/next/no-img-element */}
                      <img 
                        src={user.picture} 
                        alt={user.name}
                        className="w-8 h-8 rounded-full"
                      />
                      <span className="text-sm text-gray-700">{user.name}</span>
                    </div>
                    <button
                      onClick={logout}
                      className="text-sm text-gray-500 hover:text-gray-700"
                    >
                      Sign out
                    </button>
                  </>
                )}
              </div>
            </div>
          </div>
        </header>

        {/* Main Content */}
        <main className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-8">
            {/* Input Section */}
            <div className="space-y-4">
              <div>
                <h2 className="text-lg font-semibold text-gray-900 mb-2">
                  Paste your content
                </h2>
                <p className="text-sm text-gray-600 mb-4">
                  Paste rich text from Google Docs, Notion, or any other source. Images will be automatically detected.
                </p>
              </div>
              
              <Editor 
                onContentChange={handleContentChange}
                initialContent={content}
              />
              
              <div className="flex space-x-3">
                <button
                  onClick={handleRehostImages}
                  disabled={transforming || !content.trim()}
                  className="flex-1 bg-hack-red text-white px-4 py-2 rounded-lg hover:bg-red-600 disabled:bg-gray-400 disabled:cursor-not-allowed transition-colors font-medium"
                >
                  {transforming ? (
                    <div className="flex items-center justify-center space-x-2">
                      <LoadingSpinner size="sm" />
                      <span>Processing...</span>
                    </div>
                  ) : (
                    'Rehost Images & Clean HTML'
                  )}
                </button>
              </div>
            </div>

            {/* Output Section */}
            <div className="space-y-4">
              <div>
                <h2 className="text-lg font-semibold text-gray-900 mb-2">
                  Gmail-safe result
                </h2>
                <p className="text-sm text-gray-600 mb-4">
                  Copy this cleaned HTML and paste it into Gmail or any email client.
                </p>
              </div>
              
              {error && (
                <div className="bg-red-50 border border-red-200 rounded-lg p-4">
                  <p className="text-sm text-red-700">{error}</p>
                </div>
              )}
              
              {transformResult ? (
                <div className="space-y-4">
                  {/* Stats */}
                  <div className="bg-green-50 border border-green-200 rounded-lg p-4">
                    <div className="grid grid-cols-2 gap-4 text-sm">
                      <div>
                        <span className="font-medium">Images processed:</span>
                        <span className="ml-2">{transformResult.stats?.images_processed || 0}</span>
                      </div>
                      <div>
                        <span className="font-medium">Images rehosted:</span>
                        <span className="ml-2">{transformResult.stats?.images_rehosted || 0}</span>
                      </div>
                      <div>
                        <span className="font-medium">Styles removed:</span>
                        <span className="ml-2">{transformResult.stats?.styles_removed || 0}</span>
                      </div>
                      <div>
                        <span className="font-medium">Scripts removed:</span>
                        <span className="ml-2">{transformResult.stats?.scripts_removed || 0}</span>
                      </div>
                    </div>
                  </div>

                  {/* Messages */}
                  {transformResult.messages && transformResult.messages.length > 0 && (
                    <div className="bg-blue-50 border border-blue-200 rounded-lg p-4">
                      <h3 className="font-medium text-blue-900 mb-2">Processing details:</h3>
                      <ul className="text-sm text-blue-800 space-y-1">
                        {transformResult.messages.map((message, index) => (
                          <li key={index}>• {message}</li>
                        ))}
                      </ul>
                    </div>
                  )}

                  {/* Preview */}
                  <div className="border rounded-lg">
                    <div className="bg-gray-50 px-4 py-2 border-b">
                      <h3 className="font-medium text-gray-900">Preview</h3>
                    </div>
                    <div 
                      className="p-4 prose max-w-none"
                      dangerouslySetInnerHTML={{ __html: transformResult.html }}
                    />
                  </div>

                  {/* Copy Button */}
                  <button
                    onClick={handleCopy}
                    data-copy-button
                    className="w-full bg-hack-green text-white px-4 py-3 rounded-lg hover:bg-green-600 transition-colors font-medium"
                  >
                    Copy to Clipboard
                  </button>
                  
                  <p className="text-xs text-gray-500 text-center">
                    The HTML is copied to your clipboard. Paste it into Gmail&apos;s compose window.
                  </p>
                </div>
              ) : (
                <div className="h-64 bg-gray-100 rounded-lg flex items-center justify-center">
                  <p className="text-gray-500">Process your content to see the result here</p>
                </div>
              )}
            </div>
          </div>
        </main>
      </div>
    </AuthGuard>
  )
}
