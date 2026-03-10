#!/bin/bash

# SMAP Project Consumer — Build & Push to Harbor Registry
# Usage: ./scripts/build-consumer.sh [build-push|login|help]

set -euo pipefail

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# ── Configuration ─────────────────────────────────────────────
REGISTRY="${HARBOR_REGISTRY:-registry.tantai.dev}"
PROJECT="smap"
SERVICE="project-consumer"
DOCKERFILE="cmd/consumer/Dockerfile"
PLATFORM="${PLATFORM:-linux/amd64}"
REGISTRY_USER="${HARBOR_USERNAME:?HARBOR_USERNAME is not set. Export it in ~/.zshrc}"
REGISTRY_PASS="${HARBOR_PASSWORD:?HARBOR_PASSWORD is not set. Export it in ~/.zshrc}"

# ── Helpers ───────────────────────────────────────────────────
info()    { echo -e "${BLUE}[INFO]${NC} $1"; }
success() { echo -e "${GREEN}[OK]${NC} $1"; }
warn()    { echo -e "${YELLOW}[WARN]${NC} $1"; }
die()     { echo -e "${RED}[ERROR]${NC} $1"; exit 1; }

generate_tag() { date +"%y%m%d-%H%M%S"; }

image_ref() {
    local tag="${1:-latest}"
    echo "${REGISTRY}/${PROJECT}/${SERVICE}:${tag}"
}

# ── Login ─────────────────────────────────────────────────────
login() {
    info "Logging into Harbor registry ${REGISTRY} ..."
    echo "${REGISTRY_PASS}" | docker login "${REGISTRY}" \
        -u "${REGISTRY_USER}" --password-stdin \
        || die "Login failed"
    success "Logged in to ${REGISTRY}"
}

# ── Prerequisites ─────────────────────────────────────────────
check_prerequisites() {
    command -v docker &>/dev/null || die "Docker is not installed"
    docker buildx version &>/dev/null || die "Docker buildx is not available"
    [ -f "${DOCKERFILE}" ]         || die "Dockerfile not found: ${DOCKERFILE}"
}

# ── Build & Push ──────────────────────────────────────────────
build_and_push() {
    check_prerequisites
    login

    local tag
    tag=$(generate_tag)
    local img
    img=$(image_ref "${tag}")
    local img_latest
    img_latest=$(image_ref latest)

    info "Registry:   ${REGISTRY}"
    info "Image:      ${img}"
    info "Platform:   ${PLATFORM}"
    info "Dockerfile: ${DOCKERFILE}"

    docker buildx build \
        --platform "${PLATFORM}" \
        --provenance=false \
        --sbom=false \
        --tag "${img}" \
        --tag "${img_latest}" \
        --file "${DOCKERFILE}" \
        --push \
        . \
        || die "Build & push failed"

    success "Pushed ${img}"
    success "Pushed ${img_latest}"
}

# ── Help ──────────────────────────────────────────────────────
show_help() {
    cat <<EOF
${GREEN}SMAP Project Consumer — Build & Push${NC}

Usage: $0 [command]

Commands:
    build-push      Build and push image (default)
    login           Login to Harbor registry
    help            Show this help

Configuration:
    Registry:       ${REGISTRY}
    Project:        ${PROJECT}
    Service:        ${SERVICE}
    Platform:       ${PLATFORM}
    Dockerfile:     ${DOCKERFILE}

Environment overrides:
    PLATFORM            Target platform        (default: linux/amd64)
    HARBOR_USERNAME     Registry user
    HARBOR_PASSWORD     Registry password
EOF
}

# ── Main ──────────────────────────────────────────────────────
case "${1:-build-push}" in
    build-push) build_and_push ;;
    login)      login ;;
    help|--help|-h) show_help ;;
    *) die "Unknown command: $1"; show_help ;;
esac
