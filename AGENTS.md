# AGENTS.md — Agent Documentation

## Overview

This document describes the agents/assistants involved in the development and operation of `mcp-paperless-ngx`. Each agent has a specific role, scope of responsibility, and set of tools available.

## Agent Roles

### User Agent
- **Role**: End-user interacting via an MCP-compatible AI assistant (e.g., Claude, Copilot).
- **Scope**: Sends natural-language queries that get translated into MCP tool calls.
- **No direct access** to Paperless-ngx API.

### MCP Server (`mcp-paperless-ngx`)
- **Role**: Mediator between the AI assistant and Paperless-ngx.
- **Scope**: Translates MCP tool invocations into Paperless-ngx REST API calls.
- **Responsible for**: Transparent token relay, request routing, response formatting.
- **Does not validate tokens** — authentication and authorization are delegated entirely to the Paperless-ngx backend.

### Paperless-ngx
- **Role**: Document management backend.
- **Scope**: Stores, indexes, and retrieves documents.
- **API**: REST JSON API under `/api/`.

## Package Layout

| Package / File                              | Purpose                                         |
|---------------------------------------------|-------------------------------------------------|
| `cmd/server/main.go`                        | Entrypoint, HTTP server, middleware wiring      |
| `config/config.go`                          | Configuration loading (`envconfig` + ozzo-validation) |
| `handlers/middleware.go`                    | Token extraction, body limit, logging, batch validation middleware |
| `handlers/ratelimit.go`                     | Rate limiting middleware (global + per-client)  |
| `handlers/tools.go`                         | MCP tool handler factories + I/O types          |
| `handlers/registration.go`                  | Tool registration via `RegisterTools()`         |
| `application/service.go`                    | Business logic / use case layer                 |
| `domain/`                                   | Domain models + repository interfaces (ports)   |
| `infrastructure/paperless/client.go`        | Paperless-ngx HTTP API client (adapters)        |
| `infrastructure/paperless/models.go`        | JSON wire models + `toDomain()` conversion      |

## Tool-to-Agent Mapping

| MCP Tool                  | Agent Role                   | Paperless-ngx Endpoint                     |
|---------------------------|------------------------------|--------------------------------------------|
| `search_documents`        | MCP Server                   | `GET /api/documents/`                      |
| `get_document_content`    | MCP Server                   | `GET /api/documents/{id}/`                 |
| `search_correspondents`   | MCP Server                   | `GET /api/correspondents/`                 |
| `get_documents_by_correspondent` | MCP Server            | `GET /api/documents/?correspondent__id=`   |
| `list_tags`               | MCP Server                   | `GET /api/tags/`                           |
| `get_documents_by_tag`    | MCP Server                   | `GET /api/documents/?tags__id__all=`       |
| `fulltext_search`         | MCP Server                   | `GET /api/documents/?query=`               |

## CI Pipeline

Every commit on any branch is checked by three workflows:

1. **golangci-lint** — static analysis with `gosec` enabled.
2. **go test** — unit tests with coverage profile (uploaded as artifact).
3. **gremlins unleash** — mutation testing on packages with highest coverage (`handlers`, `application`, `infrastructure/paperless`, `config`). Runs as `continue-on-error` — informational only, does not block the PR.

Workflow files:
- `.github/workflows/ci.yml` — lint + test + coverage upload
- `.github/workflows/gremlins.yml` — mutation testing

## Development Agents

When building or modifying this server, the following specialized agents may be involved:

| Agent       | Responsible For                                    |
|-------------|----------------------------------------------------|
| `architect` | High-level design decisions, system boundaries     |
| `developer` | Writing Go code, implementing tools and client     |
| `qa`        | Writing tests, verifying correctness               |
| `security`  | Reviewing auth flow, token handling, CVE scanning  |
| `code-review` | Reviewing merge requests before deployment      |
| `devops`    | CI/CD pipelines, Docker image, deployment, mutation testing |

## Conflict Resolution

If multiple agents provide contradictory recommendations:

1. **Security first** — any recommendation that weakens the auth boundary is rejected.
2. **SPEC compliance** — the choice that best matches SPEC.md wins.
3. **Simplicity** — prefer the solution with fewer moving parts.
4. **Go idioms** — prefer standard library over external dependencies.

The final decision is recorded in the project TODO list by the orchestrating agent.
