# Setup Guide for Format

This guide will help you set up the Format application locally and in production.

## Prerequisites

- **Go 1.22+**
- **Node.js 18+**
- **libvips** (for image processing)
- **Air** (for Go hot reloading in development)
- **Docker** (optional, for containerized deployment)
- **Google Cloud Console access** (for OAuth setup)
- **Cloudflare R2 account** (for image storage)

## Local Development Setup

### 1. Install System Dependencies

#### macOS
```bash
# Install libvips
brew install vips

# Install Air for hot reloading
go install github.com/air-verse/air@latest

# Install oxipng (optional, for PNG optimization)
brew install oxipng
```

#### Ubuntu/Debian
```bash
# Install libvips
sudo apt-get update
sudo apt-get install libvips-dev

# Install oxipng
sudo apt-get install oxipng
```

### 2. Clone and Setup

```bash
git clone <repository-url>
cd format
make setup
```

### 3. Configure Environment

Copy `.env.example` to `.env` and configure:

```env
# Server configuration
PORT=8080
APP_BASE_URL=http://localhost:3000
SESSION_SECRET=your-very-secret-session-key-here

# Google OAuth
GOOGLE_OAUTH_CLIENT_ID=your-google-oauth-client-id
GOOGLE_OAUTH_CLIENT_SECRET=your-google-oauth-client-secret

# Domain restrictions
ALLOWED_DOMAINS=hackclub.com

# Image processing
MAX_IMAGE_W=1600
MAX_IMAGE_H=1600
JPEG_QUALITY=84
JPEG_PROGRESSIVE=true
PNG_STRIP=true

# Cloudflare R2 Storage
R2_ACCOUNT_ID=your-r2-account-id
R2_ACCESS_KEY_ID=your-r2-access-key
R2_SECRET_ACCESS_KEY=your-r2-secret-key
R2_BUCKET=format-assets
R2_PUBLIC_BASE_URL=https://i.format.hackclub.com
R2_S3_ENDPOINT=https://your-account-id.r2.cloudflarestorage.com
```

### 4. Google OAuth Setup

1. Go to [Google Cloud Console](https://console.cloud.google.com/)
2. Create a new project or select existing one
3. Enable the Google+ API
4. Create OAuth 2.0 credentials:
   - Application type: Web application
   - Authorized redirect URIs: `http://localhost:3000/api/auth/callback`
   - For production: `https://format.hackclub.com/api/auth/callback`
5. Copy the Client ID to your `.env` file

### 5. Cloudflare R2 Setup

1. Log in to [Cloudflare Dashboard](https://dash.cloudflare.com/)
2. Go to R2 Object Storage
3. Create a bucket (e.g., `format-assets`)
4. Create R2 API tokens:
   - Go to "Manage R2 API Tokens"
   - Create token with read/write permissions
5. Set up custom domain (optional but recommended):
   - Go to your bucket settings
   - Connect a custom domain (e.g., `i.format.hackclub.com`)
   - Update `R2_PUBLIC_BASE_URL` in your `.env`

### 6. Start Development Servers

```bash
# Start both backend and frontend
make dev

# Or start individually
make dev-backend  # Runs on :8080
make dev-frontend # Runs on :3000
```

## Building libvips with jpegli (Optional)

For optimal JPEG compression, you can build libvips with jpegli support:

### Building jpegli

```bash
# Clone libjxl (contains jpegli)
git clone https://github.com/libjxl/libjxl.git
cd libjxl
git submodule update --init --recursive

# Build with jpegli
mkdir build && cd build
cmake .. -DJPEGXL_ENABLE_JPEGLI=ON -DJPEGXL_ENABLE_PLUGINS=ON
make -j$(nproc)
sudo make install
```

### Building libvips with jpegli

```bash
# Download libvips source
wget https://github.com/libvips/libvips/releases/download/v8.15.1/vips-8.15.1.tar.xz
tar xf vips-8.15.1.tar.xz && cd vips-8.15.1

# Configure with jpegli
meson setup build --buildtype=release -Djpeg=enabled
cd build && ninja && sudo ninja install
```

## Production Deployment

### Docker Deployment

```bash
# Build and start with Docker Compose
make docker-prod
```

### Manual Deployment

1. Build the application:
```bash
make build
```

2. Set up production environment variables

3. Deploy the binary to your server

4. Set up reverse proxy (nginx example):
```nginx
server {
    listen 80;
    server_name format.hackclub.com;
    
    location / {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

## Environment Variables Reference

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `PORT` | Server port | `8080` | No |
| `APP_BASE_URL` | Frontend URL | `http://localhost:3000` | Yes |
| `SESSION_SECRET` | Session encryption key | - | Yes |
| `GOOGLE_OAUTH_CLIENT_ID` | Google OAuth client ID | - | Yes |
| `GOOGLE_OAUTH_CLIENT_SECRET` | Google OAuth client secret | - | Yes |
| `ALLOWED_DOMAINS` | Comma-separated allowed domains | `hackclub.com` | Yes |
| `MAX_IMAGE_W` | Maximum image width | `1600` | No |
| `MAX_IMAGE_H` | Maximum image height | `1600` | No |
| `JPEG_QUALITY` | JPEG quality (0-100) | `84` | No |
| `JPEG_PROGRESSIVE` | Progressive JPEG | `true` | No |
| `PNG_STRIP` | Strip PNG metadata | `true` | No |
| `R2_ACCOUNT_ID` | Cloudflare R2 account ID | - | Yes |
| `R2_ACCESS_KEY_ID` | R2 access key | - | Yes |
| `R2_SECRET_ACCESS_KEY` | R2 secret key | - | Yes |
| `R2_BUCKET` | R2 bucket name | `format-assets` | Yes |
| `R2_PUBLIC_BASE_URL` | CDN base URL | - | Yes |
| `R2_S3_ENDPOINT` | R2 S3 endpoint | - | Yes |

## Troubleshooting

### libvips Issues
- Ensure libvips is installed and accessible
- On macOS, you may need to set `PKG_CONFIG_PATH=/opt/homebrew/lib/pkgconfig`

### OAuth Issues
- Verify redirect URIs match your environment
- Check that the Google+ API is enabled
- Ensure the client ID is correct

### R2 Connection Issues
- Verify R2 credentials and endpoint
- Check bucket permissions
- Test connectivity with AWS CLI configured for R2

### Image Processing Issues
- Check that uploaded images are valid
- Verify image format support
- Monitor server logs for processing errors

## Development Commands

```bash
make help           # Show all available commands
make dev            # Start development servers
make build          # Build application
make test           # Run tests
make check          # Type checking and linting
make clean          # Clean build artifacts
make docker-dev     # Start with Docker
make logs           # View application logs
```

## Production Checklist

- [ ] Configure HTTPS/TLS
- [ ] Set secure session secret
- [ ] Configure proper CORS settings
- [ ] Set up monitoring and logging
- [ ] Configure backup strategy for R2
- [ ] Set up rate limiting
- [ ] Configure CDN caching rules
- [ ] Test OAuth flow in production
- [ ] Verify image processing pipeline
- [ ] Set up health checks
