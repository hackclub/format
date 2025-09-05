# format.hackclub.com

A paste-clean-copy tool: you paste any rich text; the app sanitizes and normalizes the HTML; images are rehosted on Hack Club's CDN (R2 + Cloudflare) with dedupe, intelligent conversion, and metadata stripping; you copy the Gmail-safe result and paste into Gmail.

Things to test:

- Image upload:
  - PNG with transparency
  - PNG no transparency (converts to jpeg)
  - JPEG
- Pasting rich HTML from Gmail with image attachments
- Pasting image directly from local fileystem
- Pasting image from a website

## Architecture

```
Browser (Next.js + Lexical)
        |
        | OAuth (Google One-Tap/Popup) → ID token
        v
Go API (chi)  ──>  Image pipeline (libvips via bimg + jpegli/mozjpeg + oxipng/libimagequant)
        |             └─ hash, dedupe, resize, convert, strip metadata
        |                        
        └─> R2 (S3 API)  <── Cloudflare CDN on custom domain (e.g., i.format.hackclub.com)
```

## Development

### Backend (Go)
```bash
cd backend
go run cmd/server/main.go
```

### Frontend (Next.js)
```bash
cd frontend
npm run dev
```

## Environment Variables

See `.env.example` for required environment variables.

## Features

- **Predictable Gmail formatting**: basic, consistent HTML subset
- **Bulletproof images**: rehosted on our domain, scaled, optimized, deduped, and cached
- **Speed**: paste → clean → copy in < 10 seconds for typical documents
- **Safety**: sanitize HTML; strip tracking & scripting
