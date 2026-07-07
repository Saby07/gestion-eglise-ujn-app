package handlers

import (
	"net/http"
	"strconv"

	"eglise_ujn/internal/middlewares"
	"eglise_ujn/internal/services"

	"github.com/gin-gonic/gin"
)

type NotificationHandler struct {
	svc *services.NotificationService
}

func NewNotificationHandler(svc *services.NotificationService) *NotificationHandler {
	return &NotificationHandler{svc: svc}
}

func (h *NotificationHandler) Read(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.Redirect(http.StatusFound, "/requisitions")
		return
	}
	user := middlewares.CurrentUser(c)
	if user == nil {
		c.Redirect(http.StatusFound, "/login")
		return
	}
	n, err := h.svc.MarkRead(c.Request.Context(), uint(id), user.ID)
	if err != nil {
		c.Redirect(http.StatusFound, "/requisitions")
		return
	}
	c.Redirect(http.StatusFound, "/requisitions/"+strconv.FormatUint(uint64(n.RequisitionID), 10))
}
