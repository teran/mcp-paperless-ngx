package server

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/teran/mcp-paperless-ngx/internal/paperless"
)

// ============================================================
// Tool input/output types
// ============================================================

// Base pagination.
type PaginationInput struct {
	Page     int `json:"page,omitempty" jsonschema:"page number (starts at 1),default=1"`
	PageSize int `json:"page_size,omitempty" jsonschema:"results per page (max 100),default=25"`
}

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
	Total   int                `json:"total"`
	Page    int                `json:"page"`
	Results []DocumentSummary  `json:"results"`
}

// --- get_document_content ---

type GetDocumentContentInput struct {
	DocumentID int `json:"document_id" jsonschema:"the ID of the document to retrieve,required"`
}

type DocumentDetail struct {
	ID                  int      `json:"id"`
	Title               string   `json:"title"`
	Content             string   `json:"content"`
	Correspondent       *int     `json:"correspondent,omitempty"`
	DocumentType        *int     `json:"document_type,omitempty"`
	Tags                []int    `json:"tags,omitempty"`
	Created             string   `json:"created"`
	Modified            string   `json:"modified"`
	Added               string   `json:"added"`
	ArchiveSerialNumber *int     `json:"archive_serial_number,omitempty"`
	OriginalFileName    string   `json:"original_file_name"`
	MimeType            string   `json:"mime_type"`
	PageCount           *int     `json:"page_count,omitempty"`
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
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Color       string `json:"color"`
	IsInboxTag  bool   `json:"is_inbox_tag"`
	DocumentCount int  `json:"document_count"`
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
	ID            int        `json:"id"`
	Title         string     `json:"title"`
	Correspondent *int       `json:"correspondent,omitempty"`
	Tags          []int      `json:"tags,omitempty"`
	Created       string     `json:"created"`
	Score         float64    `json:"score,omitempty"`
	Highlights    string     `json:"highlights,omitempty"`
	Rank          int        `json:"rank,omitempty"`
}

type FulltextSearchOutput struct {
	Total   int                       `json:"total"`
	Page    int                       `json:"page"`
	Results []FulltextSearchResultItem `json:"results"`
}

// ============================================================
// Helper: get client from context
// ============================================================

func mustGetClient(ctx context.Context) (*paperless.Client, error) {
	client := ClientFromContext(ctx)
	if client == nil {
		return nil, fmt.Errorf("paperless client not found in context — missing Authorization header or middleware misconfiguration")
	}
	return client, nil
}

// ============================================================
// Tool handlers
// ============================================================

