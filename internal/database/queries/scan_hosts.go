package queries

import "context"

// InsertScanHost records that a host was present in a scan's results.
// Uses INSERT IGNORE so duplicates within the same scan are silently skipped.
func (q *Queries) InsertScanHost(ctx context.Context, scanID, host string) error {
	_, err := q.db.ExecContext(ctx,
		`INSERT IGNORE INTO scan_hosts (scan_id, host) VALUES (?, ?)`,
		scanID, host)
	return err
}
