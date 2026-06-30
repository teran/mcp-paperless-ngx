# mcp-paperless-ngx

**Remote MCP server for Paperless-ngx** — connects AI assistants to your Paperless-ngx document management system via the [Model Context Protocol](https://modelcontextprotocol.io/) over HTTP.

## Key Features

- **Remote (HTTP) transport** — uses MCP Streamable HTTP protocol, not stdio-bound
- **Per-request token authentication** — the Paperless-ngx API token is received from the `Authorization` header of every MCP request, not from an environment variable
- **7 MCP tools** for interacting with Paperless-ngx
- **Written in Go** using the official [MCP Go SDK](https://github.com/modelcontextprotocol/go-sdk)

## Architecture

```
┌──────────────────┐     MCP (Streamable HTTP)       ┌───────────────────────┐
│  MCP Client      │  ◄──────────────────────────►   │  mcp-paperless-ngx    │
│  (AI Assistant)  │    Authorization: Bearer <tok>  │  (Go server)          │
└──────────────────┘                                  └──────┬────────────────
                                                             │ HTTP (Token auth)
                                                             ▼
                                                  ┌───────────────────────┐
                                                  │  Paperless-ngx        │
                                                  │  REST API             │
                                                  └───────────────────────┘
```

## Configuration

| Environment Variable | Required | Default | Description                              |
|----------------------|----------|---------|------------------------------------------|
| `PAPERLESS_URL`      | Yes      | —       | Base URL of your Paperless-ngx instance  |
| `LISTEN_ADDR`        | No       | `:8080` | TCP address for the MCP server to listen |

The Paperless-ngx API token is supplied by the MCP client in the `Authorization` header:
```
Authorization: Bearer <your-paperless-api-token>
```

## MCP Tools

| Tool                          | Description                                              |
|-------------------------------|----------------------------------------------------------|
| `search_documents`            | Search documents with filters (query, correspondent, tags, date range) |
| `get_document_content`        | Retrieve full OCR text and metadata of a specific document |
| `search_correspondents`       | Search correspondents by name                            |
| `get_documents_by_correspondent` | List documents for a specific correspondent          |
| `list_tags`                   | List all tags, optionally filtered by name               |
| `get_documents_by_tag`        | List documents for a specific tag                        |
| `fulltext_search`             | Full-text search with score and highlights               |

## Quick Start

```bash
# 1. Set your Paperless-ngx instance URL
export PAPERLESS_URL=http://localhost:8000

# 2. Run the server
go run .

# 3. Configure your MCP client to connect:
#    Endpoint: http://localhost:8080/mcp
#    Headers:  Authorization: Bearer <your-paperless-api-token>
```

## Client Configuration Example

### Claude Desktop (claude.json)

```json
{
  "mcpServers": {
    "paperless-ngx": {
      "transport": "http",
      "url": "http://localhost:8080/mcp",
      "headers": {
        "Authorization": "Bearer your-api-token-here"
      }
    }
  }
}
```

## Build

```bash
go build -o mcp-paperless-ngx .
```

## Development

### Prerequisites

- Go 1.22+
- golangci-lint (for linting)

### Linting

```bash
golangci-lint run ./...
```

### Adding a new tool

1. Define input/output types in `internal/server/tools.go`
2. Write the handler function
3. Add `mcp.AddTool(server, &mcp.Tool{...}, handler)` in `RegisterTools()`

## Project Structure

```
.
├── main.go                    # Entry point with middleware & server setup
├── AGENTS.md                  # Agent documentation
├── SPEC.md                    # Full specification
├── .golangci.yml              # Linter configuration
├── internal/
│   ├── paperless/
│   │   ├── models.go          # Paperless-ngx data models
│   │   └── client.go          # Paperless-ngx HTTP client
│   └── server/
│       ├── context.go         # Context helpers for PaperlessClient
│       └── tools.go           # MCP tool definitions and handlers
```

## License

MIT
