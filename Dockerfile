# syntax=docker/dockerfile:1.7
# Stage 1: Build static binary
FROM golang:1.25-alpine AS builder

WORKDIR /src

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source tree
COPY . .

# Compile static binary with stripped debug symbols and DWARF tables
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w" -o /bin/syshud ./cmd/hud

# Stage 2: Minimalist, hardened runtime container
FROM alpine:3.21

# Install runtime ca-certificates and create an unprivileged user
RUN apk --no-cache add ca-certificates tzdata && \
    addgroup -g 10001 -S syshud && \
    adduser -u 10001 -S syshud -G syshud

USER 10001:10001
WORKDIR /home/syshud

COPY --from=builder --chown=10001:10001 /bin/syshud /usr/local/bin/syshud

# Terminal HUD interactive execution
ENTRYPOINT ["/usr/local/bin/syshud"]

