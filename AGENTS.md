# Format.hackclub.com - Agent Instructions

This file contains comprehensive instructions for AI agents working on this project.

## Overview

**format.hackclub.com** is a paste-clean-copy tool that sanitizes HTML content and automatically rehoists images to Hack Club's CDN. It features Gmail API integration for automatic attachment processing and produces Gmail-compatible HTML output.

## How to Run the Project

### Prerequisites Setup

**Install Required Dependencies:**
```bash
# macOS
brew install vips jpeg-xl oxipng      # Image processing libraries
brew install jphastings/tools/jpegli  # State-of-the-art JPEG encoder
go install github.com/air-verse/air@latest  # Go hot reload

# Verify installations
pkg-config --exists vips && echo "✅ vips installed" || echo "❌ vips missing"
command -v cjpegli >/dev/null && echo "✅ jpegli installed" || echo "❌ jpegli missing"
command -v oxipng >/dev/null && echo "✅ oxipng installed" || echo "❌ oxipng missing"
air -v                               # Should show Air version
```

**Configure Environment:**
```bash
# 1. Copy environment template
cp .env.example .env

# 2. Edit .env with your credentials:
#    - Google OAuth Client ID/Secret (with Gmail API enabled)
#    - Cloudflare R2 credentials (account, keys, bucket, endpoint)
#    - Session secret (any random string)
#    - Domain restrictions (hackclub.com by default)
```

### Development Workflow

**Quick Start (Recommended):**
```bash
# Install dependencies + setup
make setup

# Start both servers with hot reload
make dev
```

**Manual Start:**
```bash
# Backend with hot reload (recommended)
cd backend && $(go env GOPATH)/bin/air

# Frontend 
cd frontend && npm run dev

# Or without hot reload
cd backend && go run cmd/server/main.go
```

**Access the Application:**
- **Frontend**: http://localhost:3000 (Next.js)
- **Backend**: http://localhost:8080 (Go API)
- **Health Check**: http://localhost:8080/healthz

### First-Time OAuth Setup

**Google Cloud Console Setup:**
1. **Create project** or select existing
2. **Enable APIs**: Google+ API and Gmail API
3. **Create OAuth 2.0 credentials**:
   - Application type: Web application
   - Authorized redirect URIs: `http://localhost:3000/api/auth/callback`
4. **Copy Client ID + Secret** to `.env`

**Cloudflare R2 Setup:**
1. **Create R2 bucket** in Cloudflare dashboard
2. **Generate R2 API tokens** with read/write permissions
3. **Optional**: Set up custom domain for CDN
4. **Add credentials** to `.env`

### Testing the Complete Workflow

**Authentication Test:**
1. Visit http://localhost:3000
2. Click "Sign in with Google"
3. Grant permissions (including Gmail access)
4. Should see full-screen editor

**Image Processing Test:**
```bash
# Test R2 connection
cd backend && go run test-r2.go

# Expected: All R2 tests pass
```

**Gmail API Test:**
1. **Paste Gmail content** with attachment images
2. **Check console**: Should see "✅ Gmail API access confirmed"
3. **Click 📋 Copy**: Should process attachments automatically
4. **Verify rehosting**: Images should use `format.hackclub-assets.com` URLs

### Development Commands

### Backend (Go) with Hot Reload
```bash
cd backend
$(go env GOPATH)/bin/air            # Start development server with hot reload (recommended)
go run cmd/server/main.go           # Start development server (no hot reload)
go build -o bin/server cmd/server/main.go  # Build binary
go test ./...                       # Run tests
go mod tidy                         # Clean up dependencies
```

### Frontend (Next.js)
```bash
cd frontend
npm run dev                         # Start development server
npm run build                       # Build for production
npm run type-check                  # Run TypeScript type checking
npm run lint                        # Run ESLint
```

### Combined Development
```bash
make dev                           # Start both backend and frontend with hot reload
make build                         # Build both components
make test                          # Run all tests
make check                         # Run type checking and linting
```

## Architecture Overview

```
Full-Screen Lexical Editor (Next.js)
        ↓ (paste rich content)
Gmail API Integration (client-side)
        ↓ (auto-fetch attachments)
Go Backend (chi + libvips + R2)
        ↓ (process & rehost images)
Cloudflare R2 + CDN
        ↓ (serve optimized images)
Gmail-Compatible HTML Output
```

## Core Technologies

