package application

import (
	"context"
	"errors"
	"testing"

	"github.com/teran/mcp-paperless-ngx/domain"
)

// ---------------------------------------------------------------------------
// Mocks
// ---------------------------------------------------------------------------

var errMock = errors.New("mock error")

// mockDocRepo implements domain.DocumentRepository for testing.
type mockDocRepo struct {
	searchFn  func(ctx context.Context, params domain.SearchDocumentsParams) (*domain.PaginatedResult[domain.Document], error)
	getByIDFn func(ctx context.Context, id int) (*domain.Document, error)
}

func (m *mockDocRepo) Search(ctx context.Context, params domain.SearchDocumentsParams) (*domain.PaginatedResult[domain.Document], error) {
	return m.searchFn(ctx, params)
}

func (m *mockDocRepo) GetByID(ctx context.Context, id int) (*domain.Document, error) {
	return m.getByIDFn(ctx, id)
}

// mockCorrespondentRepo implements domain.CorrespondentRepository for testing.
type mockCorrespondentRepo struct {
	searchFn func(ctx context.Context, query string, page, pageSize int) (*domain.PaginatedResult[domain.Correspondent], error)
}

func (m *mockCorrespondentRepo) Search(ctx context.Context, query string, page, pageSize int) (*domain.PaginatedResult[domain.Correspondent], error) {
	return m.searchFn(ctx, query, page, pageSize)
}

// mockTagRepo implements domain.TagRepository for testing.
type mockTagRepo struct {
	listFn func(ctx context.Context, query string, page, pageSize int) (*domain.PaginatedResult[domain.Tag], error)
}

func (m *mockTagRepo) List(ctx context.Context, query string, page, pageSize int) (*domain.PaginatedResult[domain.Tag], error) {
	return m.listFn(ctx, query, page, pageSize)
}

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

func ctx() context.Context {
	return context.Background()
}

func ptrInt(v int) *int {
	return &v
}

func ptrStr(v string) *string {
	return &v
}

// ---------------------------------------------------------------------------
// DocumentService tests
// ---------------------------------------------------------------------------

func TestDocumentService_Search(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		expected := &domain.PaginatedResult[domain.Document]{
			Total: 2,
			Results: []domain.Document{
				{ID: 1, Title: "Doc 1"}, //nolint:exhaustruct
				{ID: 2, Title: "Doc 2"}, //nolint:exhaustruct
			},
		}

		repo := &mockDocRepo{ //nolint:exhaustruct
			searchFn: func(ctx context.Context, params domain.SearchDocumentsParams) (*domain.PaginatedResult[domain.Document], error) {
				if params.Page != 1 || params.PageSize != 25 {
					t.Errorf("unexpected params: %+v", params)
				}
				return expected, nil
			},
		}

		svc := NewDocumentService(repo)
		result, err := svc.Search(ctx(), domain.SearchDocumentsParams{Page: 1, PageSize: 25}) //nolint:exhaustruct
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Total != expected.Total {
			t.Errorf("expected total=%d, got %d", expected.Total, result.Total)
		}
		if len(result.Results) != len(expected.Results) {
			t.Fatalf("expected %d results, got %d", len(expected.Results), len(result.Results))
		}
		for i := range result.Results {
			if result.Results[i].ID != expected.Results[i].ID {
				t.Errorf("result[%d].ID = %d, want %d", i, result.Results[i].ID, expected.Results[i].ID)
			}
		}
	})

	t.Run("error propagation", func(t *testing.T) {
		repo := &mockDocRepo{ //nolint:exhaustruct
			searchFn: func(ctx context.Context, params domain.SearchDocumentsParams) (*domain.PaginatedResult[domain.Document], error) {
				return nil, errMock
			},
		}

		svc := NewDocumentService(repo)
		_, err := svc.Search(ctx(), domain.SearchDocumentsParams{}) //nolint:exhaustruct
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, errMock) {
			t.Errorf("expected wrapping of %v, got %v", errMock, err)
		}
	})

	t.Run("nil result", func(t *testing.T) {
		repo := &mockDocRepo{ //nolint:exhaustruct
			searchFn: func(ctx context.Context, params domain.SearchDocumentsParams) (*domain.PaginatedResult[domain.Document], error) {
				return nil, nil
			},
		}

		svc := NewDocumentService(repo)
		result, err := svc.Search(ctx(), domain.SearchDocumentsParams{}) //nolint:exhaustruct
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != nil {
			t.Errorf("expected nil result, got %+v", result)
		}
	})

	t.Run("empty results", func(t *testing.T) {
		expected := &domain.PaginatedResult[domain.Document]{
			Total:   0,
			Results: []domain.Document{},
		}

		repo := &mockDocRepo{ //nolint:exhaustruct
			searchFn: func(ctx context.Context, params domain.SearchDocumentsParams) (*domain.PaginatedResult[domain.Document], error) {
				return expected, nil
			},
		}

		svc := NewDocumentService(repo)
		result, err := svc.Search(ctx(), domain.SearchDocumentsParams{}) //nolint:exhaustruct
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Total != 0 {
			t.Errorf("expected total=0, got %d", result.Total)
		}
		if len(result.Results) != 0 {
			t.Errorf("expected 0 results, got %d", len(result.Results))
		}
	})
}

