//go:build e2e

package e2e

// Real end-to-end test against an actual Paperless-ngx instance launched via
// go-docker-testsuite. The MCP server is driven programmatically over the
// in-memory transport (no IPC), using the RegisterToolsWithServices seam wired
// with the real infrastructure client and application services.
//
// Requires Docker. Skipped when the container cannot be started.
//
// Run with: go test -tags e2e ./e2e/

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/prometheus/client_golang/prometheus"
	"resty.dev/v3"

	paperlessngx "github.com/teran/go-docker-testsuite/applications/paperless-ngx"
	"github.com/teran/go-docker-testsuite/images"

	"github.com/teran/mcp-paperless-ngx/application"
	"github.com/teran/mcp-paperless-ngx/handlers"
	infra "github.com/teran/mcp-paperless-ngx/infrastructure/paperless"
)

// uniqueSuffix is shared across the seeded entities so that every test run
// creates distinct names/ids, keeping the test idempotent across repeated
// runs against the same Paperless-ngx instance.
func uniqueSuffix() string {
	return fmt.Sprintf("E2E%d", time.Now().UnixNano())
}

func TestEndToEnd(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Minute)
	defer cancel()

	app, err := paperlessngx.NewWithT(t, ctx, images.PaperlessNGX)
	if err != nil {
		t.Skipf("docker unavailable: %v", err)
	}

	baseURL := app.MustURL()
	username := app.Username()
	password := app.Password()

	unique := uniqueSuffix()
	corrName := "Correspondent " + unique
	tagName := "Tag " + unique
	docTitle := "Document " + unique
	fulltextWord := unique

	// Seed the Paperless-ngx instance via its REST API.
	corrID := createCorrespondent(t, ctx, baseURL, username, password, corrName)
	tagID := createTag(t, ctx, baseURL, username, password, tagName)
	// post_document returns a task UUID (async consumption), not the document
	// id, so the document id is discovered by polling for the processed doc.
	uploadDocument(t, ctx, baseURL, username, password, docTitle, fulltextWord, corrID, tagID)
	docID := waitForDocumentProcessed(t, ctx, baseURL, username, password, fulltextWord)

	// Build real application services backed by the Paperless-ngx client. The
	// MCP server's infrastructure client authenticates via `Authorization:
	// Token <token>`, so obtain a real Paperless-ngx API token for the admin
	// user (created via the token endpoint using the wrapper's Basic creds).
	apiToken := createAPIToken(t, ctx, baseURL, username, password)

	restyClient := resty.New().SetRedirectPolicy(resty.RedirectNoPolicy())
	client := infra.NewClient(baseURL, apiToken, restyClient)
	docSvc := application.NewDocumentService(client)
	corrSvc := application.NewCorrespondentService(infra.NewCorrespondentRepo(client))
	docTypeSvc := application.NewDocumentTypeService(infra.NewDocumentTypeRepo(client))
	tagSvc := application.NewTagService(infra.NewTagRepo(client))

	// Wire the MCP server in-memory.
	srv := mcp.NewServer(&mcp.Implementation{Name: "e2e", Version: "test"}, &mcp.ServerOptions{
		Capabilities: &mcp.ServerCapabilities{
			Tools: &mcp.ToolCapabilities{ListChanged: false},
		},
	})
	metrics := handlers.NewMetrics(prometheus.NewRegistry())
	handlers.RegisterToolsWithServices(srv, metrics, docSvc, corrSvc, docTypeSvc, tagSvc)

	serverT, clientT := mcp.NewInMemoryTransports()
	go func() { _ = srv.Run(ctx, serverT) }()

	cli := mcp.NewClient(&mcp.Implementation{Name: "e2e-client", Version: "test"}, nil)
	session, err := cli.Connect(ctx, clientT, &mcp.ClientSessionOptions{})
	if err != nil {
		t.Fatalf("client connect: %v", err)
	}

	// search_documents — find the seeded document by title.
	searchDocs := callToolJSON[handlers.SearchDocumentsOutput](t, session, "search_documents", map[string]any{
		"query": docTitle,
	})
	if searchDocs.Total == 0 {
		t.Fatal("search_documents: expected the seeded document, found none")
	}
	if !containsDoc(searchDocs.Results, docID) {
		t.Errorf("search_documents: document id %d not found in results: %+v", docID, searchDocs.Results)
	}

	// get_document_content — retrieve the seeded document by id.
	var detail handlers.DocumentDetail
	mustUnmarshal(t, callToolText(t, session, "get_document_content", map[string]any{
		"document_id": docID,
	}), &detail)
	if detail.ID != docID {
		t.Errorf("get_document_content: expected ID=%d, got %d", docID, detail.ID)
	}

	// search_correspondents — find the seeded correspondent by name.
	searchCorrs := callToolJSON[handlers.SearchCorrespondentsOutput](t, session, "search_correspondents", map[string]any{
		"query": corrName,
	})
	if searchCorrs.Total == 0 {
		t.Errorf("search_correspondents: expected the seeded correspondent, found none")
	}

	// list_tags — find the seeded tag by name.
	tags := callToolJSON[handlers.ListTagsOutput](t, session, "list_tags", map[string]any{
		"query": tagName,
	})
	if tags.Total == 0 {
		t.Errorf("list_tags: expected the seeded tag, found none")
	}

	// fulltext_search — match the unique word embedded in the PDF content.
	fulltext := callToolJSON[handlers.FulltextSearchOutput](t, session, "fulltext_search", map[string]any{
		"query": fulltextWord,
	})
	if fulltext.Total == 0 {
		t.Errorf("fulltext_search: expected the seeded document, found none")
	}
}

