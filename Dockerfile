# Production Dockerfile - Multi-stage build
FROM golang:1.26-alpine@sha256:0178a641fbb4858c5f1b48e34bdaabe0350a330a1b1149aabd498d0699ff5fb2 AS builder

ARG BUILD_COMMIT=""

# Install build dependencies
RUN apk add --no-cache \
    gcc \
    musl-dev

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build binary with static linking.
#
# The app name is derived from go.mod's module path and injected into
# main.appName, which cmd/main.go uses as cobra's Use:/Short:. Without it the
# binary falls back to the literal "servicepack" and introduces itself by the
# framework's name in its own --help.
RUN APP_NAME="$(head -n 1 go.mod | awk '{print $2}' | awk -F'/' '{print $NF}')" && \
    CGO_ENABLED=0 go build -a \
    -ldflags "-X main.appName=${APP_NAME} -X main.buildCommit=${BUILD_COMMIT}" \
    -o ./build/app ./cmd

# Final stage - minimal runtime image
# Pinned by digest, not by tag: a tag is mutable, so `alpine:latest` lets an
# upstream republish different bytes under the same name and your next build
# silently picks them up. Bump this deliberately.
FROM alpine:3.24@sha256:28bd5fe8b56d1bd048e5babf5b10710ebe0bae67db86916198a6eec434943f8b

# ca-certificates for HTTPS; github-cli because gitrakz shells out to `gh` at
# runtime (it authenticates via the GH_TOKEN env var — no interactive login).
RUN apk --no-cache add ca-certificates github-cli

# The documented host bind mount defaults to UID/GID 1000. Keep the image's
# non-root runtime identity stable so an installer's ./data is writable.
RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser -s /bin/sh appuser

# /data is the container-side mountpoint for the host install directory's
# ./data. It is deliberately not declared as a Docker volume: persistence is
# explicit and visible on the host.
RUN mkdir -p /data && chown appuser:appuser /data

# Set working directory
WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /app/build/app .

# Change ownership to non-root user
RUN chown appuser:appuser /app/app

# Switch to non-root user
USER appuser

# gitrakz always listens on :8080 inside the container; publish it on the host
# with `docker run -p <host-port>:8080`.
EXPOSE 8080

# Set entrypoint to the app binary
ENTRYPOINT ["./app"]

# Default command if no args provided
CMD ["--help"]