func TestDocumentService_GetByID(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		expected := &domain.Document{ID: 42, Title: "The Answer"} //nolint:exhaustruct
		called := false

		repo := &mockDocRepo{ //nolint:exhaustruct
			getByIDFn: func(ctx context.Context, id int) (*domain.Document, error) {
				called = true
				if id != 42 {
					t.Errorf("expected id=42, got %d", id)
				}
				return expected, nil
			},
		}

		svc := NewDocumentService(repo)
		doc, err := svc.GetByID(ctx(), 42)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !called {
			t.Error("expected repo.GetByID to be called")
		}
		if doc.ID != expected.ID || doc.Title != expected.Title {
			t.Errorf("got %+v, want %+v", doc, expected)
		}
	})

	t.Run("error propagation", func(t *testing.T) {
		repo := &mockDocRepo{ //nolint:exhaustruct
			getByIDFn: func(ctx context.Context, id int) (*domain.Document, error) {
				return nil, errMock
			},
		}

		svc := NewDocumentService(repo)
		_, err := svc.GetByID(ctx(), 1)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, errMock) {
			t.Errorf("expected wrapping of %v, got %v", errMock, err)
		}
	})

	t.Run("nil result", func(t *testing.T) {
		repo := &mockDocRepo{ //nolint:exhaustruct
			getByIDFn: func(ctx context.Context, id int) (*domain.Document, error) {
				return nil, nil
			},
		}

		svc := NewDocumentService(repo)
		doc, err := svc.GetByID(ctx(), 99)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if doc != nil {
			t.Errorf("expected nil document, got %+v", doc)
		}
	})
}

func TestDocumentService_GetByCorrespondent(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		expected := &domain.PaginatedResult[domain.Document]{
			Total: 1,
			Results: []domain.Document{
				{ID: 10, Title: "Correspondent Doc", Correspondent: ptrInt(7)}, //nolint:exhaustruct
			},
		}

		var capturedParams domain.SearchDocumentsParams

		repo := &mockDocRepo{ //nolint:exhaustruct
			searchFn: func(ctx context.Context, params domain.SearchDocumentsParams) (*domain.PaginatedResult[domain.Document], error) {
				capturedParams = params
				return expected, nil
			},
		}

		svc := NewDocumentService(repo)
		result, err := svc.GetByCorrespondent(ctx(), 7, 1, 50)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify the correct params were passed
		if capturedParams.CorrespondentID != 7 {
			t.Errorf("expected CorrespondentID=7, got %d", capturedParams.CorrespondentID)
		}
		if capturedParams.Page != 1 {
			t.Errorf("expected Page=1, got %d", capturedParams.Page)
		}
		if capturedParams.PageSize != 50 {
			t.Errorf("expected PageSize=50, got %d", capturedParams.PageSize)
		}
		if capturedParams.Query != "" {
			t.Errorf("expected empty Query, got %q", capturedParams.Query)
		}
		if len(capturedParams.TagIDs) != 0 {
			t.Errorf("expected empty TagIDs, got %v", capturedParams.TagIDs)
		}

		// Verify the result
		if result.Total != expected.Total {
			t.Errorf("expected total=%d, got %d", expected.Total, result.Total)
		}
		if len(result.Results) != len(expected.Results) {
			t.Fatalf("expected %d results, got %d", len(expected.Results), len(result.Results))
		}
		if result.Results[0].ID != expected.Results[0].ID {
			t.Errorf("result[0].ID = %d, want %d", result.Results[0].ID, expected.Results[0].ID)
		}
	})

	t.Run("zero correspondent ID", func(t *testing.T) {
		repo := &mockDocRepo{ //nolint:exhaustruct
			searchFn: func(ctx context.Context, params domain.SearchDocumentsParams) (*domain.PaginatedResult[domain.Document], error) {
				return &domain.PaginatedResult[domain.Document]{Total: 0, Results: []domain.Document{}}, nil
			},
		}

		svc := NewDocumentService(repo)
		result, err := svc.GetByCorrespondent(ctx(), 0, 1, 10)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Total != 0 {
			t.Errorf("expected total=0, got %d", result.Total)
		}
	})

	t.Run("error propagation", func(t *testing.T) {
		repo := &mockDocRepo{ //nolint:exhaustruct
			searchFn: func(ctx context.Context, params domain.SearchDocumentsParams) (*domain.PaginatedResult[domain.Document], error) {
				return nil, errMock
			},
		}

		svc := NewDocumentService(repo)
		_, err := svc.GetByCorrespondent(ctx(), 1, 1, 10)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, errMock) {
			t.Errorf("expected wrapping of %v, got %v", errMock, err)
		}
	})
}

