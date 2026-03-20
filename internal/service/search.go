// internal/service/search.go
package service

import (
	"context"
	"database/sql"

	"github.com/cyberoptic/openvas-tracker/internal/database/queries"
)

type SearchService struct {
	q *queries.Queries
}

func NewSearchService(db *sql.DB) *SearchService {
	return &SearchService{q: queries.New(db)}
}

func (s *SearchService) Search(ctx context.Context, query string, limit int32) ([]queries.SearchAllRow, error) {
	return s.q.SearchAll(ctx, queries.SearchAllParams{Column1: query, Limit: limit})
}
