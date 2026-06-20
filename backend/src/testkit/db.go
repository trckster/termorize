package testkit

import (
	"strings"
	"testing"

	"termorize/src/data/db"
)

// Truncate empties every application table for per-test isolation. It discovers
// table names dynamically from information_schema (so it stays correct as the
// schema evolves) and runs a single TRUNCATE ... RESTART IDENTITY CASCADE,
// excluding the internal "migrations" table.
//
// Call it at the top of each test that touches the database:
//
//	func TestX(t *testing.T) {
//	    testkit.Truncate(t)
//	    ...
//	}
func Truncate(t *testing.T) {
	t.Helper()

	var tables []string
	err := db.DB.Raw(`
		SELECT tablename
		FROM pg_tables
		WHERE schemaname = 'public'
		  AND tablename <> 'migrations'
	`).Scan(&tables).Error
	if err != nil {
		t.Fatalf("testkit.Truncate: failed to list tables: %v", err)
	}

	if len(tables) == 0 {
		return
	}

	quoted := make([]string, len(tables))
	for i, name := range tables {
		quoted[i] = `"` + name + `"`
	}

	stmt := "TRUNCATE TABLE " + strings.Join(quoted, ", ") + " RESTART IDENTITY CASCADE"
	if err := db.DB.Exec(stmt).Error; err != nil {
		t.Fatalf("testkit.Truncate: failed to truncate tables: %v", err)
	}
}
