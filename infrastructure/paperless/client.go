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

	"github.com/teran/mcp-paperless-ngx/domain"
)

var errAPIClient = errors.New("API error")

// Client is the Paperless-ngx HTTP client implementing domain repositories.
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

// Search implements domain.DocumentRepository.
func (c *Client) Search(ctx context.Context, params domain.SearchDocumentsParams) (*domain.PaginatedResult[domain.Document], error) {
	q := buildSearchQuery(params)

	body, err := c.doRequest(ctx, "/api/documents/", q)
	if err != nil {
		return nil, err
	}

	var raw rawPaginatedDocuments
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("unmarshal documents: %w", err)
	}

	documents := make([]domain.Document, 0, len(raw.Results))
	for _, d := range raw.Results {
		documents = append(documents, d.toDomain())
	}

	return &domain.PaginatedResult[domain.Document]{
		Total:   raw.Count,
		Results: documents,
	}, nil
}

// GetByID implements domain.DocumentRepository.
func (c *Client) GetByID(ctx context.Context, id int) (*domain.Document, error) {
	path := fmt.Sprintf("/api/documents/%d/", id)
	body, err := c.doRequest(ctx, path, nil)
	if err != nil {
		return nil, err
	}

	var raw rawDocument
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("unmarshal document: %w", err)
	}

	doc := raw.toDomain()
	return &doc, nil
}

// SearchCorrespondents searches correspondents by name query.
func (c *Client) SearchCorrespondents(ctx context.Context, query string, page, pageSize int) (*domain.PaginatedResult[domain.Correspondent], error) {
	return fetchPaginated(ctx, c, "/api/correspondents/", query, page, pageSize,
		func(body []byte) ([]domain.Correspondent, int, error) {
			var raw rawPaginatedCorrespondents
			if err := json.Unmarshal(body, &raw); err != nil {
				return nil, 0, fmt.Errorf("unmarshal correspondents: %w", err)
			}
			result := make([]domain.Correspondent, 0, len(raw.Results))
			for _, r := range raw.Results {
				result = append(result, r.toDomain())
			}
			return result, raw.Count, nil
		})
}

// ListTags lists all tags, optionally filtered by name.
func (c *Client) ListTags(ctx context.Context, query string, page, pageSize int) (*domain.PaginatedResult[domain.Tag], error) {
	return fetchPaginated(ctx, c, "/api/tags/", query, page, pageSize,
		func(body []byte) ([]domain.Tag, int, error) {
			var raw rawPaginatedTags
			if err := json.Unmarshal(body, &raw); err != nil {
				return nil, 0, fmt.Errorf("unmarshal tags: %w", err)
			}
			result := make([]domain.Tag, 0, len(raw.Results))
			for _, r := range raw.Results {
				result = append(result, r.toDomain())
			}
			return result, raw.Count, nil
		})
}

// fetchPaginated is a generic helper that queries a paginated endpoint,
// unmarshals the results using the provided decode function, and returns
// a PaginatedResult.
func fetchPaginated[T any](ctx context.Context, c *Client, path, query string, page, pageSize int, decode func([]byte) ([]T, int, error)) (*domain.PaginatedResult[T], error) {
	q := url.Values{}
	if query != "" {
		q.Set("name__icontains", query)
	}
	q.Set("page", strconv.Itoa(page))
	q.Set("page_size", strconv.Itoa(pageSize))

	body, err := c.doRequest(ctx, path, q)
	if err != nil {
		return nil, err
	}

	items, total, err := decode(body)
	if err != nil {
		return nil, err
	}

	return &domain.PaginatedResult[T]{
		Total:   total,
		Results: items,
	}, nil
}

// -- Repository adapters --
//
// Client implements domain.DocumentRepository directly.
// For correspondent and tag repositories we need separate adapter types
// because Go does not allow two methods with the same name on one type.

// CorrespondentRepo adapts Client to domain.CorrespondentRepository.
type CorrespondentRepo struct {
	client *Client
}

// NewCorrespondentRepo creates a new CorrespondentRepo.
func NewCorrespondentRepo(client *Client) *CorrespondentRepo {
	return &CorrespondentRepo{client: client}
}

// Search implements domain.CorrespondentRepository.
func (r *CorrespondentRepo) Search(ctx context.Context, query string, page, pageSize int) (*domain.PaginatedResult[domain.Correspondent], error) {
	return r.client.SearchCorrespondents(ctx, query, page, pageSize)
}

// TagRepo adapts Client to domain.TagRepository.
type TagRepo struct {
	client *Client
}

// NewTagRepo creates a new TagRepo.
func NewTagRepo(client *Client) *TagRepo {
	return &TagRepo{client: client}
}

// List implements domain.TagRepository.
func (r *TagRepo) List(ctx context.Context, query string, page, pageSize int) (*domain.PaginatedResult[domain.Tag], error) {
	return r.client.ListTags(ctx, query, page, pageSize)
}

// -- private HTTP helpers --

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

func buildSearchQuery(params domain.SearchDocumentsParams) url.Values {
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
