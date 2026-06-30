package paperless

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

var errAPIClient = errors.New("API error")

// Client is an HTTP client for the Paperless-ngx REST API.
type Client struct {
	baseURL    string
	authToken  string
	httpClient *http.Client
}

// NewClient creates a new Paperless-ngx API client.
func NewClient(baseURL, authToken string) *Client {
	return &Client{
		baseURL:   baseURL,
		authToken: authToken,
		httpClient: &http.Client{ //nolint:exhaustruct
			Timeout: 30 * time.Second,
		},
	}
}

// SearchDocuments searches documents with the given filters.
// Returns paginated results.
func (c *Client) SearchDocuments(ctx context.Context, params SearchDocumentsParams) (*PaginatedResponse, error) {
	q := buildSearchQuery(params)

	body, err := c.doRequest(ctx, "/api/documents/", q)
	if err != nil {
		return nil, err
	}

	var resp PaginatedResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}
	return &resp, nil
}

// SearchDocumentsParams holds search parameters.
type SearchDocumentsParams struct {
	Query           string
	CorrespondentID int
	TagIDs          []int
	CreatedAfter    string
	CreatedBefore   string
	Page            int
	PageSize        int
}

// GetDocument retrieves a single document by ID.
func (c *Client) GetDocument(ctx context.Context, documentID int) (*Document, error) {
	path := fmt.Sprintf("/api/documents/%d/", documentID)
	body, err := c.doRequest(ctx, path, nil)
	if err != nil {
		return nil, err
	}

	var doc Document
	if err := json.Unmarshal(body, &doc); err != nil {
		return nil, fmt.Errorf("unmarshal document: %w", err)
	}
	return &doc, nil
}

// SearchCorrespondents searches correspondents by name.
func (c *Client) SearchCorrespondents(ctx context.Context, query string, page, pageSize int) (*CorrespondentPaginatedResponse, error) {
	q := url.Values{}
	if query != "" {
		q.Set("name__icontains", query)
	}
	if page > 0 {
		q.Set("page", strconv.Itoa(page))
	} else {
		q.Set("page", "1")
	}
	if pageSize > 0 {
		q.Set("page_size", strconv.Itoa(pageSize))
	} else {
		q.Set("page_size", "25")
	}

	body, err := c.doRequest(ctx, "/api/correspondents/", q)
	if err != nil {
		return nil, err
	}

	var resp CorrespondentPaginatedResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal correspondents: %w", err)
	}
	return &resp, nil
}

// GetDocumentsByCorrespondent retrieves documents for a specific correspondent.
func (c *Client) GetDocumentsByCorrespondent(ctx context.Context, correspondentID, page, pageSize int) (*PaginatedResponse, error) {
	return c.SearchDocuments(ctx, SearchDocumentsParams{
		Query:           "",
		CorrespondentID: correspondentID,
		TagIDs:          nil,
		CreatedAfter:    "",
		CreatedBefore:   "",
		Page:            page,
		PageSize:        pageSize,
	})
}

// ListTags retrieves all tags, optionally filtered by name.
func (c *Client) ListTags(ctx context.Context, query string, page, pageSize int) (*TagPaginatedResponse, error) {
	q := url.Values{}
	if query != "" {
		q.Set("name__icontains", query)
	}
	if page > 0 {
		q.Set("page", strconv.Itoa(page))
	} else {
		q.Set("page", "1")
	}
	if pageSize > 0 {
		q.Set("page_size", strconv.Itoa(pageSize))
	} else {
		q.Set("page_size", "25")
	}

	body, err := c.doRequest(ctx, "/api/tags/", q)
	if err != nil {
		return nil, err
	}

	var resp TagPaginatedResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal tags: %w", err)
	}
	return &resp, nil
}

// GetDocumentsByTag retrieves documents for a specific tag.
func (c *Client) GetDocumentsByTag(ctx context.Context, tagID, page, pageSize int) (*PaginatedResponse, error) {
	q := url.Values{}
	q.Set("tags__id__all", strconv.Itoa(tagID))
	if page > 0 {
		q.Set("page", strconv.Itoa(page))
	} else {
		q.Set("page", "1")
	}
	if pageSize > 0 {
		q.Set("page_size", strconv.Itoa(pageSize))
	} else {
		q.Set("page_size", "25")
	}

	body, err := c.doRequest(ctx, "/api/documents/", q)
	if err != nil {
		return nil, err
	}

	var resp PaginatedResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal documents: %w", err)
	}
	return &resp, nil
}

// FulltextSearch performs a full-text search across all documents.
func (c *Client) FulltextSearch(ctx context.Context, query string, page, pageSize int) (*PaginatedResponse, error) {
	return c.SearchDocuments(ctx, SearchDocumentsParams{
		Query:           query,
		CorrespondentID: 0,
		TagIDs:          nil,
		CreatedAfter:    "",
		CreatedBefore:   "",
		Page:            page,
		PageSize:        pageSize,
	})
}

// buildSearchQuery converts SearchDocumentsParams into URL query values.
func buildSearchQuery(params SearchDocumentsParams) url.Values {
	q := url.Values{}
	if params.Query != "" {
		q.Set("query", params.Query)
	}
	if params.CorrespondentID > 0 {
		q.Set("correspondent__id", strconv.Itoa(params.CorrespondentID))
	}
	for _, tid := range params.TagIDs {
		q.Add("tags__id__all", strconv.Itoa(tid))
	}
	if params.CreatedAfter != "" {
		q.Set("created__date__gte", params.CreatedAfter)
	}
	if params.CreatedBefore != "" {
		q.Set("created__date__lte", params.CreatedBefore)
	}
	page := params.Page
	if page <= 0 {
		page = 1
	}
	q.Set("page", strconv.Itoa(page))
	pageSize := params.PageSize
	if pageSize <= 0 {
		pageSize = 25
	}
	q.Set("page_size", strconv.Itoa(pageSize))

	return q
}

// doRequest performs an authenticated GET request.
func (c *Client) doRequest(ctx context.Context, path string, query url.Values) ([]byte, error) {
	u, err := url.JoinPath(c.baseURL, path)
	if err != nil {
		return nil, fmt.Errorf("build URL: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	if query != nil {
		req.URL.RawQuery = query.Encode()
	}

	req.Header.Set("Authorization", "Token "+c.authToken)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("API status=%d body=%s: %w", resp.StatusCode, string(body), errAPIClient)
	}

	return body, nil
}
