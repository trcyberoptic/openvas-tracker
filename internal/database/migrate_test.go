package database

import (
	"strings"
	"testing"
)

// A migration file that begins with comment lines followed by a single
// CREATE TABLE must still yield the CREATE TABLE statement. Regression: a
// comment-led chunk used to be dropped wholesale (HasPrefix "--"), so the
// table was silently never created on the AutoMigrate path while the
// migration was still recorded as applied.
func TestSplitStatements_CommentLedCreateTable(t *testing.T) {
	sql := "-- header comment one\n-- header comment two\nCREATE TABLE feed_status (\n    feed_type VARCHAR(32) NOT NULL PRIMARY KEY\n);"
	stmts := splitStatements(sql)
	if len(stmts) != 1 {
		t.Fatalf("expected 1 statement, got %d: %#v", len(stmts), stmts)
	}
	if !strings.HasPrefix(stmts[0], "CREATE TABLE feed_status") {
		t.Errorf("expected statement to start with CREATE TABLE feed_status, got: %q", stmts[0])
	}
}

func TestSplitStatements_MultipleStatementsAfterComment(t *testing.T) {
	sql := "-- header\nCREATE TABLE a (id INT);\nCREATE INDEX idx_a ON a(id);"
	stmts := splitStatements(sql)
	if len(stmts) != 2 {
		t.Fatalf("expected 2 statements, got %d: %#v", len(stmts), stmts)
	}
	if !strings.HasPrefix(stmts[0], "CREATE TABLE a") {
		t.Errorf("stmt[0] = %q", stmts[0])
	}
	if !strings.HasPrefix(stmts[1], "CREATE INDEX idx_a") {
		t.Errorf("stmt[1] = %q", stmts[1])
	}
}

// A chunk that is only comments/whitespace must be skipped (no empty statement).
func TestSplitStatements_PureCommentSkipped(t *testing.T) {
	if stmts := splitStatements("-- just a trailing comment\n"); len(stmts) != 0 {
		t.Fatalf("expected 0 statements, got %d: %#v", len(stmts), stmts)
	}
}
