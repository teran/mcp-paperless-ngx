package handlers

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/teran/mcp-paperless-ngx/application"
	"github.com/teran/mcp-paperless-ngx/domain"
)

// ============================================================
// Mock repositories
// ============================================================

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

type mockDocTypeRepo struct {
	getByIDFn func(ctx context.Context, id int) (*domain.DocumentType, error)
}

func (m *mockDocTypeRepo) GetByID(ctx context.Context, id int) (*domain.DocumentType, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return &domain.DocumentType{ID: id, Name: "Document Type"}, nil //nolint:exhaustruct
}

type mockCorrespondentRepo struct {
	searchFn  func(ctx context.Context, query string, page, pageSize int) (*domain.PaginatedResult[domain.Correspondent], error)
	getByIDFn func(ctx context.Context, id int) (*domain.Correspondent, error)
}

func (m *mockCorrespondentRepo) Search(ctx context.Context, query string, page, pageSize int) (*domain.PaginatedResult[domain.Correspondent], error) {
	return m.searchFn(ctx, query, page, pageSize)
}

func (m *mockCorrespondentRepo) GetByID(ctx context.Context, id int) (*domain.Correspondent, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return &domain.Correspondent{ID: id, Name: fmt.Sprintf("Correspondent %d", id)}, nil //nolint:exhaustruct
}

type mockTagRepo struct {
	listFn func(ctx context.Context, query string, page, pageSize int) (*domain.PaginatedResult[domain.Tag], error)
}

func (m *mockTagRepo) List(ctx context.Context, query string, page, pageSize int) (*domain.PaginatedResult[domain.Tag], error) {
	return m.listFn(ctx, query, page, pageSize)
}

// ============================================================
// Test helpers
// ============================================================

func newTestCorrSvc() *application.CorrespondentService {
	return application.NewCorrespondentService(&mockCorrespondentRepo{}) //nolint:exhaustruct
}

func newTestDocTypeSvc(getByIDFn ...func(ctx context.Context, id int) (*domain.DocumentType, error)) *application.DocumentTypeService {
	var fn func(ctx context.Context, id int) (*domain.DocumentType, error)
	if len(getByIDFn) > 0 {
		fn = getByIDFn[0]
	}
	return application.NewDocumentTypeService(&mockDocTypeRepo{getByIDFn: fn}) //nolint:exhaustruct
}

func ctx() context.Context {
	return context.Background()
}

func intPtr(v int) *int {
	return &v
}

// ============================================================
// Test data
// ============================================================

var testDocs = []domain.Document{ //nolint:gochecknoglobals
	{ //nolint:exhaustruct
		ID:                  1,
		Title:               "Invoice March 2024", //nolint:goconst
		Content:             "This is the OCR text of the invoice.",
		Correspondent:       intPtr(5),
		DocumentType:        intPtr(2),
		Tags:                []int{7, 10},
		Created:             "2024-03-15", //nolint:goconst
		CreatedDate:         "2024-03-15",
		Modified:            "2024-03-16T10:00:00Z",
		Added:               "2024-03-16T12:00:00Z",
		ArchiveSerialNumber: intPtr(42),
		OriginalFileName:    "invoice.pdf",
		ArchivedFileName:    strPtr("invoice_archive.pdf"),
		MimeType:            "application/pdf", //nolint:goconst
		PageCount:           intPtr(3),
		SearchHit:           &domain.SearchHit{Score: 12.345, Highlights: "OCR text of the invoice", Rank: 1},
	},
	{ //nolint:exhaustruct
		ID:               2,
		Title:            "Contract Signed",
		Content:          "Contract terms and conditions.",
		Correspondent:    intPtr(5),
		Tags:             []int{8},
		Created:          "2024-06-01",
		CreatedDate:      "2024-06-01",
		Modified:         "2024-06-02T08:30:00Z",
		Added:            "2024-06-02T09:00:00Z",
		OriginalFileName: "contract.pdf",
		MimeType:         "application/pdf",
		PageCount:        intPtr(5),
	},
	{ //nolint:exhaustruct
		ID:               3,
		Title:            "Photo",
		Content:          "A nice landscape photo.",
		Tags:             []int{9},
		Created:          "2024-09-10",
		CreatedDate:      "2024-09-10",
		Modified:         "2024-09-11T00:00:00Z",
		Added:            "2024-09-11T00:00:00Z",
		OriginalFileName: "photo.png",
		MimeType:         "image/png",
		PageCount:        intPtr(1),
		SearchHit:        &domain.SearchHit{Score: 5.0, Highlights: "landscape", Rank: 2},
	},
}

var testCorrespondents = []domain.Correspondent{ //nolint:gochecknoglobals
	{ID: 1, Name: "Acme Corp", Slug: "acme-corp", DocumentCount: 10, MatchingAlgorithm: 1, IsInsensitive: true},          //nolint:exhaustruct
	{ID: 2, Name: "Bob's Supplies", Slug: "bobs-supplies", DocumentCount: 5, MatchingAlgorithm: 1, IsInsensitive: false}, //nolint:exhaustruct
	{ID: 3, Name: "Charlie & Co", Slug: "charlie-co", DocumentCount: 0, MatchingAlgorithm: 0, IsInsensitive: false},      //nolint:exhaustruct
}

var testTags = []domain.Tag{ //nolint:gochecknoglobals
	{ID: 1, Name: "Important", Slug: "important", Color: "red", DocumentCount: 12, IsInboxTag: false},         //nolint:exhaustruct
	{ID: 2, Name: "Inbox", Slug: "inbox", Color: "blue", DocumentCount: 3, IsInboxTag: true},                  //nolint:exhaustruct,goconst
	{ID: 3, Name: "Review Later", Slug: "review-later", Color: "yellow", DocumentCount: 7, IsInboxTag: false}, //nolint:exhaustruct
}

var errMock = errors.New("mock error")

func strPtr(v string) *string {
	return &v
}

// ============================================================
// normalizePagination tests
// ============================================================

func TestNormalizePagination(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		page, pageSize int
		wantPage       int
		wantPageSize   int
	}{
		{name: "defaults when page=0, pageSize=0", page: 0, pageSize: 0, wantPage: 1, wantPageSize: 25},
		{name: "defaults when page=-1, pageSize=-1", page: -1, pageSize: -1, wantPage: 1, wantPageSize: 25},
		{name: "defaults when page=0, pageSize=101", page: 0, pageSize: 101, wantPage: 1, wantPageSize: 25},
		{name: "keeps values when within range", page: 2, pageSize: 50, wantPage: 2, wantPageSize: 50},
		{name: "keeps pageSize=100 as valid", page: 1, pageSize: 100, wantPage: 1, wantPageSize: 100},
		{name: "caps pageSize at 100", page: 1, pageSize: 200, wantPage: 1, wantPageSize: 25},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			page, pageSize := normalizePagination(tt.page, tt.pageSize)
			if page != tt.wantPage {
				t.Errorf("expected page=%d, got %d", tt.wantPage, page)
			}
			if pageSize != tt.wantPageSize {
				t.Errorf("expected pageSize=%d, got %d", tt.wantPageSize, pageSize)
			}
		})
	}
}

