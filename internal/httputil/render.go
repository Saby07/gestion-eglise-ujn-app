package httputil

import (
	"net/http"

	"github.com/a-h/templ"
	"github.com/gin-gonic/gin"
)

func Render(c *gin.Context, status int, comp templ.Component) {
	c.Status(status)
	if err := comp.Render(c.Request.Context(), c.Writer); err != nil {
		c.String(http.StatusInternalServerError, "erreur rendu template")
	}
}

func RedirectWithAlert(c *gin.Context, path, alert string) {
	RedirectFlash(c, path, alert)
}
