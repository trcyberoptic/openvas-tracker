package queries

import "context"

// GetAppSettings returns every stored runtime setting (migration 024).
func (q *Queries) GetAppSettings(ctx context.Context) (map[string]string, error) {
	rows, err := q.db.QueryContext(ctx, `SELECT name, value FROM app_settings`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]string{}
	for rows.Next() {
		var name, value string
		if err := rows.Scan(&name, &value); err != nil {
			return nil, err
		}
		out[name] = value
	}
	return out, rows.Err()
}

func (q *Queries) SetAppSetting(ctx context.Context, name, value string) error {
	_, err := q.db.ExecContext(ctx,
		`INSERT INTO app_settings (name, value) VALUES (?, ?) ON DUPLICATE KEY UPDATE value = VALUES(value)`,
		name, value)
	return err
}

// GetUserEmailNotifications reports whether the user wants assignment mails (migration 025).
func (q *Queries) GetUserEmailNotifications(ctx context.Context, userID string) (bool, error) {
	var on bool
	err := q.db.QueryRowContext(ctx, `SELECT email_notifications FROM users WHERE id = ?`, userID).Scan(&on)
	return on, err
}

func (q *Queries) SetUserEmailNotifications(ctx context.Context, userID string, on bool) error {
	_, err := q.db.ExecContext(ctx, `UPDATE users SET email_notifications = ?, updated_at = NOW() WHERE id = ?`, on, userID)
	return err
}