// ============================================================
// toDocumentSummaries tests
// ============================================================

func TestToDocumentSummaries(t *testing.T) { //nolint:gocognit
	t.Parallel()

	t.Run("converts documents to summaries", func(t *testing.T) {
		t.Parallel()

		docs := []domain.Document{
			{ //nolint:exhaustruct
				ID: 1, Title: "Test", Correspondent: intPtr(5), Tags: []int{7},
				Created: "2024-01-01", MimeType: "application/pdf", //nolint:goconst
				ArchiveSerialNumber: intPtr(42), PageCount: intPtr(3),
			},
			{ //nolint:exhaustruct
				ID: 2, Title: "No extras", Created: "2024-02-02", MimeType: "image/png",
			},
		}

		summaries := toDocumentSummaries(docs, nil, nil)

		if len(summaries) != 2 {
			t.Fatalf("expected 2 summaries, got %d", len(summaries))
		}

		// First doc — full fields.
		s0 := summaries[0]
		if s0.ID != 1 || s0.Title != "Test" || s0.MimeType != "application/pdf" {
			t.Errorf("unexpected summary[0]: %+v", s0)
		}
		if s0.Correspondent == nil || *s0.Correspondent != 5 {
			t.Errorf("expected correspondent=5, got %v", s0.Correspondent)
		}
		if len(s0.Tags) != 1 || s0.Tags[0] != 7 {
			t.Errorf("expected tags=[7], got %v", s0.Tags)
		}
		if s0.ArchiveSerial == nil || *s0.ArchiveSerial != 42 {
			t.Errorf("expected archive_serial=42, got %v", s0.ArchiveSerial)
		}
		if s0.PageCount == nil || *s0.PageCount != 3 {
			t.Errorf("expected page_count=3, got %v", s0.PageCount)
		}

		// Second doc — nil pointer fields.
		s1 := summaries[1]
		if s1.Correspondent != nil {
			t.Errorf("expected nil correspondent, got %v", s1.Correspondent)
		}
		if s1.ArchiveSerial != nil {
			t.Errorf("expected nil archive_serial, got %v", s1.ArchiveSerial)
		}
		if s1.PageCount != nil {
			t.Errorf("expected nil page_count, got %v", s1.PageCount)
		}
	})

	t.Run("nil input returns empty slice", func(t *testing.T) {
		t.Parallel()

		summaries := toDocumentSummaries(nil, nil, nil)
		if summaries == nil {
			t.Errorf("expected non-nil empty slice, got nil")
		}
		if len(summaries) != 0 {
			t.Errorf("expected 0 summaries, got %d", len(summaries))
		}
	})

	t.Run("empty input returns empty slice", func(t *testing.T) {
		t.Parallel()

		summaries := toDocumentSummaries([]domain.Document{}, nil, nil)
		if summaries == nil {
			t.Errorf("expected non-nil empty slice, got nil")
		}
		if len(summaries) != 0 {
			t.Errorf("expected 0 summaries, got %d", len(summaries))
		}
	})
}

// ============================================================
// search_documents
// ============================================================

