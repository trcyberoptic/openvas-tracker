// internal/service/search.go
package service

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/cyberoptic/vulntrack/internal/database/queries"
)

type SearchService struct {
	q *queries.Queries
}

func NewSearchService(pool *pgxpool.Pool) *SearchService {
	return &SearchService{q: queries.New(pool)}
}

func (s *SearchService) Search(ctx context.Context, query string, limit int32) ([]queries.SearchAllRow, error) {
	return s.q.SearchAll(ctx, queries.SearchAllParams{Column1: query, Limit: limit})
}
