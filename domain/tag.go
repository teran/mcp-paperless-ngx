package domain

// Tag represents a tag in Paperless-ngx.
type Tag struct {
	ID                int
	Slug              string
	Name              string
	Color             string
	TextColor         string
	Match             string
	MatchingAlgorithm int
	IsInsensitive     bool
	IsInboxTag        bool
	DocumentCount     int
	Owner             *int
	Parent            *int
	UserCanChange     bool
}