- **Backend**: Go 1.22+, chi router, bimg (libvips), Air hot reload
- **Frontend**: Next.js 14, React 18, Lexical editor, TailwindCSS
- **Authentication**: Google OAuth with Gmail API scope (client-side token storage)
- **Image Processing**: libvips with intelligent format conversion (JPEG/PNG)
- **Storage**: Cloudflare R2 with CDN and deduplication
- **UI**: Full-screen editor with floating controls and rich text toolbar

## Environment Setup

### Required Dependencies
```bash
# macOS
brew install vips jpeg-xl oxipng
go install github.com/air-verse/air@latest

# Ubuntu/Debian  
sudo apt-get update
sudo apt-get install libvips-dev libjxl-tools oxipng
```

### Environment Variables
All stored in `.env` (never committed):
```bash
# Core
SESSION_SECRET=your-session-secret
APP_BASE_URL=http://localhost:3000

# Google OAuth (requires Gmail API enabled in GCP)
GOOGLE_OAUTH_CLIENT_ID=your-client-id
GOOGLE_OAUTH_CLIENT_SECRET=your-client-secret
ALLOWED_DOMAINS=hackclub.com,gmail.com  # Comma-separated

# Image Processing (hard-coded: 3840px max, 5MB triggers resize)
JPEG_QUALITY=84
JPEG_PROGRESSIVE=true
PNG_STRIP=true

# Cloudflare R2 Storage
R2_ACCOUNT_ID=your-account-id
R2_ACCESS_KEY_ID=your-access-key
R2_SECRET_ACCESS_KEY=your-secret
R2_BUCKET=your-bucket-name
R2_PUBLIC_BASE_URL=https://your-cdn-domain.com
R2_S3_ENDPOINT=https://account-id.r2.cloudflarestorage.com
```

## Backend Architecture (Go)

### Directory Structure
```
backend/
├── cmd/server/main.go              # Application entry point
├── internal/
│   ├── auth/oidc.go               # Google OAuth + Gmail scope
│   ├── assets/                    # Image processing service
│   │   ├── service.go             # Core image pipeline orchestrator
│   │   └── handler.go             # HTTP handlers for uploads
│   ├── config/config.go           # Environment configuration
│   ├── gmail/client.go            # Gmail API client (unused - client-side instead)
│   ├── html/transform.go          # Gmail-compatible HTML transformation
│   ├── http/router.go             # Chi router + middleware + handlers
│   ├── imageproc/                 # libvips image processing
│   │   ├── vips.go               # Main processor with format conversion
│   │   └── simple.go             # Fallback processor (unused)
│   ├── session/cookie.go          # Session management
│   ├── storage/                   # Cloudflare R2 integration
│   │   ├── r2.go                 # Real R2 client with S3 API
│   │   ├── mock.go               # Mock client (unused)
│   │   └── interface.go          # Storage interface (unused)
│   └── util/                      # Utilities
│       ├── hash.go               # SHA-256 hashing + Base32 keys
│       ├── mime.go               # MIME detection + format decisions
│       └── httpfetch.go          # SSRF-safe HTTP fetching
└── .air.toml                     # Air hot reload configuration
```

### Key Backend Endpoints

```
GET  /healthz                     # Health check
GET  /api/auth/login              # OAuth login (includes Gmail scope)
GET  /api/auth/callback           # OAuth callback (returns tokens in URL fragment)
POST /api/auth/logout             # Clear session
GET  /api/auth/me                 # Get current user

POST /api/assets                  # Upload single image (file/URL/data URI)
POST /api/assets/batch            # Upload multiple images
GET  /api/assets/{id}             # Get asset metadata

POST /api/html/transform          # Transform HTML to Gmail format + rehost images
```

### Image Processing Pipeline

**Resize Triggers** (hard-coded):
- Width > 3840px OR Height > 3840px OR File size > 5MB

**Format Conversion Logic**:
- **JPEG → JPEG**: Stay as JPEG with compression
- **PNG with transparency → PNG**: Preserve alpha channel
- **PNG without transparency → JPEG**: Convert for better compression
- **Other formats → JPEG**: Default conversion

**Processing Flow**:
1. Decode with libvips → sRGB color space
2. Resize if needed (maintain aspect, never upscale)
3. Format decision based on transparency + input type
4. Encode with quality 84, progressive JPEG, strip metadata
5. Hash SHA-256 of final bytes → Base32 key with 2-char sharding
6. Check R2 for existing file → Upload if new
7. Return CDN URL

## Frontend Architecture (Next.js)

