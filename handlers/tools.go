package handlers

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"golang.org/x/sync/errgroup"

	"github.com/teran/mcp-paperless-ngx/application"
	"github.com/teran/mcp-paperless-ngx/domain"
)

// ============================================================
// Input / output types
// ============================================================

// --- search_documents ---

type SearchDocumentsInput struct {
	Query           string `json:"query,omitempty" jsonschema:"full-text search query"`
	CorrespondentID int    `json:"correspondent_id,omitempty" jsonschema:"filter by correspondent ID"`
	TagIDs          []int  `json:"tag_ids,omitempty" jsonschema:"filter by tag IDs (document must have all specified tags)"`
	CreatedAfter    string `json:"created_after,omitempty" jsonschema:"filter by creation date (ISO 8601, e.g. 2024-01-01)"`
	CreatedBefore   string `json:"created_before,omitempty" jsonschema:"filter by creation date (ISO 8601)"`
	Page            int    `json:"page,omitempty" jsonschema:"page number (starts at 1),default=1"`
	PageSize        int    `json:"page_size,omitempty" jsonschema:"results per page (max 100),default=25"`
}

type DocumentSummary struct {
	ID                int    `json:"id"`
	Title             string `json:"title"`
	Correspondent     *int   `json:"correspondent,omitempty"`
	CorrespondentName string `json:"correspondent_name,omitempty"`
	DocumentType      *int   `json:"document_type,omitempty"`
	DocumentTypeName  string `json:"document_type_name,omitempty"`
	Tags              []int  `json:"tags,omitempty"`
	Created           string `json:"created"`
	MimeType          string `json:"mime_type"`
	ArchiveSerial     *int   `json:"archive_serial_number,omitempty"`
	PageCount         *int   `json:"page_count,omitempty"`
}

type SearchDocumentsOutput struct {
	Total   int               `json:"total"`
	Page    int               `json:"page"`
	Results []DocumentSummary `json:"results"`
}

// --- get_document_content ---

type GetDocumentContentInput struct {
	DocumentID int `json:"document_id" jsonschema:"the ID of the document to retrieve,required"`
}

type DocumentDetail struct {
	ID                  int    `json:"id"`
	Title               string `json:"title"`
	Content             string `json:"content"`
	Correspondent       *int   `json:"correspondent,omitempty"`
	CorrespondentName   string `json:"correspondent_name,omitempty"`
	DocumentType        *int   `json:"document_type,omitempty"`
	DocumentTypeName    string `json:"document_type_name,omitempty"`
	Tags                []int  `json:"tags,omitempty"`
	Created             string `json:"created"`
	Modified            string `json:"modified"`
	Added               string `json:"added"`
	ArchiveSerialNumber *int   `json:"archive_serial_number,omitempty"`
	OriginalFileName    string `json:"original_file_name"`
	MimeType            string `json:"mime_type"`
	PageCount           *int   `json:"page_count,omitempty"`
}

// --- search_correspondents ---

type SearchCorrespondentsInput struct {
	Query    string `json:"query" jsonschema:"name search query (substring match),required"`
	Page     int    `json:"page,omitempty" jsonschema:"page number (starts at 1),default=1"`
	PageSize int    `json:"page_size,omitempty" jsonschema:"results per page (max 100),default=25"`
}

type CorrespondentSummary struct {
	ID            int    `json:"id"`
	Name          string `json:"name"`
	DocumentCount int    `json:"document_count"`
	Slug          string `json:"slug"`
}

type SearchCorrespondentsOutput struct {
	Total   int                    `json:"total"`
	Page    int                    `json:"page"`
	Results []CorrespondentSummary `json:"results"`
}

// --- get_documents_by_correspondent ---

type GetDocumentsByCorrespondentInput struct {
	CorrespondentID int `json:"correspondent_id" jsonschema:"ID of the correspondent,required"`
	Page            int `json:"page,omitempty" jsonschema:"page number (starts at 1),default=1"`
	PageSize        int `json:"page_size,omitempty" jsonschema:"results per page (max 100),default=25"`
}

// --- list_tags ---

type ListTagsInput struct {
	Query    string `json:"query,omitempty" jsonschema:"filter tags by name (substring match)"`
	Page     int    `json:"page,omitempty" jsonschema:"page number (starts at 1),default=1"`
	PageSize int    `json:"page_size,omitempty" jsonschema:"results per page (max 100),default=25"`
}

type TagSummary struct {
	ID            int    `json:"id"`
	Name          string `json:"name"`
	Color         string `json:"color"`
	IsInboxTag    bool   `json:"is_inbox_tag"`
	DocumentCount int    `json:"document_count"`
}

