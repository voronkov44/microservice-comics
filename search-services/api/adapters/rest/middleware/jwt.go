package middleware

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	superuserSubject = "superuser"
	jwtSecret        = "123" // Секрет хардкодом
)

type JWT struct {
	secret []byte
	ttl    time.Duration
}

func NewJWT(ttl time.Duration) *JWT {
	return &JWT{
		secret: []byte(jwtSecret),
		ttl:    ttl,
	}
}

// GenerateSuperuserToken генерируем токен только для superuser
func (j *JWT) GenerateSuperuserToken() (string, error) {
	now := time.Now()

	claims := jwt.RegisteredClaims{
		Subject:   superuserSubject,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(j.ttl)),
	}

	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	s, err := t.SignedString(j.secret)
	if err != nil {
		return "", err
	}
	return s, nil
}

// Разбираем токен и проверяем подпись и валидность
func (j *JWT) parse(tokenStr string) (*jwt.RegisteredClaims, bool) {
	claims := &jwt.RegisteredClaims{}

	t, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
		// Проверяем, что алгоритм - HMAC
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %T", token.Method)
		}
		return j.secret, nil
	})
	if err != nil || !t.Valid {
		return nil, false
	}

	return claims, true
}

// IsSuperuserToken - токен валиден и subject == "superuser"
func (j *JWT) IsSuperuserToken(tokenStr string) bool {
	claims, ok := j.parse(tokenStr)
	if !ok {
		return false
	}
	return claims.Subject == superuserSubject
}