### Directory Structure
```
frontend/src/
├── app/
│   ├── layout.tsx                 # Root layout
│   ├── page.tsx                   # Main full-screen editor page
│   └── globals.css               # Global styles + Lexical theme
├── components/
│   ├── Editor.tsx                # Lexical rich text editor wrapper
│   ├── EditorToolbar.tsx         # Floating formatting toolbar (bottom center)
│   ├── ImageNode.tsx             # Custom Lexical image node
│   ├── DragDropPlugin.tsx        # Drag & drop image upload
│   ├── KeyboardShortcutsPlugin.tsx # Cmd+B/I/U/K shortcuts
│   ├── AuthGuard.tsx             # Authentication wrapper
│   └── LoadingSpinner.tsx        # Reusable spinner
├── hooks/
│   ├── useAuth.ts                # Authentication state management
│   ├── useGmailAPI.ts            # Gmail API access checking
│   └── useOAuthTokens.ts         # OAuth token capture from URL
├── lib/
│   ├── api.ts                    # Backend API client
│   ├── gmailAPI.ts               # Client-side Gmail API integration
│   └── tokenValidator.ts        # Gmail scope validation
└── types/index.ts                # TypeScript type definitions
```

### Key Frontend Features

**Full-Screen Editor**:
- Lexical rich text editor with Gmail-compatible theme
- 800px max-width centered content area for optimal readability
- Custom ImageNode for proper image handling
- Paste preprocessing for Gmail signature formatting

**Rich Text Toolbar** (floating bottom center):
- **B** I **U** S̶ | • 1. | 🔗 🖼️ | **📋 Copy**
- Keyboard shortcuts: Cmd+B/I/U (format), Cmd+K (link)
- Image upload button + drag & drop support
- Integrated copy button that processes and copies in one click

**Gmail API Integration** (client-side):
- OAuth tokens stored in localStorage (never server-side)
- Automatic Gmail attachment detection and fetching
- Magic bytes MIME type detection for proper processing
- Fallback to manual upload if API access unavailable

## Authentication Flow

### Google OAuth Setup Required
1. **Google Cloud Console**: Enable Gmail API for your project
2. **OAuth 2.0 Client**: Configure with redirect URI `http://localhost:3000/api/auth/callback`
3. **Scopes**: `openid`, `profile`, `email`, `https://www.googleapis.com/auth/gmail.readonly`

### Authentication Process
1. User clicks login → `/api/auth/login` 
2. Redirects to Google OAuth with Gmail scope
3. User grants permissions (including Gmail access)
4. Callback → `/api/auth/callback` sets session + returns tokens in URL fragment
5. Frontend captures tokens → stores in localStorage
6. Session cookie enables API access, tokens enable Gmail API

### Domain Restrictions
- Only users from `ALLOWED_DOMAINS` can sign in
- Verified via Google Workspace `hd` (hosted domain) claim
- Default: `hackclub.com` (configurable)

## Image Processing Details

### Smart Format Conversion
```go
// In internal/imageproc/vips.go
if shouldConvertToJPEG || originalContentType == "image/jpeg" || originalContentType == "image/jpg" {
    // JPEG: Stay as JPEG with compression
    options.Type = bimg.JPEG
    outputContentType = "image/jpeg"
} else {
    // PNG: Only for images with transparency
    options.Type = bimg.PNG  
    outputContentType = "image/png"
}
```

### Resize Logic
```go
const maxFileSize = 5 * 1024 * 1024 // 5MB
needsResize := metadata.Size.Width > 3840 || 
              metadata.Size.Height > 3840 || 
              originalSize > maxFileSize
```

### Deduplication
- Hash **final processed bytes** (not input)
- Base32 encoding with 2-char sharding: `ab/qwelkjq9.jpg`
- Check R2 → upload only if new
- Perfect deduplication across different inputs with same output

## Gmail Integration Architecture

### Client-Side Token Management (Security-First)
```typescript
// Tokens stored in browser localStorage only
interface GmailTokens {
  access_token: string
  refresh_token?: string  
  expires_at?: number
}
```

### Gmail Attachment Processing
1. **URL Parsing**: Extract messageId + attachmentId from Gmail URLs
2. **Message Format**: Always use `msg-f:{messageId}` for Gmail API calls
3. **Attachment Search**: Find specific attachment ID, fallback to first image
4. **Magic Bytes Detection**: Fix `application/octet-stream` → proper MIME type
5. **Automatic Processing**: Gmail attachment → data URI → backend processing → R2