type ListTagsOutput struct {
	Total   int          `json:"total"`
	Page    int          `json:"page"`
	Results []TagSummary `json:"results"`
}

// --- get_documents_by_tag ---

type GetDocumentsByTagInput struct {
	TagID    int `json:"tag_id" jsonschema:"ID of the tag,required"`
	Page     int `json:"page,omitempty" jsonschema:"page number (starts at 1),default=1"`
	PageSize int `json:"page_size,omitempty" jsonschema:"results per page (max 100),default=25"`
}

// --- fulltext_search ---

type FulltextSearchInput struct {
	Query    string `json:"query" jsonschema:"full-text search query,required"`
	Page     int    `json:"page,omitempty" jsonschema:"page number (starts at 1),default=1"`
	PageSize int    `json:"page_size,omitempty" jsonschema:"results per page (max 100),default=25"`
}

type FulltextSearchResultItem struct {
	ID                int    `json:"id"`
	Title             string `json:"title"`
	Correspondent     *int   `json:"correspondent,omitempty"`
	CorrespondentName string `json:"correspondent_name,omitempty"`
	DocumentType      *int   `json:"document_type,omitempty"`
	DocumentTypeName  string `json:"document_type_name,omitempty"`
	Tags              []int  `json:"tags,omitempty"`
	Created           string `json:"created"`
	Highlights        string `json:"highlights,omitempty"`
}

type FulltextSearchOutput struct {
	Total   int                        `json:"total"`
	Page    int                        `json:"page"`
	Results []FulltextSearchResultItem `json:"results"`
}

// ============================================================
// Helpers
// ============================================================

func normalizePagination(page, pageSize int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 25
	}
	return page, pageSize
}

func toDocumentSummaries(docs []domain.Document, corrNames map[int]string, docTypeNames map[int]string) []DocumentSummary {
	result := make([]DocumentSummary, 0, len(docs))
	for _, doc := range docs {
		summary := DocumentSummary{ //nolint:exhaustruct
			ID:            doc.ID,
			Title:         doc.Title,
			Correspondent: doc.Correspondent,
			DocumentType:  doc.DocumentType,
			Tags:          doc.Tags,
			Created:       doc.Created,
			MimeType:      doc.MimeType,
			ArchiveSerial: doc.ArchiveSerialNumber,
			PageCount:     doc.PageCount,
		}
		if doc.Correspondent != nil {
			summary.CorrespondentName = corrNames[*doc.Correspondent]
		}
		if doc.DocumentType != nil {
			summary.DocumentTypeName = docTypeNames[*doc.DocumentType]
		}
		result = append(result, summary)
	}
	return result
}

// resolveCorrespondentNames fetches names for all unique correspondent IDs
// present in the documents and returns a map of ID → name.
// Uses errgroup for concurrent resolution to avoid N+1 sequential HTTP calls.
func resolveCorrespondentNames(ctx context.Context, corrSvc *application.CorrespondentService, docs []domain.Document) map[int]string {
	ids := make(map[int]struct{})
	for _, doc := range docs {
		if doc.Correspondent != nil {
			ids[*doc.Correspondent] = struct{}{}
		}
	}
	if len(ids) == 0 {
		return nil
	}

	var mu sync.Mutex
	names := make(map[int]string, len(ids))

	g, ctx := errgroup.WithContext(ctx)
	for id := range ids {
		id := id
		g.Go(func() error {
			corr, err := corrSvc.GetByID(ctx, id)
			if err != nil {
				log.Printf("resolve correspondent name: id=%d: %v", id, err)
				return nil // best-effort: swallow error
			}
			if corr != nil {
				mu.Lock()
				names[id] = corr.Name
				mu.Unlock()
			}
			return nil
		})
	}
	_ = g.Wait()

	return names
}

// resolveDocumentTypeNames fetches names for all unique document type IDs
// present in the documents and returns a map of ID → name.
// Uses errgroup for concurrent resolution to avoid N+1 sequential HTTP calls.
func resolveDocumentTypeNames(ctx context.Context, docTypeSvc *application.DocumentTypeService, docs []domain.Document) map[int]string {
	ids := make(map[int]struct{})
	for _, doc := range docs {
		if doc.DocumentType != nil {
			ids[*doc.DocumentType] = struct{}{}
		}
	}
	if len(ids) == 0 {
		return nil
	}

	var mu sync.Mutex
	names := make(map[int]string, len(ids))

	g, ctx := errgroup.WithContext(ctx)
	for id := range ids {
		id := id
		g.Go(func() error {
			dt, err := docTypeSvc.GetByID(ctx, id)
			if err != nil {
				log.Printf("resolve document type name: id=%d: %v", id, err)
				return nil // best-effort: swallow error
			}
			if dt != nil {
				mu.Lock()
				names[id] = dt.Name
				mu.Unlock()
			}
			return nil
		})
	}
	_ = g.Wait()

	return names
}

