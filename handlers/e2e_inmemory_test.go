package handlers_test

// Black-box integration test for the whole MCP protocol over the in-memory
// transport, using the RegisterToolsWithServices seam and mock repositories.
//
// This test does NOT require Docker. It verifies that the production seam
// (handlers.RegisterToolsWithServices) wires all 7 tools with the provided
// services and that a real MCP client can list and call them, receiving the
// expected JSON output.

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/teran/mcp-paperless-ngx/application"
	"github.com/teran/mcp-paperless-ngx/domain"
	"github.com/teran/mcp-paperless-ngx/handlers"
)

const (
	testDocID       = 1
	testDocTitle    = "Test Document"
	testDocContent  = "unique-e2e-inmemory full text"
	testDocMimeType = "application/pdf"

	testCorrID   = 1
	testCorrName = "Test Correspondent"

	testTagID   = 1
	testTagName = "Test Tag"
)

// ============================================================
// Mock repositories
// ============================================================

func intPtr(v int) *int { return &v }

func testDocument() domain.Document {
	return domain.Document{ //nolint:exhaustruct
		ID:               testDocID,
		Title:            testDocTitle,
		Content:          testDocContent,
		Correspondent:    intPtr(testCorrID),
		Tags:             []int{testTagID},
		Created:          "2024-01-01",
		MimeType:         testDocMimeType,
		OriginalFileName: "test.pdf",
	}
}

func testCorrespondent() domain.Correspondent {
	return domain.Correspondent{ //nolint:exhaustruct
		ID:            testCorrID,
		Name:          testCorrName,
		Slug:          "test-correspondent",
		DocumentCount: 1,
	}
}

func testTag() domain.Tag {
	return domain.Tag{ //nolint:exhaustruct
		ID:            testTagID,
		Name:          testTagName,
		Color:         "#aabbcc",
		DocumentCount: 1,
	}
}

type mockDocRepo struct{}

func (mockDocRepo) Search(context.Context, domain.SearchDocumentsParams) (*domain.PaginatedResult[domain.Document], error) {
	return &domain.PaginatedResult[domain.Document]{ //nolint:exhaustruct
		Total:   1,
		Results: []domain.Document{testDocument()},
	}, nil
}

func (mockDocRepo) GetByID(context.Context, int) (*domain.Document, error) {
	doc := testDocument()
	return &doc, nil
}

type mockCorrRepo struct{}

func (mockCorrRepo) Search(context.Context, string, int, int) (*domain.PaginatedResult[domain.Correspondent], error) {
	return &domain.PaginatedResult[domain.Correspondent]{ //nolint:exhaustruct
		Total:   1,
		Results: []domain.Correspondent{testCorrespondent()},
	}, nil
}

func (mockCorrRepo) GetByID(context.Context, int) (*domain.Correspondent, error) {
	corr := testCorrespondent()
	return &corr, nil
}

type mockDocTypeRepo struct{}

func (mockDocTypeRepo) GetByID(_ context.Context, id int) (*domain.DocumentType, error) {
	return &domain.DocumentType{ID: id, Name: "Test Document Type"}, nil //nolint:exhaustruct
}

type mockTagRepo struct{}

func (mockTagRepo) List(context.Context, string, int, int) (*domain.PaginatedResult[domain.Tag], error) {
	return &domain.PaginatedResult[domain.Tag]{ //nolint:exhaustruct
		Total:   1,
		Results: []domain.Tag{testTag()},
	}, nil
}

// ============================================================
// Test helpers
// ============================================================

// setupInMemoryMCPServer wires the production seam RegisterToolsWithServices
// with mock-backed services onto an in-memory MCP transport and returns a
// connected client session.
func setupInMemoryMCPServer(t *testing.T) *mcp.ClientSession {
	t.Helper()
	ctx := t.Context()

	docSvc := application.NewDocumentService(mockDocRepo{})
	corrSvc := application.NewCorrespondentService(mockCorrRepo{})
	docTypeSvc := application.NewDocumentTypeService(mockDocTypeRepo{})
	tagSvc := application.NewTagService(mockTagRepo{})

	metrics := handlers.NewMetrics(prometheus.NewRegistry())

	srv := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "test"}, &mcp.ServerOptions{
		Capabilities: &mcp.ServerCapabilities{
			Tools: &mcp.ToolCapabilities{ListChanged: false},
		},
	})

	handlers.RegisterToolsWithServices(srv, metrics, docSvc, corrSvc, docTypeSvc, tagSvc)

	serverT, clientT := mcp.NewInMemoryTransports()
	go func() { _ = srv.Run(ctx, serverT) }()

	cli := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "test"}, nil)
	session, err := cli.Connect(ctx, clientT, &mcp.ClientSessionOptions{})
	if err != nil {
		t.Fatalf("client connect: %v", err)
	}

	return session
}

// callTool invokes a tool over the in-memory MCP session and returns the
// concatenated text of all text content blocks in the result.
func callTool(t *testing.T, session *mcp.ClientSession, name string, args map[string]any) string {
	t.Helper()

	res, err := session.CallTool(t.Context(), &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatalf("CallTool(%s): %v", name, err)
	}
	if res.IsError {
		t.Fatalf("CallTool(%s): IsError=true, content=%v", name, res.Content)
	}

	var sb strings.Builder
	for _, c := range res.Content {
		if tc, ok := c.(*mcp.TextContent); ok {
			sb.WriteString(tc.Text)
		}
	}
	return sb.String()
}

