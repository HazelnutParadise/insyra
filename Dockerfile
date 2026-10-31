# Insyra CLI on the official Go image, able to use the host's GPU through Vulkan.
#
# GO_IMAGE is pinned by digest in the release workflow; the default is for a
# local build only.
ARG GO_IMAGE=golang:1.26.10-trixie

# Cross-compile on the build machine's own architecture: insyra has no cgo, and
# goffi loads libvulkan.so.1 with dlopen even under CGO_ENABLED=0.
FROM --platform=$BUILDPLATFORM ${GO_IMAGE} AS build
ARG TARGETOS TARGETARCH
WORKDIR /src
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY . .
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -trimpath -ldflags="-s -w" -o /out/insyra ./cmd/insyra

FROM ${GO_IMAGE}
# libvulkan1 is the Vulkan loader the accel backend opens. mesa-vulkan-drivers
# carries the AMD (radv) and Intel (anv) drivers that run inside the container;
# NVIDIA's driver is mounted from the host by the NVIDIA Container Toolkit.
RUN apt-get update \
 && apt-get install -y --no-install-recommends libvulkan1 mesa-vulkan-drivers vulkan-tools \
 && rm -rf /var/lib/apt/lists/* \
 && mkdir -m 1777 /home/insyra
# Without "graphics" the NVIDIA toolkit mounts no Vulkan driver (default is compute,utility).
# HOME and GOCACHE are writable by any user, so --user works too, and the CLI's
# environments (~/.insyra) live in one place a volume can keep.
ENV NVIDIA_VISIBLE_DEVICES=all \
    NVIDIA_DRIVER_CAPABILITIES=compute,utility,graphics \
    HOME=/home/insyra \
    GOCACHE=/go/cache
COPY --from=build /out/insyra /usr/local/bin/insyra
WORKDIR /work
CMD ["insyra"]