func containsDoc(results []handlers.DocumentSummary, id int) bool {
	for _, r := range results {
		if r.ID == id {
			return true
		}
	}
	return false
}

// ============================================================
// MCP call helpers
// ============================================================

func callToolText(t *testing.T, session *mcp.ClientSession, name string, args map[string]any) string {
	t.Helper()

	res, err := session.CallTool(t.Context(), &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatalf("CallTool(%s): %v", name, err)
	}
	if res.IsError {
		t.Fatalf("CallTool(%s): IsError=true, content=%v", name, res.Content)
	}

	var sb bytes.Buffer
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
	mustUnmarshal(t, callToolText(t, session, name, args), &out)
	return out
}

// ============================================================
// Paperless-ngx REST seeding helpers (Basic auth)
// ============================================================

func doJSON(t *testing.T, ctx context.Context, baseURL, user, pass, method, path string, body []byte) *http.Response {
	t.Helper()

	req, err := http.NewRequestWithContext(ctx, method, baseURL+path, bytes.NewReader(body))
	if err != nil {
		t.Fatalf("new request %s: %v", path, err)
	}
	req.SetBasicAuth(user, pass)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request %s: %v", path, err)
	}
	return resp
}

func decodeCreatedID(t *testing.T, resp *http.Response) int {
	t.Helper()
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		t.Fatalf("unexpected status=%d body=%s", resp.StatusCode, string(body))
	}

	var created struct {
		ID int `json:"id"`
	}
	if err := json.Unmarshal(body, &created); err != nil {
		t.Fatalf("decode created entity: %v (body=%s)", err, string(body))
	}
	return created.ID
}

// createAPIToken creates a Paperless-ngx API token for the admin user. It is
// obtained via the `ObtainAuthToken` view (`POST /api/token/`), which
// validates the JSON username/password and returns {"token": "..."}. The
// endpoint is CSRF-protected, so we first fetch a page to obtain the
// `csrftoken` cookie (via a cookie jar) and echo it back in `X-CSRFToken`.
func createAPIToken(t *testing.T, ctx context.Context, baseURL, user, pass string) string {
	t.Helper()

	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("cookiejar: %v", err)
	}
	cli := &http.Client{Jar: jar}

	// 1. Fetch the API root to obtain the csrftoken cookie.
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/api/", nil)
	if err != nil {
		t.Fatalf("new csrf request: %v", err)
	}
	if _, err := cli.Do(req); err != nil {
		t.Fatalf("fetch csrf cookie: %v", err)
	}

	csrf := ""
	base, _ := url.Parse(baseURL)
	for _, c := range jar.Cookies(base) {
		if c.Name == "csrftoken" {
			csrf = c.Value
		}
	}
	if csrf == "" {
		t.Fatalf("no csrftoken cookie received from %s", baseURL+"/api/")
	}

	// 2. POST the token view with the CSRF token.
	payload, err := json.Marshal(map[string]string{"username": user, "password": pass})
	if err != nil {
		t.Fatalf("marshal token request: %v", err)
	}

	req, err = http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/api/token/", bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("new token request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-CSRFToken", csrf)

	resp, err := cli.Do(req)
	if err != nil {
		t.Fatalf("create api token: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		t.Fatalf("create api token: unexpected status=%d\nBODY:\n%s", resp.StatusCode, string(body))
	}

	var out struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatalf("decode api token: %v (body=%s)", err, string(body))
	}
	if out.Token == "" {
		t.Fatalf("create api token: empty token (body=%s)", string(body))
	}
	return out.Token
}

