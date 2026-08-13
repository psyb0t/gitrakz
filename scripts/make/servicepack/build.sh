#!/bin/bash

set -euo pipefail

# Source common functions
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

readonly APP_NAME="$(head -n 1 go.mod | awk '{print $2}' | awk -F'/' '{print $NF}')"
readonly BUILD_COMMIT="$(git rev-parse --verify HEAD 2>/dev/null || true)"

# Pinned by digest, not by tag. This is the image `make build` actually uses --
# the Dockerfiles are a separate path (`make docker-build`), so pinning them
# alone leaves the default build consuming a mutable tag. Bump deliberately.
readonly GO_BUILD_IMAGE="golang:1.26-alpine@sha256:0178a641fbb4858c5f1b48e34bdaabe0350a330a1b1149aabd498d0699ff5fb2"

section "Building Application"
info "Building $APP_NAME binary using Docker..."
info "Build commit: ${BUILD_COMMIT:-unavailable}"

# Create build directory
mkdir -p ./build

# Build using Docker
docker run --rm \
    -v "$(pwd)":/app \
    -w /app \
    -e USER_UID="$(id -u)" \
    -e USER_GID="$(id -g)" \
    -e "APP_NAME=$APP_NAME" \
    -e "BUILD_COMMIT=$BUILD_COMMIT" \
    "$GO_BUILD_IMAGE" \
    sh -ceu '
        apk add --no-cache gcc musl-dev && \
        CGO_ENABLED=0 go build -a \
        -ldflags "-X main.appName=${APP_NAME} -X main.buildCommit=${BUILD_COMMIT}" \
        -o "./build/${APP_NAME}" ./cmd/... && \
        chown "${USER_UID}:${USER_GID}" "./build/${APP_NAME}"
    '

success "Binary built successfully: ./build/$APP_NAME"
