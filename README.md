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
- Pasting from Gmail an email with multiple different images in it and make sure they all come through

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

## Docker Deployment

### Quick Start
```bash
# Copy and configure environment
cp .env.example .env
# Edit .env with your values...

# Start the application
docker-compose up -d app
```

### Port Configuration
By default, the app runs on port 8080. You can customize the external port using environment variables:

```bash
# Use different external port
HOST_PORT=9000 docker-compose up -d app

# Or set in .env file
HOST_PORT=9000
```

**Note:** HTTPS is handled by your hosting platform's reverse proxy, not by the container.

## Features

- **Predictable Gmail formatting**: basic, consistent HTML subset
- **Bulletproof images**: rehosted on our domain, scaled, optimized, deduped, and cached
- **Speed**: paste → clean → copy in < 10 seconds for typical documents
- **Safety**: sanitize HTML; strip tracking & scripting
