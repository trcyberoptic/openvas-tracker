// internal/service/asset.go
package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/cyberoptic/vulntrack/internal/database/queries"
)

type AssetService struct {
	q *queries.Queries
}

func NewAssetService(pool *pgxpool.Pool) *AssetService {
	return &AssetService{q: queries.New(pool)}
}

func (s *AssetService) Upsert(ctx context.Context, params queries.UpsertAssetParams) (queries.Asset, error) {
	return s.q.UpsertAsset(ctx, params)
}

func (s *AssetService) Get(ctx context.Context, id, userID uuid.UUID) (queries.Asset, error) {
	return s.q.GetAsset(ctx, queries.GetAssetParams{ID: id, UserID: userID})
}

func (s *AssetService) List(ctx context.Context, userID uuid.UUID, limit, offset int32) ([]queries.Asset, error) {
	return s.q.ListAssets(ctx, queries.ListAssetsParams{UserID: userID, Limit: limit, Offset: offset})
}

func (s *AssetService) Delete(ctx context.Context, id, userID uuid.UUID) error {
	return s.q.DeleteAsset(ctx, queries.DeleteAssetParams{ID: id, UserID: userID})
}
