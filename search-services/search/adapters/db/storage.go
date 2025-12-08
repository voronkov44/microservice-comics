package db

import (
	"context"
	"fmt"
	"log/slog"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"yadro.com/course/search/core"
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

func (db *DB) Find(ctx context.Context, tokens []string) ([]core.Comics, error) {
	// && - overlaps(есть ли пересечение двух множеств) $1 - наш tokens
	// выбрать все комиксы, у которых хотя бы один токен из запроса встречается
	// в title или в alt, или в words
	const q = `
		SELECT id, img_url, title, alt, words
		FROM comics
		WHERE title && $1 OR alt && $1 OR words && $1;
	`

	var rows []ComicsRow // используем промежуточную модель
	if err := db.conn.SelectContext(ctx, &rows, q, tokens); err != nil {
		db.log.Error("find comics failed", "tokens", tokens, "error", err)
		return nil, fmt.Errorf("find comics by tokens: %w", err)
	}

	// конвертируем обратно
	comics := make([]core.Comics, 0, len(rows))
	for _, r := range rows {
		comics = append(comics, core.Comics{
			ID:    r.ID,
			URL:   r.URL,
			Title: []string(r.Title),
			Alt:   []string(r.Alt),
			Words: []string(r.Words),
		})
	}

	return comics, nil
}

func (db *DB) All(ctx context.Context) ([]core.Comics, error) {
	const q = `
		SELECT id, img_url, title, alt, words
		FROM comics;
	`

	var rows []ComicsRow
	if err := db.conn.SelectContext(ctx, &rows, q); err != nil {
		db.log.Error("get all comics failed", "error", err)
		return nil, fmt.Errorf("get all comics: %w", err)
	}

	comics := make([]core.Comics, 0, len(rows))
	for _, r := range rows {
		comics = append(comics, core.Comics{
			ID:    r.ID,
			URL:   r.URL,
			Title: []string(r.Title),
			Alt:   []string(r.Alt),
			Words: []string(r.Words),
		})
	}

	return comics, nil
}
