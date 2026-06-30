package domain

// Correspondent represents a correspondent in Paperless-ngx.
type Correspondent struct {
	ID                 int
	Slug               string
	Name               string
	Match              string
	MatchingAlgorithm  int
	IsInsensitive      bool
	DocumentCount      int
	LastCorrespondence string
	Owner              *int
	UserCanChange      bool
}
