package domain

import (
	"testing"
)

// ---------------------------------------------------------------------------
// Document tests
// ---------------------------------------------------------------------------

func TestDocument_Populate(t *testing.T) { //nolint:gocyclo
	correspondent := 10
	docType := 20
	storagePath := 30
	archiveSerial := 1
	archivedFileName := "doc_1.pdf"
	owner := 42
	pageCount := 5

	doc := Document{
		ID:                  1,
		Correspondent:       &correspondent,
		DocumentType:        &docType,
		StoragePath:         &storagePath,
		Title:               "Test Document",
		Content:             "Full text content of the document",
		Tags:                []int{3, 7, 11},
		Created:             "2024-01-15T10:00:00Z",
		CreatedDate:         "2024-01-15",
		Modified:            "2024-01-16T12:30:00Z",
		Added:               "2024-01-16T14:00:00Z",
		ArchiveSerialNumber: &archiveSerial,
		OriginalFileName:    "original.pdf",
		ArchivedFileName:    &archivedFileName,
		Owner:               &owner,
		PageCount:           &pageCount,
		MimeType:            "application/pdf",
		SearchHit:           &SearchHit{Score: 0.95, Highlights: "Test <b>Document</b>", Rank: 1}, //nolint:exhaustruct
	}

	if doc.ID != 1 {
		t.Errorf("ID = %d, want %d", doc.ID, 1)
	}
	if doc.Correspondent == nil || *doc.Correspondent != 10 {
		t.Errorf("Correspondent = %v, want %d", doc.Correspondent, 10)
	}
	if doc.DocumentType == nil || *doc.DocumentType != 20 {
		t.Errorf("DocumentType = %v, want %d", doc.DocumentType, 20)
	}
	if doc.StoragePath == nil || *doc.StoragePath != 30 {
		t.Errorf("StoragePath = %v, want %d", doc.StoragePath, 30)
	}
	if doc.Title != "Test Document" {
		t.Errorf("Title = %q, want %q", doc.Title, "Test Document")
	}
	if doc.Content != "Full text content of the document" {
		t.Errorf("Content = %q, want %q", doc.Content, "Full text content of the document")
	}
	if len(doc.Tags) != 3 || doc.Tags[0] != 3 || doc.Tags[1] != 7 || doc.Tags[2] != 11 {
		t.Errorf("Tags = %v, want [3 7 11]", doc.Tags)
	}
	if doc.Created != "2024-01-15T10:00:00Z" {
		t.Errorf("Created = %q, want %q", doc.Created, "2024-01-15T10:00:00Z")
	}
	if doc.CreatedDate != "2024-01-15" {
		t.Errorf("CreatedDate = %q, want %q", doc.CreatedDate, "2024-01-15")
	}
	if doc.Modified != "2024-01-16T12:30:00Z" {
		t.Errorf("Modified = %q, want %q", doc.Modified, "2024-01-16T12:30:00Z")
	}
	if doc.Added != "2024-01-16T14:00:00Z" {
		t.Errorf("Added = %q, want %q", doc.Added, "2024-01-16T14:00:00Z")
	}
	if doc.ArchiveSerialNumber == nil || *doc.ArchiveSerialNumber != 1 {
		t.Errorf("ArchiveSerialNumber = %v, want %d", doc.ArchiveSerialNumber, 1)
	}
	if doc.OriginalFileName != "original.pdf" {
		t.Errorf("OriginalFileName = %q, want %q", doc.OriginalFileName, "original.pdf")
	}
	if doc.ArchivedFileName == nil || *doc.ArchivedFileName != "doc_1.pdf" {
		t.Errorf("ArchivedFileName = %v, want %q", doc.ArchivedFileName, "doc_1.pdf")
	}
	if doc.Owner == nil || *doc.Owner != 42 {
		t.Errorf("Owner = %v, want %d", doc.Owner, 42)
	}
	if doc.PageCount == nil || *doc.PageCount != 5 {
		t.Errorf("PageCount = %v, want %d", doc.PageCount, 5)
	}
	if doc.MimeType != "application/pdf" {
		t.Errorf("MimeType = %q, want %q", doc.MimeType, "application/pdf")
	}
	if doc.SearchHit == nil || doc.SearchHit.Score != 0.95 || doc.SearchHit.Highlights != "Test <b>Document</b>" || doc.SearchHit.Rank != 1 {
		t.Errorf("SearchHit = %+v, want Score=0.95, Highlights=..., Rank=1", doc.SearchHit)
	}
}

