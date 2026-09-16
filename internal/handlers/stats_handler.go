package handlers

import (
	"net/http"
	"strconv"
	"time"

	"eglise_ujn/internal/httputil"
	"eglise_ujn/internal/services"
	"eglise_ujn/templates/pages"

	"github.com/gin-gonic/gin"
)

type StatsHandler struct {
	svc *services.StatsService
}

func NewStatsHandler(svc *services.StatsService) *StatsHandler {
	return &StatsHandler{svc: svc}
}

func (h *StatsHandler) Index(c *gin.Context) {
	year := time.Now().Year()
	if yStr := c.Query("year"); yStr != "" {
		if y, err := strconv.Atoi(yStr); err == nil && y > 0 {
			year = y
		}
	}
	stats, err := h.svc.AdvancedStats(c.Request.Context(), year)
	if err != nil {
		httputil.RedirectFlash(c, "/dashboard", "Impossible de charger les statistiques")
		return
	}
	httputil.Render(c, http.StatusOK, pages.Stats(layoutFromCtx(c, "Statistiques", "stats"), mapStatsVM(stats)))
}
