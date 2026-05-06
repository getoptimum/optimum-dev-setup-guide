# ==============================================================================
# Stage 1: Build the Go binaries
# ==============================================================================
FROM --platform=$BUILDPLATFORM golang:latest AS builder

ARG TARGETOS
ARG TARGETARCH

WORKDIR /src

# Copy go mod and sum files first to leverage Docker cache
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build the Linux binaries dynamically based on the target platform
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -o /p2p-multi-subscribe ./cmd/multi-subscribe
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -o /p2p-multi-publish ./cmd/multi-publish

# ==============================================================================
# Stage 2: Build the final Python image
# ==============================================================================
FROM python:3.10-slim

WORKDIR /app

COPY k8_cron_job/entrypoint.sh k8_cron_job/analyze_latency.py ./
RUN chmod +x entrypoint.sh

# The P2P node IP list will be mounted dynamically via K8s ConfigMap at runtime
# (Do NOT copy it here so you don't have to rebuild the image when IPs change)

# Copy the compiled Go binaries from the builder stage
COPY --from=builder /p2p-multi-subscribe /p2p-multi-publish ./

# Set the default command when the container starts
ENTRYPOINT ["./entrypoint.sh"]