func TestDocument_ZeroValues(t *testing.T) {
	var doc Document //nolint:exhaustruct

	if doc.ID != 0 {
		t.Errorf("ID = %d, want 0", doc.ID)
	}
	if doc.Correspondent != nil {
		t.Errorf("Correspondent = %v, want nil", doc.Correspondent)
	}
	if doc.DocumentType != nil {
		t.Errorf("DocumentType = %v, want nil", doc.DocumentType)
	}
	if doc.StoragePath != nil {
		t.Errorf("StoragePath = %v, want nil", doc.StoragePath)
	}
	if doc.Title != "" {
		t.Errorf("Title = %q, want empty", doc.Title)
	}
	if doc.Content != "" {
		t.Errorf("Content = %q, want empty", doc.Content)
	}
	if doc.Tags != nil {
		t.Errorf("Tags = %v, want nil", doc.Tags)
	}
	if doc.Created != "" {
		t.Errorf("Created = %q, want empty", doc.Created)
	}
	if doc.CreatedDate != "" {
		t.Errorf("CreatedDate = %q, want empty", doc.CreatedDate)
	}
	if doc.Modified != "" {
		t.Errorf("Modified = %q, want empty", doc.Modified)
	}
	if doc.Added != "" {
		t.Errorf("Added = %q, want empty", doc.Added)
	}
	if doc.ArchiveSerialNumber != nil {
		t.Errorf("ArchiveSerialNumber = %v, want nil", doc.ArchiveSerialNumber)
	}
	if doc.OriginalFileName != "" {
		t.Errorf("OriginalFileName = %q, want empty", doc.OriginalFileName)
	}
	if doc.ArchivedFileName != nil {
		t.Errorf("ArchivedFileName = %v, want nil", doc.ArchivedFileName)
	}
	if doc.Owner != nil {
		t.Errorf("Owner = %v, want nil", doc.Owner)
	}
	if doc.PageCount != nil {
		t.Errorf("PageCount = %v, want nil", doc.PageCount)
	}
	if doc.MimeType != "" {
		t.Errorf("MimeType = %q, want empty", doc.MimeType)
	}
	if doc.SearchHit != nil {
		t.Errorf("SearchHit = %v, want nil", doc.SearchHit)
	}
}

// ---------------------------------------------------------------------------
// SearchHit tests
// ---------------------------------------------------------------------------

func TestSearchHit_Populate(t *testing.T) {
	sh := SearchHit{
		Score:      3.14,
		Highlights: "highlighted <b>text</b>",
		Rank:       5,
	}

	if sh.Score != 3.14 {
		t.Errorf("Score = %f, want %f", sh.Score, 3.14)
	}
	if sh.Highlights != "highlighted <b>text</b>" {
		t.Errorf("Highlights = %q, want %q", sh.Highlights, "highlighted <b>text</b>")
	}
	if sh.Rank != 5 {
		t.Errorf("Rank = %d, want %d", sh.Rank, 5)
	}
}

func TestSearchHit_ZeroValues(t *testing.T) {
	var sh SearchHit

	if sh.Score != 0.0 {
		t.Errorf("Score = %f, want 0.0", sh.Score)
	}
	if sh.Highlights != "" {
		t.Errorf("Highlights = %q, want empty", sh.Highlights)
	}
	if sh.Rank != 0 {
		t.Errorf("Rank = %d, want 0", sh.Rank)
	}
}

// ---------------------------------------------------------------------------
// PaginatedResult tests
// ---------------------------------------------------------------------------

