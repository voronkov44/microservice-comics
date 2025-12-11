package core

import (
	"context"
	"fmt"
	"log/slog"
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
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

// Service - доменный сервис авторизации
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

// Register регистрирует нового пользователя и возвращает его числовой ID
func (s *Service) Register(ctx context.Context, email, password string) (string, error) {
	// Проверка, что данные не пустые
	if email == "" || password == "" {
		return "", fmt.Errorf("email and password must not be empty")
	}

	// Валидация email
	if err := validateEmail(email); err != nil {
		return "", ErrInvalidEmail
	}

	// Проверяем, что такого юзера ещё нет
	if _, err := s.db.GetUserByEmail(ctx, email); err == nil {
		return "", ErrUserAlreadyExists
	}

	// Хэшируем пароль
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		s.log.Error("failed to hash password", "error", err)
		return "", fmt.Errorf("failed to hash password: %w", err)
	}

	u := Users{
		Email:        email,
		PasswordHash: string(hash),
		CreatedAt:    time.Now(),
	}

	created, err := s.db.CreateUser(ctx, u)
	if err != nil {
		s.log.Error("failed to create user", "email", email, "error", err)
		return "", fmt.Errorf("failed to create user: %w", err)
	}

	// Генерируем JWT для только что созданного пользователя
	token, err := s.generateToken(created)
	if err != nil {
		s.log.Error("failed to generate jwt on register", "email", email, "error", err)
		return "", fmt.Errorf("failed to generate token: %w", err)
	}

	return token, nil
}

// Login проверяет логин и пароль и возвращает jwt-токен
func (s *Service) Login(ctx context.Context, email, password string) (string, error) {
	if email == "" || password == "" {
		return "", ErrInvalidCredentials
	}

	// Валидация email
	if err := validateEmail(email); err != nil {
		return "", ErrInvalidEmail
	}

	u, err := s.db.GetUserByEmail(ctx, email)
	if err != nil {
		s.log.Warn("user not found or db error on login", "email", email, "error", err)
		return "", ErrInvalidCredentials
	}

	// Сравниваем хэш
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		return "", ErrInvalidCredentials
	}

	// Генерируем JWT
	token, err := s.generateToken(u)
	if err != nil {
		s.log.Error("failed to generate jwt", "email", email, "error", err)
		return "", fmt.Errorf("failed to generate token: %w", err)
	}

	return token, nil
}

func (s *Service) generateToken(u Users) (string, error) {
	now := time.Now()

	// Проверка положительного id, мб излишне, в бд serial, на всякий случай
	if u.ID < 0 {
		return "", fmt.Errorf("negative user id: %d", u.ID)
	}
	userID := uint32(u.ID)

	claims := JWTClaims{
		UserID: userID,
		Email:  u.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   strconv.FormatUint(uint64(userID), 10),
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
