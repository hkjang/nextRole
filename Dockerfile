# syntax=docker/dockerfile:1
FROM node:24.21.0-bookworm-slim@sha256:0e0ff40c39bc087845bfb27465a0df4ea419520094bc35842ff83dd8cbe6f9b6 AS web-build
WORKDIR /build/web
COPY web/package.json web/package-lock.json ./
RUN npm ci --no-audit --no-fund
COPY web/ ./
RUN npm run build

FROM golang:1.26.8-bookworm@sha256:a688600ca24f8a4d3ca77f95b0dd40704a9fc787c826660eb7ba0b641b8b175d AS go-build
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY cmd/ ./cmd/
COPY internal/ ./internal/
COPY VERSION ./VERSION
RUN go test ./...
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w -X main.version=$(cat VERSION)" -o /out/nextrole ./cmd/nextrole

FROM debian:bookworm-slim@sha256:3783cc01769c7b2b1b83a5c5ad96c815348e28ed7da68e2e3687004faa906251
RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates curl poppler-utils \
    && rm -rf /var/lib/apt/lists/* \
    && groupadd --gid 10001 nextrole \
    && useradd --uid 10001 --gid 10001 --no-create-home --home-dir /app --shell /usr/sbin/nologin nextrole
WORKDIR /app
COPY --from=go-build --chown=10001:10001 /out/nextrole /app/nextrole
COPY --from=web-build --chown=10001:10001 /build/web/dist /app/web/dist
COPY --chown=10001:10001 VERSION /app/VERSION
COPY --chown=10001:10001 THIRD_PARTY_NOTICES.txt /app/THIRD_PARTY_NOTICES.txt
USER 10001:10001
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=5s --start-period=30s --retries=3 CMD ["curl", "--fail", "--silent", "--max-time", "3", "http://127.0.0.1:8080/healthz"]
LABEL org.opencontainers.image.title="NextRole" \
      org.opencontainers.image.description="Offline-ready career transition simulator" \
      org.opencontainers.image.source="https://github.com/hkjang/nextRole"
ENTRYPOINT ["/app/nextrole"]