func TestPaginatedResult_WithInt(t *testing.T) {
	r := PaginatedResult[int]{
		Total:   3,
		Results: []int{10, 20, 30},
	}

	if r.Total != 3 {
		t.Errorf("Total = %d, want %d", r.Total, 3)
	}
	if len(r.Results) != 3 {
		t.Fatalf("len(Results) = %d, want %d", len(r.Results), 3)
	}
	for i, v := range r.Results {
		if v != (i+1)*10 {
			t.Errorf("Results[%d] = %d, want %d", i, v, (i+1)*10)
		}
	}
}

func TestPaginatedResult_WithString(t *testing.T) {
	r := PaginatedResult[string]{
		Total:   2,
		Results: []string{"foo", "bar"},
	}

	if r.Total != 2 {
		t.Errorf("Total = %d, want %d", r.Total, 2)
	}
	if len(r.Results) != 2 {
		t.Fatalf("len(Results) = %d, want %d", len(r.Results), 2)
	}
	if r.Results[0] != "foo" || r.Results[1] != "bar" {
		t.Errorf("Results = %v, want [foo bar]", r.Results)
	}
}

func TestPaginatedResult_WithCustomType(t *testing.T) {
	r := PaginatedResult[SearchHit]{
		Total: 1,
		Results: []SearchHit{
			{Score: 2.5, Highlights: "match", Rank: 1}, //nolint:exhaustruct
		},
	}

	if r.Total != 1 {
		t.Errorf("Total = %d, want %d", r.Total, 1)
	}
	if len(r.Results) != 1 {
		t.Fatalf("len(Results) = %d, want %d", len(r.Results), 1)
	}
	if r.Results[0].Score != 2.5 || r.Results[0].Rank != 1 {
		t.Errorf("Results[0] = %+v, want Score=2.5, Rank=1", r.Results[0])
	}
}

func TestPaginatedResult_ZeroValues(t *testing.T) {
	var r PaginatedResult[int]

	if r.Total != 0 {
		t.Errorf("Total = %d, want 0", r.Total)
	}
	if r.Results != nil {
		t.Errorf("Results = %v, want nil", r.Results)
	}
}

func TestPaginatedResult_EmptyResults(t *testing.T) {
	r := PaginatedResult[string]{
		Total:   0,
		Results: []string{},
	}

	if r.Total != 0 {
		t.Errorf("Total = %d, want 0", r.Total)
	}
	if len(r.Results) != 0 {
		t.Errorf("len(Results) = %d, want 0", len(r.Results))
	}
}

// ---------------------------------------------------------------------------
// SearchDocumentsParams tests
// ---------------------------------------------------------------------------

func TestSearchDocumentsParams_Populate(t *testing.T) {
	p := SearchDocumentsParams{
		Query:           "invoice",
		CorrespondentID: 5,
		TagIDs:          []int{2, 4},
		CreatedAfter:    "2024-01-01",
		CreatedBefore:   "2024-12-31",
		Page:            1,
		PageSize:        50,
	}

	if p.Query != "invoice" {
		t.Errorf("Query = %q, want %q", p.Query, "invoice")
	}
	if p.CorrespondentID != 5 {
		t.Errorf("CorrespondentID = %d, want %d", p.CorrespondentID, 5)
	}
	if len(p.TagIDs) != 2 || p.TagIDs[0] != 2 || p.TagIDs[1] != 4 {
		t.Errorf("TagIDs = %v, want [2 4]", p.TagIDs)
	}
	if p.CreatedAfter != "2024-01-01" {
		t.Errorf("CreatedAfter = %q, want %q", p.CreatedAfter, "2024-01-01")
	}
	if p.CreatedBefore != "2024-12-31" {
		t.Errorf("CreatedBefore = %q, want %q", p.CreatedBefore, "2024-12-31")
	}
	if p.Page != 1 {
		t.Errorf("Page = %d, want %d", p.Page, 1)
	}
	if p.PageSize != 50 {
		t.Errorf("PageSize = %d, want %d", p.PageSize, 50)
	}
}