### Gmail URL Format
```
https://mail.google.com/mail/u/1?ui=2&ik=abc&attid=0.1&permmsgid=msg-f:123456&realattid=ii_xyz
```
- `permmsgid`: Message ID (needs `msg-f:` prefix for API)
- `realattid`: Attachment identifier for search
- URLs contain HTML entities (`&amp;`) that need decoding

## HTML Transformation Pipeline

### Gmail-Compatible Output Format
- **Paragraphs**: `<div style="color: rgb(34, 34, 34); font-family: Arial, Helvetica, sans-serif; ...">content</div>`
- **Blank Lines**: `<div style="..."><br></div>` (not CSS margins)
- **Bold/Italic**: Native `<b>` and `<i>` tags preserved
- **Links**: `<a href="..." style="color: rgb(17, 85, 204);">text</a>`
- **Lists**: Native `<ol>/<ul><li>` structure
- **Blockquotes**: `<blockquote class="gmail_quote" style="...">`

### Transformation Process
1. **Image Processing**: Detect blob/Gmail URLs → rehost to R2
2. **Gmail Format Conversion**: Convert all elements to Gmail-compatible structure
3. **Security Sanitization**: Remove scripts, events, dangerous attributes
4. **Link Normalization**: Add mailto: for emails, clean tracking params
5. **Preserve Existing Gmail Format**: Detect and preserve when already correct

## UI/UX Design Principles

### Full-Screen Minimal Interface
- **Editor**: 800px max-width, centered, full-height
- **Toolbar**: Floating bottom center with formatting controls
- **Action Button**: Single 📋 Copy button that processes + copies
- **Status**: Floating notifications (errors/messages) when needed
- **Sign Out**: Small button in bottom-right corner

### User Workflow
1. **Paste**: Rich content from any source (Gmail, Docs, Notion)
2. **Edit**: Use rich text formatting if needed
3. **Process & Copy**: Single click → automatic image processing + copy to clipboard
4. **Paste**: Into Gmail composer with perfect formatting

## Critical Implementation Details

### Gmail API Message ID Format
- **URL contains**: `permmsgid=msg-f:1842260674151743696`
- **API requires**: `msg-f:1842260674151743696` (with prefix)
- **Don't strip prefix** - always use `msg-f:{id}` format

### Image Format Decision Bug Fix
```go
// CRITICAL: JPEGs must stay as JPEG, not convert to PNG
if shouldConvertToJPEG || originalContentType == "image/jpeg" || originalContentType == "image/jpg" {
    // Keep/convert to JPEG
} else {
    // Only PNG for transparency
}
```

### Data URI MIME Type Handling
Gmail API returns `application/octet-stream` - must detect actual type via magic bytes:
- **JPEG**: `FF D8 FF` (preserve compression)
- **PNG**: `89 50 4E 47` (check transparency)

### Lexical Editor Configuration
```typescript
// Essential nodes for rich text + images
nodes: [HeadingNode, ListNode, ListItemNode, QuoteNode, CodeNode, 
        CodeHighlightNode, LinkNode, AutoLinkNode, ImageNode]
```

## Security Model

### Client-Side OAuth Tokens
- **Storage**: Browser localStorage only (never server-side database)
- **Scope**: `gmail.readonly` for attachment access
- **Validation**: Test scope with Gmail profile API call
- **Cleanup**: Auto-clear invalid tokens + force re-auth

### SSRF Protection
```go
// In internal/util/httpfetch.go
// Blocks private IP ranges, requires HTTPS, enforces timeouts
```

### Input Sanitization
- **DOMPurify**: Client-side paste sanitization
- **Server-side**: Remove scripts, events, normalize structure
- **Image URLs**: Only process safe sources + authenticated Gmail attachments

## Debugging and Troubleshooting

### Image Processing Issues
1. Check backend logs for resize decisions: `🔄 Image resize triggered` or `✅ Image resize skipped`
2. Verify format decisions: `🎨 Format decision: image/jpeg → JPEG`
3. Monitor R2 operations: `object already exists` vs `uploaded new object`

### Gmail API Issues  
1. Token validation: Check `tokenValidator.ts` logs for 200/403 responses
2. Message access: Look for `msg-f:` prefix usage in logs
3. Attachment search: Verify specific vs fallback attachment finding

