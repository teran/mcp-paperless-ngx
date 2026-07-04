FROM golang:1.26-alpine AS build
RUN apk add --no-cache ca-certificates
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /mcp-paperless-ngx ./cmd/server

FROM scratch
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build /mcp-paperless-ngx /mcp-paperless-ngx
EXPOSE 8080
ENTRYPOINT ["/mcp-paperless-ngx"]
LABEL org.opencontainers.image.source="https://github.com/teran/mcp-paperless-ngx"
LABEL org.opencontainers.image.description="Remote MCP server for Paperless-ngx"
LABEL org.opencontainers.image.licenses="MIT"