func TestSearchDocumentsParams_ZeroValues(t *testing.T) {
	var p SearchDocumentsParams

	if p.Query != "" {
		t.Errorf("Query = %q, want empty", p.Query)
	}
	if p.CorrespondentID != 0 {
		t.Errorf("CorrespondentID = %d, want 0", p.CorrespondentID)
	}
	if p.TagIDs != nil {
		t.Errorf("TagIDs = %v, want nil", p.TagIDs)
	}
	if p.CreatedAfter != "" {
		t.Errorf("CreatedAfter = %q, want empty", p.CreatedAfter)
	}
	if p.CreatedBefore != "" {
		t.Errorf("CreatedBefore = %q, want empty", p.CreatedBefore)
	}
	if p.Page != 0 {
		t.Errorf("Page = %d, want 0", p.Page)
	}
	if p.PageSize != 0 {
		t.Errorf("PageSize = %d, want 0", p.PageSize)
	}
}

func TestSearchDocumentsParams_FilterByQuery(t *testing.T) {
	p := SearchDocumentsParams{Query: "electric bill", Page: 1, PageSize: 10} //nolint:exhaustruct

	if p.Query != "electric bill" {
		t.Errorf("Query = %q, want %q", p.Query, "electric bill")
	}
	if p.Page != 1 || p.PageSize != 10 {
		t.Errorf("Page/PageSize = %d/%d, want 1/10", p.Page, p.PageSize)
	}
}

func TestSearchDocumentsParams_FilterByCorrespondent(t *testing.T) {
	p := SearchDocumentsParams{CorrespondentID: 99, Page: 2, PageSize: 20} //nolint:exhaustruct

	if p.CorrespondentID != 99 {
		t.Errorf("CorrespondentID = %d, want %d", p.CorrespondentID, 99)
	}
	if p.Page != 2 || p.PageSize != 20 {
		t.Errorf("Page/PageSize = %d/%d, want 2/20", p.Page, p.PageSize)
	}
}

func TestSearchDocumentsParams_FilterByTags(t *testing.T) {
	p := SearchDocumentsParams{TagIDs: []int{1, 2, 3}, Page: 1, PageSize: 25} //nolint:exhaustruct

	if len(p.TagIDs) != 3 || p.TagIDs[0] != 1 || p.TagIDs[1] != 2 || p.TagIDs[2] != 3 {
		t.Errorf("TagIDs = %v, want [1 2 3]", p.TagIDs)
	}
	if p.Page != 1 || p.PageSize != 25 {
		t.Errorf("Page/PageSize = %d/%d, want 1/25", p.Page, p.PageSize)
	}
}

func TestSearchDocumentsParams_FilterByDateRange(t *testing.T) {
	p := SearchDocumentsParams{CreatedAfter: "2024-06-01", CreatedBefore: "2024-06-30"} //nolint:exhaustruct

	if p.CreatedAfter != "2024-06-01" {
		t.Errorf("CreatedAfter = %q, want %q", p.CreatedAfter, "2024-06-01")
	}
	if p.CreatedBefore != "2024-06-30" {
		t.Errorf("CreatedBefore = %q, want %q", p.CreatedBefore, "2024-06-30")
	}
}

// ---------------------------------------------------------------------------
// Correspondent tests
// ---------------------------------------------------------------------------

func TestCorrespondent_Populate(t *testing.T) {
	owner := 55

	c := Correspondent{
		ID:                 1,
		Slug:               "alice-smith",
		Name:               "Alice Smith",
		Match:              "alice",
		MatchingAlgorithm:  2,
		IsInsensitive:      true,
		DocumentCount:      42,
		LastCorrespondence: "2024-03-10T09:00:00Z",
		Owner:              &owner,
		UserCanChange:      true,
	}

	if c.ID != 1 {
		t.Errorf("ID = %d, want %d", c.ID, 1)
	}
	if c.Slug != "alice-smith" {
		t.Errorf("Slug = %q, want %q", c.Slug, "alice-smith")
	}
	if c.Name != "Alice Smith" {
		t.Errorf("Name = %q, want %q", c.Name, "Alice Smith")
	}
	if c.Match != "alice" {
		t.Errorf("Match = %q, want %q", c.Match, "alice")
	}
	if c.MatchingAlgorithm != 2 {
		t.Errorf("MatchingAlgorithm = %d, want %d", c.MatchingAlgorithm, 2)
	}
	if !c.IsInsensitive {
		t.Errorf("IsInsensitive = false, want true")
	}
	if c.DocumentCount != 42 {
		t.Errorf("DocumentCount = %d, want %d", c.DocumentCount, 42)
	}
	if c.LastCorrespondence != "2024-03-10T09:00:00Z" {
		t.Errorf("LastCorrespondence = %q, want %q", c.LastCorrespondence, "2024-03-10T09:00:00Z")
	}
	if c.Owner == nil || *c.Owner != 55 {
		t.Errorf("Owner = %v, want %d", c.Owner, 55)
	}
	if !c.UserCanChange {
		t.Errorf("UserCanChange = false, want true")
	}
}

