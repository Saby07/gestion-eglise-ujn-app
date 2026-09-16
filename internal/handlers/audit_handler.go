package handlers

import (
	"net/http"

	"eglise_ujn/internal/httputil"
	"eglise_ujn/internal/services"
	"eglise_ujn/templates/pages"

	"github.com/gin-gonic/gin"
)

type AuditHandler struct {
	svc *services.AuditService
}

func NewAuditHandler(svc *services.AuditService) *AuditHandler {
	return &AuditHandler{svc: svc}
}

func (h *AuditHandler) List(c *gin.Context) {
	list, _ := h.svc.List(c.Request.Context(), 200)
	httputil.Render(c, http.StatusOK, pages.AuditLog(layoutFromCtx(c, "Journal d'audit", "audit"), mapAuditLogs(list)))
}