func createCorrespondent(t *testing.T, ctx context.Context, baseURL, user, pass, name string) int {
	t.Helper()
	resp := doJSON(t, ctx, baseURL, user, pass, http.MethodPost, "/api/correspondents/", []byte(fmt.Sprintf(`{"name":%q}`, name)))
	return decodeCreatedID(t, resp)
}

func createTag(t *testing.T, ctx context.Context, baseURL, user, pass, name string) int {
	t.Helper()
	resp := doJSON(t, ctx, baseURL, user, pass, http.MethodPost, "/api/tags/", []byte(fmt.Sprintf(`{"name":%q}`, name)))
	return decodeCreatedID(t, resp)
}

func uploadDocument(t *testing.T, ctx context.Context, baseURL, user, pass, title, pdfText string, corrID, tagID int) {
	t.Helper()

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)

	fw, err := w.CreateFormFile("document", "e2e.pdf")
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := fw.Write(buildPDF(pdfText)); err != nil {
		t.Fatalf("write pdf: %v", err)
	}
	if err := w.WriteField("title", title); err != nil {
		t.Fatalf("write title field: %v", err)
	}
	if err := w.WriteField("correspondent", strconv.Itoa(corrID)); err != nil {
		t.Fatalf("write correspondent field: %v", err)
	}
	if err := w.WriteField("tags", strconv.Itoa(tagID)); err != nil {
		t.Fatalf("write tags field: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/api/documents/post_document/", &buf)
	if err != nil {
		t.Fatalf("new post document request: %v", err)
	}
	req.SetBasicAuth(user, pass)
	req.Header.Set("Content-Type", w.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("post document: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("post document: unexpected status=%d body=%s", resp.StatusCode, string(body))
	}
}

// waitForDocumentProcessed polls the documents endpoint until the seeded
// document has been consumed and its extracted content is available, then
// returns its id.
func waitForDocumentProcessed(t *testing.T, ctx context.Context, baseURL, user, pass, query string) int {
	t.Helper()

	deadline := time.Now().Add(3 * time.Minute)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			t.Fatalf("context done while waiting for document processing: %v", ctx.Err())
		default:
		}

		endpoint := baseURL + "/api/documents/?" + url.Values{"query": {query}}.Encode()
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		if err != nil {
			t.Fatal(err)
		}
		req.SetBasicAuth(user, pass)

		resp, err := http.DefaultClient.Do(req)
		if err == nil {
			var list struct {
				Count   int `json:"count"`
				Results []struct {
					ID      int    `json:"id"`
					Content string `json:"content"`
				} `json:"results"`
			}
			decodeErr := json.NewDecoder(resp.Body).Decode(&list)
			_ = resp.Body.Close()
			if decodeErr == nil {
				for _, d := range list.Results {
					if d.Content != "" {
						return d.ID
					}
				}
			}
		}

		time.Sleep(2 * time.Second)
	}

	t.Fatalf("document matching %q was not processed in time", query)
	return 0
}

// buildPDF generates a minimal, strictly-valid single-page PDF whose embedded
// text is the given string. Byte offsets for the xref table are computed so
// that PDF tooling (tika/poppler used by Paperless-ngx) can read the text.
func buildPDF(text string) []byte {
	stream := fmt.Sprintf("BT /F1 24 Tf 72 720 Td (%s) Tj ET", text)
	objs := []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Contents 4 0 R /Resources << /Font << /F1 5 0 R >> >> >>",
		fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", len(stream), stream),
		"<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>",
	}

	var buf bytes.Buffer
	buf.WriteString("%PDF-1.4\n")

	offsets := make([]int, len(objs)+1)
	for i, o := range objs {
		offsets[i+1] = buf.Len()
		fmt.Fprintf(&buf, "%d 0 obj\n%s\nendobj\n", i+1, o)
	}

	xrefPos := buf.Len()
	buf.WriteString("xref\n0 6\n0000000000 65535 f \n")
	for i := 1; i <= len(objs); i++ {
		fmt.Fprintf(&buf, "%010d 00000 n \n", offsets[i])
	}
	buf.WriteString("trailer\n<< /Size 6 /Root 1 0 R >>\nstartxref\n")
	fmt.Fprintf(&buf, "%d\n%%%%EOF\n", xrefPos)

	return buf.Bytes()
}