// ============================================================
// Tool handler factories
// ============================================================

// NewSearchDocumentsHandler creates a handler for search_documents.
func NewSearchDocumentsHandler(svc *application.DocumentService, corrSvc *application.CorrespondentService, docTypeSvc *application.DocumentTypeService) mcp.ToolHandlerFor[SearchDocumentsInput, SearchDocumentsOutput] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input SearchDocumentsInput) (*mcp.CallToolResult, SearchDocumentsOutput, error) {
		page, pageSize := normalizePagination(input.Page, input.PageSize)

		result, err := svc.Search(ctx, domain.SearchDocumentsParams{
			Query:           input.Query,
			CorrespondentID: input.CorrespondentID,
			TagIDs:          input.TagIDs,
			CreatedAfter:    input.CreatedAfter,
			CreatedBefore:   input.CreatedBefore,
			Page:            page,
			PageSize:        pageSize,
		})
		if err != nil {
			return nil, SearchDocumentsOutput{}, fmt.Errorf("search documents: %w", err)
		}

		corrNames := resolveCorrespondentNames(ctx, corrSvc, result.Results)
		docTypeNames := resolveDocumentTypeNames(ctx, docTypeSvc, result.Results)

		return nil, SearchDocumentsOutput{
			Total:   result.Total,
			Page:    page,
			Results: toDocumentSummaries(result.Results, corrNames, docTypeNames),
		}, nil
	}
}

// NewGetDocumentContentHandler creates a handler for get_document_content.
func NewGetDocumentContentHandler(svc *application.DocumentService, corrSvc *application.CorrespondentService, docTypeSvc *application.DocumentTypeService) mcp.ToolHandlerFor[GetDocumentContentInput, DocumentDetail] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input GetDocumentContentInput) (*mcp.CallToolResult, DocumentDetail, error) {
		doc, err := svc.GetByID(ctx, input.DocumentID)
		if err != nil {
			return nil, DocumentDetail{}, fmt.Errorf("get document content: %w", err)
		}

		corrName := ""
		if doc.Correspondent != nil {
			if corr, err := corrSvc.GetByID(ctx, *doc.Correspondent); err == nil && corr != nil {
				corrName = corr.Name
			}
		}

		docTypeName := ""
		if doc.DocumentType != nil {
			if dt, err := docTypeSvc.GetByID(ctx, *doc.DocumentType); err == nil && dt != nil {
				docTypeName = dt.Name
			}
		}

		return nil, DocumentDetail{
			ID:                  doc.ID,
			Title:               doc.Title,
			Content:             doc.Content,
			Correspondent:       doc.Correspondent,
			CorrespondentName:   corrName,
			DocumentType:        doc.DocumentType,
			DocumentTypeName:    docTypeName,
			Tags:                doc.Tags,
			Created:             doc.Created,
			Modified:            doc.Modified,
			Added:               doc.Added,
			ArchiveSerialNumber: doc.ArchiveSerialNumber,
			OriginalFileName:    doc.OriginalFileName,
			MimeType:            doc.MimeType,
			PageCount:           doc.PageCount,
		}, nil
	}
}

// NewSearchCorrespondentsHandler creates a handler for search_correspondents.
func NewSearchCorrespondentsHandler(svc *application.CorrespondentService) mcp.ToolHandlerFor[SearchCorrespondentsInput, SearchCorrespondentsOutput] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input SearchCorrespondentsInput) (*mcp.CallToolResult, SearchCorrespondentsOutput, error) {
		page, pageSize := normalizePagination(input.Page, input.PageSize)

		result, err := svc.Search(ctx, input.Query, page, pageSize)
		if err != nil {
			return nil, SearchCorrespondentsOutput{}, fmt.Errorf("search correspondents: %w", err)
		}

		items := make([]CorrespondentSummary, 0, len(result.Results))
		for _, c := range result.Results {
			items = append(items, CorrespondentSummary{
				ID:            c.ID,
				Name:          c.Name,
				DocumentCount: c.DocumentCount,
				Slug:          c.Slug,
			})
		}

		return nil, SearchCorrespondentsOutput{
			Total:   result.Total,
			Page:    page,
			Results: items,
		}, nil
	}
}

