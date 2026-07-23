package testkit

import (
	"strings"
	"testing"

	"termorize/src/data/db"
)

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