func TestDocumentService_GetByTag(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		expected := &domain.PaginatedResult[domain.Document]{
			Total: 2,
			Results: []domain.Document{
				{ID: 5, Title: "Tagged Doc 1", Tags: []int{3}},    //nolint:exhaustruct
				{ID: 6, Title: "Tagged Doc 2", Tags: []int{3, 7}}, //nolint:exhaustruct
			},
		}

		var capturedParams domain.SearchDocumentsParams

		repo := &mockDocRepo{ //nolint:exhaustruct
			searchFn: func(ctx context.Context, params domain.SearchDocumentsParams) (*domain.PaginatedResult[domain.Document], error) {
				capturedParams = params
				return expected, nil
			},
		}

		svc := NewDocumentService(repo)
		result, err := svc.GetByTag(ctx(), 3, 2, 20)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify the correct params were passed
		if len(capturedParams.TagIDs) != 1 || capturedParams.TagIDs[0] != 3 {
			t.Errorf("expected TagIDs=[3], got %v", capturedParams.TagIDs)
		}
		if capturedParams.Page != 2 {
			t.Errorf("expected Page=2, got %d", capturedParams.Page)
		}
		if capturedParams.PageSize != 20 {
			t.Errorf("expected PageSize=20, got %d", capturedParams.PageSize)
		}
		if capturedParams.CorrespondentID != 0 {
			t.Errorf("expected CorrespondentID=0, got %d", capturedParams.CorrespondentID)
		}

		// Verify the result
		if result.Total != expected.Total {
			t.Errorf("expected total=%d, got %d", expected.Total, result.Total)
		}
		if len(result.Results) != len(expected.Results) {
			t.Fatalf("expected %d results, got %d", len(expected.Results), len(result.Results))
		}
	})

	t.Run("error propagation", func(t *testing.T) {
		repo := &mockDocRepo{ //nolint:exhaustruct
			searchFn: func(ctx context.Context, params domain.SearchDocumentsParams) (*domain.PaginatedResult[domain.Document], error) {
				return nil, errMock
			},
		}

		svc := NewDocumentService(repo)
		_, err := svc.GetByTag(ctx(), 1, 1, 10)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, errMock) {
			t.Errorf("expected wrapping of %v, got %v", errMock, err)
		}
	})

	t.Run("empty tag ID list on repo call", func(t *testing.T) {
		repo := &mockDocRepo{ //nolint:exhaustruct
			searchFn: func(ctx context.Context, params domain.SearchDocumentsParams) (*domain.PaginatedResult[domain.Document], error) {
				if len(params.TagIDs) != 1 || params.TagIDs[0] != 99 {
					t.Errorf("expected TagIDs=[99], got %v", params.TagIDs)
				}
				return &domain.PaginatedResult[domain.Document]{Total: 1, Results: []domain.Document{{ID: 1}}}, nil //nolint:exhaustruct
			},
		}

		svc := NewDocumentService(repo)
		result, err := svc.GetByTag(ctx(), 99, 1, 10)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Total != 1 {
			t.Errorf("expected total=1, got %d", result.Total)
		}
	})
}