// NewGetDocumentsByCorrespondentHandler creates a handler for get_documents_by_correspondent.
func NewGetDocumentsByCorrespondentHandler(svc *application.DocumentService, corrSvc *application.CorrespondentService, docTypeSvc *application.DocumentTypeService) mcp.ToolHandlerFor[GetDocumentsByCorrespondentInput, SearchDocumentsOutput] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input GetDocumentsByCorrespondentInput) (*mcp.CallToolResult, SearchDocumentsOutput, error) {
		page, pageSize := normalizePagination(input.Page, input.PageSize)

		result, err := svc.GetByCorrespondent(ctx, input.CorrespondentID, page, pageSize)
		if err != nil {
			return nil, SearchDocumentsOutput{}, fmt.Errorf("get documents by correspondent: %w", err)
		}

		corrNames := resolveCorrespondentNames(ctx, corrSvc, result.Results)
		docTypeNames := resolveDocumentTypeNames(ctx, docTypeSvc, result.Results)

		return nil, SearchDocumentsOutput{
			Total:   result.Total,
			Page:    page,
			Results: toDocumentSummaries(result.Results, corrNames, docTypeNames),
		}, nil
	}
}

// NewListTagsHandler creates a handler for list_tags.
func NewListTagsHandler(svc *application.TagService) mcp.ToolHandlerFor[ListTagsInput, ListTagsOutput] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input ListTagsInput) (*mcp.CallToolResult, ListTagsOutput, error) {
		page, pageSize := normalizePagination(input.Page, input.PageSize)

		result, err := svc.List(ctx, input.Query, page, pageSize)
		if err != nil {
			return nil, ListTagsOutput{}, fmt.Errorf("list tags: %w", err)
		}

		items := make([]TagSummary, 0, len(result.Results))
		for _, t := range result.Results {
			items = append(items, TagSummary{
				ID:            t.ID,
				Name:          t.Name,
				Color:         t.Color,
				IsInboxTag:    t.IsInboxTag,
				DocumentCount: t.DocumentCount,
			})
		}

		return nil, ListTagsOutput{
			Total:   result.Total,
			Page:    page,
			Results: items,
		}, nil
	}
}

// NewGetDocumentsByTagHandler creates a handler for get_documents_by_tag.
func NewGetDocumentsByTagHandler(svc *application.DocumentService, corrSvc *application.CorrespondentService, docTypeSvc *application.DocumentTypeService) mcp.ToolHandlerFor[GetDocumentsByTagInput, SearchDocumentsOutput] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input GetDocumentsByTagInput) (*mcp.CallToolResult, SearchDocumentsOutput, error) {
		page, pageSize := normalizePagination(input.Page, input.PageSize)

		result, err := svc.GetByTag(ctx, input.TagID, page, pageSize)
		if err != nil {
			return nil, SearchDocumentsOutput{}, fmt.Errorf("get documents by tag: %w", err)
		}

		corrNames := resolveCorrespondentNames(ctx, corrSvc, result.Results)
		docTypeNames := resolveDocumentTypeNames(ctx, docTypeSvc, result.Results)

		return nil, SearchDocumentsOutput{
			Total:   result.Total,
			Page:    page,
			Results: toDocumentSummaries(result.Results, corrNames, docTypeNames),
		}, nil
	}
}

// NewFulltextSearchHandler creates a handler for fulltext_search.
func NewFulltextSearchHandler(svc *application.DocumentService, corrSvc *application.CorrespondentService, docTypeSvc *application.DocumentTypeService) mcp.ToolHandlerFor[FulltextSearchInput, FulltextSearchOutput] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input FulltextSearchInput) (*mcp.CallToolResult, FulltextSearchOutput, error) {
		page, pageSize := normalizePagination(input.Page, input.PageSize)

		result, err := svc.FulltextSearch(ctx, input.Query, page, pageSize)
		if err != nil {
			return nil, FulltextSearchOutput{}, fmt.Errorf("fulltext search: %w", err)
		}

		corrNames := resolveCorrespondentNames(ctx, corrSvc, result.Results)
		docTypeNames := resolveDocumentTypeNames(ctx, docTypeSvc, result.Results)

		items := make([]FulltextSearchResultItem, 0, len(result.Results))
		for _, doc := range result.Results {
			item := FulltextSearchResultItem{ //nolint:exhaustruct
				ID:            doc.ID,
				Title:         doc.Title,
				Correspondent: doc.Correspondent,
				DocumentType:  doc.DocumentType,
				Tags:          doc.Tags,
				Created:       doc.Created,
				Highlights:    "",
			}
			if doc.Correspondent != nil {
				item.CorrespondentName = corrNames[*doc.Correspondent]
			}
			if doc.DocumentType != nil {
				item.DocumentTypeName = docTypeNames[*doc.DocumentType]
			}
			if doc.SearchHit != nil {
				item.Highlights = doc.SearchHit.Highlights
			}
			items = append(items, item)
		}

		return nil, FulltextSearchOutput{
			Total:   result.Total,
			Page:    page,
			Results: items,
		}, nil
	}
}
