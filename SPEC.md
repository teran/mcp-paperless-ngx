# MCP Paperless-ngx — Specification

## Overview

An MCP (Model Context Protocol) server for [Paperless-ngx](https://docs.paperless-ngx.com/).  
This server exposes Paperless-ngx functionality through the MCP protocol using **Streamable HTTP** transport (remote mode), allowing AI assistants to interact with a Paperless-ngx document management system.

## Key Differentiators

- **Remote (HTTP) transport** — uses MCP Streamable HTTP protocol; not stdio-bound.
- **Token from request headers** — the Paperless-ngx API token is read from the `Authorization` header of each MCP request, not from an environment variable. This enables per-user authentication in multi-tenant setups.

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

## Configuration (Environment Variables)

| Variable           | Required | Default | Description                          |
|--------------------|----------|---------|--------------------------------------|
| `PAPERLESS_URL`    | Yes      | —       | Base URL of the Paperless-ngx instance (e.g. `http://paperless:8000`) |
| `LISTEN_ADDR`      | No       | `:8080` | TCP address to listen on             |

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

The server applies three middleware layers to every HTTP request, executed in this order:

### 1. TokenMiddleware (`handlers/middleware.go`)

Extracts the Paperless-ngx API token from the `Authorization` header. Supports both `Bearer <token>` and `Token <token>` schemes (case-insensitive). The raw token string is stored in the request context. Returns **401 Unauthorized** if the header is missing or malformed.

### 2. BodyLimitMiddleware (`handlers/middleware.go`)

Limits the request body size to 1 MB using `http.MaxBytesReader`, preventing resource exhaustion from large requests. Applicable only to requests with a body (POST to `/mcp`).

### 3. injectClientMiddleware (`cmd/server/main.go`)

Retrieves the token from the context (placed there by `TokenMiddleware`). Creates the Paperless-ngx API client and four application services (`DocumentService`, `CorrespondentService`, `DocumentTypeService`, `TagService`), then stores them in the request context for downstream tool handlers. Returns **401 Unauthorized** if the token is absent from context.

No token verification is performed by this server — authentication and authorization are delegated entirely to the Paperless-ngx backend. The MCP server is a transparent proxy for the token.

## Authentication Flow

1. MCP Client sends a POST request to the Streamable HTTP endpoint.
2. The `Authorization: Bearer <paperless-api-token>` header is included in the request.
3. `TokenMiddleware` extracts the token from the header and stores it in the request context.
4. `BodyLimitMiddleware` enforces the 1 MB request body limit.
5. `injectClientMiddleware` creates a Paperless-ngx API client and builds application services, storing them in the context.
6. Tool handlers retrieve services from context and call the Paperless-ngx API using the token.
7. The token is never stored on the server — it exists only as long as the request is being processed.

## Error Handling

- **401 Unauthorized**: Missing or invalid `Authorization` header.
- **502 Bad Gateway**: Paperless-ngx instance unreachable or returns errors.
- **404 Not Found**: Requested resource does not exist.
- **400 Bad Request**: Invalid input parameters.
- Internal errors are returned with `isError: true` in the MCP response.

## Security Considerations

- The server does **not** store or cache tokens.
- The server must be deployed behind TLS in production.
- The `Authorization` header is read-only; it never appears in logs or error messages.
- No user management or session persistence is implemented — delegate to the MCP client layer.

## Development

### Prerequisites

- Go 1.26+
- golangci-lint (for linting)
- goreleaser (for building)

### Building

```bash
# Build for the current platform (binary)
goreleaser build --snapshot --clean
# The binary will be in dist/ (path depends on OS/arch)

# Build and push Docker image
docker buildx build --platform linux/amd64,linux/arm64 \
  -t ghcr.io/teran/mcp-paperless-ngx:latest --push .
```
