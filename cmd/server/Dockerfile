# syntax=docker/dockerfile:1

# =============================================================================
# STAGE 1: BUILDER
# --platform=$BUILDPLATFORM ensures the Go compiler runs natively (no QEMU)
# =============================================================================
FROM --platform=$BUILDPLATFORM golang:1.25-alpine AS builder

WORKDIR /app

ARG TARGETOS
ARG TARGETARCH

# Install swag alongside apk deps — cached before source COPY
RUN apk add --no-cache git tzdata && \
    go install github.com/swaggo/swag/cmd/swag@v1.8.12

# Cache dependencies (layer changes only when go.mod/go.sum change)
COPY go.mod go.sum ./
COPY vendor/ vendor/

COPY . .

# Generate Swagger docs (swag already installed above)
RUN swag init -g cmd/server/main.go --parseVendor

# Build static binary — trimpath for reproducible & smaller output
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 \
    GOOS=${TARGETOS} \
    GOARCH=${TARGETARCH} \
    go build \
    -mod=vendor \
    -trimpath \
    -ldflags="-s -w" \
    -o project-srv ./cmd/server

# =============================================================================
# STAGE 2: RUNTIME (Distroless Static — ~2 MB base, no shell, non-root)
# =============================================================================
FROM gcr.io/distroless/static-debian12:nonroot AS runtime

LABEL org.opencontainers.image.source="https://github.com/smap-hcmut/project-srv" \
      org.opencontainers.image.title="smap-project-srv"

WORKDIR /app

# Only copy the needed timezone instead of all ~5 MB of zoneinfo
COPY --link --from=builder /usr/share/zoneinfo/Asia/Ho_Chi_Minh /usr/share/zoneinfo/Asia/Ho_Chi_Minh
COPY --link --from=builder /usr/share/zoneinfo/UTC /usr/share/zoneinfo/UTC
ENV TZ=Asia/Ho_Chi_Minh

COPY --link --from=builder --chown=nonroot:nonroot /app/project-srv .

USER nonroot:nonroot

EXPOSE 8080

ENTRYPOINT ["./project-srv"]
