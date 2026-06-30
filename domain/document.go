package domain

// Document represents a document in Paperless-ngx.
type Document struct {
	ID                  int
	Correspondent       *int
	DocumentType        *int
	StoragePath         *int
	Title               string
	Content             string
	Tags                []int
	Created             string
	CreatedDate         string
	Modified            string
	Added               string
	ArchiveSerialNumber *int
	OriginalFileName    string
	ArchivedFileName    *string
	Owner               *int
	PageCount           *int
	MimeType            string
	SearchHit           *SearchHit
}

// SearchHit contains full-text search metadata.
type SearchHit struct {
	Score      float64
	Highlights string
	Rank       int
}

// PaginatedResult is a generic paginated result.
type PaginatedResult[T any] struct {
	Total   int
	Results []T
}

// SearchDocumentsParams holds search parameters for documents.
type SearchDocumentsParams struct {
	Query           string
	CorrespondentID int
	TagIDs          []int
	CreatedAfter    string
	CreatedBefore   string
	Page            int
	PageSize        int
}
