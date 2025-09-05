// Client-side Gmail API integration to automatically fetch attachment images

interface GmailAttachmentInfo {
  messageId: string
  attachmentId: string
  threadId?: string
  originalUrl: string
}

interface GmailTokens {
  access_token: string
  refresh_token?: string
  expires_at?: number
}

// Client-side Gmail API class
class GmailAPIClient {
  private tokens: GmailTokens | null = null

  constructor() {
    this.loadTokens()
  }

  private loadTokens(): void {
    if (typeof window === 'undefined') return // Skip during SSR
    
    try {
      const stored = localStorage.getItem('gmail_tokens')
      if (stored) {
        this.tokens = JSON.parse(stored)
      }
    } catch (error) {
      console.error('Failed to load Gmail tokens:', error)
      this.tokens = null
    }
  }

  private saveTokens(tokens: GmailTokens): void {
    if (typeof window === 'undefined') return // Skip during SSR
    
    try {
      localStorage.setItem('gmail_tokens', JSON.stringify(tokens))
      this.tokens = tokens
    } catch (error) {
      console.error('Failed to save Gmail tokens:', error)
    }
  }

  setTokens(tokens: GmailTokens): void {
    this.saveTokens(tokens)
  }

  hasValidTokens(): boolean {
    if (!this.tokens?.access_token) return false
    
    // Check if token is expired (if we have expiry info)
    if (this.tokens.expires_at) {
      return Date.now() < this.tokens.expires_at
    }
    
    return true // Assume valid if no expiry
  }

  async getValidAccessToken(): Promise<string | null> {
    if (!this.hasValidTokens()) {
      console.log('No valid Gmail tokens available')
      return null
    }
    return this.tokens!.access_token
  }

  async fetchAttachment(info: GmailAttachmentInfo): Promise<Blob | null> {
    try {
      const accessToken = await this.getValidAccessToken()
      if (!accessToken) {
        throw new Error('No valid access token')
      }

      console.log('📧 Fetching Gmail attachment via API:', info.messageId, info.attachmentId)

      // Gmail API always requires "msg-f:" prefix for message IDs
      const apiMessageId = `msg-f:${info.messageId}`
      console.log('📧 Using message ID format:', apiMessageId)

      // Get the message
      const messageResponse = await fetch(
        `https://www.googleapis.com/gmail/v1/users/me/messages/${encodeURIComponent(apiMessageId)}`,
        {
          headers: {
            'Authorization': `Bearer ${accessToken}`,
          }
        }
      )

      console.log(`📧 Response: ${messageResponse.status}`)

      if (!messageResponse.ok) {
        if (messageResponse.status === 403) {
          throw new Error(`Gmail API access denied (403) - please sign out and sign in again to grant Gmail permissions`)
        }
        const errorText = await messageResponse.text()
        console.log(`❌ Gmail API error:`, messageResponse.status, errorText.substring(0, 200))
        throw new Error(`Failed to fetch message: ${messageResponse.status}`)
      }

      const message = await messageResponse.json()
      console.log('✅ Found message with API format')
      const successfulMessageId = apiMessageId
      
      console.log('📧 Message structure - payload keys:', Object.keys(message.payload))
      console.log('📧 Looking for attachment ID:', info.attachmentId)
      
      // Debug: log the message parts structure
      if (message.payload.parts) {
        console.log('📧 Message has', message.payload.parts.length, 'parts')
        message.payload.parts.forEach((part: any, index: number) => {
          console.log(`📧 Part ${index}:`, {
            mimeType: part.mimeType,
            filename: part.filename,
            hasBody: !!part.body,
            attachmentId: part.body?.attachmentId,
            hasAttachmentId: !!part.body?.attachmentId
          })
        })
      }
      
      // Find the attachment in the message parts using the correct attachmentId
      let attachmentBodyId = this.findAttachmentBodyId(message.payload, info.attachmentId)
      if (!attachmentBodyId) {
        // As a fallback, try to find any image if the specific ID fails (sometimes realattid is weird)
        const fallbackBodyId = this.findImageAttachmentBodyId(message.payload)
        if (!fallbackBodyId) {
          throw new Error(`Attachment with ID ${info.attachmentId} not found in message`)
        }
        console.log(`⚠️ Could not find specific attachment ID, falling back to first image: ${fallbackBodyId}`)
        attachmentBodyId = fallbackBodyId
      } else {
        console.log('✅ Found specific attachment with body ID:', attachmentBodyId)
      }

      // Fetch the actual attachment data using the successful message ID
      const attachmentResponse = await fetch(
        `https://www.googleapis.com/gmail/v1/users/me/messages/${encodeURIComponent(successfulMessageId)}/attachments/${attachmentBodyId}`,
        {
          headers: {
            'Authorization': `Bearer ${accessToken}`,
          }
        }
      )

      if (!attachmentResponse.ok) {
        throw new Error(`Failed to fetch attachment: ${attachmentResponse.status}`)
      }

      const attachmentData = await attachmentResponse.json()
      
      // Decode base64url data
      const binaryString = atob(attachmentData.data.replace(/-/g, '+').replace(/_/g, '/'))
      const bytes = new Uint8Array(binaryString.length)
      for (let i = 0; i < binaryString.length; i++) {
        bytes[i] = binaryString.charCodeAt(i)
      }
      
      return new Blob([bytes])
    } catch (error) {
      console.error('Gmail API fetch failed:', error)
      return null
    }
  }

