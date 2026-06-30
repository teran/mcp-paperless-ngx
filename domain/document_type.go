package domain

// DocumentType represents a document type in Paperless-ngx.
type DocumentType struct {
	ID                int
	Slug              string
	Name              string
	Match             string
	MatchingAlgorithm int
	IsInsensitive     bool
	DocumentCount     int
	Owner             *int
	UserCanChange     bool
}