func TestNewSearchDocumentsHandler(t *testing.T) { //nolint:gocognit,gocyclo,maintidx
	t.Parallel()

	t.Run("success with no filters", func(t *testing.T) {
		t.Parallel()

		svc := application.NewDocumentService(&mockDocRepo{ //nolint:exhaustruct
			searchFn: func(ctx context.Context, params domain.SearchDocumentsParams) (*domain.PaginatedResult[domain.Document], error) {
				if params.Query != "" {
					t.Errorf("expected empty query, got %q", params.Query)
				}
				return &domain.PaginatedResult[domain.Document]{
					Total:   3,
					Results: testDocs,
				}, nil
			},
		})

		handler := NewSearchDocumentsHandler(svc, newTestCorrSvc(), newTestDocTypeSvc())
		_, output, err := handler(ctx(), nil, SearchDocumentsInput{}) //nolint:exhaustruct
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if output.Total != 3 {
			t.Errorf("expected total=3, got %d", output.Total)
		}
		if output.Page != 1 {
			t.Errorf("expected page=1, got %d", output.Page)
		}
		if len(output.Results) != 3 {
			t.Errorf("expected 3 results, got %d", len(output.Results))
		}
	})

	t.Run("success with all filters", func(t *testing.T) {
		t.Parallel()

		svc := application.NewDocumentService(&mockDocRepo{ //nolint:exhaustruct
			searchFn: func(ctx context.Context, params domain.SearchDocumentsParams) (*domain.PaginatedResult[domain.Document], error) {
				if params.Query != "invoice" { //nolint:goconst
					t.Errorf("expected query=%q, got %q", "invoice", params.Query)
				}
				if params.CorrespondentID != 5 {
					t.Errorf("expected correspondent_id=5, got %d", params.CorrespondentID)
				}
				if len(params.TagIDs) != 1 || params.TagIDs[0] != 7 {
					t.Errorf("expected tag_ids=[7], got %v", params.TagIDs)
				}
				if params.CreatedAfter != "2024-01-01" {
					t.Errorf("expected created_after=%q, got %q", "2024-01-01", params.CreatedAfter)
				}
				if params.CreatedBefore != "2024-12-31" {
					t.Errorf("expected created_before=%q, got %q", "2024-12-31", params.CreatedBefore)
				}
				if params.Page != 2 {
					t.Errorf("expected page=2, got %d", params.Page)
				}
				if params.PageSize != 10 {
					t.Errorf("expected page_size=10, got %d", params.PageSize)
				}
				return &domain.PaginatedResult[domain.Document]{
					Total:   1,
					Results: testDocs[:1],
				}, nil
			},
		})

		handler := NewSearchDocumentsHandler(svc, newTestCorrSvc(), newTestDocTypeSvc())
		_, output, err := handler(ctx(), nil, SearchDocumentsInput{
			Query:           "invoice",
			CorrespondentID: 5,
			TagIDs:          []int{7},
			CreatedAfter:    "2024-01-01",
			CreatedBefore:   "2024-12-31",
			Page:            2,
			PageSize:        10,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if output.Total != 1 {
			t.Errorf("expected total=1, got %d", output.Total)
		}
		if output.Page != 2 {
			t.Errorf("expected page=2, got %d", output.Page)
		}
		if len(output.Results) != 1 {
			t.Errorf("expected 1 result, got %d", len(output.Results))
		}
		if output.Results[0].ID != 1 {
			t.Errorf("expected result ID=1, got %d", output.Results[0].ID)
		}
	})

	t.Run("pagination defaults applied", func(t *testing.T) {
		t.Parallel()

		svc := application.NewDocumentService(&mockDocRepo{ //nolint:exhaustruct
			searchFn: func(ctx context.Context, params domain.SearchDocumentsParams) (*domain.PaginatedResult[domain.Document], error) {
				if params.Page != 1 {
					t.Errorf("expected default page=1, got %d", params.Page)
				}
				if params.PageSize != 25 {
					t.Errorf("expected default page_size=25, got %d", params.PageSize)
				}
				return &domain.PaginatedResult[domain.Document]{Total: 0, Results: []domain.Document{}}, nil
			},
		})

		handler := NewSearchDocumentsHandler(svc, newTestCorrSvc(), newTestDocTypeSvc())
		_, output, err := handler(ctx(), nil, SearchDocumentsInput{Page: 0, PageSize: 0}) //nolint:exhaustruct
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if output.Page != 1 {
			t.Errorf("expected output page=1, got %d", output.Page)
		}
	})

	t.Run("empty results", func(t *testing.T) {
		t.Parallel()

		svc := application.NewDocumentService(&mockDocRepo{ //nolint:exhaustruct
			searchFn: func(ctx context.Context, params domain.SearchDocumentsParams) (*domain.PaginatedResult[domain.Document], error) {
				return &domain.PaginatedResult[domain.Document]{Total: 0, Results: []domain.Document{}}, nil
			},
		})

		handler := NewSearchDocumentsHandler(svc, newTestCorrSvc(), newTestDocTypeSvc())
		_, output, err := handler(ctx(), nil, SearchDocumentsInput{Query: "nonexistent"}) //nolint:exhaustruct,goconst
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if output.Total != 0 {
			t.Errorf("expected total=0, got %d", output.Total)
		}
		if len(output.Results) != 0 {
			t.Errorf("expected 0 results, got %d", len(output.Results))
		}
	})

	t.Run("error propagation", func(t *testing.T) {
		t.Parallel()

		svc := application.NewDocumentService(&mockDocRepo{ //nolint:exhaustruct
			searchFn: func(ctx context.Context, params domain.SearchDocumentsParams) (*domain.PaginatedResult[domain.Document], error) {
				return nil, errMock
			},
		})

		handler := NewSearchDocumentsHandler(svc, newTestCorrSvc(), newTestDocTypeSvc())
		_, _, err := handler(ctx(), nil, SearchDocumentsInput{Query: "fail"}) //nolint:exhaustruct,goconst
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, ErrSearchFailed) {
			t.Errorf("expected %v, got %v", ErrSearchFailed, err)
		}
	})

	t.Run("result fields mapped correctly", func(t *testing.T) {
		t.Parallel()

		svc := application.NewDocumentService(&mockDocRepo{ //nolint:exhaustruct
			searchFn: func(ctx context.Context, params domain.SearchDocumentsParams) (*domain.PaginatedResult[domain.Document], error) {
				return &domain.PaginatedResult[domain.Document]{
					Total:   2,
					Results: testDocs[:2],
				}, nil
			},
		})

		handler := NewSearchDocumentsHandler(svc, newTestCorrSvc(), newTestDocTypeSvc())
		_, output, err := handler(ctx(), nil, SearchDocumentsInput{}) //nolint:exhaustruct
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// First result — full data.
		r0 := output.Results[0]
		if r0.ID != 1 {
			t.Errorf("expected ID=1, got %d", r0.ID)
		}
		if r0.Title != "Invoice March 2024" {
			t.Errorf("expected Title=%q, got %q", "Invoice March 2024", r0.Title)
		}
		if r0.Correspondent == nil || *r0.Correspondent != 5 {
			t.Errorf("expected correspondent=5, got %v", r0.Correspondent)
		}
		if len(r0.Tags) != 2 || r0.Tags[0] != 7 || r0.Tags[1] != 10 {
			t.Errorf("expected tags=[7,10], got %v", r0.Tags)
		}
		if r0.Created != "2024-03-15" {
			t.Errorf("expected Created=%q, got %q", "2024-03-15", r0.Created)
		}
		if r0.MimeType != "application/pdf" {
			t.Errorf("expected MimeType=%q, got %q", "application/pdf", r0.MimeType)
		}
		if r0.ArchiveSerial == nil || *r0.ArchiveSerial != 42 {
			t.Errorf("expected archive_serial=42, got %v", r0.ArchiveSerial)
		}
		if r0.PageCount == nil || *r0.PageCount != 3 {
			t.Errorf("expected page_count=3, got %v", r0.PageCount)
		}

		// Second result — without optional fields.
		r1 := output.Results[1]
		if r1.ID != 2 {
			t.Errorf("expected ID=2, got %d", r1.ID)
		}
		if r1.Correspondent == nil || *r1.Correspondent != 5 {
			t.Errorf("expected correspondent=5, got %v", r1.Correspondent)
		}
		if r1.ArchiveSerial != nil {
			t.Errorf("expected nil archive_serial, got %v", r1.ArchiveSerial)
		}
	})
}

// ============================================================
// get_document_content
// ============================================================

func TestNewGetDocumentContentHandler(t *testing.T) { //nolint:gocognit,gocyclo
	t.Parallel()

	t.Run("existing document", func(t *testing.T) {
		t.Parallel()

		svc := application.NewDocumentService(&mockDocRepo{ //nolint:exhaustruct
			getByIDFn: func(ctx context.Context, id int) (*domain.Document, error) {
				if id != 1 {
					t.Errorf("expected id=1, got %d", id)
				}
				return &testDocs[0], nil
			},
		})

		handler := NewGetDocumentContentHandler(svc, newTestCorrSvc(), newTestDocTypeSvc())
		_, output, err := handler(ctx(), nil, GetDocumentContentInput{DocumentID: 1})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if output.ID != 1 {
			t.Errorf("expected ID=1, got %d", output.ID)
		}
		if output.Title != "Invoice March 2024" {
			t.Errorf("expected Title=%q, got %q", "Invoice March 2024", output.Title)
		}
		if output.Content != "This is the OCR text of the invoice." {
			t.Errorf("unexpected content: %q", output.Content)
		}
		if output.Correspondent == nil || *output.Correspondent != 5 {
			t.Errorf("expected correspondent=5, got %v", output.Correspondent)
		}
		if output.DocumentType == nil || *output.DocumentType != 2 {
			t.Errorf("expected document_type=2, got %v", output.DocumentType)
		}
		if len(output.Tags) != 2 {
			t.Errorf("expected 2 tags, got %d", len(output.Tags))
		}
		if output.Created != "2024-03-15" {
			t.Errorf("expected Created=%q, got %q", "2024-03-15", output.Created)
		}
		if output.Modified != "2024-03-16T10:00:00Z" {
			t.Errorf("expected Modified=%q, got %q", "2024-03-16T10:00:00Z", output.Modified)
		}
		if output.Added != "2024-03-16T12:00:00Z" {
			t.Errorf("expected Added=%q, got %q", "2024-03-16T12:00:00Z", output.Added)
		}
		if output.ArchiveSerialNumber == nil || *output.ArchiveSerialNumber != 42 {
			t.Errorf("expected ArchiveSerialNumber=42, got %v", output.ArchiveSerialNumber)
		}
		if output.OriginalFileName != "invoice.pdf" {
			t.Errorf("expected OriginalFileName=%q, got %q", "invoice.pdf", output.OriginalFileName)
		}
		if output.MimeType != "application/pdf" {
			t.Errorf("expected MimeType=%q, got %q", "application/pdf", output.MimeType)
		}
		if output.PageCount == nil || *output.PageCount != 3 {
			t.Errorf("expected PageCount=3, got %v", output.PageCount)
		}
	})

	t.Run("non-existing document", func(t *testing.T) {
		t.Parallel()

		svc := application.NewDocumentService(&mockDocRepo{ //nolint:exhaustruct
			getByIDFn: func(ctx context.Context, id int) (*domain.Document, error) {
				return nil, errors.New("document not found") //nolint:err113
			},
		})

		handler := NewGetDocumentContentHandler(svc, newTestCorrSvc(), newTestDocTypeSvc())
		_, _, err := handler(ctx(), nil, GetDocumentContentInput{DocumentID: 999})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, ErrDocumentNotFound) {
			t.Errorf("expected %v, got %v", ErrDocumentNotFound, err)
		}
	})

	t.Run("error propagation", func(t *testing.T) {
		t.Parallel()

		svc := application.NewDocumentService(&mockDocRepo{ //nolint:exhaustruct
			getByIDFn: func(ctx context.Context, id int) (*domain.Document, error) {
				return nil, errMock
			},
		})

		handler := NewGetDocumentContentHandler(svc, newTestCorrSvc(), newTestDocTypeSvc())
		_, _, err := handler(ctx(), nil, GetDocumentContentInput{DocumentID: 1})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, ErrDocumentNotFound) {
			t.Errorf("expected %v, got %v", ErrDocumentNotFound, err)
		}
	})

	t.Run("document with minimal fields", func(t *testing.T) {
		t.Parallel()

		minimalDoc := &domain.Document{ //nolint:exhaustruct
			ID: 99, Title: "Minimal", Content: "Minimal content",
			Created: "2024-01-01", Modified: "2024-01-01", Added: "2024-01-01",
			OriginalFileName: "minimal.txt", MimeType: "text/plain",
		}

		svc := application.NewDocumentService(&mockDocRepo{ //nolint:exhaustruct
			getByIDFn: func(ctx context.Context, id int) (*domain.Document, error) {
				return minimalDoc, nil
			},
		})

		handler := NewGetDocumentContentHandler(svc, newTestCorrSvc(), newTestDocTypeSvc())
		_, output, err := handler(ctx(), nil, GetDocumentContentInput{DocumentID: 99})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if output.ID != 99 || output.Title != "Minimal" {
			t.Errorf("unexpected output: %+v", output)
		}
		// All pointer fields should be nil.
		if output.Correspondent != nil {
			t.Errorf("expected nil correspondent")
		}
		if output.DocumentType != nil {
			t.Errorf("expected nil document_type")
		}
		if output.ArchiveSerialNumber != nil {
			t.Errorf("expected nil archive_serial_number")
		}
		if output.PageCount != nil {
			t.Errorf("expected nil page_count")
		}
	})

	t.Run("document type name resolution error is silently swallowed", func(t *testing.T) {
		t.Parallel()

		svc := application.NewDocumentService(&mockDocRepo{ //nolint:exhaustruct
			getByIDFn: func(ctx context.Context, id int) (*domain.Document, error) {
				return &testDocs[0], nil
			},
		})

		docTypeSvc := newTestDocTypeSvc(func(ctx context.Context, id int) (*domain.DocumentType, error) {
			return nil, errMock
		})
		handler := NewGetDocumentContentHandler(svc, newTestCorrSvc(), docTypeSvc)
		_, output, err := handler(ctx(), nil, GetDocumentContentInput{DocumentID: 1})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if output.DocumentType != nil && output.DocumentTypeName != "" {
			t.Errorf("expected empty DocumentTypeName when GetByID fails, got %q", output.DocumentTypeName)
		}
	})
}