func TestDocumentService_FulltextSearch(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		expected := &domain.PaginatedResult[domain.Document]{
			Total: 1,
			Results: []domain.Document{
				{ID: 100, Title: "Search Result", SearchHit: &domain.SearchHit{Score: 1.5, Rank: 1}}, //nolint:exhaustruct
			},
		}

		var capturedParams domain.SearchDocumentsParams

		repo := &mockDocRepo{ //nolint:exhaustruct
			searchFn: func(ctx context.Context, params domain.SearchDocumentsParams) (*domain.PaginatedResult[domain.Document], error) {
				capturedParams = params
				return expected, nil
			},
		}

		svc := NewDocumentService(repo)
		result, err := svc.FulltextSearch(ctx(), "important query", 1, 25)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify the correct params were passed
		if capturedParams.Query != "important query" {
			t.Errorf("expected Query=%q, got %q", "important query", capturedParams.Query)
		}
		if capturedParams.Page != 1 {
			t.Errorf("expected Page=1, got %d", capturedParams.Page)
		}
		if capturedParams.PageSize != 25 {
			t.Errorf("expected PageSize=25, got %d", capturedParams.PageSize)
		}

		// Verify the result
		if result.Total != expected.Total {
			t.Errorf("expected total=%d, got %d", expected.Total, result.Total)
		}
		if len(result.Results) != len(expected.Results) {
			t.Fatalf("expected %d results, got %d", len(expected.Results), len(result.Results))
		}
		if result.Results[0].ID != 100 {
			t.Errorf("expected ID=100, got %d", result.Results[0].ID)
		}
		if result.Results[0].SearchHit.Score != 1.5 {
			t.Errorf("expected Score=1.5, got %f", result.Results[0].SearchHit.Score)
		}
	})

	t.Run("error propagation", func(t *testing.T) {
		repo := &mockDocRepo{ //nolint:exhaustruct
			searchFn: func(ctx context.Context, params domain.SearchDocumentsParams) (*domain.PaginatedResult[domain.Document], error) {
				return nil, errMock
			},
		}

		svc := NewDocumentService(repo)
		_, err := svc.FulltextSearch(ctx(), "query", 1, 10)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, errMock) {
			t.Errorf("expected wrapping of %v, got %v", errMock, err)
		}
	})

	t.Run("empty query string", func(t *testing.T) {
		repo := &mockDocRepo{ //nolint:exhaustruct
			searchFn: func(ctx context.Context, params domain.SearchDocumentsParams) (*domain.PaginatedResult[domain.Document], error) {
				if params.Query != "" {
					t.Errorf("expected empty query, got %q", params.Query)
				}
				return &domain.PaginatedResult[domain.Document]{Total: 0, Results: []domain.Document{}}, nil
			},
		}

		svc := NewDocumentService(repo)
		result, err := svc.FulltextSearch(ctx(), "", 1, 10)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Total != 0 {
			t.Errorf("expected total=0, got %d", result.Total)
		}
	})
}

// ---------------------------------------------------------------------------
// DocumentService — error message wrapping tests
// ---------------------------------------------------------------------------

