package paperless

// Document represents a document in Paperless-ngx.
type Document struct {
	ID                  int        `json:"id"`
	Correspondent       *int       `json:"correspondent"`
	DocumentType        *int       `json:"document_type"`
	StoragePath         *int       `json:"storage_path"`
	Title               string     `json:"title"`
	Content             string     `json:"content"`
	Tags                []int      `json:"tags"`
	Created             string     `json:"created"`
	CreatedDate         string     `json:"created_date"`
	Modified            string     `json:"modified"`
	Added               string     `json:"added"`
	ArchiveSerialNumber *int       `json:"archive_serial_number"`
	OriginalFileName    string     `json:"original_file_name"`
	ArchivedFileName    *string    `json:"archived_file_name"`
	Owner               *int       `json:"owner"`
	PageCount           *int       `json:"page_count"`
	MimeType            string     `json:"mime_type"`
	SearchHit           *SearchHit `json:"__search_hit__,omitempty"`
}

// SearchHit contains full-text search metadata.
type SearchHit struct {
	Score      float64 `json:"score"`
	Highlights string  `json:"highlights"`
	Rank       int     `json:"rank"`
}

// PaginatedResponse is a generic paginated response from Paperless-ngx.
type PaginatedResponse struct {
	Count    int        `json:"count"`
	Next     *string    `json:"next"`
	Previous *string    `json:"previous"`
	Results  []Document `json:"results"`
}

// Correspondent represents a correspondent in Paperless-ngx.
type Correspondent struct {
	ID                 int    `json:"id"`
	Slug               string `json:"slug"`
	Name               string `json:"name"`
	Match              string `json:"match"`
	MatchingAlgorithm  int    `json:"matching_algorithm"`
	IsInsensitive      bool   `json:"is_insensitive"`
	DocumentCount      int    `json:"document_count"`
	LastCorrespondence string `json:"last_correspondence"`
	Owner              *int   `json:"owner"`
	UserCanChange      bool   `json:"user_can_change"`
}

// CorrespondentPaginatedResponse is the paginated response for correspondents.
type CorrespondentPaginatedResponse struct {
	Count    int             `json:"count"`
	Next     *string         `json:"next"`
	Previous *string         `json:"previous"`
	Results  []Correspondent `json:"results"`
}

// Tag represents a tag in Paperless-ngx.
type Tag struct {
	ID                int    `json:"id"`
	Slug              string `json:"slug"`
	Name              string `json:"name"`
	Color             string `json:"color"`
	TextColor         string `json:"text_color"`
	Match             string `json:"match"`
	MatchingAlgorithm int    `json:"matching_algorithm"`
	IsInsensitive     bool   `json:"is_insensitive"`
	IsInboxTag        bool   `json:"is_inbox_tag"`
	DocumentCount     int    `json:"document_count"`
	Owner             *int   `json:"owner"`
	Parent            *int   `json:"parent"`
	UserCanChange     bool   `json:"user_can_change"`
}

// TagPaginatedResponse is the paginated response for tags.
type TagPaginatedResponse struct {
	Count    int     `json:"count"`
	Next     *string `json:"next"`
	Previous *string `json:"previous"`
	Results  []Tag   `json:"results"`
}
