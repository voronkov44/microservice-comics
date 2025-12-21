package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"log/slog"

	"yadro.com/course/favorites/core"
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
	return &DB{log: log, conn: db}, nil
}

func (db *DB) Ping(ctx context.Context) error {
	return db.conn.PingContext(ctx)
}

func (db *DB) Add(ctx context.Context, userID uint32, comicID int32) error {
	const q = `INSERT INTO favorites(user_id, comic_id) VALUES ($1, $2)`
	_, err := db.conn.ExecContext(ctx, q, userID, comicID)
	if err == nil {
		return nil
	}

	// в бд, pk(user_id и comics_id), по этому если комикс уже добавлен у пользователя,
	//бд вернет ошибку 23505, которую нужно поймать
	if isUniqueViolation(err) {
		return core.ErrAlreadyExists
	}
	return fmt.Errorf("insert favorite: %w", err)
}

func (db *DB) Delete(ctx context.Context, userID uint32, comicID int32) error {
	const q = `DELETE FROM favorites WHERE user_id=$1 AND comic_id=$2`
	res, err := db.conn.ExecContext(ctx, q, userID, comicID)
	if err != nil {
		return fmt.Errorf("delete favorite: %w", err)
	}

	// проверка на удаление несуществующей записи
	aff, _ := res.RowsAffected()
	if aff == 0 {
		return core.ErrNotFound
	}
	return nil
}

func (db *DB) List(ctx context.Context, userID uint32) ([]core.Favorite, error) {
	const q = `SELECT comic_id, created_at FROM favorites WHERE user_id=$1 ORDER BY created_at DESC`

	var rows []struct {
		ComicID   int32     `db:"comic_id"`
		CreatedAt time.Time `db:"created_at"`
	}

	if err := db.conn.SelectContext(ctx, &rows, q, userID); err != nil {
		return nil, fmt.Errorf("select favorites: %w", err)
	}

	out := make([]core.Favorite, 0, len(rows))
	for _, r := range rows {
		out = append(out, core.Favorite{
			ComicID:   r.ComicID,
			CreatedAt: r.CreatedAt,
		})
	}
	return out, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}
