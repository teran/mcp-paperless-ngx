package paperless

import "github.com/teran/mcp-paperless-ngx/domain"

// raw models — JSON representation matching Paperless-ngx API wire format.

type rawDocument struct {
	ID                  int           `json:"id"`
	Correspondent       *int          `json:"correspondent"`
	DocumentType        *int          `json:"document_type"`
	StoragePath         *int          `json:"storage_path"`
	Title               string        `json:"title"`
	Content             string        `json:"content"`
	Tags                []int         `json:"tags"`
	Created             string        `json:"created"`
	CreatedDate         string        `json:"created_date"`
	Modified            string        `json:"modified"`
	Added               string        `json:"added"`
	ArchiveSerialNumber *int          `json:"archive_serial_number"`
	OriginalFileName    string        `json:"original_file_name"`
	ArchivedFileName    *string       `json:"archived_file_name"`
	Owner               *int          `json:"owner"`
	PageCount           *int          `json:"page_count"`
	MimeType            string        `json:"mime_type"`
	SearchHit           *rawSearchHit `json:"__search_hit__,omitempty"`
}

type rawSearchHit struct {
	Score      float64 `json:"score"`
	Highlights string  `json:"highlights"`
	Rank       int     `json:"rank"`
}

type rawCorrespondent struct {
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

type rawTag struct {
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

type rawPaginatedDocuments struct {
	Count   int           `json:"count"`
	Results []rawDocument `json:"results"`
}

type rawPaginatedCorrespondents struct {
	Count   int                `json:"count"`
	Results []rawCorrespondent `json:"results"`
}

type rawDocumentType struct {
	ID                int    `json:"id"`
	Slug              string `json:"slug"`
	Name              string `json:"name"`
	Match             string `json:"match"`
	MatchingAlgorithm int    `json:"matching_algorithm"`
	IsInsensitive     bool   `json:"is_insensitive"`
	DocumentCount     int    `json:"document_count"`
	Owner             *int   `json:"owner"`
	UserCanChange     bool   `json:"user_can_change"`
}

type rawPaginatedTags struct {
	Count   int      `json:"count"`
	Results []rawTag `json:"results"`
}

// -- domain conversion --

func (r rawDocument) toDomain() domain.Document {
	var sh *domain.SearchHit
	if r.SearchHit != nil {
		sh = &domain.SearchHit{
			Score:      r.SearchHit.Score,
			Highlights: r.SearchHit.Highlights,
			Rank:       r.SearchHit.Rank,
		}
	}

	return domain.Document{
		ID:                  r.ID,
		Correspondent:       r.Correspondent,
		DocumentType:        r.DocumentType,
		StoragePath:         r.StoragePath,
		Title:               r.Title,
		Content:             r.Content,
		Tags:                r.Tags,
		Created:             r.Created,
		CreatedDate:         r.CreatedDate,
		Modified:            r.Modified,
		Added:               r.Added,
		ArchiveSerialNumber: r.ArchiveSerialNumber,
		OriginalFileName:    r.OriginalFileName,
		ArchivedFileName:    r.ArchivedFileName,
		Owner:               r.Owner,
		PageCount:           r.PageCount,
		MimeType:            r.MimeType,
		SearchHit:           sh,
	}
}

func (r rawCorrespondent) toDomain() domain.Correspondent {
	return domain.Correspondent{
		ID:                 r.ID,
		Slug:               r.Slug,
		Name:               r.Name,
		Match:              r.Match,
		MatchingAlgorithm:  r.MatchingAlgorithm,
		IsInsensitive:      r.IsInsensitive,
		DocumentCount:      r.DocumentCount,
		LastCorrespondence: r.LastCorrespondence,
		Owner:              r.Owner,
		UserCanChange:      r.UserCanChange,
	}
}

func (r rawDocumentType) toDomain() domain.DocumentType {
	return domain.DocumentType{
		ID:                r.ID,
		Slug:              r.Slug,
		Name:              r.Name,
		Match:             r.Match,
		MatchingAlgorithm: r.MatchingAlgorithm,
		IsInsensitive:     r.IsInsensitive,
		DocumentCount:     r.DocumentCount,
		Owner:             r.Owner,
		UserCanChange:     r.UserCanChange,
	}
}

func (r rawTag) toDomain() domain.Tag {
	return domain.Tag{
		ID:                r.ID,
		Slug:              r.Slug,
		Name:              r.Name,
		Color:             r.Color,
		TextColor:         r.TextColor,
		Match:             r.Match,
		MatchingAlgorithm: r.MatchingAlgorithm,
		IsInsensitive:     r.IsInsensitive,
		IsInboxTag:        r.IsInboxTag,
		DocumentCount:     r.DocumentCount,
		Owner:             r.Owner,
		Parent:            r.Parent,
		UserCanChange:     r.UserCanChange,
	}
}
