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

	// Bootstrap: if schema_migrations is empty but the database already has tables
	// (e.g. set up via docker-init.sql or manual migrate-up), mark pre-existing
	// migrations as applied. We detect this by checking for the users table
	// (created by migration 001). Only migrations whose CREATE TABLE target
	// already exists are marked — new migrations will run normally.
	if len(applied) == 0 {
		var exists int
		db.QueryRow(`SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = 'users'`).Scan(&exists)
		if exists > 0 {
			for _, file := range files {
				version := strings.TrimSuffix(file, ".up.sql")
				// Check if this migration creates a table that already exists
				content, _ := fs.ReadFile(migrationsFS, file)
				tableName := extractCreateTable(string(content))
				if tableName != "" {
					var tblExists int
					db.QueryRow(`SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = ?`, tableName).Scan(&tblExists)
					if tblExists == 0 {
						continue // table doesn't exist yet — this migration needs to run
					}
				}
				if _, err := db.Exec(`INSERT IGNORE INTO schema_migrations (version) VALUES (?)`, version); err != nil {
					return fmt.Errorf("bootstrap migration %s: %w", version, err)
				}
				applied[version] = true
			}
			log.Printf("migration: existing database detected, marked %d migrations as already applied", len(applied))
		}
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

// extractCreateTable returns the first table name from a CREATE TABLE statement, or "".
func extractCreateTable(sqlText string) string {
	upper := strings.ToUpper(sqlText)
	idx := strings.Index(upper, "CREATE TABLE ")
	if idx < 0 {
		return ""
	}
	rest := strings.TrimSpace(sqlText[idx+len("CREATE TABLE "):])
	// Skip optional "IF NOT EXISTS"
	if strings.HasPrefix(strings.ToUpper(rest), "IF NOT EXISTS ") {
		rest = strings.TrimSpace(rest[len("IF NOT EXISTS "):])
	}
	// First token is the table name
	name := strings.FieldsFunc(rest, func(r rune) bool {
		return r == ' ' || r == '(' || r == '\n' || r == '\r' || r == '\t'
	})[0]
	return strings.Trim(name, "`")
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
