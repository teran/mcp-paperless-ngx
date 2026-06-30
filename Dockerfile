FROM alpine:3 AS build
RUN apk add --no-cache ca-certificates && \
    printf '%s\n' \
      'root:x:0:0:root:/root:/bin/sh' \
      'nobody:x:65534:65534:nobody:/nonexistent:/sbin/nologin' > /tmp/passwd && \
    printf '%s\n' \
      'root:x:0:root' \
      'nobody:x:65534:' \
      'nogroup:x:65533:' > /tmp/group

FROM scratch
ARG TARGETARCH
ARG SUFFIX
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build /tmp/passwd /etc/passwd
COPY --from=build /tmp/group /etc/group
COPY dist/mcp-paperless-ngx_linux_${TARGETARCH}${SUFFIX}/mcp-paperless-ngx /mcp-paperless-ngx
USER nobody
ENTRYPOINT ["/mcp-paperless-ngx"]
