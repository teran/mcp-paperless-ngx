package domain

import "context"

// DocumentRepository defines the interface for document persistence.
type DocumentRepository interface {
	Search(ctx context.Context, params SearchDocumentsParams) (*PaginatedResult[Document], error)
	GetByID(ctx context.Context, id int) (*Document, error)
}

// CorrespondentRepository defines the interface for correspondent persistence.
type CorrespondentRepository interface {
	Search(ctx context.Context, query string, page, pageSize int) (*PaginatedResult[Correspondent], error)
	GetByID(ctx context.Context, id int) (*Correspondent, error)
}

// TagRepository defines the interface for tag persistence.
type TagRepository interface {
	List(ctx context.Context, query string, page, pageSize int) (*PaginatedResult[Tag], error)
}
