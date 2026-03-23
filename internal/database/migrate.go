package database

import (
	"database/sql"
	"fmt"
	"io/fs"
	"log"
	"sort"
	"strings"
)

// AutoMigrate applies all pending .up.sql migrations from the given filesystem.
// Migration files must be named NNN_description.up.sql (e.g. 001_create_users.up.sql).
// Tracks applied migrations in a schema_migrations table.
func AutoMigrate(db *sql.DB, migrationsFS fs.FS) error {
	// Ensure tracking table exists
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		version VARCHAR(255) PRIMARY KEY,
		applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`); err != nil {
		return fmt.Errorf("create schema_migrations table: %w", err)
	}

	// List all .up.sql files
	files, err := fs.Glob(migrationsFS, "*.up.sql")
	if err != nil {
		return fmt.Errorf("glob migrations: %w", err)
	}
	sort.Strings(files)

	// Get already-applied versions
	applied := make(map[string]bool)
	rows, err := db.Query(`SELECT version FROM schema_migrations`)
	if err != nil {
		return fmt.Errorf("query schema_migrations: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return err
		}
		applied[v] = true
	}

	// Apply pending migrations in order
	for _, file := range files {
		version := strings.TrimSuffix(file, ".up.sql")
		if applied[version] {
			continue
		}

		content, err := fs.ReadFile(migrationsFS, file)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", file, err)
		}

		// Execute each statement separated by semicolons
		for _, stmt := range splitStatements(string(content)) {
			if stmt == "" {
				continue
			}
			if _, err := db.Exec(stmt); err != nil {
				return fmt.Errorf("migration %s failed: %w\nstatement: %s", file, err, stmt)
			}
		}

		if _, err := db.Exec(`INSERT INTO schema_migrations (version) VALUES (?)`, version); err != nil {
			return fmt.Errorf("record migration %s: %w", file, err)
		}
		log.Printf("migration: applied %s", version)
	}

	return nil
}

// splitStatements splits SQL text on semicolons, trimming whitespace and
// skipping empty results. It handles the simple case where statements
// are separated by ";\n" which covers all our migration files.
func splitStatements(sql string) []string {
	parts := strings.Split(sql, ";")
	var out []string
	for _, p := range parts {
		s := strings.TrimSpace(p)
		if s != "" && !strings.HasPrefix(s, "--") {
			out = append(out, s)
		}
	}
	return out
}
