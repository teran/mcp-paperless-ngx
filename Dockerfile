# Uses a pre-built mcp-paperless-ngx binary from goreleaser.
# Usage:
#   goreleaser build --snapshot --clean
#   cp dist/mcp-paperless-ngx_linux_amd64_v1/mcp-paperless-ngx mcp-paperless-ngx-linux-amd64
#   cp dist/mcp-paperless-ngx_linux_arm64_v8.0/mcp-paperless-ngx mcp-paperless-ngx-linux-arm64
#   docker buildx build --platform linux/amd64,linux/arm64 -t image:tag .

FROM alpine:latest AS base
RUN apk add --no-cache ca-certificates && \
    echo 'nobody:x:65534:65534:nobody:/:/sbin/nologin' > /etc/passwd-minimal

FROM scratch
ARG TARGETARCH
COPY --from=base /etc/passwd-minimal /etc/passwd
COPY --from=base /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY mcp-paperless-ngx-linux-${TARGETARCH} /mcp-paperless-ngx
USER 65534:65534
EXPOSE 8080
ENTRYPOINT ["/mcp-paperless-ngx"]
LABEL org.opencontainers.image.source="https://github.com/teran/mcp-paperless-ngx"
LABEL org.opencontainers.image.description="Remote MCP server for Paperless-ngx"
LABEL org.opencontainers.image.licenses="Apache-2.0"
