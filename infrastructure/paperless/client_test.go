package paperless_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/teran/mcp-paperless-ngx/domain"
	"github.com/teran/mcp-paperless-ngx/infrastructure/paperless"
)

// testHTTPClient is a shared HTTP client for tests that never follows redirects.
var testHTTPClient = &http.Client{ //nolint:gochecknoglobals
	Timeout: 5 * time.Second,
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	},
}

// ---------------------------------------------------------------------------
// helper functions
// ---------------------------------------------------------------------------

// newTestServer returns an httptest.Server whose handler asserts the method
// and decodes query parameters so each test can plug in its own handler.
func newTestServer(handler func(w http.ResponseWriter, r *http.Request)) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(handler))
}

// newClient builds a paperless.Client pointed at the given baseURL.
func newClient(baseURL string) *paperless.Client {
	return paperless.NewClient(baseURL, "test-token-123", testHTTPClient)
}

// assertQueryParam checks that the query string in the request contains the
// expected key=value pair.
func assertQueryParam(t *testing.T, r *http.Request, key, want string) {
	t.Helper()
	got := r.URL.Query().Get(key)
	if got != want {
		t.Errorf("query param %q = %q, want %q", key, got, want)
	}
}

// assertQueryParamNotSet checks that a key is absent from the query string.
func assertQueryParamNotSet(t *testing.T, r *http.Request, key string) {
	t.Helper()
	if v := r.URL.Query().Get(key); v != "" {
		t.Errorf("query param %q = %q, want not set", key, v)
	}
}

// assertAuthHeader checks that the Authorization header is present.
func assertAuthHeader(t *testing.T, r *http.Request) {
	t.Helper()
	if v := r.Header.Get("Authorization"); v != "Token test-token-123" {
		t.Errorf("Authorization header = %q, want %q", v, "Token test-token-123")
	}
}

// assertAcceptHeader checks that the Accept header is set correctly.
func assertAcceptHeader(t *testing.T, r *http.Request) {
	t.Helper()
	if v := r.Header.Get("Accept"); v != "application/json" {
		t.Errorf("Accept header = %q, want %q", v, "application/json")
	}
}

// ---------------------------------------------------------------------------
// test data
// ---------------------------------------------------------------------------

var sampleDocumentRaw = map[string]any{ //nolint:gochecknoglobals
	"id":                     123,
	"correspondent":          5,                                          //nolint:goconst
	"document_type":          2,                                          //nolint:goconst
	"storage_path":           nil,                                        //nolint:goconst
	"title":                  "Bank Statement March 2024",                //nolint:goconst
	"content":                "Full OCR text content of the document...", //nolint:goconst
	"tags":                   []int{7, 12, 15},                           //nolint:goconst
	"created":                "2024-03-15",                               //nolint:goconst
	"created_date":           "2024-03-15",                               //nolint:goconst
	"modified":               "2024-03-15T14:30:00+00:00",                //nolint:goconst
	"added":                  "2024-03-15T10:00:00+00:00",                //nolint:goconst
	"deleted_at":             nil,
	"archive_serial_number":  42,                            //nolint:goconst
	"original_file_name":     "statement.pdf",               //nolint:goconst
	"archived_file_name":     "20240315_bank_statement.pdf", //nolint:goconst
	"owner":                  1,                             //nolint:goconst
	"user_can_change":        true,                          //nolint:goconst
	"is_shared_by_requester": false,                         //nolint:goconst
	"notes":                  []any{},                       //nolint:goconst
	"custom_fields":          []any{},                       //nolint:goconst
	"page_count":             3,                             //nolint:goconst
	"mime_type":              "application/pdf",             //nolint:goconst
}

var sampleDocumentWithSearchHit = map[string]any{ //nolint:gochecknoglobals
	"id":                     123,
	"correspondent":          nil,
	"document_type":          nil,
	"storage_path":           nil,
	"title":                  "Bank Statement March 2024",
	"content":                "Full OCR text content of the document...",
	"tags":                   []int{7},
	"created":                "2024-03-15",
	"created_date":           "2024-03-15",
	"modified":               "2024-03-15T14:30:00+00:00",
	"added":                  "2024-03-15T10:00:00+00:00",
	"archive_serial_number":  nil,
	"original_file_name":     "statement.pdf",
	"archived_file_name":     nil,
	"owner":                  nil,
	"page_count":             3,
	"mime_type":              "application/pdf",
	"user_can_change":        true,
	"is_shared_by_requester": false,
	"notes":                  []any{},
	"custom_fields":          []any{},
	"__search_hit__": map[string]any{
		"score":      12.345,
		"highlights": "text with <b>matched</b> terms highlighted",
		"rank":       1,
	},
}

var sampleDocumentNullFields = map[string]any{ //nolint:gochecknoglobals
	"id":                     456,
	"correspondent":          nil,
	"document_type":          nil,
	"storage_path":           nil,
	"title":                  "Receipt",
	"content":                "Some content",
	"tags":                   []int{},
	"created":                "2024-04-01",
	"created_date":           "2024-04-01",
	"modified":               "2024-04-01T12:00:00+00:00",
	"added":                  "2024-04-01T12:00:00+00:00",
	"deleted_at":             nil,
	"archive_serial_number":  nil,
	"original_file_name":     "receipt.pdf",
	"archived_file_name":     nil,
	"owner":                  nil,
	"page_count":             nil,
	"mime_type":              "application/pdf",
	"user_can_change":        true,
	"is_shared_by_requester": false,
	"notes":                  []any{},
	"custom_fields":          []any{},
}

var sampleCorrespondentRaw = map[string]any{ //nolint:gochecknoglobals
	"id":                  5,
	"slug":                "acme-corp", //nolint:goconst
	"name":                "Acme Corp", //nolint:goconst
	"match":               "acme",      //nolint:goconst
	"matching_algorithm":  1,           //nolint:goconst
	"is_insensitive":      true,        //nolint:goconst
	"document_count":      23,          //nolint:goconst
	"last_correspondence": "2024-06-15",
	"owner":               1,
	"user_can_change":     true,
}

var sampleTagRaw = map[string]any{ //nolint:gochecknoglobals
	"id":                 7,
	"slug":               "invoice", //nolint:goconst
	"name":               "Invoice", //nolint:goconst
	"color":              "#a6cee3",
	"text_color":         "#000000",
	"match":              "invoice",
	"matching_algorithm": 1,
	"is_insensitive":     true,
	"is_inbox_tag":       false,
	"document_count":     15,
	"owner":              nil,
	"user_can_change":    true,
	"parent":             nil,
	"children":           []any{},
}

var sampleDocumentTypeRaw = map[string]any{ //nolint:gochecknoglobals
	"id":                 2,
	"slug":               "invoice", //nolint:goconst
	"name":               "Invoice", //nolint:goconst
	"match":              "invoice",
	"matching_algorithm": 1,    //nolint:goconst
	"is_insensitive":     true, //nolint:goconst
	"document_count":     15,   //nolint:goconst
	"owner":              1,
	"user_can_change":    true, //nolint:goconst
}

// paginatedResponse wraps results in the standard Paperless-ngx paginated format.
func paginatedResponse(results []any) map[string]any {
	ids := make([]int, 0, len(results))
	for _, r := range results {
		if m, ok := r.(map[string]any); ok {
			if id, ok := m["id"].(int); ok {
				ids = append(ids, id)
			}
		}
	}
	return map[string]any{
		"count":    len(results),
		"next":     nil,
		"previous": nil,
		"all":      ids,
		"results":  results,
	}
}

