package paperless_test

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/teran/mcp-paperless-ngx/domain"
	"github.com/teran/mcp-paperless-ngx/infrastructure/paperless"
)

// errBody is an io.Reader that always returns the given error.
type errBody struct{ err error }

func (e *errBody) Read([]byte) (int, error) { return 0, e.err }
func (e *errBody) Close() error             { return nil }

// errReaderTransport returns a response whose body fails to read, simulating
// an I/O error while reading the upstream response body.
type errReaderTransport struct{}

func (*errReaderTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       &errBody{err: errors.New("simulated read failure")},
		Header:     make(http.Header),
	}, nil
}

// ---------------------------------------------------------------------------
// decode (malformed JSON) error paths
// ---------------------------------------------------------------------------

func TestClient_SearchCorrespondents_MalformedJSON(t *testing.T) {
	t.Parallel()

	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`not json at all`))
	})
	defer srv.Close()

	client := newClient(srv.URL)
	_, err := client.SearchCorrespondents(t.Context(), "foo", 1, 25)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "unmarshal correspondents") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestClient_ListTags_MalformedJSON(t *testing.T) {
	t.Parallel()

	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`not json at all`))
	})
	defer srv.Close()

	client := newClient(srv.URL)
	_, err := client.ListTags(t.Context(), "foo", 1, 25)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "unmarshal tags") {
		t.Errorf("unexpected error: %v", err)
	}
}

// ---------------------------------------------------------------------------
// doRequest error paths
// ---------------------------------------------------------------------------

func TestClient_InvalidBaseURL_BuildURL(t *testing.T) {
	t.Parallel()

	// An unparseable base URL makes url.JoinPath fail inside doRequest.
	client := paperless.NewClient("http://[::1", "token", testHTTPClient)
	_, err := client.Search(t.Context(), domainSearchParams())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "build URL") {
		t.Errorf("expected 'build URL' error, got: %v", err)
	}
}

func TestClient_ResponseBodyReadError(t *testing.T) {
	t.Parallel()

	client := paperless.NewClient("http://example.com", "token", &http.Client{
		Transport: &errReaderTransport{},
	})
	_, err := client.Search(t.Context(), domainSearchParams())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "read response body") {
		t.Errorf("expected 'read response body' error, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// extractErrorDetail branches (via error responses)
// ---------------------------------------------------------------------------

func TestClient_Error_EmptyBody_StatusText(t *testing.T) {
	t.Parallel()

	// Empty error body -> extractErrorDetail falls back to http.StatusText.
	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	})
	defer srv.Close()

	client := newClient(srv.URL)
	_, err := client.Search(t.Context(), domainSearchParams())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, paperless.ErrAPIClient) {
		t.Errorf("expected ErrAPIClient, got %v", err)
	}
	if !strings.Contains(err.Error(), "Bad Gateway") {
		t.Errorf("expected StatusText in error, got: %v", err)
	}
}

func TestClient_Error_MultilineBody_FirstLine(t *testing.T) {
	t.Parallel()

	// Multi-line error body -> extractErrorDetail keeps only the first line.
	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("first line of detail\nsecond line leaked"))
	})
	defer srv.Close()

	client := newClient(srv.URL)
	_, err := client.Search(t.Context(), domainSearchParams())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "first line of detail") {
		t.Errorf("expected first line in error, got: %v", err)
	}
	if strings.Contains(err.Error(), "second line") {
		t.Errorf("expected multiline body to be truncated, got: %v", err)
	}
}

func TestClient_Error_LongBody_Truncated(t *testing.T) {
	t.Parallel()

	// Error body longer than 512 bytes -> extractErrorDetail truncates.
	longBody := strings.Repeat("x", 600)
	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(longBody))
	})
	defer srv.Close()

	client := newClient(srv.URL)
	_, err := client.Search(t.Context(), domainSearchParams())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "...") {
		t.Errorf("expected truncation marker in error, got: %v", err)
	}
	if strings.Contains(err.Error(), strings.Repeat("x", 513)) {
		t.Errorf("expected long error body to be truncated to 512 bytes")
	}
}

// domainSearchParams returns a minimal set of search params for the client.
func domainSearchParams() domain.SearchDocumentsParams {
	return domain.SearchDocumentsParams{ //nolint:exhaustruct
		Page:     1,
		PageSize: 25,
	}
}
