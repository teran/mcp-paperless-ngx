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

// services bundles the application services required by the tool handlers.
type services struct {
	doc     *application.DocumentService
	corr    *application.CorrespondentService
	docType *application.DocumentTypeService
	tag     *application.TagService
}

// registerTools registers all MCP tools on the server. It is the single
// source of truth for tool registration: the required application services
// are resolved through the provided resolver, so callers can source them from
// the request context (RegisterTools) or from explicit arguments
// (RegisterToolsWithServices).
// If metrics is non-nil, each tool handler is wrapped with WrapToolHandler for
// per-tool Prometheus metrics (request count and duration).
func registerTools(s *mcp.Server, metrics *Metrics, resolve func(ctx context.Context) services) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "search_documents",
		Title:       "Search documents",
		Description: "Search documents with filters (query, correspondent, tags, date range).",
		Annotations: readOnlyAnnotations(),
	}, WrapToolHandler(metrics, "search_documents", func(ctx context.Context, _ *mcp.CallToolRequest, in SearchDocumentsInput) (*mcp.CallToolResult, SearchDocumentsOutput, error) {
		svc := resolve(ctx)
		return NewSearchDocumentsHandler(svc.doc, svc.corr, svc.docType)(ctx, nil, in)
	}))

	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_document_content",
		Title:       "Get document content",
		Description: "Get full OCR text and metadata of a document.",
		Annotations: readOnlyAnnotations(),
	}, WrapToolHandler(metrics, "get_document_content", func(ctx context.Context, _ *mcp.CallToolRequest, in GetDocumentContentInput) (*mcp.CallToolResult, DocumentDetail, error) {
		svc := resolve(ctx)
		return NewGetDocumentContentHandler(svc.doc, svc.corr, svc.docType)(ctx, nil, in)
	}))

	mcp.AddTool(s, &mcp.Tool{
		Name:        "search_correspondents",
		Title:       "Search correspondents",
		Description: "Search correspondents by name.",
		Annotations: readOnlyAnnotations(),
	}, WrapToolHandler(metrics, "search_correspondents", func(ctx context.Context, _ *mcp.CallToolRequest, in SearchCorrespondentsInput) (*mcp.CallToolResult, SearchCorrespondentsOutput, error) {
		svc := resolve(ctx)
		return NewSearchCorrespondentsHandler(svc.corr)(ctx, nil, in)
	}))

	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_documents_by_correspondent",
		Title:       "Get documents by correspondent",
		Description: "List documents for a correspondent.",
		Annotations: readOnlyAnnotations(),
	}, WrapToolHandler(metrics, "get_documents_by_correspondent", func(ctx context.Context, _ *mcp.CallToolRequest, in GetDocumentsByCorrespondentInput) (*mcp.CallToolResult, SearchDocumentsOutput, error) {
		svc := resolve(ctx)
		return NewGetDocumentsByCorrespondentHandler(svc.doc, svc.corr, svc.docType)(ctx, nil, in)
	}))

	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_tags",
		Title:       "List tags",
		Description: "List all tags.",
		Annotations: readOnlyAnnotations(),
	}, WrapToolHandler(metrics, "list_tags", func(ctx context.Context, _ *mcp.CallToolRequest, in ListTagsInput) (*mcp.CallToolResult, ListTagsOutput, error) {
		svc := resolve(ctx)
		return NewListTagsHandler(svc.tag)(ctx, nil, in)
	}))

	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_documents_by_tag",
		Title:       "Get documents by tag",
		Description: "List documents for a tag.",
		Annotations: readOnlyAnnotations(),
	}, WrapToolHandler(metrics, "get_documents_by_tag", func(ctx context.Context, _ *mcp.CallToolRequest, in GetDocumentsByTagInput) (*mcp.CallToolResult, SearchDocumentsOutput, error) {
		svc := resolve(ctx)
		return NewGetDocumentsByTagHandler(svc.doc, svc.corr, svc.docType)(ctx, nil, in)
	}))

	mcp.AddTool(s, &mcp.Tool{
		Name:        "fulltext_search",
		Title:       "Full-text search",
		Description: "Full-text search across all documents.",
		Annotations: readOnlyAnnotations(),
	}, WrapToolHandler(metrics, "fulltext_search", func(ctx context.Context, _ *mcp.CallToolRequest, in FulltextSearchInput) (*mcp.CallToolResult, FulltextSearchOutput, error) {
		svc := resolve(ctx)
		return NewFulltextSearchHandler(svc.doc, svc.corr, svc.docType)(ctx, nil, in)
	}))
}

// RegisterTools registers all MCP tools on the server.
// Each tool handler retrieves its required services from request context
// at runtime via the ContextWithServices chain set up by injectClientMiddleware.
func RegisterTools(s *mcp.Server, metrics *Metrics) {
	registerTools(s, metrics, func(ctx context.Context) services {
		return services{
			doc:     DocServiceFromContext(ctx),
			corr:    CorrServiceFromContext(ctx),
			docType: DocTypeServiceFromContext(ctx),
			tag:     TagServiceFromContext(ctx),
		}
	})
}

// RegisterToolsWithServices registers all MCP tools on the server, wiring
// them with the explicitly provided application services. Unlike
// RegisterTools, it ignores the request context and uses the supplied
// services directly (e.g. for in-memory and end-to-end tests).
func RegisterToolsWithServices(
	s *mcp.Server,
	metrics *Metrics,
	docSvc *application.DocumentService,
	corrSvc *application.CorrespondentService,
	docTypeSvc *application.DocumentTypeService,
	tagSvc *application.TagService,
) {
	registerTools(s, metrics, func(context.Context) services {
		return services{
			doc:     docSvc,
			corr:    corrSvc,
			docType: docTypeSvc,
			tag:     tagSvc,
		}
	})
}