### Common Issues
- **Redirect loops**: Missing session creation in OAuth callback
- **CORS errors**: Gmail images cannot be canvas-captured (by design)
- **Format conversion**: JPEG→PNG conversion indicates logic bug
- **Missing Gmail scope**: Requires API enablement in Google Cloud Console

## Production Deployment

### Required Services
1. **Cloudflare R2**: Bucket with custom domain for CDN
2. **Google Cloud**: OAuth client + Gmail API enabled
3. **Container Runtime**: Docker with libvips support

### Environment Validation
```bash
# Test R2 connection
cd backend && go run test-r2.go

# Test image processing
# Check libvips installation: pkg-config --exists vips
```

## Code Style and Patterns

### Backend (Go)
- **Error handling**: Always wrap with context
- **Logging**: Structured logging with zerolog
- **Config**: Environment-driven with defaults
- **No comments**: Code should be self-documenting

### Frontend (TypeScript)
- **Hooks**: Custom hooks for state management
- **API**: Centralized client in `lib/api.ts`
- **Types**: Comprehensive TypeScript definitions
- **State**: React state for UI, localStorage for persistence

## Testing Strategy

### Backend Testing
```bash
cd backend && go test ./...
# Focus on: image processing, hash generation, MIME detection
```

### Integration Testing
1. **OAuth Flow**: Sign in → token capture → Gmail API access
2. **Image Pipeline**: Upload → process → resize/convert → R2 storage
3. **Gmail Workflow**: Paste → auto-fetch attachments → rehost → copy
4. **Format Output**: Gmail compatibility testing

## Performance Characteristics

### Image Processing
- **Resize threshold**: 3840px or 5MB triggers processing
- **Typical JPEG**: 25-35% size reduction with quality 84
- **Large PNG→JPEG**: Often 80-90% size reduction
- **Processing time**: <600ms for images <5MB

### Gmail API
- **Message lookup**: ~100-200ms per attachment
- **Attachment fetch**: ~300-500ms depending on size
- **Total Gmail processing**: Usually <2s for typical emails

## Future Enhancement Areas

1. **Token Refresh**: Automatic refresh token handling
2. **Batch Gmail Processing**: Multiple attachments in parallel
3. **Format Options**: User choice for JPEG quality/format
4. **Advanced Paste**: Better handling of complex nested structures
5. **Error Recovery**: Retry logic for transient failures

## Critical Success Factors

1. **Gmail API must be enabled** in Google Cloud Console
2. **OAuth callback must set both** session cookie AND provide tokens
3. **Image format logic must preserve** JPEG compression
4. **Message ID format must use** `msg-f:` prefix for Gmail API
5. **Client-side tokens** provide security without server storage risk

## Common Startup Issues & Solutions

### "SESSION_SECRET is required"
```bash
# Check .env file exists and has SESSION_SECRET
cat .env | grep SESSION_SECRET

# If missing, add any secure random string
echo "SESSION_SECRET=$(openssl rand -base64 32)" >> .env
```

### "vips not found" / Build errors
```bash
# Install libvips
brew install vips  # macOS
sudo apt-get install libvips-dev  # Ubuntu

# Verify installation
pkg-config --exists vips && echo "✅ OK" || echo "❌ Failed"
```

### "air: command not found"
```bash
# Install Air
go install github.com/air-verse/air@latest

# Add Go bin to PATH in ~/.zshrc
echo 'export PATH="$PATH:$(go env GOPATH)/bin"' >> ~/.zshrc
source ~/.zshrc

# Or use full path in terminal
$(go env GOPATH)/bin/air
```

### OAuth Redirect Loop
```bash
# Check if both Client ID AND Secret are set
grep -E "GOOGLE_OAUTH_CLIENT_(ID|SECRET)" .env

# Enable Google+ API and Gmail API in Google Cloud Console
# Verify redirect URI: http://localhost:3000/api/auth/callback
```

### R2 Upload Failures
```bash
# Test R2 credentials
cd backend && go run test-r2.go

# Common issues:
# - Wrong endpoint format (must include account ID)
# - Missing bucket permissions (needs read/write)
# - API token doesn't match account
```

### Gmail API 403 Errors
- **Enable Gmail API** in Google Cloud Console first
- **Clear tokens**: `localStorage.removeItem('gmail_tokens')` in browser console
- **Re-authenticate**: Click "Sign out" → "Sign in" to get new scope
- **Check scopes**: Look for `gmail.readonly` in OAuth consent screen

This application represents a complete end-to-end solution for Gmail-optimized content preparation with automatic image processing and rehosting.