func TestCorrespondent_ZeroValues(t *testing.T) {
	var c Correspondent

	if c.ID != 0 {
		t.Errorf("ID = %d, want 0", c.ID)
	}
	if c.Slug != "" {
		t.Errorf("Slug = %q, want empty", c.Slug)
	}
	if c.Name != "" {
		t.Errorf("Name = %q, want empty", c.Name)
	}
	if c.Match != "" {
		t.Errorf("Match = %q, want empty", c.Match)
	}
	if c.MatchingAlgorithm != 0 {
		t.Errorf("MatchingAlgorithm = %d, want 0", c.MatchingAlgorithm)
	}
	if c.IsInsensitive {
		t.Errorf("IsInsensitive = true, want false")
	}
	if c.DocumentCount != 0 {
		t.Errorf("DocumentCount = %d, want 0", c.DocumentCount)
	}
	if c.LastCorrespondence != "" {
		t.Errorf("LastCorrespondence = %q, want empty", c.LastCorrespondence)
	}
	if c.Owner != nil {
		t.Errorf("Owner = %v, want nil", c.Owner)
	}
	if c.UserCanChange {
		t.Errorf("UserCanChange = true, want false")
	}
}

// ---------------------------------------------------------------------------
// DocumentType tests
// ---------------------------------------------------------------------------

func TestDocumentType_Populate(t *testing.T) {
	owner := 10

	dt := DocumentType{
		ID:                2,
		Slug:              "invoice",
		Name:              "Invoice",
		Match:             "invoice",
		MatchingAlgorithm: 1,
		IsInsensitive:     false,
		DocumentCount:     100,
		Owner:             &owner,
		UserCanChange:     true,
	}

	if dt.ID != 2 {
		t.Errorf("ID = %d, want %d", dt.ID, 2)
	}
	if dt.Slug != "invoice" {
		t.Errorf("Slug = %q, want %q", dt.Slug, "invoice")
	}
	if dt.Name != "Invoice" {
		t.Errorf("Name = %q, want %q", dt.Name, "Invoice")
	}
	if dt.Match != "invoice" {
		t.Errorf("Match = %q, want %q", dt.Match, "invoice")
	}
	if dt.MatchingAlgorithm != 1 {
		t.Errorf("MatchingAlgorithm = %d, want %d", dt.MatchingAlgorithm, 1)
	}
	if dt.IsInsensitive {
		t.Errorf("IsInsensitive = true, want false")
	}
	if dt.DocumentCount != 100 {
		t.Errorf("DocumentCount = %d, want %d", dt.DocumentCount, 100)
	}
	if dt.Owner == nil || *dt.Owner != 10 {
		t.Errorf("Owner = %v, want %d", dt.Owner, 10)
	}
	if !dt.UserCanChange {
		t.Errorf("UserCanChange = false, want true")
	}
}

func TestDocumentType_ZeroValues(t *testing.T) {
	var dt DocumentType

	if dt.ID != 0 {
		t.Errorf("ID = %d, want 0", dt.ID)
	}
	if dt.Slug != "" {
		t.Errorf("Slug = %q, want empty", dt.Slug)
	}
	if dt.Name != "" {
		t.Errorf("Name = %q, want empty", dt.Name)
	}
	if dt.Match != "" {
		t.Errorf("Match = %q, want empty", dt.Match)
	}
	if dt.MatchingAlgorithm != 0 {
		t.Errorf("MatchingAlgorithm = %d, want 0", dt.MatchingAlgorithm)
	}
	if dt.IsInsensitive {
		t.Errorf("IsInsensitive = true, want false")
	}
	if dt.DocumentCount != 0 {
		t.Errorf("DocumentCount = %d, want 0", dt.DocumentCount)
	}
	if dt.Owner != nil {
		t.Errorf("Owner = %v, want nil", dt.Owner)
	}
	if dt.UserCanChange {
		t.Errorf("UserCanChange = true, want false")
	}
}