// ============================================================
// search_correspondents
// ============================================================

func TestNewSearchCorrespondentsHandler(t *testing.T) { //nolint:gocognit
	t.Parallel()

	t.Run("success with query", func(t *testing.T) {
		t.Parallel()

		svc := application.NewCorrespondentService(&mockCorrespondentRepo{
			searchFn: func(ctx context.Context, query string, page, pageSize int) (*domain.PaginatedResult[domain.Correspondent], error) {
				if query != "acme" {
					t.Errorf("expected query=%q, got %q", "acme", query)
				}
				return &domain.PaginatedResult[domain.Correspondent]{
					Total:   1,
					Results: testCorrespondents[:1],
				}, nil
			},
			getByIDFn: nil,
		})

		handler := NewSearchCorrespondentsHandler(svc)
		_, output, err := handler(ctx(), nil, SearchCorrespondentsInput{Query: "acme"}) //nolint:exhaustruct
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if output.Total != 1 {
			t.Errorf("expected total=1, got %d", output.Total)
		}
		if output.Page != 1 {
			t.Errorf("expected page=1, got %d", output.Page)
		}
		if len(output.Results) != 1 {
			t.Errorf("expected 1 result, got %d", len(output.Results))
		}
		if output.Results[0].ID != 1 {
			t.Errorf("expected ID=1, got %d", output.Results[0].ID)
		}
		if output.Results[0].Name != "Acme Corp" {
			t.Errorf("expected Name=%q, got %q", "Acme Corp", output.Results[0].Name)
		}
		if output.Results[0].Slug != "acme-corp" {
			t.Errorf("expected Slug=%q, got %q", "acme-corp", output.Results[0].Slug)
		}
		if output.Results[0].DocumentCount != 10 {
			t.Errorf("expected DocumentCount=10, got %d", output.Results[0].DocumentCount)
		}
	})

	t.Run("success without query returns all", func(t *testing.T) {
		t.Parallel()

		svc := application.NewCorrespondentService(&mockCorrespondentRepo{
			searchFn: func(ctx context.Context, query string, page, pageSize int) (*domain.PaginatedResult[domain.Correspondent], error) {
				return &domain.PaginatedResult[domain.Correspondent]{
					Total:   3,
					Results: testCorrespondents,
				}, nil
			},
			getByIDFn: nil,
		})

		handler := NewSearchCorrespondentsHandler(svc)
		_, output, err := handler(ctx(), nil, SearchCorrespondentsInput{Query: ""}) //nolint:exhaustruct
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if output.Total != 3 {
			t.Errorf("expected total=3, got %d", output.Total)
		}
		if len(output.Results) != 3 {
			t.Errorf("expected 3 results, got %d", len(output.Results))
		}
	})

	t.Run("pagination", func(t *testing.T) {
		t.Parallel()

		svc := application.NewCorrespondentService(&mockCorrespondentRepo{
			searchFn: func(ctx context.Context, query string, page, pageSize int) (*domain.PaginatedResult[domain.Correspondent], error) {
				if page != 2 {
					t.Errorf("expected page=2, got %d", page)
				}
				if pageSize != 10 {
					t.Errorf("expected page_size=10, got %d", pageSize)
				}
				return &domain.PaginatedResult[domain.Correspondent]{
					Total:   3,
					Results: testCorrespondents,
				}, nil
			},
			getByIDFn: nil,
		})

		handler := NewSearchCorrespondentsHandler(svc)
		_, output, err := handler(ctx(), nil, SearchCorrespondentsInput{Query: "", Page: 2, PageSize: 10})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if output.Page != 2 {
			t.Errorf("expected page=2, got %d", output.Page)
		}
	})

	t.Run("pagination defaults", func(t *testing.T) {
		t.Parallel()

		svc := application.NewCorrespondentService(&mockCorrespondentRepo{
			searchFn: func(ctx context.Context, query string, page, pageSize int) (*domain.PaginatedResult[domain.Correspondent], error) {
				if page != 1 {
					t.Errorf("expected default page=1, got %d", page)
				}
				if pageSize != 25 {
					t.Errorf("expected default page_size=25, got %d", pageSize)
				}
				return &domain.PaginatedResult[domain.Correspondent]{Total: 0, Results: []domain.Correspondent{}}, nil
			},
			getByIDFn: nil,
		})

		handler := NewSearchCorrespondentsHandler(svc)
		_, _, err := handler(ctx(), nil, SearchCorrespondentsInput{Query: ""}) //nolint:exhaustruct
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("empty results", func(t *testing.T) {
		t.Parallel()

		svc := application.NewCorrespondentService(&mockCorrespondentRepo{
			searchFn: func(ctx context.Context, query string, page, pageSize int) (*domain.PaginatedResult[domain.Correspondent], error) {
				return &domain.PaginatedResult[domain.Correspondent]{Total: 0, Results: []domain.Correspondent{}}, nil
			},
			getByIDFn: nil,
		})

		handler := NewSearchCorrespondentsHandler(svc)
		_, output, err := handler(ctx(), nil, SearchCorrespondentsInput{Query: "zzz"}) //nolint:exhaustruct
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if output.Total != 0 || len(output.Results) != 0 {
			t.Errorf("expected empty result, got %+v", output)
		}
	})

	t.Run("error propagation", func(t *testing.T) {
		t.Parallel()

		svc := application.NewCorrespondentService(&mockCorrespondentRepo{
			searchFn: func(ctx context.Context, query string, page, pageSize int) (*domain.PaginatedResult[domain.Correspondent], error) {
				return nil, errMock
			},
		})

		handler := NewSearchCorrespondentsHandler(svc)
		_, _, err := handler(ctx(), nil, SearchCorrespondentsInput{Query: "fail"}) //nolint:exhaustruct
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, ErrCorrespondentNotFound) {
			t.Errorf("expected %v, got %v", ErrCorrespondentNotFound, err)
		}
	})
}

// ============================================================
// get_documents_by_correspondent
// ============================================================

func TestNewGetDocumentsByCorrespondentHandler(t *testing.T) { //nolint:gocognit
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		svc := application.NewDocumentService(&mockDocRepo{ //nolint:exhaustruct
			searchFn: func(ctx context.Context, params domain.SearchDocumentsParams) (*domain.PaginatedResult[domain.Document], error) {
				if params.CorrespondentID != 5 {
					t.Errorf("expected correspondent_id=5, got %d", params.CorrespondentID)
				}
				return &domain.PaginatedResult[domain.Document]{
					Total:   2,
					Results: testDocs[:2],
				}, nil
			},
		})

		handler := NewGetDocumentsByCorrespondentHandler(svc, newTestCorrSvc(), newTestDocTypeSvc())
		_, output, err := handler(ctx(), nil, GetDocumentsByCorrespondentInput{CorrespondentID: 5}) //nolint:exhaustruct
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if output.Total != 2 {
			t.Errorf("expected total=2, got %d", output.Total)
		}
		if output.Page != 1 {
			t.Errorf("expected page=1, got %d", output.Page)
		}
		if len(output.Results) != 2 {
			t.Errorf("expected 2 results, got %d", len(output.Results))
		}
	})

	//nolint:dupl
	t.Run("pagination", func(t *testing.T) {
		t.Parallel()

		svc := application.NewDocumentService(&mockDocRepo{ //nolint:exhaustruct
			searchFn: func(ctx context.Context, params domain.SearchDocumentsParams) (*domain.PaginatedResult[domain.Document], error) {
				if params.Page != 3 {
					t.Errorf("expected page=3, got %d", params.Page)
				}
				if params.PageSize != 50 {
					t.Errorf("expected page_size=50, got %d", params.PageSize)
				}
				return &domain.PaginatedResult[domain.Document]{Total: 0, Results: []domain.Document{}}, nil
			},
		})

		handler := NewGetDocumentsByCorrespondentHandler(svc, newTestCorrSvc(), newTestDocTypeSvc())
		_, output, err := handler(ctx(), nil, GetDocumentsByCorrespondentInput{
			CorrespondentID: 5,
			Page:            3,
			PageSize:        50,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if output.Page != 3 {
			t.Errorf("expected page=3, got %d", output.Page)
		}
	})

	t.Run("no results", func(t *testing.T) {
		t.Parallel()

		svc := application.NewDocumentService(&mockDocRepo{ //nolint:exhaustruct
			searchFn: func(ctx context.Context, params domain.SearchDocumentsParams) (*domain.PaginatedResult[domain.Document], error) {
				return &domain.PaginatedResult[domain.Document]{Total: 0, Results: []domain.Document{}}, nil
			},
		})

		handler := NewGetDocumentsByCorrespondentHandler(svc, newTestCorrSvc(), newTestDocTypeSvc())
		_, output, err := handler(ctx(), nil, GetDocumentsByCorrespondentInput{CorrespondentID: 999}) //nolint:exhaustruct
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if output.Total != 0 {
			t.Errorf("expected total=0, got %d", output.Total)
		}
	})

	t.Run("error propagation", func(t *testing.T) {
		t.Parallel()

		svc := application.NewDocumentService(&mockDocRepo{ //nolint:exhaustruct
			searchFn: func(ctx context.Context, params domain.SearchDocumentsParams) (*domain.PaginatedResult[domain.Document], error) {
				return nil, errMock
			},
		})

		handler := NewGetDocumentsByCorrespondentHandler(svc, newTestCorrSvc(), newTestDocTypeSvc())
		_, _, err := handler(ctx(), nil, GetDocumentsByCorrespondentInput{CorrespondentID: 1}) //nolint:exhaustruct
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, ErrSearchFailed) {
			t.Errorf("expected %v, got %v", ErrSearchFailed, err)
		}
	})

	t.Run("document type name resolution error is silently swallowed", func(t *testing.T) {
		t.Parallel()

		svc := application.NewDocumentService(&mockDocRepo{ //nolint:exhaustruct
			searchFn: func(ctx context.Context, params domain.SearchDocumentsParams) (*domain.PaginatedResult[domain.Document], error) {
				return &domain.PaginatedResult[domain.Document]{
					Total:   1,
					Results: testDocs[:1],
				}, nil
			},
		})

		docTypeSvc := newTestDocTypeSvc(func(ctx context.Context, id int) (*domain.DocumentType, error) {
			return nil, errMock
		})
		handler := NewGetDocumentsByCorrespondentHandler(svc, newTestCorrSvc(), docTypeSvc)
		_, output, err := handler(ctx(), nil, GetDocumentsByCorrespondentInput{CorrespondentID: 5}) //nolint:exhaustruct
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		for _, r := range output.Results {
			if r.DocumentType != nil && r.DocumentTypeName != "" {
				t.Errorf("expected empty DocumentTypeName when GetByID fails, got %q for doc %d", r.DocumentTypeName, r.ID)
			}
		}
	})
}

