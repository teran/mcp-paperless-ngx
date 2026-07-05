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
	"strings"

	"github.com/teran/mcp-paperless-ngx/domain"
)

var ErrAPIClient = errors.New("API error")

// Client is the Paperless-ngx HTTP client implementing domain repositories.
type Client struct {
	baseURL    string
	authToken  string
	httpClient *http.Client
}

// NewClient creates a new Paperless-ngx API client with the given HTTP client.
// The caller should provide an *http.Client with CheckRedirect set to prevent
// credential forwarding, and a shared Transport for connection reuse.
func NewClient(baseURL, authToken string, httpClient *http.Client) *Client {
	return &Client{
		baseURL:    baseURL,
		authToken:  authToken,
		httpClient: httpClient,
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

// GetCorrespondentByID fetches a single correspondent by ID.
func (c *Client) GetCorrespondentByID(ctx context.Context, id int) (*domain.Correspondent, error) {
	path := fmt.Sprintf("/api/correspondents/%d/", id)
	body, err := c.doRequest(ctx, path, nil)
	if err != nil {
		return nil, err
	}

	var raw rawCorrespondent
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("unmarshal correspondent: %w", err)
	}

	corr := raw.toDomain()
	return &corr, nil
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

// GetDocumentTypeByID fetches a single document type by ID.
func (c *Client) GetDocumentTypeByID(ctx context.Context, id int) (*domain.DocumentType, error) {
	path := fmt.Sprintf("/api/document_types/%d/", id)
	body, err := c.doRequest(ctx, path, nil)
	if err != nil {
		return nil, err
	}

	var raw rawDocumentType
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("unmarshal document type: %w", err)
	}

	dt := raw.toDomain()
	return &dt, nil
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

// GetByID implements domain.CorrespondentRepository.
func (r *CorrespondentRepo) GetByID(ctx context.Context, id int) (*domain.Correspondent, error) {
	return r.client.GetCorrespondentByID(ctx, id)
}

// DocumentTypeRepo adapts Client to domain.DocumentTypeRepository.
type DocumentTypeRepo struct {
	client *Client
}

// NewDocumentTypeRepo creates a new DocumentTypeRepo.
func NewDocumentTypeRepo(client *Client) *DocumentTypeRepo {
	return &DocumentTypeRepo{client: client}
}

// GetByID implements domain.DocumentTypeRepository.
func (r *DocumentTypeRepo) GetByID(ctx context.Context, id int) (*domain.DocumentType, error) {
	return r.client.GetDocumentTypeByID(ctx, id)
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

	// Limit response body to 100 MB to prevent memory exhaustion
	// from oversized responses (e.g. a document with a very large OCR text).
	body, err := io.ReadAll(io.LimitReader(resp.Body, 100<<20))
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		detail := extractErrorDetail(body, resp.StatusCode)
		return nil, fmt.Errorf("API status=%d: %s: %w", resp.StatusCode, detail, ErrAPIClient)
	}

	return body, nil
}

// extractErrorDetail extracts a human-readable detail from an error response body.
// It first attempts to parse JSON {"detail": "..."} from Paperless-ngx, then falls
// back to the first line of the body, truncated to 512 bytes to avoid leaking
// large responses in error messages.
func extractErrorDetail(body []byte, statusCode int) string {
	if len(body) == 0 {
		return http.StatusText(statusCode)
	}

	// Try JSON {"detail":"..."} first.
	var errResp struct {
		Detail string `json:"detail"`
	}
	if json.Unmarshal(body, &errResp) == nil && errResp.Detail != "" {
		return errResp.Detail
	}

	// Fallback: take first line, sanitize.
	detail := string(body)
	if idx := strings.IndexAny(detail, "\n\r"); idx >= 0 {
		detail = detail[:idx]
	}
	if len(detail) > 512 {
		detail = detail[:512] + "..."
	}
	return detail
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
	q.Set("page", strconv.Itoa(params.Page))
	q.Set("page_size", strconv.Itoa(params.PageSize))

	return q
}
