# =============================================================================
# search-gin Dockerfile — Multistage Production Build
# =============================================================================

# ---- Stage 1: Frontend build ----
FROM node:20-alpine AS frontend
WORKDIR /build/frontend
COPY frontend/package.json frontend/yarn.lock ./
RUN yarn install --frozen-lockfile
COPY frontend/ .
RUN yarn build
RUN ls -la dist/spa/

# ---- Stage 2: Go build ----
FROM golang:1.25-alpine AS builder
RUN apk add --no-cache gcc musl-dev
WORKDIR /build

# Cache deps first
COPY go.mod go.sum ./
RUN go mod download

# Copy all source
COPY . .

# Copy frontend dist from Stage 1
COPY --from=frontend /build/frontend/dist/spa ./dist/

# Build (Linux prod binary)
RUN CGO_ENABLED=0 go build -tags=prod -ldflags="-s -w" -o /search-gin .

# ---- Stage 3: Runtime ----
FROM alpine:3.20
RUN apk add --no-cache \
    ffmpeg \
    ca-certificates \
    tzdata
RUN adduser -D -u 1000 search
WORKDIR /app
COPY --from=builder /search-gin /app/
COPY --from=builder /build/setting.json /app/ 2>/dev/null || true
COPY --from=builder /build/ffmpeg /app/ 2>/dev/null || true
COPY --from=builder /build/ffplay /app/ 2>/dev/null || true

USER search
EXPOSE 10081 10082
VOLUME ["/app/data", "/media"]
ENTRYPOINT ["/app/search-gin"]
