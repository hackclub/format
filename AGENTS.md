# Format.hackclub.com - Agent Instructions

This file contains instructions for AI agents working on this project.

## Development Commands

### Backend (Go)
```bash
cd backend
$(go env GOPATH)/bin/air            # Start development server with hot reload
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
make dev                           # Start both backend and frontend
make build                         # Build both components
make test                          # Run all tests
make check                         # Run type checking and linting
```

## Key Technologies

- **Backend**: Go 1.22+, chi router, bimg (libvips), Cloudflare R2
- **Frontend**: Next.js 14, React 18, Lexical editor, TailwindCSS
- **Authentication**: Google OAuth with domain restrictions
- **Image Processing**: libvips with mozjpeg/jpegli support
- **Storage**: Cloudflare R2 with CDN

## Environment Setup

1. Install libvips: `brew install vips` (macOS) or `apt-get install libvips-dev` (Ubuntu)
2. Copy `.env.example` to `.env` and configure
3. Run `make setup` to install dependencies

## Architecture

The application follows a clean architecture with:
- Go backend API with chi router
- Next.js frontend with Lexical editor
- Google OAuth for authentication with domain restrictions
- Cloudflare R2 for image storage
- libvips for high-performance image processing

## Testing

- Backend: Go standard testing with table-driven tests
- Frontend: React Testing Library (to be implemented)
- Integration: Manual testing with real Gmail paste workflow

## Deployment

- Docker-based deployment with multi-stage builds
- Environment variables for configuration
- Health checks and graceful shutdown
- CDN integration with Cloudflare

## Code Style

- Go: Standard Go formatting with gofmt
- TypeScript: Strict mode enabled
- Linting: ESLint for frontend, golangci-lint for backend
- No unnecessary comments in code
- Consistent error handling patterns

## Security Considerations

- HTTPS-only image fetching with SSRF protection
- Session-based authentication with secure cookies
- Domain-restricted OAuth (hackclub.com only by default)
- Input validation and sanitization
- Metadata stripping from processed images
