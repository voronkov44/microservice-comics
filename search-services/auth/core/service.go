package core

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"regexp"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var emailRe = regexp.MustCompile(`^[A-Za-z0-9._%+\-]+@[A-Za-z0-9.\-]+\.[A-Za-z]{2,}$`)

func validateEmail(email string) error {
	if !emailRe.MatchString(email) {
		return ErrInvalidEmail
	}
	return nil
}

// JWTClaims - набор данных, который мы кладём в токен
type JWTClaims struct {
	UserID uint32 `json:"user_id"`
	jwt.RegisteredClaims
}

type Service struct {
	log       *slog.Logger
	db        DB
	jwtSecret []byte
	tokenTTL  time.Duration
}

func NewService(log *slog.Logger, db DB, jwtSecret string, tokenTTL time.Duration) (*Service, error) {
	if jwtSecret == "" {
		return nil, fmt.Errorf("empty jwt secret")
	}
	if tokenTTL <= 0 {
		return nil, fmt.Errorf("token ttl must be positive")
	}
	return &Service{
		log:       log,
		db:        db,
		jwtSecret: []byte(jwtSecret),
		tokenTTL:  tokenTTL,
	}, nil
}

// Register - регистрация по email/password
func (s *Service) Register(ctx context.Context, email, password string) (string, error) {
	// Проверка, что данные не пустые
	if email == "" || password == "" {
		return "", fmt.Errorf("email and password must not be empty")
	}

	// Валидация email
	if err := validateEmail(email); err != nil {
		return "", ErrInvalidEmail
	}

	// Хэшируем пароль
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		s.log.Error("failed to hash password", "error", err)
		return "", fmt.Errorf("failed to hash password: %w", err)
	}

	u, err := s.db.CreateComicshubUser(ctx, ComicsHubProfile{
		Email:        email,
		PasswordHash: string(hash),
	})
	if err != nil {
		if errors.Is(err, ErrUserAlreadyExists) {
			return "", ErrUserAlreadyExists
		}
		s.log.Error("failed to create comicshub user", "email", email, "error", err)
		return "", fmt.Errorf("failed to create user: %w", err)
	}

	// Генерируем JWT для только что созданного пользователя
	token, err := s.generateToken(u.ID)
	if err != nil {
		s.log.Error("failed to generate jwt on register", "email", email, "error", err)
		return "", fmt.Errorf("failed to generate token: %w", err)
	}

	return token, nil
}

// Login проверяет логин и пароль и возвращает jwt
func (s *Service) Login(ctx context.Context, email, password string) (string, error) {
	// Проверка, что данные не пустые
	if email == "" || password == "" {
		return "", ErrInvalidCredentials
	}

	// Валидация email
	if err := validateEmail(email); err != nil {
		return "", ErrInvalidEmail
	}

	u, hash, err := s.db.GetComicshubByEmail(ctx, email)
	if err != nil {
		s.log.Warn("user not found or db error on login", "email", email, "error", err)
		return "", ErrInvalidCredentials
	}

	// Сравниваем хэш
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		return "", ErrInvalidCredentials
	}

	// Генерируем JWT
	token, err := s.generateToken(u.ID)
	if err != nil {
		s.log.Error("failed to generate jwt on login", "email", email, "error", err)
		return "", fmt.Errorf("failed to generate token: %w", err)
	}

	return token, nil
}

// BotLoginTelegram - upsert и jwt
func (s *Service) BotLoginTelegram(ctx context.Context, tg TelegramProfile) (string, error) {
	if tg.TgID <= 0 {
		return "", fmt.Errorf("tg_id is required")
	}

	u, err := s.db.UpsertTelegramUser(ctx, tg)
	if err != nil {
		return "", err
	}

	token, err := s.generateToken(u.ID)
	if err != nil {
		return "", err
	}

	return token, nil
}

func (s *Service) generateToken(userID int64) (string, error) {
	now := time.Now()

	// Проверка положительного id, мб излишне, в бд serial, на всякий случай
	if userID <= 0 {
		return "", fmt.Errorf("bad user id: %d", userID)
	}
	if userID > math.MaxUint32 {
		return "", fmt.Errorf("user id too large for uint32 claim: %d", userID)
	}

	uid := uint32(userID)

	claims := JWTClaims{
		UserID: uid,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   strconv.FormatUint(uint64(uid), 10),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.tokenTTL)),
		},
	}

	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString(s.jwtSecret)
}

func (s *Service) Ping(ctx context.Context) error {
	return s.db.Ping(ctx)
}
