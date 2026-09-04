// Package migrate runs a minimal schema migration: a version table plus a list
// of SQL statements executed exactly once.
// AutoMigrate only creates missing tables; column/index changes on existing
// databases are registered here, so they stay traceable and never run twice.
package migrate

import (
	"context"
	"fmt"

	"gorm.io/gorm"
)

// Migration is one schema change; Version is globally unique and never reused
// (convention: YYYYMMDDNN_description).
type Migration struct {
	Version string
	SQL     []string
}

const (
	table       = "schema_migrations"
	createTable = `CREATE TABLE IF NOT EXISTS ` + table + ` (
		version varchar(64) PRIMARY KEY,
		applied_at timestamptz NOT NULL DEFAULT now()
	)`
	insertVersion = `INSERT INTO ` + table + ` (version) VALUES (?)`
)

// Run applies the pending migrations in order, one transaction each, stopping
// at the first failure.
func Run(ctx context.Context, db *gorm.DB, migrations []Migration) error {
	db = db.WithContext(ctx)
	if err := db.Exec(createTable).Error; err != nil {
		return fmt.Errorf("create %s: %w", table, err)
	}
	// Applied versions are read once instead of querying per migration.
	var versions []string
	if err := db.Table(table).Pluck("version", &versions).Error; err != nil {
		return fmt.Errorf("list applied %s: %w", table, err)
	}
	applied := make(map[string]struct{}, len(versions))
	for _, v := range versions {
		applied[v] = struct{}{}
	}
	for _, m := range migrations {
		if _, ok := applied[m.Version]; ok {
			continue
		}
		if err := db.Transaction(func(tx *gorm.DB) error {
			for _, s := range m.SQL {
				if err := tx.Exec(s).Error; err != nil {
					return err
				}
			}
			return tx.Exec(insertVersion, m.Version).Error
		}); err != nil {
			return fmt.Errorf("migration %s: %w", m.Version, err)
		}
	}
	return nil
}
