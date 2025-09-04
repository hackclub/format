'use client'

import { useEffect } from 'react'
import { gmailClient } from '@/lib/gmailAPI'

export function useOAuthTokens() {
  useEffect(() => {
    // Check if we have OAuth tokens in the URL fragment
    const fragment = window.location.hash.substring(1)
    console.log('🔍 Checking URL fragment for tokens:', fragment ? 'Found' : 'Empty')
    
    if (!fragment) return

    const params = new URLSearchParams(fragment)
    const accessToken = params.get('access_token')
    const refreshToken = params.get('refresh_token')
    const expiresIn = params.get('expires_in')

    console.log('📧 OAuth redirect params:', { 
      hasAccessToken: !!accessToken, 
      hasRefreshToken: !!refreshToken, 
      expiresIn 
    })

    if (accessToken) {
      console.log('📧 Received OAuth tokens from redirect')
      console.log('📧 Access token (first 20 chars):', accessToken.substring(0, 20) + '...')
      
      // Calculate expiration time
      let expiresAt: number | undefined
      if (expiresIn) {
        expiresAt = Date.now() + (parseInt(expiresIn) * 1000)
        console.log('📧 Token expires at:', new Date(expiresAt))
      }

      // Store tokens in Gmail client
      gmailClient.setTokens({
        access_token: accessToken,
        refresh_token: refreshToken || undefined,
        expires_at: expiresAt
      })

      // Clean up URL fragment
      window.history.replaceState({}, document.title, window.location.pathname + window.location.search)
      
      console.log('✅ Gmail API tokens stored successfully')
    } else {
      console.log('❌ No access token found in URL fragment')
    }
  }, [])
}
