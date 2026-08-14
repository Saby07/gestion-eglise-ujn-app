package config

import (
	"os"
	"strings"
)

// SecureCookies indique si les cookies doivent avoir le flag Secure.
// En production (GIN_MODE=release), activé par défaut sauf COOKIE_SECURE=false.
func SecureCookies() bool {
	if v := strings.TrimSpace(os.Getenv("COOKIE_SECURE")); v != "" {
		return strings.EqualFold(v, "true")
	}
	return strings.EqualFold(strings.TrimSpace(os.Getenv("GIN_MODE")), "release")
}
