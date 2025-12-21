package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
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
	return &DB{log: log, conn: db}, nil
}

func (db *DB) Ping(ctx context.Context) error {
	return db.conn.PingContext(ctx)
}

// CreateComicshubUser регистрирует пользователя через ComicsHub (email/password)
//
// Логика:
// 1) создаём запись в users (получаем user_id)
// 2) создаём запись в user_comicshub с email и password_hash, привязав её к user_id
// Всё делаем в транзакции, чтобы не получить "users есть, а аккаунта нет"
func (db *DB) CreateComicshubUser(ctx context.Context, profile core.ComicsHubProfile) (core.User, error) {
	tx, err := db.conn.BeginTxx(ctx, nil)
	if err != nil {
		return core.User{}, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	const insUser = `INSERT INTO users DEFAULT VALUES RETURNING id, created_at;`
	var u core.User
	if err := tx.QueryRowContext(ctx, insUser).Scan(&u.ID, &u.CreatedAt); err != nil {
		return core.User{}, fmt.Errorf("insert users: %w", err)
	}

	const insAcc = `
		INSERT INTO user_comicshub (user_id, email, password_hash)
		VALUES ($1, $2, $3);
	`
	if _, err := tx.ExecContext(ctx, insAcc, u.ID, profile.Email, profile.PasswordHash); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return core.User{}, core.ErrUserAlreadyExists
		}
		return core.User{}, fmt.Errorf("insert user_comicshub: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return core.User{}, fmt.Errorf("commit tx: %w", err)
	}
	return u, nil
}

// GetComicshubByEmail ищет аккаунт ComicsHub по email и возвращает пользователя и hash:
// Мы делаем JOIN user_comicshub  users, потому что email хранится в user_comicshub,
// а user_id/created_at - в users
func (db *DB) GetComicshubByEmail(ctx context.Context, email string) (core.User, string, error) {
	const q = `
		SELECT u.id, u.created_at, c.password_hash
		FROM user_comicshub c
		JOIN users u ON u.id = c.user_id
		WHERE c.email = $1;
	`

	var u core.User
	var hash string

	err := db.conn.QueryRowContext(ctx, q, email).Scan(&u.ID, &u.CreatedAt, &hash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return core.User{}, "", core.ErrNotFound
		}
		return core.User{}, "", fmt.Errorf("select user_comicshub user by email: %w", err)
	}
	return u, hash, nil
}

// UpsertTelegramUser создаёт и обновляет профиль пользователя telegram
//
// Поведение:
//
//  1. Если tg_id уже есть в user_telegram:
//     обновляем username/first_name/last_name и updated_at
//     возвращаем связанного пользователя из таблицы users
//
//  2. Если tg_id ещё нет:
//     создаём новую строку в users (получаем id)
//     вставляем профиль в user_telegram
//     если в момент вставки кто-то параллельно успел вставить тот же tg_id (гонка),
//     то переключаемся на сценарий (1): обновляем существующий профиль и возвращаем user
func (db *DB) UpsertTelegramUser(ctx context.Context, tg core.TelegramProfile) (core.User, error) {
	tx, err := db.conn.BeginTxx(ctx, nil)
	if err != nil {
		return core.User{}, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// helper: обновить профиль tg по tg_id
	updateProfile := func() error {
		const upd = `
			UPDATE user_telegram
			SET username=$1, first_name=$2, last_name=$3, updated_at=now()
			WHERE tg_id=$4;
		`
		_, err := tx.ExecContext(ctx, upd, tg.Username, tg.FirstName, tg.LastName, tg.TgID)
		if err != nil {
			return fmt.Errorf("update user_telegram: %w", err)
		}
		return nil
	}

	// helper: загрузить пользователя из users по id
	loadUser := func(id int64) (core.User, error) {
		const getU = `SELECT id, created_at FROM users WHERE id=$1;`
		var u core.User
		if err := tx.QueryRowContext(ctx, getU, id).Scan(&u.ID, &u.CreatedAt); err != nil {
			return core.User{}, fmt.Errorf("load users: %w", err)
		}
		return u, nil
	}

	// 1) Пытаемся найти привязку по tg_id (если есть - обновляем и выходим)
	const sel = `SELECT user_id FROM user_telegram WHERE tg_id = $1;`
	var userID int64
	err = tx.QueryRowContext(ctx, sel, tg.TgID).Scan(&userID)
	if err == nil {
		if err := updateProfile(); err != nil {
			return core.User{}, err
		}

		u, err := loadUser(userID)
		if err != nil {
			return core.User{}, err
		}

		if err := tx.Commit(); err != nil {
			return core.User{}, fmt.Errorf("commit tx: %w", err)
		}
		return u, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return core.User{}, fmt.Errorf("select user_telegram: %w", err)
	}

	// 2) tg_id не найден - создаём нового пользователя в users и user_telegram
	const insUser = `INSERT INTO users DEFAULT VALUES RETURNING id, created_at;`
	var u core.User
	if err := tx.QueryRowContext(ctx, insUser).Scan(&u.ID, &u.CreatedAt); err != nil {
		return core.User{}, fmt.Errorf("insert users: %w", err)
	}

	const insTG = `
		INSERT INTO user_telegram (user_id, tg_id, username, first_name, last_name)
		VALUES ($1, $2, $3, $4, $5);
	`
	if _, err := tx.ExecContext(ctx, insTG, u.ID, tg.TgID, tg.Username, tg.FirstName, tg.LastName); err != nil {
		// Возможна гонка - другой запрос успел вставить тот же tg_id,
		//по этому дефаем это переключением на решение, где пользователь уже создан
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			// Переключаемся на решение, где пользователь уже создан - грузим user_id, обновляем профиль, возвращаем user
			if err := tx.QueryRowContext(ctx, sel, tg.TgID).Scan(&userID); err != nil {
				return core.User{}, fmt.Errorf("reload user_telegram after conflict: %w", err)
			}
			if err := updateProfile(); err != nil {
				return core.User{}, err
			}
			u2, err := loadUser(userID)
			if err != nil {
				return core.User{}, err
			}
			if err := tx.Commit(); err != nil {
				return core.User{}, fmt.Errorf("commit tx: %w", err)
			}
			return u2, nil
		}

		return core.User{}, fmt.Errorf("insert user_telegram: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return core.User{}, fmt.Errorf("commit tx: %w", err)
	}
	return u, nil
}