// ============================================================
// list_tags
// ============================================================

func TestNewListTagsHandler(t *testing.T) { //nolint:gocognit
	t.Parallel()

	t.Run("success without query", func(t *testing.T) {
		t.Parallel()

		svc := application.NewTagService(&mockTagRepo{
			listFn: func(ctx context.Context, query string, page, pageSize int) (*domain.PaginatedResult[domain.Tag], error) {
				return &domain.PaginatedResult[domain.Tag]{
					Total:   3,
					Results: testTags,
				}, nil
			},
		})

		handler := NewListTagsHandler(svc)
		_, output, err := handler(ctx(), nil, ListTagsInput{}) //nolint:exhaustruct
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if output.Total != 3 {
			t.Errorf("expected total=3, got %d", output.Total)
		}
		if len(output.Results) != 3 {
			t.Errorf("expected 3 results, got %d", len(output.Results))
		}
	})

	t.Run("success with query filter", func(t *testing.T) {
		t.Parallel()

		svc := application.NewTagService(&mockTagRepo{
			listFn: func(ctx context.Context, query string, page, pageSize int) (*domain.PaginatedResult[domain.Tag], error) {
				if query != "Inbox" {
					t.Errorf("expected query=%q, got %q", "Inbox", query)
				}
				return &domain.PaginatedResult[domain.Tag]{
					Total:   1,
					Results: testTags[1:2],
				}, nil
			},
		})

		handler := NewListTagsHandler(svc)
		_, output, err := handler(ctx(), nil, ListTagsInput{Query: "Inbox"}) //nolint:exhaustruct
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if output.Total != 1 {
			t.Errorf("expected total=1, got %d", output.Total)
		}
		if len(output.Results) != 1 {
			t.Errorf("expected 1 result, got %d", len(output.Results))
		}
		if output.Results[0].ID != 2 {
			t.Errorf("expected ID=2, got %d", output.Results[0].ID)
		}
		if output.Results[0].Name != "Inbox" {
			t.Errorf("expected Name=%q, got %q", "Inbox", output.Results[0].Name)
		}
		if output.Results[0].Color != "blue" {
			t.Errorf("expected Color=%q, got %q", "blue", output.Results[0].Color)
		}
		if !output.Results[0].IsInboxTag {
			t.Errorf("expected IsInboxTag=true")
		}
		if output.Results[0].DocumentCount != 3 {
			t.Errorf("expected DocumentCount=3, got %d", output.Results[0].DocumentCount)
		}
	})

	t.Run("pagination defaults", func(t *testing.T) {
		t.Parallel()

		svc := application.NewTagService(&mockTagRepo{
			listFn: func(ctx context.Context, query string, page, pageSize int) (*domain.PaginatedResult[domain.Tag], error) {
				if page != 1 {
					t.Errorf("expected page=1, got %d", page)
				}
				if pageSize != 25 {
					t.Errorf("expected page_size=25, got %d", pageSize)
				}
				return &domain.PaginatedResult[domain.Tag]{Total: 0, Results: []domain.Tag{}}, nil
			},
		})

		handler := NewListTagsHandler(svc)
		_, _, err := handler(ctx(), nil, ListTagsInput{}) //nolint:exhaustruct
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("pagination passed through", func(t *testing.T) {
		t.Parallel()

		svc := application.NewTagService(&mockTagRepo{
			listFn: func(ctx context.Context, query string, page, pageSize int) (*domain.PaginatedResult[domain.Tag], error) {
				if page != 2 {
					t.Errorf("expected page=2, got %d", page)
				}
				if pageSize != 50 {
					t.Errorf("expected page_size=50, got %d", pageSize)
				}
				return &domain.PaginatedResult[domain.Tag]{
					Total:   3,
					Results: testTags,
				}, nil
			},
		})

		handler := NewListTagsHandler(svc)
		_, output, err := handler(ctx(), nil, ListTagsInput{Page: 2, PageSize: 50}) //nolint:exhaustruct
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if output.Page != 2 {
			t.Errorf("expected page=2, got %d", output.Page)
		}
	})

	t.Run("empty results", func(t *testing.T) {
		t.Parallel()

		svc := application.NewTagService(&mockTagRepo{
			listFn: func(ctx context.Context, query string, page, pageSize int) (*domain.PaginatedResult[domain.Tag], error) {
				return &domain.PaginatedResult[domain.Tag]{Total: 0, Results: []domain.Tag{}}, nil
			},
		})

		handler := NewListTagsHandler(svc)
		_, output, err := handler(ctx(), nil, ListTagsInput{Query: "nonexistent"}) //nolint:exhaustruct
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if output.Total != 0 || len(output.Results) != 0 {
			t.Errorf("expected empty result, got %+v", output)
		}
	})

	t.Run("error propagation", func(t *testing.T) {
		t.Parallel()

		svc := application.NewTagService(&mockTagRepo{
			listFn: func(ctx context.Context, query string, page, pageSize int) (*domain.PaginatedResult[domain.Tag], error) {
				return nil, errMock
			},
		})

		handler := NewListTagsHandler(svc)
		_, _, err := handler(ctx(), nil, ListTagsInput{Query: "fail"}) //nolint:exhaustruct
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, ErrListTagsFailed) {
			t.Errorf("expected %v, got %v", ErrListTagsFailed, err)
		}
	})
}

// ============================================================
// fulltext_search
// ============================================================
// get_documents_by_tag
// ============================================================

func TestNewGetDocumentsByTagHandler(t *testing.T) { //nolint:gocognit
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		svc := application.NewDocumentService(&mockDocRepo{ //nolint:exhaustruct
			searchFn: func(ctx context.Context, params domain.SearchDocumentsParams) (*domain.PaginatedResult[domain.Document], error) {
				if len(params.TagIDs) != 1 || params.TagIDs[0] != 7 {
					t.Errorf("expected tag_ids=[7], got %v", params.TagIDs)
				}
				return &domain.PaginatedResult[domain.Document]{
					Total:   2,
					Results: testDocs[:2],
				}, nil
			},
		})

		handler := NewGetDocumentsByTagHandler(svc, newTestCorrSvc(), newTestDocTypeSvc())
		_, output, err := handler(ctx(), nil, GetDocumentsByTagInput{TagID: 7}) //nolint:exhaustruct
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if output.Total != 2 {
			t.Errorf("expected total=2, got %d", output.Total)
		}
		if output.Page != 1 {
			t.Errorf("expected page=1, got %d", output.Page)
		}
		if len(output.Results) != 2 {
			t.Errorf("expected 2 results, got %d", len(output.Results))
		}
	})

	//nolint:dupl
	t.Run("pagination", func(t *testing.T) {
		t.Parallel()

		svc := application.NewDocumentService(&mockDocRepo{ //nolint:exhaustruct
			searchFn: func(ctx context.Context, params domain.SearchDocumentsParams) (*domain.PaginatedResult[domain.Document], error) {
				if params.Page != 2 {
					t.Errorf("expected page=2, got %d", params.Page)
				}
				if params.PageSize != 30 {
					t.Errorf("expected page_size=30, got %d", params.PageSize)
				}
				return &domain.PaginatedResult[domain.Document]{Total: 0, Results: []domain.Document{}}, nil
			},
		})

		handler := NewGetDocumentsByTagHandler(svc, newTestCorrSvc(), newTestDocTypeSvc())
		_, output, err := handler(ctx(), nil, GetDocumentsByTagInput{TagID: 7, Page: 2, PageSize: 30})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if output.Page != 2 {
			t.Errorf("expected page=2, got %d", output.Page)
		}
	})

	t.Run("no results", func(t *testing.T) {
		t.Parallel()

		svc := application.NewDocumentService(&mockDocRepo{ //nolint:exhaustruct
			searchFn: func(ctx context.Context, params domain.SearchDocumentsParams) (*domain.PaginatedResult[domain.Document], error) {
				return &domain.PaginatedResult[domain.Document]{Total: 0, Results: []domain.Document{}}, nil
			},
		})

		handler := NewGetDocumentsByTagHandler(svc, newTestCorrSvc(), newTestDocTypeSvc())
		_, output, err := handler(ctx(), nil, GetDocumentsByTagInput{TagID: 999}) //nolint:exhaustruct
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if output.Total != 0 {
			t.Errorf("expected total=0, got %d", output.Total)
		}
	})

	t.Run("error propagation", func(t *testing.T) {
		t.Parallel()

		svc := application.NewDocumentService(&mockDocRepo{ //nolint:exhaustruct
			searchFn: func(ctx context.Context, params domain.SearchDocumentsParams) (*domain.PaginatedResult[domain.Document], error) {
				return nil, errMock
			},
		})

		handler := NewGetDocumentsByTagHandler(svc, newTestCorrSvc(), newTestDocTypeSvc())
		_, _, err := handler(ctx(), nil, GetDocumentsByTagInput{TagID: 1}) //nolint:exhaustruct
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, ErrSearchFailed) {
			t.Errorf("expected %v, got %v", ErrSearchFailed, err)
		}
	})

	t.Run("document type name resolution error is silently swallowed", func(t *testing.T) {
		t.Parallel()

		svc := application.NewDocumentService(&mockDocRepo{ //nolint:exhaustruct
			searchFn: func(ctx context.Context, params domain.SearchDocumentsParams) (*domain.PaginatedResult[domain.Document], error) {
				return &domain.PaginatedResult[domain.Document]{
					Total:   2,
					Results: testDocs[:2],
				}, nil
			},
		})

		docTypeSvc := newTestDocTypeSvc(func(ctx context.Context, id int) (*domain.DocumentType, error) {
			return nil, errMock
		})
		handler := NewGetDocumentsByTagHandler(svc, newTestCorrSvc(), docTypeSvc)
		_, output, err := handler(ctx(), nil, GetDocumentsByTagInput{TagID: 7}) //nolint:exhaustruct
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		for _, r := range output.Results {
			if r.DocumentType != nil && r.DocumentTypeName != "" {
				t.Errorf("expected empty DocumentTypeName when GetByID fails, got %q for doc %d", r.DocumentTypeName, r.ID)
			}
		}
	})
}