  private findImageAttachmentBodyId(part: any): string | null {
    // Check if this part is an image attachment
    if (part.body?.attachmentId && part.mimeType?.startsWith('image/')) {
      console.log('🖼️ Found image attachment:', part.mimeType, part.filename)
      return part.body.attachmentId
    }

    // Search in nested parts
    if (part.parts) {
      for (const subPart of part.parts) {
        const found = this.findImageAttachmentBodyId(subPart)
        if (found) return found
      }
    }

    return null
  }

  private findAttachmentBodyId(part: any, target: string): string | null {
    // Match the UI's realattid=ii_... against the header
    const xAtt = part.headers?.find(
      (h: any) => h.name?.toLowerCase() === 'x-attachment-id'
    )?.value;
    if (xAtt && xAtt === target && part.body?.attachmentId) {
      return part.body.attachmentId;
    }
    // Also allow a direct match to body.attachmentId
    if (part.body?.attachmentId === target) {
      return part.body.attachmentId;
    }
    if (part.parts) {
      for (const sub of part.parts) {
        const found = this.findAttachmentBodyId(sub, target);
        if (found) return found;
      }
    }
    return null;
  }

  clearTokens(): void {
    if (typeof window === 'undefined') return // Skip during SSR
    
    localStorage.removeItem('gmail_tokens')
    this.tokens = null
  }
}

// Global instance
export const gmailClient = new GmailAPIClient()

export function parseGmailAttachmentUrl(url: string): GmailAttachmentInfo | null {
  try {
    // Decode HTML entities first
    const decodedUrl = url
      .replace(/&amp;/g, '&')
      .replace(/&lt;/g, '<')
      .replace(/&gt;/g, '>')
      .replace(/&quot;/g, '"')
    
    console.log('📧 Parsing Gmail URL:', decodedUrl.substring(0, 120) + '...')
    
    const urlObj = new URL(decodedUrl)
    
    // Extract key parameters from Gmail attachment URL
    const attid = urlObj.searchParams.get('attid')
    const permmsgid = urlObj.searchParams.get('permmsgid')
    const th = urlObj.searchParams.get('th')
    const realattid = urlObj.searchParams.get('realattid')
    
    console.log('📧 URL params:', { attid, permmsgid, th, realattid })
    
    if (!attid && !realattid) {
      console.log('❌ No attachment ID found')
      return null
    }
    if (!permmsgid) {
      console.log('❌ No message ID found')
      return null
    }
    
    // Convert permmsgid format (msg-f:1842260674151743696) to just the ID
    const messageId = permmsgid.replace('msg-f:', '').replace('msg-a:', '')
    
    const result = {
      messageId,
      attachmentId: realattid || attid || '',
      threadId: th || undefined,
      originalUrl: url
    }
    
    console.log('✅ Parsed Gmail attachment info:', result)
    return result
  } catch (error) {
    console.error('Failed to parse Gmail URL:', error)
    return null
  }
}