func TestDocumentService_ErrorWrapping(t *testing.T) {
	tests := []struct {
		name    string
		method  func(svc *DocumentService, ctx context.Context) error
		wantMsg string
	}{
		{
			name: "Search",
			method: func(svc *DocumentService, ctx context.Context) error {
				_, err := svc.Search(ctx, domain.SearchDocumentsParams{Page: 1, PageSize: 10}) //nolint:exhaustruct
				return err
			},
			wantMsg: "search documents: mock error",
		},
		{
			name: "GetByID",
			method: func(svc *DocumentService, ctx context.Context) error {
				_, err := svc.GetByID(ctx, 1)
				return err
			},
			wantMsg: "get document: mock error",
		},
		{
			name: "GetByCorrespondent",
			method: func(svc *DocumentService, ctx context.Context) error {
				_, err := svc.GetByCorrespondent(ctx, 1, 1, 10)
				return err
			},
			wantMsg: "get documents by correspondent: mock error",
		},
		{
			name: "GetByTag",
			method: func(svc *DocumentService, ctx context.Context) error {
				_, err := svc.GetByTag(ctx, 1, 1, 10)
				return err
			},
			wantMsg: "get documents by tag: mock error",
		},
		{
			name: "FulltextSearch",
			method: func(svc *DocumentService, ctx context.Context) error {
				_, err := svc.FulltextSearch(ctx, "test", 1, 10)
				return err
			},
			wantMsg: "fulltext search: mock error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockDocRepo{
				searchFn: func(ctx context.Context, params domain.SearchDocumentsParams) (*domain.PaginatedResult[domain.Document], error) {
					return nil, errMock
				},
				getByIDFn: func(ctx context.Context, id int) (*domain.Document, error) {
					return nil, errMock
				},
			}

			svc := NewDocumentService(repo)
			err := tt.method(svc, ctx())
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if err.Error() != tt.wantMsg {
				t.Errorf("expected error message %q, got %q", tt.wantMsg, err.Error())
			}
			if !errors.Is(err, errMock) {
				t.Errorf("expected %v to wrap %v", err, errMock)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// CorrespondentService tests
// ---------------------------------------------------------------------------

func TestCorrespondentService_Search(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		expected := &domain.PaginatedResult[domain.Correspondent]{
			Total: 2,
			Results: []domain.Correspondent{
				{ID: 1, Name: "Alice"}, //nolint:exhaustruct
				{ID: 2, Name: "Bob"},   //nolint:exhaustruct
			},
		}

		var capturedQuery string
		var capturedPage, capturedPageSize int

		repo := &mockCorrespondentRepo{
			searchFn: func(ctx context.Context, query string, page, pageSize int) (*domain.PaginatedResult[domain.Correspondent], error) {
				capturedQuery = query
				capturedPage = page
				capturedPageSize = pageSize
				return expected, nil
			},
		}

		svc := NewCorrespondentService(repo)
		result, err := svc.Search(ctx(), "alice", 1, 25)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify params forwarded correctly
		if capturedQuery != "alice" {
			t.Errorf("expected query=%q, got %q", "alice", capturedQuery)
		}
		if capturedPage != 1 {
			t.Errorf("expected page=1, got %d", capturedPage)
		}
		if capturedPageSize != 25 {
			t.Errorf("expected pageSize=25, got %d", capturedPageSize)
		}

		// Verify result
		if result.Total != expected.Total {
			t.Errorf("expected total=%d, got %d", expected.Total, result.Total)
		}
		if len(result.Results) != len(expected.Results) {
			t.Fatalf("expected %d results, got %d", len(expected.Results), len(result.Results))
		}
		for i, r := range result.Results {
			if r.ID != expected.Results[i].ID || r.Name != expected.Results[i].Name {
				t.Errorf("result[%d] = %+v, want %+v", i, r, expected.Results[i])
			}
		}
	})

	t.Run("error propagation", func(t *testing.T) {
		repo := &mockCorrespondentRepo{
			searchFn: func(ctx context.Context, query string, page, pageSize int) (*domain.PaginatedResult[domain.Correspondent], error) {
				return nil, errMock
			},
		}

		svc := NewCorrespondentService(repo)
		_, err := svc.Search(ctx(), "test", 1, 10)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, errMock) {
			t.Errorf("expected wrapping of %v, got %v", errMock, err)
		}
	})

	t.Run("nil result", func(t *testing.T) {
		repo := &mockCorrespondentRepo{
			searchFn: func(ctx context.Context, query string, page, pageSize int) (*domain.PaginatedResult[domain.Correspondent], error) {
				return nil, nil
			},
		}

		svc := NewCorrespondentService(repo)
		result, err := svc.Search(ctx(), "nonexistent", 1, 10)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != nil {
			t.Errorf("expected nil result, got %+v", result)
		}
	})

	t.Run("empty results", func(t *testing.T) {
		expected := &domain.PaginatedResult[domain.Correspondent]{
			Total:   0,
			Results: []domain.Correspondent{},
		}

		repo := &mockCorrespondentRepo{
			searchFn: func(ctx context.Context, query string, page, pageSize int) (*domain.PaginatedResult[domain.Correspondent], error) {
				return expected, nil
			},
		}

		svc := NewCorrespondentService(repo)
		result, err := svc.Search(ctx(), "zzz", 1, 10)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Total != 0 {
			t.Errorf("expected total=0, got %d", result.Total)
		}
		if len(result.Results) != 0 {
			t.Errorf("expected 0 results, got %d", len(result.Results))
		}
	})

	t.Run("error message wrapping", func(t *testing.T) {
		repo := &mockCorrespondentRepo{
			searchFn: func(ctx context.Context, query string, page, pageSize int) (*domain.PaginatedResult[domain.Correspondent], error) {
				return nil, errMock
			},
		}

		svc := NewCorrespondentService(repo)
		_, err := svc.Search(ctx(), "x", 1, 10)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if err.Error() != "search correspondents: mock error" {
			t.Errorf("unexpected error message: %q", err.Error())
		}
	})
}