// ---------------------------------------------------------------------------
// Client.Search — documents
// ---------------------------------------------------------------------------

func TestClient_Search_WithQuery(t *testing.T) {
	t.Parallel()

	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		assertAuthHeader(t, r)
		assertAcceptHeader(t, r)

		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/api/documents/") {
			t.Errorf("path = %s, want /api/documents/", r.URL.Path)
		}
		assertQueryParam(t, r, "query", "bank statement")
		assertQueryParam(t, r, "page", "1")
		assertQueryParam(t, r, "page_size", "25")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(paginatedResponse([]any{
			sampleDocumentWithSearchHit,
		}))
	})
	defer srv.Close()

	client := newClient(srv.URL)
	result, err := client.Search(context.Background(), domain.SearchDocumentsParams{ //nolint:exhaustruct
		Query: "bank statement",
		Page:  1, PageSize: 25,
	})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if result.Total != 1 {
		t.Errorf("Total = %d, want 1", result.Total)
	}
	if len(result.Results) != 1 {
		t.Fatalf("len(Results) = %d, want 1", len(result.Results))
	}

	doc := result.Results[0]
	if doc.ID != 123 {
		t.Errorf("doc.ID = %d, want 123", doc.ID)
	}
	if doc.Title != "Bank Statement March 2024" {
		t.Errorf("doc.Title = %q, want %q", doc.Title, "Bank Statement March 2024")
	}
	if doc.Content != "Full OCR text content of the document..." {
		t.Errorf("doc.Content = %q, want %q", doc.Content, "Full OCR text content of the document...")
	}
	if doc.MimeType != "application/pdf" {
		t.Errorf("doc.MimeType = %q, want %q", doc.MimeType, "application/pdf")
	}

	// SearchHit must be populated
	if doc.SearchHit == nil {
		t.Fatal("doc.SearchHit = nil, want non-nil")
	}
	if doc.SearchHit.Score != 12.345 {
		t.Errorf("doc.SearchHit.Score = %f, want 12.345", doc.SearchHit.Score)
	}
	if doc.SearchHit.Highlights != "text with <b>matched</b> terms highlighted" {
		t.Errorf("doc.SearchHit.Highlights = %q, want %q", doc.SearchHit.Highlights, "text with <b>matched</b> terms highlighted")
	}
	if doc.SearchHit.Rank != 1 {
		t.Errorf("doc.SearchHit.Rank = %d, want 1", doc.SearchHit.Rank)
	}
}

func TestClient_Search_WithCorrespondentID(t *testing.T) {
	t.Parallel()

	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		assertQueryParam(t, r, "correspondent__id", "5")
		assertQueryParamNotSet(t, r, "query")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(paginatedResponse([]any{
			sampleDocumentRaw,
		}))
	})
	defer srv.Close()

	client := newClient(srv.URL)
	result, err := client.Search(context.Background(), domain.SearchDocumentsParams{ //nolint:exhaustruct
		CorrespondentID: 5,
	})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if result.Total != 1 {
		t.Errorf("Total = %d, want 1", result.Total)
	}
	if len(result.Results) != 1 {
		t.Fatalf("len(Results) = %d, want 1", len(result.Results))
	}
	if result.Results[0].Correspondent == nil || *result.Results[0].Correspondent != 5 {
		t.Errorf("doc.Correspondent = %v, want 5", result.Results[0].Correspondent)
	}
}

func TestClient_Search_WithTagIDs(t *testing.T) {
	t.Parallel()

	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		// tag_ids__all is repeated for each tag.
		wantTags := []string{"7", "12", "15"}
		gotTags := r.URL.Query()["tags__id__all"]
		if len(gotTags) != len(wantTags) {
			t.Errorf("tags__id__all = %v, want %v", gotTags, wantTags)
		} else {
			for i := range wantTags {
				if gotTags[i] != wantTags[i] {
					t.Errorf("tags__id__all[%d] = %s, want %s", i, gotTags[i], wantTags[i])
				}
			}
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(paginatedResponse([]any{
			sampleDocumentRaw,
		}))
	})
	defer srv.Close()

	client := newClient(srv.URL)
	result, err := client.Search(context.Background(), domain.SearchDocumentsParams{ //nolint:exhaustruct
		TagIDs: []int{7, 12, 15},
	})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if result.Total != 1 {
		t.Errorf("Total = %d, want 1", result.Total)
	}
	if len(result.Results) != 1 {
		t.Fatalf("len(Results) = %d, want 1", len(result.Results))
	}
}

func TestClient_Search_WithDateRange(t *testing.T) {
	t.Parallel()

	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		assertQueryParam(t, r, "created__date__gte", "2024-01-01")
		assertQueryParam(t, r, "created__date__lte", "2024-03-31")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(paginatedResponse([]any{
			sampleDocumentRaw,
		}))
	})
	defer srv.Close()

	client := newClient(srv.URL)
	result, err := client.Search(context.Background(), domain.SearchDocumentsParams{ //nolint:exhaustruct
		CreatedAfter:  "2024-01-01", //nolint:goconst
		CreatedBefore: "2024-03-31",
	})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if result.Total != 1 {
		t.Errorf("Total = %d, want 1", result.Total)
	}
	if len(result.Results) != 1 {
		t.Fatalf("len(Results) = %d, want 1", len(result.Results))
	}
	if result.Results[0].Created != "2024-03-15" {
		t.Errorf("doc.Created = %q, want %q", result.Results[0].Created, "2024-03-15")
	}
	if result.Results[0].CreatedDate != "2024-03-15" {
		t.Errorf("doc.CreatedDate = %q, want %q", result.Results[0].CreatedDate, "2024-03-15")
	}
}

func TestClient_Search_WithPagination(t *testing.T) {
	t.Parallel()

	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		assertQueryParam(t, r, "page", "3")
		assertQueryParam(t, r, "page_size", "10")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(paginatedResponse([]any{
			sampleDocumentRaw,
		}))
	})
	defer srv.Close()

	client := newClient(srv.URL)
	result, err := client.Search(context.Background(), domain.SearchDocumentsParams{ //nolint:exhaustruct
		Page:     3,
		PageSize: 10,
	})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if result.Total != 1 {
		t.Errorf("Total = %d, want 1", result.Total)
	}
	if len(result.Results) != 1 {
		t.Fatalf("len(Results) = %d, want 1", len(result.Results))
	}
}

func TestClient_Search_WithEmptyParams(t *testing.T) {
	t.Parallel()

	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		assertQueryParamNotSet(t, r, "query")
		assertQueryParamNotSet(t, r, "correspondent__id")
		assertQueryParamNotSet(t, r, "tags__id__all")
		assertQueryParamNotSet(t, r, "created__date__gte")
		assertQueryParamNotSet(t, r, "created__date__lte")
		assertQueryParam(t, r, "page", "1")
		assertQueryParam(t, r, "page_size", "25")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(paginatedResponse([]any{
			sampleDocumentRaw,
		}))
	})
	defer srv.Close()

	client := newClient(srv.URL)
	result, err := client.Search(context.Background(), domain.SearchDocumentsParams{ //nolint:exhaustruct
		Page:     1,
		PageSize: 25,
	})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if result.Total != 1 {
		t.Errorf("Total = %d, want 1", result.Total)
	}
	if len(result.Results) != 1 {
		t.Fatalf("len(Results) = %d, want 1", len(result.Results))
	}
}

