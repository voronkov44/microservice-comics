package db

import (
	_ "embed"
	"fmt"
)

//go:embed migrations/000001_create_favorites.up.sql
var favoritesMigrationUp string

// Migrate применяет миграцию для auth-сервиса
func (db *DB) Migrate() error {
	db.log.Debug("running favorites migrations")

	if _, err := db.conn.Exec(favoritesMigrationUp); err != nil {
		return fmt.Errorf("apply favorites migration: %w", err)
	}

	db.log.Debug("favorites migrations finished")
	return nil
}
