package application

import (
	"context"
	"fmt"

	"github.com/teran/mcp-paperless-ngx/domain"
)

// DocumentService implements document-related use cases.
type DocumentService struct {
	docs domain.DocumentRepository
}

// NewDocumentService creates a new DocumentService.
func NewDocumentService(docs domain.DocumentRepository) *DocumentService {
	return &DocumentService{docs: docs}
}

// Search searches documents with optional filters.
func (s *DocumentService) Search(ctx context.Context, params domain.SearchDocumentsParams) (*domain.PaginatedResult[domain.Document], error) {
	result, err := s.docs.Search(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("search documents: %w", err)
	}
	return result, nil
}

// GetByID retrieves a single document by its ID.
func (s *DocumentService) GetByID(ctx context.Context, id int) (*domain.Document, error) {
	doc, err := s.docs.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get document: %w", err)
	}
	return doc, nil
}

// GetByCorrespondent retrieves documents for a given correspondent.
func (s *DocumentService) GetByCorrespondent(ctx context.Context, correspondentID, page, pageSize int) (*domain.PaginatedResult[domain.Document], error) {
	params := domain.SearchDocumentsParams{
		Query:           "",
		CorrespondentID: correspondentID,
		TagIDs:          nil,
		CreatedAfter:    "",
		CreatedBefore:   "",
		Page:            page,
		PageSize:        pageSize,
	}
	result, err := s.docs.Search(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("get documents by correspondent: %w", err)
	}
	return result, nil
}

// GetByTag retrieves documents for a given tag.
func (s *DocumentService) GetByTag(ctx context.Context, tagID, page, pageSize int) (*domain.PaginatedResult[domain.Document], error) {
	params := domain.SearchDocumentsParams{
		Query:           "",
		CorrespondentID: 0,
		TagIDs:          []int{tagID},
		CreatedAfter:    "",
		CreatedBefore:   "",
		Page:            page,
		PageSize:        pageSize,
	}
	result, err := s.docs.Search(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("get documents by tag: %w", err)
	}
	return result, nil
}

// FulltextSearch performs a full-text search across all documents.
func (s *DocumentService) FulltextSearch(ctx context.Context, query string, page, pageSize int) (*domain.PaginatedResult[domain.Document], error) {
	params := domain.SearchDocumentsParams{
		Query:           query,
		CorrespondentID: 0,
		TagIDs:          nil,
		CreatedAfter:    "",
		CreatedBefore:   "",
		Page:            page,
		PageSize:        pageSize,
	}
	result, err := s.docs.Search(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("fulltext search: %w", err)
	}
	return result, nil
}

// CorrespondentService implements correspondent-related use cases.
type CorrespondentService struct {
	repo domain.CorrespondentRepository
}

// NewCorrespondentService creates a new CorrespondentService.
func NewCorrespondentService(repo domain.CorrespondentRepository) *CorrespondentService {
	return &CorrespondentService{repo: repo}
}

// Search searches correspondents by name.
func (s *CorrespondentService) Search(ctx context.Context, query string, page, pageSize int) (*domain.PaginatedResult[domain.Correspondent], error) {
	result, err := s.repo.Search(ctx, query, page, pageSize)
	if err != nil {
		return nil, fmt.Errorf("search correspondents: %w", err)
	}
	return result, nil
}

// GetByID retrieves a single correspondent by its ID.
func (s *CorrespondentService) GetByID(ctx context.Context, id int) (*domain.Correspondent, error) {
	result, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get correspondent: %w", err)
	}
	return result, nil
}

// DocumentTypeService implements document-type-related use cases.
type DocumentTypeService struct {
	repo domain.DocumentTypeRepository
}

// NewDocumentTypeService creates a new DocumentTypeService.
func NewDocumentTypeService(repo domain.DocumentTypeRepository) *DocumentTypeService {
	return &DocumentTypeService{repo: repo}
}

// GetByID retrieves a single document type by its ID.
func (s *DocumentTypeService) GetByID(ctx context.Context, id int) (*domain.DocumentType, error) {
	result, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get document type: %w", err)
	}
	return result, nil
}

// TagService implements tag-related use cases.
type TagService struct {
	repo domain.TagRepository
}

// NewTagService creates a new TagService.
func NewTagService(repo domain.TagRepository) *TagService {
	return &TagService{repo: repo}
}

// List lists all tags, optionally filtered by name.
func (s *TagService) List(ctx context.Context, query string, page, pageSize int) (*domain.PaginatedResult[domain.Tag], error) {
	result, err := s.repo.List(ctx, query, page, pageSize)
	if err != nil {
		return nil, fmt.Errorf("list tags: %w", err)
	}
	return result, nil
}
