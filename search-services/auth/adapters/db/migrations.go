package db

import (
	_ "embed"
	"fmt"
)

//go:embed migrations/000001_create_users.up.sql
var usersMigrationUp string

// Migrate применяет миграцию для auth-сервиса
func (db *DB) Migrate() error {
	db.log.Debug("running auth migrations")

	if _, err := db.conn.Exec(usersMigrationUp); err != nil {
		return fmt.Errorf("apply users migration: %w", err)
	}

	db.log.Debug("auth migrations finished")
	return nil
}
