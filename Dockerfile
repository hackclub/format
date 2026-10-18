# Build stage for frontend
FROM dhi.io/bun:1-alpine3.24-dev AS frontend-builder

WORKDIR /app/frontend
COPY frontend/package.json frontend/bun.lock ./
RUN bun install --frozen-lockfile

COPY frontend/ ./
RUN bun run build

FROM dhi.io/alpine-base:3.24-dev AS png-optimizer

ARG TARGETARCH
RUN case "$TARGETARCH" in \
      arm64) target=aarch64; checksum=ff21aa2a005b324c8dfbdf13cd2f469fb6728b27dacded82635824ecd12e3b35 ;; \
      amd64) target=x86_64; checksum=1813750ef592c5350ca79c88f98b2c0876d05c826dc7156245256d4255c1ad17 ;; \
      *) exit 1 ;; \
    esac && \
    wget -O /tmp/oxipng.tar.gz "https://github.com/oxipng/oxipng/releases/download/v10.2.1/oxipng-10.2.1-${target}-unknown-linux-musl.tar.gz" && \
    echo "$checksum  /tmp/oxipng.tar.gz" | sha256sum -c - && \
    tar -xzf /tmp/oxipng.tar.gz -C /tmp && \
    mkdir -p /opt/oxipng/bin && \
    cp "/tmp/oxipng-10.2.1-${target}-unknown-linux-musl/oxipng" /opt/oxipng/bin/oxipng && \
    mkdir -p /opt/oxipng/share/licenses/oxipng && \
    cp "/tmp/oxipng-10.2.1-${target}-unknown-linux-musl/LICENSE" /opt/oxipng/share/licenses/oxipng/LICENSE

# Build stage for backend
FROM dhi.io/golang:1.27-alpine3.24-dev AS backend-builder

# Install build dependencies
RUN apk add --no-cache \
    gcc \
    musl-dev \
    pkgconfig \
    vips-dev \
    oxipng

WORKDIR /app/backend
COPY backend/go.mod ./
COPY backend/go.sum ./
RUN go mod download

COPY backend/ ./
COPY --from=png-optimizer /opt/oxipng/bin/oxipng /usr/local/bin/oxipng
RUN --mount=type=cache,id=format-go-build,target=/root/.cache/go-build go test ./...
RUN --mount=type=cache,id=format-go-build,target=/root/.cache/go-build CGO_ENABLED=1 go build -o server ./cmd/server

FROM backend-builder AS image-libraries

ARG TARGETARCH
RUN apk add --no-cache build-base meson ninja xz pax-utils
RUN wget -O /tmp/vips.tar.xz https://github.com/libvips/libvips/releases/download/v8.18.7/vips-8.18.7.tar.xz && \
    echo '5baaead3b0bb20ffdb9e9ff09aa9fda08620923df77b63b436654cb5e0b3bf94  /tmp/vips.tar.xz' | sha256sum -c - && \
    tar -xJf /tmp/vips.tar.xz -C /tmp && \
    meson setup /tmp/vips-build /tmp/vips-8.18.7 \
      --prefix=/opt/vips --libdir=lib --buildtype=release \
      -Dintrospection=disabled -Dcplusplus=false -Dmodules=disabled -Dexamples=false && \
    meson compile -C /tmp/vips-build -j 2 && \
    meson install -C /tmp/vips-build
COPY --from=png-optimizer /opt/oxipng /opt/vips

ENV LD_LIBRARY_PATH=/opt/vips/lib
ENV PATH=/opt/vips/bin:$PATH
RUN --mount=type=cache,id=format-go-build,target=/root/.cache/go-build vips --version && oxipng --version && go test -count=1 ./internal/imageproc
RUN scanelf --needed --nobanner --format '%n#p' --recursive /opt/vips/lib /opt/vips/bin | \
    tr ',' '\n' | sort -u | \
    awk 'system("test -e /opt/vips/lib/" $1) != 0 { print "so:" $1 }' > /opt/vips/runtime-dependencies

FROM dhi.io/alpine-base:3.24 AS runtime-base

FROM backend-builder AS runtime-dependencies
COPY --from=runtime-base / /runtime/
COPY --from=image-libraries /opt/vips/runtime-dependencies /tmp/runtime-dependencies
RUN cp -a /etc/apk/keys/. /runtime/etc/apk/keys/ && \
    cp /etc/apk/repositories /runtime/etc/apk/repositories && \
    apk --root /runtime --no-cache --no-scripts upgrade && \
    apk --root /runtime --no-cache --no-scripts add tzdata $(cat /tmp/runtime-dependencies)

FROM runtime-base AS runtime

COPY --from=runtime-dependencies /runtime/ /
COPY --from=image-libraries /opt/vips /opt/vips
ENV LD_LIBRARY_PATH=/opt/vips/lib
ENV PATH=/opt/vips/bin:$PATH

WORKDIR /app
COPY --from=backend-builder /app/backend/server ./server
COPY --from=frontend-builder /app/frontend/.next ./.next
COPY --from=frontend-builder /app/frontend/public ./public

USER 65532:65532

EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/healthz || exit 1

CMD ["./server"]