func TestClient_Search_EmptyResults(t *testing.T) {
	t.Parallel()

	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(paginatedResponse([]any{}))
	})
	defer srv.Close()

	client := newClient(srv.URL)
	result, err := client.Search(context.Background(), domain.SearchDocumentsParams{ //nolint:exhaustruct
		Query: "nonexistent",
	})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if result.Total != 0 {
		t.Errorf("Total = %d, want 0", result.Total)
	}
	if len(result.Results) != 0 {
		t.Errorf("len(Results) = %d, want 0", len(result.Results))
	}
}

func TestClient_Search_RequestValidation(t *testing.T) {
	t.Parallel()

	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		assertAuthHeader(t, r)
		assertAcceptHeader(t, r)

		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/api/documents/") {
			t.Errorf("path = %s, want /api/documents/", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(paginatedResponse([]any{}))
	})
	defer srv.Close()

	client := newClient(srv.URL)
	_, err := client.Search(context.Background(), domain.SearchDocumentsParams{Page: 1, PageSize: 25}) //nolint:exhaustruct
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
}

// ---------------------------------------------------------------------------
// Client.GetByID — single document
// ---------------------------------------------------------------------------

func TestClient_GetByID_Existing(t *testing.T) { //nolint:gocognit
	t.Parallel()

	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		assertAuthHeader(t, r)
		assertAcceptHeader(t, r)

		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/api/documents/123/") {
			t.Errorf("path = %s, want /api/documents/123/", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(sampleDocumentRaw)
	})
	defer srv.Close()

	client := newClient(srv.URL)
	doc, err := client.GetByID(context.Background(), 123)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if doc.ID != 123 {
		t.Errorf("doc.ID = %d, want 123", doc.ID)
	}
	if doc.Title != "Bank Statement March 2024" {
		t.Errorf("doc.Title = %q, want %q", doc.Title, "Bank Statement March 2024")
	}
	if doc.Content != "Full OCR text content of the document..." {
		t.Errorf("doc.Content = %q, want %q", doc.Content, "Full OCR text content of the document...")
	}
	if doc.MimeType != "application/pdf" {
		t.Errorf("doc.MimeType = %q, want %q", doc.MimeType, "application/pdf")
	}
	if doc.OriginalFileName != "statement.pdf" {
		t.Errorf("doc.OriginalFileName = %q, want %q", doc.OriginalFileName, "statement.pdf")
	}
	if doc.ArchivedFileName == nil || *doc.ArchivedFileName != "20240315_bank_statement.pdf" {
		t.Errorf("doc.ArchivedFileName = %v, want %q", doc.ArchivedFileName, "20240315_bank_statement.pdf")
	}
	if doc.ArchiveSerialNumber == nil || *doc.ArchiveSerialNumber != 42 {
		t.Errorf("doc.ArchiveSerialNumber = %v, want 42", doc.ArchiveSerialNumber)
	}
	if doc.Owner == nil || *doc.Owner != 1 {
		t.Errorf("doc.Owner = %v, want 1", doc.Owner)
	}
	if doc.PageCount == nil || *doc.PageCount != 3 {
		t.Errorf("doc.PageCount = %v, want 3", doc.PageCount)
	}
	if doc.Correspondent == nil || *doc.Correspondent != 5 {
		t.Errorf("doc.Correspondent = %v, want 5", doc.Correspondent)
	}
	if doc.DocumentType == nil || *doc.DocumentType != 2 {
		t.Errorf("doc.DocumentType = %v, want 2", doc.DocumentType)
	}
	if doc.StoragePath != nil {
		t.Errorf("doc.StoragePath = %v, want nil", doc.StoragePath)
	}

	// Tags
	if len(doc.Tags) != 3 {
		t.Fatalf("len(doc.Tags) = %d, want 3", len(doc.Tags))
	}
	expectedTags := []int{7, 12, 15}
	for i, v := range doc.Tags {
		if v != expectedTags[i] {
			t.Errorf("doc.Tags[%d] = %d, want %d", i, v, expectedTags[i])
		}
	}

	// Timestamps
	if doc.Created != "2024-03-15" {
		t.Errorf("doc.Created = %q, want %q", doc.Created, "2024-03-15")
	}
	if doc.CreatedDate != "2024-03-15" {
		t.Errorf("doc.CreatedDate = %q, want %q", doc.CreatedDate, "2024-03-15")
	}
	if doc.Modified != "2024-03-15T14:30:00+00:00" {
		t.Errorf("doc.Modified = %q, want %q", doc.Modified, "2024-03-15T14:30:00+00:00")
	}
	if doc.Added != "2024-03-15T10:00:00+00:00" {
		t.Errorf("doc.Added = %q, want %q", doc.Added, "2024-03-15T10:00:00+00:00")
	}

	// SearchHit should be nil for non-search single-document endpoint.
	if doc.SearchHit != nil {
		t.Errorf("doc.SearchHit = %v, want nil", doc.SearchHit)
	}
}