// ============================================================
// get_document_content
// ============================================================
// fulltext_search
// ============================================================

func TestNewFulltextSearchHandler(t *testing.T) { //nolint:gocognit,gocyclo,maintidx
	t.Parallel()

	t.Run("success with query", func(t *testing.T) {
		t.Parallel()

		svc := application.NewDocumentService(&mockDocRepo{ //nolint:exhaustruct
			searchFn: func(ctx context.Context, params domain.SearchDocumentsParams) (*domain.PaginatedResult[domain.Document], error) {
				if params.Query != "invoice" {
					t.Errorf("expected query=%q, got %q", "invoice", params.Query)
				}
				return &domain.PaginatedResult[domain.Document]{
					Total:   2,
					Results: []domain.Document{testDocs[0], testDocs[1]},
				}, nil
			},
		})

		handler := NewFulltextSearchHandler(svc, newTestCorrSvc(), newTestDocTypeSvc())
		_, output, err := handler(ctx(), nil, FulltextSearchInput{Query: "invoice"}) //nolint:exhaustruct
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if output.Total != 2 {
			t.Errorf("expected total=2, got %d", output.Total)
		}
		if output.Page != 1 {
			t.Errorf("expected page=1, got %d", output.Page)
		}
		if len(output.Results) != 2 {
			t.Errorf("expected 2 results, got %d", len(output.Results))
		}
	})

	t.Run("highlights from search hit", func(t *testing.T) {
		t.Parallel()

		svc := application.NewDocumentService(&mockDocRepo{ //nolint:exhaustruct
			searchFn: func(ctx context.Context, params domain.SearchDocumentsParams) (*domain.PaginatedResult[domain.Document], error) {
				return &domain.PaginatedResult[domain.Document]{
					Total:   1,
					Results: []domain.Document{testDocs[0]},
				}, nil
			},
		})

		handler := NewFulltextSearchHandler(svc, newTestCorrSvc(), newTestDocTypeSvc())
		_, output, err := handler(ctx(), nil, FulltextSearchInput{Query: "invoice"}) //nolint:exhaustruct
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(output.Results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(output.Results))
		}
		r := output.Results[0]
		if r.ID != 1 {
			t.Errorf("expected ID=1, got %d", r.ID)
		}
		if r.Title != "Invoice March 2024" {
			t.Errorf("expected Title=%q, got %q", "Invoice March 2024", r.Title)
		}
		if r.Correspondent == nil || *r.Correspondent != 5 {
			t.Errorf("expected correspondent=5, got %v", r.Correspondent)
		}
		if len(r.Tags) != 2 {
			t.Errorf("expected 2 tags, got %d", len(r.Tags))
		}
		if r.Created != "2024-03-15" {
			t.Errorf("expected Created=%q, got %q", "2024-03-15", r.Created)
		}
		if r.Highlights != "OCR text of the invoice" {
			t.Errorf("expected Highlights=%q, got %q", "OCR text of the invoice", r.Highlights)
		}
	})

	t.Run("nil search hit yields empty highlights", func(t *testing.T) {
		t.Parallel()

		svc := application.NewDocumentService(&mockDocRepo{ //nolint:exhaustruct
			searchFn: func(ctx context.Context, params domain.SearchDocumentsParams) (*domain.PaginatedResult[domain.Document], error) {
				return &domain.PaginatedResult[domain.Document]{
					Total:   1,
					Results: []domain.Document{testDocs[1]}, // testDocs[1] has no SearchHit
				}, nil
			},
		})

		handler := NewFulltextSearchHandler(svc, newTestCorrSvc(), newTestDocTypeSvc())
		_, output, err := handler(ctx(), nil, FulltextSearchInput{Query: "contract"}) //nolint:exhaustruct
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(output.Results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(output.Results))
		}
		if output.Results[0].Highlights != "" {
			t.Errorf("expected empty highlights, got %q", output.Results[0].Highlights)
		}
	})

	t.Run("pagination defaults", func(t *testing.T) {
		t.Parallel()

		svc := application.NewDocumentService(&mockDocRepo{ //nolint:exhaustruct
			searchFn: func(ctx context.Context, params domain.SearchDocumentsParams) (*domain.PaginatedResult[domain.Document], error) {
				if params.Page != 1 {
					t.Errorf("expected page=1, got %d", params.Page)
				}
				if params.PageSize != 25 {
					t.Errorf("expected page_size=25, got %d", params.PageSize)
				}
				return &domain.PaginatedResult[domain.Document]{Total: 0, Results: []domain.Document{}}, nil
			},
		})

		handler := NewFulltextSearchHandler(svc, newTestCorrSvc(), newTestDocTypeSvc())
		_, _, err := handler(ctx(), nil, FulltextSearchInput{Query: "test"}) //nolint:exhaustruct
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	//nolint:dupl
	t.Run("pagination passed through", func(t *testing.T) {
		t.Parallel()

		svc := application.NewDocumentService(&mockDocRepo{ //nolint:exhaustruct
			searchFn: func(ctx context.Context, params domain.SearchDocumentsParams) (*domain.PaginatedResult[domain.Document], error) {
				if params.Page != 3 {
					t.Errorf("expected page=3, got %d", params.Page)
				}
				if params.PageSize != 50 {
					t.Errorf("expected page_size=50, got %d", params.PageSize)
				}
				return &domain.PaginatedResult[domain.Document]{Total: 0, Results: []domain.Document{}}, nil
			},
		})

		handler := NewFulltextSearchHandler(svc, newTestCorrSvc(), newTestDocTypeSvc())
		_, output, err := handler(ctx(), nil, FulltextSearchInput{Query: "test", Page: 3, PageSize: 50})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if output.Page != 3 {
			t.Errorf("expected page=3, got %d", output.Page)
		}
	})

	t.Run("empty results", func(t *testing.T) {
		t.Parallel()

		svc := application.NewDocumentService(&mockDocRepo{ //nolint:exhaustruct
			searchFn: func(ctx context.Context, params domain.SearchDocumentsParams) (*domain.PaginatedResult[domain.Document], error) {
				return &domain.PaginatedResult[domain.Document]{Total: 0, Results: []domain.Document{}}, nil
			},
		})

		handler := NewFulltextSearchHandler(svc, newTestCorrSvc(), newTestDocTypeSvc())
		_, output, err := handler(ctx(), nil, FulltextSearchInput{Query: "nonexistent"}) //nolint:exhaustruct
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if output.Total != 0 || len(output.Results) != 0 {
			t.Errorf("expected empty result, got %+v", output)
		}
	})

	t.Run("error propagation", func(t *testing.T) {
		t.Parallel()

		svc := application.NewDocumentService(&mockDocRepo{ //nolint:exhaustruct
			searchFn: func(ctx context.Context, params domain.SearchDocumentsParams) (*domain.PaginatedResult[domain.Document], error) {
				return nil, errMock
			},
		})

		handler := NewFulltextSearchHandler(svc, newTestCorrSvc(), newTestDocTypeSvc())
		_, _, err := handler(ctx(), nil, FulltextSearchInput{Query: "fail"}) //nolint:exhaustruct
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, ErrSearchFailed) {
			t.Errorf("expected %v, got %v", ErrSearchFailed, err)
		}
	})

	t.Run("document type name resolution error is silently swallowed", func(t *testing.T) {
		t.Parallel()

		svc := application.NewDocumentService(&mockDocRepo{ //nolint:exhaustruct
			searchFn: func(ctx context.Context, params domain.SearchDocumentsParams) (*domain.PaginatedResult[domain.Document], error) {
				return &domain.PaginatedResult[domain.Document]{
					Total:   1,
					Results: testDocs[:1],
				}, nil
			},
		})

		docTypeSvc := newTestDocTypeSvc(func(ctx context.Context, id int) (*domain.DocumentType, error) {
			return nil, errMock
		})
		handler := NewFulltextSearchHandler(svc, newTestCorrSvc(), docTypeSvc)
		_, output, err := handler(ctx(), nil, FulltextSearchInput{Query: "invoice"}) //nolint:exhaustruct
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		for _, r := range output.Results {
			if r.DocumentType != nil && r.DocumentTypeName != "" {
				t.Errorf("expected empty DocumentTypeName when GetByID fails, got %q for doc %d", r.DocumentTypeName, r.ID)
			}
		}
	})
}