func searchDocumentsHandler(ctx context.Context, _ *mcp.CallToolRequest, input SearchDocumentsInput) (*mcp.CallToolResult, SearchDocumentsOutput, error) {
	client, err := mustGetClient(ctx)
	if err != nil {
		return nil, SearchDocumentsOutput{}, err
	}

	page := input.Page
	if page == 0 {
		page = 1
	}
	pageSize := input.PageSize
	if pageSize == 0 || pageSize > 100 {
		pageSize = 25
	}

	resp, err := client.SearchDocuments(ctx, paperless.SearchDocumentsParams{
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

	results := make([]DocumentSummary, 0, len(resp.Results))
	for _, doc := range resp.Results {
		results = append(results, DocumentSummary{
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

	return nil, SearchDocumentsOutput{
		Total:   resp.Count,
		Page:    page,
		Results: results,
	}, nil
}

func getDocumentContentHandler(ctx context.Context, _ *mcp.CallToolRequest, input GetDocumentContentInput) (*mcp.CallToolResult, DocumentDetail, error) {
	client, err := mustGetClient(ctx)
	if err != nil {
		return nil, DocumentDetail{}, err
	}

	doc, err := client.GetDocument(ctx, input.DocumentID)
	if err != nil {
		return nil, DocumentDetail{}, fmt.Errorf("get document: %w", err)
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

func searchCorrespondentsHandler(ctx context.Context, _ *mcp.CallToolRequest, input SearchCorrespondentsInput) (*mcp.CallToolResult, SearchCorrespondentsOutput, error) {
	client, err := mustGetClient(ctx)
	if err != nil {
		return nil, SearchCorrespondentsOutput{}, err
	}

	page := input.Page
	if page == 0 {
		page = 1
	}
	pageSize := input.PageSize
	if pageSize == 0 || pageSize > 100 {
		pageSize = 25
	}

	resp, err := client.SearchCorrespondents(ctx, input.Query, page, pageSize)
	if err != nil {
		return nil, SearchCorrespondentsOutput{}, fmt.Errorf("search correspondents: %w", err)
	}

	results := make([]CorrespondentSummary, 0, len(resp.Results))
	for _, c := range resp.Results {
		results = append(results, CorrespondentSummary{
			ID:            c.ID,
			Name:          c.Name,
			DocumentCount: c.DocumentCount,
			Slug:          c.Slug,
		})
	}

	return nil, SearchCorrespondentsOutput{
		Total:   resp.Count,
		Page:    page,
		Results: results,
	}, nil
}

func getDocumentsByCorrespondentHandler(ctx context.Context, _ *mcp.CallToolRequest, input GetDocumentsByCorrespondentInput) (*mcp.CallToolResult, SearchDocumentsOutput, error) {
	client, err := mustGetClient(ctx)
	if err != nil {
		return nil, SearchDocumentsOutput{}, err
	}

	page := input.Page
	if page == 0 {
		page = 1
	}
	pageSize := input.PageSize
	if pageSize == 0 || pageSize > 100 {
		pageSize = 25
	}

	resp, err := client.GetDocumentsByCorrespondent(ctx, input.CorrespondentID, page, pageSize)
	if err != nil {
		return nil, SearchDocumentsOutput{}, fmt.Errorf("get documents by correspondent: %w", err)
	}

	results := make([]DocumentSummary, 0, len(resp.Results))
	for _, doc := range resp.Results {
		results = append(results, DocumentSummary{
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

	return nil, SearchDocumentsOutput{
		Total:   resp.Count,
		Page:    page,
		Results: results,
	}, nil
}

func listTagsHandler(ctx context.Context, _ *mcp.CallToolRequest, input ListTagsInput) (*mcp.CallToolResult, ListTagsOutput, error) {
	client, err := mustGetClient(ctx)
	if err != nil {
		return nil, ListTagsOutput{}, err
	}

	page := input.Page
	if page == 0 {
		page = 1
	}
	pageSize := input.PageSize
	if pageSize == 0 || pageSize > 100 {
		pageSize = 25
	}

	resp, err := client.ListTags(ctx, input.Query, page, pageSize)
	if err != nil {
		return nil, ListTagsOutput{}, fmt.Errorf("list tags: %w", err)
	}

	results := make([]TagSummary, 0, len(resp.Results))
	for _, t := range resp.Results {
		results = append(results, TagSummary{
			ID:            t.ID,
			Name:          t.Name,
			Color:         t.Color,
			IsInboxTag:    t.IsInboxTag,
			DocumentCount: t.DocumentCount,
		})
	}

	return nil, ListTagsOutput{
		Total:   resp.Count,
		Page:    page,
		Results: results,
	}, nil
}

func getDocumentsByTagHandler(ctx context.Context, _ *mcp.CallToolRequest, input GetDocumentsByTagInput) (*mcp.CallToolResult, SearchDocumentsOutput, error) {
	client, err := mustGetClient(ctx)
	if err != nil {
		return nil, SearchDocumentsOutput{}, err
	}

	page := input.Page
	if page == 0 {
		page = 1
	}
	pageSize := input.PageSize
	if pageSize == 0 || pageSize > 100 {
		pageSize = 25
	}

	resp, err := client.GetDocumentsByTag(ctx, input.TagID, page, pageSize)
	if err != nil {
		return nil, SearchDocumentsOutput{}, fmt.Errorf("get documents by tag: %w", err)
	}

	results := make([]DocumentSummary, 0, len(resp.Results))
	for _, doc := range resp.Results {
		results = append(results, DocumentSummary{
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

	return nil, SearchDocumentsOutput{
		Total:   resp.Count,
		Page:    page,
		Results: results,
	}, nil
}

func fulltextSearchHandler(ctx context.Context, _ *mcp.CallToolRequest, input FulltextSearchInput) (*mcp.CallToolResult, FulltextSearchOutput, error) {
	client, err := mustGetClient(ctx)
	if err != nil {
		return nil, FulltextSearchOutput{}, err
	}

	page := input.Page
	if page == 0 {
		page = 1
	}
	pageSize := input.PageSize
	if pageSize == 0 || pageSize > 100 {
		pageSize = 25
	}

	resp, err := client.FulltextSearch(ctx, input.Query, page, pageSize)
	if err != nil {
		return nil, FulltextSearchOutput{}, fmt.Errorf("fulltext search: %w", err)
	}

	results := make([]FulltextSearchResultItem, 0, len(resp.Results))
	for _, doc := range resp.Results {
		item := FulltextSearchResultItem{
			ID:            doc.ID,
			Title:         doc.Title,
			Correspondent: doc.Correspondent,
			Tags:          doc.Tags,
			Created:       doc.Created,
		}
		if doc.SearchHit != nil {
			item.Score = doc.SearchHit.Score
			item.Highlights = doc.SearchHit.Highlights
			item.Rank = doc.SearchHit.Rank
		}
		results = append(results, item)
	}

	return nil, FulltextSearchOutput{
		Total:   resp.Count,
		Page:    page,
		Results: results,
	}, nil
}

// ============================================================
// Tool registration
// ============================================================

// RegisterTools registers all tools on the given MCP server.
func RegisterTools(s *mcp.Server) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "search_documents",
		Description: "Search documents in Paperless-ngx with optional filters (query, correspondent, tags, date range). Returns paginated results.",
	}, searchDocumentsHandler)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_document_content",
		Description: "Retrieve the full OCR text content and metadata of a specific document by its ID.",
	}, getDocumentContentHandler)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "search_correspondents",
		Description: "Search correspondents by name (substring match). Returns matching correspondents with document counts.",
	}, searchCorrespondentsHandler)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_documents_by_correspondent",
		Description: "List all documents associated with a specific correspondent ID. Returns paginated results.",
	}, getDocumentsByCorrespondentHandler)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_tags",
		Description: "List all tags in Paperless-ngx, optionally filtered by name. Returns tags with colors and document counts.",
	}, listTagsHandler)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_documents_by_tag",
		Description: "List all documents associated with a specific tag ID. Returns paginated results.",
	}, getDocumentsByTagHandler)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "fulltext_search",
		Description: "Perform a full-text search across all documents. Returns paginated results with search score and highlights.",
	}, fulltextSearchHandler)
}