func TestClient_GetByID_NotFound(t *testing.T) {
	t.Parallel()

	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"detail":"Not found"}`))
	})
	defer srv.Close()

	client := newClient(srv.URL)
	_, err := client.GetByID(context.Background(), 999)
	if err == nil {
		t.Fatal("GetByID() expected error, got nil")
	}

	// Verify it is an API error.
	if !errors.Is(err, paperless.ErrAPIClient) {
		t.Errorf("error does not wrap ErrAPIClient: %v", err)
	}
}

func TestClient_GetByID_NullFields(t *testing.T) {
	t.Parallel()

	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(sampleDocumentNullFields)
	})
	defer srv.Close()

	client := newClient(srv.URL)
	doc, err := client.GetByID(context.Background(), 456)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	// ID and basic fields
	if doc.ID != 456 {
		t.Errorf("doc.ID = %d, want 456", doc.ID)
	}
	if doc.Title != "Receipt" {
		t.Errorf("doc.Title = %q, want %q", doc.Title, "Receipt")
	}

	// Nil pointer fields
	if doc.Correspondent != nil {
		t.Errorf("doc.Correspondent = %v, want nil", doc.Correspondent)
	}
	if doc.DocumentType != nil {
		t.Errorf("doc.DocumentType = %v, want nil", doc.DocumentType)
	}
	if doc.StoragePath != nil {
		t.Errorf("doc.StoragePath = %v, want nil", doc.StoragePath)
	}
	if doc.ArchiveSerialNumber != nil {
		t.Errorf("doc.ArchiveSerialNumber = %v, want nil", doc.ArchiveSerialNumber)
	}
	if doc.ArchivedFileName != nil {
		t.Errorf("doc.ArchivedFileName = %v, want nil", doc.ArchivedFileName)
	}
	if doc.Owner != nil {
		t.Errorf("doc.Owner = %v, want nil", doc.Owner)
	}
	if doc.PageCount != nil {
		t.Errorf("doc.PageCount = %v, want nil", doc.PageCount)
	}
	if doc.SearchHit != nil {
		t.Errorf("doc.SearchHit = %v, want nil", doc.SearchHit)
	}

	// Empty tags
	if len(doc.Tags) != 0 {
		t.Errorf("len(doc.Tags) = %d, want 0", len(doc.Tags))
	}
}

// ---------------------------------------------------------------------------
// Client.SearchCorrespondents
// ---------------------------------------------------------------------------

func TestClient_SearchCorrespondents_ByName(t *testing.T) {
	t.Parallel()

	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		assertAuthHeader(t, r)
		assertAcceptHeader(t, r)

		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/api/correspondents/") {
			t.Errorf("path = %s, want /api/correspondents/", r.URL.Path)
		}
		assertQueryParam(t, r, "name__icontains", "acme")
		assertQueryParam(t, r, "page", "1")
		assertQueryParam(t, r, "page_size", "25")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(paginatedResponse([]any{
			sampleCorrespondentRaw,
		}))
	})
	defer srv.Close()

	client := newClient(srv.URL)
	result, err := client.SearchCorrespondents(context.Background(), "acme", 1, 25)
	if err != nil {
		t.Fatalf("SearchCorrespondents() error = %v", err)
	}
	if result.Total != 1 {
		t.Errorf("Total = %d, want 1", result.Total)
	}
	if len(result.Results) != 1 {
		t.Fatalf("len(Results) = %d, want 1", len(result.Results))
	}

	c := result.Results[0]
	if c.ID != 5 {
		t.Errorf("correspondent.ID = %d, want 5", c.ID)
	}
	if c.Name != "Acme Corp" {
		t.Errorf("correspondent.Name = %q, want %q", c.Name, "Acme Corp")
	}
	if c.Slug != "acme-corp" {
		t.Errorf("correspondent.Slug = %q, want %q", c.Slug, "acme-corp")
	}
	if c.Match != "acme" {
		t.Errorf("correspondent.Match = %q, want %q", c.Match, "acme")
	}
	if c.MatchingAlgorithm != 1 {
		t.Errorf("correspondent.MatchingAlgorithm = %d, want 1", c.MatchingAlgorithm)
	}
	if !c.IsInsensitive {
		t.Errorf("correspondent.IsInsensitive = false, want true")
	}
	if c.DocumentCount != 23 {
		t.Errorf("correspondent.DocumentCount = %d, want 23", c.DocumentCount)
	}
	if c.LastCorrespondence != "2024-06-15" {
		t.Errorf("correspondent.LastCorrespondence = %q, want %q", c.LastCorrespondence, "2024-06-15")
	}
	if c.Owner == nil || *c.Owner != 1 {
		t.Errorf("correspondent.Owner = %v, want 1", c.Owner)
	}
	if !c.UserCanChange {
		t.Errorf("correspondent.UserCanChange = false, want true")
	}
}

func TestClient_SearchCorrespondents_EmptyQuery(t *testing.T) {
	t.Parallel()

	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		assertQueryParamNotSet(t, r, "name__icontains")
		assertQueryParam(t, r, "page", "1")
		assertQueryParam(t, r, "page_size", "25")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(paginatedResponse([]any{
			sampleCorrespondentRaw,
		}))
	})
	defer srv.Close()

	client := newClient(srv.URL)
	result, err := client.SearchCorrespondents(context.Background(), "", 1, 25)
	if err != nil {
		t.Fatalf("SearchCorrespondents() error = %v", err)
	}
	if result.Total != 1 {
		t.Errorf("Total = %d, want 1", result.Total)
	}
	if len(result.Results) != 1 {
		t.Fatalf("len(Results) = %d, want 1", len(result.Results))
	}
}

func TestClient_SearchCorrespondents_Pagination(t *testing.T) {
	t.Parallel()

	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		assertQueryParam(t, r, "page", "2")
		assertQueryParam(t, r, "page_size", "50")

		// Return 2 correspondents for page 2.
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(paginatedResponse([]any{
			sampleCorrespondentRaw,
			map[string]any{
				"id":                  8,
				"slug":                "xyz-llc",
				"name":                "XYZ LLC",
				"match":               "",
				"matching_algorithm":  0,
				"is_insensitive":      false,
				"document_count":      5,
				"last_correspondence": "2024-02-10",
				"owner":               nil,
				"user_can_change":     true,
			},
		}))
	})
	defer srv.Close()

	client := newClient(srv.URL)
	result, err := client.SearchCorrespondents(context.Background(), "", 2, 50)
	if err != nil {
		t.Fatalf("SearchCorrespondents() error = %v", err)
	}
	if result.Total != 2 {
		t.Errorf("Total = %d, want 2", result.Total)
	}
	if len(result.Results) != 2 {
		t.Fatalf("len(Results) = %d, want 2", len(result.Results))
	}
	if result.Results[0].ID != 5 {
		t.Errorf("Results[0].ID = %d, want 5", result.Results[0].ID)
	}
	if result.Results[1].ID != 8 {
		t.Errorf("Results[1].ID = %d, want 8", result.Results[1].ID)
	}
}

func TestClient_SearchCorrespondents_EmptyResults(t *testing.T) {
	t.Parallel()

	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(paginatedResponse([]any{}))
	})
	defer srv.Close()

	client := newClient(srv.URL)
	result, err := client.SearchCorrespondents(context.Background(), "zzz_nonexistent", 1, 25)
	if err != nil {
		t.Fatalf("SearchCorrespondents() error = %v", err)
	}
	if result.Total != 0 {
		t.Errorf("Total = %d, want 0", result.Total)
	}
	if len(result.Results) != 0 {
		t.Errorf("len(Results) = %d, want 0", len(result.Results))
	}
}

// ---------------------------------------------------------------------------
// Client.ListTags
// ---------------------------------------------------------------------------

func TestClient_ListTags_All(t *testing.T) {
	t.Parallel()

	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		assertAuthHeader(t, r)
		assertAcceptHeader(t, r)

		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/api/tags/") {
			t.Errorf("path = %s, want /api/tags/", r.URL.Path)
		}
		assertQueryParamNotSet(t, r, "name__icontains")
		assertQueryParam(t, r, "page", "1")
		assertQueryParam(t, r, "page_size", "25")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(paginatedResponse([]any{
			sampleTagRaw,
		}))
	})
	defer srv.Close()

	client := newClient(srv.URL)
	result, err := client.ListTags(context.Background(), "", 1, 25)
	if err != nil {
		t.Fatalf("ListTags() error = %v", err)
	}
	if result.Total != 1 {
		t.Errorf("Total = %d, want 1", result.Total)
	}
	if len(result.Results) != 1 {
		t.Fatalf("len(Results) = %d, want 1", len(result.Results))
	}

	tag := result.Results[0]
	if tag.ID != 7 {
		t.Errorf("tag.ID = %d, want 7", tag.ID)
	}
	if tag.Slug != "invoice" {
		t.Errorf("tag.Slug = %q, want %q", tag.Slug, "invoice")
	}
	if tag.Name != "Invoice" {
		t.Errorf("tag.Name = %q, want %q", tag.Name, "Invoice")
	}
	if tag.Color != "#a6cee3" {
		t.Errorf("tag.Color = %q, want %q", tag.Color, "#a6cee3")
	}
	if tag.TextColor != "#000000" {
		t.Errorf("tag.TextColor = %q, want %q", tag.TextColor, "#000000")
	}
	if tag.Match != "invoice" {
		t.Errorf("tag.Match = %q, want %q", tag.Match, "invoice")
	}
	if tag.MatchingAlgorithm != 1 {
		t.Errorf("tag.MatchingAlgorithm = %d, want 1", tag.MatchingAlgorithm)
	}
	if !tag.IsInsensitive {
		t.Errorf("tag.IsInsensitive = false, want true")
	}
	if tag.IsInboxTag {
		t.Errorf("tag.IsInboxTag = true, want false")
	}
	if tag.DocumentCount != 15 {
		t.Errorf("tag.DocumentCount = %d, want 15", tag.DocumentCount)
	}
	if tag.Owner != nil {
		t.Errorf("tag.Owner = %v, want nil", tag.Owner)
	}
	if tag.Parent != nil {
		t.Errorf("tag.Parent = %v, want nil", tag.Parent)
	}
	if !tag.UserCanChange {
		t.Errorf("tag.UserCanChange = false, want true")
	}
}

func TestClient_ListTags_FilterByName(t *testing.T) {
	t.Parallel()

	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		assertQueryParam(t, r, "name__icontains", "invoice")
		assertQueryParam(t, r, "page", "1")
		assertQueryParam(t, r, "page_size", "25")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(paginatedResponse([]any{
			sampleTagRaw,
		}))
	})
	defer srv.Close()

	client := newClient(srv.URL)
	result, err := client.ListTags(context.Background(), "invoice", 1, 25)
	if err != nil {
		t.Fatalf("ListTags() error = %v", err)
	}
	if result.Total != 1 {
		t.Errorf("Total = %d, want 1", result.Total)
	}
	if len(result.Results) != 1 {
		t.Fatalf("len(Results) = %d, want 1", len(result.Results))
	}
	if result.Results[0].Name != "Invoice" {
		t.Errorf("tag.Name = %q, want %q", result.Results[0].Name, "Invoice")
	}
}

func TestClient_ListTags_Pagination(t *testing.T) {
	t.Parallel()

	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		assertQueryParam(t, r, "page", "3")
		assertQueryParam(t, r, "page_size", "10")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(paginatedResponse([]any{
			sampleTagRaw,
		}))
	})
	defer srv.Close()

	client := newClient(srv.URL)
	result, err := client.ListTags(context.Background(), "", 3, 10)
	if err != nil {
		t.Fatalf("ListTags() error = %v", err)
	}
	if result.Total != 1 {
		t.Errorf("Total = %d, want 1", result.Total)
	}
	if len(result.Results) != 1 {
		t.Fatalf("len(Results) = %d, want 1", len(result.Results))
	}
}

func TestClient_ListTags_EmptyResults(t *testing.T) {
	t.Parallel()

	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(paginatedResponse([]any{}))
	})
	defer srv.Close()

	client := newClient(srv.URL)
	result, err := client.ListTags(context.Background(), "nonexistent", 1, 25)
	if err != nil {
		t.Fatalf("ListTags() error = %v", err)
	}
	if result.Total != 0 {
		t.Errorf("Total = %d, want 0", result.Total)
	}
	if len(result.Results) != 0 {
		t.Errorf("len(Results) = %d, want 0", len(result.Results))
	}
}

// ---------------------------------------------------------------------------
// Error handling
// ---------------------------------------------------------------------------

func TestClient_Error_401(t *testing.T) {
	t.Parallel()

	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"detail":"Invalid token"}`))
	})
	defer srv.Close()

	client := newClient(srv.URL)
	_, err := client.Search(context.Background(), domain.SearchDocumentsParams{Page: 1, PageSize: 25}) //nolint:exhaustruct
	if err == nil {
		t.Fatal("Search() expected error, got nil")
	}
	if !errors.Is(err, paperless.ErrAPIClient) {
		t.Errorf("error does not wrap ErrAPIClient: %v", err)
	}
}

