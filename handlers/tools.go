package handlers

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

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
	ID            int    `json:"id"`
	Title         string `json:"title"`
	Correspondent *int   `json:"correspondent,omitempty"`
	Tags          []int  `json:"tags,omitempty"`
	Created       string `json:"created"`
	MimeType      string `json:"mime_type"`
	ArchiveSerial *int   `json:"archive_serial_number,omitempty"`
	PageCount     *int   `json:"page_count,omitempty"`
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
	DocumentType        *int   `json:"document_type,omitempty"`
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
	ID            int    `json:"id"`
	Title         string `json:"title"`
	Correspondent *int   `json:"correspondent,omitempty"`
	Tags          []int  `json:"tags,omitempty"`
	Created       string `json:"created"`
	Highlights    string `json:"highlights,omitempty"`
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

func toDocumentSummaries(docs []domain.Document) []DocumentSummary {
	result := make([]DocumentSummary, 0, len(docs))
	for _, doc := range docs {
		result = append(result, DocumentSummary{
			ID:            doc.ID,
			Title:         doc.Title,
			Correspondent: doc.Correspondent,
			Tags:          doc.Tags,
			Created:       doc.Created,
			MimeType:      doc.MimeType,
			ArchiveSerial: doc.ArchiveSerialNumber,
			PageCount:     doc.PageCount,
		})
	}
	return result
}

// ============================================================
// Tool handler factories
// ============================================================

// NewSearchDocumentsHandler creates a handler for search_documents.
func NewSearchDocumentsHandler(svc *application.DocumentService) mcp.ToolHandlerFor[SearchDocumentsInput, SearchDocumentsOutput] {
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

		return nil, SearchDocumentsOutput{
			Total:   result.Total,
			Page:    page,
			Results: toDocumentSummaries(result.Results),
		}, nil
	}
}

// NewGetDocumentContentHandler creates a handler for get_document_content.
func NewGetDocumentContentHandler(svc *application.DocumentService) mcp.ToolHandlerFor[GetDocumentContentInput, DocumentDetail] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input GetDocumentContentInput) (*mcp.CallToolResult, DocumentDetail, error) {
		doc, err := svc.GetByID(ctx, input.DocumentID)
		if err != nil {
			return nil, DocumentDetail{}, fmt.Errorf("get document content: %w", err)
		}

		return nil, DocumentDetail{
			ID:                  doc.ID,
			Title:               doc.Title,
			Content:             doc.Content,
			Correspondent:       doc.Correspondent,
			DocumentType:        doc.DocumentType,
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
func NewGetDocumentsByCorrespondentHandler(svc *application.DocumentService) mcp.ToolHandlerFor[GetDocumentsByCorrespondentInput, SearchDocumentsOutput] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input GetDocumentsByCorrespondentInput) (*mcp.CallToolResult, SearchDocumentsOutput, error) {
		page, pageSize := normalizePagination(input.Page, input.PageSize)

		result, err := svc.GetByCorrespondent(ctx, input.CorrespondentID, page, pageSize)
		if err != nil {
			return nil, SearchDocumentsOutput{}, fmt.Errorf("get documents by correspondent: %w", err)
		}

		return nil, SearchDocumentsOutput{
			Total:   result.Total,
			Page:    page,
			Results: toDocumentSummaries(result.Results),
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
func NewGetDocumentsByTagHandler(svc *application.DocumentService) mcp.ToolHandlerFor[GetDocumentsByTagInput, SearchDocumentsOutput] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input GetDocumentsByTagInput) (*mcp.CallToolResult, SearchDocumentsOutput, error) {
		page, pageSize := normalizePagination(input.Page, input.PageSize)

		result, err := svc.GetByTag(ctx, input.TagID, page, pageSize)
		if err != nil {
			return nil, SearchDocumentsOutput{}, fmt.Errorf("get documents by tag: %w", err)
		}

		return nil, SearchDocumentsOutput{
			Total:   result.Total,
			Page:    page,
			Results: toDocumentSummaries(result.Results),
		}, nil
	}
}

// NewFulltextSearchHandler creates a handler for fulltext_search.
func NewFulltextSearchHandler(svc *application.DocumentService) mcp.ToolHandlerFor[FulltextSearchInput, FulltextSearchOutput] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input FulltextSearchInput) (*mcp.CallToolResult, FulltextSearchOutput, error) {
		page, pageSize := normalizePagination(input.Page, input.PageSize)

		result, err := svc.FulltextSearch(ctx, input.Query, page, pageSize)
		if err != nil {
			return nil, FulltextSearchOutput{}, fmt.Errorf("fulltext search: %w", err)
		}

		items := make([]FulltextSearchResultItem, 0, len(result.Results))
		for _, doc := range result.Results {
			item := FulltextSearchResultItem{
				ID:            doc.ID,
				Title:         doc.Title,
				Correspondent: doc.Correspondent,
				Tags:          doc.Tags,
				Created:       doc.Created,
				Highlights:    "",
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
