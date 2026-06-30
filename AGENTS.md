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
- **Responsible for**: Authentication, request routing, response formatting.

### Paperless-ngx
- **Role**: Document management backend.
- **Scope**: Stores, indexes, and retrieves documents.
- **API**: REST JSON API under `/api/`.

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

## Development Agents

When building or modifying this server, the following specialized agents may be involved:

| Agent       | Responsible For                                    |
|-------------|----------------------------------------------------|
| `architect` | High-level design decisions, system boundaries     |
| `developer` | Writing Go code, implementing tools and client     |
| `qa`        | Writing tests, verifying correctness               |
| `security`  | Reviewing auth flow, token handling, CVE scanning  |
| `code-review` | Reviewing merge requests before deployment      |
| `devops`    | CI/CD pipelines, Docker image, deployment          |

## Conflict Resolution

If multiple agents provide contradictory recommendations:

1. **Security first** — any recommendation that weakens the auth boundary is rejected.
2. **SPEC compliance** — the choice that best matches SPEC.md wins.
3. **Simplicity** — prefer the solution with fewer moving parts.
4. **Go idioms** — prefer standard library over external dependencies.

The final decision is recorded in the project TODO list by the orchestrating agent.