func TestClient_Error_500(t *testing.T) {
	t.Parallel()

	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`Internal Server Error`))
	})
	defer srv.Close()

	client := newClient(srv.URL)
	_, err := client.GetByID(context.Background(), 1)
	if err == nil {
		t.Fatal("GetByID() expected error, got nil")
	}
	if !errors.Is(err, paperless.ErrAPIClient) {
		t.Errorf("error does not wrap ErrAPIClient: %v", err)
	}
}

func TestClient_Error_NetworkError(t *testing.T) {
	t.Parallel()

	// Point at a server that will never respond (unreachable).
	client := paperless.NewClient("http://127.0.0.1:1", "test-token", testHTTPClient)

	_, err := client.Search(context.Background(), domain.SearchDocumentsParams{Page: 1, PageSize: 25}) //nolint:exhaustruct
	if err == nil {
		t.Fatal("Search() expected error for unreachable server, got nil")
	}
}

func TestClient_Error_NetworkError_GetByID(t *testing.T) {
	t.Parallel()

	client := paperless.NewClient("http://127.0.0.1:1", "test-token", testHTTPClient)

	_, err := client.GetByID(context.Background(), 1)
	if err == nil {
		t.Fatal("GetByID() expected error for unreachable server, got nil")
	}
}

func TestClient_Error_NetworkError_SearchCorrespondents(t *testing.T) {
	t.Parallel()

	client := paperless.NewClient("http://127.0.0.1:1", "test-token", testHTTPClient)

	_, err := client.SearchCorrespondents(context.Background(), "", 1, 25)
	if err == nil {
		t.Fatal("SearchCorrespondents() expected error for unreachable server, got nil")
	}
}

func TestClient_Error_NetworkError_ListTags(t *testing.T) {
	t.Parallel()

	client := paperless.NewClient("http://127.0.0.1:1", "test-token", testHTTPClient)

	_, err := client.ListTags(context.Background(), "", 1, 25)
	if err == nil {
		t.Fatal("ListTags() expected error for unreachable server, got nil")
	}
}

// ---------------------------------------------------------------------------
// Error handling for each endpoint with 401 and 500
// ---------------------------------------------------------------------------

func TestClient_SearchCorrespondents_Error_401(t *testing.T) {
	t.Parallel()

	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"detail":"Invalid token"}`))
	})
	defer srv.Close()

	client := newClient(srv.URL)
	_, err := client.SearchCorrespondents(context.Background(), "anything", 1, 25)
	if err == nil {
		t.Fatal("SearchCorrespondents() expected error, got nil")
	}
	if !errors.Is(err, paperless.ErrAPIClient) {
		t.Errorf("error does not wrap ErrAPIClient: %v", err)
	}
}

func TestClient_SearchCorrespondents_Error_500(t *testing.T) {
	t.Parallel()

	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`Server Error`))
	})
	defer srv.Close()

	client := newClient(srv.URL)
	_, err := client.SearchCorrespondents(context.Background(), "anything", 1, 25)
	if err == nil {
		t.Fatal("SearchCorrespondents() expected error, got nil")
	}
	if !errors.Is(err, paperless.ErrAPIClient) {
		t.Errorf("error does not wrap ErrAPIClient: %v", err)
	}
}

func TestClient_ListTags_Error_401(t *testing.T) {
	t.Parallel()

	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"detail":"Invalid token"}`))
	})
	defer srv.Close()

	client := newClient(srv.URL)
	_, err := client.ListTags(context.Background(), "", 1, 25)
	if err == nil {
		t.Fatal("ListTags() expected error, got nil")
	}
	if !errors.Is(err, paperless.ErrAPIClient) {
		t.Errorf("error does not wrap ErrAPIClient: %v", err)
	}
}

