package db

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Migration struct {
	Name string
	Up   func() error
}

var migrations []Migration
var appliedMigrations map[string]bool

func RegisterMigration(name string, up func() error) {
	migrations = append(migrations, Migration{
		Name: name,
		Up:   up,
	})
}

func RunMigrations() error {
	if err := ensureMigrationsTable(); err != nil {
		return fmt.Errorf("failed to ensure migrations table: %w", err)
	}

	appliedMigrations = make(map[string]bool)
	if err := loadAppliedMigrations(); err != nil {
		return fmt.Errorf("failed to load applied migrations: %w", err)
	}

	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Name < migrations[j].Name
	})

	for _, migration := range migrations {
		if appliedMigrations[migration.Name] {
			continue
		}

		log.Printf("Running migration: %s\n", migration.Name)
		if err := migration.Up(); err != nil {
			return fmt.Errorf("migration %s failed: %w", migration.Name, err)
		}

		if err := recordMigration(migration.Name); err != nil {
			return fmt.Errorf("failed to record migration %s: %w", migration.Name, err)
		}

		log.Printf("Migration completed: %s\n", migration.Name)
	}

	return nil
}

func ensureMigrationsTable() error {
	return DB.Exec(`
		CREATE TABLE IF NOT EXISTS migrations (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255) UNIQUE NOT NULL,
			applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`).Error
}

func loadAppliedMigrations() error {
	var names []string
	if err := DB.Raw("SELECT name FROM migrations").Scan(&names).Error; err != nil {
		return err
	}

	for _, name := range names {
		appliedMigrations[name] = true
	}
	return nil
}

func recordMigration(name string) error {
	return DB.Exec("INSERT INTO migrations (name) VALUES (?)", name).Error
}

func LoadMigrationsFromDir(dir string) error {
	files, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("failed to read migrations directory: %w", err)
	}

	var fileNames []string
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".sql") {
			fileNames = append(fileNames, file.Name())
		}
	}

	sort.Strings(fileNames)

	for _, fileName := range fileNames {
		filePath := filepath.Join(dir, fileName)
		content, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("failed to read migration file %s: %w", fileName, err)
		}

		migrationName := strings.TrimSuffix(fileName, ".sql")
		sql := string(content)

		RegisterMigration(migrationName, func() error {
			return DB.Exec(sql).Error
		})
	}

	return nil
}
