package config

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type JWTConfig struct {
	Secret         []byte
	TTL            time.Duration
	CookieName     string
	CookiePath     string
	CookieHTTPOnly bool
	CookieSameSite string
}

func LoadJWTConfig() (JWTConfig, error) {
	secret := strings.TrimSpace(os.Getenv("JWT_SECRET"))
	if secret == "" {
		return JWTConfig{}, errors.New("JWT_SECRET manquant")
	}
	ttlHours := 24
	if v := strings.TrimSpace(os.Getenv("JWT_TTL_HOURS")); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n <= 0 {
			return JWTConfig{}, fmt.Errorf("JWT_TTL_HOURS invalide: %q", v)
		}
		ttlHours = n
	}
	cookieName := strings.TrimSpace(os.Getenv("COOKIE_NAME"))
	if cookieName == "" {
		cookieName = "eglise_auth"
	}
	return JWTConfig{
		Secret:         []byte(secret),
		TTL:            time.Duration(ttlHours) * time.Hour,
		CookieName:     cookieName,
		CookiePath:     "/",
		CookieHTTPOnly: true,
		CookieSameSite: "lax",
	}, nil
}

type Claims struct {
	UserID uint `json:"uid"`
	jwt.RegisteredClaims
}

func NewToken(cfg JWTConfig, userID uint, ttl time.Duration) (string, error) {
	if ttl <= 0 {
		ttl = cfg.TTL
	}
	claims := Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(cfg.Secret)
}

func ParseToken(cfg JWTConfig, token string) (*Claims, error) {
	t, err := jwt.ParseWithClaims(token, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		return cfg.Secret, nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := t.Claims.(*Claims)
	if !ok || !t.Valid {
		return nil, errors.New("token invalide")
	}
	return claims, nil
}

func NewCSRFToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func ValidateCSRF(form, cookie string) error {
	form = strings.TrimSpace(form)
	cookie = strings.TrimSpace(cookie)
	if form == "" || cookie == "" || form != cookie {
		return errors.New("csrf invalide")
	}
	return nil
}
