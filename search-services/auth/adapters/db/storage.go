package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"log/slog"
	"yadro.com/course/auth/core"
)

type DB struct {
	log  *slog.Logger
	conn *sqlx.DB
}

func New(log *slog.Logger, address string) (*DB, error) {

	db, err := sqlx.Connect("pgx", address)
	if err != nil {
		log.Error("connection problem", "address", address, "error", err)
		return nil, err
	}

	return &DB{
		log:  log,
		conn: db,
	}, nil
}

func (db *DB) Ping(ctx context.Context) error {
	return db.conn.PingContext(ctx)
}

// CreateUser создаёт пользователя в БД и возвращает созданную запись
func (db *DB) CreateUser(ctx context.Context, user core.Users) (core.Users, error) {
	const query = `
		INSERT INTO users (email, password_hash, created_at)
		VALUES ($1, $2, $3)
		RETURNING id, email, password_hash, created_at;
	`

	var u core.Users
	err := db.conn.QueryRowContext(ctx, query,
		user.Email,
		user.PasswordHash,
		user.CreatedAt,
	).Scan(
		&u.ID,
		&u.Email,
		&u.PasswordHash,
		&u.CreatedAt,
	)
	if err != nil {
		return core.Users{}, fmt.Errorf("insert user: %w", err)
	}

	return u, nil
}

// GetUserByEmail возвращает пользователя по email
func (db *DB) GetUserByEmail(ctx context.Context, email string) (core.Users, error) {
	const query = `
		SELECT id, email, password_hash, created_at
		FROM users
		WHERE email = $1;
	`

	var u core.Users
	err := db.conn.QueryRowContext(ctx, query, email).Scan(
		&u.ID,
		&u.Email,
		&u.PasswordHash,
		&u.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return core.Users{}, fmt.Errorf("user not found")
		}
		return core.Users{}, fmt.Errorf("select user by email: %w", err)
	}

	return u, nil
}