// ---------------------------------------------------------------------------
// TagService tests
// ---------------------------------------------------------------------------

func TestTagService_List(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		expected := &domain.PaginatedResult[domain.Tag]{
			Total: 3,
			Results: []domain.Tag{
				{ID: 1, Name: "Important"}, //nolint:exhaustruct
				{ID: 2, Name: "Urgent"},    //nolint:exhaustruct
				{ID: 3, Name: "Review"},    //nolint:exhaustruct
			},
		}

		var capturedQuery string
		var capturedPage, capturedPageSize int

		repo := &mockTagRepo{
			listFn: func(ctx context.Context, query string, page, pageSize int) (*domain.PaginatedResult[domain.Tag], error) {
				capturedQuery = query
				capturedPage = page
				capturedPageSize = pageSize
				return expected, nil
			},
		}

		svc := NewTagService(repo)
		result, err := svc.List(ctx(), "urgent", 2, 50)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify params forwarded correctly
		if capturedQuery != "urgent" {
			t.Errorf("expected query=%q, got %q", "urgent", capturedQuery)
		}
		if capturedPage != 2 {
			t.Errorf("expected page=2, got %d", capturedPage)
		}
		if capturedPageSize != 50 {
			t.Errorf("expected pageSize=50, got %d", capturedPageSize)
		}

		// Verify result
		if result.Total != expected.Total {
			t.Errorf("expected total=%d, got %d", expected.Total, result.Total)
		}
		if len(result.Results) != len(expected.Results) {
			t.Fatalf("expected %d results, got %d", len(expected.Results), len(result.Results))
		}
		for i, r := range result.Results {
			if r.ID != expected.Results[i].ID || r.Name != expected.Results[i].Name {
				t.Errorf("result[%d] = %+v, want %+v", i, r, expected.Results[i])
			}
		}
	})

	t.Run("error propagation", func(t *testing.T) {
		repo := &mockTagRepo{
			listFn: func(ctx context.Context, query string, page, pageSize int) (*domain.PaginatedResult[domain.Tag], error) {
				return nil, errMock
			},
		}

		svc := NewTagService(repo)
		_, err := svc.List(ctx(), "test", 1, 10)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, errMock) {
			t.Errorf("expected wrapping of %v, got %v", errMock, err)
		}
	})

	t.Run("nil result", func(t *testing.T) {
		repo := &mockTagRepo{
			listFn: func(ctx context.Context, query string, page, pageSize int) (*domain.PaginatedResult[domain.Tag], error) {
				return nil, nil
			},
		}

		svc := NewTagService(repo)
		result, err := svc.List(ctx(), "missing", 1, 10)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != nil {
			t.Errorf("expected nil result, got %+v", result)
		}
	})

	t.Run("empty results", func(t *testing.T) {
		expected := &domain.PaginatedResult[domain.Tag]{
			Total:   0,
			Results: []domain.Tag{},
		}

		repo := &mockTagRepo{
			listFn: func(ctx context.Context, query string, page, pageSize int) (*domain.PaginatedResult[domain.Tag], error) {
				return expected, nil
			},
		}

		svc := NewTagService(repo)
		result, err := svc.List(ctx(), "", 1, 10)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Total != 0 {
			t.Errorf("expected total=0, got %d", result.Total)
		}
		if len(result.Results) != 0 {
			t.Errorf("expected 0 results, got %d", len(result.Results))
		}
	})

	t.Run("error message wrapping", func(t *testing.T) {
		repo := &mockTagRepo{
			listFn: func(ctx context.Context, query string, page, pageSize int) (*domain.PaginatedResult[domain.Tag], error) {
				return nil, errMock
			},
		}

		svc := NewTagService(repo)
		_, err := svc.List(ctx(), "x", 1, 10)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if err.Error() != "list tags: mock error" {
			t.Errorf("unexpected error message: %q", err.Error())
		}
	})
}