func TestClient_ListTags_Error_500(t *testing.T) {
	t.Parallel()

	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`Server Error`))
	})
	defer srv.Close()

	client := newClient(srv.URL)
	_, err := client.ListTags(context.Background(), "", 1, 25)
	if err == nil {
		t.Fatal("ListTags() expected error, got nil")
	}
	if !errors.Is(err, paperless.ErrAPIClient) {
		t.Errorf("error does not wrap ErrAPIClient: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Edge: malformed JSON from server
// ---------------------------------------------------------------------------

func TestClient_Search_MalformedJSON(t *testing.T) {
	t.Parallel()

	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{invalid json`))
	})
	defer srv.Close()

	client := newClient(srv.URL)
	_, err := client.Search(context.Background(), domain.SearchDocumentsParams{Page: 1, PageSize: 25}) //nolint:exhaustruct
	if err == nil {
		t.Fatal("Search() expected error for malformed JSON, got nil")
	}
}

func TestClient_GetByID_MalformedJSON(t *testing.T) {
	t.Parallel()

	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{invalid json`))
	})
	defer srv.Close()

	client := newClient(srv.URL)
	_, err := client.GetByID(context.Background(), 123)
	if err == nil {
		t.Fatal("GetByID() expected error for malformed JSON, got nil")
	}
}

// ---------------------------------------------------------------------------
// Edge: context cancellation
// ---------------------------------------------------------------------------

func TestClient_Search_ContextCancelled(t *testing.T) {
	t.Parallel()

	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		// Wait for context cancellation.
		<-r.Context().Done()
	})
	defer srv.Close()

	client := newClient(srv.URL)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := client.Search(ctx, domain.SearchDocumentsParams{Page: 1, PageSize: 25}) //nolint:exhaustruct
	if err == nil {
		t.Fatal("Search() expected error for cancelled context, got nil")
	}
}

func TestClient_GetByID_ContextCancelled(t *testing.T) {
	t.Parallel()

	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	})
	defer srv.Close()

	client := newClient(srv.URL)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := client.GetByID(ctx, 123)
	if err == nil {
		t.Fatal("GetByID() expected error for cancelled context, got nil")
	}
}

// ---------------------------------------------------------------------------
// Edge: HttpServer parameter URL query parsing for Search
// ---------------------------------------------------------------------------

func TestClient_Search_QueryParamsUsed(t *testing.T) {
	t.Parallel()

	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()

		// Verify all query parameters are set.
		if q.Get("query") != "test" {
			t.Errorf("query param 'query' = %q, want %q", q.Get("query"), "test")
		}
		if q.Get("correspondent__id") != "3" {
			t.Errorf("query param 'correspondent__id' = %q, want %q", q.Get("correspondent__id"), "3")
		}
		tagIDs := q["tags__id__all"]
		if len(tagIDs) != 2 || tagIDs[0] != "1" || tagIDs[1] != "2" {
			t.Errorf("query param 'tags__id__all' = %v, want [1 2]", tagIDs)
		}
		if q.Get("created__date__gte") != "2024-01-01" {
			t.Errorf("query param 'created__date__gte' = %q, want %q", q.Get("created__date__gte"), "2024-01-01")
		}
		if q.Get("created__date__lte") != "2024-12-31" {
			t.Errorf("query param 'created__date__lte' = %q, want %q", q.Get("created__date__lte"), "2024-12-31")
		}
		if q.Get("page") != "2" {
			t.Errorf("query param 'page' = %q, want %q", q.Get("page"), "2")
		}
		if q.Get("page_size") != "50" {
			t.Errorf("query param 'page_size' = %q, want %q", q.Get("page_size"), "50")
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(paginatedResponse([]any{
			sampleDocumentRaw,
		}))
	})
	defer srv.Close()

	client := newClient(srv.URL)
	result, err := client.Search(context.Background(), domain.SearchDocumentsParams{
		Query:           "test",
		CorrespondentID: 3,
		TagIDs:          []int{1, 2},
		CreatedAfter:    "2024-01-01",
		CreatedBefore:   "2024-12-31",
		Page:            2,
		PageSize:        50,
	})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if result.Total != 1 {
		t.Errorf("Total = %d, want 1", result.Total)
	}
}

// ---------------------------------------------------------------------------
// Exported error sentinel check
// ---------------------------------------------------------------------------

func TestClient_ErrAPIClient_Exported(t *testing.T) {
	t.Parallel()

	if paperless.ErrAPIClient == nil {
		t.Fatal("paperless.ErrAPIClient is nil")
	}
}

// ---------------------------------------------------------------------------
// Adapter types (CorrespondentRepo, TagRepo)
// ---------------------------------------------------------------------------

func TestClient_CorrespondentRepo_Search(t *testing.T) {
	t.Parallel()

	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/api/correspondents/") {
			t.Errorf("path = %s, want /api/correspondents/", r.URL.Path)
		}
		assertQueryParam(t, r, "name__icontains", "acme")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(paginatedResponse([]any{
			sampleCorrespondentRaw,
		}))
	})
	defer srv.Close()

	client := newClient(srv.URL)
	repo := paperless.NewCorrespondentRepo(client)

	result, err := repo.Search(context.Background(), "acme", 1, 25)
	if err != nil {
		t.Fatalf("CorrespondentRepo.Search() error = %v", err)
	}
	if result.Total != 1 {
		t.Errorf("Total = %d, want 1", result.Total)
	}
	if len(result.Results) != 1 {
		t.Fatalf("len(Results) = %d, want 1", len(result.Results))
	}
	if result.Results[0].Name != "Acme Corp" {
		t.Errorf("Name = %q, want %q", result.Results[0].Name, "Acme Corp")
	}
}

func TestClient_TagRepo_List(t *testing.T) {
	t.Parallel()

	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/api/tags/") {
			t.Errorf("path = %s, want /api/tags/", r.URL.Path)
		}
		assertQueryParamNotSet(t, r, "name__icontains")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(paginatedResponse([]any{
			sampleTagRaw,
		}))
	})
	defer srv.Close()

	client := newClient(srv.URL)
	repo := paperless.NewTagRepo(client)

	result, err := repo.List(context.Background(), "", 1, 25)
	if err != nil {
		t.Fatalf("TagRepo.List() error = %v", err)
	}
	if result.Total != 1 {
		t.Errorf("Total = %d, want 1", result.Total)
	}
	if len(result.Results) != 1 {
		t.Fatalf("len(Results) = %d, want 1", len(result.Results))
	}
	if result.Results[0].Name != "Invoice" {
		t.Errorf("Name = %q, want %q", result.Results[0].Name, "Invoice")
	}
}

// ---------------------------------------------------------------------------
// Client.GetCorrespondentByID
// ---------------------------------------------------------------------------

