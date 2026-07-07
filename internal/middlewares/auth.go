package middlewares

import (
	"net/http"
	"strings"

	"eglise_ujn/internal/config"
	"eglise_ujn/internal/models"
	"eglise_ujn/internal/services"

	"github.com/gin-gonic/gin"
)

const (
	ContextUserKey = "user"
)

func AuthMiddleware(authSvc *services.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		jcfg, err := config.LoadJWTConfig()
		if err != nil {
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}
		token, err := c.Cookie(jcfg.CookieName)
		if err != nil || token == "" {
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}
		claims, err := config.ParseToken(jcfg, token)
		if err != nil {
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}
		user, err := authSvc.GetUserByID(c.Request.Context(), claims.UserID)
		if err != nil || !user.IsActive {
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
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
		form := c.PostForm("csrf_token")
		if form == "" && strings.HasPrefix(c.GetHeader("Content-Type"), "multipart/form-data") {
			_ = c.Request.ParseMultipartForm(32 << 20)
			form = c.PostForm("csrf_token")
		}
		cookie, _ := c.Cookie("csrf_token")
		if err := config.ValidateCSRF(form, cookie); err != nil {
			c.String(http.StatusForbidden, "CSRF invalide")
			c.Abort()
			return
		}
		c.Next()
	}
}

func SetCSRFCookie(c *gin.Context) string {
	token, err := config.NewCSRFToken()
	if err != nil {
		return ""
	}
	secure := c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https"
	c.SetCookie("csrf_token", token, 3600, "/", "", secure, true)
	return token
}
