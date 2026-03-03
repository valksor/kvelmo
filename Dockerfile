# Build stage for Go binary
FROM golang:1.26-alpine AS go-builder

WORKDIR /app

# Install git for go mod download
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o kvelmo ./cmd/kvelmo

# Build stage for web UI
FROM node:24-alpine AS web-builder

WORKDIR /app/web

# Copy package files
COPY web/package*.json ./
RUN npm ci

# Copy web source
COPY web/ ./

# Build
RUN npm run build

# Final stage
FROM alpine:3.23

WORKDIR /app

# Install ca-certificates for HTTPS
RUN apk add --no-cache ca-certificates git

# Copy binary
COPY --from=go-builder /app/kvelmo /usr/local/bin/kvelmo

# Copy web assets
COPY --from=web-builder /app/web/dist /app/web/dist

# Create non-root user
RUN adduser -D -u 1000 kvelmo
USER kvelmo

# Create config directory
RUN mkdir -p /home/kvelmo/.valksor/kvelmo

EXPOSE 3000

ENTRYPOINT ["kvelmo"]
CMD ["--help"]
