package queries

import (
	"context"
	"time"
)

type FeedStatus struct {
	FeedType    string    `json:"feed_type"`
	FeedName    string    `json:"feed_name"`
	Version     string    `json:"version"`
	FirstSeen   time.Time `json:"first_seen"`
	LastSeen    time.Time `json:"last_seen"`
	LastChanged time.Time `json:"last_changed"`
}

type UpsertFeedStatusParams struct {
	FeedType string
	FeedName string
	Version  string
}

// UpsertFeedStatus records the current version of a feed. last_changed is only
// bumped when the version value actually changes; last_seen is bumped every call.
// NB: MariaDB supports VALUES() in ON DUPLICATE KEY UPDATE; last_changed is
// assigned BEFORE version so it still sees the previous version value.
func (q *Queries) UpsertFeedStatus(ctx context.Context, arg UpsertFeedStatusParams) error {
	_, err := q.db.ExecContext(ctx,
		`INSERT INTO feed_status (feed_type, feed_name, version, first_seen, last_seen, last_changed)
		 VALUES (?, ?, ?, NOW(), NOW(), NOW())
		 ON DUPLICATE KEY UPDATE
		   last_changed = IF(feed_status.version <> VALUES(version), NOW(), feed_status.last_changed),
		   feed_name    = VALUES(feed_name),
		   last_seen    = NOW(),
		   version      = VALUES(version)`,
		arg.FeedType, arg.FeedName, arg.Version)
	return err
}

// ListFeedStatus returns all known feeds ordered by feed_type.
func (q *Queries) ListFeedStatus(ctx context.Context) ([]FeedStatus, error) {
	rows, err := q.db.QueryContext(ctx,
		`SELECT feed_type, feed_name, version, first_seen, last_seen, last_changed
		 FROM feed_status ORDER BY feed_type`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []FeedStatus
	for rows.Next() {
		var f FeedStatus
		if err := rows.Scan(&f.FeedType, &f.FeedName, &f.Version, &f.FirstSeen, &f.LastSeen, &f.LastChanged); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}
