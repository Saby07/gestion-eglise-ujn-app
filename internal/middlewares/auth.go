package middlewares

import (
	"net/http"
	"strings"

	"eglise_ujn/internal/config"
	"eglise_ujn/internal/httputil"
	"eglise_ujn/internal/models"
	"eglise_ujn/internal/services"

	"github.com/gin-gonic/gin"
)

const (
	ContextUserKey = "user"
)

func wantsJSONResponse(c *gin.Context) bool {
	if strings.HasPrefix(c.Request.URL.Path, "/api/") {
		return true
	}
	return strings.Contains(c.GetHeader("Accept"), "application/json")
}

func authUnauthorized(c *gin.Context) {
	if wantsJSONResponse(c) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Non authentifié"})
		c.Abort()
		return
	}
	c.Redirect(http.StatusFound, "/login")
	c.Abort()
}

func AuthMiddleware(authSvc *services.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		jcfg, err := config.LoadJWTConfig()
		if err != nil {
			authUnauthorized(c)
			return
		}
		token, err := c.Cookie(jcfg.CookieName)
		if err != nil || token == "" {
			authUnauthorized(c)
			return
		}
		claims, err := config.ParseToken(jcfg, token)
		if err != nil {
			authUnauthorized(c)
			return
		}
		user, err := authSvc.GetUserByID(c.Request.Context(), claims.UserID)
		if err != nil || !user.IsActive {
			authUnauthorized(c)
			return
		}
		c.Set(ContextUserKey, user)
		c.Next()
	}
}

func CurrentUser(c *gin.Context) *models.User {
	v, ok := c.Get(ContextUserKey)
	if !ok {
		return nil
	}
	u, _ := v.(*models.User)
	return u
}

func HasRole(c *gin.Context, keys ...models.RoleKey) bool {
	u := CurrentUser(c)
	if u == nil {
		return false
	}
	for _, k := range keys {
		if u.HasRole(k) {
			return true
		}
	}
	return false
}

func RequireRoles(keys ...models.RoleKey) gin.HandlerFunc {
	return func(c *gin.Context) {
		if HasRole(c, keys...) {
			c.Next()
			return
		}
		if wantsJSONResponse(c) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Accès non autorisé"})
			c.Abort()
			return
		}
		c.Redirect(http.StatusFound, "/dashboard")
		c.Abort()
	}
}

func CSRFMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method == http.MethodGet || c.Request.Method == http.MethodHead {
			c.Next()
			return
		}
		if strings.HasPrefix(c.Request.URL.Path, "/api/") {
			c.Next()
			return
		}
		if err := c.Request.ParseForm(); err != nil {
			c.String(http.StatusForbidden, "CSRF invalide")
			c.Abort()
			return
		}
		form := c.PostForm("csrf_token")
		if form == "" && strings.HasPrefix(c.GetHeader("Content-Type"), "multipart/form-data") {
			_ = c.Request.ParseMultipartForm(32 << 20)
			form = c.PostForm("csrf_token")
		}
		cookie, _ := c.Cookie(csrfCookieName)
		if err := config.ValidateCSRF(form, cookie); err != nil {
			c.String(http.StatusForbidden, "CSRF invalide")
			c.Abort()
			return
		}
		c.Next()
	}
}

const csrfCookieName = "csrf_token"

func SetCSRFCookie(c *gin.Context) string {
	if existing, err := c.Cookie(csrfCookieName); err == nil && isValidCSRFToken(existing) {
		return existing
	}
	token, err := config.NewCSRFToken()
	if err != nil {
		return ""
	}
	writeCSRFCookie(c, token)
	return token
}

func isValidCSRFToken(token string) bool {
	token = strings.TrimSpace(token)
	return len(token) >= 32 && len(token) <= 64
}

func writeCSRFCookie(c *gin.Context, token string) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     csrfCookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   3600,
		HttpOnly: true,
		Secure:   requestUsesSecureCookies(c),
		SameSite: http.SameSiteLaxMode,
	})
}

func requestUsesSecureCookies(c *gin.Context) bool {
	if config.SecureCookies() {
		return true
	}
	if c.Request.TLS != nil {
		return true
	}
	return strings.EqualFold(c.GetHeader("X-Forwarded-Proto"), "https")
}

func SetAuthCookie(c *gin.Context, name, value string, maxAge int) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   requestUsesSecureCookies(c),
		SameSite: http.SameSiteLaxMode,
	})
}

func LoginCSRFMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method != http.MethodPost {
			c.Next()
			return
		}
		if err := c.Request.ParseForm(); err != nil {
			c.String(http.StatusForbidden, "CSRF invalide")
			c.Abort()
			return
		}
		form := c.PostForm("csrf_token")
		cookie, _ := c.Cookie(csrfCookieName)
		if err := config.ValidateCSRF(form, cookie); err != nil {
			httputil.SetFlash(c, "Session expirée, veuillez réessayer")
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}
		c.Next()
	}
}
