'use client'

import { useEffect, useState, useCallback } from 'react'
import { gmailClient } from '@/lib/gmailAPI'
import { testTokenAndFixPermissions } from '@/lib/tokenValidator'

export function useGmailAPI() {
  const [hasGmailAccess, setHasGmailAccess] = useState(false)
  const [loading, setLoading] = useState(true)

  const checkGmailAccess = useCallback(async () => {
    try {
      setLoading(true)
      const hasTokens = gmailClient.hasValidTokens()
      
      console.log('🔍 Gmail token check:', hasTokens ? 'Valid tokens found' : 'No valid tokens')
      
      // Debug: show what tokens we actually have
      const storedTokens = localStorage.getItem('gmail_tokens')
      if (storedTokens) {
        const parsed = JSON.parse(storedTokens)
        console.log('📧 Stored tokens:', {
          hasAccessToken: !!parsed.access_token,
          accessTokenStart: parsed.access_token?.substring(0, 20) + '...',
          hasRefreshToken: !!parsed.refresh_token,
          expiresAt: parsed.expires_at ? new Date(parsed.expires_at) : 'No expiry'
        })
      } else {
        console.log('❌ No tokens in localStorage')
      }
      
      setHasGmailAccess(hasTokens)
      
      if (hasTokens) {
        console.log('✅ Gmail API access available')
        
        // Test if the token actually has Gmail scope
        const token = await gmailClient.getValidAccessToken()
        if (token) {
          await testTokenAndFixPermissions(token)
        }
      } else {
        console.log('❌ No Gmail API access')
      }
    } catch (error) {
      console.error('Failed to check Gmail access:', error)
      setHasGmailAccess(false)
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    checkGmailAccess()
  }, [checkGmailAccess])



  const clearGmailAccess = () => {
    gmailClient.clearTokens()
    setHasGmailAccess(false)
  }

  return {
    hasGmailAccess,
    loading,
    clearGmailAccess,
    checkGmailAccess,
  }
}