func mustUnmarshal(t *testing.T, data string, v any) {
	t.Helper()
	if err := json.Unmarshal([]byte(data), v); err != nil {
		t.Fatalf("unmarshal JSON output: %v\noutput=%s", err, data)
	}
}

func callToolJSON[T any](t *testing.T, session *mcp.ClientSession, name string, args map[string]any) T {
	t.Helper()
	var out T
	mustUnmarshal(t, callTool(t, session, name, args), &out)
	return out
}

// ============================================================
// Tests
// ============================================================

func TestRegisterToolsWithServices_ListsAllTools(t *testing.T) {
	session := setupInMemoryMCPServer(t)

	res, err := session.ListTools(t.Context(), &mcp.ListToolsParams{})
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}

	want := []string{
		"search_documents",
		"get_document_content",
		"search_correspondents",
		"get_documents_by_correspondent",
		"list_tags",
		"get_documents_by_tag",
		"fulltext_search",
	}

	if len(res.Tools) != len(want) {
		t.Fatalf("expected %d tools, got %d", len(want), len(res.Tools))
	}

	got := make(map[string]bool, len(res.Tools))
	for _, tool := range res.Tools {
		got[tool.Name] = true
	}
	for _, name := range want {
		if !got[name] {
			t.Errorf("missing tool %q", name)
		}
	}
}

func TestRegisterToolsWithServices_SearchDocuments(t *testing.T) {
	session := setupInMemoryMCPServer(t)

	out := callToolJSON[handlers.SearchDocumentsOutput](t, session, "search_documents", map[string]any{
		"query": "test",
	})

	if out.Total != 1 {
		t.Errorf("expected Total=1, got %d", out.Total)
	}
	if len(out.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(out.Results))
	}
	if out.Results[0].ID != testDocID || out.Results[0].Title != testDocTitle {
		t.Errorf("unexpected document: %+v", out.Results[0])
	}
}

func TestRegisterToolsWithServices_GetDocumentContent(t *testing.T) {
	session := setupInMemoryMCPServer(t)

	var out handlers.DocumentDetail
	mustUnmarshal(t, callTool(t, session, "get_document_content", map[string]any{
		"document_id": testDocID,
	}), &out)

	if out.ID != testDocID {
		t.Errorf("expected ID=%d, got %d", testDocID, out.ID)
	}
	if out.Title != testDocTitle {
		t.Errorf("expected title=%q, got %q", testDocTitle, out.Title)
	}
	if out.Content != testDocContent {
		t.Errorf("expected content=%q, got %q", testDocContent, out.Content)
	}
}

func TestRegisterToolsWithServices_SearchCorrespondents(t *testing.T) {
	session := setupInMemoryMCPServer(t)

	out := callToolJSON[handlers.SearchCorrespondentsOutput](t, session, "search_correspondents", map[string]any{
		"query": "Test",
	})

	if out.Total != 1 {
		t.Errorf("expected Total=1, got %d", out.Total)
	}
	if len(out.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(out.Results))
	}
	if out.Results[0].ID != testCorrID || out.Results[0].Name != testCorrName {
		t.Errorf("unexpected correspondent: %+v", out.Results[0])
	}
}

func TestRegisterToolsWithServices_GetDocumentsByCorrespondent(t *testing.T) {
	session := setupInMemoryMCPServer(t)

	out := callToolJSON[handlers.SearchDocumentsOutput](t, session, "get_documents_by_correspondent", map[string]any{
		"correspondent_id": testCorrID,
	})

	if out.Total != 1 {
		t.Errorf("expected Total=1, got %d", out.Total)
	}
	if len(out.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(out.Results))
	}
	if out.Results[0].ID != testDocID {
		t.Errorf("unexpected document: %+v", out.Results[0])
	}
}

func TestRegisterToolsWithServices_ListTags(t *testing.T) {
	session := setupInMemoryMCPServer(t)

	out := callToolJSON[handlers.ListTagsOutput](t, session, "list_tags", map[string]any{
		"query": "Test",
	})

	if out.Total != 1 {
		t.Errorf("expected Total=1, got %d", out.Total)
	}
	if len(out.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(out.Results))
	}
	if out.Results[0].ID != testTagID || out.Results[0].Name != testTagName {
		t.Errorf("unexpected tag: %+v", out.Results[0])
	}
}

func TestRegisterToolsWithServices_GetDocumentsByTag(t *testing.T) {
	session := setupInMemoryMCPServer(t)

	out := callToolJSON[handlers.SearchDocumentsOutput](t, session, "get_documents_by_tag", map[string]any{
		"tag_id": testTagID,
	})

	if out.Total != 1 {
		t.Errorf("expected Total=1, got %d", out.Total)
	}
	if len(out.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(out.Results))
	}
	if out.Results[0].ID != testDocID {
		t.Errorf("unexpected document: %+v", out.Results[0])
	}
}

func TestRegisterToolsWithServices_FulltextSearch(t *testing.T) {
	session := setupInMemoryMCPServer(t)

	out := callToolJSON[handlers.FulltextSearchOutput](t, session, "fulltext_search", map[string]any{
		"query": "unique-e2e-inmemory",
	})

	if out.Total != 1 {
		t.Errorf("expected Total=1, got %d", out.Total)
	}
	if len(out.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(out.Results))
	}
	if out.Results[0].ID != testDocID || out.Results[0].Title != testDocTitle {
		t.Errorf("unexpected document: %+v", out.Results[0])
	}
}