export async function convertGmailAttachmentsToDataUris(html: string): Promise<{ html: string; processed: number; failed: number }> {
  // Find all Gmail attachment URLs in the HTML
  const gmailImageRegex = /<img[^>]*src="(https:\/\/mail\.google\.com\/[^"]*)"[^>]*>/g
  const matches: RegExpExecArray[] = []
  let match
  while ((match = gmailImageRegex.exec(html)) !== null) {
    matches.push(match)
  }
  
  console.log(`📧 Found ${matches.length} Gmail attachment images`)
  
  let processedHtml = html
  let processed = 0
  let failed = 0
  
  for (const match of matches) {
    const [fullImgTag, gmailUrl] = match
    
    const attachmentInfo = parseGmailAttachmentUrl(gmailUrl)
    if (!attachmentInfo) {
      console.error('❌ Could not parse Gmail URL:', gmailUrl)
      failed++
      continue
    }
    
    try {
      console.log('📥 Downloading Gmail attachment...')
      const blob = await gmailClient.fetchAttachment(attachmentInfo)
      
      if (blob) {
        console.log('📧 Gmail attachment blob type:', blob.type)
        
        // Convert blob to data URI with correct MIME type
        const dataUri = await new Promise<string>((resolve) => {
          const reader = new FileReader()
          reader.onload = () => {
            let result = reader.result as string
            
            // Fix MIME type if it's generic octet-stream but we know it's an image
            if (result.startsWith('data:application/octet-stream')) {
              // Try to detect actual image type from magic bytes
              const base64Data = result.split(',')[1]
              const binaryData = atob(base64Data.substring(0, 32)) // Read more bytes for detection
              
              console.log('🔬 Magic bytes detection, first 8 bytes:', 
                Array.from(binaryData.substring(0, 8), c => c.charCodeAt(0).toString(16).padStart(2, '0')).join(' '))
              
              let actualMimeType = 'image/jpeg' // Default fallback
              
              // PNG magic bytes: 89 50 4E 47 0D 0A 1A 0A
              if (binaryData.charCodeAt(0) === 0x89 && 
                  binaryData.charCodeAt(1) === 0x50 && 
                  binaryData.charCodeAt(2) === 0x4E && 
                  binaryData.charCodeAt(3) === 0x47) {
                actualMimeType = 'image/png'
                console.log('🖼️ Detected PNG via magic bytes')
              }
              // JPEG magic bytes: FF D8 FF
              else if (binaryData.charCodeAt(0) === 0xFF && 
                       binaryData.charCodeAt(1) === 0xD8 && 
                       binaryData.charCodeAt(2) === 0xFF) {
                actualMimeType = 'image/jpeg'
                console.log('🖼️ Detected JPEG via magic bytes')
              }
              
              result = `data:${actualMimeType};base64,${base64Data}`
              console.log('🔧 Fixed MIME type from octet-stream to:', actualMimeType)
            }
            
            resolve(result)
          }
          reader.readAsDataURL(blob)
        })
        
        console.log('📏 Image details:', {
          originalSize: blob.size,
          dataUriLength: dataUri.length,
          sizeMB: (blob.size / (1024 * 1024)).toFixed(1) + 'MB',
          mimeType: dataUri.split(';')[0].replace('data:', '')
        })
        
        // Replace the Gmail URL with data URI in HTML
        processedHtml = processedHtml.replace(gmailUrl, dataUri)
        processed++
        console.log('✅ Converted Gmail attachment to data URI')
      } else {
        failed++
      }
    } catch (error) {
      console.error('❌ Failed to process Gmail attachment:', error)
      failed++
    }
  }
  
  return { html: processedHtml, processed, failed }
}

export function hasGmailAttachments(html: string): boolean {
  return html.includes('mail.google.com') && html.includes('attid=')
}
