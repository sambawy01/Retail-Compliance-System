// Package migrations provides embedded SQL migrations that run automatically
// on server startup. Migrations are tracked in a schema_migrations table.
package migrations

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"log/slog"
	"sort"
	"strings"
)

//go:embed versions/*.sql
var migrationFS embed.FS

// Run executes all pending migrations in order.
// Uses a simple version tracking table: schema_migrations(version TEXT PRIMARY KEY, applied_at TIMESTAMPTZ DEFAULT now()).
func Run(ctx context.Context, db *sql.DB) error {
	// Create tracking table
	_, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		version TEXT PRIMARY KEY,
		applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
	)`)
	if err != nil {
		return fmt.Errorf("migrations: create tracking table: %w", err)
	}

	// Read all migration files
	entries, err := fs.ReadDir(migrationFS, "versions")
	if err != nil {
		return fmt.Errorf("migrations: read embedded files: %w", err)
	}

	// Sort by filename
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		version := entry.Name()

		// Check if already applied
		var exists string
		err := db.QueryRowContext(ctx, "SELECT version FROM schema_migrations WHERE version = $1", version).Scan(&exists)
		if err == nil {
			continue // already applied
		}
		if err != sql.ErrNoRows {
			return fmt.Errorf("migrations: check %s: %w", version, err)
		}

		// Read file content
		content, err := fs.ReadFile(migrationFS, "versions/"+version)
		if err != nil {
			return fmt.Errorf("migrations: read %s: %w", version, err)
		}

		// Execute migration in a transaction
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("migrations: begin tx for %s: %w", version, err)
		}

		_, err = tx.ExecContext(ctx, string(content))
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("migrations: exec %s: %w", version, err)
		}

		_, err = tx.ExecContext(ctx, "INSERT INTO schema_migrations (version) VALUES ($1)", version)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("migrations: record %s: %w", version, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("migrations: commit %s: %w", version, err)
		}

		slog.Info("migration_applied", "version", version)
	}

	return nil
}