// ---------------------------------------------------------------------------
// Tag tests
// ---------------------------------------------------------------------------

func TestTag_Populate(t *testing.T) {
	owner := 33
	parent := 1

	tag := Tag{
		ID:                3,
		Slug:              "important",
		Name:              "Important",
		Color:             "#ff0000",
		TextColor:         "#ffffff",
		Match:             "important",
		MatchingAlgorithm: 0,
		IsInsensitive:     true,
		IsInboxTag:        false,
		DocumentCount:     75,
		Owner:             &owner,
		Parent:            &parent,
		UserCanChange:     true,
	}

	if tag.ID != 3 {
		t.Errorf("ID = %d, want %d", tag.ID, 3)
	}
	if tag.Slug != "important" {
		t.Errorf("Slug = %q, want %q", tag.Slug, "important")
	}
	if tag.Name != "Important" {
		t.Errorf("Name = %q, want %q", tag.Name, "Important")
	}
	if tag.Color != "#ff0000" {
		t.Errorf("Color = %q, want %q", tag.Color, "#ff0000")
	}
	if tag.TextColor != "#ffffff" {
		t.Errorf("TextColor = %q, want %q", tag.TextColor, "#ffffff")
	}
	if tag.Match != "important" {
		t.Errorf("Match = %q, want %q", tag.Match, "important")
	}
	if tag.MatchingAlgorithm != 0 {
		t.Errorf("MatchingAlgorithm = %d, want %d", tag.MatchingAlgorithm, 0)
	}
	if !tag.IsInsensitive {
		t.Errorf("IsInsensitive = false, want true")
	}
	if tag.IsInboxTag {
		t.Errorf("IsInboxTag = true, want false")
	}
	if tag.DocumentCount != 75 {
		t.Errorf("DocumentCount = %d, want %d", tag.DocumentCount, 75)
	}
	if tag.Owner == nil || *tag.Owner != 33 {
		t.Errorf("Owner = %v, want %d", tag.Owner, 33)
	}
	if tag.Parent == nil || *tag.Parent != 1 {
		t.Errorf("Parent = %v, want %d", tag.Parent, 1)
	}
	if !tag.UserCanChange {
		t.Errorf("UserCanChange = false, want true")
	}
}

func TestTag_ZeroValues(t *testing.T) {
	var tag Tag

	if tag.ID != 0 {
		t.Errorf("ID = %d, want 0", tag.ID)
	}
	if tag.Slug != "" {
		t.Errorf("Slug = %q, want empty", tag.Slug)
	}
	if tag.Name != "" {
		t.Errorf("Name = %q, want empty", tag.Name)
	}
	if tag.Color != "" {
		t.Errorf("Color = %q, want empty", tag.Color)
	}
	if tag.TextColor != "" {
		t.Errorf("TextColor = %q, want empty", tag.TextColor)
	}
	if tag.Match != "" {
		t.Errorf("Match = %q, want empty", tag.Match)
	}
	if tag.MatchingAlgorithm != 0 {
		t.Errorf("MatchingAlgorithm = %d, want 0", tag.MatchingAlgorithm)
	}
	if tag.IsInsensitive {
		t.Errorf("IsInsensitive = true, want false")
	}
	if tag.IsInboxTag {
		t.Errorf("IsInboxTag = true, want false")
	}
	if tag.DocumentCount != 0 {
		t.Errorf("DocumentCount = %d, want 0", tag.DocumentCount)
	}
	if tag.Owner != nil {
		t.Errorf("Owner = %v, want nil", tag.Owner)
	}
	if tag.Parent != nil {
		t.Errorf("Parent = %v, want nil", tag.Parent)
	}
	if tag.UserCanChange {
		t.Errorf("UserCanChange = true, want false")
	}
}
