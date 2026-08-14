#!/bin/bash

set -euo pipefail

# gitrakz override of the framework's build.sh. The Makefile's find_script picks
# scripts/make/<name> ahead of scripts/make/servicepack/<name>, so this file
# wins. It is the framework script verbatim EXCEPT it builds ./cmd (the app
# entrypoint) instead of ./cmd/...: gitrakz ships cmd/repogen (a code generator,
# package main) alongside cmd/main.go, and `go build -o <file> ./cmd/...` fails
# with "cannot write multiple packages to non-directory". The release image
# (Dockerfile) already builds ./cmd; this keeps `make build` in step. Keep in
# step with the framework script on updates.

# Source common functions from the framework script dir alongside this override.
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/servicepack/common.sh"

APP_NAME="$(head -n 1 go.mod | awk '{print $2}' | awk -F'/' '{print $NF}')"
readonly APP_NAME
BUILD_COMMIT="$(git rev-parse --verify HEAD 2>/dev/null || true)"
readonly BUILD_COMMIT

# Pinned by digest, not by tag. This is the image `make build` actually uses --
# the Dockerfiles are a separate path (`make docker-build`), so pinning them
# alone leaves the default build consuming a mutable tag. Bump deliberately.
readonly GO_BUILD_IMAGE="golang:1.26.4-alpine@sha256:3ad57304ad93bbec8548a0437ad9e06a455660655d9af011d58b993f6f615648"

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
        -o "./build/${APP_NAME}" ./cmd && \
        chown "${USER_UID}:${USER_GID}" "./build/${APP_NAME}"
    '

success "Binary built successfully: ./build/$APP_NAME"