func TestClient_GetCorrespondentByID_Existing(t *testing.T) {
	t.Parallel()

	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		assertAuthHeader(t, r)
		assertAcceptHeader(t, r)

		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/api/correspondents/5/") {
			t.Errorf("path = %s, want /api/correspondents/5/", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(sampleCorrespondentRaw)
	})
	defer srv.Close()

	client := newClient(srv.URL)
	corr, err := client.GetCorrespondentByID(context.Background(), 5)
	if err != nil {
		t.Fatalf("GetCorrespondentByID() error = %v", err)
	}
	if corr.ID != 5 {
		t.Errorf("corr.ID = %d, want 5", corr.ID)
	}
	if corr.Name != "Acme Corp" {
		t.Errorf("corr.Name = %q, want %q", corr.Name, "Acme Corp")
	}
	if corr.Slug != "acme-corp" {
		t.Errorf("corr.Slug = %q, want %q", corr.Slug, "acme-corp")
	}
	if corr.Match != "acme" {
		t.Errorf("corr.Match = %q, want %q", corr.Match, "acme")
	}
	if corr.MatchingAlgorithm != 1 {
		t.Errorf("corr.MatchingAlgorithm = %d, want 1", corr.MatchingAlgorithm)
	}
	if !corr.IsInsensitive {
		t.Errorf("corr.IsInsensitive = false, want true")
	}
	if corr.DocumentCount != 23 {
		t.Errorf("corr.DocumentCount = %d, want 23", corr.DocumentCount)
	}
	if corr.LastCorrespondence != "2024-06-15" {
		t.Errorf("corr.LastCorrespondence = %q, want %q", corr.LastCorrespondence, "2024-06-15")
	}
	if corr.Owner == nil || *corr.Owner != 1 {
		t.Errorf("corr.Owner = %v, want 1", corr.Owner)
	}
	if !corr.UserCanChange {
		t.Errorf("corr.UserCanChange = false, want true")
	}
}

func TestClient_GetCorrespondentByID_NotFound(t *testing.T) {
	t.Parallel()

	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"detail":"Not found"}`))
	})
	defer srv.Close()

	client := newClient(srv.URL)
	_, err := client.GetCorrespondentByID(context.Background(), 999)
	if err == nil {
		t.Fatal("GetCorrespondentByID() expected error, got nil")
	}
	if !errors.Is(err, paperless.ErrAPIClient) {
		t.Errorf("error does not wrap ErrAPIClient: %v", err)
	}
}

func TestClient_GetCorrespondentByID_MalformedJSON(t *testing.T) {
	t.Parallel()

	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{invalid json`))
	})
	defer srv.Close()

	client := newClient(srv.URL)
	_, err := client.GetCorrespondentByID(context.Background(), 5)
	if err == nil {
		t.Fatal("GetCorrespondentByID() expected error for malformed JSON, got nil")
	}
}

func TestClient_GetCorrespondentByID_ContextCancelled(t *testing.T) {
	t.Parallel()

	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	})
	defer srv.Close()

	client := newClient(srv.URL)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := client.GetCorrespondentByID(ctx, 5)
	if err == nil {
		t.Fatal("GetCorrespondentByID() expected error for cancelled context, got nil")
	}
}

func TestClient_GetCorrespondentByID_Error_401(t *testing.T) {
	t.Parallel()

	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"detail":"Invalid token"}`))
	})
	defer srv.Close()

	client := newClient(srv.URL)
	_, err := client.GetCorrespondentByID(context.Background(), 5)
	if err == nil {
		t.Fatal("GetCorrespondentByID() expected error, got nil")
	}
	if !errors.Is(err, paperless.ErrAPIClient) {
		t.Errorf("error does not wrap ErrAPIClient: %v", err)
	}
}

func TestClient_GetCorrespondentByID_Error_500(t *testing.T) {
	t.Parallel()

	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`Server Error`))
	})
	defer srv.Close()

	client := newClient(srv.URL)
	_, err := client.GetCorrespondentByID(context.Background(), 5)
	if err == nil {
		t.Fatal("GetCorrespondentByID() expected error, got nil")
	}
	if !errors.Is(err, paperless.ErrAPIClient) {
		t.Errorf("error does not wrap ErrAPIClient: %v", err)
	}
}

func TestClient_GetCorrespondentByID_NetworkError(t *testing.T) {
	t.Parallel()

	client := paperless.NewClient("http://127.0.0.1:1", "test-token", testHTTPClient)

	_, err := client.GetCorrespondentByID(context.Background(), 5)
	if err == nil {
		t.Fatal("GetCorrespondentByID() expected error for unreachable server, got nil")
	}
}

// ---------------------------------------------------------------------------
// Client.GetDocumentTypeByID
// ---------------------------------------------------------------------------

func TestClient_GetDocumentTypeByID_Existing(t *testing.T) {
	t.Parallel()

	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		assertAuthHeader(t, r)
		assertAcceptHeader(t, r)

		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/api/document_types/2/") {
			t.Errorf("path = %s, want /api/document_types/2/", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(sampleDocumentTypeRaw)
	})
	defer srv.Close()

	client := newClient(srv.URL)
	dt, err := client.GetDocumentTypeByID(context.Background(), 2)
	if err != nil {
		t.Fatalf("GetDocumentTypeByID() error = %v", err)
	}
	if dt.ID != 2 {
		t.Errorf("dt.ID = %d, want 2", dt.ID)
	}
	if dt.Slug != "invoice" {
		t.Errorf("dt.Slug = %q, want %q", dt.Slug, "invoice")
	}
	if dt.Name != "Invoice" {
		t.Errorf("dt.Name = %q, want %q", dt.Name, "Invoice")
	}
	if dt.Match != "invoice" {
		t.Errorf("dt.Match = %q, want %q", dt.Match, "invoice")
	}
	if dt.MatchingAlgorithm != 1 {
		t.Errorf("dt.MatchingAlgorithm = %d, want 1", dt.MatchingAlgorithm)
	}
	if !dt.IsInsensitive {
		t.Errorf("dt.IsInsensitive = false, want true")
	}
	if dt.DocumentCount != 15 {
		t.Errorf("dt.DocumentCount = %d, want 15", dt.DocumentCount)
	}
	if dt.Owner == nil || *dt.Owner != 1 {
		t.Errorf("dt.Owner = %v, want 1", dt.Owner)
	}
	if !dt.UserCanChange {
		t.Errorf("dt.UserCanChange = false, want true")
	}
}

func TestClient_GetDocumentTypeByID_NotFound(t *testing.T) {
	t.Parallel()

	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"detail":"Not found"}`))
	})
	defer srv.Close()

	client := newClient(srv.URL)
	_, err := client.GetDocumentTypeByID(context.Background(), 999)
	if err == nil {
		t.Fatal("GetDocumentTypeByID() expected error, got nil")
	}
	if !errors.Is(err, paperless.ErrAPIClient) {
		t.Errorf("error does not wrap ErrAPIClient: %v", err)
	}
}

func TestClient_GetDocumentTypeByID_MalformedJSON(t *testing.T) {
	t.Parallel()

	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{invalid json`))
	})
	defer srv.Close()

	client := newClient(srv.URL)
	_, err := client.GetDocumentTypeByID(context.Background(), 2)
	if err == nil {
		t.Fatal("GetDocumentTypeByID() expected error for malformed JSON, got nil")
	}
}

func TestClient_GetDocumentTypeByID_ContextCancelled(t *testing.T) {
	t.Parallel()

	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	})
	defer srv.Close()

	client := newClient(srv.URL)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := client.GetDocumentTypeByID(ctx, 2)
	if err == nil {
		t.Fatal("GetDocumentTypeByID() expected error for cancelled context, got nil")
	}
}

func TestClient_GetDocumentTypeByID_Error_401(t *testing.T) {
	t.Parallel()

	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"detail":"Invalid token"}`))
	})
	defer srv.Close()

	client := newClient(srv.URL)
	_, err := client.GetDocumentTypeByID(context.Background(), 2)
	if err == nil {
		t.Fatal("GetDocumentTypeByID() expected error, got nil")
	}
	if !errors.Is(err, paperless.ErrAPIClient) {
		t.Errorf("error does not wrap ErrAPIClient: %v", err)
	}
}

