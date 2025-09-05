# Build stage for frontend
FROM node:18-alpine AS frontend-builder

WORKDIR /app/frontend
COPY frontend/package*.json ./
RUN npm ci

COPY frontend/ ./
RUN npm run build

# Build stage for backend
FROM golang:1.23-alpine AS backend-builder

# Install build dependencies
RUN apk add --no-cache \
    gcc \
    musl-dev \
    pkgconfig \
    vips-dev

WORKDIR /app/backend
COPY backend/go.mod ./
COPY backend/go.sum ./
RUN go mod download

COPY backend/ ./
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o server cmd/server/main.go

# Runtime stage
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache \
    vips \
    oxipng \
    ca-certificates \
    tzdata

# Create non-root user
RUN addgroup -g 1000 app && \
    adduser -u 1000 -G app -D app

WORKDIR /app

# Copy backend binary
COPY --from=backend-builder /app/backend/server .

# Copy frontend static files
COPY --from=frontend-builder /app/frontend/.next/static ./static
COPY --from=frontend-builder /app/frontend/public ./public
COPY --from=frontend-builder /app/frontend/.next/build-manifest.json ./build-manifest.json
COPY --from=frontend-builder /app/frontend/.next/app-build-manifest.json ./app-build-manifest.json

# Set ownership
RUN chown -R app:app /app

USER app

EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/healthz || exit 1

CMD ["./server"]
