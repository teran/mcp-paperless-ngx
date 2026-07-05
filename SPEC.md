# MCP Paperless-ngx — Specification

## Overview

An MCP (Model Context Protocol) server for [Paperless-ngx](https://docs.paperless-ngx.com/).  
This server exposes Paperless-ngx functionality through the MCP protocol using **Streamable HTTP** transport (remote mode), allowing AI assistants to interact with a Paperless-ngx document management system.

## Key Differentiators

- **Remote (HTTP) transport** — uses MCP Streamable HTTP protocol; not stdio-bound.
- **Token from request headers** — the Paperless-ngx API token is read from the `Authorization` header of each MCP request, not from an environment variable. This enables per-user authentication in multi-tenant setups.
- **Transparent token relay** — the MCP server never inspects or validates the token; it passes it through to Paperless-ngx, which handles all authentication and authorization.

## Architectural Notes

The server follows a standard hexagonal (ports & adapters) architecture with a middleware chain for cross-cutting concerns. Services are injected into request context and extracted by tool handlers at runtime — a pragmatic pattern for stateless MCP servers where handler registration is declarative. This is a normal and intentional design choice for a read-only proxy of this scale; no additional state management, caching, or orchestration layer is required.

## Architecture

```
┌──────────────────┐      MCP (Streamable HTTP)      ┌───────────────────────┐
│  MCP Client      │  ◄──────────────────────────►   │  mcp-paperless-ngx    │
│  (AI Assistant)  │     Authorization: Bearer <tok>  │  (Go server)          │
└──────────────────┘                                   └──────┬────────────────
                                                              │ HTTP (Token auth)
                                                              ▼
                                                   ┌───────────────────────┐
                                                   │  Paperless-ngx        │
                                                   │  REST API             │
                                                   └───────────────────────┘
```

## Technology Stack

| Component         | Choice                                                          |
|-------------------|-----------------------------------------------------------------|
| Language          | Go                                                              |
| MCP SDK           | `github.com/modelcontextprotocol/go-sdk`                        |
| Transport         | Streamable HTTP (MCP spec 2025-03-26+, remote-capable)          |
| HTTP Router       | `net/http` standard library + middleware pattern                |
| Tool Registration | `handlers/registration.go` — `RegisterTools()` function         |

## Configuration (Environment Variables)

| Variable               | Required | Default | Description                          |
|------------------------|----------|---------|--------------------------------------|
| `PAPERLESS_URL`        | Yes      | —       | Base URL of the Paperless-ngx instance (e.g. `http://paperless:8000`) |
| `LISTEN_ADDR`          | No       | `:8080` | TCP address to listen on             |
| `RATE_LIMIT_GLOBAL`    | No       | `100`   | Global rate limit (requests/second)  |
| `RATE_LIMIT_PER_CLIENT`| No       | `10`    | Per-client IP rate limit (requests/second) |
| `WRITE_TIMEOUT`        | No       | `300`   | HTTP write timeout in seconds (0 disables) |

The Paperless-ngx API token is **not** set via environment variables. It is supplied per-request in the `Authorization` header as `Bearer <token>` (the SDK also accepts `Token <token>` for Paperless-ngx compatibility).

The MCP server listens on the `/mcp` HTTP path via the Streamable HTTP handler.

## MCP Tools

### 1. `search_documents`

Search documents with configurable filters.

**Input**:

| Parameter          | Type   | Required | Description                                        |
|--------------------|--------|----------|----------------------------------------------------|
| `query`            | string | no       | Full-text search query                             |
| `correspondent_id` | int    | no       | Filter by correspondent ID                         |
| `tag_ids`          | []int  | no       | Filter by tag IDs (AND semantics — doc must have all) |
| `created_after`    | string | no       | Filter by creation date (ISO 8601, e.g. `2024-01-01`) |
| `created_before`   | string | no       | Filter by creation date (ISO 8601)                 |
| `page`             | int    | no       | Page number (default: 1)                           |
| `page_size`        | int    | no       | Results per page (default: 25, max: 100)           |

**Output**: Paginated list of document summaries (id, title, correspondent, correspondent_name, document_type, document_type_name, tags, created, mime_type, archive_serial_number, page_count).

---

### 2. `get_document_content`

Retrieve the full text content (OCR text) of a specific document.

**Input**:

| Parameter    | Type | Required | Description            |
|--------------|------|----------|------------------------|
| `document_id` | int  | yes      | ID of the document     |

**Output**: Full document details including OCR text, metadata, tags, correspondent name, document_type name, page count, archive serial number, original file name, creation date, modification date, and added date.

| Field                 | Type   | Description                                  |
|-----------------------|--------|----------------------------------------------|
| `id`                  | int    | Document ID                                  |
| `title`               | string | Document title                               |
| `content`             | string | Full OCR text content                        |
| `correspondent`       | *int   | Correspondent ID (nullable)                  |
| `correspondent_name`  | string | Correspondent display name (resolved)        |
| `document_type`       | *int   | Document type ID (nullable)                  |
| `document_type_name`  | string | Document type display name (resolved)        |
| `tags`                | []int  | Tag IDs                                      |
| `created`             | string | Creation date (ISO 8601)                     |
| `modified`            | string | Last modification date (ISO 8601)            |
| `added`               | string | Date added to Paperless-ngx (ISO 8601)       |
| `archive_serial_number` | *int | Archive serial number (nullable)             |
| `original_file_name`  | string | Original uploaded file name                  |
| `mime_type`           | string | MIME type (e.g. `application/pdf`)           |
| `page_count`          | *int   | Number of pages (nullable)                   |

---

### 3. `search_correspondents`

Search correspondents by name.

**Input**:

| Parameter | Type   | Required | Description                        |
|-----------|--------|----------|------------------------------------|
| `query`   | string | yes      | Name search query (substring match) |
| `page`    | int    | no       | Page number (default: 1)           |
| `page_size` | int  | no       | Results per page (default: 25)     |

**Output**: List of matching correspondents (id, name, slug, document_count).

---

### 4. `get_documents_by_correspondent`

List documents associated with a specific correspondent.

**Input**:

| Parameter         | Type | Required | Description                              |
|-------------------|------|----------|------------------------------------------|
| `correspondent_id` | int  | yes      | ID of the correspondent                  |
| `page`            | int  | no       | Page number (default: 1)                 |
| `page_size`       | int  | no       | Results per page (default: 25, max: 100) |

**Output**: Paginated list of documents for the given correspondent (id, title, correspondent, correspondent_name, document_type, document_type_name, tags, created, mime_type, archive_serial_number, page_count).

---

### 5. `list_tags`

Retrieve the full list of tags.

**Input**:

| Parameter   | Type   | Required | Description                            |
|-------------|--------|----------|----------------------------------------|
| `query`     | string | no       | Filter tags by name (substring match)  |
| `page`      | int    | no       | Page number (default: 1)               |
| `page_size` | int    | no       | Results per page (default: 25)         |

**Output**: List of tags (id, name, color, is_inbox_tag, document_count).

---

### 6. `get_documents_by_tag`

List documents associated with a specific tag.

**Input**:

| Parameter   | Type | Required | Description                              |
|-------------|------|----------|------------------------------------------|
| `tag_id`    | int  | yes      | ID of the tag                            |
| `page`      | int  | no       | Page number (default: 1)                 |
| `page_size` | int  | no       | Results per page (default: 25, max: 100) |

**Output**: Paginated list of documents for the given tag (id, title, correspondent, correspondent_name, document_type, document_type_name, tags, created, mime_type, archive_serial_number, page_count).

---

### 7. `fulltext_search`

Performs a full-text search across all documents.

**Input**:

| Parameter   | Type   | Required | Description                              |
|-------------|--------|----------|------------------------------------------|
| `query`     | string | yes      | Full-text search query                   |
| `page`      | int    | no       | Page number (default: 1)                 |
| `page_size` | int    | no       | Results per page (default: 25, max: 100) |

**Output**: Paginated list of document results with search highlights (id, title, correspondent, correspondent_name, document_type, document_type_name, tags, created, highlights).

---

## Middleware Chain

The server applies five middleware layers to every HTTP request, executed in this order (outermost first). The first two layers are the cheapest checks and prevent resource exhaustion before any body parsing or token extraction occurs.

### 1. RateLimitMiddleware (`handlers/ratelimit.go`)

The outermost middleware. Implements two-tier token-bucket rate limiting using `golang.org/x/time/rate`:
- **Global limit** (default 100 rps, configurable via `RATE_LIMIT_GLOBAL`) — prevents overall request flooding.
- **Per-client limit** (default 10 rps, configurable via `RATE_LIMIT_PER_CLIENT`) — per-IP limiting using `RemoteAddr` or proxy headers (`X-Client-IP` highest priority, then `X-Forwarded-For` first IP in chain).

Returns **429 Too Many Requests** when the limit is exceeded. Placed outermost because it is the cheapest check — no body reading or header parsing is required.

### 2. BodyLimitMiddleware (`handlers/middleware.go`)

Limits the request body size to 1 MB using `http.MaxBytesReader`, preventing resource exhaustion from large requests. Placed before `LoggingMiddleware` so that `io.ReadAll` in the logging layer is bounded — a maliciously large payload is rejected by `MaxBytesReader` before the logging middleware reads it into memory.

### 3. LoggingMiddleware (`handlers/middleware.go`)

Reads and buffers the request body to parse the JSON-RPC method name (and tool name for `tools/call` requests). Validates that batch JSON-RPC requests do not exceed `MaxBatchSize` (100), rejecting larger batches with **400 Bad Request** to prevent amplification attacks. Then measures request duration and captures the HTTP status code and response body size via a wrapped `ResponseWriter`. Logs a single line at INFO level with the `mcp_log` prefix:

```
INFO mcp_log method=<name> duration=<Go duration> req_size=<bytes> resp_size=<bytes> status=<HTTP code>
```

For `tools/call` requests, the `method` field reports the tool name (e.g. `search_documents`). The `Authorization` header (Bearer token) is **never** logged — only the MCP method name, timing, body sizes, and HTTP status code are recorded. All log strings are sanitized via `handlers.SanitizeLog` to remove control characters (0x00-0x1f, 0x7f) that could be used for log injection.

### 4. TokenMiddleware (`handlers/middleware.go`)

Extracts the Paperless-ngx API token from the `Authorization` header. Supports both `Bearer <token>` and `Token <token>` schemes (case-insensitive). The token length is validated against `MaxTokenLength` (512 bytes) to prevent DoS via oversized headers. The raw token string is stored in the request context. Returns **401 Unauthorized** if the header is missing, malformed, or exceeds the maximum length.

### 5. injectClientMiddleware (`cmd/server/main.go`)

Retrieves the token from the context (placed there by `TokenMiddleware`). Creates the Paperless-ngx API client using a shared `http.Client` with:
- **CheckRedirect** set to `http.ErrUseLastResponse` — prevents credential forwarding via redirects.
- **Shared Transport** with connection pooling (`MaxIdleConns=100`, `IdleConnTimeout=90s`).
- **LimitReader** (100 MB) on responses to prevent memory exhaustion from oversized OCR text.
Creates four application services (`DocumentService`, `CorrespondentService`, `DocumentTypeService`, `TagService`), then stores them in the request context for downstream tool handlers. Returns **401 Unauthorized** if the token is absent from context.

No token verification is performed by this server — authentication and authorization are delegated entirely to the Paperless-ngx backend. The MCP server is a transparent proxy for the token.

## Authentication Flow

1. MCP Client sends a POST request to the Streamable HTTP endpoint.
2. The `Authorization: Bearer <paperless-api-token>` header is included in the request.
3. `RateLimitMiddleware` checks the request against global and per-client rate limits. Returns **429 Too Many Requests** if exceeded.
4. `BodyLimitMiddleware` enforces the 1 MB request body limit, preventing large payloads from reaching downstream layers.
5. `LoggingMiddleware` reads and buffers the bounded body, records the MCP method name and request size, validates batch size (max 100), and starts the duration timer (without logging the token).
6. `TokenMiddleware` extracts the token from the header (validates length ≤ 512 bytes) and stores it in the request context.
7. `injectClientMiddleware` creates a Paperless-ngx API client with a shared `http.Client` (connection pooling, redirect protection, response body limit), and builds application services, storing them in the context.
8. Tool handlers retrieve services from context and call the Paperless-ngx API using the token.
9. On response, `LoggingMiddleware` emits the INFO log line with duration, response size, and HTTP status.
10. The token is never stored on the server — it exists only as long as the request is being processed.

## Error Handling

Errors occur at two distinct levels:

### HTTP Level (Middleware)

These errors are returned directly as HTTP status codes before the request reaches any tool handler:

| Status | Cause | Source |
|--------|-------|--------|
| **429 Too Many Requests** | Request frequency exceeds rate limit | `RateLimitMiddleware` |
| **401 Unauthorized** | Missing or malformed `Authorization` header | `TokenMiddleware` |
| **401 Unauthorized** | Token exceeds maximum length (512 bytes) | `TokenMiddleware` |
| **401 Unauthorized** | Token not found in request context | `injectClientMiddleware` |
| **400 Bad Request** | Batch JSON-RPC request exceeds maximum size (100) | `LoggingMiddleware` |

### MCP Level (Tool Handlers)

These errors are returned as `isError: true` in the MCP response (HTTP 200 with an error payload). The Paperless-ngx API response determines the error content:

| Scenario | MCP Error | Cause |
|----------|-----------|-------|
| Resource not found | `isError: true` | Paperless-ngx returns 404 (e.g. document ID does not exist) |
| Paperless-ngx unavailable | `isError: true` | Network error or Paperless-ngx returns 5xx |
| Invalid token | `isError: true` | Paperless-ngx returns 401 (token rejected by backend) |
| Invalid input parameters | `isError: true` | Parameter validation fails in the tool handler |

Authentication and authorization are handled entirely by the Paperless-ngx backend. Invalid tokens are rejected at the Paperless-ngx API level, not by this MCP server.

## Security Considerations

- The server does **not** store or cache tokens.
- The server must be deployed behind TLS in production.
- The `Authorization` header is read-only; it never appears in logs or error messages. The `LoggingMiddleware` explicitly avoids logging header content — only the MCP method name, timing, body sizes, and HTTP status code are recorded.
- Token length is enforced at 512 bytes maximum in `TokenMiddleware` to prevent DoS via oversized headers.
- HTTP redirects are disabled (`CheckRedirect: http.ErrUseLastResponse`) — the token cannot be forwarded to an external URL via a 302 response from Paperless-ngx.
- Response bodies from Paperless-ngx are limited to 100 MB via `io.LimitReader` to prevent memory exhaustion from oversized OCR text.
- Global (100 rps) and per-client (10 rps) rate limiting via `RateLimitMiddleware` — configurable via environment variables.
- Batch JSON-RPC requests are limited to 100 items per batch to prevent amplification attacks.
- All log strings are sanitized via `handlers.SanitizeLog` — control characters (0x00-0x1f, 0x7f) are stripped to prevent log injection.
- No user management or session persistence is implemented — delegate to the MCP client layer.

## Development

### Prerequisites

- Go 1.26+
- golangci-lint (for linting)
- goreleaser (highly recommended for building/releasing)
- gremlins (for mutation testing, optional)

### Building

```bash
# Quick build (no extra tools required)
go build -o mcp-paperless-ngx ./cmd/server

# Release build using goreleaser (recommended)
goreleaser build --snapshot --clean
# The binary will be in dist/ (path depends on OS/arch)

# Build and push Docker image
docker buildx build --platform linux/amd64,linux/arm64 \
  -t ghcr.io/teran/mcp-paperless-ngx:latest --push .
```

### Quality gates (CI pipeline)

Every commit on any branch is checked by three workflows:

1. **golangci-lint** — static analysis with `gosec` enabled.
2. **go test** — unit tests with coverage profile.
3. **gremlins unleash** — mutation testing (informational, does not block).

### Linting

```bash
golangci-lint run ./...
```

The linter configuration (`.golangci.yml`) enables `gosec` for security-relevant static analysis. All linters pass with zero issues.

### Running tests

```bash
go test -count=1 ./...
```

### Test coverage

```bash
go test -coverprofile=coverage.out -count=1 ./...
go tool cover -func=coverage.out
```

Current coverage by package:

| Package                     | Coverage |
|-----------------------------|----------|
| `application`               | 100.0%   |
| `cmd/server`                | 28.6%    |
| `config`                    | 100.0%   |
| `domain`                    | no stmts |
| `handlers`                  | 92.3%    |
| `infrastructure/paperless`  | 93.0%    |

### Mutation testing (gremlins)

[Mutation testing](https://en.wikipedia.org/wiki/Mutation_testing) evaluates test quality by introducing small changes (mutations) into the source code and checking whether the test suite catches them. The mutation testing workflow runs on every commit:

```bash
# Install gremlins (one-time)
go install github.com/go-gremlins/gremlins/cmd/gremlins@latest

# Run mutation testing on packages with high coverage
gremlins unleash handlers application infrastructure/paperless config
```

Current mutation testing results are informational (not blocking) — the project has no KILLED mutants because all covered mutants result in TIMED OUT (condition negation changes test timing/retry behaviour). This is typical for projects with HTTP handler and network tests. Mutation coverage will improve as more edge-case tests are added.

### Adding a new tool

1. Define input/output types in `handlers/tools.go`
2. Write the handler factory function in `handlers/tools.go`
3. Register the tool via `RegisterTools()` in `handlers/registration.go`
4. If a new domain entity is needed, define it in `domain/` and add a repository interface
5. If a new service is needed, wire it in `injectClientMiddleware` (`cmd/server/main.go`)

### Dependency Management

Dependencies are updated automatically via [Dependabot](https://docs.github.com/code-security/dependabot) (`.github/dependabot.yml`):
- Go module dependencies — checked weekly
- Docker base image (`golang:1.26-alpine`) — checked weekly
- GitHub Actions — checked weekly