func TestClient_GetDocumentTypeByID_Error_500(t *testing.T) {
	t.Parallel()

	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`Server Error`))
	})
	defer srv.Close()

	client := newClient(srv.URL)
	_, err := client.GetDocumentTypeByID(context.Background(), 2)
	if err == nil {
		t.Fatal("GetDocumentTypeByID() expected error, got nil")
	}
	if !errors.Is(err, paperless.ErrAPIClient) {
		t.Errorf("error does not wrap ErrAPIClient: %v", err)
	}
}

func TestClient_GetDocumentTypeByID_NetworkError(t *testing.T) {
	t.Parallel()

	client := paperless.NewClient("http://127.0.0.1:1", "test-token", testHTTPClient)

	_, err := client.GetDocumentTypeByID(context.Background(), 2)
	if err == nil {
		t.Fatal("GetDocumentTypeByID() expected error for unreachable server, got nil")
	}
}

// ---------------------------------------------------------------------------
// Adapter types: CorrespondentRepo.GetByID, DocumentTypeRepo
// ---------------------------------------------------------------------------

func TestClient_CorrespondentRepo_GetByID(t *testing.T) {
	t.Parallel()

	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/api/correspondents/5/") {
			t.Errorf("path = %s, want /api/correspondents/5/", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(sampleCorrespondentRaw)
	})
	defer srv.Close()

	client := newClient(srv.URL)
	repo := paperless.NewCorrespondentRepo(client)

	corr, err := repo.GetByID(context.Background(), 5)
	if err != nil {
		t.Fatalf("CorrespondentRepo.GetByID() error = %v", err)
	}
	if corr.ID != 5 {
		t.Errorf("corr.ID = %d, want 5", corr.ID)
	}
	if corr.Name != "Acme Corp" {
		t.Errorf("corr.Name = %q, want %q", corr.Name, "Acme Corp")
	}
}

func TestClient_CorrespondentRepo_GetByID_NotFound(t *testing.T) {
	t.Parallel()

	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"detail":"Not found"}`))
	})
	defer srv.Close()

	client := newClient(srv.URL)
	repo := paperless.NewCorrespondentRepo(client)

	_, err := repo.GetByID(context.Background(), 999)
	if err == nil {
		t.Fatal("CorrespondentRepo.GetByID() expected error, got nil")
	}
	if !errors.Is(err, paperless.ErrAPIClient) {
		t.Errorf("error does not wrap ErrAPIClient: %v", err)
	}
}

func TestClient_DocumentTypeRepo_New(t *testing.T) {
	t.Parallel()

	client := newClient("http://example.com")
	repo := paperless.NewDocumentTypeRepo(client)
	if repo == nil {
		t.Fatal("NewDocumentTypeRepo() returned nil")
	}
}

func TestClient_DocumentTypeRepo_GetByID(t *testing.T) {
	t.Parallel()

	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/api/document_types/2/") {
			t.Errorf("path = %s, want /api/document_types/2/", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(sampleDocumentTypeRaw)
	})
	defer srv.Close()

	client := newClient(srv.URL)
	repo := paperless.NewDocumentTypeRepo(client)

	dt, err := repo.GetByID(context.Background(), 2)
	if err != nil {
		t.Fatalf("DocumentTypeRepo.GetByID() error = %v", err)
	}
	if dt.ID != 2 {
		t.Errorf("dt.ID = %d, want 2", dt.ID)
	}
	if dt.Name != "Invoice" {
		t.Errorf("dt.Name = %q, want %q", dt.Name, "Invoice")
	}
}

func TestClient_DocumentTypeRepo_GetByID_NotFound(t *testing.T) {
	t.Parallel()

	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"detail":"Not found"}`))
	})
	defer srv.Close()

	client := newClient(srv.URL)
	repo := paperless.NewDocumentTypeRepo(client)

	_, err := repo.GetByID(context.Background(), 999)
	if err == nil {
		t.Fatal("DocumentTypeRepo.GetByID() expected error, got nil")
	}
	if !errors.Is(err, paperless.ErrAPIClient) {
		t.Errorf("error does not wrap ErrAPIClient: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Edge: non-200 successful status codes
// ---------------------------------------------------------------------------

func TestClient_Search_Status201(t *testing.T) {
	t.Parallel()

	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated) // non-standard but still 2xx
		_ = json.NewEncoder(w).Encode(paginatedResponse([]any{
			sampleDocumentRaw,
		}))
	})
	defer srv.Close()

	client := newClient(srv.URL)
	result, err := client.Search(context.Background(), domain.SearchDocumentsParams{}) //nolint:exhaustruct
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if result.Total != 1 {
		t.Errorf("Total = %d, want 1", result.Total)
	}
}

// ---------------------------------------------------------------------------
// Helper: testing for exported error sentinel value
// ---------------------------------------------------------------------------

func TestClient_Error_ErrorMessageContainsStatusCodeAndDetail(t *testing.T) {
	t.Parallel()

	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"detail":"Not found"}`))
	})
	defer srv.Close()

	client := newClient(srv.URL)
	_, err := client.GetByID(context.Background(), 999)
	if err == nil {
		t.Fatal("GetByID() expected error, got nil")
	}

	errStr := err.Error()
	if !strings.Contains(errStr, "404") {
		t.Errorf("error message does not contain status code 404: %q", errStr)
	}
	if !strings.Contains(errStr, "Not found") {
		t.Errorf("error message should contain response detail: %q", errStr)
	}
	if !errors.Is(err, paperless.ErrAPIClient) {
		t.Errorf("error should wrap ErrAPIClient: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Search with explicit pagination values (defaults handled by normalizePagination
// at the handlers layer)
// ---------------------------------------------------------------------------

func TestClient_Search_WithExplicitPagination(t *testing.T) {
	t.Parallel()

	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		assertQueryParam(t, r, "page", "2")
		assertQueryParam(t, r, "page_size", "50")
		assertQueryParamNotSet(t, r, "query")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(paginatedResponse([]any{
			sampleDocumentRaw,
		}))
	})
	defer srv.Close()

	client := newClient(srv.URL)
	result, err := client.Search(context.Background(), domain.SearchDocumentsParams{ //nolint:exhaustruct
		Page:     2,
		PageSize: 50,
	})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if result.Total != 1 {
		t.Errorf("Total = %d, want 1", result.Total)
	}
}

// ---------------------------------------------------------------------------
// Regression: HTTP 300 is rejected (catches CONDITIONALS_BOUNDARY on >= 300)
// ---------------------------------------------------------------------------

func TestClient_Search_Status300_Rejected(t *testing.T) {
	t.Parallel()

	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusMultipleChoices)        // 300
		w.Write([]byte(`{"detail":"multiple choices"}`)) //nolint:errcheck,gosec
	})
	defer srv.Close()

	client := newClient(srv.URL)
	_, err := client.Search(context.Background(), domain.SearchDocumentsParams{}) //nolint:exhaustruct
	if err == nil {
		t.Fatal("expected error for HTTP 300")
	}
	if !errors.Is(err, paperless.ErrAPIClient) {
		t.Errorf("error should wrap ErrAPIClient, got %T", err)
	}
}
