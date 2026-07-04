package handlers

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/teran/mcp-paperless-ngx/application"
)

// Context keys for dependency injection.
type (
	docServiceCtxKey     struct{}
	corrServiceCtxKey    struct{}
	docTypeServiceCtxKey struct{}
	tagServiceCtxKey     struct{}
)

// ContextWithServices stores application services in context for retrieval
// by tool handlers at runtime.
func ContextWithServices(ctx context.Context, docSvc *application.DocumentService, corrSvc *application.CorrespondentService, docTypeSvc *application.DocumentTypeService, tagSvc *application.TagService) context.Context {
	ctx = context.WithValue(ctx, docServiceCtxKey{}, docSvc)
	ctx = context.WithValue(ctx, corrServiceCtxKey{}, corrSvc)
	ctx = context.WithValue(ctx, docTypeServiceCtxKey{}, docTypeSvc)
	ctx = context.WithValue(ctx, tagServiceCtxKey{}, tagSvc)
	return ctx
}

func DocServiceFromContext(ctx context.Context) *application.DocumentService {
	v, _ := ctx.Value(docServiceCtxKey{}).(*application.DocumentService)
	return v
}

func CorrServiceFromContext(ctx context.Context) *application.CorrespondentService {
	v, _ := ctx.Value(corrServiceCtxKey{}).(*application.CorrespondentService)
	return v
}

func DocTypeServiceFromContext(ctx context.Context) *application.DocumentTypeService {
	v, _ := ctx.Value(docTypeServiceCtxKey{}).(*application.DocumentTypeService)
	return v
}

func TagServiceFromContext(ctx context.Context) *application.TagService {
	v, _ := ctx.Value(tagServiceCtxKey{}).(*application.TagService)
	return v
}

// RegisterTools registers all MCP tools on the server.
// Each tool handler retrieves its required services from request context
// at runtime via the ContextWithServices chain set up by injectClientMiddleware.
func RegisterTools(s *mcp.Server) {
	mcp.AddTool(s, &mcp.Tool{ //nolint:exhaustruct
		Name:        "search_documents",
		Description: "Search documents with filters (query, correspondent, tags, date range).",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in SearchDocumentsInput) (*mcp.CallToolResult, SearchDocumentsOutput, error) {
		return NewSearchDocumentsHandler(DocServiceFromContext(ctx), CorrServiceFromContext(ctx), DocTypeServiceFromContext(ctx))(ctx, nil, in)
	})

	mcp.AddTool(s, &mcp.Tool{ //nolint:exhaustruct
		Name:        "get_document_content",
		Description: "Get full OCR text and metadata of a document.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in GetDocumentContentInput) (*mcp.CallToolResult, DocumentDetail, error) {
		return NewGetDocumentContentHandler(DocServiceFromContext(ctx), CorrServiceFromContext(ctx), DocTypeServiceFromContext(ctx))(ctx, nil, in)
	})

	mcp.AddTool(s, &mcp.Tool{ //nolint:exhaustruct
		Name:        "search_correspondents",
		Description: "Search correspondents by name.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in SearchCorrespondentsInput) (*mcp.CallToolResult, SearchCorrespondentsOutput, error) {
		return NewSearchCorrespondentsHandler(CorrServiceFromContext(ctx))(ctx, nil, in)
	})

	mcp.AddTool(s, &mcp.Tool{ //nolint:exhaustruct
		Name:        "get_documents_by_correspondent",
		Description: "List documents for a correspondent.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in GetDocumentsByCorrespondentInput) (*mcp.CallToolResult, SearchDocumentsOutput, error) {
		return NewGetDocumentsByCorrespondentHandler(DocServiceFromContext(ctx), CorrServiceFromContext(ctx), DocTypeServiceFromContext(ctx))(ctx, nil, in)
	})

	mcp.AddTool(s, &mcp.Tool{ //nolint:exhaustruct
		Name:        "list_tags",
		Description: "List all tags.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in ListTagsInput) (*mcp.CallToolResult, ListTagsOutput, error) {
		return NewListTagsHandler(TagServiceFromContext(ctx))(ctx, nil, in)
	})

	mcp.AddTool(s, &mcp.Tool{ //nolint:exhaustruct
		Name:        "get_documents_by_tag",
		Description: "List documents for a tag.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in GetDocumentsByTagInput) (*mcp.CallToolResult, SearchDocumentsOutput, error) {
		return NewGetDocumentsByTagHandler(DocServiceFromContext(ctx), CorrServiceFromContext(ctx), DocTypeServiceFromContext(ctx))(ctx, nil, in)
	})

	mcp.AddTool(s, &mcp.Tool{ //nolint:exhaustruct
		Name:        "fulltext_search",
		Description: "Full-text search across all documents.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in FulltextSearchInput) (*mcp.CallToolResult, FulltextSearchOutput, error) {
		return NewFulltextSearchHandler(DocServiceFromContext(ctx), CorrServiceFromContext(ctx), DocTypeServiceFromContext(ctx))(ctx, nil, in)
	})
}
