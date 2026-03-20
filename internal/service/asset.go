// internal/service/asset.go
package service

import (
	"context"
	"database/sql"

	"github.com/cyberoptic/openvas-tracker/internal/database/queries"
)

type AssetService struct {
	q *queries.Queries
}

func NewAssetService(db *sql.DB) *AssetService {
	return &AssetService{q: queries.New(db)}
}

func (s *AssetService) Upsert(ctx context.Context, params queries.UpsertAssetParams) (queries.Asset, error) {
	return s.q.UpsertAsset(ctx, params)
}

func (s *AssetService) Get(ctx context.Context, id, userID string) (queries.Asset, error) {
	return s.q.GetAsset(ctx, queries.GetAssetParams{ID: id, UserID: userID})
}

func (s *AssetService) List(ctx context.Context, userID string, limit, offset int32) ([]queries.Asset, error) {
	return s.q.ListAssets(ctx, queries.ListAssetsParams{UserID: userID, Limit: limit, Offset: offset})
}

func (s *AssetService) Delete(ctx context.Context, id, userID string) error {
	return s.q.DeleteAsset(ctx, queries.DeleteAssetParams{ID: id, UserID: userID})
}
