package db

import (
	"context"
	"fmt"
	"log/slog"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"yadro.com/course/update/core"
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

// Add - идемпотентный upsert по id
// comics.Words - передаем напрямую, sqlx сам конвертирует []string в text[]
func (db *DB) Add(ctx context.Context, comics core.Comics) error {
	// не пускаем нил в бд
	title := comics.Title
	if title == nil {
		title = []string{}
	}

	alt := comics.Alt
	if alt == nil {
		alt = []string{}
	}

	words := comics.Words
	if words == nil {
		words = []string{}
	}

	_, err := db.conn.ExecContext(ctx, `
		INSERT INTO comics (id, img_url, title, alt, words)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (id) DO UPDATE SET
			img_url   = EXCLUDED.img_url,
		    title     = EXCLUDED.title,
		    alt       = EXCLUDED.alt,
			words     = EXCLUDED.words,
			fetched_at= NOW()
	`, comics.ID, comics.URL, title, alt, words)
	if err != nil {
		return fmt.Errorf("upsert comics: %w", err)
	}
	return nil
}

// Stats - возвращает агрегированную статистику по таблице
// words_total = суммарное количество слов во всех комиксах
// words_unique = количество уникальных нормализованных слов среди всех комиксов
// comics_fetched = общее количество сохраненных комиксов
func (db *DB) Stats(ctx context.Context) (core.DBStats, error) {
	var st core.DBStats

	// Суммируем cardinality(words) по всем строкам
	// Если таблица пуста, sum вернет null - мы заменим на 0
	if err := db.conn.GetContext(ctx, &st.WordsTotal, `
		SELECT COALESCE(SUM(COALESCE(cardinality(words),0)),0) 
		FROM comics
	`); err != nil {
		return core.DBStats{}, err
	}

	// Разворачиваем все массивы words в отдельные строки через unnest,
	// затем считаем количество уникальных значений
	if err := db.conn.GetContext(ctx, &st.WordsUnique, `
		SELECT COALESCE(COUNT(DISTINCT w),0)
		FROM comics, UNNEST(words) AS w
	`); err != nil {
		// если таблица пустая, unnest ничего не вернёт - count вернёт 0
		return core.DBStats{}, err
	}

	// Количество записей (сколько комиксов скачано)
	// count(*) всегда возвращает целое число, даже для пустой таблицы, coalesce не нужен
	if err := db.conn.GetContext(ctx, &st.ComicsFetched, `
		SELECT COUNT(*) FROM comics
	`); err != nil {
		return core.DBStats{}, err
	}

	return st, nil
}

// IDs - слайс уже загруженных id комиксов для идемпотентности
func (db *DB) IDs(ctx context.Context) ([]int, error) {
	var out []int
	if err := db.conn.SelectContext(ctx, &out, `SELECT id FROM comics`); err != nil {
		return nil, fmt.Errorf("get ids: %w", err)
	}
	return out, nil
}

// Drop - каскадно удаляем все строки из таблицы и сбрасываем счетчик для чистоты
func (db *DB) Drop(ctx context.Context) error {
	_, err := db.conn.ExecContext(ctx, `TRUNCATE TABLE comics RESTART IDENTITY CASCADE`)
	return err
}
