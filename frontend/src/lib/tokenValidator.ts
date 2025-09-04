// Test if access tokens have the correct Gmail scope

export async function validateGmailScope(accessToken: string): Promise<boolean> {
  try {
    console.log('🔍 Validating Gmail scope for token...')
    
    // Try to access Gmail API with a simple call
    const response = await fetch('https://www.googleapis.com/gmail/v1/users/me/profile', {
      headers: {
        'Authorization': `Bearer ${accessToken}`
      }
    })
    
    console.log('📧 Gmail profile API response:', response.status)
    
    if (response.ok) {
      const profile = await response.json()
      console.log('✅ Gmail API access confirmed, email:', profile.emailAddress)
      return true
    } else if (response.status === 403) {
      console.log('❌ Gmail API access denied - token lacks gmail.readonly scope')
      return false
    } else {
      console.log('⚠️  Gmail API returned:', response.status)
      return false
    }
  } catch (error) {
    console.error('Gmail scope validation failed:', error)
    return false
  }
}

export async function testTokenAndFixPermissions(accessToken: string): Promise<void> {
  const hasGmailScope = await validateGmailScope(accessToken)
  
  if (!hasGmailScope) {
    console.log('🔄 Token lacks Gmail scope - clearing tokens and forcing re-auth')
    
    // Clear invalid tokens immediately
    localStorage.removeItem('gmail_tokens')
    
    // Force immediate re-authentication with Gmail scope  
    console.log('🔄 Redirecting to OAuth with Gmail scope...')
    window.location.href = '/api/auth/login'
  }
}
