# ── Stage 1: Build ───────────────────────────────────────────────────────────
FROM docker.io/library/golang:1.24-bookworm AS builder

WORKDIR /app

# Cache module downloads separately from source code
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source and compile
COPY . .

# Always build the CLI tool so it is available inside the container.
# TARGET selects the server binary (default: witti-web).
ARG TARGET=witti-web
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w" -o /out/witti     ./cmd/witti && \
    go build -trimpath -ldflags="-s -w" -o /out/witti-app ./cmd/${TARGET}

# ── Stage 2: Runtime ─────────────────────────────────────────────────────────
# debian:bookworm-slim is the smallest Debian image with apt support
FROM docker.io/library/debian:bookworm-slim

# Install only what is needed: TLS certificates for outbound HTTPS (if ever used)
RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Server entrypoint binary + CLI tool
COPY --from=builder /out/witti-app /usr/local/bin/witti-app
COPY --from=builder /out/witti     /usr/local/bin/witti

# Run as a non-root user
RUN useradd --system --no-create-home witti
USER witti

EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/witti-app"]
CMD ["-addr", ":8080"]

