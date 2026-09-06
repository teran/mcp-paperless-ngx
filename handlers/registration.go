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

// boolPtr returns a pointer to the given bool value. It is required because
// mcp.ToolAnnotations uses *bool for the DestructiveHint and OpenWorldHint
// fields (to distinguish an explicit false from an omitted hint), while the
// ReadOnlyHint and IdempotentHint fields are plain bool.
func boolPtr(v bool) *bool {
	return &v
}

// readOnlyAnnotations returns the mcp.ToolAnnotations shared by every tool.
// All mcp-paperless-ngx tools are read-only, idempotent proxies into the
// closed Paperless-ngx domain: they never modify state (ReadOnlyHint), are
// safe to call repeatedly (IdempotentHint), perform no destructive updates
// (DestructiveHint=false) and do not interact with an open external world
// (OpenWorldHint=false).
func readOnlyAnnotations() *mcp.ToolAnnotations {
	return &mcp.ToolAnnotations{
		ReadOnlyHint:    true,
		IdempotentHint:  true,
		DestructiveHint: boolPtr(false),
		OpenWorldHint:   boolPtr(false),
	}
}

// RegisterTools registers all MCP tools on the server.
// Each tool handler retrieves its required services from request context
// at runtime via the ContextWithServices chain set up by injectClientMiddleware.
// If metrics is non-nil, each tool handler is wrapped with WrapToolHandler for
// per-tool Prometheus metrics (request count and duration).
func RegisterTools(s *mcp.Server, metrics *Metrics) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "search_documents",
		Title:       "Search documents",
		Description: "Search documents with filters (query, correspondent, tags, date range).",
		Annotations: readOnlyAnnotations(),
	}, WrapToolHandler(metrics, "search_documents", func(ctx context.Context, _ *mcp.CallToolRequest, in SearchDocumentsInput) (*mcp.CallToolResult, SearchDocumentsOutput, error) {
		return NewSearchDocumentsHandler(DocServiceFromContext(ctx), CorrServiceFromContext(ctx), DocTypeServiceFromContext(ctx))(ctx, nil, in)
	}))

	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_document_content",
		Title:       "Get document content",
		Description: "Get full OCR text and metadata of a document.",
		Annotations: readOnlyAnnotations(),
	}, WrapToolHandler(metrics, "get_document_content", func(ctx context.Context, _ *mcp.CallToolRequest, in GetDocumentContentInput) (*mcp.CallToolResult, DocumentDetail, error) {
		return NewGetDocumentContentHandler(DocServiceFromContext(ctx), CorrServiceFromContext(ctx), DocTypeServiceFromContext(ctx))(ctx, nil, in)
	}))

	mcp.AddTool(s, &mcp.Tool{
		Name:        "search_correspondents",
		Title:       "Search correspondents",
		Description: "Search correspondents by name.",
		Annotations: readOnlyAnnotations(),
	}, WrapToolHandler(metrics, "search_correspondents", func(ctx context.Context, _ *mcp.CallToolRequest, in SearchCorrespondentsInput) (*mcp.CallToolResult, SearchCorrespondentsOutput, error) {
		return NewSearchCorrespondentsHandler(CorrServiceFromContext(ctx))(ctx, nil, in)
	}))

	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_documents_by_correspondent",
		Title:       "Get documents by correspondent",
		Description: "List documents for a correspondent.",
		Annotations: readOnlyAnnotations(),
	}, WrapToolHandler(metrics, "get_documents_by_correspondent", func(ctx context.Context, _ *mcp.CallToolRequest, in GetDocumentsByCorrespondentInput) (*mcp.CallToolResult, SearchDocumentsOutput, error) {
		return NewGetDocumentsByCorrespondentHandler(DocServiceFromContext(ctx), CorrServiceFromContext(ctx), DocTypeServiceFromContext(ctx))(ctx, nil, in)
	}))

	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_tags",
		Title:       "List tags",
		Description: "List all tags.",
		Annotations: readOnlyAnnotations(),
	}, WrapToolHandler(metrics, "list_tags", func(ctx context.Context, _ *mcp.CallToolRequest, in ListTagsInput) (*mcp.CallToolResult, ListTagsOutput, error) {
		return NewListTagsHandler(TagServiceFromContext(ctx))(ctx, nil, in)
	}))

	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_documents_by_tag",
		Title:       "Get documents by tag",
		Description: "List documents for a tag.",
		Annotations: readOnlyAnnotations(),
	}, WrapToolHandler(metrics, "get_documents_by_tag", func(ctx context.Context, _ *mcp.CallToolRequest, in GetDocumentsByTagInput) (*mcp.CallToolResult, SearchDocumentsOutput, error) {
		return NewGetDocumentsByTagHandler(DocServiceFromContext(ctx), CorrServiceFromContext(ctx), DocTypeServiceFromContext(ctx))(ctx, nil, in)
	}))

	mcp.AddTool(s, &mcp.Tool{
		Name:        "fulltext_search",
		Title:       "Full-text search",
		Description: "Full-text search across all documents.",
		Annotations: readOnlyAnnotations(),
	}, WrapToolHandler(metrics, "fulltext_search", func(ctx context.Context, _ *mcp.CallToolRequest, in FulltextSearchInput) (*mcp.CallToolResult, FulltextSearchOutput, error) {
		return NewFulltextSearchHandler(DocServiceFromContext(ctx), CorrServiceFromContext(ctx), DocTypeServiceFromContext(ctx))(ctx, nil, in)
	}))
}
