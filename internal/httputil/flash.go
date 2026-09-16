package httputil

import (
	"net/http"
	"net/url"
	"strings"

	"eglise_ujn/internal/config"

	"github.com/gin-gonic/gin"
)

const flashCookieName = "flash_msg"

func SetFlash(c *gin.Context, message string) {
	message = strings.TrimSpace(message)
	if message == "" {
		return
	}
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     flashCookieName,
		Value:    url.QueryEscape(message),
		Path:     "/",
		MaxAge:   120,
		HttpOnly: true,
		Secure:   secureRequest(c),
		SameSite: http.SameSiteLaxMode,
	})
}

func PopFlash(c *gin.Context) string {
	cookie, err := c.Cookie(flashCookieName)
	if err != nil || cookie == "" {
		return ""
	}
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     flashCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   secureRequest(c),
		SameSite: http.SameSiteLaxMode,
	})
	msg, err := url.QueryUnescape(cookie)
	if err != nil {
		return ""
	}
	return msg
}

func RedirectFlash(c *gin.Context, path, message string) {
	SetFlash(c, message)
	c.Redirect(http.StatusFound, path)
}

func secureRequest(c *gin.Context) bool {
	if config.SecureCookies() {
		return true
	}
	if c.Request.TLS != nil {
		return true
	}
	return strings.EqualFold(c.GetHeader("X-Forwarded-Proto"), "https")
}
